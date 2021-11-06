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

package hcso

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/s3cli"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/obs"
)

type SBucket struct {
	multicloud.SBaseBucket
	multicloud.HuaweiTags

	region *SRegion

	Name         string
	Location     string
	CreationDate time.Time
}

func (b *SBucket) GetProjectId() string {
	resp, err := b.region.HeadBucket(b.Name)
	if err != nil {
		return ""
	}
	epid, _ := resp.ResponseHeaders["epid"]
	if len(epid) > 0 {
		return epid[0]
	}
	return ""
}

func (b *SBucket) GetGlobalId() string {
	return b.Name
}

func (b *SBucket) GetName() string {
	return b.Name
}

func (b *SBucket) GetLocation() string {
	return b.Location
}

func (b *SBucket) GetIRegion() cloudprovider.ICloudRegion {
	return b.region
}

func (b *SBucket) GetCreateAt() time.Time {
	return b.CreationDate
}

/*
 service returned error: Status=405 Method Not Allowed, Code=MethodNotAllowed,
 Message=The specified method is not allowed against this resource.,
 RequestId=00000175B0E9D138440B9EF092DF8A7A
https://support.huaweicloud.com/productdesc-modelarts/modelarts_01_0017.html
*/
func (b *SBucket) GetStorageClass() string {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		log.Errorf("b.region.getOBSClient error %s", err)
		return ""
	}
	output, err := obscli.GetBucketStoragePolicy(b.Name)
	if err != nil {
		log.Errorf("obscli.GetBucketStoragePolicy error %s", err)
		return ""
	}
	return output.StorageClass
}

func obsAcl2CannedAcl(acls []obs.Grant) cloudprovider.TBucketACLType {
	switch {
	case len(acls) == 1:
		if acls[0].Grantee.URI == "" && acls[0].Permission == s3cli.PERMISSION_FULL_CONTROL {
			return cloudprovider.ACLPrivate
		}
	case len(acls) == 2:
		for _, g := range acls {
			if g.Grantee.URI == s3cli.GRANTEE_GROUP_URI_AUTH_USERS && g.Permission == s3cli.PERMISSION_READ {
				return cloudprovider.ACLAuthRead
			}
			if g.Grantee.URI == s3cli.GRANTEE_GROUP_URI_ALL_USERS && g.Permission == s3cli.PERMISSION_READ {
				return cloudprovider.ACLPublicRead
			}
		}
	case len(acls) == 3:
		for _, g := range acls {
			if g.Grantee.URI == s3cli.GRANTEE_GROUP_URI_ALL_USERS && g.Permission == s3cli.PERMISSION_WRITE {
				return cloudprovider.ACLPublicReadWrite
			}
		}
	}
	return cloudprovider.ACLUnknown
}

func (b *SBucket) GetAcl() cloudprovider.TBucketACLType {
	acl := cloudprovider.ACLPrivate
	obscli, err := b.region.getOBSClient()
	if err != nil {
		log.Errorf("b.region.getOBSClient error %s", err)
		return acl
	}
	output, err := obscli.GetBucketAcl(b.Name)
	if err != nil {
		log.Errorf("obscli.GetBucketAcl error %s", err)
		return acl
	}
	acl = obsAcl2CannedAcl(output.Grants)
	return acl
}

func (b *SBucket) SetAcl(acl cloudprovider.TBucketACLType) error {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return errors.Wrap(err, "b.region.getOBSClient")
	}
	input := &obs.SetBucketAclInput{}
	input.Bucket = b.Name
	input.ACL = obs.AclType(string(acl))
	_, err = obscli.SetBucketAcl(input)
	if err != nil {
		return errors.Wrap(err, "obscli.SetBucketAcl")
	}
	return nil
}

func (b *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	return []cloudprovider.SBucketAccessUrl{
		{
			Url:         fmt.Sprintf("https://%s.%s", b.Name, b.region.getOBSEndpoint()),
			Description: "bucket url",
			Primary:     true,
		},
		{
			Url:         fmt.Sprintf("https://%s/%s", b.region.getOBSEndpoint(), b.Name),
			Description: "obs url",
		},
	}
}

