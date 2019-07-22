package cloudprovider

import (
	"context"
	"io"
	"time"

	"strings"
	"yunion.io/x/pkg/errors"
)

type SBucketAccessUrl struct {
	Url         string
	Description string
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

	GetGlobalId() string
	GetName() string
	GetAcl() string
	GetLocation() string
	GetIRegion() ICloudRegion
	GetCreateAt() time.Time
	GetStorageClass() string
	GetAccessUrls() []SBucketAccessUrl

	ListObjects(prefix string, marker string, delimiter string, maxCount int) (SListObjectResult, error)
	GetIObjects(prefix string, isRecursive bool) ([]ICloudObject, error)
	PutObject(ctx context.Context, key string, input io.ReadSeeker, contType string, storageClass string) error
	DeleteObject(ctx context.Context, keys string) error
	GetTempUrl(method string, key string, expire time.Duration) (string, error)
	// ObjectExist(key string) (bool, error)
}

type ICloudObject interface {
	GetIBucket() ICloudBucket

	GetKey() string
	GetSizeBytes() int64
	GetLastModified() time.Time
	GetStorageClass() string
	GetETag() string
	GetContentType() string
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
		if len(result.Objects) > 0 {
			ret = append(ret, result.Objects...)
			marker = result.Objects[len(result.Objects)-1].GetKey()
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

func Makedir(ctx context.Context, bucket ICloudBucket, key string) error {
	segs := make([]string, 0)
	for _, seg := range strings.Split(key, "/") {
		if len(seg) > 0 {
			segs = append(segs, seg)
		}
	}
	path := strings.Join(segs, "/") + "/"
	err := bucket.PutObject(ctx, path, strings.NewReader(""), "", "")
	if err != nil {
		return errors.Wrap(err, "PutObject")
	}
	return nil
}
