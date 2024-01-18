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

package aliyun

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SBucket struct {
	multicloud.SBaseBucket
	AliyunTags

	region *SRegion

	Name         string
	Location     string
	CreationDate time.Time
	StorageClass string
}

func (b *SBucket) GetProjectId() string {
	return ""
}

func (b *SBucket) GetGlobalId() string {
	return b.Name
}

func (b *SBucket) GetName() string {
	return b.Name
}

func (b *SBucket) GetAcl() cloudprovider.TBucketACLType {
	acl := cloudprovider.ACLPrivate
	osscli, err := b.getOssClient()
	if err != nil {
		log.Errorf("b.region.GetOssClient fail %s", err)
		return acl
	}
	aclResp, err := osscli.GetBucketACL(b.Name)
	if err != nil {
		log.Errorf("osscli.GetBucketACL fail %s", err)
		return acl
	}
	acl = cloudprovider.TBucketACLType(aclResp.ACL)
	return acl
}

func (b *SBucket) GetLocation() string {
	return b.Location
}

func (b *SBucket) GetIRegion() cloudprovider.ICloudRegion {
	return b.region
}

func (b *SBucket) GetCreatedAt() time.Time {
	return b.CreationDate
}

func (b *SBucket) GetStorageClass() string {
	return b.StorageClass
}

func (b *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	ret := []cloudprovider.SBucketAccessUrl{
		{
			Url:         fmt.Sprintf("%s.%s", b.Name, b.region.getOSSExternalDomain()),
			Description: "ExtranetEndpoint",
			Primary:     true,
		},
		{
			Url:         fmt.Sprintf("%s.%s", b.Name, b.region.getOSSInternalDomain()),
			Description: "IntranetEndpoint",
		},
	}
	return ret
}

func (b *SBucket) GetStats() cloudprovider.SBucketStats {
	stats, err := cloudprovider.GetIBucketStats(b)
	if err != nil {
		log.Errorf("GetStats fail %s", err)
	}
	return stats
}

func (b *SBucket) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		log.Errorf("b.region.GetOssClient fail %s", err)
		return errors.Wrap(err, "b.region.GetOssClient")
	}
	acl, err := str2Acl(string(aclStr))
	if err != nil {
		return errors.Wrap(err, "str2Acl")
	}
	err = osscli.SetBucketACL(b.Name, acl)
	if err != nil {
		return errors.Wrap(err, "SetBucketACL")
	}
	return nil
}

func (b *SBucket) getOssClient() (*oss.Client, error) {
	if b.region.client.GetAccessEnv() == ALIYUN_FINANCE_CLOUDENV {
		osscli, err := b.region.GetOssClient()
		if err != nil {
			return nil, errors.Wrapf(err, "GetOssClient")
		}
		info, err := osscli.GetBucketInfo(b.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "GetBucketInfo")
		}
		if len(info.BucketInfo.ExtranetEndpoint) > 0 {
			return b.region.client.getOssClientByEndpoint(info.BucketInfo.ExtranetEndpoint)
		}
	}
	return b.region.GetOssClient()
}

