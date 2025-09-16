// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package qcloud

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
	"gopkg.in/fatih/set.v0"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/s3cli"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	COS_META_HEADER = "X-Cos-Meta-"
)

type SBucket struct {
	multicloud.SBaseBucket
	QcloudTags

	appId string

	region *SRegion
	zone   *SZone

	Name       string
	Location   string
	CreateDate time.Time
}

func (b *SBucket) GetProjectId() string {
	return ""
}

func (b *SBucket) GetGlobalId() string {
	if b.getAppId() == b.region.client.GetAppId() {
		return b.Name
	} else {
		return b.getFullName()
	}
}

func (b *SBucket) GetName() string {
	return b.GetGlobalId()
}

func (b *SBucket) GetLocation() string {
	return b.Location
}

func (b *SBucket) GetIRegion() cloudprovider.ICloudRegion {
	return b.region
}

func (b *SBucket) GetCreatedAt() time.Time {
	return b.CreateDate
}

func (b *SBucket) GetStorageClass() string {
	return ""
}

const (
	ACL_GROUP_URI_ALL_USERS  = "http://cam.qcloud.com/groups/global/AllUsers"
	ACL_GROUP_URI_AUTH_USERS = "http://cam.qcloud.com/groups/global/AuthenticatedUsers"
)

func cosAcl2CannedAcl(acls []cos.ACLGrant) cloudprovider.TBucketACLType {
	switch {
	case len(acls) == 1:
		if acls[0].Grantee.URI == "" && acls[0].Permission == s3cli.PERMISSION_FULL_CONTROL {
			return cloudprovider.ACLPrivate
		}
	case len(acls) == 2:
		for _, g := range acls {
			if g.Grantee.URI == ACL_GROUP_URI_AUTH_USERS && g.Permission == s3cli.PERMISSION_READ {
				return cloudprovider.ACLAuthRead
			}
			if g.Grantee.URI == ACL_GROUP_URI_ALL_USERS && g.Permission == s3cli.PERMISSION_READ {
				return cloudprovider.ACLPublicRead
			}
		}
	case len(acls) == 3:
		for _, g := range acls {
			if g.Grantee.URI == ACL_GROUP_URI_ALL_USERS && g.Permission == s3cli.PERMISSION_WRITE {
				return cloudprovider.ACLPublicReadWrite
			}
		}
	}
	return cloudprovider.ACLUnknown
}

func (b *SBucket) GetAcl() cloudprovider.TBucketACLType {
	acl := cloudprovider.ACLPrivate
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		log.Errorf("GetCosClient fail %s", err)
		return acl
	}
	result, _, err := coscli.Bucket.GetACL(context.Background())
	if err != nil {
		log.Errorf("coscli.Bucket.GetACL fail %s", err)
		return acl
	}
	return cosAcl2CannedAcl(result.AccessControlList)
}

func (b *SBucket) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "b.region.GetCosClient")
	}
	opts := &cos.BucketPutACLOptions{}
	opts.Header = &cos.ACLHeaderOptions{}
	opts.Header.XCosACL = string(aclStr)
	_, err = coscli.Bucket.PutACL(context.Background(), opts)
	if err != nil {
		return errors.Wrap(err, "PutACL")
	}
	return nil
}

func (b *SBucket) getAppId() string {
	if len(b.appId) > 0 {
		return b.appId
	}
	if b.zone != nil {
		return b.zone.region.client.GetAppId()
	}
	return b.region.client.GetAppId()
}

func (b *SBucket) getFullName() string {
	return fmt.Sprintf("%s-%s", b.Name, b.getAppId())
}

func (b *SBucket) getBucketUrlHost() string {
	if b.zone != nil {
		return fmt.Sprintf("%s.%s", b.getFullName(), b.zone.getCosEndpoint())
	} else {
		return fmt.Sprintf("%s.%s", b.getFullName(), b.region.getCosEndpoint())
	}
}

func (b *SBucket) getBucketUrl() string {
	return fmt.Sprintf("https://%s", b.getBucketUrlHost())
}

func (b *SBucket) getBucketWebsiteUrlHost() string {
	if b.zone != nil {
		return fmt.Sprintf("%s.%s", b.getFullName(), b.zone.getCosWebsiteEndpoint())
	} else {
		return fmt.Sprintf("%s.%s", b.getFullName(), b.region.getCosWebsiteEndpoint())
	}
}

func (b *SBucket) getWebsiteUrl() string {
	return fmt.Sprintf("https://%s", b.getBucketWebsiteUrlHost())
}

func (b *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	return []cloudprovider.SBucketAccessUrl{
		{
			Url:         b.getBucketUrl(),
			Description: "bucket domain",
			Primary:     true,
		},
		{
			Url:         fmt.Sprintf("https://%s/%s", b.region.getCosEndpoint(), b.getFullName()),
			Description: "cos domain",
		},
	}
}

