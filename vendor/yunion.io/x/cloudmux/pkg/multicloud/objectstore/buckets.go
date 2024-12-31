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

package objectstore

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/s3cli"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SBucket struct {
	multicloud.SBaseBucket
	multicloud.STagBase

	client IBucketProvider

	Name         string
	Location     string
	CreatedAt    time.Time
	StorageClass string
}

func (bucket *SBucket) GetIBucketProvider() IBucketProvider {
	return bucket.client
}

func (bucket *SBucket) GetId() string {
	return ""
}

func (bucket *SBucket) GetStatus() string {
	return api.BUCKET_STATUS_READY
}

func (bucket *SBucket) GetProjectId() string {
	return ""
}

func (bucket *SBucket) IsEmulated() bool {
	return false
}

func (bucket *SBucket) Refresh() error {
	return nil
}

func (bucket *SBucket) GetGlobalId() string {
	return bucket.Name
}

func (bucket *SBucket) GetName() string {
	return bucket.Name
}

func (bucket *SBucket) GetAcl() cloudprovider.TBucketACLType {
	acl, _ := bucket.client.GetIBucketAcl(bucket.Name)
	return acl
}

func (bucket *SBucket) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	return bucket.client.SetIBucketAcl(bucket.Name, aclStr)
}

func (bucket *SBucket) GetLocation() string {
	return bucket.Location
}

func (bucket *SBucket) GetIRegion() cloudprovider.ICloudRegion {
	return bucket.client
}

func (bucket *SBucket) GetCreatedAt() time.Time {
	return bucket.CreatedAt
}

func (bucket *SBucket) GetStorageClass() string {
	return bucket.StorageClass
}

func (bucket *SBucket) GetStats() cloudprovider.SBucketStats {
	stats, _ := cloudprovider.GetIBucketStats(bucket)
	return stats
}

func joinPath(ep, path string) string {
	return strings.TrimRight(ep, "/") + "/" + strings.TrimLeft(path, "/")
}

func (bucket *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	return []cloudprovider.SBucketAccessUrl{
		{
			Url:         joinPath(bucket.client.GetEndpoint(), bucket.Name),
			Description: fmt.Sprintf("%s", bucket.Location),
			Primary:     true,
		},
	}
}

func (bucket *SBucket) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	ret := cloudprovider.SListObjectResult{}
	result, err := bucket.client.S3Client().ListObjectsQuery(bucket.Name, prefix, marker, delimiter, maxCount)
	if err != nil {
		return ret, errors.Wrap(err, "ListObjectsQuery")
	}
	ret.NextMarker = result.NextMarker
	ret.IsTruncated = result.IsTruncated
	ret.CommonPrefixes = make([]cloudprovider.ICloudObject, len(result.CommonPrefixes))
	for i := range result.CommonPrefixes {
		ret.CommonPrefixes[i] = &SObject{
			bucket: bucket,
			SBaseCloudObject: cloudprovider.SBaseCloudObject{
				Key: result.CommonPrefixes[i].Prefix,
			},
		}
	}
	ret.Objects = make([]cloudprovider.ICloudObject, len(result.Contents))
	for i := range result.Contents {
		object := result.Contents[i]
		ret.Objects[i] = &SObject{
			bucket: bucket,
			SBaseCloudObject: cloudprovider.SBaseCloudObject{
				StorageClass: object.StorageClass,
				Key:          object.Key,
				SizeBytes:    object.Size,
				ETag:         object.ETag,
				LastModified: object.LastModified,
			},
		}
	}
	return ret, nil
}

