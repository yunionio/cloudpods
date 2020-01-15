// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/osprofile"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/billing"
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
	DataDisks      []DataDisk     `json:"dataDisks,allowempty"`
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

type Statuses struct {
	Code          string
	Level         string
	DisplayStatus string `json:"displayStatus,omitempty"`
	Message       string
	//Time          time.Time
}

type SVMAgent struct {
	VmAgentVersion string     `json:"vmAgentVersion,omitempty"`
	Statuses       []Statuses `json:"statuses,omitempty"`
}

type SExtension struct {
	Name               string
	Type               string
	TypeHandlerVersion string     `json:"typeHandlerVersion,omitempty"`
	Statuses           []Statuses `json:"statuses,omitempty"`
}

type VirtualMachineInstanceView struct {
	Statuses   []Statuses   `json:"statuses,omitempty"`
	VMAgent    SVMAgent     `json:"vmAgent,omitempty"`
	Extensions []SExtension `json:"extensions,omitempty"`
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
	multicloud.SInstanceBase
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

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	secgroupIds := []string{}
	if nics, err := self.getNics(); err == nil {
		for _, nic := range nics {
			if nic.Properties.NetworkSecurityGroup != nil {
				if len(nic.Properties.NetworkSecurityGroup.ID) > 0 {
					secgroupIds = append(secgroupIds, strings.ToLower(nic.Properties.NetworkSecurityGroup.ID))
				}
			}
		}
	}
	return secgroupIds, nil
}

func (self *SInstance) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	tags := jsonutils.NewDict()
	for k, v := range self.Tags {
		tags.Set(k, jsonutils.NewString(v))
	}
	data.Update(tags)
	if osDistribution := self.Properties.StorageProfile.ImageReference.Publisher; len(osDistribution) > 0 {
		data.Add(jsonutils.NewString(osDistribution), "os_distribution")
	}
	if loginAccount := self.Properties.OsProfile.AdminUsername; len(loginAccount) > 0 {
		data.Add(jsonutils.NewString(loginAccount), "login_account")
	}
	if loginKey := self.Properties.OsProfile.AdminPassword; len(loginKey) > 0 {
		data.Add(jsonutils.NewString(loginKey), "login_key")
	}

	data.Add(jsonutils.NewString(self.host.zone.GetGlobalId()), "zone_ext_id")
	priceKey := fmt.Sprintf("%s::%s", self.Properties.HardwareProfile.VMSize, self.host.zone.region.Name)
	data.Add(jsonutils.NewString(priceKey), "price_key")
	return data
}

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_AZURE
}

func (self *SInstance) IsEmulated() bool {
	return false
}

func (self *SInstance) GetInstanceType() string {
	return self.Properties.HardwareProfile.VMSize
}

