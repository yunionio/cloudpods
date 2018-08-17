package azure

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SStoragecache struct {
	region *SRegion

	iimages []cloudprovider.ICloudImage
}

func (self *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerId, self.region.GetId())
}

func (self *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerName, self.region.GetId())
}

func (self *SStoragecache) GetStatus() string {
	return "available"
}

func (self *SStoragecache) Refresh() error {
	return nil
}

func (self *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerId, self.region.GetGlobalId())
}

func (self *SStoragecache) IsEmulated() bool {
	return false
}

func (self *SStoragecache) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SStoragecache) GetIImages() ([]cloudprovider.ICloudImage, error) {
	// if self.iimages == nil {
	// 	err := self.fetchImages()
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }
	return self.iimages, nil
}

func (self *SStoragecache) UploadImage(userCred mcclient.TokenCredential, imageId string, extId string, isForce bool) (string, error) {

	return "", nil
}
