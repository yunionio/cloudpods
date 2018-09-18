package azure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/secrules"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
)

const (
	DEFAULT_EXTENSION_NAME = "enablevmaccess"
)

type HardwareProfile struct {
	VMSize string
}

type ImageReference struct {
	Publisher string
	Offer     string
	Sku       string
	Version   string
	ID        string
}

type OperatingSystemTypes string

const (
	// Linux ...
	Linux OperatingSystemTypes = "Linux"
	// Windows ...
	Windows OperatingSystemTypes = "Windows"
)

type OSDisk struct {
	OsType      OperatingSystemTypes
	Name        string
	DiskSizeGB  int32
	ManagedDisk ManagedDiskParameters
}

type VirtualHardDisk struct {
	URI string
}

type CachingTypes string

const (
	// CachingTypesNone ...
	CachingTypesNone CachingTypes = "None"
	// CachingTypesReadOnly ...
	CachingTypesReadOnly CachingTypes = "ReadOnly"
	// CachingTypesReadWrite ...
	CachingTypesReadWrite CachingTypes = "ReadWrite"
)

type ManagedDiskParameters struct {
	StorageAccountType string
	ID                 string
}

type DataDisk struct {
	Lun         int32
	Name        string
	Vhd         VirtualHardDisk
	Caching     CachingTypes
	DiskSizeGB  int32
	ManagedDisk ManagedDiskParameters
}

type StorageProfile struct {
	ImageReference ImageReference
	OsDisk         OSDisk
	DataDisks      []DataDisk
}

type SSHPublicKey struct {
	Path    string
	KeyData string
}

type SSHConfiguration struct {
	PublicKeys []SSHPublicKey
}

type LinuxConfiguration struct {
	DisablePasswordAuthentication bool
	SSH                           SSHConfiguration
}

type SubResource struct {
	ID string
}

type VaultCertificate struct {
	CertificateURL   string
	CertificateStore string
}

type VaultSecretGroup struct {
	SourceVault       SubResource
	VaultCertificates []VaultCertificate
}

type OsProfile struct {
	ComputerName       string
	AdminUsername      string
	AdminPassword      string
	CustomData         string
	LinuxConfiguration LinuxConfiguration
	Secrets            []VaultSecretGroup
}

type NetworkInterfaceReference struct {
	ID string
}

type NetworkProfile struct {
	NetworkInterfaces []NetworkInterfaceReference
}

type InstanceViewStatus struct {
	Code          string
	Level         string
	DisplayStatus string
	Message       string
	Time          time.Time
}

type VirtualMachineInstanceView struct {
	ComputerName string
	OsName       string
	OsVersion    string
	Statuses     []InstanceViewStatus
}

type VirtualMachineProperties struct {
	HardwareProfile HardwareProfile
	StorageProfile  StorageProfile
	OsProfile       OsProfile
	NetworkProfile  NetworkProfile
	InstanceView    VirtualMachineInstanceView
	VmId            string
}

type SInstance struct {
	host *SHost

	idisks []cloudprovider.ICloudDisk

	CreationTime time.Time

	Properties VirtualMachineProperties
	ID         string
	Name       string
	Type       string
	Location   string
	vmSize     *SVMSize
	Tags       map[string]string
}

func (self *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	instance := SInstance{}
	computeClient := compute.NewVirtualMachinesClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	computeClient.Authorizer = self.client.authorizer
	_, resourceGroup, instanceName := pareResourceGroupWithName(instanceId, INSTANCE_RESOURCE)
	if len(instanceName) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if _instance, err := computeClient.Get(context.Background(), resourceGroup, instanceName, "instanceView"); err != nil {
		if _instance.Response.StatusCode == 404 {
			return nil, cloudprovider.ErrNotFound
		}
		return nil, err
	} else if err := jsonutils.Update(&instance, _instance); err != nil {
		return nil, err
	} else {
		return &instance, nil
	}
}

func (self *SRegion) GetInstances() ([]SInstance, error) {
	instances := make([]SInstance, 0)
	computeClient := compute.NewVirtualMachinesClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	computeClient.Authorizer = self.client.authorizer
	if instanceList, err := computeClient.ListAll(context.Background()); err != nil {
		return instances, err
	} else {
		for _, _instance := range instanceList.Values() {
			if *_instance.Location == self.Name {
				if instance, err := self.GetInstance(*_instance.ID); err != nil {
					return nil, err
				} else {
					instances = append(instances, *instance)
				}
			}
		}
	}
	return instances, nil
}

