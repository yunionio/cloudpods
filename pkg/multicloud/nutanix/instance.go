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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type Boot struct {
	UefiBoot bool `json:"uefi_boot"`
}

type VMFeatures struct {
	VGACONSOLE bool `json:"VGA_CONSOLE"`
	AGENTVM    bool `json:"AGENT_VM"`
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

func (self *SInstance) AssignSecurityGroup(id string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetBios() string {
	if self.Boot.UefiBoot {
		return "UEFI"
	}
	return "BIOS"
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
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		storage, err := self.host.zone.GetIStorageById(disks[i].StorageContainerUUID)
		if err != nil {
			log.Errorf("can not found disk %s storage %s", disks[i].DiskAddress, disks[i].StorageContainerUUID)
			continue
		}
		disks[i].storage = storage.(*SStorage)
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

func (self *SInstance) GetOSName() string {
	return ""
}

func (self *SInstance) GetOsType() cloudprovider.TOsType {
	if strings.Contains(strings.ToLower(self.Name), "win") {
		return cloudprovider.OsTypeWindows
	}
	return cloudprovider.OsTypeLinux
}

func (self *SInstance) GetProjectId() string {
	return ""
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	return []string{}, nil
}

func (self *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetVcpuCount() int {
	return self.NumVcpus * self.NumVcpus
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
	return "", cloudprovider.ErrNotImplemented
}

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) StartVM(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	return cloudprovider.ErrNotSupported
}