func (bucket *SBucket) GetIObjects(prefix string, isRecursive bool) ([]cloudprovider.ICloudObject, error) {
	doneCh := make(chan struct{})
	defer close(doneCh)

	ret := make([]cloudprovider.ICloudObject, 0)
	objectCh := bucket.client.S3Client().ListObjects(bucket.Name, prefix, isRecursive, doneCh)
	for object := range objectCh {
		if object.Err != nil {
			return nil, errors.Wrap(object.Err, "ListObjects")
		}
		if !isRecursive && prefix == object.Key {
			continue
		}
		obj := &SObject{
			bucket: bucket,
			SBaseCloudObject: cloudprovider.SBaseCloudObject{
				StorageClass: object.StorageClass,
				Key:          object.Key,
				SizeBytes:    object.Size,
				ETag:         object.ETag,
				LastModified: object.LastModified,
			},
		}
		ret = append(ret, obj)
	}
	return ret, nil
}

func metaInitOptions(meta http.Header) s3cli.PutObjectOptions {
	opts := s3cli.PutObjectOptions{}
	if meta != nil {
		val := meta.Get(cloudprovider.META_HEADER_CONTENT_TYPE)
		if len(val) > 0 {
			opts.ContentType = val
		}
		val = meta.Get(cloudprovider.META_HEADER_CACHE_CONTROL)
		if len(val) > 0 {
			opts.CacheControl = val
		}
		val = meta.Get(cloudprovider.META_HEADER_CONTENT_DISPOSITION)
		if len(val) > 0 {
			opts.ContentDisposition = val
		}
		val = meta.Get(cloudprovider.META_HEADER_CONTENT_ENCODING)
		if len(val) > 0 {
			opts.ContentEncoding = val
		}
		val = meta.Get(cloudprovider.META_HEADER_CONTENT_LANGUAGE)
		if len(val) > 0 {
			opts.ContentLanguage = val
		}
		userMeta := make(map[string]string)
		for k, v := range meta {
			if utils.IsInStringArray(k, []string{
				cloudprovider.META_HEADER_CONTENT_TYPE,
				cloudprovider.META_HEADER_CACHE_CONTROL,
				cloudprovider.META_HEADER_CONTENT_DISPOSITION,
				cloudprovider.META_HEADER_CONTENT_ENCODING,
				cloudprovider.META_HEADER_CONTENT_LANGUAGE,
			}) {
				continue
			}
			if len(v) > 0 {
				userMeta[http.CanonicalHeaderKey(k)] = v[0]
			}
		}
		opts.UserMetadata = userMeta
	}
	return opts
}

func (bucket *SBucket) PutObject(ctx context.Context, key string, input io.Reader, sizeBytes int64, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	opts := metaInitOptions(meta)
	if len(storageClassStr) > 0 {
		opts.StorageClass = storageClassStr
	}
	opts.PartSize = uint64(cloudprovider.MAX_PUT_OBJECT_SIZEBYTES)
	_, err := bucket.client.S3Client().PutObjectDo(ctx, bucket.Name, key, input, "", "", sizeBytes, opts)
	if err != nil {
		return errors.Wrap(err, "PutObjectWithContext")
	}
	obj, err := cloudprovider.GetIObject(bucket, key)
	if err != nil {
		return errors.Wrap(err, "cloudprovider.GetIObject")
	}
	if len(cannedAcl) == 0 {
		cannedAcl = bucket.GetAcl()
	}
	err = obj.SetAcl(cannedAcl)
	if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented {
		log.Errorf("PubObject SetAcl fail %s", err)
		// return errors.Wrap(err, "obj.SetAcl")
	}
	return nil
}

func (bucket *SBucket) NewMultipartUpload(ctx context.Context, key string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) (string, error) {
	opts := metaInitOptions(meta)
	if len(storageClassStr) > 0 {
		opts.StorageClass = storageClassStr
	}
	result, err := bucket.client.S3Client().InitiateMultipartUpload(ctx, bucket.Name, key, opts)
	if err != nil {
		return "", errors.Wrap(err, "InitiateMultipartUpload")
	}
	return result.UploadID, nil
}

func (bucket *SBucket) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, input io.Reader, partSize int64, offset, totalSize int64) (string, error) {
	part, err := bucket.client.S3Client().UploadPart(ctx, bucket.Name, key, uploadId, input, partIndex, "", "", partSize, nil)
	if err != nil {
		return "", errors.Wrap(err, "UploadPart")
	}
	return part.ETag, nil
}