func (self *SRegion) doDeleteVM(instanceId string) error {
	_, resourceGroup, instanceName := pareResourceGroupWithName(instanceId, INSTANCE_RESOURCE)
	computeClient := compute.NewVirtualMachinesClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	computeClient.Authorizer = self.client.authorizer
	if resulte, err := computeClient.Delete(context.Background(), resourceGroup, instanceName); err != nil {
		return err
	} else if err := resulte.WaitForCompletion(context.Background(), computeClient.Client); err != nil {
		return err
	}
	return nil
}

func (self *SInstance) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	if osDistribution := self.Properties.StorageProfile.ImageReference.Publisher; len(osDistribution) > 0 {
		data.Add(jsonutils.NewString(osDistribution), "os_distribution")
	}
	if loginAccount := self.Properties.OsProfile.AdminUsername; len(loginAccount) > 0 {
		data.Add(jsonutils.NewString(loginAccount), "login_account")
	}
	if loginKey := self.Properties.OsProfile.AdminPassword; len(loginKey) > 0 {
		data.Add(jsonutils.NewString(loginKey), "login_key")
	}

	data.Add(jsonutils.NewString(self.Properties.HardwareProfile.VMSize), "price_key")
	return data
}

func (self *SInstance) GetHypervisor() string {
	return models.HYPERVISOR_AZURE
}

func (self *SInstance) IsEmulated() bool {
	return false
}

func (self *SInstance) getOsDisk() (*SDisk, error) {
	diskId := self.Properties.StorageProfile.OsDisk.ManagedDisk.ID
	if osDisk, err := self.getDiskWithStore(diskId); err != nil {
		log.Errorf("Failed to find instance %s os disk: %s", self.Name, diskId)
		return nil, err
	} else {
		return osDisk, nil
	}
}

func (self *SInstance) getDisks() ([]SDisk, error) {
	disks := make([]SDisk, 0)
	diskId := self.Properties.StorageProfile.OsDisk.ManagedDisk.ID
	if osdisk, err := self.getDiskWithStore(diskId); err != nil {
		log.Errorf("Failed to find instance %s os disk: %s", self.Name, diskId)
		return nil, err
	} else {
		disks = append(disks, *osdisk)
	}
	for _, _disk := range self.Properties.StorageProfile.DataDisks {
		if disk, err := self.getDiskWithStore(_disk.ManagedDisk.ID); err != nil {
			log.Errorf("Failed to find instance %s data disk: %s", self.Name, _disk.ManagedDisk.ID)
			return nil, err
		} else {
			disks = append(disks, *disk)
		}
	}
	return disks, nil
}

func (self *SInstance) getNics() ([]SInstanceNic, error) {
	nics := make([]SInstanceNic, 0)
	for _, _nic := range self.Properties.NetworkProfile.NetworkInterfaces {
		if nic, err := self.host.zone.region.GetNetworkInterfaceDetail(_nic.ID); err != nil {
			log.Errorf("Failed to find instance %s nic: %s", self.Name, _nic.ID)
			return nil, err
		} else {
			nic.instance = self
			nics = append(nics, *nic)
		}
	}
	return nics, nil
}

func (self *SInstance) Refresh() error {
	if instance, err := self.host.zone.region.GetInstance(self.ID); err != nil {
		return err
	} else {
		return jsonutils.Update(self, instance)
	}
}

func (self *SInstance) GetStatus() string {
	for _, statuses := range self.Properties.InstanceView.Statuses {
		if code := strings.Split(statuses.Code, "/"); len(code) == 2 {
			if code[0] == "PowerState" {
				switch code[1] {
				case "stopped":
					return models.VM_READY
				case "deallocated":
					return models.VM_DEALLOCATED
				case "running":
					return models.VM_RUNNING
				case "stopping":
					return models.VM_START_STOP
				case "starting":
					return models.VM_STARTING
				case "deleting":
					return models.VM_DELETING
				default:
					log.Errorf("Unknow instance status %s", code[1])
					return models.VM_UNKNOWN
				}
			}
		}
		if statuses.Level == "Error" {
			log.Errorf("Find error code: [%s] message: %s", statuses.Code, statuses.Message)
		}
	}
	return models.VM_UNKNOWN
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) AttachDisk(diskId string) error {
	if err := self.host.zone.region.AttachDisk(self.ID, diskId); err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, self.GetStatus(), 10*time.Second, 300*time.Second)
}