func (b *SBucket) GetStats() cloudprovider.SBucketStats {
	stats, _ := cloudprovider.GetIBucketStats(b)
	return stats
}

func (b *SBucket) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	result := cloudprovider.SListObjectResult{}
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return result, errors.Wrap(err, "GetCosClient")
	}
	opts := &cos.BucketGetOptions{}
	if len(prefix) > 0 {
		opts.Prefix = prefix
	}
	if len(marker) > 0 {
		opts.Marker = marker
	}
	if len(delimiter) > 0 {
		opts.Delimiter = delimiter
	}
	if maxCount > 0 {
		opts.MaxKeys = maxCount
	}
	oResult, _, err := coscli.Bucket.Get(context.Background(), opts)
	if err != nil {
		return result, errors.Wrap(err, "coscli.Bucket.Get")
	}
	result.Objects = make([]cloudprovider.ICloudObject, 0)
	for _, object := range oResult.Contents {
		lastModified, _ := timeutils.ParseTimeStr(object.LastModified)
		obj := &SObject{
			bucket: b,
			SBaseCloudObject: cloudprovider.SBaseCloudObject{
				StorageClass: string(object.StorageClass),
				Key:          object.Key,
				SizeBytes:    int64(object.Size),
				ETag:         object.ETag,
				LastModified: lastModified,
			},
		}
		result.Objects = append(result.Objects, obj)
	}
	if oResult.CommonPrefixes != nil {
		result.CommonPrefixes = make([]cloudprovider.ICloudObject, 0)
		for _, commPrefix := range oResult.CommonPrefixes {
			obj := &SObject{
				bucket: b,
				SBaseCloudObject: cloudprovider.SBaseCloudObject{
					Key: commPrefix,
				},
			}
			result.CommonPrefixes = append(result.CommonPrefixes, obj)
		}
	}
	result.IsTruncated = oResult.IsTruncated
	result.NextMarker = oResult.NextMarker
	return result, nil
}

func (b *SBucket) PutObject(ctx context.Context, key string, reader io.Reader, sizeBytes int64, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "GetCosClient")
	}
	opts := &cos.ObjectPutOptions{
		ACLHeaderOptions:       &cos.ACLHeaderOptions{},
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{},
	}
	if sizeBytes > 0 {
		opts.ContentLength = sizeBytes
	}
	if meta != nil {
		extraHdr := http.Header{}
		for k, v := range meta {
			if len(v) == 0 || len(v[0]) == 0 {
				continue
			}
			switch http.CanonicalHeaderKey(k) {
			case cloudprovider.META_HEADER_CACHE_CONTROL:
				opts.CacheControl = v[0]
			case cloudprovider.META_HEADER_CONTENT_TYPE:
				opts.ContentType = v[0]
			case cloudprovider.META_HEADER_CONTENT_MD5:
				opts.ContentMD5 = v[0]
			case cloudprovider.META_HEADER_CONTENT_ENCODING:
				opts.ContentEncoding = v[0]
			case cloudprovider.META_HEADER_CONTENT_DISPOSITION:
				opts.ContentDisposition = v[0]
			default:
				extraHdr.Add(fmt.Sprintf("%s%s", COS_META_HEADER, k), v[0])
			}
		}
		if len(extraHdr) > 0 {
			opts.XCosMetaXXX = &extraHdr
		}
	}
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	opts.XCosACL = string(cannedAcl)
	if len(storageClassStr) > 0 {
		opts.XCosStorageClass = storageClassStr
	}
	_, err = coscli.Object.Put(ctx, key, reader, opts)
	if err != nil {
		return errors.Wrap(err, "coscli.Object.Put")
	}
	return nil
}

func (b *SBucket) NewMultipartUpload(ctx context.Context, key string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) (string, error) {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return "", errors.Wrap(err, "GetCosClient")
	}
	opts := &cos.InitiateMultipartUploadOptions{
		ACLHeaderOptions:       &cos.ACLHeaderOptions{},
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{},
	}
	if meta != nil {
		extraHdr := http.Header{}
		for k, v := range meta {
			if len(v) == 0 || len(v[0]) == 0 {
				continue
			}
			switch http.CanonicalHeaderKey(k) {
			case cloudprovider.META_HEADER_CACHE_CONTROL:
				opts.CacheControl = v[0]
			case cloudprovider.META_HEADER_CONTENT_TYPE:
				opts.ContentType = v[0]
			case cloudprovider.META_HEADER_CONTENT_MD5:
				opts.ContentMD5 = v[0]
			case cloudprovider.META_HEADER_CONTENT_ENCODING:
				opts.ContentEncoding = v[0]
			case cloudprovider.META_HEADER_CONTENT_DISPOSITION:
				opts.ContentDisposition = v[0]
			default:
				extraHdr.Add(fmt.Sprintf("%s%s", COS_META_HEADER, k), v[0])
			}
		}
		if len(extraHdr) > 0 {
			opts.XCosMetaXXX = &extraHdr
		}
	}
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	opts.XCosACL = string(cannedAcl)
	if len(storageClassStr) > 0 {
		opts.XCosStorageClass = storageClassStr
	}
	result, _, err := coscli.Object.InitiateMultipartUpload(ctx, key, opts)
	if err != nil {
		return "", errors.Wrap(err, "InitiateMultipartUpload")
	}

	return result.UploadID, nil
}