func (b *SBucket) GetStats() cloudprovider.SBucketStats {
	stats := cloudprovider.SBucketStats{}
	obscli, err := b.region.getOBSClient()
	if err != nil {
		log.Errorf("b.region.getOBSClient error %s", err)
		stats.SizeBytes = -1
		stats.ObjectCount = -1
		return stats
	}
	output, err := obscli.GetBucketStorageInfo(b.Name)
	if err != nil {
		log.Errorf("obscli.GetBucketStorageInfo error %s", err)
		stats.SizeBytes = -1
		stats.ObjectCount = -1
		return stats
	}
	stats.SizeBytes = output.Size
	stats.ObjectCount = output.ObjectNumber
	return stats
}

func (b *SBucket) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	result := cloudprovider.SListObjectResult{}
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return result, errors.Wrap(err, "GetOBSClient")
	}
	input := &obs.ListObjectsInput{}
	input.Bucket = b.Name
	if len(prefix) > 0 {
		input.Prefix = prefix
	}
	if len(marker) > 0 {
		input.Marker = marker
	}
	if len(delimiter) > 0 {
		input.Delimiter = delimiter
	}
	if maxCount > 0 {
		input.MaxKeys = maxCount
	}
	oResult, err := obscli.ListObjects(input)
	if err != nil {
		return result, errors.Wrap(err, "ListObjects")
	}
	result.Objects = make([]cloudprovider.ICloudObject, 0)
	for _, object := range oResult.Contents {
		obj := &SObject{
			bucket: b,
			SBaseCloudObject: cloudprovider.SBaseCloudObject{
				StorageClass: string(object.StorageClass),
				Key:          object.Key,
				SizeBytes:    object.Size,
				ETag:         object.ETag,
				LastModified: object.LastModified,
			},
		}
		result.Objects = append(result.Objects, obj)
	}
	if oResult.CommonPrefixes != nil {
		result.CommonPrefixes = make([]cloudprovider.ICloudObject, 0)
		for _, commonPrefix := range oResult.CommonPrefixes {
			obj := &SObject{
				bucket: b,
				SBaseCloudObject: cloudprovider.SBaseCloudObject{
					Key: commonPrefix,
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
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return errors.Wrap(err, "GetOBSClient")
	}
	input := &obs.PutObjectInput{}
	input.Bucket = b.Name
	input.Key = key
	input.Body = reader

	if sizeBytes > 0 {
		input.ContentLength = sizeBytes
	}
	if len(storageClassStr) > 0 {
		input.StorageClass, err = str2StorageClass(storageClassStr)
		if err != nil {
			return err
		}
	}
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	input.ACL = obs.AclType(string(cannedAcl))
	if meta != nil {
		val := meta.Get(cloudprovider.META_HEADER_CONTENT_TYPE)
		if len(val) > 0 {
			input.ContentType = val
		}
		val = meta.Get(cloudprovider.META_HEADER_CONTENT_MD5)
		if len(val) > 0 {
			input.ContentMD5 = val
		}
		extraMeta := make(map[string]string)
		for k, v := range meta {
			if utils.IsInStringArray(k, []string{
				cloudprovider.META_HEADER_CONTENT_TYPE,
				cloudprovider.META_HEADER_CONTENT_MD5,
			}) {
				continue
			}
			if len(v[0]) > 0 {
				extraMeta[k] = v[0]
			}
		}
		input.Metadata = extraMeta
	}
	_, err = obscli.PutObject(input)
	if err != nil {
		return errors.Wrap(err, "PutObject")
	}
	return nil
}

func (b *SBucket) NewMultipartUpload(ctx context.Context, key string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) (string, error) {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return "", errors.Wrap(err, "GetOBSClient")
	}

	input := &obs.InitiateMultipartUploadInput{}
	input.Bucket = b.Name
	input.Key = key
	if meta != nil {
		val := meta.Get(cloudprovider.META_HEADER_CONTENT_TYPE)
		if len(val) > 0 {
			input.ContentType = val
		}
		extraMeta := make(map[string]string)
		for k, v := range meta {
			if utils.IsInStringArray(k, []string{
				cloudprovider.META_HEADER_CONTENT_TYPE,
			}) {
				continue
			}
			if len(v[0]) > 0 {
				extraMeta[k] = v[0]
			}
		}
		input.Metadata = extraMeta
	}
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	input.ACL = obs.AclType(string(cannedAcl))
	if len(storageClassStr) > 0 {
		input.StorageClass, err = str2StorageClass(storageClassStr)
		if err != nil {
			return "", errors.Wrap(err, "str2StorageClass")
		}
	}
	output, err := obscli.InitiateMultipartUpload(input)
	if err != nil {
		return "", errors.Wrap(err, "InitiateMultipartUpload")
	}

	return output.UploadId, nil
}

func (b *SBucket) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, part io.Reader, partSize int64, offset, totalSize int64) (string, error) {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return "", errors.Wrap(err, "GetOBSClient")
	}

	input := &obs.UploadPartInput{}
	input.Bucket = b.Name
	input.Key = key
	input.UploadId = uploadId
	input.PartNumber = partIndex
	input.PartSize = partSize
	input.Body = part
	output, err := obscli.UploadPart(input)
	if err != nil {
		return "", errors.Wrap(err, "UploadPart")
	}

	return output.ETag, nil
}

