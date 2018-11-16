package azure

import (
	"fmt"
	"time"

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
	return fmt.Sprintf("%s/%s", self.zone.region.GetName(), self.zone.region.client.subscriptionId)
}

func (self *SHost) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.zone.region.GetGlobalId(), self.zone.region.SubscriptionID)
}

func (self *SHost) IsEmulated() bool {
	return true
}

func (self *SHost) GetStatus() string {
	return models.HOST_STATUS_RUNNING
}

func (self *SHost) Refresh() error {
	return nil
}

func (self *SHost) CreateVM(name string, imgId string, sysDiskSize int, cpu int, memMB int, networkId string, ipAddr string, desc string, passwd string, storageType string, diskSizes []int, publicKey string, secgroupId string) (cloudprovider.ICloudVM, error) {
	nicId := ""
	if net := self.zone.getNetworkById(networkId); net == nil {
		return nil, fmt.Errorf("invalid network ID %s", networkId)
	} else if nic, err := self.zone.region.CreateNetworkInterface(fmt.Sprintf("%s-ipconfig", name), ipAddr, net.GetId(), secgroupId); err != nil {
		return nil, err
	} else {
		nicId = nic.ID
	}
	vmId, err := self._createVM(name, imgId, int32(sysDiskSize), cpu, memMB, nicId, ipAddr, desc, passwd, storageType, diskSizes, publicKey)
	if err != nil {
		self.zone.region.DeleteNetworkInterface(nicId)
		return nil, err
	}
	if vm, err := self.zone.region.GetInstance(vmId); err != nil {
		return nil, err
	} else {
		vm.host = self
		return vm, err
	}
}

func (self *SHost) _createVM(name string, imgId string, sysDiskSize int32, cpu int, memMB int, nicId string, ipAddr string, desc string, passwd string, storageType string, diskSizes []int, publicKey string) (string, error) {
	image, err := self.zone.region.GetImage(imgId)
	if err != nil {
		log.Errorf("Get Image %s fail %s", imgId, err)
		return "", err
	}

	if image.Properties.ProvisioningState != ImageStatusAvailable {
		log.Errorf("image %s status %s", imgId, image.Properties.ProvisioningState)
		return "", fmt.Errorf("image not ready")
	}
	storage, err := self.zone.getStorageByType(storageType)
	if err != nil {
		return "", fmt.Errorf("Storage %s not avaiable: %s", storageType, err)
	}

	instance := SInstance{
		Name:     name,
		Location: self.zone.region.Name,
		Properties: VirtualMachineProperties{
			HardwareProfile: HardwareProfile{
				VMSize: "",
			},
			OsProfile: OsProfile{
				ComputerName:  name,
				AdminUsername: DEFAULT_USER,
				AdminPassword: passwd,
			},
			NetworkProfile: NetworkProfile{
				NetworkInterfaces: []NetworkInterfaceReference{
					NetworkInterfaceReference{
						ID: nicId,
					},
				},
			},
			StorageProfile: StorageProfile{
				ImageReference: ImageReference{
					ID: image.ID,
				},
				OsDisk: OSDisk{
					Name:    fmt.Sprintf("vdisk_%s_%d", name, time.Now().UnixNano()),
					Caching: "ReadWrite",
					ManagedDisk: &ManagedDiskParameters{
						StorageAccountType: storage.Name,
					},
					CreateOption: "FromImage",
					DiskSizeGB:   &sysDiskSize,
					OsType:       image.GetOsType(),
				},
			},
		},
		Type: "Microsoft.Compute/virtualMachines",
	}
	if len(publicKey) > 0 {
		instance.Properties.OsProfile.LinuxConfiguration = &LinuxConfiguration{
			DisablePasswordAuthentication: false,
			SSH: &SSHConfiguration{
				PublicKeys: []SSHPublicKey{
					SSHPublicKey{KeyData: publicKey},
				},
			},
		}
	}

	dataDisks := []DataDisk{}
	for i := 0; i < len(diskSizes); i++ {
		diskName := fmt.Sprintf("vdisk_%s_%d", name, time.Now().UnixNano())
		size := int32(diskSizes[i])
		lun := int32(i)
		dataDisks = append(dataDisks, DataDisk{
			Name:         diskName,
			DiskSizeGB:   &size,
			CreateOption: "Empty",
			Lun:          lun,
		})
	}
	if len(dataDisks) > 0 {
		instance.Properties.StorageProfile.DataDisks = dataDisks
	}

	for _, profile := range self.zone.region.getHardwareProfile(cpu, memMB) {
		instance.Properties.HardwareProfile.VMSize = profile
		log.Debugf("Try HardwareProfile : %s", profile)
		err := self.zone.region.client.Create(jsonutils.Marshal(instance), &instance)
		if err != nil {
			log.Errorf("Failed for %s: %s", profile, err)
			continue
		}
		return instance.ID, nil
	}
	return "", fmt.Errorf("Failed to create, specification not supported")
}

func (self *SHost) GetAccessIp() string {
	return ""
}

func (self *SHost) GetAccessMac() string {
	return ""
}

func (self *SHost) GetCpuCount() int8 {
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
func (self *SHost) GetEnabled() bool {
	return true
}

func (self *SHost) GetHostStatus() string {
	return models.HOST_ONLINE
}
func (self *SHost) GetNodeCount() int8 {
	return 0
}

func (self *SHost) GetHostType() string {
	return models.HOST_TYPE_AZURE
}

func (self *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return self.zone.GetIStorageById(id)
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_AZURE), "manufacture")
	return info
}

func (self *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return self.zone.istorages, nil
}

func (self *SHost) GetIVMById(instanceId string) (cloudprovider.ICloudVM, error) {
	instance, err := self.zone.region.GetInstance(instanceId)
	if err != nil {
		return nil, err
	}
	instance.host = self
	return instance, nil
}

func (self *SHost) GetStorageSizeMB() int {
	return 0
}

func (self *SHost) GetStorageType() string {
	return models.DISK_TYPE_HYBRID
}

func (self *SHost) GetSN() string {
	return ""
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := self.zone.region.GetInstances()
	if err != nil {
		return nil, err
	}
	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := 0; i < len(vms); i++ {
		vms[i].host = self
		ivms[i] = &vms[i]
		log.Debugf("find vm %s for host %s", vms[i].GetName(), self.GetName())
	}
	return ivms, nil
}

func (self *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.zone.GetIWires()
}

func (self *SHost) GetManagerId() string {
	return self.zone.region.client.providerId
}