func (b *SBucket) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, input io.Reader, partSize int64, offset, totalSize int64) (string, error) {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return "", errors.Wrap(err, "GetCosClient")
	}
	opts := &cos.ObjectUploadPartOptions{}
	opts.ContentLength = partSize
	resp, err := coscli.Object.UploadPart(ctx, key, uploadId, partIndex, input, opts)
	if err != nil {
		return "", errors.Wrap(err, "UploadPart")
	}

	return resp.Header.Get("Etag"), nil
}

func (b *SBucket) CompleteMultipartUpload(ctx context.Context, key string, uploadId string, partEtags []string) error {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "GetCosClient")
	}
	opts := &cos.CompleteMultipartUploadOptions{}
	parts := make([]cos.Object, len(partEtags))
	for i := range partEtags {
		parts[i] = cos.Object{
			PartNumber: i + 1,
			ETag:       partEtags[i],
		}
	}
	opts.Parts = parts
	_, _, err = coscli.Object.CompleteMultipartUpload(ctx, key, uploadId, opts)

	if err != nil {
		return errors.Wrap(err, "CompleteMultipartUpload")
	}

	return nil
}

func (b *SBucket) AbortMultipartUpload(ctx context.Context, key string, uploadId string) error {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "GetCosClient")
	}

	_, err = coscli.Object.AbortMultipartUpload(ctx, key, uploadId)
	if err != nil {
		return errors.Wrap(err, "AbortMultipartUpload")
	}

	return nil
}

func (b *SBucket) DeleteObject(ctx context.Context, key string) error {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "GetCosClient")
	}
	_, err = coscli.Object.Delete(ctx, key)
	if err != nil {
		return errors.Wrap(err, "coscli.Object.Delete")
	}
	return nil
}

func (b *SBucket) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	if method != "GET" && method != "PUT" && method != "DELETE" {
		return "", errors.Error("unsupported method")
	}
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return "", errors.Wrap(err, "GetCosClient")
	}
	url, err := coscli.Object.GetPresignedURL(context.Background(), method, key,
		b.region.client.secretId,
		b.region.client.secretKey,
		expire, nil)
	if err != nil {
		return "", errors.Wrap(err, "coscli.Object.GetPresignedURL")
	}
	return url.String(), nil
}

func (b *SBucket) CopyObject(ctx context.Context, destKey string, srcBucketName, srcKey string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "GetCosClient")
	}
	opts := &cos.ObjectCopyOptions{
		ObjectCopyHeaderOptions: &cos.ObjectCopyHeaderOptions{},
		ACLHeaderOptions:        &cos.ACLHeaderOptions{},
	}
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	opts.XCosACL = string(cannedAcl)
	if len(storageClassStr) > 0 {
		opts.XCosStorageClass = storageClassStr
	}
	if meta != nil {
		opts.XCosMetadataDirective = "Replaced"
		extraHdr := http.Header{}
		for k, v := range meta {
			if len(v) == 0 || len(v[0]) == 0 {
				continue
			}
			switch http.CanonicalHeaderKey(k) {
			case cloudprovider.META_HEADER_CACHE_CONTROL:
				opts.CacheControl = v[0]
			case cloudprovider.META_HEADER_CONTENT_TYPE:
				opts.ContentType = v[0]
			case cloudprovider.META_HEADER_CONTENT_ENCODING:
				opts.ContentEncoding = v[0]
			case cloudprovider.META_HEADER_CONTENT_DISPOSITION:
				opts.ContentDisposition = v[0]
			default:
				extraHdr.Add(fmt.Sprintf("%s%s", COS_META_HEADER, k), v[0])
			}
		}
		if len(extraHdr) > 0 {
			opts.XCosMetaXXX = &extraHdr
		}
	} else {
		opts.XCosMetadataDirective = "Copy"
	}
	srcBucket := SBucket{
		region: b.region,
		Name:   srcBucketName,
	}
	srcUrl := fmt.Sprintf("%s/%s", srcBucket.getBucketUrlHost(), srcKey)
	_, _, err = coscli.Object.Copy(ctx, destKey, srcUrl, opts)
	if err != nil {
		return errors.Wrap(err, "coscli.Object.Copy")
	}
	return nil
}