func (b *SBucket) CompleteMultipartUpload(ctx context.Context, key string, uploadId string, partEtags []string) error {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return errors.Wrap(err, "GetOBSClient")
	}
	input := &obs.CompleteMultipartUploadInput{}
	input.Bucket = b.Name
	input.Key = key
	input.UploadId = uploadId
	parts := make([]obs.Part, len(partEtags))
	for i := range partEtags {
		parts[i] = obs.Part{
			PartNumber: i + 1,
			ETag:       partEtags[i],
		}
	}
	input.Parts = parts
	_, err = obscli.CompleteMultipartUpload(input)
	if err != nil {
		return errors.Wrap(err, "CompleteMultipartUpload")
	}

	return nil
}

func (b *SBucket) AbortMultipartUpload(ctx context.Context, key string, uploadId string) error {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return errors.Wrap(err, "GetOBSClient")
	}

	input := &obs.AbortMultipartUploadInput{}
	input.Bucket = b.Name
	input.Key = key
	input.UploadId = uploadId

	_, err = obscli.AbortMultipartUpload(input)
	if err != nil {
		return errors.Wrap(err, "AbortMultipartUpload")
	}

	return nil
}

func (b *SBucket) DeleteObject(ctx context.Context, key string) error {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return errors.Wrap(err, "GetOBSClient")
	}
	input := &obs.DeleteObjectInput{}
	input.Bucket = b.Name
	input.Key = key
	_, err = obscli.DeleteObject(input)
	if err != nil {
		return errors.Wrap(err, "DeleteObject")
	}
	return nil
}

func (b *SBucket) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return "", errors.Wrap(err, "GetOBSClient")
	}
	input := obs.CreateSignedUrlInput{}
	input.Bucket = b.Name
	input.Key = key
	input.Expires = int(expire / time.Second)
	switch method {
	case "GET":
		input.Method = obs.HttpMethodGet
	case "PUT":
		input.Method = obs.HttpMethodPut
	case "DELETE":
		input.Method = obs.HttpMethodDelete
	default:
		return "", errors.Error("unsupported method")
	}
	output, err := obscli.CreateSignedUrl(&input)
	return output.SignedUrl, nil
}

func (b *SBucket) LimitSupport() cloudprovider.SBucketStats {
	return cloudprovider.SBucketStats{
		SizeBytes:   1,
		ObjectCount: -1,
	}
}

func (b *SBucket) GetLimit() cloudprovider.SBucketStats {
	stats := cloudprovider.SBucketStats{}
	obscli, err := b.region.getOBSClient()
	if err != nil {
		log.Errorf("getOBSClient error %s", err)
		return stats
	}
	output, err := obscli.GetBucketQuota(b.Name)
	if err != nil {
		return stats
	}
	stats.SizeBytes = output.Quota
	return stats
}

func (b *SBucket) SetLimit(limit cloudprovider.SBucketStats) error {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return errors.Wrap(err, "getOBSClient")
	}
	input := &obs.SetBucketQuotaInput{}
	input.Bucket = b.Name
	input.Quota = limit.SizeBytes
	_, err = obscli.SetBucketQuota(input)
	if err != nil {
		return errors.Wrap(err, "SetBucketQuota")
	}
	return nil
}