func (self *SInstance) WaitEnableVMAccessReady() error {
	if self.Properties.InstanceView == nil {
		return fmt.Errorf("instance may not install VMAgent or VMAgent not running")
	}
	if len(self.Properties.InstanceView.VMAgent.VmAgentVersion) > 0 {
		startTime := time.Now()
		timeout := time.Minute * 5
		for {
			status := ""
			for _, vmAgent := range self.Properties.InstanceView.VMAgent.Statuses {
				status = vmAgent.DisplayStatus
				if status == "Ready" {
					break
				}
				log.Debugf("vmAgent %s status: %s waite for ready", self.Properties.InstanceView.VMAgent.VmAgentVersion, vmAgent.DisplayStatus)
				time.Sleep(time.Second * 5)
			}
			if status == "Ready" {
				break
			}
			self.Refresh()
			if time.Now().Sub(startTime) > timeout {
				return fmt.Errorf("timeout for waitting vmAgent ready, current status: %s", status)
			}
		}
		return nil
	}

	for _, extension := range self.Properties.InstanceView.Extensions {
		if extension.Name == "enablevmaccess" {
			displayStatus := ""
			for _, status := range extension.Statuses {
				displayStatus = status.DisplayStatus
				if displayStatus == "Provisioning succeeded" {
					return nil
				}
			}
			return self.host.zone.region.deleteExtension(self.ID, "enablevmaccess")
		}
	}

	return fmt.Errorf("instance may not install VMAgent or VMAgent not running")
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
			return api.VM_UNKNOWN
		}
	}
	for _, statuses := range self.Properties.InstanceView.Statuses {
		if code := strings.Split(statuses.Code, "/"); len(code) == 2 {
			if code[0] == "PowerState" {
				switch code[1] {
				case "stopped", "deallocated":
					return api.VM_READY
				case "running":
					return api.VM_RUNNING
				case "stopping":
					return api.VM_STOPPING
				case "starting":
					return api.VM_STARTING
				case "deleting":
					return api.VM_DELETING
				default:
					log.Errorf("Unknow instance status %s", code[1])
					return api.VM_UNKNOWN
				}
			}
		}
		if statuses.Level == "Error" {
			log.Errorf("Find error code: [%s] message: %s", statuses.Code, statuses.Message)
		}
	}
	return api.VM_UNKNOWN
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	status := self.GetStatus()
	if err := self.host.zone.region.AttachDisk(self.ID, diskId); err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, status, 10*time.Second, 300*time.Second)
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

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	status := self.GetStatus()
	if err := self.host.zone.region.DetachDisk(self.ID, diskId); err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, status, 10*time.Second, 300*time.Second)
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

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	if len(config.InstanceType) > 0 {
		return self.ChangeConfig2(ctx, config.InstanceType)
	}
	status := self.GetStatus()
	for _, vmSize := range self.host.zone.region.getHardwareProfile(config.Cpu, config.MemoryMB) {
		self.Properties.HardwareProfile.VMSize = vmSize
		self.Properties.ProvisioningState = ""
		self.Properties.InstanceView = nil
		log.Debugf("Try HardwareProfile : %s", vmSize)
		err := self.host.zone.region.client.Update(jsonutils.Marshal(self), nil)
		if err == nil {
			return cloudprovider.WaitStatus(self, status, 10*time.Second, 300*time.Second)
		} else {
			log.Debugf("ChangeConfig %s", err)
		}
	}
	return fmt.Errorf("Failed to change vm config, specification not supported")
}

func (self *SInstance) ChangeConfig2(ctx context.Context, instanceType string) error {
	status := self.GetStatus()
	self.Properties.HardwareProfile.VMSize = instanceType
	self.Properties.ProvisioningState = ""
	self.Properties.InstanceView = nil
	log.Debugf("Try HardwareProfile : %s", instanceType)
	err := self.host.zone.region.client.Update(jsonutils.Marshal(self), nil)
	if err == nil {
		return cloudprovider.WaitStatus(self, status, 10*time.Second, 300*time.Second)
	} else {
		log.Errorf("ChangeConfig2 %s", err)
	}

	return fmt.Errorf("Failed to change vm config, specification not supported")
}

func (region *SRegion) ChangeVMConfig2(ctx context.Context, instanceId string, instanceType string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	return instance.ChangeConfig2(ctx, instanceType)
}

func (region *SRegion) ChangeVMConfig(ctx context.Context, instanceId string, ncpu int, vmem int) error {
	instacen, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	return instacen.ChangeConfig(ctx, &cloudprovider.SManagedVMChangeConfig{Cpu: ncpu, MemoryMB: vmem})
}

