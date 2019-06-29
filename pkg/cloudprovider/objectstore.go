package cloudprovider

import (
	"time"

	"yunion.io/x/pkg/errors"
)

type SBucketAccessUrl struct {
	Url         string
	Description string
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