func (bucket *SBucket) CompleteMultipartUpload(ctx context.Context, key string, uploadId string, partEtags []string) error {
	complete := s3cli.CompleteMultipartUpload{}
	complete.Parts = make([]s3cli.CompletePart, len(partEtags))
	for i := 0; i < len(partEtags); i += 1 {
		complete.Parts[i] = s3cli.CompletePart{
			PartNumber: i + 1,
			ETag:       partEtags[i],
		}
	}
	_, err := bucket.client.S3Client().CompleteMultipartUpload(ctx, bucket.Name, key, uploadId, complete)
	if err != nil {
		return errors.Wrap(err, "CompleteMultipartUpload")
	}
	return nil
}

func (bucket *SBucket) AbortMultipartUpload(ctx context.Context, key string, uploadId string) error {
	err := bucket.client.S3Client().AbortMultipartUpload(ctx, bucket.Name, key, uploadId)
	if err != nil {
		return errors.Wrap(err, "AbortMultipartUpload")
	}
	return nil
}

func (bucket *SBucket) DeleteObject(ctx context.Context, key string) error {
	err := bucket.client.S3Client().RemoveObject(bucket.Name, key)
	if err != nil {
		return errors.Wrap(err, "RemoveObject")
	}
	return nil
}

func (bucket *SBucket) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	if method != "GET" && method != "PUT" && method != "DELETE" {
		return "", errors.Error("unsupported method")
	}
	url, err := bucket.client.S3Client().Presign(method, bucket.Name, key, expire, nil)
	if err != nil {
		return "", errors.Wrap(err, "Presign")
	}
	return url.String(), nil
}

func (bucket *SBucket) CopyObject(ctx context.Context, destKey string, srcBucket, srcKey string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, dstMeta http.Header) error {
	meta := make(map[string]string)
	if dstMeta != nil {
		for k, v := range dstMeta {
			meta[http.CanonicalHeaderKey(k)] = v[0]
		}
	}
	if len(storageClassStr) > 0 {
		meta[http.CanonicalHeaderKey("x-amz-storage-class")] = storageClassStr
	}
	dest, err := s3cli.NewDestinationInfo(bucket.Name, destKey, nil, meta)
	if err != nil {
		return errors.Wrap(err, "NewDestinationInfo")
	}
	src := s3cli.NewSourceInfo(srcBucket, srcKey, nil)
	err = bucket.client.S3Client().CopyObject(dest, src)
	if err != nil {
		return errors.Wrap(err, "CopyObject")
	}
	obj, err := cloudprovider.GetIObject(bucket, destKey)
	if err != nil {
		return errors.Wrap(err, "GetIObject")
	}
	if len(cannedAcl) == 0 {
		cannedAcl = bucket.GetAcl()
	}
	err = obj.SetAcl(cannedAcl)
	if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented {
		return errors.Wrap(err, "obj.SetAcl")
	}
	return nil
}

func (bucket *SBucket) GetObject(ctx context.Context, key string, rangeOpt *cloudprovider.SGetObjectRange) (io.ReadCloser, error) {
	opts := s3cli.GetObjectOptions{}
	if rangeOpt != nil {
		opts.SetRange(rangeOpt.Start, rangeOpt.End)
	}
	output, err := bucket.client.S3Client().GetObject(bucket.Name, key, opts)
	if err != nil {
		return nil, errors.Wrap(err, "GetObject")
	}
	return output, nil
}

func (bucket *SBucket) CopyPart(ctx context.Context, key string, uploadId string, partNumber int, srcBucket string, srcKey string, srcOffset int64, srcLength int64) (string, error) {
	result, err := bucket.client.S3Client().CopyObjectPartDo(ctx, srcBucket, srcKey, bucket.Name, key, uploadId, partNumber, srcOffset, srcLength, nil)
	if err != nil {
		return "", errors.Wrap(err, "CopyObjectPartDo")
	}
	return result.ETag, nil
}
