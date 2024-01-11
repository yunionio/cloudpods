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
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/version"

	"yunion.io/x/cloudmux/pkg/apis"
	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
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

type ManagedDiskParameters struct {
	StorageAccountType string `json:"storageAccountType,omitempty"`
	ID                 string
}

type StorageProfile struct {
	ImageReference ImageReference `json:"imageReference,omitempty"`
	OsDisk         SOsDisk        `json:"osDisk,omitempty"`
	DataDisks      []SDataDisk    `json:"dataDisks,allowempty"`
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
	TimeCreated       time.Time                   `json:"timeCreated,omitempty"`
}

type SExtensionResourceProperties struct {
	AutoUpgradeMinorVersion bool
	ProvisioningState       string
	Publisher               string
	Type                    string
	TypeHandlerVersion      string
}

type SExtensionResource struct {
	Id       string
	Name     string
	Type     string
	Location string

	Properties SExtensionResourceProperties
}

type SInstance struct {
	multicloud.SInstanceBase
	AzureTags
	host *SHost

	Properties VirtualMachineProperties
	ID         string
	Name       string
	Type       string
	Location   string
	vmSize     *SVMSize

	Resources []SExtensionResource
}

func (self *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	instance := SInstance{}
	params := url.Values{}
	params.Set("$expand", "instanceView")
	return &instance, self.get(instanceId, params, &instance)
}

func (self *SRegion) GetInstanceScaleSets() ([]SInstance, error) {
	instance := []SInstance{}
	return instance, self.client.list("Microsoft.Compute/virtualMachineScaleSets", url.Values{}, &instance)
}

