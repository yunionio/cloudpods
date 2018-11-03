package qcloud

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/utils"
)

type InstanceChargeType string

const (
	PrePaidInstanceChargeType  InstanceChargeType = "PREPAID"
	PostPaidInstanceChargeType InstanceChargeType = "POSTPAID_BY_HOUR"
	CdhPaidInstanceChargeType  InstanceChargeType = "CDHPAID"
	DefaultInstanceChargeType                     = PostPaidInstanceChargeType
)

type SZone struct {
	region *SRegion

	iwires []cloudprovider.ICloudWire

	host *SHost

	istorages []cloudprovider.ICloudStorage

	instanceTypes []string
	refreshTime   time.Time

	Zone      string
	ZoneId    string
	ZoneName  string
	ZoneState string
}

func (self *SZone) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SZone) GetId() string {
	return self.Zone
}

func (self *SZone) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_QCLOUD_CN, self.ZoneName)
}

func (self *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.region.GetGlobalId(), self.Zone)
}

func (self *SZone) IsEmulated() bool {
	return false
}

func (self *SZone) Refresh() error {
	// do nothing
	return nil
}

func (self *SZone) GetStatus() string {
	if self.ZoneState == "AVAILABLE" {
		return models.ZONE_ENABLE
	}
	return models.ZONE_SOLDOUT
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
	self.istorages = []cloudprovider.ICloudStorage{}
	for _, storageType := range []string{"CLOUD_BASIC", "CLOUD_PREMIUM", "CLOUD_SSD"} {
		storage := SStorage{zone: self, storageType: storageType}
		self.istorages = append(self.istorages, &storage)
	}
	for _, localstorageType := range []string{"LOCAL_BASIC", "LOCAL_SSD"} {
		storage := SLocalStorage{zone: self, storageType: localstorageType}
		self.istorages = append(self.istorages, &storage)
	}
	return nil
}

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	if self.istorages == nil {
		self.fetchStorages()
	}
	return self.istorages, nil
}

func (self *SZone) getLocalStorageByCategory(category string) (*SLocalStorage, error) {
	if utils.IsInStringArray(strings.ToLower(category), []string{"local_basic", "local_ssd"}) {
		return &SLocalStorage{zone: self, storageType: strings.ToUpper(category)}, nil
	}
	return nil, fmt.Errorf("No such storage %s", category)
}

func (self *SZone) validateStorageType(category string) error {
	if utils.IsInStringArray(strings.ToLower(category), []string{"local_basic", "local_ssd", "cloud_basic", "cloud_ssd", "cloud_permium"}) {
		return nil
	}
	return fmt.Errorf("No such storage %s", category)
}

func (self *SZone) getStorageByCategory(category string) (*SStorage, error) {
	storages, err := self.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(storages); i++ {
		if utils.IsInStringArray(storages[i].GetStorageType(), []string{"local_basic", "local_ssd"}) {
			continue
			//return &SStorage{zone: self, storageType: strings.ToUpper(storages[i].GetStorageType())}, nil
		}
		storage := storages[i].(*SStorage)
		if storage.storageType == category {
			return storage, nil
		}
	}
	return nil, fmt.Errorf("No such storage %s", category)
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

func (self *SZone) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
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

func (self *SZone) fetchInstanceTypes() {
	self.instanceTypes = []string{}
	params := map[string]string{}
	params["Region"] = self.region.Region
	params["Filters.0.Name"] = "zone"
	params["Filters.0.Values.0"] = self.Zone
	if body, err := self.region.cvmRequest("DescribeInstanceTypeConfigs", params); err != nil {
		log.Errorf("DescribeInstanceTypeConfigs error: %v", err)
	} else if configSet, err := body.GetArray("InstanceTypeConfigSet"); err != nil {
		log.Errorf("Get InstanceTypeConfigSet error: %v", err)
	} else {
		for _, config := range configSet {
			if instanceType, err := config.GetString("InstanceType"); err == nil && !utils.IsInStringArray(instanceType, self.instanceTypes) {
				self.instanceTypes = append(self.instanceTypes, instanceType)
			}
		}
		self.refreshTime = time.Now()
	}
}

func (self *SZone) getAvaliableInstanceTypes() []string {
	if self.instanceTypes == nil || len(self.instanceTypes) == 0 || time.Now().Sub(self.refreshTime).Hours() > refreshHours() {
		self.fetchInstanceTypes()
	}
	return self.instanceTypes
}

func refreshHours() float64 {
	return 5
}
