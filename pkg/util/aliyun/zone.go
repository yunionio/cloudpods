package aliyun

import (
	"fmt"

	"github.com/yunionio/log"

	"github.com/yunionio/onecloud/pkg/cloudprovider"
	"github.com/yunionio/onecloud/pkg/compute/models"
)

type InstanceChargeType string

const (
	PrePaidInstanceChargeType  InstanceChargeType = "PrePaid"
	PostPaidInstanceChargeType InstanceChargeType = "PostPaid"
	DefaultInstanceChargeType                     = PostPaidInstanceChargeType
)

type SpotStrategyType string

const (
	NoSpotStrategy             SpotStrategyType = "NoSpot"
	SpotWithPriceLimitStrategy SpotStrategyType = "SpotWithPriceLimit"
	SpotAsPriceGoStrategy      SpotStrategyType = "SpotAsPriceGo"
	DefaultSpotStrategy                         = NoSpotStrategy
)

type SDedicatedHostGenerations struct {
	DedicatedHostGeneration []string
}

type SVolumeCategories struct {
	VolumeCategories []string
}

type SSupportedDataDiskCategories struct {
	SupportedDataDiskCategory []string
}

type SSupportedInstanceGenerations struct {
	SupportedInstanceGeneration []string
}

type SSupportedInstanceTypeFamilies struct {
	SupportedInstanceTypeFamily []string
}

type SSupportedInstanceTypes struct {
	SupportedInstanceType []string
}

type SSupportedNetworkTypes struct {
	SupportedNetworkCategory []string
}

type SSupportedSystemDiskCategories struct {
	SupportedSystemDiskCategory []string
}

type SResourcesInfo struct {
	DataDiskCategories   SSupportedDataDiskCategories
	InstanceGenerations  SSupportedInstanceGenerations
	InstanceTypeFamilies SSupportedInstanceTypeFamilies
	InstanceTypes        SSupportedInstanceTypes
	IoOptimized          bool
	NetworkTypes         SSupportedNetworkTypes
	SystemDiskCategories SSupportedSystemDiskCategories
}

type SResources struct {
	ResourcesInfo []SResourcesInfo
}

type SResourceCreation struct {
	ResourceTypes []string
}

type SInstanceTypes struct {
	InstanceTypes []string
}

type SDiskCategories struct {
	DiskCategories []string
}

type SDedicatedHostTypes struct {
	DedicatedHostType []string
}

type SZone struct {
	region *SRegion

	iwires []cloudprovider.ICloudWire

	host *SHost

	istorages []cloudprovider.ICloudStorage

	ZoneId                    string
	LocalName                 string
	DedicatedHostGenerations  SDedicatedHostGenerations
	AvailableVolumeCategories SVolumeCategories
	/* 可供创建的具体资源，AvailableResourcesType 组成的数组 */
	AvailableResources SResources
	/* 允许创建的资源类型集合 */
	AvailableResourceCreation SResourceCreation
	/* 允许创建的实例规格类型 */
	AvailableInstanceTypes SInstanceTypes
	/* 支持的磁盘种类集合 */
	AvailableDiskCategories     SDiskCategories
	AvailableDedicatedHostTypes SDedicatedHostTypes
}

func (self *SZone) GetId() string {
	return self.ZoneId
}

func (self *SZone) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_ALIYUN_CN, self.LocalName)
}

func (self *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.region.GetGlobalId(), self.ZoneId)
}

func (self *SZone) GetStatus() string {
	if len(self.AvailableResourceCreation.ResourceTypes) == 0 {
		return models.ZONE_SOLDOUT
	} else {
		return models.ZONE_ENABLE
	}
}

func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SZone) fetchStorages() error {
	self.istorages = make([]cloudprovider.ICloudStorage, len(self.AvailableDiskCategories.DiskCategories))

	for i, sc := range self.AvailableDiskCategories.DiskCategories {
		storage := SStorage{zone: self, storageType: sc}
		self.istorages[i] = &storage
	}
	return nil
}

func (self *SZone) getStorageByCategory(category string) (*SStorage, error) {
	storages, err := self.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(storages); i += 1 {
		storage := storages[i].(*SStorage)
		if storage.storageType == category {
			return storage, nil
		}
	}
	return nil, fmt.Errorf("No such storage %s", category)
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

func (self *SZone) getHost() *SHost {
	if self.host == nil {
		self.host = &SHost{zone: self}
	}
	return self.host
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

func (self *SZone) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.iwires, nil
}

func (self *SZone) getNetworkById(vswitchId string) *SVSwitch {
	log.Debugf("Search in wires %d", len(self.iwires))
	for i := 0; i < len(self.iwires); i += 1 {
		log.Debugf("Search in wire %s", self.iwires[i].GetName())
		wire := self.iwires[i].(*SWire)
		net := wire.getNetworkById(vswitchId)
		if net != nil {
			return net
		}
	}
	return nil
}