func (region *SRegion) UpdateInstance(instanceId string, params compute.VirtualMachineUpdate) error {
	computeClient := compute.NewVirtualMachinesClientWithBaseURI(region.client.baseUrl, region.client.subscriptionId)
	computeClient.Authorizer = region.client.authorizer
	_, resourceGroup, instanceName := pareResourceGroupWithName(instanceId, INSTANCE_RESOURCE)
	log.Errorf("update VM: %s", jsonutils.Marshal(params).PrettyString())
	if _, err := computeClient.Update(context.Background(), resourceGroup, instanceName, params); err != nil {
		return err
	}
	return nil
}

func (region *SRegion) AttachDisk(instanceId, diskId string) error {
	if instance, err := region.GetInstance(instanceId); err != nil {
		return err
	} else if disk, err := region.GetDisk(diskId); err != nil {
		return err
	} else {
		dataDisks := []compute.DataDisk{}
		maxLun := int32(0)
		for i := 0; i < len(instance.Properties.StorageProfile.DataDisks); i++ {
			_disk := instance.Properties.StorageProfile.DataDisks[i]
			if disk.ID == _disk.ManagedDisk.ID {
				return nil
			} else {
				if maxLun < _disk.Lun {
					maxLun = _disk.Lun
				}
				dataDisks = append(dataDisks, compute.DataDisk{
					Lun:          &_disk.Lun,
					CreateOption: compute.DiskCreateOptionTypesAttach,
					ManagedDisk: &compute.ManagedDiskParameters{
						ID: &_disk.ManagedDisk.ID,
					},
				})
			}
		}
		maxLun++
		dataDisks = append(dataDisks, compute.DataDisk{
			Lun:          &maxLun,
			CreateOption: compute.DiskCreateOptionTypesAttach,
			ManagedDisk: &compute.ManagedDiskParameters{
				ID: &disk.ID,
			},
		})
		params := compute.VirtualMachineUpdate{
			VirtualMachineProperties: &compute.VirtualMachineProperties{
				StorageProfile: &compute.StorageProfile{
					DataDisks: &dataDisks,
				},
			},
		}
		return region.UpdateInstance(instanceId, params)
	}
}

func (self *SInstance) DetachDisk(diskId string) error {
	if err := self.host.zone.region.DetachDisk(self.ID, diskId); err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, self.GetStatus(), 10*time.Second, 300*time.Second)
}

func (region *SRegion) DetachDisk(instanceId, diskId string) error {
	if instance, err := region.GetInstance(instanceId); err != nil {
		return err
	} else if disk, err := region.GetDisk(diskId); err != nil {
		return err
	} else {
		dataDisks := []compute.DataDisk{}
		for i := 0; i < len(instance.Properties.StorageProfile.DataDisks); i++ {
			if instance.Properties.StorageProfile.DataDisks[i].ManagedDisk.ID == disk.ID {
				continue
			}
			dataDisks = append(dataDisks, compute.DataDisk{
				Lun: &instance.Properties.StorageProfile.DataDisks[i].Lun,
				ManagedDisk: &compute.ManagedDiskParameters{
					ID: &instance.Properties.StorageProfile.DataDisks[i].ManagedDisk.ID,
				},
			})
		}
		params := compute.VirtualMachineUpdate{
			VirtualMachineProperties: &compute.VirtualMachineProperties{
				StorageProfile: &compute.StorageProfile{
					DataDisks: &dataDisks,
				},
			},
		}
		return region.UpdateInstance(instanceId, params)
	}
}

func (self *SInstance) ChangeConfig(instanceId string, ncpu int, vmem int) error {
	if err := self.host.zone.region.ChangeVMConfig(instanceId, ncpu, vmem); err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, self.GetStatus(), 10*time.Second, 300*time.Second)
}

func (region *SRegion) ChangeVMConfig(instanceId string, ncpu int, vmem int) error {
	for _, vmSize := range region.getHardwareProfile(ncpu, vmem) {
		params := compute.VirtualMachineUpdate{
			VirtualMachineProperties: &compute.VirtualMachineProperties{
				HardwareProfile: &compute.HardwareProfile{
					VMSize: compute.VirtualMachineSizeTypes(vmSize),
				},
			},
		}
		log.Debugf("Try HardwareProfile : %s", vmSize)
		if err := region.UpdateInstance(instanceId, params); err == nil {
			return nil
		}

	}
	return fmt.Errorf("Failed to change vm config, specification not supported")
}

