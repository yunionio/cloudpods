package objectstore

import (
	"fmt"
	"path"
	"time"

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
