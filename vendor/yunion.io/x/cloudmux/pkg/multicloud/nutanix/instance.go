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

package nutanix

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/apis"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type Boot struct {
	UefiBoot bool `json:"uefi_boot"`
}

type VMFeatures struct {
	VGACONSOLE bool `json:"VGA_CONSOLE"`
	AGENTVM    bool `json:"AGENT_VM"`
}

type DiskAddress struct {
	DeviceBus    string `json:"device_bus"`
	DeviceIndex  int    `json:"device_index"`
	DiskLabel    string `json:"disk_label"`
	NdfsFilepath string `json:"ndfs_filepath"`
	VmdiskUUID   string `json:"vmdisk_uuid"`
	DeviceUUID   string `json:"device_uuid"`
}

type VMDiskInfo struct {
	IsCdrom              bool              `json:"is_cdrom"`
	IsEmpty              bool              `json:"is_empty"`
	FlashModeEnabled     bool              `json:"flash_mode_enabled"`
	IsScsiPassthrough    bool              `json:"is_scsi_passthrough"`
	IsHotRemoveEnabled   bool              `json:"is_hot_remove_enabled"`
	IsThinProvisioned    bool              `json:"is_thin_provisioned"`
	Shared               bool              `json:"shared"`
	SourceDiskAddress    SourceDiskAddress `json:"source_disk_address,omitempty"`
	StorageContainerUUID string            `json:"storage_container_uuid,omitempty"`
	Size                 int64             `json:"size,omitempty"`
	DataSourceURL        string            `json:"data_source_url"`
	DiskAddress          DiskAddress       `json:"disk_address,omitempty"`
}

type SourceDiskAddress struct {
	VmdiskUUID string `json:"vmdisk_uuid"`
}

type VMNics struct {
	MacAddress  string `json:"mac_address"`
	NetworkUUID string `json:"network_uuid"`
	NicUUID     string `json:"nic_uuid"`
	Model       string `json:"model"`
	VlanMode    string `json:"vlan_mode"`
	IsConnected bool   `json:"is_connected"`
}

type SInstance struct {
	multicloud.STagBase
	multicloud.SInstanceBase

	host *SHost

	AllowLiveMigrate   bool       `json:"allow_live_migrate"`
	GpusAssigned       bool       `json:"gpus_assigned"`
	Boot               Boot       `json:"boot"`
	HaPriority         int        `json:"ha_priority"`
	HostUUID           string     `json:"host_uuid"`
	MemoryMb           int        `json:"memory_mb"`
	Name               string     `json:"name"`
	NumCoresPerVcpu    int        `json:"num_cores_per_vcpu"`
	NumVcpus           int        `json:"num_vcpus"`
	PowerState         string     `json:"power_state"`
	Timezone           string     `json:"timezone"`
	UUID               string     `json:"uuid"`
	VMFeatures         VMFeatures `json:"vm_features"`
	VMLogicalTimestamp int        `json:"vm_logical_timestamp"`
	MachineType        string     `json:"machine_type"`

	VMDiskInfo []VMDiskInfo `json:"vm_disk_info"`
	VMNics     []VMNics     `json:"vm_nics"`
}

func (self *SRegion) GetInstances() ([]SInstance, error) {
	vms := []SInstance{}
	params := url.Values{}
	params.Set("include_vm_disk_config", "true")
	params.Set("include_vm_nic_config", "true")
	return vms, self.listAll("vms", params, &vms)
}

func (self *SRegion) GetInstance(id string) (*SInstance, error) {
	vm := &SInstance{}
	params := url.Values{}
	params.Set("include_vm_disk_config", "true")
	params.Set("include_vm_nic_config", "true")
	return vm, self.get("vms", id, params, vm)
}

func (self *SInstance) GetName() string {
	return self.Name
}

func (self *SInstance) GetId() string {
	return self.UUID
}

func (self *SInstance) GetGlobalId() string {
	return self.UUID
}

func (self *SInstance) Refresh() error {
	ins, err := self.host.zone.region.GetInstance(self.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, ins)
}

func (self *SInstance) CreateDisk(ctx context.Context, opts *cloudprovider.GuestDiskCreateOptions) (string, error) {
	driver := opts.Driver
	if !utils.IsInStringArray(driver, []string{"ide", "scsi", "pci", "sata"}) {
		driver = "scsi"
	}
	idx, ids := -1, []int{}
	for _, disk := range self.VMDiskInfo {
		if disk.DiskAddress.DeviceBus == driver {
			ids = append(ids, disk.DiskAddress.DeviceIndex)
		}
	}
	sort.Ints(ids)
	for _, id := range ids {
		if id == idx+1 {
			idx = id
		}
	}
	params := map[string]interface{}{
		"vm_disks": []map[string]interface{}{
			{
				"is_cdrom": false,
				"disk_address": map[string]interface{}{
					"device_bus":   driver,
					"device_index": idx + 1,
				},
				"vm_disk_create": map[string]interface{}{
					"storage_container_uuid": opts.StorageId,
					"size":                   opts.SizeMb * 1024 * 1024,
				},
			},
		},
	}
	ret := struct {
		TaskUUID string
	}{}
	res := fmt.Sprintf("vms/%s/disks/attach", self.UUID)
	err := self.host.zone.region.post(res, jsonutils.Marshal(params), &ret)
	if err != nil {
		return "", err
	}
	return self.host.zone.region.cli.wait(ret.TaskUUID)
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) ChangeConfig(ctx context.Context, opts *cloudprovider.SManagedVMChangeConfig) error {
	params := map[string]interface{}{
		"memory_mb":          opts.MemoryMB,
		"num_cores_per_vcpu": self.NumCoresPerVcpu,
		"num_vcpus":          opts.Cpu / self.NumCoresPerVcpu,
	}
	return self.host.zone.region.update("vms", self.UUID, jsonutils.Marshal(params), nil)
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return self.host.zone.region.DeleteVM(self.UUID)
}