func (self *SInstance) DeployVM(name string, password string, publicKey string, deleteKeypair bool, description string) error {
	return self.host.zone.region.DeployVM(self.ID, name, password, publicKey, deleteKeypair, description)
}

type VirtualMachineExtensionProperties struct {
	Publisher          string
	Type               string
	TypeHandlerVersion string
}

type SVirtualMachineExtension struct {
	Location   string
	Properties VirtualMachineExtensionProperties
}

func (region *SRegion) resetLoginInfo(instanceId string, setting map[string]string) error {
	_, resourceGroup, instanceName := pareResourceGroupWithName(instanceId, INSTANCE_RESOURCE)
	extensionClient := compute.NewVirtualMachineExtensionsClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
	extensionClient.Authorizer = region.client.authorizer

	publisher := "Microsoft.OSTCExtensions"
	Type := "VMAccessForLinux"
	typeHandlerVersion := "1.4"

	params := compute.VirtualMachineExtension{
		Location: &region.Name,
		VirtualMachineExtensionProperties: &compute.VirtualMachineExtensionProperties{
			Publisher:          &publisher,
			Type:               &Type,
			TypeHandlerVersion: &typeHandlerVersion,
			ProtectedSettings:  setting,
		},
	}

	if result, err := extensionClient.CreateOrUpdate(context.Background(), resourceGroup, instanceName, DEFAULT_EXTENSION_NAME, params); err != nil {
		return err
	} else if err := result.WaitForCompletion(context.Background(), extensionClient.Client); err != nil {
		return err
	}
	return nil
}

func (region *SRegion) resetPublicKey(instanceId string, username, publicKey string) error {
	setting := map[string]string{
		"username": username,
		"ssh_key":  publicKey,
	}
	return region.resetLoginInfo(instanceId, setting)
}

func (region *SRegion) resetPassword(instanceId, username, password string) error {
	setting := map[string]string{
		"username": username,
		"password": password,
	}
	return region.resetLoginInfo(instanceId, setting)
}

func (region *SRegion) DeployVM(instanceId, name, password, publicKey string, deleteKeypair bool, description string) error {
	if instance, err := region.GetInstance(instanceId); err != nil {
		return err
	} else {
		if deleteKeypair {
			return nil
		}
		if len(publicKey) > 0 {
			return region.resetPublicKey(instanceId, instance.Properties.OsProfile.AdminUsername, publicKey)
		} else {
			return region.resetPassword(instanceId, instance.Properties.OsProfile.AdminUsername, password)
		}
		return nil
	}
}

func (self *SInstance) RebuildRoot(imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	return self.host.zone.region.ReplaceSystemDisk(self.ID, imageId, passwd, publicKey, int32(sysSizeGB))
}