func (b *SBucket) GetObject(ctx context.Context, key string, rangeOpt *cloudprovider.SGetObjectRange) (io.ReadCloser, error) {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return nil, errors.Wrap(err, "GetCosClient")
	}
	opts := &cos.ObjectGetOptions{}
	if rangeOpt != nil {
		opts.Range = rangeOpt.String()
	}
	resp, err := coscli.Object.Get(ctx, key, opts)
	if err != nil {
		return nil, errors.Wrap(err, "coscli.Object.Get")
	}
	return resp.Body, nil
}

func (b *SBucket) CopyPart(ctx context.Context, key string, uploadId string, partIndex int, srcBucketName string, srcKey string, srcOffset int64, srcLength int64) (string, error) {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return "", errors.Wrap(err, "GetCosClient")
	}
	srcBucket := SBucket{
		region: b.region,
		Name:   srcBucketName,
	}
	opts := cos.ObjectCopyPartOptions{}
	srcUrl := fmt.Sprintf("%s/%s", srcBucket.getBucketUrlHost(), srcKey)
	opts.XCosCopySourceRange = fmt.Sprintf("bytes=%d-%d", srcOffset, srcOffset+srcLength-1)
	result, _, err := coscli.Object.CopyPart(ctx, key, uploadId, partIndex, srcUrl, &opts)
	if err != nil {
		return "", errors.Wrap(err, "coscli.Object.CopyPart")
	}
	return result.ETag, nil
}

func (b *SBucket) SetWebsite(websitConf cloudprovider.SBucketWebsiteConf) error {
	if len(websitConf.Index) == 0 {
		return errors.Wrap(cloudprovider.ErrNotSupported, "missing Index")
	}
	if len(websitConf.ErrorDocument) == 0 {
		return errors.Wrap(cloudprovider.ErrNotSupported, "missing ErrorDocument")
	}
	if websitConf.Protocol != "http" && websitConf.Protocol != "https" {
		return errors.Wrap(cloudprovider.ErrNotSupported, "missing Protocol")
	}

	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "b.region.GetCosClient")
	}

	rulesOpts := []cos.WebsiteRoutingRule{}
	for i := range websitConf.Rules {
		rulesOpts = append(rulesOpts, cos.WebsiteRoutingRule{
			ConditionErrorCode: websitConf.Rules[i].ConditionErrorCode,
			ConditionPrefix:    websitConf.Rules[i].ConditionPrefix,

			RedirectProtocol:         websitConf.Rules[i].RedirectProtocol,
			RedirectReplaceKey:       websitConf.Rules[i].RedirectReplaceKey,
			RedirectReplaceKeyPrefix: websitConf.Rules[i].ConditionPrefix,
		})
	}
	opts := &cos.BucketPutWebsiteOptions{
		Index:            websitConf.Index,
		Error:            &cos.ErrorDocument{Key: websitConf.ErrorDocument},
		RedirectProtocol: &cos.RedirectRequestsProtocol{Protocol: websitConf.Protocol},
	}
	if len(rulesOpts) > 0 {
		opts.RoutingRules = &cos.WebsiteRoutingRules{Rules: rulesOpts}
	}

	_, err = coscli.Bucket.PutWebsite(context.Background(), opts)
	if err != nil {
		return errors.Wrap(err, "PutWebsite")
	}
	return nil
}

func (b *SBucket) GetWebsiteConf() (cloudprovider.SBucketWebsiteConf, error) {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return cloudprovider.SBucketWebsiteConf{}, errors.Wrap(err, "b.region.GetCosClient")
	}
	websiteResult, _, err := coscli.Bucket.GetWebsite(context.Background())
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchWebsiteConfiguration") {
			return cloudprovider.SBucketWebsiteConf{}, nil
		}
		return cloudprovider.SBucketWebsiteConf{}, errors.Wrap(err, "coscli.Bucket.GetWebsite")
	}

	result := cloudprovider.SBucketWebsiteConf{
		Index: websiteResult.Index,
	}
	if websiteResult.Error != nil {
		result.ErrorDocument = websiteResult.Error.Key
	}
	if websiteResult.RedirectProtocol != nil {
		result.Protocol = websiteResult.RedirectProtocol.Protocol
	}
	routingRules := []cloudprovider.SBucketWebsiteRoutingRule{}
	if websiteResult.RoutingRules != nil {
		for i := range websiteResult.RoutingRules.Rules {
			routingRules = append(routingRules, cloudprovider.SBucketWebsiteRoutingRule{
				ConditionErrorCode: websiteResult.RoutingRules.Rules[i].ConditionErrorCode,
				ConditionPrefix:    websiteResult.RoutingRules.Rules[i].ConditionPrefix,

				RedirectProtocol:         websiteResult.RoutingRules.Rules[i].RedirectProtocol,
				RedirectReplaceKey:       websiteResult.RoutingRules.Rules[i].RedirectReplaceKey,
				RedirectReplaceKeyPrefix: websiteResult.RoutingRules.Rules[i].RedirectReplaceKeyPrefix,
			})
		}
	}
	result.Rules = routingRules
	result.Url = b.getWebsiteUrl()
	return result, nil
}

