package aws

import (
	"fmt"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
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
	log.Debugf("GetIVMById %s", gid)
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
	return ""
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
	passwd, storageType string, diskSizes []int, publicKey string, secgroupId string, userData string) (cloudprovider.ICloudVM, error) {
	vmId, err := self._createVM(name, imgId, sysDiskSize, cpu, memMB, "", networkId, ipAddr, desc, passwd, storageType, diskSizes, publicKey, secgroupId, userData)
	if err != nil {
		return nil, err
	}

	vm, err := self.GetInstanceById(vmId)
	if err != nil {
		return nil, err
	}

	return vm, err
}

func (self *SHost) CreateVM2(name, imgId string, sysDiskSize int, instanceType string, networkId, ipAddr, desc,
	passwd, storageType string, diskSizes []int, publicKey string, secgroupId string, userData string) (cloudprovider.ICloudVM, error) {
	vmId, err := self._createVM(name, imgId, sysDiskSize, 0, 0, instanceType, networkId, ipAddr, desc, passwd, storageType, diskSizes, publicKey, secgroupId, userData)
	if err != nil {
		return nil, err
	}

	vm, err := self.GetInstanceById(vmId)
	if err != nil {
		return nil, err
	}

	return vm, err
}

func (self *SHost) _createVM(name, imgId string, sysDiskSize int, cpu, memMB int, instanceType string,
	networkId, ipAddr, desc, passwd,
	storageType string, diskSizes []int, publicKey string, secgroupId string, userData string) (string, error) {
	// 网络配置及安全组绑定
	net := self.zone.getNetworkById(networkId)
	if net == nil {
		return "", fmt.Errorf("invalid network ID %s", networkId)
	}

	if net.wire == nil {
		log.Errorf("network's wire is empty")
		return "", fmt.Errorf("network's wire is empty")
	}

	if net.wire.vpc == nil {
		log.Errorf("wire's vpc is empty")
		return "", fmt.Errorf("wire's vpc is empty")
	}

	if len(secgroupId) == 0 {
		secgroups, err := net.wire.vpc.GetISecurityGroups()
		if err != nil {
			return "", fmt.Errorf("get security group error %s", err)
		}

		if len(secgroups) == 0 {
			// aws 默认就已经创建好了一个默认安全组。正常情况下并不需要手动创建
			secId, err := self.zone.region.createDefaultSecurityGroup(net.wire.vpc.VpcId)
			if err != nil {
				return "", fmt.Errorf("no secgroup for vpc and failed to create a default One!!")
			} else {
				secgroupId = secId
			}
		} else {
			secgroupId = secgroups[0].GetId()
		}
	}
	// 同步keypair
	var err error
	keypair := ""
	if len(publicKey) > 0 {
		keypair, err = self.zone.region.syncKeypair(publicKey)
		if err != nil {
			return "", err
		}
	}

	// 镜像及硬盘配置
	img, err := self.zone.region.GetImage(imgId)
	if err != nil {
		log.Errorf("getiamge %s fail %s", imgId, err)
		return "", err
	}
	if img.Status != ImageStatusAvailable {
		log.Errorf("image %s status %s", imgId, img.Status)
		return "", fmt.Errorf("image not ready")
	}

	disks := make([]SDisk, len(diskSizes)+1)
	disks[0].Size = img.Size
	if sysDiskSize > 0 && sysDiskSize > img.Size {
		disks[0].Size = sysDiskSize
	}
	disks[0].Category = storageType

	for i, sz := range diskSizes {
		disks[i+1].Size = sz
		disks[i+1].Category = storageType
	}

	// 创建实例
	if len(instanceType) > 0 {
		log.Debugf("Try instancetype : %s", instanceType)
		vmId, err := self.zone.region.CreateInstance(name, imgId, instanceType, networkId, secgroupId, self.zone.ZoneId, desc, disks, ipAddr, keypair, userData)
		if err != nil {
			log.Errorf("Failed for %s: %s", instanceType, err)
			return "", fmt.Errorf("Failed to create, specification %s not supported", instanceType)
		} else {
			return vmId, nil
		}
	}

	// 匹配实例类型
	instanceTypes, err := self.zone.region.GetMatchInstanceTypes(cpu, memMB, 0, self.zone.ZoneId)
	if err != nil {
		return "", err
	}
	if len(instanceTypes) == 0 {
		return "", fmt.Errorf("instance type %dC%dMB not avaiable", cpu, memMB)
	}

	for _, instType := range instanceTypes {
		instanceTypeId := instType.InstanceTypeId
		log.Debugf("Try instancetype : %s", instanceTypeId)
		vmId, err := self.zone.region.CreateInstance(name, imgId, instanceTypeId, networkId, secgroupId, self.zone.ZoneId, desc, disks, ipAddr, keypair, userData)
		if err != nil {
			log.Errorf("Failed for %s: %s", instanceTypeId, err)
		} else {
			return vmId, nil
		}
	}

	return "", fmt.Errorf("Failed to create, specification not supported")
}

func (self *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return AWS_API_VERSION
}