func (region *SRegion) ReplaceSystemDisk(instanceId, imageId, passwd, publicKey string, sysSizeGB int32) (string, error) {
	log.Debugf("ReplaceSystemDisk %s image: %s", instanceId, imageId)
	if err := region.DeallocateVM(instanceId); err != nil {
		return "", err
	}
	if instance, err := region.GetInstance(instanceId); err != nil {
		return "", err
	} else {
		image, err := region.GetImage(imageId)
		if err != nil {
			return "", err
		}
		orgOsType := instance.GetOSType()
		destOsType := image.GetOsType()
		if orgOsType != destOsType {
			return "", fmt.Errorf("Cannot replease osType %s => %s", orgOsType, destOsType)
		}
		diskName := fmt.Sprintf("vdisk_%s_%d", instance.Name, time.Now().UnixNano())
		storageType := string(instance.Properties.StorageProfile.OsDisk.ManagedDisk.StorageAccountType)
		//oldDiskId := instance.Properties.StorageProfile.OsDisk.ManagedDisk.ID
		if diskId, err := region.createDisk(storageType, diskName, sysSizeGB, "", imageId); err != nil {
			return "", err
		} else {
			osType := compute.OperatingSystemTypes(image.GetOsType())
			disk, _ := region.GetDisk(diskId)
			sshKeys := []compute.SSHPublicKey{compute.SSHPublicKey{KeyData: &publicKey}}
			params := compute.VirtualMachineUpdate{
				VirtualMachineProperties: &compute.VirtualMachineProperties{
					StorageProfile: &compute.StorageProfile{
						ImageReference: &compute.ImageReference{ID: &image.ID},
						OsDisk: &compute.OSDisk{
							Name: &disk.Name,
							ManagedDisk: &compute.ManagedDiskParameters{
								ID:                 &disk.ID,
								StorageAccountType: compute.StorageAccountTypes(storageType),
							},
							CreateOption: compute.DiskCreateOptionTypesFromImage,
							OsType:       osType,
							DiskSizeGB:   &sysSizeGB,
						},
					},
					OsProfile: &compute.OSProfile{
						AdminUsername: &instance.Properties.OsProfile.AdminUsername,
						AdminPassword: &passwd,
						LinuxConfiguration: &compute.LinuxConfiguration{
							SSH: &compute.SSHConfiguration{
								PublicKeys: &sshKeys,
							},
						},
					},
				},
			}
			if err := region.UpdateInstance(instanceId, params); err != nil {
				region.deleteDisk(diskId)
				return "", err
			}
			// if err := region.deleteDisk(oldDiskId); err != nil {
			// 	return "", err
			// }
			return disk.ID, nil
		}
	}
}

func (region *SRegion) RebuildRoot(instanceId string) error {
	_, resourceGroup, instanceName := pareResourceGroupWithName(instanceId, INSTANCE_RESOURCE)
	computeClient := compute.NewVirtualMachinesClientWithBaseURI(region.client.baseUrl, region.client.subscriptionId)
	computeClient.Authorizer = region.client.authorizer
	if _, err := computeClient.Redeploy(context.Background(), resourceGroup, instanceName); err != nil {
		return err
	}
	return nil
}

func (self *SInstance) UpdateVM(name string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) GetId() string {
	return self.ID
}

func (self *SInstance) GetName() string {
	return self.Name
}

func (self *SInstance) GetGlobalId() string {
	return self.ID
}

func (self *SRegion) GetInstanceStatus(instanceId string) (string, error) {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return "", err
	}
	return instance.GetStatus(), nil
}

func (self *SRegion) DeleteVM(instanceId string) error {
	return self.doDeleteVM(instanceId)
}

func (self *SInstance) DeleteVM() error {
	if err := self.host.zone.region.DeleteVM(self.ID); err != nil {
		return err
	}
	// if osDisk, err := self.getOsDisk(); err != nil {
	// 	return err
	// } else if err := osDisk.Delete(); err != nil {
	// 	if err != cloudprovider.ErrNotFound {
	// 		return err
	// 	}
	// }
	if nics, err := self.getNics(); err != nil {
		return err
	} else {
		for _, nic := range nics {
			if err := nic.Delete(); err != nil {
				if err != cloudprovider.ErrNotFound {
					return err
				}
			}
		}
	}
	return nil
}

func (self *SInstance) getDiskWithStore(diskId string) (*SDisk, error) {
	if disk, err := self.host.zone.region.GetDisk(diskId); err != nil {
		return nil, err
	} else if store, err := self.host.zone.getStorageByType(string(disk.Sku.Name)); err != nil {
		log.Errorf("fail to find storage for disk(%s) : %v", disk.Name, err)
		return nil, err
	} else {
		disk.storage = store
		return disk, nil
	}
}

func (self *SInstance) fetchDisks() error {
	self.idisks = make([]cloudprovider.ICloudDisk, len(self.Properties.StorageProfile.DataDisks)+1)
	if disks, err := self.getDisks(); err != nil {
		return err
	} else {
		for i := 0; i < len(disks); i++ {
			self.idisks[i] = &disks[i]
		}
	}
	return nil
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	if self.idisks == nil {
		if err := self.fetchDisks(); err != nil {
			return nil, err
		}
	}
	return self.idisks, nil
}

func (self *SInstance) GetOSType() string {
	return osprofile.NormalizeOSType(string(self.Properties.StorageProfile.OsDisk.OsType))
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics := make([]cloudprovider.ICloudNic, 0)
	if _nics, err := self.getNics(); err != nil {
		return nil, err
	} else {
		for i := 0; i < len(_nics); i++ {
			_nics[i].instance = self
			nics = append(nics, &_nics[i])
		}
	}
	return nics, nil
}

