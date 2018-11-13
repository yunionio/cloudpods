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
	Publisher string `json:"publisher,omitempty"`
	Offer     string `json:"offer,omitempty"`
	Sku       string `json:"sku,omitempty"`
	Version   string `json:"version,omitempty"`
	ID        string `json:"id,omitempty"`
}

type VirtualHardDisk struct {
	Uri string `json:"uri,omitempty"`
}

type OSDisk struct {
	OsType       string `json:"osType,omitempty"`
	Caching      string `json:"caching,omitempty"`
	Name         string
	DiskSizeGB   *int32                 `json:"diskSizeGB,omitempty"`
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
	Name           string                 `json:"name,omitempty"`
	DiskName       string                 `json:"diskName,omitempty"`
	Vhd            *VirtualHardDisk       `json:"vhd,omitempty"`
	Caching        string                 `json:"caching,omitempty"`
	DiskSizeGB     *int32                 `json:"diskSizeGB,omitempty"`
	IoType         string                 `json:"ioType,omitempty"`
	CreateOption   string                 `json:"createOption,omitempty"`
	ManagedDisk    *ManagedDiskParameters `json:"managedDisk,omitempty"`
	VhdUri         string                 `json:"vhdUri,omitempty"`
	StorageAccount *SubResource           `json:"storageAccount,omitempty"`
}

type StorageProfile struct {
	ImageReference ImageReference `json:"imageReference,omitempty"`
	OsDisk         OSDisk         `json:"osDisk,omitempty"`
	DataDisks      []DataDisk     `json:"dataDisks,omitempty"`
}

type SSHPublicKey struct {
	Path    string `json:"path,omitempty"`
	KeyData string `json:"keyData,omitempty"`
}

type SSHConfiguration struct {
	PublicKeys []SSHPublicKey `json:"publicKeys,omitempty"`
}

type LinuxConfiguration struct {
	DisablePasswordAuthentication bool              `json:"disablePasswordAuthentication,omitempty"`
	SSH                           *SSHConfiguration `json:"ssh,omitempty"`
}

type VaultCertificate struct {
	CertificateURL   string `json:"certificateURL,omitempty"`
	CertificateStore string `json:"certificateStore,omitempty"`
}

type VaultSecretGroup struct {
	SourceVault       SubResource        `json:"sourceVault,omitempty"`
	VaultCertificates []VaultCertificate `json:"vaultCertificates,omitempty"`
}

type OsProfile struct {
	ComputerName       string              `json:"computerName,omitempty"`
	AdminUsername      string              `json:"adminUsername,omitempty"`
	AdminPassword      string              `json:"adminPassword,omitempty"`
	CustomData         string              `json:"customData,omitempty"`
	LinuxConfiguration *LinuxConfiguration `json:"linuxConfiguration,omitempty"`
	Secrets            []VaultSecretGroup  `json:"secrets,omitempty"`
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
	BootDiagnosticsEnabled   *bool  `json:"bootDiagnosticsEnabled,omitempty"`
	ConsoleScreenshotBlobUri string `json:"consoleScreenshotBlobUri,omitempty"`
	SerialOutputBlobUri      string `json:"serialOutputBlobUri,omitempty"`
}

type VirtualMachineProperties struct {
	ProvisioningState string                      `json:"provisioningState,omitempty"`
	InstanceView      *VirtualMachineInstanceView `json:"instanceView,omitempty"`
	DomainName        *DomainName                 `json:"domainName,omitempty"`
	HardwareProfile   HardwareProfile             `json:"hardwareProfile,omitempty"`
	NetworkProfile    NetworkProfile              `json:"networkProfile,omitempty"`
	StorageProfile    StorageProfile              `json:"storageProfile,omitempty"`
	DebugProfile      *DebugProfile               `json:"debugProfile,omitempty"`
	OsProfile         OsProfile                   `json:"osProfile,omitempty"`
	VmId              string                      `json:"vmId,omitempty"`
}

type SInstance struct {
	host *SHost

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
	return &instance, self.client.Get(instanceId, []string{"$expand=instanceView"}, &instance)
}