func (b *SBucket) CopyObject(ctx context.Context, destKey string, srcBucket, srcKey string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return errors.Wrap(err, "GetOBSClient")
	}
	input := &obs.CopyObjectInput{}
	input.Bucket = b.Name
	input.Key = destKey
	input.CopySourceBucket = srcBucket
	input.CopySourceKey = srcKey
	if len(storageClassStr) > 0 {
		input.StorageClass, err = str2StorageClass(storageClassStr)
		if err != nil {
			return err
		}
	}
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	input.ACL = obs.AclType(string(cannedAcl))
	if meta != nil {
		val := meta.Get(cloudprovider.META_HEADER_CONTENT_TYPE)
		if len(val) > 0 {
			input.ContentType = val
		}
		extraMeta := make(map[string]string)
		for k, v := range meta {
			if utils.IsInStringArray(k, []string{
				cloudprovider.META_HEADER_CONTENT_TYPE,
			}) {
				continue
			}
			if len(v[0]) > 0 {
				extraMeta[k] = v[0]
			}
		}
		input.Metadata = extraMeta
		input.MetadataDirective = obs.ReplaceMetadata
	} else {
		input.MetadataDirective = obs.CopyMetadata
	}
	_, err = obscli.CopyObject(input)
	if err != nil {
		return errors.Wrap(err, "obscli.CopyObject")
	}
	return nil
}

func (b *SBucket) GetObject(ctx context.Context, key string, rangeOpt *cloudprovider.SGetObjectRange) (io.ReadCloser, error) {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetOBSClient")
	}
	input := &obs.GetObjectInput{}
	input.Bucket = b.Name
	input.Key = key
	if rangeOpt != nil {
		input.RangeStart = rangeOpt.Start
		input.RangeEnd = rangeOpt.End
	}
	output, err := obscli.GetObject(input)
	if err != nil {
		return nil, errors.Wrap(err, "obscli.GetObject")
	}
	return output.Body, nil
}

func (b *SBucket) CopyPart(ctx context.Context, key string, uploadId string, partIndex int, srcBucket string, srcKey string, srcOffset int64, srcLength int64) (string, error) {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return "", errors.Wrap(err, "GetOBSClient")
	}
	input := &obs.CopyPartInput{}
	input.Bucket = b.Name
	input.Key = key
	input.UploadId = uploadId
	input.PartNumber = partIndex
	input.CopySourceBucket = srcBucket
	input.CopySourceKey = srcKey
	input.CopySourceRangeStart = srcOffset
	input.CopySourceRangeEnd = srcOffset + srcLength - 1
	output, err := obscli.CopyPart(input)
	if err != nil {
		return "", errors.Wrap(err, "CopyPart")
	}
	return output.ETag, nil
}

func (b *SBucket) SetWebsite(websitConf cloudprovider.SBucketWebsiteConf) error {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return errors.Wrap(err, "GetOBSClient")
	}

	obsWebConf := obs.SetBucketWebsiteConfigurationInput{}
	obsWebConf.Bucket = b.Name
	obsWebConf.BucketWebsiteConfiguration = obs.BucketWebsiteConfiguration{
		IndexDocument: obs.IndexDocument{Suffix: websitConf.Index},
		ErrorDocument: obs.ErrorDocument{Key: websitConf.ErrorDocument},
	}
	_, err = obscli.SetBucketWebsiteConfiguration(&obsWebConf)
	if err != nil {
		return errors.Wrap(err, "obscli.SetBucketWebsiteConfiguration(&obsWebConf)")
	}
	return nil
}

func (b *SBucket) GetWebsiteConf() (cloudprovider.SBucketWebsiteConf, error) {
	result := cloudprovider.SBucketWebsiteConf{}
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return result, errors.Wrap(err, "GetOBSClient")
	}
	out, err := obscli.GetBucketWebsiteConfiguration(b.Name)
	if out == nil {
		return result, nil
	}
	result.Index = out.IndexDocument.Suffix
	result.ErrorDocument = out.ErrorDocument.Key
	endpoint := b.region.client.endpoints.GetEndpoint("obs-website", b.region.GetId())
	result.Url = fmt.Sprintf("https://%s.%s", b.Name, endpoint)
	return result, nil
}

func (b *SBucket) DeleteWebSiteConf() error {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return errors.Wrap(err, "GetOBSClient")
	}
	_, err = obscli.DeleteBucketWebsiteConfiguration(b.Name)
	if err != nil {
		return errors.Wrapf(err, "obscli.DeleteBucketWebsiteConfiguration(%s)", b.Name)
	}
	return nil
}

