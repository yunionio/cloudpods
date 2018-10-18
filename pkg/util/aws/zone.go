package aws

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"fmt"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/log"
)

var StorageTypes = []string{
	models.STORAGE_GP2_SSD,
	models.STORAGE_IO1_SSD,
	models.STORAGE_ST1_HDD,
	models.STORAGE_SC1_SSD,
	models.STORAGE_STANDARD_SSD,
}

type SZone struct {
	region *SRegion
	host   *SHost

	iwires    []cloudprovider.ICloudWire
	istorages []cloudprovider.ICloudStorage

	ZoneId    string // 沿用阿里云ZoneId,对应Aws ZoneName
	LocalName string
	State     string

	/* 支持的磁盘种类集合 */
	storageTypes     []string
}

func (self *SZone) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SZone) getHost() *SHost {
	if self.host == nil {
		self.host = &SHost{zone: self}
	}
	return self.host
}

func (self *SZone) getStorageType() {
	if len(self.storageTypes) == 0 {
		self.storageTypes = StorageTypes
	}
}

func (self *SZone) fetchStorages() error {
	self.getStorageType()
	self.istorages = make([]cloudprovider.ICloudStorage, len(self.storageTypes))

	for i, sc := range self.storageTypes {
		storage := SStorage{zone: self, storageType: sc}
		self.istorages[i] = &storage
	}
	return nil
}

func (self *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.iwires, nil
}

func (self *SZone) getNetworkById(networkId string) *SNetwork {
	log.Debugf("Search in wires %d", len(self.iwires))
	for i := 0; i < len(self.iwires); i += 1 {
		log.Debugf("Search in wire %s", self.iwires[i].GetName())
		wire := self.iwires[i].(*SWire)
		net := wire.getNetworkById(networkId)
		if net != nil {
			return net
		}
	}
	return nil
}

func (self *SZone) GetId() string {
	return self.ZoneId
}

func (self *SZone) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_AWS_CN, self.LocalName)
}

func (self *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.region.GetGlobalId(), self.ZoneId)
}

func (self *SZone) GetStatus() string {
	if self.State == "unavailable" {
		return models.ZONE_SOLDOUT
	} else {
		return models.ZONE_ENABLE
	}
}

func (self *SZone) Refresh() error {
	return nil
}

func (self *SZone) IsEmulated() bool {
	return false
}

func (self *SZone) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return []cloudprovider.ICloudHost{self.getHost()}, nil
}

func (self *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host := self.getHost()
	if host.GetGlobalId() == id {
		return host, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	if self.istorages == nil {
		self.fetchStorages()
	}
	return self.istorages, nil
}

func (self *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	if self.istorages == nil {
		self.fetchStorages()
	}
	for i := 0; i < len(self.istorages); i += 1 {
		if self.istorages[i].GetGlobalId() == id {
			return self.istorages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) getZoneById(id string) (*SZone, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		zone := izones[i].(*SZone)
		if zone.ZoneId == id {
			return zone, nil
		}
	}
	return nil, fmt.Errorf("no such zone %s", id)
}