func (self *SRegion) GetInstanceScaleSets() ([]SInstance, error) {
	instance := []SInstance{}
	return instance, self.client.ListAll("Microsoft.Compute/virtualMachineScaleSets", &instance)
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

func (self *SInstance) getStorageInfoByUri(uri string) (*SStorage, *SClassicStorage, error) {
	_storageName := strings.Split(strings.Replace(uri, "https://", "", -1), ".")
	storageName := ""
	if len(_storageName) > 0 {
		storageName = _storageName[0]
	}
	if len(storageName) == 0 {
		return nil, nil, fmt.Errorf("bad uri %s for search storageaccount", uri)
	}
	storageaccounts, err := self.host.zone.region.GetClassicStorageAccounts()
	if err != nil {
		return nil, nil, err
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
			return nil, &storage, nil
		}
	}
	storageaccounts, err = self.host.zone.region.GetStorageAccounts()
	if err != nil {
		return nil, nil, err
	}
	for i := 0; i < len(storageaccounts); i++ {
		if storageaccounts[i].Name == storageName {
			storage := SStorage{
				zone:        self.host.zone,
				Name:        storageName,
				storageType: storageaccounts[i].Sku.Name,
			}
			return &storage, nil, nil
		}
	}
	return nil, nil, fmt.Errorf("failed to found classic storageaccount for %s", uri)
}

type BasicDisk struct {
	Name         string
	DiskSizeGB   int32
	Caching      string
	CreateOption string
	OsType       string
}

func (self *SInstance) getDisksByUri(uri string, disk *BasicDisk) ([]SDisk, []SClassicDisk, error) {
	storage, classicStorage, err := self.getStorageInfoByUri(uri)
	if err != nil {
		return nil, nil, err
	}
	disks, classicDisks := []SDisk{}, []SClassicDisk{}
	if classicStorage != nil {
		classicDisks = append(classicDisks, SClassicDisk{
			storage:    classicStorage,
			DiskName:   disk.Name,
			DiskSizeGB: disk.DiskSizeGB,
			Caching:    disk.Caching,
			VhdUri:     uri,
		})
	}
	if storage != nil {
		disks = append(disks, SDisk{
			storage: storage,
			ID:      uri,
			Name:    disk.Name,
			Properties: DiskProperties{
				OsType: disk.OsType,
				CreationData: CreationData{
					CreateOption: disk.CreateOption,
				},
				DiskSizeGB: disk.DiskSizeGB,
			},
		})
	}
	return disks, classicDisks, nil
}

func (self *SInstance) getDisks() ([]SDisk, []SClassicDisk, error) {
	instance, err := self.host.zone.region.GetInstance(self.ID)
	if err != nil {
		return nil, nil, err
	}
	disks, classicDisks := []SDisk{}, []SClassicDisk{}
	if instance.Properties.StorageProfile.OsDisk.Vhd != nil {
		disk := self.Properties.StorageProfile.OsDisk
		diskSizeGB := int32(0)
		if disk.DiskSizeGB != nil {
			diskSizeGB = *disk.DiskSizeGB
		}
		basicDisk := &BasicDisk{
			Name:         disk.Name,
			DiskSizeGB:   diskSizeGB,
			Caching:      disk.Caching,
			CreateOption: disk.CreateOption,
			OsType:       disk.OsType,
		}
		_disks, _classicDisks, err := self.getDisksByUri(disk.Vhd.Uri, basicDisk)
		if err != nil {
			return nil, nil, err
		}
		disks = append(disks, _disks...)
		classicDisks = append(classicDisks, _classicDisks...)
	} else if instance.Properties.StorageProfile.OsDisk.ManagedDisk != nil {
		disk, err := self.getDiskWithStore(self.Properties.StorageProfile.OsDisk.ManagedDisk.ID)
		if err != nil {
			log.Errorf("Failed to find instance %s os disk: %s", self.Name, self.Properties.StorageProfile.OsDisk.ManagedDisk.ID)
			return nil, nil, err
		}
		disks = append(disks, *disk)
	}
	for _, _disk := range instance.Properties.StorageProfile.DataDisks {
		diskSizeGB := int32(0)
		if _disk.DiskSizeGB != nil {
			diskSizeGB = *_disk.DiskSizeGB
		} else {
			if _disk.ManagedDisk != nil {
				disk, err := self.host.zone.region.GetDisk(_disk.ManagedDisk.ID)
				if err != nil {
					diskSizeGB = disk.Properties.DiskSizeGB
				}
			}
		}
		if _disk.Vhd != nil {
			basicDisk := &BasicDisk{
				Name:         _disk.Name,
				DiskSizeGB:   diskSizeGB,
				Caching:      _disk.Caching,
				CreateOption: _disk.CreateOption,
			}
			_disks, _classicDisks, err := self.getDisksByUri(_disk.Vhd.Uri, basicDisk)
			if err != nil {
				return nil, nil, err
			}
			disks = append(disks, _disks...)
			classicDisks = append(classicDisks, _classicDisks...)
		} else if _disk.ManagedDisk != nil {
			disk, err := self.getDiskWithStore(_disk.ManagedDisk.ID)
			if err != nil {
				log.Errorf("Failed to find instance %s os disk: %s", self.Name, _disk.ManagedDisk.ID)
				return nil, nil, err
			}
			disks = append(disks, *disk)
		}
	}

	return disks, classicDisks, nil
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
	instance, err := self.host.zone.region.GetInstance(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, instance)
}