func (b *SBucket) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	result := cloudprovider.SListObjectResult{}
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return result, errors.Wrap(err, "GetOssClient")
	}
	bucket, err := osscli.Bucket(b.Name)
	if err != nil {
		return result, errors.Wrap(err, "Bucket")
	}
	opts := make([]oss.Option, 0)
	if len(prefix) > 0 {
		opts = append(opts, oss.Prefix(prefix))
	}
	if len(delimiter) > 0 {
		opts = append(opts, oss.Delimiter(delimiter))
	}
	if len(marker) > 0 {
		opts = append(opts, oss.Marker(marker))
	}
	if maxCount > 0 {
		opts = append(opts, oss.MaxKeys(maxCount))
	}
	oResult, err := bucket.ListObjects(opts...)
	if err != nil {
		return result, errors.Wrap(err, "ListObjects")
	}
	result.Objects = make([]cloudprovider.ICloudObject, 0)
	for _, object := range oResult.Objects {
		obj := &SObject{
			bucket: b,
			SBaseCloudObject: cloudprovider.SBaseCloudObject{
				StorageClass: object.StorageClass,
				Key:          object.Key,
				SizeBytes:    object.Size,
				ETag:         object.ETag,
				LastModified: object.LastModified,
			},
		}
		result.Objects = append(result.Objects, obj)
	}
	if oResult.CommonPrefixes != nil {
		result.CommonPrefixes = make([]cloudprovider.ICloudObject, len(oResult.CommonPrefixes))
		for i, commPrefix := range oResult.CommonPrefixes {
			result.CommonPrefixes[i] = &SObject{
				bucket:           b,
				SBaseCloudObject: cloudprovider.SBaseCloudObject{Key: commPrefix},
			}
		}
	}
	result.IsTruncated = oResult.IsTruncated
	result.NextMarker = oResult.NextMarker
	return result, nil
}

func metaOpts(opts []oss.Option, meta http.Header) []oss.Option {
	for k, v := range meta {
		if len(v) == 0 {
			continue
		}
		switch http.CanonicalHeaderKey(k) {
		case cloudprovider.META_HEADER_CONTENT_TYPE:
			opts = append(opts, oss.ContentType(v[0]))
		case cloudprovider.META_HEADER_CONTENT_MD5:
			opts = append(opts, oss.ContentMD5(v[0]))
		case cloudprovider.META_HEADER_CONTENT_LANGUAGE:
			opts = append(opts, oss.ContentLanguage(v[0]))
		case cloudprovider.META_HEADER_CONTENT_ENCODING:
			opts = append(opts, oss.ContentEncoding(v[0]))
		case cloudprovider.META_HEADER_CONTENT_DISPOSITION:
			opts = append(opts, oss.ContentDisposition(v[0]))
		case cloudprovider.META_HEADER_CACHE_CONTROL:
			opts = append(opts, oss.CacheControl(v[0]))
		default:
			opts = append(opts, oss.Meta(http.CanonicalHeaderKey(k), v[0]))
		}
	}
	return opts
}

func (b *SBucket) PutObject(ctx context.Context, key string, input io.Reader, sizeBytes int64, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return errors.Wrap(err, "GetOssClient")
	}
	bucket, err := osscli.Bucket(b.Name)
	if err != nil {
		return errors.Wrap(err, "Bucket")
	}
	opts := make([]oss.Option, 0)
	if sizeBytes > 0 {
		opts = append(opts, oss.ContentLength(sizeBytes))
	}
	if meta != nil {
		opts = metaOpts(opts, meta)
	}
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	acl, err := str2Acl(string(cannedAcl))
	if err != nil {
		return errors.Wrap(err, "")
	}
	opts = append(opts, oss.ObjectACL(acl))
	if len(storageClassStr) > 0 {
		storageClass, err := str2StorageClass(storageClassStr)
		if err != nil {
			return errors.Wrap(err, "str2StorageClass")
		}
		opts = append(opts, oss.ObjectStorageClass(storageClass))
	}
	return bucket.PutObject(key, input, opts...)
}

func (b *SBucket) NewMultipartUpload(ctx context.Context, key string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) (string, error) {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return "", errors.Wrap(err, "GetOssClient")
	}
	bucket, err := osscli.Bucket(b.Name)
	if err != nil {
		return "", errors.Wrap(err, "Bucket")
	}
	opts := make([]oss.Option, 0)
	if meta != nil {
		opts = metaOpts(opts, meta)
	}
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	acl, err := str2Acl(string(cannedAcl))
	if err != nil {
		return "", errors.Wrap(err, "str2Acl")
	}
	opts = append(opts, oss.ObjectACL(acl))
	if len(storageClassStr) > 0 {
		storageClass, err := str2StorageClass(storageClassStr)
		if err != nil {
			return "", errors.Wrap(err, "str2StorageClass")
		}
		opts = append(opts, oss.ObjectStorageClass(storageClass))
	}
	result, err := bucket.InitiateMultipartUpload(key, opts...)
	if err != nil {
		return "", errors.Wrap(err, "bucket.InitiateMultipartUpload")
	}
	return result.UploadID, nil
}