func (self *SRegion) GetInstances() ([]SInstance, error) {
	result := []SInstance{}
	resource := fmt.Sprintf("Microsoft.Compute/locations/%s/virtualMachines", self.Name)
	err := self.client.list(resource, url.Values{}, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (self *SRegion) doDeleteVM(instanceId string) error {
	return self.del(instanceId)
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	secgroupIds := []string{}
	if nics, err := self.getNics(); err == nil {
		for _, nic := range nics {
			if nic.Properties.NetworkSecurityGroup != nil && len(nic.Properties.NetworkSecurityGroup.ID) > 0 {
				secgroupIds = append(secgroupIds, strings.ToLower(nic.Properties.NetworkSecurityGroup.ID))
			}
		}
	}
	return secgroupIds, nil
}

func (self *SInstance) GetSysTags() map[string]string {
	data := map[string]string{}
	if osDistribution := self.Properties.StorageProfile.ImageReference.Publisher; len(osDistribution) > 0 {
		data["os_distribution"] = osDistribution
	}
	if loginAccount := self.Properties.OsProfile.AdminUsername; len(loginAccount) > 0 {
		data["login_account"] = loginAccount
	}
	if loginKey := self.Properties.OsProfile.AdminPassword; len(loginKey) > 0 {
		data["login_key"] = loginKey
	}
	for _, res := range self.Resources {
		if strings.HasSuffix(strings.ToLower(res.Id), "databricksbootstrap") {
			data[apis.IS_SYSTEM] = "true"
			break
		}
	}
	return data
}

func (self *SInstance) GetTags() (map[string]string, error) {
	return self.Tags, nil
}

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_AZURE
}

func (self *SInstance) GetInstanceType() string {
	return self.Properties.HardwareProfile.VMSize
}

func (self *SInstance) WaitVMAgentReady() error {
	status := ""
	err := cloudprovider.Wait(time.Second*5, time.Minute*5, func() (bool, error) {
		if self.Properties.InstanceView == nil {
			self.Refresh()
			return false, nil
		}

		for _, vmAgent := range self.Properties.InstanceView.VMAgent.Statuses {
			status = vmAgent.DisplayStatus
			if status == "Ready" {
				break
			}
			log.Debugf("vmAgent %s status: %s waite for ready", self.Properties.InstanceView.VMAgent.VmAgentVersion, vmAgent.DisplayStatus)
		}
		if status == "Ready" {
			return true, nil
		}
		return false, self.Refresh()
	})
	if err != nil {
		return errors.Wrapf(err, "waitting vmAgent ready, current status: %s", status)
	}
	return nil

}

func (self *SInstance) WaitEnableVMAccessReady() error {
	if self.Properties.InstanceView == nil {
		return fmt.Errorf("instance may not install VMAgent or VMAgent not running")
	}
	if len(self.Properties.InstanceView.VMAgent.VmAgentVersion) > 0 {
		return self.WaitVMAgentReady()
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

func (self *SInstance) getNics() ([]SInstanceNic, error) {
	nics := []SInstanceNic{}
	for _, _nic := range self.Properties.NetworkProfile.NetworkInterfaces {
		nic, err := self.host.zone.region.GetNetworkInterface(_nic.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "GetNetworkInterface(%s)", _nic.ID)
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
	err = jsonutils.Update(self, instance)
	if err != nil {
		return err
	}
	self.Tags = instance.Tags
	return nil
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

func (ins *SInstance) GetPowerStates() string {
	status := ins.GetStatus()
	switch status {
	case api.VM_READY:
		return api.VM_POWER_STATES_OFF
	case api.VM_UNKNOWN:
		return api.VM_POWER_STATES_OFF
	default:
		return api.VM_POWER_STATES_ON
	}
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return self.host.zone.region.AttachDisk(self.ID, diskId)
}

func (region *SRegion) AttachDisk(instanceId, diskId string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return errors.Wrapf(err, "GetInstance(%s)", instanceId)
	}
	lunMaps := map[int32]bool{}
	dataDisks := jsonutils.NewArray()
	for _, disk := range instance.Properties.StorageProfile.DataDisks {
		if disk.ManagedDisk != nil && strings.ToLower(disk.ManagedDisk.ID) == strings.ToLower(diskId) {
			return nil
		}
		lunMaps[disk.Lun] = true
		dataDisks.Add(jsonutils.Marshal(disk))
	}
	lun := func() int32 {
		idx := int32(0)
		for {
			if _, ok := lunMaps[idx]; !ok {
				return idx
			}
			idx++
		}
	}()
	dataDisks.Add(jsonutils.Marshal(map[string]interface{}{
		"Lun":          lun,
		"CreateOption": "Attach",
		"ManagedDisk": map[string]string{
			"Id": diskId,
		},
	}))
	params := jsonutils.NewDict()
	params.Add(dataDisks, "Properties", "StorageProfile", "DataDisks")
	params.Add(jsonutils.Marshal(instance.Properties.StorageProfile.OsDisk), "Properties", "StorageProfile", "OsDisk")
	_, err = region.patch(instanceId, params)
	return err
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return self.host.zone.region.DetachDisk(self.ID, diskId)
}

func (region *SRegion) DetachDisk(instanceId, diskId string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return errors.Wrapf(err, "GetInstance(%s)", instanceId)
	}
	diskMaps := map[string]bool{}
	dataDisks := jsonutils.NewArray()
	for _, disk := range instance.Properties.StorageProfile.DataDisks {
		if disk.ManagedDisk != nil {
			diskMaps[strings.ToLower(disk.ManagedDisk.ID)] = true
			if strings.ToLower(disk.ManagedDisk.ID) == strings.ToLower(diskId) {
				continue
			}
		}
		dataDisks.Add(jsonutils.Marshal(disk))
	}
	if _, ok := diskMaps[strings.ToLower(diskId)]; !ok {
		log.Warningf("not find disk %s with instance %s", diskId, instance.Name)
		return nil
	}
	params := jsonutils.NewDict()
	params.Add(dataDisks, "Properties", "StorageProfile", "DataDisks")
	params.Add(jsonutils.Marshal(instance.Properties.StorageProfile.OsDisk), "Properties", "StorageProfile", "OsDisk")
	_, err = region.patch(instanceId, params)
	return err
}

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	if len(config.InstanceType) > 0 {
		return self.host.zone.region.ChangeConfig(self.ID, config.InstanceType)
	}
	var err error
	for _, vmSize := range self.host.zone.region.getHardwareProfile(config.Cpu, config.MemoryMB) {
		log.Debugf("Try HardwareProfile : %s", vmSize)
		err = self.host.zone.region.ChangeConfig(self.ID, vmSize)
		if err == nil {
			return nil
		}
	}
	if err != nil {
		return errors.Wrap(err, "ChangeConfig")
	}
	return fmt.Errorf("Failed to change vm config, specification not supported")
}

func (self *SRegion) ChangeConfig(instanceId, instanceType string) error {
	params := map[string]interface{}{
		"Properties": map[string]interface{}{
			"HardwareProfile": map[string]string{
				"vmSize": instanceType,
			},
		},
	}
	log.Debugf("Try HardwareProfile : %s", instanceType)
	_, err := self.patch(instanceId, jsonutils.Marshal(params))
	return err
}

func (region *SRegion) ChangeVMConfig(ctx context.Context, instanceId string, ncpu int, vmem int) error {
	instacen, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	return instacen.ChangeConfig(ctx, &cloudprovider.SManagedVMChangeConfig{Cpu: ncpu, MemoryMB: vmem})
}

func (self *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	if len(opts.PublicKey) > 0 || len(opts.Password) > 0 {
		// 先判断系统是否安装了vmAgent,然后等待扩展准备完成后再重置密码
		err := self.WaitEnableVMAccessReady()
		if err != nil {
			return err
		}
	}
	return self.host.zone.region.DeployVM(ctx, self.ID, string(self.GetOsType()), opts)
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
	resource := fmt.Sprintf("%s/extensions/CustomScript", instanceId)
	_, err := region.put(resource, jsonutils.Marshal(extension))
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
	return region.del(fmt.Sprintf("%s/extensions/%s", instanceId, extensionName))
}
func (region *SRegion) resetLoginInfo(osType, instanceId string, setting map[string]interface{}) error {
	// https://github.com/Azure/azure-linux-extensions/blob/master/VMAccess/README.md
	handlerVersion := "1.5"
	properties := map[string]interface{}{
		"Publisher":          "Microsoft.OSTCExtensions",
		"Type":               "VMAccessForLinux",
		"TypeHandlerVersion": handlerVersion,
		"Settings":           map[string]string{},
		"protectedSettings":  setting,

		"autoUpgradeMinorVersion": true,
	}
	if osType == osprofile.OS_TYPE_WINDOWS {
		// https://github.com/Azure/azure-cli/blob/dev/src/azure-cli/azure/cli/command_modules/vm/custom.py
		handlerVersion = "2.4"
		properties["TypeHandlerVersion"] = handlerVersion
		properties["Publisher"] = "Microsoft.Compute"
		properties["Type"] = "VMAccessAgent"
	}
	params := map[string]interface{}{
		"Location":   region.Name,
		"Properties": properties,
	}
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return errors.Wrapf(err, "GetInstance(%s)", instanceId)
	}
	for _, extension := range instance.Resources {
		if extension.Name == "enablevmaccess" {
			if version.GT(extension.Properties.TypeHandlerVersion, handlerVersion) {
				properties["TypeHandlerVersion"] = extension.Properties.TypeHandlerVersion
				break
			}
		}
	}
	resource := fmt.Sprintf("%s/extensions/enablevmaccess", instanceId)
	_, err = region.put(resource, jsonutils.Marshal(params))
	if err != nil {
		switch osType {
		case osprofile.OS_TYPE_WINDOWS:
			return err
		default:
			err = region.deleteExtension(instanceId, "enablevmaccess")
			if err != nil {
				return err
			}
			err = region.resetOvsEnv(instanceId)
			if err != nil {
				return err
			}
			resource := fmt.Sprintf("%s/extensions/enablevmaccess", instanceId)
			_, err = region.put(resource, jsonutils.Marshal(params))
			return err
		}
	}
	err = cloudprovider.Wait(time.Second*5, time.Minute*5, func() (bool, error) {
		instance, err := region.GetInstance(instanceId)
		if err != nil {
			return false, errors.Wrapf(err, "GetInstance(%s)", instanceId)
		}
		for _, extension := range instance.Resources {
			if extension.Name == "enablevmaccess" {
				if extension.Properties.ProvisioningState == "Succeeded" {
					return true, nil
				}
				log.Debugf("enablevmaccess status %s expect Succeeded", extension.Properties.ProvisioningState)
				if extension.Properties.ProvisioningState == "Failed" {
					if instance.Properties.InstanceView != nil {
						for _, info := range instance.Properties.InstanceView.Extensions {
							if info.Name == "enablevmaccess" && len(info.Statuses) > 0 {
								return false, fmt.Errorf("details: %s", jsonutils.Marshal(info.Statuses))
							}
						}
					}
					return false, fmt.Errorf("reset passwod failed")
				}
			}
		}
		return false, nil
	})
	if err != nil {
		return errors.Wrapf(err, "wait for enablevmaccess error: %v", err)
	}
	return nil
}

func (region *SRegion) resetPublicKey(osType, instanceId string, username, publicKey string) error {
	setting := map[string]interface{}{
		"username": username,
		"ssh_key":  publicKey,
	}
	return region.resetLoginInfo(osType, instanceId, setting)
}

func (region *SRegion) resetPassword(osType, instanceId, username, password string) error {
	setting := map[string]interface{}{
		"username": username,
		"password": password,
	}
	return region.resetLoginInfo(osType, instanceId, setting)
}

func (region *SRegion) DeployVM(ctx context.Context, instanceId, osType string, opts *cloudprovider.SInstanceDeployOptions) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	if opts.DeleteKeypair {
		return nil
	}
	if len(opts.PublicKey) > 0 {
		return region.resetPublicKey(osType, instanceId, instance.Properties.OsProfile.AdminUsername, opts.PublicKey)
	}
	if len(opts.Password) > 0 {
		return region.resetPassword(osType, instanceId, instance.Properties.OsProfile.AdminUsername, opts.Password)
	}
	return nil
}