func (b *SBucket) DeleteWebSiteConf() error {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "b.region.GetCosClient")
	}
	_, err = coscli.Bucket.DeleteWebsite(context.Background())
	if err != nil {
		return errors.Wrap(err, "coscli.Bucket.DeleteWebsite")
	}
	return nil
}

func (b *SBucket) SetCORS(rules []cloudprovider.SBucketCORSRule) error {
	if len(rules) == 0 {
		return nil
	}
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "b.region.GetCosClient")
	}
	input := cos.BucketPutCORSOptions{}
	for i := range rules {
		input.Rules = append(input.Rules, cos.BucketCORSRule{
			AllowedOrigins: rules[i].AllowedOrigins,
			AllowedMethods: rules[i].AllowedMethods,
			AllowedHeaders: rules[i].AllowedHeaders,
			MaxAgeSeconds:  rules[i].MaxAgeSeconds,
			ExposeHeaders:  rules[i].ExposeHeaders,
			ID:             rules[i].Id,
		})
	}

	_, err = coscli.Bucket.PutCORS(context.Background(), &input)
	if err != nil {
		return errors.Wrap(err, "coscli.Bucket.PutCORS")
	}
	return nil
}

func (b *SBucket) GetCORSRules() ([]cloudprovider.SBucketCORSRule, error) {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return nil, errors.Wrap(err, "b.region.GetCosClient")
	}
	conf, _, err := coscli.Bucket.GetCORS(context.Background())
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchCORSConfiguration") {
			return nil, nil
		}
		return nil, errors.Wrap(err, "b.region.GetCORS")
	}
	result := []cloudprovider.SBucketCORSRule{}
	for i := range conf.Rules {
		result = append(result, cloudprovider.SBucketCORSRule{
			AllowedOrigins: conf.Rules[i].AllowedOrigins,
			AllowedMethods: conf.Rules[i].AllowedMethods,
			AllowedHeaders: conf.Rules[i].AllowedHeaders,
			MaxAgeSeconds:  conf.Rules[i].MaxAgeSeconds,
			ExposeHeaders:  conf.Rules[i].ExposeHeaders,
			Id:             strconv.Itoa(i),
		})
	}
	return result, nil
}

func (b *SBucket) DeleteCORS() error {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "b.region.GetCosClient")
	}
	_, err = coscli.Bucket.DeleteCORS(context.Background())
	if err != nil {
		return errors.Wrap(err, "coscli.Bucket.DeleteCORS")
	}
	return nil
}

func (b *SBucket) SetReferer(conf cloudprovider.SBucketRefererConf) error {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "b.region.GetCosClient")
	}

	if !conf.Enabled {
		_, err = coscli.Bucket.PutReferer(context.Background(), nil)
		return errors.Wrap(err, "Disable Refer")
	}

	opts := cos.BucketPutRefererOptions{
		Status:                  "Enabled",
		EmptyReferConfiguration: "Deny",
		RefererType:             conf.RefererType,
		DomainList:              conf.DomainList,
	}

	if conf.AllowEmptyRefer {
		opts.EmptyReferConfiguration = "Allow"
	}

	_, err = coscli.Bucket.PutReferer(context.Background(), &opts)
	if err != nil {
		return errors.Wrap(err, "coscli.Bucket.PutReferer")
	}
	return nil
}
func (b *SBucket) GetReferer() (cloudprovider.SBucketRefererConf, error) {
	result := cloudprovider.SBucketRefererConf{}
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return result, errors.Wrap(err, "b.region.GetCosClient")
	}

	referResult, _, err := coscli.Bucket.GetReferer(context.Background())
	if err != nil {
		return result, errors.Wrap(err, " coscli.Bucket.GetReferer")
	}

	result.AllowEmptyRefer = (referResult.EmptyReferConfiguration == "Allow")
	result.Enabled = (referResult.Status == "Enabled")
	result.RefererType = referResult.RefererType
	result.DomainList = referResult.DomainList

	return result, nil
}

func toAPICdnArea(area string) string {
	switch area {
	case "mainland":
		return api.CDN_DOMAIN_AREA_MAINLAND
	case "overseas":
		return api.CDN_DOMAIN_AREA_OVERSEAS
	case "global":
		return api.CDN_DOMAIN_AREA_GLOBAL
	default:
		return ""
	}
}
func toAPICdnStatus(status string) string {
	switch status {
	case "online":
		return api.CDN_DOMAIN_STATUS_ONLINE
	case "offline":
		return api.CDN_DOMAIN_STATUS_OFFLINE
	case "processing":
		return api.CDN_DOMAIN_STATUS_PROCESSING
	case "rejected":
		return api.CDN_DOMAIN_STATUS_REJECTED
	default:
		return ""
	}
}

