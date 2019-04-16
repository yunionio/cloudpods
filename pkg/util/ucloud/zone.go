package ucloud

import (
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

// https://docs.ucloud.cn/api/udisk-api/create_udisk
// UDisk 类型: DataDisk（普通数据盘），SSDDataDisk（SSD数据盘），默认值（DataDisk）
var StorageTypes = []string{
	api.STORAGE_UCLOUD_CLOUD_NORMAL,
	api.STORAGE_UCLOUD_CLOUD_SSD,
}

type SZone struct {
	region *SRegion
	host   *SHost

	iwires    []cloudprovider.ICloudWire
	istorages []cloudprovider.ICloudStorage

	RegionId string
	ZoneId   string

	/* 支持的磁盘种类集合 */
	storageTypes []string
}

func (self *SZone) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SZone) getHost() *SHost {
	if self.host == nil {
		self.host = &SHost{zone: self, projectId: self.region.client.projectId}
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

func (self *SZone) GetId() string {
	return self.ZoneId
}

func (self *SZone) GetName() string {
	if name, exists := UCLOUD_ZONE_NAMES[self.GetId()]; exists {
		return name
	}

	return self.GetId()
}

func (self *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.region.GetGlobalId(), self.GetId())
}

func (self *SZone) GetStatus() string {
	return api.ZONE_ENABLE
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

// https://docs.ucloud.cn/api/uhost-api/describe_uhost_instance
func (self *SZone) GetInstances() ([]SInstance, error) {
	return self.region.GetInstances(self.GetId(), "")
}

func (self *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.iwires, nil
}

// https://docs.ucloud.cn/api/uhost-api/describe_uhost_instance
func (self *SRegion) GetInstances(zoneId string, instanceId string) ([]SInstance, error) {
	instances := make([]SInstance, 0)

	params := NewUcloudParams()
	if len(zoneId) > 0 {
		params.Set("Zone", zoneId)
	}

	if len(instanceId) > 0 {
		params.Set("UHostIds.0", instanceId)
	}

	err := self.DoListAll("DescribeUHostInstance", params, &instances)
	return instances, err
}