func (self *SInstance) RebuildRoot(ctx context.Context, desc *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	cpu := self.GetVcpuCount()
	memoryMb := self.GetVmemSizeMB()
	opts := &cloudprovider.ServerStopOptions{
		IsForce: true,
	}
	self.StopVM(ctx, opts)
	return self.host.zone.region.ReplaceSystemDisk(self, cpu, memoryMb, desc.ImageId, desc.Password, desc.PublicKey, desc.SysSizeGB)
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
	image, err := region.GetImageById(imageId)
	if err != nil {
		return "", errors.Wrapf(err, "GetImageById(%s)", imageId)
	}
	if minOsDiskSizeGB := image.GetMinOsDiskSizeGb(); minOsDiskSizeGB > sysSizeGB {
		sysSizeGB = minOsDiskSizeGB
	}
	osType := instance.Properties.StorageProfile.OsDisk.OsType
	// https://support.microsoft.com/zh-cn/help/4018933/the-default-size-of-windows-server-images-in-azure-is-changed-from-128
	// windows默认系统盘是128G, 若重装系统时，系统盘小于128G，则会出现 {"error":{"code":"ResizeDiskError","message":"Disk size reduction is not supported. Current size is 137438953472 bytes, requested size is 33285996544 bytes.","target":"osDisk.diskSizeGB"}} 错误
	if osType == osprofile.OS_TYPE_WINDOWS && sysSizeGB < 128 {
		sysSizeGB = 128
	}
	nicId := instance.Properties.NetworkProfile.NetworkInterfaces[0].ID
	nic, err := region.GetNetworkInterface(nicId)
	if err != nil {
		return "", errors.Wrapf(err, "GetNetworkInterface(%s)", nicId)
	}
	if len(nic.Properties.IPConfigurations) == 0 {
		return "", fmt.Errorf("failed to find networkId for nic %s", nicId)
	}
	networkId := nic.Properties.IPConfigurations[0].Properties.Subnet.ID

	nic, err = region.CreateNetworkInterface("", fmt.Sprintf("%s-temp-ifconfig", instance.Name), "", networkId, "")
	if err != nil {
		return "", errors.Wrapf(err, "CreateNetworkInterface")
	}

	newInstance, err := region.CreateInstanceSimple(instance.Name+"-rebuild", imageId, osType, cpu, memoryMb, sysSizeGB, storageType, []int{}, nic.ID, passwd, publicKey)
	if err != nil {
		return "", errors.Wrapf(err, "CreateInstanceSimple")
	}
	newInstance.host = instance.host
	cloudprovider.Wait(time.Second*10, time.Minute*5, func() (bool, error) {
		err = newInstance.WaitVMAgentReady()
		if err != nil {
			log.Warningf("WaitVMAgentReady for %s error: %v", newInstance.Name, err)
			return false, nil
		}
		return true, nil
	})

	opts := &cloudprovider.ServerStopOptions{
		IsForce: true,
	}
	newInstance.StopVM(context.Background(), opts)

	pruneDiskId := instance.Properties.StorageProfile.OsDisk.ManagedDisk.ID
	newDiskId := newInstance.Properties.StorageProfile.OsDisk.ManagedDisk.ID

	err = newInstance.deleteVM(context.TODO(), true)
	if err != nil {
		log.Warningf("delete vm %s error: %v", newInstance.ID, err)
	}

	defer func() {
		if len(pruneDiskId) > 0 {
			err := cloudprovider.Wait(time.Second*3, time.Minute, func() (bool, error) {
				err = region.DeleteDisk(pruneDiskId)
				if err != nil {
					log.Errorf("delete prune disk %s error: %v", pruneDiskId, err)
					return false, nil
				}
				return true, nil
			})
			if err != nil {
				log.Errorf("timeout for delete prune disk %s", pruneDiskId)
			}
		}
	}()

	//交换系统盘
	params := map[string]interface{}{
		"Id":       instance.ID,
		"Location": instance.Location,
		"Properties": map[string]interface{}{
			"StorageProfile": map[string]interface{}{
				"OsDisk": map[string]interface{}{
					"createOption": "FromImage",
					"ManagedDisk": map[string]interface{}{
						"Id":                 newDiskId,
						"storageAccountType": nil,
					},
					"osType":     osType,
					"DiskSizeGB": nil,
				},
			},
		},
	}
	err = region.update(jsonutils.Marshal(params), nil)
	if err != nil {
		// 更新失败，需要删除新建的系统盘
		pruneDiskId = newDiskId
		return "", errors.Wrapf(err, "region.update")
	}
	return strings.ToLower(newDiskId), nil
}