func (b *SBucket) GetCdnDomains() ([]cloudprovider.SCdnDomain, error) {
	result := []cloudprovider.SCdnDomain{}
	bucketHost := b.getBucketUrlHost()
	bucketWebsiteHost := b.getBucketWebsiteUrlHost()

	bucketCdnDomains, err := b.region.client.DescribeAllCdnDomains(nil, []string{bucketHost}, "cos")
	if err != nil {
		return nil, errors.Wrapf(err, `b.region.client.DescribeAllCdnDomains(nil, []string{%s}, "cos")`, bucketHost)
	}

	for i := range bucketCdnDomains {
		result = append(result, cloudprovider.SCdnDomain{
			Domain:     bucketCdnDomains[i].Domain,
			Status:     toAPICdnStatus(bucketCdnDomains[i].Status),
			Cname:      bucketCdnDomains[i].Cname,
			Area:       toAPICdnArea(bucketCdnDomains[i].Area),
			Origin:     bucketHost,
			OriginType: api.CDN_DOMAIN_ORIGIN_TYPE_BUCKET,
		})
	}

	bucketWebsiteCdnDomains, err := b.region.client.DescribeAllCdnDomains(nil, []string{bucketWebsiteHost}, "cos")
	if err != nil {
		return nil, errors.Wrapf(err, `b.region.client.DescribeAllCdnDomains(nil, []string{%s}, "cos")`, bucketWebsiteHost)
	}

	for i := range bucketWebsiteCdnDomains {
		result = append(result, cloudprovider.SCdnDomain{
			Domain:     bucketWebsiteCdnDomains[i].Domain,
			Status:     toAPICdnStatus(bucketWebsiteCdnDomains[i].Status),
			Cname:      bucketWebsiteCdnDomains[i].Cname,
			Area:       toAPICdnArea(bucketWebsiteCdnDomains[i].Area),
			Origin:     bucketWebsiteHost,
			OriginType: api.CDN_DOMAIN_ORIGIN_TYPE_BUCKET,
		})
	}
	return result, nil
}

func getQcsResourcePath(resource []string) []string {
	path := []string{}
	for i := range resource {
		strs := strings.Split(resource[i], ":")
		path = append(path, strs[len(strs)-1])
	}
	return path
}

func getQcsUserId(principal []string) []string {
	ids := []string{}
	for i := range principal {
		//  qcs::cam::uin/100008182714:uin/100008182714
		//  qcs::cam::uin/100008182714:service/cdn
		//  qcs::cam::anyone:anyone

		strs := strings.Split(principal[i], "::")
		ids = append(ids, strings.Replace(strs[len(strs)-1], "uin/", "", 2))
	}

	return ids
}

var cannedReadActions = [...]string{
	"name/cos:GetBucket",
	"name/cos:GetBucketObjectVersions",
	"name/cos:HeadBucket",
	"name/cos:ListMultipartUploads",
	"name/cos:ListParts",
	"name/cos:GetObject",
	"name/cos:HeadObject",
	"name/cos:OptionsObject",
}

var cannedReadWriteActions = [...]string{
	"name/cos:GetBucket",
	"name/cos:GetBucketObjectVersions",
	"name/cos:HeadBucket",
	"name/cos:ListMultipartUploads",
	"name/cos:ListParts",
	"name/cos:GetObject",
	"name/cos:HeadObject",
	"name/cos:OptionsObject",

	"name/cos:PutObject",
	"name/cos:PostObject",
	"name/cos:DeleteObject",
	"name/cos:InitiateMultipartUpload",
	"name/cos:UploadPart",
	"name/cos:CompleteMultipartUpload",
	"name/cos:AbortMultipartUpload",
}

func getCannedAction(action []string) string {
	cannedAction := ""

	actionSet := set.New(set.NonThreadSafe)
	for i := range action {
		actionSet.Add(action[i])
	}
	if actionSet.Has("name/cos:*") {
		return "FullControl"
	}

	readSet := set.New(set.NonThreadSafe)
	for i := range cannedReadActions {
		readSet.Add(cannedReadActions[i])
	}
	if set.Difference(readSet, actionSet).Size() == 0 {
		cannedAction = "Read"
	}

	readWriteSet := set.New(set.NonThreadSafe)
	for i := range cannedReadWriteActions {
		readWriteSet.Add(cannedReadWriteActions[i])
	}
	if set.Difference(readWriteSet, actionSet).Size() == 0 {
		cannedAction = "ReadWrite"
	}
	return cannedAction
}

