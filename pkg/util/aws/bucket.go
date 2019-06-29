package aws

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
	return ""
}

func (b *SBucket) GetAcl() string {
	return ""
}

func (b *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	return []cloudprovider.SBucketAccessUrl{
		{
			Url:         fmt.Sprintf("https://%s.%s", b.Name, b.region.getS3Endpoint()),
			Description: "bucket domain",
		},
		{
			Url:         fmt.Sprintf("https://%s/%s", b.region.getS3Endpoint(), b.Name),
			Description: "s3 domain",
		},
	}
}
