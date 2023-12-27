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

package remotefile

import (
	"context"

	"yunion.io/x/pkg/util/imagetools"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SInstance struct {
	SResourceBase

	host *SHost

	osInfo *imagetools.ImageInfo

	HostId           string
	Hostname         string
	SecurityGroupIds []string
	VcpuCount        int
	CpuSockets       int
	VmemSizeMb       int
	BootOrder        string
	Vga              string
	Vdi              string
	OsArch           string
	OsType           string
	OsName           string
	Bios             string
	Machine          string
	InstanceType     string
	Bandwidth        int
	Throughput       int
	EipId            string
	Disks            []SDisk

	Nics []SInstanceNic
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	return self.SecurityGroupIds, nil
}

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) StartVM(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) GetIHostId() string {
	return self.HostId
}

func (self *SInstance) GetInternetMaxBandwidthOut() int {
	return self.Bandwidth
}

func (self *SInstance) GetThroughput() int {
	return self.Throughput
}

func (self *SInstance) GetDescription() string {
	return ""
}

func (self *SInstance) GetSerialOutput(port int) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SInstance) ConvertPublicIpToEip() error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) RebuildRoot(ctx context.Context, config *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) CreateDisk(ctx context.Context, opts *cloudprovider.GuestDiskCreateOptions) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SInstance) MigrateVM(hostid string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) LiveMigrateVM(hostid string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) GetError() error {
	return nil
}

func (self *SInstance) CreateInstanceSnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudInstanceSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SInstance) GetInstanceSnapshot(idStr string) (cloudprovider.ICloudInstanceSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SInstance) GetInstanceSnapshots() ([]cloudprovider.ICloudInstanceSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SInstance) ResetToInstanceSnapshot(ctx context.Context, idStr string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) SaveImage(opts *cloudprovider.SaveImageOptions) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SInstance) AllocatePublicIpAddress() (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SInstance) GetCpuSockets() int {
	return self.CpuSockets
}

func (self *SInstance) GetVcpuCount() int {
	return self.VcpuCount
}

func (self *SInstance) GetVmemSizeMB() int {
	return self.VmemSizeMb
}

func (self *SInstance) GetBootOrder() string {
	return self.BootOrder
}

func (self *SInstance) GetVga() string {
	return self.Vga
}

func (self *SInstance) GetVdi() string {
	return self.Vdi
}

func (ins *SInstance) getNormalizedOsInfo() *imagetools.ImageInfo {
	if ins.osInfo == nil {
		osInfo := imagetools.NormalizeImageInfo(ins.OsName, ins.OsArch, ins.OsType, "", "")
		ins.osInfo = &osInfo
	}
	return ins.osInfo
}

func (ins *SInstance) GetOsArch() string {
	return ins.getNormalizedOsInfo().OsArch
}

func (ins *SInstance) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(ins.getNormalizedOsInfo().OsType)
}

func (ins *SInstance) GetFullOsName() string {
	return ins.OsName
}

func (ins *SInstance) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(ins.Bios)
}

func (ins *SInstance) GetOsLang() string {
	return ins.getNormalizedOsInfo().OsLang
}

func (ins *SInstance) GetOsDist() string {
	return ins.getNormalizedOsInfo().OsDistro
}

func (ins *SInstance) GetOsVersion() string {
	return ins.getNormalizedOsInfo().OsVersion
}

func (self *SInstance) GetMachine() string {
	return self.Machine
}

func (self *SInstance) GetInstanceType() string {
	return self.InstanceType
}

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_REMOTEFILE
}

func (self *SInstance) GetHostname() string {
	return self.Hostname
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	ret := []cloudprovider.ICloudDisk{}
	storages, err := self.host.zone.region.client.GetStorages()
	if err != nil {
		return nil, err
	}
	for i := range self.Disks {
		for _, storage := range storages {
			if storage.Id == self.Disks[i].StorageId {
				self.Disks[i].SetStorage(storage)
				ret = append(ret, &self.Disks[i])
			}
		}
	}
	return ret, nil
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	return self.host.zone.region.GetIEipById(self.EipId)
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	ret := []cloudprovider.ICloudNic{}
	for i := range self.Nics {
		ret = append(ret, &self.Nics[i])
	}
	return ret, nil
}

func (self *SInstance) GetPowerStates() string {
	if self.Status == api.VM_RUNNING {
		return api.VM_POWER_STATES_ON
	}
	return api.VM_POWER_STATES_OFF
}
