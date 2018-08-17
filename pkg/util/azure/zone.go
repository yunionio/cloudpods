package azure

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SZone struct {
	region *SRegion

	iwires    []cloudprovider.ICloudWire
	istorages []cloudprovider.ICloudStorage

	host *SHost
	Name string
}

func (self *SZone) GetId() string {
	return self.region.client.providerId
}

func (self *SZone) GetName() string {
	return self.region.GetName()
}

func (self *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.region.GetGlobalId(), self.region.client.subscriptionId)
}

func (self *SZone) IsEmulated() bool {
	return true
}

func (self *SZone) GetStatus() string {
	return "enable"
}

func (self *SZone) Refresh() error {
	// do nothing
	return nil
}

func (self *SZone) getHost() *SHost {
	if self.host == nil {
		self.host = &SHost{zone: self}
	}
	return self.host
}

func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SZone) fetchStorages() error {
	if storages, err := self.region.getStorage(); err != nil {
		return err
	} else {
		self.istorages = make([]cloudprovider.ICloudStorage, len(storages))
		for i := 0; i < len(storages); i++ {
			storages[i].zone = self
			self.istorages[i] = &storages[i]
		}
	}
	return nil
}

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	if err := self.fetchStorages(); err != nil {
		return nil, err
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

func (self *SZone) getStorageByTier(tier string) (*SStorage, error) {
	storages, err := self.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(storages); i += 1 {
		storage := storages[i].(*SStorage)
		if storage.Tier == tier {
			return storage, nil
		}
	}
	return nil, fmt.Errorf("No such storage %s", tier)
}

func (self *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host := self.getHost()
	if host.GetGlobalId() == id {
		return host, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return []cloudprovider.ICloudHost{self.getHost()}, nil
}

func (self *SZone) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.iwires, nil
}