func (b *SBucket) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, input io.Reader, partSize int64, offset, totalSize int64) (string, error) {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return "", errors.Wrap(err, "GetOssClient")
	}
	bucket, err := osscli.Bucket(b.Name)
	if err != nil {
		return "", errors.Wrap(err, "Bucket")
	}
	imur := oss.InitiateMultipartUploadResult{
		Bucket:   b.Name,
		Key:      key,
		UploadID: uploadId,
	}
	part, err := bucket.UploadPart(imur, input, partSize, partIndex)
	if err != nil {
		return "", errors.Wrap(err, "bucket.UploadPart")
	}
	if b.region.client.debug {
		log.Debugf("upload part key:%s uploadId:%s partIndex:%d etag:%s", key, uploadId, partIndex, part.ETag)
	}
	return part.ETag, nil
}

func (b *SBucket) CompleteMultipartUpload(ctx context.Context, key string, uploadId string, partEtags []string) error {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return errors.Wrap(err, "GetOssClient")
	}
	bucket, err := osscli.Bucket(b.Name)
	if err != nil {
		return errors.Wrap(err, "Bucket")
	}
	imur := oss.InitiateMultipartUploadResult{
		Bucket:   b.Name,
		Key:      key,
		UploadID: uploadId,
	}
	parts := make([]oss.UploadPart, len(partEtags))
	for i := range partEtags {
		parts[i] = oss.UploadPart{
			PartNumber: i + 1,
			ETag:       partEtags[i],
		}
	}
	result, err := bucket.CompleteMultipartUpload(imur, parts)
	if err != nil {
		return errors.Wrap(err, "bucket.CompleteMultipartUpload")
	}
	if b.region.client.debug {
		log.Debugf("CompleteMultipartUpload bucket:%s key:%s etag:%s location:%s", result.Bucket, result.Key, result.ETag, result.Location)
	}
	return nil
}

func (b *SBucket) AbortMultipartUpload(ctx context.Context, key string, uploadId string) error {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return errors.Wrap(err, "GetOssClient")
	}
	bucket, err := osscli.Bucket(b.Name)
	if err != nil {
		return errors.Wrap(err, "Bucket")
	}
	imur := oss.InitiateMultipartUploadResult{
		Bucket:   b.Name,
		Key:      key,
		UploadID: uploadId,
	}
	err = bucket.AbortMultipartUpload(imur)
	if err != nil {
		return errors.Wrap(err, "AbortMultipartUpload")
	}
	return nil
}

func (b *SBucket) DeleteObject(ctx context.Context, key string) error {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return errors.Wrap(err, "GetOssClient")
	}
	bucket, err := osscli.Bucket(b.Name)
	if err != nil {
		return errors.Wrap(err, "Bucket")
	}
	err = bucket.DeleteObject(key)
	if err != nil {
		return errors.Wrap(err, "DeleteObject")
	}
	return nil
}

func (b *SBucket) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	if method != "GET" && method != "PUT" && method != "DELETE" {
		return "", errors.Error("unsupported method")
	}
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return "", errors.Wrap(err, "GetOssClient")
	}
	bucket, err := osscli.Bucket(b.Name)
	if err != nil {
		return "", errors.Wrap(err, "Bucket")
	}
	urlStr, err := bucket.SignURL(key, oss.HTTPMethod(method), int64(expire/time.Second))
	if err != nil {
		return "", errors.Wrap(err, "SignURL")
	}
	return urlStr, nil
}

