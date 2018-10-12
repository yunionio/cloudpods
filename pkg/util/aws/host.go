package aws

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SHost struct {
	zone *SZone
}

func (self *SHost) GetId() string {
	panic("implement me")
}

func (self *SHost) GetName() string {
	panic("implement me")
}

func (self *SHost) GetGlobalId() string {
	panic("implement me")
}

func (self *SHost) GetStatus() string {
	panic("implement me")
}

func (self *SHost) Refresh() error {
	panic("implement me")
}

func (self *SHost) IsEmulated() bool {
	panic("implement me")
}

func (self *SHost) GetMetadata() *jsonutils.JSONDict {
	panic("implement me")
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	panic("implement me")
}

func (self *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	panic("implement me")
}

func (self *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	panic("implement me")
}

func (self *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	panic("implement me")
}

func (self *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	panic("implement me")
}

func (self *SHost) GetEnabled() bool {
	panic("implement me")
}

func (self *SHost) GetHostStatus() string {
	panic("implement me")
}

func (self *SHost) GetAccessIp() string {
	panic("implement me")
}

func (self *SHost) GetAccessMac() string {
	panic("implement me")
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	panic("implement me")
}

func (self *SHost) GetSN() string {
	panic("implement me")
}

func (self *SHost) GetCpuCount() int8 {
	panic("implement me")
}

func (self *SHost) GetNodeCount() int8 {
	panic("implement me")
}

func (self *SHost) GetCpuDesc() string {
	panic("implement me")
}

func (self *SHost) GetCpuMhz() int {
	panic("implement me")
}

func (self *SHost) GetMemSizeMB() int {
	panic("implement me")
}

func (self *SHost) GetStorageSizeMB() int {
	panic("implement me")
}

func (self *SHost) GetStorageType() string {
	panic("implement me")
}

func (self *SHost) GetHostType() string {
	panic("implement me")
}

func (self *SHost) GetManagerId() string {
	panic("implement me")
}

func (self *SHost) CreateVM(name string, imgId string, sysDiskSize int, cpu int, memMB int, vswitchId string, ipAddr string, desc string,
	passwd string, storageType string, diskSizes []int, publicKey string, extSecGrpId string) (cloudprovider.ICloudVM, error) {
	panic("implement me")
}