func (self *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	if len(publicKey) > 0 || len(password) > 0 {
		// 先判断系统是否安装了vmAgent,然后等待扩展准备完成后再重置密码
		err := self.WaitEnableVMAccessReady()
		if err != nil {
			return err
		}
	}
	return self.host.zone.region.DeployVM(ctx, self.ID, name, password, publicKey, deleteKeypair, description)
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

func (region *SRegion) DeployVM(ctx context.Context, instanceId, name, password, publicKey string, deleteKeypair bool, description string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	if deleteKeypair {
		return nil
	}
	if len(publicKey) > 0 {
		return region.resetPublicKey(instanceId, instance.Properties.OsProfile.AdminUsername, publicKey)
	}
	if len(password) > 0 {
		return region.resetPassword(instanceId, instance.Properties.OsProfile.AdminUsername, password)
	}
	return nil
}

func (self *SInstance) RebuildRoot(ctx context.Context, imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	cpu := self.GetVcpuCount()
	memoryMb := self.GetVmemSizeMB()
	self.StopVM(ctx, true)
	return self.host.zone.region.ReplaceSystemDisk(self, cpu, memoryMb, imageId, passwd, publicKey, sysSizeGB)
}

func (region *SRegion) ReplaceSystemDisk(instance *SInstance, cpu int, memoryMb int, imageId, passwd, publicKey string, sysSizeGB int) (string, error) {
	log.Debugf("ReplaceSystemDisk %s image: %s", instance.ID, imageId)
	storageType := instance.Properties.StorageProfile.OsDisk.ManagedDisk.StorageAccountType
	if len(storageType) == 0 {
		_disk, err := region.GetDisk(instance.Properties.StorageProfile.OsDisk.ManagedDisk.ID)
		if err != nil {
			return "", err
		}
		storageType = _disk.Sku.Name
	}
	if len(instance.Properties.NetworkProfile.NetworkInterfaces) == 0 {
		return "", fmt.Errorf("failed to find network for instance: %s", instance.Name)
	}
	nicId := instance.Properties.NetworkProfile.NetworkInterfaces[0].ID
	nic, err := region.GetNetworkInterfaceDetail(nicId)
	if err != nil {
		log.Errorf("failed to find nic %s error: %v", nicId, err)
		return "", err
	}
	if len(nic.Properties.IPConfigurations) == 0 {
		return "", fmt.Errorf("failed to find networkId for nic %s", nicId)
	}
	if instance.Properties.StorageProfile.OsDisk.DiskSizeGB != nil && *instance.Properties.StorageProfile.OsDisk.DiskSizeGB > int32(sysSizeGB) {
		sysSizeGB = int(*instance.Properties.StorageProfile.OsDisk.DiskSizeGB)
	}
	image, err := region.GetImageById(imageId)
	if err != nil {
		return "", err
	}
	if minOsDiskSizeGB := image.GetMinOsDiskSizeGb(); minOsDiskSizeGB > sysSizeGB {
		sysSizeGB = minOsDiskSizeGB
	}

	networkId := nic.Properties.IPConfigurations[0].Properties.Subnet.ID
	osType := instance.Properties.StorageProfile.OsDisk.OsType

	// https://support.microsoft.com/zh-cn/help/4018933/the-default-size-of-windows-server-images-in-azure-is-changed-from-128
	// windows默认系统盘是128G, 若重装系统时，系统盘小于128G，则会出现 {"error":{"code":"ResizeDiskError","message":"Disk size reduction is not supported. Current size is 137438953472 bytes, requested size is 33285996544 bytes.","target":"osDisk.diskSizeGB"}} 错误
	if osType == osprofile.OS_TYPE_WINDOWS && sysSizeGB < 128 {
		sysSizeGB = 128
	}

	newInstance, err := region.CreateInstanceSimple(instance.Name+"-1", imageId, osType, cpu, memoryMb, sysSizeGB, storageType, []int{}, networkId, passwd, publicKey)
	if err != nil {
		return "", err
	}

	newInstance.StopVM(context.Background(), true)
	cloudprovider.WaitStatus(newInstance, api.VM_READY, time.Second*5, time.Minute*5)

	newInstance.deleteVM(context.Background(), true)

	//交换系统盘
	instance.Properties.StorageProfile.OsDisk.ManagedDisk.ID, newInstance.Properties.StorageProfile.OsDisk.ManagedDisk.ID = newInstance.Properties.StorageProfile.OsDisk.ManagedDisk.ID, instance.Properties.StorageProfile.OsDisk.ManagedDisk.ID
	instance.Properties.StorageProfile.OsDisk.DiskSizeGB = newInstance.Properties.StorageProfile.OsDisk.DiskSizeGB
	instance.Properties.StorageProfile.OsDisk.Name = ""
	instance.Properties.ProvisioningState = ""
	instance.Properties.InstanceView = nil
	err = region.client.Update(jsonutils.Marshal(instance), nil)
	if err != nil {
		// 更新失败，需要删除之前交换过的系统盘
		region.DeleteDisk(instance.Properties.StorageProfile.OsDisk.ManagedDisk.ID)
		return "", err
	}
	// 交换成功需要删掉旧的系统盘
	region.DeleteDisk(newInstance.Properties.StorageProfile.OsDisk.ManagedDisk.ID)
	return strings.ToLower(instance.Properties.StorageProfile.OsDisk.ManagedDisk.ID), nil
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
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

func (self *SInstance) deleteVM(ctx context.Context, keepSysDisk bool) error {
	sysDiskId := ""
	if self.Properties.StorageProfile.OsDisk.ManagedDisk != nil {
		sysDiskId = self.Properties.StorageProfile.OsDisk.ManagedDisk.ID
	}
	err := self.host.zone.region.DeleteVM(self.ID)
	if err != nil {
		return err
	}
	if len(sysDiskId) > 0 && !keepSysDisk {
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

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return self.deleteVM(ctx, false)
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

func (self *SInstance) GetVcpuCount() int {
	err := self.fetchVMSize()
	if err != nil {
		log.Errorf("fetchVMSize error: %v", err)
		return 0
	}
	return self.vmSize.NumberOfCores
}

func (self *SInstance) GetVmemSizeMB() int {
	err := self.fetchVMSize()
	if err != nil {
		log.Errorf("fetchVMSize error: %v", err)
		return 0
	}
	return int(self.vmSize.MemoryInMB)
}

func (self *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	return ret, nil
}

func (self *SRegion) StartVM(instanceId string) error {
	_, err := self.client.PerformAction(instanceId, "start", "")
	return err
}

func (self *SInstance) StartVM(ctx context.Context) error {
	if err := self.host.zone.region.StartVM(self.ID); err != nil {
		return err
	}
	self.host.zone.region.client.jsonRequest("PATCH", self.ID, jsonutils.Marshal(self).String())
	return cloudprovider.WaitStatus(self, api.VM_RUNNING, 10*time.Second, 300*time.Second)
}

func (self *SInstance) StopVM(ctx context.Context, isForce bool) error {
	err := self.host.zone.region.StopVM(self.ID, isForce)
	if err != nil {
		return err
	}
	self.host.zone.region.client.jsonRequest("PATCH", self.ID, jsonutils.Marshal(self).String())
	return cloudprovider.WaitStatus(self, api.VM_READY, 10*time.Second, 300*time.Second)
}

func (self *SRegion) StopVM(instanceId string, isForce bool) error {
	_, err := self.client.PerformAction(instanceId, "deallocate", "")
	return err
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

func (self *SInstance) AssignSecurityGroup(secgroupId string) error {
	return self.host.zone.region.SetSecurityGroup(self.ID, secgroupId)
}

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	if len(secgroupIds) == 1 {
		return self.host.zone.region.SetSecurityGroup(self.ID, secgroupIds[0])
	}
	return fmt.Errorf("Unexpect segroup count %d", len(secgroupIds))
}

func (self *SInstance) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SInstance) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SInstance) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) GetProjectId() string {
	return getResourceGroup(self.ID)
}

func (self *SInstance) GetError() error {
	return nil
}