func (self *SInstance) GetOSName() string {
	return self.Properties.StorageProfile.ImageReference.Offer
}

func (self *SInstance) GetBios() string {
	return "BIOS"
}

func (self *SInstance) GetMachine() string {
	return "pc"
}

func (self *SInstance) GetBootOrder() string {
	return "dcn"
}

func (self *SInstance) GetVga() string {
	return "std"
}

func (self *SInstance) GetVdi() string {
	return "vnc"
}

func (self *SInstance) fetchVMSize() error {
	if vmSize, err := self.host.zone.region.getVMSize(self.Properties.HardwareProfile.VMSize); err != nil {
		return err
	} else {
		self.vmSize = vmSize
	}
	return nil
}

func (self *SInstance) GetVcpuCount() int8 {
	self.fetchVMSize()
	return int8(self.vmSize.NumberOfCores)
}

func (self *SInstance) GetVmemSizeMB() int {
	self.fetchVMSize()
	return int(self.vmSize.MemoryInMB)
}

func (self *SInstance) GetCreateTime() time.Time {
	return self.CreationTime
}

func (self *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	return ret, nil
}

func (self *SRegion) StartVM(instanceId string) error {
	_, resourceGroup, instanceName := pareResourceGroupWithName(instanceId, INSTANCE_RESOURCE)
	computeClient := compute.NewVirtualMachinesClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	computeClient.Authorizer = self.client.authorizer
	if _, err := computeClient.Start(context.Background(), resourceGroup, instanceName); err != nil {
		return err
	}
	return nil
}

func (self *SInstance) StartVM() error {
	if err := self.host.zone.region.StartVM(self.ID); err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, models.VM_RUNNING, 10*time.Second, 300*time.Second)
}

func (self *SInstance) StopVM(isForce bool) error {
	if err := self.host.zone.region.StopVM(self.ID, isForce); err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, models.VM_READY, 10*time.Second, 300*time.Second)
}

func (self *SRegion) StopVM(instanceId string, isForce bool) error {
	return self.doStopVM(instanceId, isForce)
}

func (self *SRegion) DeallocateVM(instanceId string) error {
	_, resourceGroup, instanceName := pareResourceGroupWithName(instanceId, INSTANCE_RESOURCE)
	computeClient := compute.NewVirtualMachinesClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	computeClient.Authorizer = self.client.authorizer
	if result, err := computeClient.PowerOff(context.Background(), resourceGroup, instanceName); err != nil {
		return err
	} else if err := result.WaitForCompletion(context.Background(), computeClient.Client); err != nil {
		return err
	}
	return nil
}

func (self *SRegion) doStopVM(instanceId string, isForce bool) error {
	_, resourceGroup, instanceName := pareResourceGroupWithName(instanceId, INSTANCE_RESOURCE)
	computeClient := compute.NewVirtualMachinesClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	computeClient.Authorizer = self.client.authorizer
	if result, err := computeClient.PowerOff(context.Background(), resourceGroup, instanceName); err != nil {
		return err
	} else if err := result.WaitForCompletion(context.Background(), computeClient.Client); err != nil {
		return err
	}
	return nil
}

func (self *SInstance) SyncSecurityGroup(secgroupId string, name string, rules []secrules.SecurityRule) error {
	nics, err := self.getNics()
	if err != nil {
		return err
	}
	if len(secgroupId) == 0 {
		for _, nic := range nics {
			if err := nic.revokeSecurityGroup(); err != nil {
				return err
			}
		}
	} else if extId, err := self.host.zone.region.syncSecurityGroup(secgroupId, name, rules); err != nil {
		return err
	} else {
		for _, nic := range nics {
			if err := nic.assignSecurityGroup(extId); err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	if nics, err := self.getNics(); err != nil {
		return nil, err
	} else {
		for _, nic := range nics {
			for _, ip := range nic.Properties.IPConfigurations {
				if len(ip.Properties.PublicIPAddress.ID) > 0 {
					return self.host.zone.region.GetEip(ip.Properties.PublicIPAddress.ID)
				}
			}
		}
	}
	return nil, nil
}

func (self *SInstance) GetBillingType() string {
	return models.BILLING_TYPE_POSTPAID
}

func (self *SInstance) GetExpiredAt() time.Time {
	return time.Now()
}
