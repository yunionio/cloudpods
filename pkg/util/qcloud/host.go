package qcloud

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SHost struct {
	zone *SZone
}

func (self *SHost) GetMetadata() *jsonutils.JSONDict {
	return nil
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

func (self *SHost) GetInstanceById(instanceId string) (*SInstance, error) {
	inst, err := self.zone.region.GetInstance(instanceId)
	if err != nil {
		return nil, err
	}
	inst.host = self
	return inst, nil
}

func (self *SHost) CreateVM(name string, imgId string, sysDiskSize int, cpu int, memMB int,
	vswitchId string, ipAddr string, desc string, passwd string,
	storageType string, diskSizes []int, publicKey string, secgroupId string, userData string) (cloudprovider.ICloudVM, error) {
	vmId, err := self._createVM(name, imgId, sysDiskSize, cpu, memMB, vswitchId, ipAddr, desc, passwd, storageType, diskSizes, publicKey, secgroupId, userData)
	if err != nil {
		return nil, err
	}
	vm, err := self.GetInstanceById(vmId)
	if err != nil {
		return nil, err
	}
	return vm, err
}

func (self *SHost) _createVM(name string, imgId string, sysDiskSize int, cpu int, memMB int,
	networkId string, ipAddr string, desc string, passwd string,
	storageType string, diskSizes []int, publicKey string, secgroupId string, userData string) (string, error) {

	net := self.zone.getNetworkById(networkId)
	if net == nil {
		return "", fmt.Errorf("invalid network ID %s", networkId)
	}
	if net.wire == nil {
		log.Errorf("network's wire is empty")
		return "", fmt.Errorf("network's wire is empty")
	}
	if net.wire.vpc == nil {
		log.Errorf("network's wire' vpc is empty")
		return "", fmt.Errorf("network's wire's vpc is empty")
	}

	var err error

	// if len(secgroupId) == 0 {
	// 	secgroups, err := net.wire.vpc.GetISecurityGroups()
	// 	if err != nil {
	// 		return "", fmt.Errorf("get security group error %s", err)
	// 	}

	// 	if len(secgroups) == 0 {
	// 		secId, err := self.zone.region.createDefaultSecurityGroup(net.wire.vpc.VpcId)
	// 		if err != nil {
	// 			return "", fmt.Errorf("no secgroup for vpc and failed to create a default One!!")
	// 		} else {
	// 			secgroupId = secId
	// 		}
	// 	} else {
	// 		secgroupId = secgroups[0].GetId()
	// 	}
	// }

	keypair := ""
	if len(publicKey) > 0 {
		keypair, err = self.zone.region.syncKeypair(publicKey)
		if err != nil {
			return "", err
		}
	}

	img, err := self.zone.region.GetImage(imgId)
	if err != nil {
		log.Errorf("getiamge fail %s", err)
		return "", err
	}
	if img.ImageState != ImageStatusAvailable {
		log.Errorf("image %s status %s", imgId, img.ImageState)
		return "", fmt.Errorf("image not ready")
	}

	err = self.zone.validateStorageType(storageType)
	if err != nil {
		return "", fmt.Errorf("Storage %s not avaiable: %s", storageType, err)
	}

	disks := make([]SDisk, len(diskSizes)+1)
	disks[0].DiskSize = img.ImageSize
	if sysDiskSize > 0 && sysDiskSize > img.ImageSize {
		disks[0].DiskSize = sysDiskSize
	}
	if disks[0].DiskSize < 50 {
		disks[0].DiskSize = 50
	}
	disks[0].DiskType = strings.ToUpper(storageType)

	for i, sz := range diskSizes {
		disks[i+1].DiskSize = sz
		disks[i+1].DiskType = strings.ToUpper(storageType)
	}

	instanceTypes, err := self.zone.region.GetMatchInstanceTypes(cpu, memMB, 0, self.zone.Zone)
	if err != nil {
		return "", err
	}
	if len(instanceTypes) == 0 {
		return "", fmt.Errorf("instance type %dC%dMB not avaiable", cpu, memMB)
	}

	for _, instType := range instanceTypes {
		instanceTypeId := instType.InstanceType
		log.Debugf("Try instancetype : %s", instanceTypeId)
		vmId, err := self.zone.region.CreateInstance(name, imgId, instanceTypeId, secgroupId, self.zone.Zone, desc, passwd, disks, networkId, ipAddr, keypair, userData)
		if err != nil {
			log.Errorf("Failed for %s: %s", instanceTypeId, err)
		} else {
			return vmId, nil
		}
	}

	return "", fmt.Errorf("Failed to create, specification not supported")
}

func (self *SHost) Refresh() error {
	return nil
}

func (self *SHost) GetAccessIp() string {
	return ""
}

func (self *SHost) GetAccessMac() string {
	return ""
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

func (self *SHost) GetEnabled() bool {
	return true
}

func (self *SHost) GetStatus() string {
	return models.HOST_STATUS_RUNNING
}

func (self *SHost) GetHostStatus() string {
	return models.HOST_ONLINE
}

func (self *SHost) GetHostType() string {
	return models.HOST_TYPE_QCLOUD
}

func (self *SHost) GetStorageType() string {
	return models.DISK_TYPE_HYBRID
}

func (self *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return self.zone.GetIStorageById(id)
}

func (self *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return self.zone.GetIStorages()
}

func (self *SHost) GetIVMById(gid string) (cloudprovider.ICloudVM, error) {
	if len(gid) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	parts, _, err := self.zone.region.GetInstances(self.zone.Zone, []string{gid}, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if len(parts) > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	parts[0].host = self
	return &parts[0], nil
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms := make([]SInstance, 0)
	for {
		parts, total, err := self.zone.region.GetInstances(self.zone.Zone, nil, len(vms), 50)
		if err != nil {
			return nil, err
		}
		vms = append(vms, parts...)
		if len(vms) >= total {
			break
		}
	}
	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := 0; i < len(vms); i++ {
		vms[i].host = self
		ivms[i] = &vms[i]
	}
	return ivms, nil
}

func (self *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.zone.GetIWires()
}

func (self *SHost) GetManagerId() string {
	return self.zone.region.client.providerId
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_QCLOUD), "manufacture")
	return info
}

func (self *SHost) IsEmulated() bool {
	return true
}