func (b *SBucket) CopyObject(ctx context.Context, destKey string, srcBucket, srcKey string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return errors.Wrap(err, "GetOssClient")
	}
	bucket, err := osscli.Bucket(b.Name)
	if err != nil {
		return errors.Wrap(err, "Bucket")
	}
	opts := make([]oss.Option, 0)
	if meta != nil {
		opts = metaOpts(opts, meta)
	}
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	acl, err := str2Acl(string(cannedAcl))
	if err != nil {
		return errors.Wrap(err, "str2Acl")
	}
	opts = append(opts, oss.ObjectACL(acl))
	if len(storageClassStr) > 0 {
		storageClass, err := str2StorageClass(storageClassStr)
		if err != nil {
			return errors.Wrap(err, "str2StorageClass")
		}
		opts = append(opts, oss.ObjectStorageClass(storageClass))
	}
	_, err = bucket.CopyObjectFrom(srcBucket, srcKey, destKey, opts...)
	if err != nil {
		return errors.Wrap(err, "CopyObjectFrom")
	}
	return nil
}

func (b *SBucket) GetObject(ctx context.Context, key string, rangeOpt *cloudprovider.SGetObjectRange) (io.ReadCloser, error) {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetOssClient")
	}
	bucket, err := osscli.Bucket(b.Name)
	if err != nil {
		return nil, errors.Wrap(err, "Bucket")
	}
	opts := make([]oss.Option, 0)
	if rangeOpt != nil {
		opts = append(opts, oss.NormalizedRange(rangeOpt.String()))
	}
	output, err := bucket.GetObject(key, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "bucket.GetObject")
	}
	return output, nil
}

func (b *SBucket) CopyPart(ctx context.Context, key string, uploadId string, partNumber int, srcBucket string, srcKey string, srcOffset int64, srcLength int64) (string, error) {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return "", errors.Wrap(err, "GetOssClient")
	}
	bucket, err := osscli.Bucket(b.Name)
	if err != nil {
		return "", errors.Wrap(err, "Bucket")
	}
	imur := oss.InitiateMultipartUploadResult{
		Bucket:   b.Name,
		Key:      key,
		UploadID: uploadId,
	}
	opts := make([]oss.Option, 0)
	part, err := bucket.UploadPartCopy(imur, srcBucket, srcKey, srcOffset, srcLength, partNumber, opts...)
	if err != nil {
		return "", errors.Wrap(err, "bucket.UploadPartCopy")
	}
	return part.ETag, nil
}

func (b *SBucket) SetWebsite(websitConf cloudprovider.SBucketWebsiteConf) error {
	if len(websitConf.Index) == 0 {
		return errors.Wrap(cloudprovider.ErrNotSupported, "missing Index")
	}
	if len(websitConf.ErrorDocument) == 0 {
		return errors.Wrap(cloudprovider.ErrNotSupported, "missing ErrorDocument")
	}
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return errors.Wrap(err, "GetOssClient")
	}

	err = osscli.SetBucketWebsite(b.Name, websitConf.Index, websitConf.ErrorDocument)
	if err != nil {
		return errors.Wrapf(err, " osscli.SetBucketWebsite(%s,%s,%s)", b.Name, websitConf.Index, websitConf.ErrorDocument)
	}
	return nil
}

func (b *SBucket) GetWebsiteConf() (cloudprovider.SBucketWebsiteConf, error) {
	result := cloudprovider.SBucketWebsiteConf{}
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return result, errors.Wrap(err, "GetOssClient")
	}
	websiteResult, err := osscli.GetBucketWebsite(b.Name)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchWebsiteConfiguration") {
			return cloudprovider.SBucketWebsiteConf{}, nil
		}
		return result, errors.Wrapf(err, "osscli.GetBucketWebsite(%s)", b.Name)
	}
	result.Index = websiteResult.IndexDocument.Suffix
	result.ErrorDocument = websiteResult.ErrorDocument.Key
	return result, nil
}

