package aws

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
)

type SRegion struct {
	client *SAwsClient

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc

	storageCache  *SStoragecache
	instanceTypes []SInstanceType

	RegionEndpoint string
	RegionId     string    // 这里为保持一致沿用阿里云RegionId的叫法, 与AWS RegionName字段对应
}

func (self *SRegion) GetId() string {
	panic("implement me")
}

func (self *SRegion) GetName() string {
	panic("implement me")
}

func (self *SRegion) GetGlobalId() string {
	panic("implement me")
}

func (self *SRegion) GetStatus() string {
	panic("implement me")
}

func (self *SRegion) Refresh() error {
	panic("implement me")
}

func (self *SRegion) IsEmulated() bool {
	panic("implement me")
}

func (self *SRegion) GetMetadata() *jsonutils.JSONDict {
	panic("implement me")
}

func (self *SRegion) GetLatitude() float32 {
	panic("implement me")
}

func (self *SRegion) GetLongitude() float32 {
	panic("implement me")
}

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	panic("implement me")
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	panic("implement me")
}

func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	panic("implement me")
}

func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	panic("implement me")
}

func (self *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	panic("implement me")
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	panic("implement me")
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	panic("implement me")
}

func (self *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	panic("implement me")
}

func (self *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	panic("implement me")
}

func (self *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	panic("implement me")
}

func (self *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	panic("implement me")
}

func (self *SRegion) CreateEIP(name string, bwMbps int, chargeType string) (cloudprovider.ICloudEIP, error) {
	panic("implement me")
}

func (self *SRegion) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	panic("implement me")
}

func (self *SRegion) GetProvider() string {
	panic("implement me")
}

