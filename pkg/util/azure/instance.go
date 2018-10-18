package azure

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/secrules"
)

const (
	DEFAULT_EXTENSION_NAME = "enablevmaccess"
)

type HardwareProfile struct {
	VMSize string `json:"vmSize,omitempty"`
}

type ImageReference struct {
	Publisher string
	Offer     string
	Sku       string
	Version   string
	ID        string
}

type VirtualHardDisk struct {
	Uri string
}

type OSDisk struct {
	OsType       string `json:"osType,omitempty"`
	Caching      string `json:"caching,omitempty"`
	Name         string
	DiskSizeGB   int32                  `json:"diskSizeGB,omitempty"`
	ManagedDisk  *ManagedDiskParameters `json:"managedDisk,omitempty"`
	CreateOption string                 `json:"createOption,omitempty"`
	Vhd          *VirtualHardDisk       `json:"vhd,omitempty"`
}

type ManagedDiskParameters struct {
	StorageAccountType string `json:"storageAccountType,omitempty"`
	ID                 string
}

type DataDisk struct {
	Lun            int32
	Name           string
	DiskName       string
	Vhd            *VirtualHardDisk `json:"vhd,omitempty"`
	Caching        string
	DiskSizeGB     int32
	DiskSize       *int32
	IoType         string
	CreateOption   string
	ManagedDisk    *ManagedDiskParameters
	VhdUri         string
	StorageAccount *StorageAccount
}

type StorageAccount struct {
	Id   string
	Name string
	Type string
}

type StorageProfile struct {
	ImageReference ImageReference `json:"imageReference,omitempty"`
	OsDisk         OSDisk         `json:"osDisk,omitempty"`
	DataDisks      *[]DataDisk
}

type SSHPublicKey struct {
	Path    string `json:"path,omitempty"`
	KeyData string `json:"keyData,omitempty"`
}

type SSHConfiguration struct {
	PublicKeys []SSHPublicKey `json:"publicKeys,omitempty"`
}

type LinuxConfiguration struct {
	DisablePasswordAuthentication bool             `json:"linuxConfiguration,omitempty"`
	SSH                           SSHConfiguration `json:"ssh,omitempty"`
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
	ComputerName       string              `json:"computerName,omitempty"`
	AdminUsername      string              `json:"adminUsername,omitempty"`
	AdminPassword      string              `json:"adminPassword,omitempty"`
	CustomData         string              `json:"customData,omitempty"`
	LinuxConfiguration *LinuxConfiguration `json:"linuxConfiguration,omitempty"`
	Secrets            []VaultSecretGroup
}

type NetworkInterfaceReference struct {
	ID string
}

type NetworkProfile struct {
	NetworkInterfaces []NetworkInterfaceReference `json:"networkInterfaces,omitempty"`
}

type InstanceViewStatus struct {
	Code          string
	Level         string
	DisplayStatus string
	Message       string
	//Time          time.Time
}

type FormattedMessage struct {
	Language string
	Message  string
}

type GuestAgentStatus struct {
	ProtocolVersion   string
	Timestamp         time.Time
	GuestAgentVersion string
	Status            string
	FormattedMessage  FormattedMessage
}

type VirtualMachineInstanceView struct {
	UpdateDomain             int
	FaultDomain              int
	Status                   string
	StatusMessage            string
	PowerState               string
	PrivateIpAddress         string
	PublicIpAddresses        []string
	FullyQualifiedDomainName string
	GuestAgentStatus         GuestAgentStatus

	ComputerName string
	OsName       string
	OsVersion    string
	Statuses     []InstanceViewStatus
}

type DomainName struct {
	Id   string
	Name string
	Type string
}

type DebugProfile struct {
	BootDiagnosticsEnabled   bool
	ConsoleScreenshotBlobUri string
	SerialOutputBlobUri      string
}

type VirtualMachineProperties struct {
	ProvisioningState string
	InstanceView      *VirtualMachineInstanceView `json:"instanceView,omitempty"`
	DomainName        *DomainName
	HardwareProfile   HardwareProfile `json:"hardwareProfile,omitempty"`
	NetworkProfile    NetworkProfile  `json:"networkProfile,omitempty"`
	StorageProfile    StorageProfile  `json:"storageProfile,omitempty"`
	DebugProfile      *DebugProfile   `json:"debugProfile,omitempty"`
	OsProfile         OsProfile       `json:"osProfile,omitempty"`
	VmId              string
}

