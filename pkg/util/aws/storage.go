package aws

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SStorage struct {
	zone        *SZone
	storageType string
}

func (self *SStorage) GetId() string {
	panic("implement me")
}

func (self *SStorage) GetName() string {
	panic("implement me")
}

func (self *SStorage) GetGlobalId() string {
	panic("implement me")
}

func (self *SStorage) GetStatus() string {
	panic("implement me")
}

func (self *SStorage) Refresh() error {
	panic("implement me")
}

func (self *SStorage) IsEmulated() bool {
	panic("implement me")
}

func (self *SStorage) GetMetadata() *jsonutils.JSONDict {
	panic("implement me")
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	panic("implement me")
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	panic("implement me")
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	panic("implement me")
}

func (self *SStorage) GetStorageType() string {
	panic("implement me")
}

func (self *SStorage) GetMediumType() string {
	panic("implement me")
}

func (self *SStorage) GetCapacityMB() int {
	panic("implement me")
}

func (self *SStorage) GetStorageConf() jsonutils.JSONObject {
	panic("implement me")
}

func (self *SStorage) GetEnabled() bool {
	panic("implement me")
}

func (self *SStorage) GetManagerId() string {
	panic("implement me")
}

func (self *SStorage) CreateIDisk(name string, sizeGb int, desc string) (cloudprovider.ICloudDisk, error) {
	panic("implement me")
}

func (self *SStorage) GetIDisk(idStr string) (cloudprovider.ICloudDisk, error) {
	panic("implement me")
}
