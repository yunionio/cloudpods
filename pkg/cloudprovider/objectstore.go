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

package cloudprovider

import (
	"context"
	"io"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/s3cli"
)

type TBucketACLType string

const (
	// 50 MB
	MAX_PUT_OBJECT_SIZEBYTES = int64(1000 * 1000 * 50)

	// ACLDefault = TBucketACLType("default")

	ACLPrivate         = TBucketACLType(s3cli.CANNED_ACL_PRIVATE)
	ACLAuthRead        = TBucketACLType(s3cli.CANNED_ACL_AUTH_READ)
	ACLPublicRead      = TBucketACLType(s3cli.CANNED_ACL_PUBLIC_READ)
	ACLPublicReadWrite = TBucketACLType(s3cli.CANNED_ACL_PUBLIC_READ_WRITE)
	ACLUnknown         = TBucketACLType("")
)

type SBucketStats struct {
	SizeBytes   int64
	ObjectCount int
}

func (s SBucketStats) Equals(s2 SBucketStats) bool {
	if s.SizeBytes == s2.SizeBytes && s.ObjectCount == s2.ObjectCount {
		return true
	} else {
		return false
	}
}

type SBucketAccessUrl struct {
	Url         string
	Description string
	Primary     bool
}

type SBaseCloudObject struct {
	Key          string
	SizeBytes    int64
	StorageClass string
	ETag         string
	LastModified time.Time
	ContentType  string
}

type SListObjectResult struct {
	Objects        []ICloudObject
	NextMarker     string
	CommonPrefixes []ICloudObject
	IsTruncated    bool
}

type ICloudBucket interface {
	IVirtualResource

	MaxPartCount() int
	MaxPartSizeBytes() int64

	//GetGlobalId() string
	//GetName() string
	GetAcl() TBucketACLType
	GetLocation() string
	GetIRegion() ICloudRegion
	GetCreateAt() time.Time
	GetStorageClass() string
	GetAccessUrls() []SBucketAccessUrl
	GetStats() SBucketStats

	SetAcl(acl TBucketACLType) error

	ListObjects(prefix string, marker string, delimiter string, maxCount int) (SListObjectResult, error)
	GetIObjects(prefix string, isRecursive bool) ([]ICloudObject, error)

	DeleteObject(ctx context.Context, keys string) error
	GetTempUrl(method string, key string, expire time.Duration) (string, error)

	PutObject(ctx context.Context, key string, input io.Reader, sizeBytes int64, contType string, cannedAcl TBucketACLType, storageClassStr string) error
	NewMultipartUpload(ctx context.Context, key string, contType string, cannedAcl TBucketACLType, storageClassStr string) (string, error)
	UploadPart(ctx context.Context, key string, uploadId string, partIndex int, input io.Reader, partSize int64) (string, error)
	CompleteMultipartUpload(ctx context.Context, key string, uploadId string, partEtags []string) error
	AbortMultipartUpload(ctx context.Context, key string, uploadId string) error
}

type ICloudObject interface {
	GetIBucket() ICloudBucket

	GetKey() string
	GetSizeBytes() int64
	GetLastModified() time.Time
	GetStorageClass() string
	GetETag() string
	GetContentType() string

	GetAcl() TBucketACLType
	SetAcl(acl TBucketACLType) error
}

func ICloudObject2BaseCloudObject(obj ICloudObject) SBaseCloudObject {
	return SBaseCloudObject{
		Key:          obj.GetKey(),
		SizeBytes:    obj.GetSizeBytes(),
		StorageClass: obj.GetStorageClass(),
		ETag:         obj.GetETag(),
		LastModified: obj.GetLastModified(),
		ContentType:  obj.GetContentType(),
	}
}

func (o *SBaseCloudObject) GetKey() string {
	return o.Key
}

func (o *SBaseCloudObject) GetSizeBytes() int64 {
	return o.SizeBytes
}

func (o *SBaseCloudObject) GetLastModified() time.Time {
	return o.LastModified
}

func (o *SBaseCloudObject) GetStorageClass() string {
	return o.StorageClass
}

func (o *SBaseCloudObject) GetETag() string {
	return o.ETag
}

func (o *SBaseCloudObject) GetContentType() string {
	return o.ContentType
}

func GetIBucketById(region ICloudRegion, name string) (ICloudBucket, error) {
	buckets, err := region.GetIBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "region.GetIBuckets")
	}
	for i := range buckets {
		if buckets[i].GetGlobalId() == name {
			return buckets[i], nil
		}
	}
	return nil, ErrNotFound
}

func GetIBucketByName(region ICloudRegion, name string) (ICloudBucket, error) {
	buckets, err := region.GetIBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "region.GetIBuckets")
	}
	for i := range buckets {
		if buckets[i].GetName() == name {
			return buckets[i], nil
		}
	}
	return nil, ErrNotFound
}

func GetIBucketStats(bucket ICloudBucket) (SBucketStats, error) {
	stats := SBucketStats{}
	objs, err := bucket.GetIObjects("", true)
	if err != nil {
		stats.ObjectCount = -1
		stats.SizeBytes = -1
		return stats, errors.Wrap(err, "GetIObjects")
	}
	for _, obj := range objs {
		stats.SizeBytes += obj.GetSizeBytes()
		stats.ObjectCount += 1
	}
	return stats, nil
}

