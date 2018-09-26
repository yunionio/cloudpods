package aws

import "yunion.io/x/onecloud/pkg/cloudprovider"

type SStoragecache struct {
	region *SRegion

	iimages []cloudprovider.ICloudImage
}