type SInstance struct {
	host *SHost

	idisks []cloudprovider.ICloudDisk

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
	return &instance, self.client.Get(fmt.Sprintf("%s?$expand=instanceView", instanceId), &instance)
}

func (self *SRegion) GetInstances() ([]SInstance, error) {
	result := []SInstance{}
	instances := []SInstance{}
	err := self.client.ListAll("Microsoft.Compute/virtualMachines", &instances)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(instances); i++ {
		if instances[i].Location == self.Name {
			result = append(result, instances[i])
		}
	}
	return result, nil
}

func (self *SRegion) doDeleteVM(instanceId string) error {
	return self.client.Delete(instanceId)
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
	if nics, err := self.getNics(); err == nil {
		for _, nic := range nics {
			if nic.Properties.NetworkSecurityGroup != nil {
				if len(nic.Properties.NetworkSecurityGroup.ID) > 0 {
					data.Add(jsonutils.NewString(nic.Properties.NetworkSecurityGroup.ID), "secgroupId")
					break
				}
			}
		}
	}

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

func (self *SInstance) getClassicStorageInfoByUri(uri string) (*SClassicStorage, error) {
	_storageName := strings.Split(strings.Replace(uri, "https://", "", -1), ".")
	storageName := ""
	if len(_storageName) > 0 {
		storageName = _storageName[0]
	}
	if len(storageName) == 0 {
		return nil, fmt.Errorf("bad uri %s for search storageaccount", uri)
	}
	storageaccounts, err := self.host.zone.region.GetClassicStorageAccounts()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(storageaccounts); i++ {
		if storageaccounts[i].Name == storageName {
			storage := SClassicStorage{
				zone:     self.host.zone,
				Name:     storageName,
				ID:       storageaccounts[i].ID,
				Type:     storageaccounts[i].Type,
				Location: storageaccounts[i].Type,
			}
			return &storage, nil
		}
	}
	return nil, fmt.Errorf("failed to found classic storageaccount for %s", uri)
}

func (self *SInstance) getDisks() ([]SDisk, []SClassicDisk, error) {
	disks, classicDisk := []SDisk{}, []SClassicDisk{}
	if self.Properties.StorageProfile.OsDisk.Vhd != nil {
		disk := self.Properties.StorageProfile.OsDisk
		storage, err := self.getClassicStorageInfoByUri(disk.Vhd.Uri)
		if err != nil {
			return nil, nil, err
		}
		classicDisk = append(classicDisk, SClassicDisk{
			storage:    storage,
			DiskName:   disk.Name,
			DiskSizeGB: disk.DiskSizeGB,
			Caching:    disk.Caching,
			VhdUri:     disk.Vhd.Uri,
		})
	} else if self.Properties.StorageProfile.OsDisk.ManagedDisk != nil {
		disk, err := self.getDiskWithStore(self.Properties.StorageProfile.OsDisk.ManagedDisk.ID)
		if err != nil {
			log.Errorf("Failed to find instance %s os disk: %s", self.Name, self.Properties.StorageProfile.OsDisk.ManagedDisk.ID)
			return nil, nil, err
		}
		disks = append(disks, *disk)
	}
	for _, _disk := range *self.Properties.StorageProfile.DataDisks {
		if _disk.Vhd != nil {
			storage, err := self.getClassicStorageInfoByUri(_disk.Vhd.Uri)
			if err != nil {
				return nil, nil, err
			}
			classicDisk = append(classicDisk, SClassicDisk{
				storage:    storage,
				DiskName:   _disk.Name,
				DiskSizeGB: _disk.DiskSizeGB,
				Caching:    _disk.Caching,
				VhdUri:     _disk.Vhd.Uri,
			})
		} else if self.Properties.StorageProfile.OsDisk.ManagedDisk != nil {
			disk, err := self.getDiskWithStore(self.Properties.StorageProfile.OsDisk.ManagedDisk.ID)
			if err != nil {
				log.Errorf("Failed to find instance %s os disk: %s", self.Name, self.Properties.StorageProfile.OsDisk.ManagedDisk.ID)
				return nil, nil, err
			}
			disks = append(disks, *disk)
		}
	}
	return disks, classicDisk, nil
}

func (self *SInstance) getNics() ([]SInstanceNic, error) {
	nics := []SInstanceNic{}
	for _, _nic := range self.Properties.NetworkProfile.NetworkInterfaces {
		nic, err := self.host.zone.region.GetNetworkInterfaceDetail(_nic.ID)
		if err != nil {
			log.Errorf("Failed to find instance %s nic: %s", self.Name, _nic.ID)
			return nil, err
		}
		nic.instance = self
		nics = append(nics, *nic)
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
	if self.Properties.InstanceView != nil {
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

func (region *SRegion) AttachDisk(instanceId, diskId string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	disk, err := region.GetDisk(diskId)
	if err != nil {
		return err
	}
	dataDisks := []DataDisk{}
	index := int32(0)
	if instance.Properties.StorageProfile.DataDisks != nil {
		for _, originDisk := range *instance.Properties.StorageProfile.DataDisks {
			originDisk.Lun = index
			dataDisks = append(dataDisks, originDisk)
			index++
		}
	}
	dataDisks = append(dataDisks, DataDisk{
		Lun:          int32(len(*instance.Properties.StorageProfile.DataDisks)),
		CreateOption: disk.Properties.CreationData.CreateOption,
		ManagedDisk: &ManagedDiskParameters{
			ID: disk.ID,
		},
	})
	instance.Properties.ProvisioningState = ""
	_, err = region.client.Update(jsonutils.Marshal(instance))
	return err
}

func (self *SInstance) DetachDisk(diskId string) error {
	if err := self.host.zone.region.DetachDisk(self.ID, diskId); err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, self.GetStatus(), 10*time.Second, 300*time.Second)
}

func (region *SRegion) DetachDisk(instanceId, diskId string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	disk, err := region.GetDisk(diskId)
	if err != nil {
		return err
	}
	dataDisks := []DataDisk{}
	index := int32(0)
	if instance.Properties.StorageProfile.DataDisks != nil {
		for _, origDisk := range *instance.Properties.StorageProfile.DataDisks {
			if origDisk.ManagedDisk.ID != disk.ID {
				origDisk.Lun = index
				dataDisks = append(dataDisks, origDisk)
				index++
			}
		}
	}
	instance.Properties.ProvisioningState = ""
	_, err = region.client.Update(jsonutils.Marshal(instance))
	return err
}

func (self *SInstance) ChangeConfig(instanceId string, ncpu int, vmem int) error {
	for _, vmSize := range self.host.zone.region.getHardwareProfile(ncpu, vmem) {
		self.Properties.HardwareProfile.VMSize = vmSize
		self.Properties.ProvisioningState = ""
		log.Debugf("Try HardwareProfile : %s", vmSize)
		_, err := self.host.zone.region.client.Update(jsonutils.Marshal(self))
		if err == nil {
			return cloudprovider.WaitStatus(self, self.GetStatus(), 10*time.Second, 300*time.Second)
		}
	}
	return fmt.Errorf("Failed to change vm config, specification not supported")
}

func (region *SRegion) ChangeVMConfig(instanceId string, ncpu int, vmem int) error {
	instacen, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	return instacen.ChangeConfig(instanceId, ncpu, vmem)
}

func (self *SInstance) DeployVM(name string, password string, publicKey string, deleteKeypair bool, description string) error {
	return self.host.zone.region.DeployVM(self.ID, name, password, publicKey, deleteKeypair, description)
}

type VirtualMachineExtensionProperties struct {
	Publisher          string
	Type               string
	TypeHandlerVersion string
	ProtectedSettings  interface{}
}

type SVirtualMachineExtension struct {
	Location   string
	Properties VirtualMachineExtensionProperties
	Type       string
}

func (region *SRegion) resetLoginInfo(instanceId string, setting map[string]string) error {
	extension := SVirtualMachineExtension{
		Location: region.Name,
		Type:     "",
		Properties: VirtualMachineExtensionProperties{
			Publisher:          "Microsoft.OSTCExtensions",
			Type:               "VMAccessForLinux",
			TypeHandlerVersion: "1.4",
		},
	}

	url := fmt.Sprintf("%s/extensions/enablevmaccess", instanceId)
	_, err := region.client.jsonRequest("PUT", url, jsonutils.Marshal(extension).String())
	return err
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
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return "", err
	}
	image, err := region.GetImage(imageId)
	if err != nil {
		return "", err
	}
	err = region.stopVM(instanceId)
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
	oldDiskId := instance.Properties.StorageProfile.OsDisk.ManagedDisk.ID
	disk, err := region.CreateDisk(storageType, diskName, sysSizeGB, "", imageId)
	if err != nil {
		return "", err
	}
	instance.Properties.StorageProfile.OsDisk.Name = disk.Name
	instance.Properties.StorageProfile.OsDisk.ManagedDisk.ID = disk.ID
	instance.Properties.StorageProfile.OsDisk.ManagedDisk.StorageAccountType = storageType

	instance.Properties.OsProfile.AdminPassword = passwd
	if len(publicKey) > 0 {
		instance.Properties.OsProfile.LinuxConfiguration = &LinuxConfiguration{
			SSH: SSHConfiguration{
				PublicKeys: []SSHPublicKey{
					SSHPublicKey{
						KeyData: publicKey,
					},
				},
			},
		}
	}
	instance.Properties.ProvisioningState = ""
	_, err = region.client.Update(jsonutils.Marshal(instance))
	if err != nil {
		return "", err
	}
	for i := 0; i < 3; i++ {
		log.Debugf("try delete old disk: %s", oldDiskId)
		if err := region.deleteDisk(oldDiskId); err == nil {
			break
		}
		time.Sleep(time.Second * time.Duration(i*10))
	}
	return disk.ID, nil
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
	return strings.ToLower(self.ID)
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
	disks, classicDisks, err := self.getDisks()
	if err != nil {
		return err
	}
	self.idisks = make([]cloudprovider.ICloudDisk, len(disks)+len(classicDisks))
	for i := 0; i < len(disks); i++ {
		self.idisks[i] = &disks[i]
	}
	for i := 0; i < len(classicDisks); i++ {
		self.idisks[i] = &classicDisks[i]
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
	_nics, err := self.getNics()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(_nics); i++ {
		_nics[i].instance = self
		nics = append(nics, &_nics[i])
	}
	log.Errorf("get nic count %d for %s", len(nics), self.Name)
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
	vmSize, err := self.host.zone.region.getVMSize(self.Properties.HardwareProfile.VMSize)
	if err != nil {
		return err
	}
	self.vmSize = vmSize
	return nil
}

func (self *SInstance) GetVcpuCount() int8 {
	err := self.fetchVMSize()
	if err != nil {
		log.Errorf("fetchVMSize error: %v", err)
		return 1
	}
	return int8(self.vmSize.NumberOfCores)
}

func (self *SInstance) GetVmemSizeMB() int {
	err := self.fetchVMSize()
	if err != nil {
		log.Errorf("fetchVMSize error: %v", err)
		return 2048
	}
	return int(self.vmSize.MemoryInMB)
}

func (self *SInstance) GetCreateTime() time.Time {
	return time.Now()
}

func (self *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	return ret, nil
}

func (self *SRegion) StartVM(instanceId string) error {
	_, err := self.client.PerformAction(instanceId, "start")
	return err
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
	return self.stopVM(instanceId)
}

func (self *SRegion) stopVM(instanceId string) error {
	_, err := self.client.PerformAction(instanceId, "shutdown")
	return err
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
	nics, err := self.getNics()
	if err != nil {
		return nil, err
	}
	for _, nic := range nics {
		for _, ip := range nic.Properties.IPConfigurations {
			if len(ip.Properties.PublicIPAddress.ID) > 0 {
				eip, err := self.host.zone.region.GetEip(ip.Properties.PublicIPAddress.ID)
				if err == nil {
					return eip, nil
				}
				log.Errorf("find eip for instance %s failed: %v", self.Name, err)
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
