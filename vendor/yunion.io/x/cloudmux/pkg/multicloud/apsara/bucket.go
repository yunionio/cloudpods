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

package apsara

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SBucket struct {
	multicloud.SBaseBucket
	ApsaraTags

	region *SRegion

	Name         string
	Location     string
	CreationDate time.Time
	StorageClass string

	ExtranetEndpoint string
	IntranetEndpoint string
	DepartmentInfo
}

func (b *SBucket) GetGlobalId() string {
	return b.Name
}

func (b *SBucket) GetName() string {
	return b.Name
}

func (self *SBucket) GetOssClient() (*oss.Client, error) {
	return self.region.GetOssClient()
}

func (b *SBucket) GetAcl() cloudprovider.TBucketACLType {
	acl := b.region.GetBucketAcl(b.Name)
	return cloudprovider.TBucketACLType(acl)
}

func (self *SRegion) GetBucketAcl(bucket string) string {
	params := map[string]string{
		"AccountInfo":      self.GetClient().getAccountInfo(),
		"x-acs-instanceid": bucket,
		"Params":           jsonutils.Marshal(map[string]string{"BucketName": bucket, "acl": "acl"}).String(),
	}
	resp, err := self.ossRequest("GetBucketAcl", params)
	if err != nil {
		return ""
	}
	acl, _ := resp.GetString("Data", "AccessControlPolicy", "AccessControlList", "Grant")
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
	return []cloudprovider.SBucketAccessUrl{
		{
			Url:         fmt.Sprintf("%s.%s", b.Name, b.ExtranetEndpoint),
			Description: "ExtranetEndpoint",
			Primary:     true,
		},
	}
}

func (self *SRegion) GetBucketSize(bucket string, department int) (int64, error) {
	params := map[string]string{
		"Namespace":  "acs_oss_dashboard",
		"MetricName": "MeteringStorageUtilization",
		"Period":     "3600",
		"Dimensions": jsonutils.Marshal([]map[string]string{
			{"BucketName": bucket},
		}).String(),
		"Department": fmt.Sprintf("%d", department),
		"StartTime":  strconv.FormatInt(time.Now().Add(time.Hour*-24*2).Unix()*1000, 10),
		"EndTime":    strconv.FormatInt(time.Now().Unix()*1000, 10),
	}
	resp, err := self.client.metricsRequest("DescribeMetricList", params)
	if err != nil {
		return 0, nil
	}
	datapoints, err := resp.GetString("Datapoints")
	if err != nil {
		return 0, errors.Wrapf(err, "get datapoints")
	}
	obj, err := jsonutils.ParseString(datapoints)
	if err != nil {
		return 0, errors.Wrapf(err, "ParseString")
	}
	data := []struct {
		Timestamp int64
		Value     int64
	}{}
	obj.Unmarshal(&data)
	for i := range data {
		return data[i].Value, nil
	}
	return 0, fmt.Errorf("no storage metric found")
}

func (b *SBucket) GetStats() cloudprovider.SBucketStats {
	ret := cloudprovider.SBucketStats{
		SizeBytes:   -1,
		ObjectCount: -1,
	}
	dep, _ := strconv.Atoi(b.Department)
	size, _ := b.region.GetBucketSize(b.Name, dep)
	if size > 0 {
		ret.SizeBytes = size
	}
	return ret
}

func (self *SRegion) GetBucketCapacity(bucket string, department int) (int64, error) {
	params := map[string]string{
		"Params": jsonutils.Marshal(map[string]string{
			"BucketName": bucket,
			"region":     self.RegionId,
		}).String(),
		// 此参数必传，可以设任意值
		"AccountInfo": self.GetClient().getAccountInfo(),
		"Department":  fmt.Sprintf("%d", department),
	}
	resp, err := self.ossRequest("GetBucketStorageCapacity", params)
	if err != nil {
		return 0, errors.Wrapf(err, "GetBucketStorageCapacity")
	}
	return resp.Int("Data", "BucketUserQos", "StorageCapacity")
}

func (b *SBucket) GetLimit() cloudprovider.SBucketStats {
	ret := cloudprovider.SBucketStats{
		SizeBytes:   -1,
		ObjectCount: -1,
	}
	dep, _ := strconv.Atoi(b.Department)
	capa, _ := b.region.GetBucketCapacity(b.Name, dep)
	if capa > 0 {
		ret.SizeBytes = capa * 1024 * 1024 * 1024
	}
	return ret
}

func (b *SBucket) LimitSupport() cloudprovider.SBucketStats {
	return b.GetLimit()
}

func (b *SBucket) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	osscli, err := b.GetOssClient()
	if err != nil {
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
	osscli, err := b.GetOssClient()
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
	osscli, err := b.GetOssClient()
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
	osscli, err := b.GetOssClient()
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
	osscli, err := b.GetOssClient()
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
	osscli, err := b.GetOssClient()
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
	osscli, err := b.GetOssClient()
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
	osscli, err := b.GetOssClient()
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
	osscli, err := b.GetOssClient()
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
	osscli, err := b.GetOssClient()
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
	osscli, err := b.GetOssClient()
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
	osscli, err := b.GetOssClient()
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
