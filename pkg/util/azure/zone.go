package azure

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/utils"
)

type SZone struct {
	region *SRegion

	iwires           []cloudprovider.ICloudWire
	iclassicWires    []cloudprovider.ICloudWire
	istorages        []cloudprovider.ICloudStorage
	iclassicStorages []cloudprovider.ICloudStorage

	storageTypes        []string
	classicStorageTypes []string
	Name                string
	host                *SHost
	classicHost         *SClassicHost
}

func (self *SZone) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SZone) GetId() string {
	return self.region.client.providerId
}

func (self *SZone) GetName() string {
	return self.region.GetName()
}

func (self *SZone) GetGlobalId() string {
	return self.region.GetGlobalId()
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

func (self *SZone) getClassicHost() *SClassicHost {
	if self.classicHost == nil {
		self.classicHost = &SClassicHost{zone: self}
	}
	return self.classicHost
}

func (self *SZone) getStorageTypes() (err error) {
	if len(self.storageTypes) > 0 {
		return nil
	}
	storages, err := self.region.GetStorageTypes()
	if err != nil {
		return err
	}
	self.storageTypes = []string{}
	for i := 0; i < len(storages); i++ {
		if !utils.IsInStringArray(storages[i].Name, self.storageTypes) {
			self.storageTypes = append(self.storageTypes, storages[i].Name)
		}
	}
	return nil
}

func (self *SRegion) GetStorageTypes() ([]SStorage, error) {
	storages := []SStorage{}
	err := self.client.ListAll("Microsoft.Storage/skus", &storages)
	if err != nil {
		return nil, err
	}
	result := []SStorage{}
	for i := 0; i < len(storages); i++ {
		if utils.IsInStringArray(self.Name, storages[i].Locations) {
			result = append(result, storages[i])
		}
	}
	return result, nil
}

func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SZone) fetchClassicStorages() error {
	storageaccounts, err := self.region.GetClassicStorageAccounts()
	if err != nil {
		return err
	}
	self.iclassicStorages = make([]cloudprovider.ICloudStorage, len(storageaccounts))
	for i := 0; i < len(storageaccounts); i++ {
		storage := SClassicStorage{
			zone:       self,
			ID:         storageaccounts[i].ID,
			Name:       storageaccounts[i].Name,
			Type:       storageaccounts[i].Type,
			Location:   storageaccounts[i].Location,
			Properties: storageaccounts[i].Properties.ClassicStorageProperties,
		}
		self.iclassicStorages[i] = &storage
	}
	return nil
}

func (self *SZone) fetchStorages() error {
	if len(self.storageTypes) == 0 {
		err := self.getStorageTypes()
		if err != nil {
			return err
		}
	}
	self.istorages = make([]cloudprovider.ICloudStorage, len(self.storageTypes))
	for i, storageType := range self.storageTypes {
		storage := SStorage{zone: self, storageType: storageType}
		self.istorages[i] = &storage
	}
	return nil
}

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	err := self.fetchStorages()
	if err != nil {
		return nil, err
	}
	self.fetchClassicStorages()
	istorages := append(self.istorages, self.iclassicStorages...)
	return istorages, nil
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

func (self *SZone) getStorageByType(storageType string) (*SStorage, error) {
	_, err := self.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(self.istorages); i += 1 {
		storage := self.istorages[i].(*SStorage)
		if strings.ToLower(storage.storageType) == strings.ToLower(storageType) {
			return storage, nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SZone) getClassicStorageByType(storageType string) (*SClassicStorage, error) {
	_, err := self.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(self.iclassicStorages); i += 1 {
		storage := self.iclassicStorages[i].(*SClassicStorage)
		if strings.ToLower(storage.Properties.AccountType) == strings.ToLower(storageType) {
			return storage, nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host := self.getHost()
	if host.GetGlobalId() == id {
		return host, nil
	}
	classicHost := self.getClassicHost()
	if classicHost.GetGlobalId() == id {
		return classicHost, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return []cloudprovider.ICloudHost{self.getHost(), self.getClassicHost()}, nil
}

func (self *SZone) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SZone) addClassicWire(wire *SClassicWire) {
	if self.iclassicWires == nil {
		self.iclassicWires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iclassicWires = append(self.iclassicWires, wire)
}

func (self *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.iwires, nil
}

func (self *SZone) GetIClassicWires() ([]cloudprovider.ICloudWire, error) {
	return self.iclassicWires, nil
}

func (self *SZone) getNetworkById(networId string) *SNetwork {
	log.Debugf("Search in wires %d", len(self.iwires))
	for i := 0; i < len(self.iwires); i += 1 {
		log.Debugf("Search in wire %s", self.iwires[i].GetName())
		wire := self.iwires[i].(*SWire)
		net := wire.getNetworkById(networId)
		if net != nil {
			return net
		}
	}
	return nil
}

func (self *SZone) getClassicNetworkById(networId string) *SClassicNetwork {
	log.Debugf("Search in wires %d", len(self.iclassicWires))
	for i := 0; i < len(self.iclassicWires); i += 1 {
		log.Debugf("Search in wire %s", self.iclassicWires[i].GetName())
		wire := self.iclassicWires[i].(*SClassicWire)
		net := wire.getNetworkById(networId)
		if net != nil {
			return net
		}
	}
	return nil
}
