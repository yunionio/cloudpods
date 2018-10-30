package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SHost struct {
	zone *SZone
}

const (
	DEFAULT_USER = "yunion"
)

func (self *SHost) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SHost) GetId() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerId, self.zone.GetId())
}

func (self *SHost) GetName() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerName, self.zone.region.Name)
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
	vmId, err := self._createVM(name, imgId, sysDiskSize, cpu, memMB, nicId, ipAddr, desc, passwd, storageType, diskSizes, publicKey)
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

func (self *SHost) _createVM(name string, imgId string, sysDiskSize int, cpu int, memMB int, nicId string, ipAddr string, desc string, passwd string, storageType string, diskSizes []int, publicKey string) (string, error) {
	computeClient := compute.NewVirtualMachinesClientWithBaseURI(self.zone.region.client.baseUrl, self.zone.region.client.subscriptionId)
	computeClient.Authorizer = self.zone.region.client.authorizer

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

	osDiskName := fmt.Sprintf("vdisk_%s_%d", name, time.Now().UnixNano())
	DataDisks := make([]compute.DataDisk, 0)
	for i := 0; i < len(diskSizes); i++ {
		diskName := fmt.Sprintf("vdisk_%s_%d", name, time.Now().UnixNano())
		size := int32(diskSizes[i])
		lun := int32(i)
		DataDisks = append(DataDisks, compute.DataDisk{
			Name:         &diskName,
			DiskSizeGB:   &size,
			CreateOption: compute.DiskCreateOptionTypesEmpty,
			Lun:          &lun,
		})
	}

	AdminUsername := DEFAULT_USER

	NetworkInterfaceReferences := []compute.NetworkInterfaceReference{
		{ID: &nicId},
	}

	osType := compute.OperatingSystemTypes(image.GetOsType())
	DiskSizeGB := int32(sysDiskSize)

	// bootDiagnostics := true
	// diagnosticsProfile := compute.DiagnosticsProfile{
	// 	BootDiagnostics: &compute.BootDiagnostics{
	// 		Enabled: &bootDiagnostics,
	// 		//StorageURI:
	// 	},
	// }
	sshKeys := []compute.SSHPublicKey{compute.SSHPublicKey{KeyData: &publicKey}}
	properties := compute.VirtualMachineProperties{
		HardwareProfile: &compute.HardwareProfile{},
		StorageProfile: &compute.StorageProfile{
			ImageReference: &compute.ImageReference{ID: &image.ID},
			OsDisk: &compute.OSDisk{
				Caching: compute.CachingTypesReadWrite,
				ManagedDisk: &compute.ManagedDiskParameters{
					StorageAccountType: compute.StorageAccountTypes(storage.storageType),
				},
				Name:         &osDiskName,
				CreateOption: compute.DiskCreateOptionTypesEmpty,
				OsType:       osType,
				DiskSizeGB:   &DiskSizeGB,
			},
			DataDisks: &DataDisks,
		},

		OsProfile: &compute.OSProfile{
			ComputerName:       &name,
			AdminUsername:      &AdminUsername,
			AdminPassword:      &passwd,
			LinuxConfiguration: &compute.LinuxConfiguration{},
		},
		NetworkProfile: &compute.NetworkProfile{NetworkInterfaces: &NetworkInterfaceReferences},
	}

	if len(publicKey) > 0 {
		properties.OsProfile.LinuxConfiguration.SSH = &compute.SSHConfiguration{PublicKeys: &sshKeys}
	}

	params := compute.VirtualMachine{Location: &self.zone.region.Name, Name: &name, VirtualMachineProperties: &properties}
	//log.Debugf("Create instance params: %s", jsonutils.Marshal(params).PrettyString())
	for _, profile := range self.zone.region.getHardwareProfile(cpu, memMB) {
		params.HardwareProfile.VMSize = compute.VirtualMachineSizeTypes(profile)
		log.Debugf("Try HardwareProfile : %s", profile)
		instanceId, resourceGroup, instanceName := pareResourceGroupWithName(name, INSTANCE_RESOURCE)
		result, err := computeClient.CreateOrUpdate(context.Background(), resourceGroup, instanceName, params)
		if err != nil {
			log.Errorf("Failed for %s: %s", profile, err)
		} else if _, err := result.Done(computeClient.Client); err != nil {
			return "", err
		} else {
			return instanceId, nil
		}
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
	return self.zone.GetIStorages()
}

func (self *SHost) GetIVMById(instanceId string) (cloudprovider.ICloudVM, error) {
	if instance, err := self.zone.region.GetInstance(instanceId); err != nil {
		return nil, err
	} else {
		instance.host = self
		return instance, nil
	}
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
	if vms, err := self.zone.region.GetInstances(); err != nil {
		return nil, err
	} else {
		ivms := make([]cloudprovider.ICloudVM, len(vms))
		for i := 0; i < len(vms); i++ {
			vms[i].host = self
			ivms[i] = &vms[i]
			log.Debugf("find vm %s for host %s", vms[i].GetName(), self.GetName())
		}
		return ivms, nil
	}
}

func (self *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.zone.GetIWires()
}

func (self *SHost) GetManagerId() string {
	return self.zone.region.client.providerId
}