func (b *SBucket) DeleteWebSiteConf() error {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return errors.Wrap(err, "GetOssClient")
	}
	log.Infof("to be delete")
	err = osscli.DeleteBucketWebsite(b.Name)
	if err != nil {
		return errors.Wrapf(err, "osscli.DeleteBucketWebsite(%s)", b.Name)
	}
	log.Infof("deleted")
	return nil
}

func (b *SBucket) SetCORS(rules []cloudprovider.SBucketCORSRule) error {
	if len(rules) == 0 {
		return nil
	}
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return errors.Wrap(err, "GetOssClient")
	}
	input := []oss.CORSRule{}
	for i := range rules {
		input = append(input, oss.CORSRule{
			AllowedOrigin: rules[i].AllowedOrigins,
			AllowedMethod: rules[i].AllowedMethods,
			AllowedHeader: rules[i].AllowedHeaders,
			MaxAgeSeconds: rules[i].MaxAgeSeconds,
			ExposeHeader:  rules[i].ExposeHeaders,
		})
	}

	err = osscli.SetBucketCORS(b.Name, input)
	if err != nil {
		return errors.Wrapf(err, "osscli.SetBucketCORS(%s,%s)", b.Name, jsonutils.Marshal(input).String())
	}
	return nil
}

func (b *SBucket) GetCORSRules() ([]cloudprovider.SBucketCORSRule, error) {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetOssClient")
	}
	conf, err := osscli.GetBucketCORS(b.Name)
	if err != nil {
		if !strings.Contains(err.Error(), "NoSuchCORSConfiguration") {
			return nil, errors.Wrapf(err, "osscli.GetBucketCORS(%s)", b.Name)
		}
	}
	result := []cloudprovider.SBucketCORSRule{}
	for i := range conf.CORSRules {
		result = append(result, cloudprovider.SBucketCORSRule{
			AllowedOrigins: conf.CORSRules[i].AllowedOrigin,
			AllowedMethods: conf.CORSRules[i].AllowedMethod,
			AllowedHeaders: conf.CORSRules[i].AllowedHeader,
			MaxAgeSeconds:  conf.CORSRules[i].MaxAgeSeconds,
			ExposeHeaders:  conf.CORSRules[i].ExposeHeader,
			Id:             strconv.Itoa(i),
		})
	}
	return result, nil
}

func (b *SBucket) DeleteCORS() error {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return errors.Wrap(err, "GetOssClient")
	}

	err = osscli.DeleteBucketCORS(b.Name)
	if err != nil {
		return errors.Wrapf(err, "osscli.DeleteBucketCORS(%s)", b.Name)
	}

	return nil
}

func (b *SBucket) SetReferer(conf cloudprovider.SBucketRefererConf) error {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return errors.Wrap(err, "GetOssClient")
	}
	if !conf.Enabled {
		return errors.Wrapf(cloudprovider.ErrNotSupported, "Disable Refer")
	}

	if conf.RefererType == "Black-List" {
		return errors.Wrapf(cloudprovider.ErrNotSupported, "Black List")
	}

	err = osscli.SetBucketReferer(b.Name, conf.DomainList, conf.AllowEmptyRefer)
	if err != nil {
		return errors.Wrapf(err, "osscli.SetBucketReferer(%s,%s,%t)", b.Name, conf.DomainList, conf.AllowEmptyRefer)
	}
	return nil
}

func (b *SBucket) GetReferer() (cloudprovider.SBucketRefererConf, error) {
	result := cloudprovider.SBucketRefererConf{}
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return result, errors.Wrap(err, "GetOssClient")
	}
	refererResult, err := osscli.GetBucketReferer(b.Name)
	if err != nil {
		return result, errors.Wrapf(err, "osscli.GetBucketReferer(%s)", b.Name)
	}
	result = cloudprovider.SBucketRefererConf{
		Enabled:         true,
		RefererType:     "White-List",
		DomainList:      refererResult.RefererList,
		AllowEmptyRefer: refererResult.AllowEmptyReferer,
	}
	return result, nil
}