func (b *SBucket) GetPolicy() ([]cloudprovider.SBucketPolicyStatement, error) {
	policyOptions := []cloudprovider.SBucketPolicyStatement{}
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return nil, errors.Wrap(err, "GetCosClient")
	}
	result, _, err := coscli.Bucket.GetPolicy(context.Background())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, errors.Wrap(err, "GetPolicy")
	}

	users, err := b.region.client.GetICloudusers()
	if err != nil {
		return nil, errors.Wrapf(err, "GetICloudusers")
	}

	userMaps := map[string]string{}
	for i := range users {
		userMaps[fmt.Sprintf("%s:%s", b.region.client.ownerName, users[i].GetGlobalId())] = users[i].GetName()
	}

	for i := range result.Statement {
		policyOption := cloudprovider.SBucketPolicyStatement{
			Principal: result.Statement[i].Principal,
			Action:    result.Statement[i].Action,
			Effect:    result.Statement[i].Effect,
			Resource:  result.Statement[i].Resource,
			Condition: result.Statement[i].Condition,

			PrincipalId:  getQcsUserId(result.Statement[i].Principal["qcs"]),
			CannedAction: getCannedAction(result.Statement[i].Action),
			ResourcePath: getQcsResourcePath(result.Statement[i].Resource),
			Id:           strconv.Itoa(i),
		}
		policyOption.PrincipalNames = func() map[string]string {
			ret := map[string]string{}
			for _, id := range policyOption.PrincipalId {
				ret[id], _ = userMaps[id]
			}
			return ret
		}()
		policyOptions = append(policyOptions, policyOption)
	}
	return policyOptions, nil
}

func (b *SBucket) SetPolicy(policy cloudprovider.SBucketPolicyStatementInput) error {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrapf(err, "GetCosClient")
	}
	opts := cos.BucketPutPolicyOptions{}
	opts.Version = "2.0"
	oldOpts, _, err := coscli.Bucket.GetPolicy(context.Background())
	if err != nil {
		if !strings.Contains(err.Error(), "404") {
			return errors.Wrap(err, "GetPolicy")
		}
	}
	if len(oldOpts.Statement) > 0 {
		opts.Statement = oldOpts.Statement
	}
	newStatement := cos.BucketStatement{}
	ids := []string{}
	for i := range policy.PrincipalId {
		id := strings.Split(policy.PrincipalId[i], ":")
		if len(id) == 1 {
			ids = append(ids, fmt.Sprintf("qcs::cam::uin/%s:uin/%s", id[0], id[0]))
		}
		if len(id) == 2 {
			// 没有主账号id,设为owner id
			if len(id[0]) == 0 {
				s, _, err := coscli.Service.Get(context.Background())
				if err != nil {
					return errors.Wrap(err, "coscli.Service.Get")
				}
				id[0] = s.Owner.DisplayName
			}
			// 没有子账号，默认和主账号相同
			if len(id[1]) == 0 {
				id[1] = id[0]
			}
			ids = append(ids, fmt.Sprintf("qcs::cam::uin/%s:uin/%s", id[0], id[1]))
		}
		if len(id) > 2 {
			return errors.Wrap(cloudprovider.ErrNotSupported, "Invalida PrincipalId Input")
		}
	}
	principal := map[string][]string{}
	principal["qcs"] = ids
	newStatement.Principal = principal
	newStatement.Effect = policy.Effect
	resources := []string{}
	for i := range policy.ResourcePath {
		resources = append(resources, fmt.Sprintf("qcs::cos:%s:uid/%s:%s%s", b.GetIRegion().GetId(), b.appId, b.getFullName(), policy.ResourcePath[i]))
	}
	newStatement.Resource = resources
	ipEqual := []string{}
	ipNotEqual := []string{}
	for i := range policy.IpEquals {
		ipEqual = append(ipEqual, policy.IpEquals[i])
	}
	for i := range policy.IpNotEquals {
		ipNotEqual = append(ipNotEqual, policy.IpNotEquals[i])
	}
	condition := map[string]map[string]interface{}{}
	newStatement.Condition = condition
	if len(ipEqual) > 0 {
		newStatement.Condition["ip_equal"] = map[string]interface{}{"qcs:ip": ipEqual}
	}
	if len(ipNotEqual) > 0 {
		newStatement.Condition["ip_not_equal"] = map[string]interface{}{"qcs:ip": ipNotEqual}
	}

	if policy.CannedAction == "FullControl" {
		newStatement.Action = []string{"name/cos:*"}
	}
	if policy.CannedAction == "Read" {
		newStatement.Action = cannedReadActions[:]
	}
	if policy.CannedAction == "ReadWrite" {
		newStatement.Action = cannedReadWriteActions[:]
	}
	opts.Statement = append([]cos.BucketStatement{newStatement}, opts.Statement...)

	_, err = coscli.Bucket.PutPolicy(context.Background(), &opts)
	if err != nil {
		log.Errorf("coscli.Bucket.GetACL fail %s", err)
		return errors.Wrapf(err, " coscli.Bucket.PutPolicy(context.Background(), %s)", jsonutils.Marshal(opts).String())
	}
	return nil
}