func (self *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetBios() cloudprovider.TBiosType {
	if self.Boot.UefiBoot {
		return cloudprovider.UEFI
	}
	return cloudprovider.BIOS
}

func (self *SInstance) GetBootOrder() string {
	return "dcn"
}

func (self *SInstance) GetError() error {
	return nil
}

func (self *SInstance) GetHostname() string {
	return self.Name
}

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_NUTANIX
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := self.host.zone.region.GetDisks("", self.GetGlobalId())
	if err != nil {
		return nil, errors.Wrapf(err, "GetInstanceDisks")
	}
	cdroms := []string{}
	for _, disk := range self.VMDiskInfo {
		if disk.IsCdrom && len(disk.DiskAddress.VmdiskUUID) > 0 {
			cdroms = append(cdroms, disk.DiskAddress.VmdiskUUID)
		}
	}
	isSys := true
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		if utils.IsInStringArray(disks[i].UUID, cdroms) { // skip cdrom disk
			continue
		}
		storage, err := self.host.zone.GetIStorageById(disks[i].StorageContainerUUID)
		if err != nil {
			log.Errorf("can not found disk %s storage %s", disks[i].DiskAddress, disks[i].StorageContainerUUID)
			continue
		}
		disks[i].isSys = isSys
		disks[i].storage = storage.(*SStorage)
		isSys = false
		ret = append(ret, &disks[i])
	}
	return ret, nil
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics, err := self.host.zone.region.GetInstanceNics(self.GetGlobalId())
	if err != nil {
		return nil, errors.Wrapf(err, "GetInstanceNics")
	}
	ret := []cloudprovider.ICloudNic{}
	for i := range nics {
		nics[i].ins = self
		ret = append(ret, &nics[i])
	}
	return ret, nil
}

func (self *SInstance) GetInstanceType() string {
	return fmt.Sprintf("ecs.g1.c%dm%d", self.GetVcpuCount(), self.GetVmemSizeMB()/1024)
}

func (self *SInstance) GetMachine() string {
	return self.MachineType
}

// "UNKNOWN", "OFF", "POWERING_ON", "ON", "SHUTTING_DOWN", "POWERING_OFF", "PAUSING", "PAUSED", "SUSPENDING", "SUSPENDED", "RESUMING", "RESETTING", "MIGRATING"
func (self *SInstance) GetStatus() string {
	switch strings.ToUpper(self.PowerState) {
	case "OFF":
		return api.VM_READY
	case "POWERING_ON":
		return api.VM_START_START
	case "ON":
		return api.VM_RUNNING
	case "SHUTTING_DOWN":
		return api.VM_START_STOP
	case "POWERING_OFF":
		return api.VM_START_STOP
	case "PAUSING", "PAUSED":
		return api.VM_READY
	case "SUSPENDING", "SUSPENDED":
		return api.VM_SUSPEND
	case "RESUMING":
		return api.VM_RESUMING
	case "RESETTING":
		return api.VM_RUNNING
	case "MIGRATING":
		return api.VM_MIGRATING
	}
	return api.VM_UNKNOWN
}

func (self *SInstance) GetFullOsName() string {
	return ""
}

func (self *SInstance) GetOsType() cloudprovider.TOsType {
	if strings.Contains(strings.ToLower(self.Name), "win") {
		return cloudprovider.OsTypeWindows
	}
	return cloudprovider.OsTypeLinux
}

func (ins *SInstance) GetOsDist() string {
	return ""
}

func (ins *SInstance) GetOsVersion() string {
	return ""
}

func (ins *SInstance) GetOsLang() string {
	return ""
}

func (ins *SInstance) GetOsArch() string {
	return apis.OS_ARCH_X86_64
}

func (self *SInstance) GetProjectId() string {
	return ""
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	return []string{}, nil
}

func (self *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetVcpuCount() int {
	return self.NumCoresPerVcpu * self.NumVcpus
}

func (self *SInstance) GetVmemSizeMB() int {
	return self.MemoryMb
}

func (self *SInstance) GetVga() string {
	return "std"
}

func (self *SInstance) GetVdi() string {
	return "vnc"
}

func (self *SInstance) RebuildRoot(ctx context.Context, desc *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) StartVM(ctx context.Context) error {
	return self.host.zone.region.SetInstancePowerState(self.UUID, "on")
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	act := "acpi_shutdown"
	if opts.IsForce {
		act = "off"
	}
	return self.host.zone.region.SetInstancePowerState(self.UUID, act)
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) SetInstancePowerState(id string, state string) error {
	res := fmt.Sprintf("vms/%s/set_power_state", id)
	ret := struct {
		TaskUUID string
	}{}
	err := self.post(res, jsonutils.Marshal(map[string]string{"transition": state}), &ret)
	if err != nil {
		return err
	}
	_, err = self.cli.wait(ret.TaskUUID)
	return err
}

func (self *SRegion) DeleteVM(id string) error {
	return self.delete("vms", id)
}