func toAPICdnArea(area string) string {
	switch area {
	case "domestic":
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
	case "configuring", "checking", "stopping", "deleting":
		return api.CDN_DOMAIN_STATUS_PROCESSING
	case "check_failed", "configure_failed":
		return api.CDN_DOMAIN_STATUS_REJECTED
	default:
		return ""
	}
}

func (b *SBucket) GetCdnDomains() ([]cloudprovider.SCdnDomain, error) {
	bucketExtUrl := fmt.Sprintf("%s.%s", b.Name, b.region.getOSSExternalDomain())
	cdnDomains, err := b.region.client.DescribeDomainsBySource(bucketExtUrl)
	if err != nil {
		return nil, errors.Wrapf(err, " b.region.client.DescribeDomainsBySource(%s)", bucketExtUrl)
	}
	result := []cloudprovider.SCdnDomain{}
	for i := range cdnDomains.DomainsData {
		if cdnDomains.DomainsData[i].Source == bucketExtUrl {
			for j := range cdnDomains.DomainsData[i].Domains.DomainNames {
				area := ""
				domain, _ := b.region.client.GetCDNDomainByName(cdnDomains.DomainsData[i].Domains.DomainNames[j])
				if domain != nil {
					area = domain.Coverage
				}
				result = append(result, cloudprovider.SCdnDomain{
					Domain:     cdnDomains.DomainsData[i].Domains.DomainNames[j],
					Status:     toAPICdnStatus(cdnDomains.DomainsData[i].DomainInfos.DomainInfo[j].Status),
					Cname:      cdnDomains.DomainsData[i].DomainInfos.DomainInfo[j].DomainCname,
					Area:       toAPICdnArea(area),
					Origin:     bucketExtUrl,
					OriginType: api.CDN_DOMAIN_ORIGIN_TYPE_BUCKET,
				})
			}
		}
	}
	return result, nil
}

func (b *SBucket) GetTags() (map[string]string, error) {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetOssClient")
	}

	tagresult, err := osscli.GetBucketTagging(b.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "GetBucketTagging %s", b.Name)
	}
	result := map[string]string{}
	for _, tag := range tagresult.Tags {
		result[tag.Key] = tag.Value
	}
	return result, nil
}

func (b *SBucket) SetTags(tags map[string]string, replace bool) error {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return errors.Wrap(err, "GetOssClient")
	}

	err = osscli.DeleteBucketTagging(b.Name)
	if err != nil {
		return errors.Wrapf(err, "DeleteBucketTagging(%s)", b.Name)
	}

	if len(tags) == 0 {
		return nil
	}

	input := []oss.Tag{}
	for k, v := range tags {
		input = append(input, oss.Tag{Key: k, Value: v})
	}

	err = osscli.SetBucketTagging(b.Name, oss.Tagging{Tags: input})
	if err != nil {
		return errors.Wrapf(err, "osscli.SetBucketTagging(%s)", jsonutils.Marshal(input))
	}
	return nil
}

func (b *SBucket) ListMultipartUploads() ([]cloudprovider.SBucketMultipartUploads, error) {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetOssClient")
	}
	result := []cloudprovider.SBucketMultipartUploads{}

	ossBucket, err := osscli.Bucket(b.Name)
	if err != nil {
		return nil, errors.Wrap(err, "osscli.Bucket(b.Name)")
	}

	keyMarker := oss.KeyMarker("")
	uploadIDMarker := oss.UploadIDMarker("")
	for {
		output, err := ossBucket.ListMultipartUploads(keyMarker, uploadIDMarker)
		if err != nil {
			return nil, errors.Wrap(err, " coscli.Bucket.ListMultipartUploads(context.Background(), &input)")
		}
		for i := range output.Uploads {
			temp := cloudprovider.SBucketMultipartUploads{
				ObjectName: output.Uploads[i].Key,
				UploadID:   output.Uploads[i].UploadID,
				Initiated:  output.Uploads[i].Initiated,
			}
			result = append(result, temp)
		}
		keyMarker = oss.KeyMarker(output.NextKeyMarker)
		uploadIDMarker = oss.UploadIDMarker(output.NextUploadIDMarker)
		if !output.IsTruncated {
			break
		}
	}

	return result, nil
}