func GetIObjects(bucket ICloudBucket, objectPrefix string, isRecursive bool) ([]ICloudObject, error) {
	delimiter := "/"
	if isRecursive {
		delimiter = ""
	}
	ret := make([]ICloudObject, 0)
	// Save marker for next request.
	var marker string
	for {
		// Get list of objects a maximum of 1000 per request.
		result, err := bucket.ListObjects(objectPrefix, marker, delimiter, 1000)
		if err != nil {
			return nil, errors.Wrap(err, "bucket.ListObjects")
		}

		// Send all objects
		for i := range result.Objects {
			if !isRecursive && result.Objects[i].GetKey() == objectPrefix {
				continue
			}
			ret = append(ret, result.Objects[i])
			marker = result.Objects[i].GetKey()
		}

		// Send all common prefixes if any.
		// NOTE: prefixes are only present if the request is delimited.
		if len(result.CommonPrefixes) > 0 {
			ret = append(ret, result.CommonPrefixes...)
		}

		// If next marker present, save it for next request.
		if result.NextMarker != "" {
			marker = result.NextMarker
		}

		// Listing ends result is not truncated, break the loop
		if !result.IsTruncated {
			break
		}
	}
	return ret, nil
}

func GetIObject(bucket ICloudBucket, objectPrefix string) (ICloudObject, error) {
	objects, err := GetIObjects(bucket, objectPrefix, true)
	if err != nil {
		return nil, errors.Wrap(err, "GetIObjects")
	}
	if len(objects) > 0 && objects[0].GetKey() == objectPrefix {
		return objects[0], nil
	}
	return nil, ErrNotFound
}

func Makedir(ctx context.Context, bucket ICloudBucket, key string) error {
	segs := make([]string, 0)
	for _, seg := range strings.Split(key, "/") {
		if len(seg) > 0 {
			segs = append(segs, seg)
		}
	}
	path := strings.Join(segs, "/") + "/"
	err := bucket.PutObject(ctx, path, strings.NewReader(""), 0, "", ACLPrivate, "")
	if err != nil {
		return errors.Wrap(err, "PutObject")
	}
	return nil
}

func UploadObject(ctx context.Context, bucket ICloudBucket, key string, blocksz int64, input io.Reader, sizeBytes int64, contType string, cannedAcl TBucketACLType, storageClass string, debug bool) error {
	if blocksz <= 0 {
		blocksz = MAX_PUT_OBJECT_SIZEBYTES
	}
	if sizeBytes < blocksz {
		if debug {
			log.Debugf("too small, put object in one shot")
		}
		return bucket.PutObject(ctx, key, input, sizeBytes, contType, cannedAcl, storageClass)
	}
	partSize := blocksz
	partCount := sizeBytes / partSize
	if partCount*partSize < sizeBytes {
		partCount += 1
	}
	if partCount > int64(bucket.MaxPartCount()) {
		partCount = int64(bucket.MaxPartCount())
		partSize = sizeBytes / partCount
		if partSize*partCount < sizeBytes {
			partSize += 1
		}
		if partSize > bucket.MaxPartSizeBytes() {
			return errors.Error("too larget object")
		}
	}
	if debug {
		log.Debugf("multipart upload part count %d part size %d", partCount, partSize)
	}
	uploadId, err := bucket.NewMultipartUpload(ctx, key, contType, cannedAcl, storageClass)
	if err != nil {
		return errors.Wrap(err, "bucket.NewMultipartUpload")
	}
	etags := make([]string, partCount)
	// offset := int64(0)
	for i := 0; i < int(partCount); i += 1 {
		if i == int(partCount)-1 {
			partSize = sizeBytes - partSize*(partCount-1)
		}
		if debug {
			log.Debugf("UploadPart %d %d", i+1, partSize)
		}
		etag, err := bucket.UploadPart(ctx, key, uploadId, i+1, io.LimitReader(input, partSize), partSize)
		if err != nil {
			err2 := bucket.AbortMultipartUpload(ctx, key, uploadId)
			if err2 != nil {
				log.Errorf("bucket.AbortMultipartUpload error %s", err2)
			}
			return errors.Wrap(err, "bucket.UploadPart")
		}
		// offset += partSize
		etags[i] = etag
	}
	err = bucket.CompleteMultipartUpload(ctx, key, uploadId, etags)
	if err != nil {
		err2 := bucket.AbortMultipartUpload(ctx, key, uploadId)
		if err2 != nil {
			log.Errorf("bucket.AbortMultipartUpload error %s", err2)
		}
		return errors.Wrap(err, "CompleteMultipartUpload")
	}
	return nil
}

func DeletePrefix(ctx context.Context, bucket ICloudBucket, prefix string) error {
	objs, err := bucket.GetIObjects(prefix, true)
	if err != nil {
		return errors.Wrap(err, "bucket.GetIObjects")
	}
	for i := range objs {
		err := bucket.DeleteObject(ctx, objs[i].GetKey())
		if err != nil {
			return errors.Wrap(err, "bucket.DeleteObject")
		}
	}
	return nil
}
