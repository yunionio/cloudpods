package huawei

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type SHost struct {
	zone *SZone

	projectId string
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
	limit := 100
	offset := 0
	for {
		parts, count, err := self.zone.region.GetInstances(offset, limit)
		if err != nil {
			return nil, err
		}

		for _, vm := range parts {
			vm.host = self
			vms = append(vms, vm)
		}

		if count < limit {
			break
		}

		offset += limit
	}

	filtedVms := make([]SInstance, 0)
	for i := range vms {
		vm := vms[i]
		if vm.OSEXTAZAvailabilityZone == self.zone.GetId() {
			filtedVms = append(filtedVms, vm)
		}
	}

	ivms := make([]cloudprovider.ICloudVM, len(filtedVms))
	for i := 0; i < len(filtedVms); i += 1 {
		filtedVms[i].host = self
		ivms[i] = &filtedVms[i]
	}
	return ivms, nil
}

func (self *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vm, err := self.zone.region.GetInstanceByID(id)
	vm.host = self
	return &vm, err
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
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_HUAWEI), "manufacture")
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
	return models.HOST_TYPE_HUAWEI
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return HUAWEI_API_VERSION
}

func (self *SHost) GetManagerId() string {
	return self.zone.region.client.providerId
}

func (self *SHost) GetInstanceById(instanceId string) (*SInstance, error) {
	instance, err := self.zone.region.GetInstanceByID(instanceId)
	if err != nil {
		return nil, err
	}

	instance.host = self
	return &instance, nil
}

func (self *SHost) CreateVM(name string, imgId string, sysDiskSize int, cpu int, memMB int, networkId string, ipAddr string, desc string,
	passwd string, storageType string, diskSizes []int, publicKey string, extSecGrpId string, userData string, bc *billing.SBillingCycle) (cloudprovider.ICloudVM, error) {
	vmId, err := self._createVM(name, imgId, sysDiskSize, cpu, memMB, "", networkId, ipAddr, desc, passwd, storageType, diskSizes, publicKey, extSecGrpId, userData, bc)
	if err != nil {
		return nil, err
	}

	vm, err := self.GetInstanceById(vmId)
	if err != nil {
		return nil, err
	}

	return vm, err
}

func (self *SHost) CreateVM2(name string, imgId string, sysDiskSize int, instanceType string, networkId string, ipAddr string, desc string,
	passwd string, storageType string, diskSizes []int, publicKey string, extSecGrpId string, userData string, bc *billing.SBillingCycle) (cloudprovider.ICloudVM, error) {
	vmId, err := self._createVM(name, imgId, sysDiskSize, 0, 0, instanceType, networkId, ipAddr, desc, passwd, storageType, diskSizes, publicKey, extSecGrpId, userData, bc)
	if err != nil {
		return nil, err
	}

	vm, err := self.GetInstanceById(vmId)
	if err != nil {
		return nil, err
	}

	return vm, err
}

func (self *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SHost) _createVM(name string, imgId string, sysDiskSize int, cpu int, memMB int, instanceType string,
	networkId string, ipAddr string, desc string, passwd string,
	storageType string, diskSizes []int, publicKey string, secgroupId string,
	userData string, bc *billing.SBillingCycle) (string, error) {
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

	// 同步keypair
	var err error
	keypair := ""
	if len(publicKey) > 0 {
		keypair, err = self.zone.region.syncKeypair(publicKey)
		if err != nil {
			return "", err
		}
	}

	//  镜像及硬盘配置
	img, err := self.zone.region.GetImage(imgId)
	if err != nil {
		log.Errorf("getiamge %s fail %s", imgId, err)
		return "", err
	}
	if img.Status != ImageStatusActive {
		log.Errorf("image %s status %s", imgId, img.Status)
		return "", fmt.Errorf("image not ready")
	}

	disks := make([]SDisk, len(diskSizes)+1)
	disks[0].SizeGB = img.SizeGB
	if sysDiskSize > 0 && sysDiskSize > img.SizeGB {
		disks[0].SizeGB = sysDiskSize
	}
	disks[0].VolumeType = storageType

	for i, sz := range diskSizes {
		disks[i+1].SizeGB = sz
		disks[i+1].VolumeType = storageType
	}

	secgroup, err := self.zone.region.GetSecurityGroupDetails(secgroupId)
	if err != nil {
		return "", err
	}

	// 创建实例
	if len(instanceType) > 0 {
		log.Debugf("Try instancetype : %s", instanceType)
		vmId, err := self.zone.region.CreateInstance(name, imgId, instanceType, networkId, secgroupId, secgroup.VpcID, self.zone.GetId(), desc, disks, ipAddr, keypair, passwd, userData, bc)
		if err != nil {
			log.Errorf("Failed for %s: %s", instanceType, err)
			return "", fmt.Errorf("Failed to create, specification %s not supported", instanceType)
		} else {
			return vmId, nil
		}
	}

	// 匹配实例类型
	instanceTypes, err := self.zone.region.GetMatchInstanceTypes(cpu, memMB, self.zone.GetId())
	if err != nil {
		return "", err
	}
	if len(instanceTypes) == 0 {
		return "", fmt.Errorf("instance type %dC%dMB not avaiable", cpu, memMB)
	}

	for _, instType := range instanceTypes {
		instanceTypeId := instType.Name
		log.Debugf("Try instancetype : %s", instanceTypeId)
		vmId, err := self.zone.region.CreateInstance(name, imgId, instanceType, networkId, secgroupId, secgroup.VpcID, self.zone.GetId(), desc, disks, ipAddr, keypair, passwd, userData, bc)
		if err != nil {
			log.Errorf("Failed for %s: %s", instanceTypeId, err)
		} else {
			return vmId, nil
		}
	}

	return "", fmt.Errorf("Failed to create, specification not supported")
}