type SBucketPolicyStatement struct {
	Version   string                          `json:"Version"`
	Statement []SBucketPolicyStatementDetails `json:"Statement"`
}

type SBucketPolicyStatementDetails struct {
	Action    []string                          `json:"Action"`
	Effect    string                            `json:"Effect"`
	Principal []string                          `json:"Principal"`
	Resource  []string                          `json:"Resource"`
	Condition map[string]map[string]interface{} `json:"Condition"`
}

func (b *SBucket) GetPolicy() ([]cloudprovider.SBucketPolicyStatement, error) {
	policies, err := b.getPolicy()
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			return []cloudprovider.SBucketPolicyStatement{}, nil
		}
		return nil, errors.Wrap(err, "getPolicy")
	}
	return b.localPolicyToCloudprovider(policies), nil
}

func (b *SBucket) getPolicy() ([]SBucketPolicyStatementDetails, error) {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetOssClient")
	}
	policies := []SBucketPolicyStatementDetails{}
	resStr, err := osscli.GetBucketPolicy(b.Name)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchBucketPolicy") {
			return policies, nil
		}
		return nil, errors.Wrap(err, "GetBucketPolicy")
	}
	obj, err := jsonutils.Parse([]byte(resStr))
	if err != nil {
		return nil, errors.Wrap(err, "Parse resStr")
	}
	return policies, obj.Unmarshal(&policies, "Statement")
}

func (b *SBucket) SetPolicy(policy cloudprovider.SBucketPolicyStatementInput) error {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return errors.Wrap(err, "GetOssClient")
	}
	policies, err := b.getPolicy()
	if err != nil {
		return errors.Wrap(err, "getPolicy")
	}
	param := SBucketPolicyStatement{}
	param.Version = "1"
	param.Statement = policies
	resources := []string{}
	for i := range policy.ResourcePath {
		resources = append(resources, fmt.Sprintf("acs:oss:*:%s:%s%s", b.region.client.GetAccountId(), b.Name, policy.ResourcePath[i]))
	}
	ids := []string{}
	for i := range policy.PrincipalId {
		id := strings.Split(policy.PrincipalId[i], ":")
		if len(id) == 1 {
			ids = append(ids, "*")
		}
		if len(id) == 2 {
			// 没有子账号，默认和主账号相同
			if len(id[1]) == 0 {
				id[1] = "*"
			}
			ids = append(ids, id[1])
		}
		if len(id) > 2 {
			return errors.Wrap(cloudprovider.ErrNotSupported, "Invalida PrincipalId Input")
		}
	}

	param.Statement = append(param.Statement, SBucketPolicyStatementDetails{
		Resource: resources,
		// Principal: policy.PrincipalId,
		Principal: ids,
		Effect:    policy.Effect,
		Condition: policy.Condition,
		Action:    b.cannedActionToAction(policy.CannedAction),
	})
	err = osscli.SetBucketPolicy(b.Name, jsonutils.Marshal(param).String())
	if err != nil {
		return errors.Wrap(err, "SetBucketPolicy")
	}
	return nil
}

