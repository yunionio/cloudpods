package azure

import (
	"context"
	"fmt"
	"regexp"
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

func PareResourceGroupWithName(s string, resourceType string) (string, string) {
	valid := regexp.MustCompile("resourceGroups/(.+)/providers/.+/(.+)$")
	if resourceGroups := valid.FindStringSubmatch(s); len(resourceGroups) == 3 {
		return resourceGroups[1], resourceGroups[2]
	}
	log.Errorf("PareResourceGroupWithName[%s] error", s)
	return DefaultResourceGroups[resourceType], s
}

func (self *SRegion) GetInstance(resourceGroup string, VMName string) (*SInstance, error) {
	instance := SInstance{}
	computeClient := compute.NewVirtualMachinesClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	computeClient.Authorizer = self.client.authorizer
	if len(VMName) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if _instance, err := computeClient.Get(context.Background(), resourceGroup, VMName, "instanceView"); err != nil {
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
			instance := SInstance{}
			if *_instance.Location == self.Name {
				if err := jsonutils.Update(&instance, _instance); err != nil {
					return instances, err
				}
				instances = append(instances, instance)
			}
		}
	}
	return instances, nil
}

func (self *SRegion) doDeleteVM(instanceId string) error {
	resourceGroup, instanceName := PareResourceGroupWithName(instanceId, INSTANCE_RESOURCE)
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

func (self *SInstance) getDisks() ([]SDisk, error) {
	disks := make([]SDisk, 0)
	resourceGroup, diskName := PareResourceGroupWithName(self.Properties.StorageProfile.OsDisk.ManagedDisk.ID, DISK_RESOURCE)
	if osdisk, err := self.getDiskWithStore(resourceGroup, diskName); err != nil {
		log.Errorf("Failed to find instance %s os disk: %s", self.Name, diskName)

	} else {
		disks = append(disks, *osdisk)
	}
	for _, _disk := range self.Properties.StorageProfile.DataDisks {
		resourceGroup, diskName := PareResourceGroupWithName(_disk.ManagedDisk.ID, DISK_RESOURCE)
		if disk, err := self.getDiskWithStore(resourceGroup, diskName); err != nil {
			log.Errorf("Failed to find instance %s data disk: %s", self.Name, diskName)
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
		resourceGroup, nicName := PareResourceGroupWithName(_nic.ID, NIC_RESOURCE)
		if nic, err := self.host.zone.region.getNetworkInterface(resourceGroup, nicName); err != nil {
			log.Errorf("Failed to find instance %s nic: %s", self.Name, nicName)
			return nil, err
		} else {
			nic.instance = self
			nics = append(nics, *nic)
		}
	}
	return nics, nil
}

func (self *SInstance) Refresh() error {
	resourceGroup, instanceName := PareResourceGroupWithName(self.ID, INSTANCE_RESOURCE)
	if instance, err := self.host.zone.region.GetInstance(resourceGroup, instanceName); err != nil {
		return err
	} else if err := jsonutils.Update(self, instance); err != nil {
		return err
	}
	return nil
}

func (self *SInstance) GetStatus() string {
	if len(self.Properties.InstanceView.Statuses) == 0 {
		self.Refresh()
	}
	for _, statuses := range self.Properties.InstanceView.Statuses {
		if code := strings.Split(statuses.Code, "/"); len(code) == 2 {
			if code[0] == "PowerState" {
				switch code[1] {
				case "stopped", "deallocated":
					return models.VM_READY
				case "running":
					return models.VM_RUNNING
				case "stopping":
					return models.VM_START_STOP
				default:
					return models.VM_UNKNOWN
				}

			}
		}
	}
	return models.VM_UNKNOWN
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) AttachDisk(diskId string) error {
	return nil
}

func (self *SInstance) DetachDisk(diskId string) error {
	return nil
}

func (self *SInstance) ChangeConfig(instanceId string, ncpu int, vmem int) error {
	return nil
}

func (self *SInstance) DeployVM(name string, password string, publicKey string, resetPassword bool, deleteKeypair bool, description string) error {
	return nil
}

func (self *SInstance) RebuildRoot(imageId string) error {
	return nil
}

func (self *SInstance) UpdateVM(name string) error {
	return nil
}

func (self *SInstance) GetId() string {
	return self.ID
}

func (self *SInstance) GetName() string {
	return self.Name
}

func (self *SInstance) GetGlobalId() string {
	resourceGroup, instanceName := PareResourceGroupWithName(self.ID, INSTANCE_RESOURCE)
	return fmt.Sprintf("resourceGroups/%s/providers/server/%s", resourceGroup, instanceName)
}

func (self *SRegion) GetInstanceStatus(instanceId string) (string, error) {
	resourceGroup, instanceName := PareResourceGroupWithName(instanceId, INSTANCE_RESOURCE)
	instance, err := self.GetInstance(resourceGroup, instanceName)
	if err != nil {
		return "", err
	}
	return instance.GetStatus(), nil
}

func (self *SRegion) DeleteVM(instanceId string) error {
	if status, err := self.GetInstanceStatus(instanceId); err != nil {
		return err
	} else if status == models.VM_RUNNING {
		if err := self.StopVM(instanceId, true); err != nil {
			return err
		}
	} else if status != models.VM_READY {
		log.Debugf("instance %s status: %s", instanceId, status)
		return cloudprovider.ErrInvalidStatus
	}
	return self.doDeleteVM(instanceId)
}

func (self *SInstance) DeleteVM() error {
	log.Debugf("delete: %s %s", self.ID, self.Name)
	if err := self.host.zone.region.DeleteVM(self.ID); err != nil {
		return err
	}
	if disks, err := self.getDisks(); err != nil {
		return err
	} else {
		for _, disk := range disks {
			if err := disk.Delete(); err != nil {
				return err
			}
		}
	}
	if nics, err := self.getNics(); err != nil {
		return err
	} else {
		for _, nic := range nics {
			if err := nic.Delete(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *SInstance) getDiskWithStore(resourceGroup string, diskName string) (*SDisk, error) {
	if disk, err := self.host.zone.region.GetDisk(resourceGroup, diskName); err != nil {
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
	for _, _nic := range self.Properties.NetworkProfile.NetworkInterfaces {
		resourceGroup, nicName := PareResourceGroupWithName(_nic.ID, NIC_RESOURCE)
		if nic, err := self.host.zone.region.getNetworkInterface(resourceGroup, nicName); err != nil {
			return nics, err
		} else {
			nic.instance = self
			nics = append(nics, nic)
		}
	}
	for _, nic := range nics {
		log.Debugf("find nic %s for instance %s", nic.GetIP(), self.Name)
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
	if self.vmSize == nil {
		if err := self.fetchVMSize(); err != nil {
			log.Errorf("fail to fetch vmSize: %v", err)
			return 0
		}
	}
	return int8(self.vmSize.NumberOfCores)
}

func (self *SInstance) GetVmemSizeMB() int {
	if self.vmSize == nil {
		if err := self.fetchVMSize(); err != nil {
			log.Errorf("fail to fetch vmSize: %v", err)
			return 0
		}
	}
	return int(self.vmSize.MemoryInMB)
}

func (self *SInstance) GetCreateTime() time.Time {
	return self.CreationTime
}

func (self *SInstance) GetEIP() cloudprovider.ICloudEIP {
	return nil
	//return &self.EipAddress
}

func (self *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	return ret, nil
}

func (self *SRegion) StartVM(instanceId string) error {
	resourceGroup, instanceName := PareResourceGroupWithName(instanceId, INSTANCE_RESOURCE)
	computeClient := compute.NewVirtualMachinesClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	computeClient.Authorizer = self.client.authorizer
	if result, err := computeClient.Start(context.Background(), resourceGroup, instanceName); err != nil {
		return err
	} else if err := result.WaitForCompletion(context.Background(), computeClient.Client); err != nil {
		return err
	}
	return nil
}

func (self *SInstance) StartVM() error {
	if err := self.host.zone.region.StartVM(self.ID); err != nil {
		return err
	}
	return nil
}

func (self *SInstance) StopVM(isForce bool) error {
	if err := self.host.zone.region.StopVM(self.ID, isForce); err != nil {
		return err
	}
	return nil
}

func (self *SRegion) StopVM(instanceId string, isForce bool) error {
	return self.doStopVM(instanceId, isForce)
}

func (self *SRegion) doStopVM(instanceId string, isForce bool) error {
	resourceGroup, instanceName := PareResourceGroupWithName(instanceId, INSTANCE_RESOURCE)
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