func (self *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) GetId() string {
	return self.ID
}

func (self *SInstance) GetName() string {
	return self.Name
}

func (self *SInstance) GetHostname() string {
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
		err := self.host.zone.region.DeleteDisk(sysDiskId)
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

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks := []cloudprovider.ICloudDisk{}
	self.Properties.StorageProfile.OsDisk.region = self.host.zone.region
	disks = append(disks, &self.Properties.StorageProfile.OsDisk)
	for i := range self.Properties.StorageProfile.DataDisks {
		self.Properties.StorageProfile.DataDisks[i].region = self.host.zone.region
		disks = append(disks, &self.Properties.StorageProfile.DataDisks[i])
	}
	return disks, nil
}

func (self *SInstance) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(osprofile.NormalizeOSType(string(self.Properties.StorageProfile.OsDisk.OsType)))
}

func (self *SInstance) GetFullOsName() string {
	return self.Properties.StorageProfile.ImageReference.Offer
}

func (self *SInstance) GetBios() cloudprovider.TBiosType {
	return "BIOS"
}

func (i *SInstance) GetOsDist() string {
	return ""
}

func (i *SInstance) GetOsVersion() string {
	return ""
}

func (i *SInstance) GetOsLang() string {
	return ""
}

func (i *SInstance) GetOsArch() string {
	return apis.OS_ARCH_X86_64
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
	if self.vmSize == nil {
		vmSize, err := self.host.zone.region.getVMSize(self.Properties.HardwareProfile.VMSize)
		if err != nil {
			return err
		}
		self.vmSize = vmSize
	}
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

func (self *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) StartVM(instanceId string) error {
	_, err := self.perform(instanceId, "start", nil)
	return err
}

func (self *SInstance) StartVM(ctx context.Context) error {
	if err := self.host.zone.region.StartVM(self.ID); err != nil {
		return err
	}
	self.host.zone.region.patch(self.ID, jsonutils.Marshal(self))
	return cloudprovider.WaitStatus(self, api.VM_RUNNING, 10*time.Second, 300*time.Second)
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	err := self.host.zone.region.StopVM(self.ID, opts.IsForce)
	if err != nil {
		return err
	}
	self.host.zone.region.patch(self.ID, jsonutils.Marshal(self))
	return cloudprovider.WaitStatus(self, api.VM_READY, 10*time.Second, 300*time.Second)
}

func (self *SRegion) StopVM(instanceId string, isForce bool) error {
	_, err := self.perform(instanceId, "deallocate", nil)
	return err
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	nics, err := self.getNics()
	if err != nil {
		return nil, err
	}
	for _, nic := range nics {
		for _, ip := range nic.Properties.IPConfigurations {
			if ip.Properties.PublicIPAddress != nil && len(ip.Properties.PublicIPAddress.ID) > 0 {
				eip, err := self.host.zone.region.GetEip(ip.Properties.PublicIPAddress.ID)
				if err != nil {
					return nil, errors.Wrapf(err, "GetEip(%s)", ip.Properties.PublicIPAddress.ID)
				}
				if len(eip.Properties.IPAddress) > 0 {
					return eip, nil
				}
			}
		}
	}
	return nil, nil
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
	return self.Properties.TimeCreated
}

func (self *SInstance) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) GetProjectId() string {
	return getResourceGroup(self.ID)
}