func (self *SInstance) GetStatus() string {
	if self.Properties.InstanceView == nil {
		err := self.Refresh()
		if err != nil {
			log.Errorf("failed to get status for instance %s", self.Name)
			return models.VM_UNKNOWN
		}
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

func (region *SRegion) AttachDisk(instanceId, diskId string) error {
	disk, err := region.GetDisk(diskId)
	if err != nil {
		return err
	}
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	dataDisks := instance.Properties.StorageProfile.DataDisks
	lun, find := -1, false
	for i := 0; i < len(dataDisks); i++ {
		if dataDisks[i].Lun != int32(i) {
			lun, find = i, true
			break
		}
	}

	if !find || lun == -1 {
		lun = len(dataDisks)
	}

	dataDisks = append(dataDisks, DataDisk{
		Lun:          int32(lun),
		CreateOption: "Attach",
		ManagedDisk: &ManagedDiskParameters{
			ID: disk.ID,
		},
	})
	instance.Properties.StorageProfile.DataDisks = dataDisks
	instance.Properties.ProvisioningState = ""
	instance.Properties.InstanceView = nil
	return region.client.Update(jsonutils.Marshal(instance), nil)
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
	for _, origDisk := range instance.Properties.StorageProfile.DataDisks {
		if origDisk.ManagedDisk.ID != disk.ID {
			dataDisks = append(dataDisks, origDisk)
		}
	}
	instance.Properties.StorageProfile.DataDisks = dataDisks
	instance.Properties.ProvisioningState = ""
	instance.Properties.InstanceView = nil
	return region.client.Update(jsonutils.Marshal(instance), nil)
}

func (self *SInstance) ChangeConfig(instanceId string, ncpu int, vmem int) error {
	for _, vmSize := range self.host.zone.region.getHardwareProfile(ncpu, vmem) {
		self.Properties.HardwareProfile.VMSize = vmSize
		self.Properties.ProvisioningState = ""
		self.Properties.InstanceView = nil
		log.Debugf("Try HardwareProfile : %s", vmSize)
		err := self.host.zone.region.client.Update(jsonutils.Marshal(self), nil)
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
	Publisher          string      `json:"publisher,omitempty"`
	Type               string      `json:"type,omitempty"`
	TypeHandlerVersion string      `json:"typeHandlerVersion,omitempty"`
	ProtectedSettings  interface{} `json:"protectedSettings,omitempty"`
	Settings           interface{} `json:"settings,omitempty"`
}

type SVirtualMachineExtension struct {
	Location   string                            `json:"location,omitempty"`
	Properties VirtualMachineExtensionProperties `json:"properties,omitempty"`
}

func (region *SRegion) execOnLinux(instanceId string, command string) error {
	extension := SVirtualMachineExtension{
		Location: region.Name,
		Properties: VirtualMachineExtensionProperties{
			Publisher:          "Microsoft.Azure.Extensions",
			Type:               "CustomScript",
			TypeHandlerVersion: "2.0",
			Settings:           map[string]string{"commandToExecute": command},
		},
	}
	url := fmt.Sprintf("%s/extensions/CustomScript", instanceId)
	_, err := region.client.jsonRequest("PUT", url, jsonutils.Marshal(extension).String())
	return err
}

func (region *SRegion) resetOvsEnv(instanceId string) error {
	ovsEnv, err := region.getOvsEnv(instanceId)
	if err != nil {
		return err
	}
	err = region.execOnLinux(instanceId, fmt.Sprintf(`echo '%s' > /var/lib/waagent/ovf-env.xml`, ovsEnv))
	if err != nil {
		return err
	}
	return region.execOnLinux(instanceId, "systemctl restart wagent")
}

func (region *SRegion) deleteExtension(instanceId, extensionName string) error {
	return region.client.Delete(fmt.Sprintf("%s/extensions/%s", instanceId, extensionName))
}
func (region *SRegion) resetLoginInfo(instanceId string, setting map[string]string) error {
	extension := SVirtualMachineExtension{
		Location: region.Name,
		Properties: VirtualMachineExtensionProperties{
			Publisher:          "Microsoft.OSTCExtensions",
			Type:               "VMAccessForLinux",
			TypeHandlerVersion: "1.4",
			ProtectedSettings:  setting,
		},
	}
	url := fmt.Sprintf("%s/extensions/enablevmaccess", instanceId)
	_, err := region.client.jsonRequest("PUT", url, jsonutils.Marshal(extension).String())
	if err != nil {
		err = region.deleteExtension(instanceId, "enablevmaccess")
		if err != nil {
			return err
		}
		err = region.resetOvsEnv(instanceId)
		if err != nil {
			return err
		}
		url := fmt.Sprintf("%s/extensions/enablevmaccess", instanceId)
		_, err = region.client.jsonRequest("PUT", url, jsonutils.Marshal(extension).String())
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
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	if deleteKeypair {
		return nil
	}
	if len(publicKey) > 0 {
		err = region.resetPublicKey(instanceId, instance.Properties.OsProfile.AdminUsername, publicKey)
	}
	return region.resetPassword(instanceId, instance.Properties.OsProfile.AdminUsername, password)
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
	err = region.StopVM(instanceId, true)
	if err != nil {
		return "", err
	}
	orgOsType := instance.GetOSType()
	destOsType := image.GetOsType()
	if orgOsType != destOsType {
		return "", fmt.Errorf("Cannot replease osType %s => %s", orgOsType, destOsType)
	}
	diskName := fmt.Sprintf("vdisk_%s_%d", instance.Name, time.Now().UnixNano())
	storageType := instance.Properties.StorageProfile.OsDisk.ManagedDisk.StorageAccountType
	if len(storageType) == 0 {
		_disk, err := region.GetDisk(instance.Properties.StorageProfile.OsDisk.ManagedDisk.ID)
		if err != nil {
			return "", err
		}
		storageType = _disk.Sku.Name
	}
	oldDiskId := instance.Properties.StorageProfile.OsDisk.ManagedDisk.ID
	disk, err := region.CreateDisk(storageType, diskName, sysSizeGB, "", imageId)
	if err != nil {
		log.Errorf("Create system disk error: %v", err)
		return "", err
	}
	instance.Properties.StorageProfile.OsDisk.Name = disk.Name
	instance.Properties.StorageProfile.OsDisk.ManagedDisk.ID = disk.ID
	instance.Properties.StorageProfile.OsDisk.ManagedDisk.StorageAccountType = storageType

	instance.Properties.OsProfile.AdminPassword = passwd
	if len(publicKey) > 0 {
		instance.Properties.OsProfile.LinuxConfiguration = &LinuxConfiguration{
			SSH: &SSHConfiguration{
				PublicKeys: []SSHPublicKey{
					SSHPublicKey{
						KeyData: publicKey,
					},
				},
			},
		}
	}
	for i := 0; i < len(instance.Properties.StorageProfile.DataDisks); i++ {
		//避免因size更新不及时导致更换系统盘失败
		instance.Properties.StorageProfile.DataDisks[i].DiskSizeGB = nil
	}

	instance.Properties.ProvisioningState = ""
	instance.Properties.InstanceView = nil
	err = region.client.Update(jsonutils.Marshal(instance), nil)
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
	// Azure 数据刷新不及时，需要稍作等待
	for i := 0; i < 3; i++ {
		instance, err := region.GetInstance(instanceId)
		if err != nil {
			return "", err
		}
		if instance.Properties.StorageProfile.OsDisk.ManagedDisk.ID == disk.ID {
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

func (self *SRegion) DeleteVM(instanceId string) error {
	return self.doDeleteVM(instanceId)
}

func (self *SInstance) DeleteVM() error {
	sysDiskId := ""
	if self.Properties.StorageProfile.OsDisk.ManagedDisk != nil {
		sysDiskId = self.Properties.StorageProfile.OsDisk.ManagedDisk.ID
	}
	err := self.host.zone.region.DeleteVM(self.ID)
	if err != nil {
		return err
	}
	if len(sysDiskId) > 0 {
		err := self.host.zone.region.deleteDisk(sysDiskId)
		if err != nil {
			return err
		}
	}

	nics, err := self.getNics()
	if err != nil {
		return err
	}
	for _, nic := range nics {
		if err := nic.Delete(); err != nil {
			if err != cloudprovider.ErrNotFound {
				return err
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

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, classicDisks, err := self.getDisks()
	if err != nil {
		return nil, err
	}
	idisks := make([]cloudprovider.ICloudDisk, len(disks)+len(classicDisks))
	for i := 0; i < len(disks); i++ {
		idisks[i] = &disks[i]
	}
	for i := 0; i < len(classicDisks); i++ {
		idisks[len(disks)+i] = &classicDisks[i]
	}
	return idisks, nil
}

func (self *SInstance) GetOSType() string {
	return osprofile.NormalizeOSType(string(self.Properties.StorageProfile.OsDisk.OsType))
}

func (self *SRegion) getOvsEnv(instanceId string) (string, error) {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return "", err
	}
	kms := map[string]string{
		"AzureGermanCloud":       "kms.core.cloudapi.de",
		"AzureChinaCloud":        "kms.core.chinacloudapi.cn",
		"AzureUSGovernmentCloud": "kms.core.usgovcloudapi.net",
		"AzurePublicCloud":       "kms.core.windows.net",
	}
	kmsServer := "kms.core.chinacloudapi.cn"
	if _kmsServer, ok := kms[self.client.envName]; ok {
		kmsServer = _kmsServer
	}
	return fmt.Sprintf(`
	<ns0:Environment xmlns:ns0="http://schemas.dmtf.org/ovf/environment/1" xmlns:ns1="http://schemas.microsoft.com/windowsazure" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
		<ns1:ProvisioningSection>
		<ns1:Version>1.0</ns1:Version>
		<ns1:LinuxProvisioningConfigurationSet>
			<ns1:ConfigurationSetType>LinuxProvisioningConfiguration</ns1:ConfigurationSetType>
			<ns1:UserName>%s</ns1:UserName>
			<ns1:DisableSshPasswordAuthentication>false</ns1:DisableSshPasswordAuthentication>
			<ns1:HostName>%s</ns1:HostName>
			<ns1:UserPassword>REDACTED</ns1:UserPassword>
		</ns1:LinuxProvisioningConfigurationSet>
		</ns1:ProvisioningSection>
			<ns1:PlatformSettingsSection>
				<ns1:Version>1.0</ns1:Version>
			<ns1:PlatformSettings>
				<ns1:KmsServerHostname>%s</ns1:KmsServerHostname>
				<ns1:ProvisionGuestAgent>true</ns1:ProvisionGuestAgent>
				<ns1:GuestAgentPackageName xsi:nil="true" />
				<ns1:RetainWindowsPEPassInUnattend>true</ns1:RetainWindowsPEPassInUnattend>
				<ns1:RetainOfflineServicingPassInUnattend>true</ns1:RetainOfflineServicingPassInUnattend>
				<ns1:PreprovisionedVm>false</ns1:PreprovisionedVm>
				<ns1:EnableTrustedImageIdentifier>false</ns1:EnableTrustedImageIdentifier>
			</ns1:PlatformSettings>
		</ns1:PlatformSettingsSection>
	</ns0:Environment>`, instance.Properties.OsProfile.AdminUsername, instance.Properties.OsProfile.ComputerName, kmsServer), nil
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
		return 0
	}
	return int8(self.vmSize.NumberOfCores)
}

func (self *SInstance) GetVmemSizeMB() int {
	err := self.fetchVMSize()
	if err != nil {
		log.Errorf("fetchVMSize error: %v", err)
		return 0
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
	_, err := self.client.PerformAction(instanceId, "start", "")
	return err
}

func (self *SInstance) StartVM() error {
	if err := self.host.zone.region.StartVM(self.ID); err != nil {
		return err
	}
	self.host.zone.region.client.jsonRequest("PATCH", self.ID, "")
	return cloudprovider.WaitStatus(self, models.VM_RUNNING, 10*time.Second, 300*time.Second)
}

func (self *SInstance) StopVM(isForce bool) error {
	err := self.host.zone.region.StopVM(self.ID, isForce)
	if err != nil {
		return err
	}
	self.host.zone.region.client.jsonRequest("PATCH", self.ID, "")
	return cloudprovider.WaitStatus(self, models.VM_READY, 10*time.Second, 300*time.Second)
}

func (self *SRegion) StopVM(instanceId string, isForce bool) error {
	_, err := self.client.PerformAction(instanceId, "deallocate", "")
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
		return nil
	}
	extId, err := self.host.zone.region.syncSecurityGroup(secgroupId, name, rules)
	if err != nil {
		return err
	}
	for _, nic := range nics {
		if err := nic.assignSecurityGroup(extId); err != nil {
			return err
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
			if ip.Properties.PublicIPAddress != nil {
				if len(ip.Properties.PublicIPAddress.ID) > 0 {
					eip, err := self.host.zone.region.GetEip(ip.Properties.PublicIPAddress.ID)
					if err == nil {
						return eip, nil
					}
					log.Errorf("find eip for instance %s failed: %v", self.Name, err)
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
