package aws

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
)

type SZone struct {
	region *SRegion
	host   *SHost

	iwires    []cloudprovider.ICloudWire
	istorages []cloudprovider.ICloudStorage

	ZoneId    string // 沿用阿里云ZoneId,对应Aws ZoneName
	LocalName string
	State     string
}

func (self *SZone) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SZone) GetId() string {
	panic("implement me")
}

func (self *SZone) GetName() string {
	panic("implement me")
}

func (self *SZone) GetGlobalId() string {
	panic("implement me")
}

func (self *SZone) GetStatus() string {
	panic("implement me")
}

func (self *SZone) Refresh() error {
	panic("implement me")
}

func (self *SZone) IsEmulated() bool {
	panic("implement me")
}

func (self *SZone) GetMetadata() *jsonutils.JSONDict {
	panic("implement me")
}

func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
	panic("implement me")
}

func (self *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	panic("implement me")
}

func (self *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	panic("implement me")
}

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	panic("implement me")
}

func (self *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	panic("implement me")
}