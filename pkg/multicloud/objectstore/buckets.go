package objectstore

import (
	"context"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/minio/minio-go"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SBucket struct {
	client *SObjectStoreClient

	Name         string
	Location     string
	CreatedAt    time.Time
	StorageClass string
	Acl          string
}

func (bucket *SBucket) GetProjectId() string {
	return ""
}

func (bucket *SBucket) GetGlobalId() string {
	return bucket.Name
}

func (bucket *SBucket) GetName() string {
	return bucket.Name
}

func (bucket *SBucket) GetAcl() string {
	return bucket.Acl
}

func (bucket *SBucket) GetLocation() string {
	return bucket.Location
}

func (bucket *SBucket) GetIRegion() cloudprovider.ICloudRegion {
	return bucket.client
}

func (bucket *SBucket) GetCreateAt() time.Time {
	return bucket.CreatedAt
}

func (bucket *SBucket) GetStorageClass() string {
	return bucket.StorageClass
}

func (bucket *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	return []cloudprovider.SBucketAccessUrl{
		{
			Url:         path.Join(bucket.client.endpoint, bucket.Name),
			Description: fmt.Sprintf("%s", bucket.Location),
		},
	}
}

func (bucket *SBucket) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	isRecursive := true
	if delimiter == "/" {
		isRecursive = false
	}
	result := cloudprovider.SListObjectResult{}
	var err error
	result.Objects, err = bucket.GetIObjects(prefix, isRecursive)
	return result, err
}

func (bucket *SBucket) GetIObjects(prefix string, isRecursive bool) ([]cloudprovider.ICloudObject, error) {
	doneCh := make(chan struct{})
	defer close(doneCh)

	ret := make([]cloudprovider.ICloudObject, 0)
	objectCh := bucket.client.client.ListObjects(bucket.Name, prefix, isRecursive, doneCh)
	for object := range objectCh {
		if object.Err != nil {
			return nil, errors.Wrap(object.Err, "ListObjects")
		}
		obj := &SObject{
			bucket: bucket,
			SBaseCloudObject: cloudprovider.SBaseCloudObject{
				StorageClass: object.StorageClass,
				Key:          object.Key,
				SizeBytes:    object.Size,
				ETag:         object.ETag,
				LastModified: object.LastModified,
				ContentType:  object.ContentType,
			},
		}
		ret = append(ret, obj)
	}
	return ret, nil
}

func (bucket *SBucket) PutObject(ctx context.Context, key string, input io.ReadSeeker, contType string, storageClass string) error {
	opts := minio.PutObjectOptions{}
	if len(contType) > 0 {
		opts.ContentType = contType
	}
	if len(storageClass) > 0 {
		opts.StorageClass = storageClass
	}
	_, err := bucket.client.client.PutObjectWithContext(ctx, bucket.Name, key, input, -1, opts)
	return err
}

func (bucket *SBucket) DeleteObject(ctx context.Context, key string) error {
	err := bucket.client.client.RemoveObject(bucket.Name, key)
	if err != nil {
		return errors.Wrap(err, "RemoveObject")
	}
	return nil
}

func (bucket *SBucket) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	if method != "GET" && method != "PUT" && method != "DELETE" {
		return "", errors.Error("unsupported method")
	}
	url, err := bucket.client.client.Presign(method, bucket.Name, key, expire, nil)
	if err != nil {
		return "", errors.Wrap(err, "Presign")
	}
	return url.String(), nil
}
