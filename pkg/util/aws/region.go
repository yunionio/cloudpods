package aws

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"fmt"
	"yunion.io/x/onecloud/pkg/compute/models"
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

/////////////////////////////////////////////////////////////////////////////
/* todo: 请不要使用这个client(AWS_DEFAULT_REGION)跨region查信息.有可能导致查询返回的信息为空。比如DescribeAvailabilityZones*/
func (self *SRegion) GetClient() (*SAwsClient) {
	return self.client
}

/////////////////////////////////////////////////////////////////////////////
func (self *SRegion) fetchInfrastructure() error {
	// todo: implement me
	return nil
}

func (self *SRegion) GetId() string {
	return self.RegionId
}

func (self *SRegion) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_AWS_CN, self.RegionId)
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_AWS, self.RegionId)
}

func (self *SRegion) GetStatus() string {
	return models.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) Refresh() error {
	return nil
}

func (self *SRegion) IsEmulated() bool {
	return false
}

func (self *SRegion) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SRegion) GetLatitude() float32 {
	return 0.0
}

func (self *SRegion) GetLongitude() float32 {
	return 0.0
}

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	if self.izones == nil {
		if err := self.fetchInfrastructure(); err != nil {
			return nil, err
		}
	}
	return self.izones, nil
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

