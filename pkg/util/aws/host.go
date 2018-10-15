package aws

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"fmt"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/log"
)

type SHost struct {
	zone *SZone
}

func (self *SHost) GetId() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerId, self.zone.GetId())
}

func (self *SHost) GetName() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerName, self.zone.GetId())
}

func (self *SHost) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerId, self.zone.GetId())
}

func (self *SHost) GetStatus() string {
	return models.HOST_STATUS_RUNNING
}

func (self *SHost) Refresh() error {
	return nil
}

func (self *SHost) IsEmulated() bool {
	return true
}

func (self *SHost) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms := make([]SInstance, 0)
	vms, _, err := self.zone.region.GetInstances(self.zone.ZoneId, nil, len(vms), 50)
	if err != nil {
		return nil, err
	}

	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := 0; i < len(vms); i += 1 {
		vms[i].host = self
		ivms[i] = &vms[i]
	}
	return ivms, nil
}

func (self *SHost) GetIVMById(gid string) (cloudprovider.ICloudVM, error) {
	ivms, _, err := self.zone.region.GetInstances(self.zone.ZoneId, []string{gid}, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(ivms) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if len(ivms) > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	ivms[0].host = self
	return &ivms[0], nil
}

func (self *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.zone.GetIWires()
}

func (self *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return self.zone.GetIStorages()
}

func (self *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return self.zone.GetIStorageById(id)
}

func (self *SHost) GetEnabled() bool {
	return true
}

func (self *SHost) GetHostStatus() string {
	return models.HOST_ONLINE
}

func (self *SHost) GetAccessIp() string {
	return ""
}

func (self *SHost) GetAccessMac() string {
	return ""
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_AWS), "manufacture")
	return info
}

func (self *SHost) GetSN() string {
	panic("implement me")
}

func (self *SHost) GetCpuCount() int8 {
	return 0
}

func (self *SHost) GetNodeCount() int8 {
	return 0
}

func (self *SHost) GetCpuDesc() string {
	return ""
}

func (self *SHost) GetCpuMhz() int {
	return 0
}

func (self *SHost) GetMemSizeMB() int {
	return 0
}

func (self *SHost) GetStorageSizeMB() int {
	return 0
}

func (self *SHost) GetStorageType() string {
	return models.DISK_TYPE_HYBRID
}

func (self *SHost) GetHostType() string {
	return models.HOST_TYPE_AWS
}

func (self *SHost) GetManagerId() string {
	return self.zone.region.client.providerId
}

func (self *SHost) GetInstanceById(instanceId string) (*SInstance, error) {
	inst, err := self.zone.region.GetInstance(instanceId)
	if err != nil {
		return nil, err
	}
	inst.host = self
	return inst, nil
}

func (self *SHost) CreateVM(name, imgId string, sysDiskSize, cpu, memMB int, networkId, ipAddr, desc,
	passwd, storageType string, diskSizes []int, publicKey string, secgroupId string) (cloudprovider.ICloudVM, error) {
	if len(publicKey) < 0 {
		return nil, fmt.Errorf("AWS instance create error: keypair required")
	}

	if len(passwd) > 0 {
		log.Debugf("Ignored: AWS not support password.Use keypair instand")
	}

	vmId, err := self._createVM(name, imgId, sysDiskSize, cpu, memMB, networkId, ipAddr, desc, passwd, storageType, diskSizes, publicKey, secgroupId)
	if err != nil {
		return nil, err
	}

	vm, err := self.GetInstanceById(vmId)
	if err != nil {
		return nil, err
	}

	return vm, err
}

func (self *SHost) _createVM(name, imgId string, sysDiskSize, cpu, memMB int,
	networkId, ipAddr, desc, passwd,
	storageType string, diskSizes []int, publicKey string, secgroupId string) (string, error) {
	// 网络配置及安全组绑定
	// todo:// https://www.guru99.com/creating-amazon-ec2-instance.html
	self.zone.getNetworkById(networkId)
	// 同步keypair

	// 镜像及硬盘配置

	// 匹配实例类型

	// 创建实例
	return "", fmt.Errorf("Failed to create, specification not supported")
}