func (b *SBucket) DeletePolicy(id []string) ([]cloudprovider.SBucketPolicyStatement, error) {
	osscli, err := b.region.GetOssClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetOssClient")
	}

	param := SBucketPolicyStatement{}
	param.Version = "1"
	param.Statement = []SBucketPolicyStatementDetails{}

	policies, err := b.getPolicy()
	if err != nil {
		return nil, errors.Wrap(err, "getPolicy")
	}
	for i, policy := range policies {
		if utils.IsInStringArray(fmt.Sprintf("%d", i), id) {
			continue
		}
		param.Statement = append(param.Statement, policy)
	}
	err = osscli.DeleteBucketPolicy(b.Name)
	if err != nil {
		return nil, errors.Wrap(err, "DeleteBucketPolicy")
	}
	if len(param.Statement) > 0 {
		err = osscli.SetBucketPolicy(b.Name, jsonutils.Marshal(param).String())
		if err != nil {
			return nil, errors.Wrap(err, "SetBucketPolicy")
		}
	}
	return b.localPolicyToCloudprovider(param.Statement), nil
}

func (b *SBucket) localPolicyToCloudprovider(policies []SBucketPolicyStatementDetails) []cloudprovider.SBucketPolicyStatement {
	res := []cloudprovider.SBucketPolicyStatement{}
	for i, policy := range policies {
		res = append(res, cloudprovider.SBucketPolicyStatement{
			Principal:    map[string][]string{"acs": policy.Principal},
			PrincipalId:  getLocalPrincipalId(policy.Principal),
			Action:       policy.Action,
			Effect:       policy.Effect,
			Resource:     b.getResourcePaths(policy.Resource),
			ResourcePath: b.getResourcePaths(policy.Resource),
			Condition:    policy.Condition,
			CannedAction: b.actionToCannedAction(policy.Action),
			Id:           fmt.Sprintf("%d", i),
		})
	}
	return res
}

var readActions = []string{
	"oss:GetObject",
	"oss:GetObjectAcl",
	"oss:ListObjects",
	"oss:RestoreObject",
	"oss:GetVodPlaylist",
	"oss:ListObjectVersions",
	"oss:GetObjectVersion",
	"oss:GetObjectVersionAcl",
	"oss:RestoreObjectVersion",
}

var readWriteActions = []string{
	"oss:GetObject",
	"oss:PutObject",
	"oss:GetObjectAcl",
	"oss:PutObjectAcl",
	"oss:ListObjects",
	"oss:AbortMultipartUpload",
	"oss:ListParts",
	"oss:RestoreObject",
	"oss:GetVodPlaylist",
	"oss:PostVodPlaylist",
	"oss:PublishRtmpStream",
	"oss:ListObjectVersions",
	"oss:GetObjectVersion",
	"oss:GetObjectVersionAcl",
	"oss:RestoreObjectVersion",
}

var fullControlActions = []string{"oss:*"}

func (b *SBucket) cannedActionToAction(s string) []string {
	switch s {
	case "Read":
		return readActions
	case "ReadWrite":
		return readWriteActions
	case "FullControl":
		return fullControlActions
	default:
		return nil
	}
}

func (b *SBucket) actionToCannedAction(actions []string) string {
	if len(actions) == len(readActions) {
		for _, action := range actions {
			if !utils.IsInStringArray(action, readActions) {
				return ""
			}
		}
		return "Read"
	} else if len(actions) == len(fullControlActions) {
		for _, action := range actions {
			if !utils.IsInStringArray(action, fullControlActions) {
				return ""
			}
		}
		return "FullControl"
	} else if len(actions) == len(readWriteActions) {
		for _, action := range actions {
			if !utils.IsInStringArray(action, readWriteActions) {
				return ""
			}
		}
		return "ReadWrite"
	} else {
		return ""
	}
}

func (b *SBucket) getResourcePaths(paths []string) []string {
	res := []string{}
	for _, path := range paths {
		res = append(res, strings.TrimPrefix(path, b.Name))
	}
	return res
}

func getLocalPrincipalId(principals []string) []string {
	res := []string{}
	for _, principal := range principals {
		res = append(res, ":"+principal)
	}
	return res
}