func (self *SInstance) GetError() error {
	if self.Properties.InstanceView != nil {
		for _, status := range self.Properties.InstanceView.Statuses {
			if status.Code == "ProvisioningState/failed/AllocationFailed" {
				return errors.Errorf("%s %s", status.Code, status.Message)
			}
		}
	}
	return nil
}

func (self *SRegion) SaveImage(osType, diskId string, opts *cloudprovider.SaveImageOptions) (*SImage, error) {
	params := map[string]interface{}{
		"Location": self.Name,
		"Name":     opts.Name,
		"Properties": map[string]interface{}{
			"storageProfile": map[string]interface{}{
				"osDisk": map[string]interface{}{
					"osType": osType,
					"managedDisk": map[string]string{
						"id": diskId,
					},
					"osState": "Generalized",
				},
			},
		},
		"Type": "Microsoft.Compute/images",
	}
	image := &SImage{storageCache: self.getStoragecache()}
	err := self.create("", jsonutils.Marshal(params), image)
	if err != nil {
		return nil, errors.Wrapf(err, "create image")
	}
	return image, nil
}

func (self *SInstance) SaveImage(opts *cloudprovider.SaveImageOptions) (cloudprovider.ICloudImage, error) {
	if self.Properties.StorageProfile.OsDisk.ManagedDisk == nil {
		return nil, fmt.Errorf("invalid os disk for save image")
	}
	image, err := self.host.zone.region.SaveImage(string(self.GetOsType()), self.Properties.StorageProfile.OsDisk.ManagedDisk.ID, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "SaveImage")
	}
	return image, nil
}

func (self *SInstance) SetTags(tags map[string]string, replace bool) error {
	if !replace {
		for k, v := range self.Tags {
			if _, ok := tags[k]; !ok {
				tags[k] = v
			}
		}
	}
	_, err := self.host.zone.region.client.SetTags(self.ID, tags)
	if err != nil {
		return errors.Wrapf(err, "self.host.zone.region.client.SetTags(%s,%s)", self.ID, jsonutils.Marshal(tags).String())
	}
	return nil
}