func (b *SBucket) SetCORS(rules []cloudprovider.SBucketCORSRule) error {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return errors.Wrap(err, "GetOBSClient")
	}
	opts := []obs.CorsRule{}
	for i := range rules {
		opts = append(opts, obs.CorsRule{
			AllowedOrigin: rules[i].AllowedOrigins,
			AllowedMethod: rules[i].AllowedMethods,
			AllowedHeader: rules[i].AllowedHeaders,
			MaxAgeSeconds: rules[i].MaxAgeSeconds,
			ExposeHeader:  rules[i].ExposeHeaders,
		})
	}

	input := obs.SetBucketCorsInput{}
	input.Bucket = b.Name
	input.BucketCors.CorsRules = opts
	_, err = obscli.SetBucketCors(&input)
	if err != nil {
		return errors.Wrapf(err, "obscli.SetBucketCors(%s)", jsonutils.Marshal(input).String())
	}
	return nil
}

func (b *SBucket) GetCORSRules() ([]cloudprovider.SBucketCORSRule, error) {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetOBSClient")
	}
	conf, err := obscli.GetBucketCors(b.Name)
	if err != nil {
		if !strings.Contains(err.Error(), "NoSuchCORSConfiguration") {
			return nil, errors.Wrapf(err, "obscli.GetBucketCors(%s)", b.Name)
		}
	}
	if conf == nil {
		return nil, nil
	}
	result := []cloudprovider.SBucketCORSRule{}
	for i := range conf.CorsRules {
		result = append(result, cloudprovider.SBucketCORSRule{
			AllowedOrigins: conf.CorsRules[i].AllowedOrigin,
			AllowedMethods: conf.CorsRules[i].AllowedMethod,
			AllowedHeaders: conf.CorsRules[i].AllowedHeader,
			MaxAgeSeconds:  conf.CorsRules[i].MaxAgeSeconds,
			ExposeHeaders:  conf.CorsRules[i].ExposeHeader,
			Id:             strconv.Itoa(i),
		})
	}
	return result, nil
}

func (b *SBucket) DeleteCORS() error {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return errors.Wrap(err, "GetOBSClient")
	}

	_, err = obscli.DeleteBucketCors(b.Name)
	if err != nil {
		return errors.Wrapf(err, "obscli.DeleteBucketCors(%s)", b.Name)
	}
	return nil
}

func (b *SBucket) GetTags() (map[string]string, error) {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetOBSClient")
	}
	tagresult, err := obscli.GetBucketTagging(b.Name)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "osscli.GetBucketTagging(%s)", b.Name)
	}
	result := map[string]string{}
	for i := range tagresult.Tags {
		result[tagresult.Tags[i].Key] = tagresult.Tags[i].Value
	}
	return result, nil
}

func (b *SBucket) SetTags(tags map[string]string, replace bool) error {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return errors.Wrap(err, "GetOBSClient")
	}

	_, err = obscli.DeleteBucketTagging(b.Name)
	if err != nil {
		return errors.Wrapf(err, "DeleteBucketTagging")
	}

	if len(tags) == 0 {
		return nil
	}

	input := obs.SetBucketTaggingInput{BucketTagging: obs.BucketTagging{}}
	input.Bucket = b.Name
	for k, v := range tags {
		input.BucketTagging.Tags = append(input.BucketTagging.Tags, obs.Tag{Key: k, Value: v})
	}

	_, err = obscli.SetBucketTagging(&input)
	if err != nil {
		return errors.Wrapf(err, "obscli.SetBucketTagging(%s)", jsonutils.Marshal(input).String())
	}
	return nil
}

func (b *SBucket) ListMultipartUploads() ([]cloudprovider.SBucketMultipartUploads, error) {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetOBSClient")
	}
	result := []cloudprovider.SBucketMultipartUploads{}

	input := obs.ListMultipartUploadsInput{Bucket: b.Name}
	keyMarker := ""
	uploadIDMarker := ""
	for {
		if len(keyMarker) > 0 {
			input.KeyMarker = keyMarker
		}
		if len(uploadIDMarker) > 0 {
			input.UploadIdMarker = uploadIDMarker
		}

		output, err := obscli.ListMultipartUploads(&input)
		if err != nil {
			return nil, errors.Wrap(err, " coscli.Bucket.ListMultipartUploads(context.Background(), &input)")
		}
		for i := range output.Uploads {
			temp := cloudprovider.SBucketMultipartUploads{
				ObjectName: output.Uploads[i].Key,
				UploadID:   output.Uploads[i].UploadId,
				Initiator:  output.Uploads[i].Initiator.DisplayName,
				Initiated:  output.Uploads[i].Initiated,
			}
			result = append(result, temp)
		}
		keyMarker = output.NextKeyMarker
		uploadIDMarker = output.NextUploadIdMarker
		if !output.IsTruncated {
			break
		}
	}

	return result, nil
}