func (b *SBucket) DeletePolicy(id []string) ([]cloudprovider.SBucketPolicyStatement, error) {
	deletedPolicy := []cloudprovider.SBucketPolicyStatement{}
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		log.Errorf("GetCosClient fail %s", err)
		return nil, errors.Wrap(err, "b.region.GetCosClient(b)")
	}
	result, _, err := coscli.Bucket.GetPolicy(context.Background())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		log.Errorf("coscli.Bucket.GetACL fail %s", err)
		return nil, errors.Wrap(err, "coscli.Bucket.GetPolicy(context.Background())")
	}
	newOpts := cos.BucketPutPolicyOptions{}
	newOpts.Version = result.Version
	newOpts.Principal = result.Principal
	excludeMap := map[int]bool{}
	for i := range id {
		index, err := strconv.Atoi(id[i])
		if err == nil {
			excludeMap[index] = true
		}
	}
	for i := range result.Statement {
		if _, ok := excludeMap[i]; !ok {
			newOpts.Statement = append(newOpts.Statement, result.Statement[i])
		} else {
			deletedPolicy = append(deletedPolicy, cloudprovider.SBucketPolicyStatement{
				Principal: result.Statement[i].Principal,
				Action:    result.Statement[i].Action,
				Effect:    result.Statement[i].Effect,
				Resource:  result.Statement[i].Resource,
				Condition: result.Statement[i].Condition,

				PrincipalId:  getQcsUserId(result.Statement[i].Principal["qcs"]),
				CannedAction: getCannedAction(result.Statement[i].Action),
				ResourcePath: getQcsResourcePath(result.Statement[i].Resource),
			})
		}
	}

	if len(newOpts.Statement) == 0 {
		_, err := coscli.Bucket.DeletePolicy(context.Background())
		if err != nil {
			log.Errorf("coscli.Bucket.DeletePolicy fail %s", err)
			return nil, errors.Wrap(err, "coscli.Bucket.DeletePolicy(context.Background())")
		}
		return deletedPolicy, nil
	}

	_, err = coscli.Bucket.PutPolicy(context.Background(), &newOpts)
	if err != nil {
		log.Errorf("coscli.Bucket.GetACL fail %s", err)
		return nil, errors.Wrapf(err, "coscli.Bucket.PutPolicy(context.Background(), %s)", jsonutils.Marshal(newOpts).String())
	}
	return deletedPolicy, nil
}

func (b *SBucket) GetTags() (map[string]string, error) {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return nil, errors.Wrap(err, "GetCosClient")
	}

	tagresult, _, err := coscli.Bucket.GetTagging(context.Background())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, errors.Wrap(err, "GetTagging")
	}
	result := map[string]string{}
	for i := range tagresult.TagSet {
		result[tagresult.TagSet[i].Key] = tagresult.TagSet[i].Value
	}
	return result, nil
}

func (b *SBucket) SetTags(tags map[string]string, replace bool) error {
	if !replace {
		return cloudprovider.ErrNotSupported
	}
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrapf(err, "b.region.GetCosClient(%s)", b.Name)
	}

	_, err = coscli.Bucket.DeleteTagging(context.Background())
	if err != nil {
		return errors.Wrapf(err, "DeleteTagging")
	}

	if len(tags) == 0 {
		return nil
	}

	input := cos.BucketPutTaggingOptions{}
	for k, v := range tags {
		input.TagSet = append(input.TagSet, cos.BucketTaggingTag{Key: k, Value: v})
	}

	_, err = coscli.Bucket.PutTagging(context.Background(), &input)
	if err != nil {
		return errors.Wrapf(err, "coscli.Bucket.PutTagging(%s)", jsonutils.Marshal(input))
	}
	return nil
}

func (b *SBucket) ListMultipartUploads() ([]cloudprovider.SBucketMultipartUploads, error) {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		log.Errorf("GetCosClient fail %s", err)
		return nil, errors.Wrap(err, "b.region.GetCosClient(b)")
	}
	result := []cloudprovider.SBucketMultipartUploads{}
	input := cos.ListMultipartUploadsOptions{}
	keyMarker := ""
	uploadIDMarker := ""
	for {
		input.KeyMarker = keyMarker
		input.UploadIDMarker = uploadIDMarker
		output, _, err := coscli.Bucket.ListMultipartUploads(context.Background(), &input)
		if err != nil {
			return nil, errors.Wrap(err, " coscli.Bucket.ListMultipartUploads(context.Background(), &input)")
		}
		for i := range output.Uploads {
			temp := cloudprovider.SBucketMultipartUploads{
				ObjectName: output.Uploads[i].Key,
				UploadID:   output.Uploads[i].UploadID,
				Initiator:  output.Uploads[i].Initiator.DisplayName,
			}
			temp.Initiated, _ = timeutils.ParseTimeStr(output.Uploads[i].Initiated)
			result = append(result, temp)
		}
		keyMarker = output.NextKeyMarker
		uploadIDMarker = output.NextUploadIDMarker
		if !output.IsTruncated {
			break
		}
	}

	return result, nil
}
