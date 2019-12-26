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
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SBucket struct {
	multicloud.SBaseBucket

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
	osscli, err := b.region.GetOssClient()
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

func (b *SBucket) GetCreateAt() time.Time {
	return b.CreationDate
}

func (b *SBucket) GetStorageClass() string {
	return b.StorageClass
}

func (b *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	return []cloudprovider.SBucketAccessUrl{
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

func (b *SBucket) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, input io.Reader, partSize int64) (string, error) {
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
	if b.region.client.Debug {
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
	if b.region.client.Debug {
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
