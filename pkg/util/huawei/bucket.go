package huawei

import (
	"fmt"
	"time"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SBucket struct {
	region *SRegion

	Name         string
	Location     string
	CreationDate time.Time

	StorageClass string
	Acl          string

	Size         int64
	ObjectNumber int
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

func (b *SBucket) GetAcl() string {
	return b.Acl
}

func (b *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	return []cloudprovider.SBucketAccessUrl{
		{
			Url:         fmt.Sprintf("https://%s.%s", b.Name, b.region.getOBSEndpoint()),
			Description: "bucket url",
		},
		{
			Url:         fmt.Sprintf("https://%s/%s", b.region.getOBSEndpoint(), b.Name),
			Description: "obs url",
		},
	}
}

func (b *SBucket) GetSizeByte() int64 {
	return b.Size
}

func (b *SBucket) GetObjectNumber() int {
	return b.ObjectNumber
}
