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

package cloudpods

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SInstance struct {
	multicloud.SInstanceBase
	multicloud.CloudpodsTags

	host *SHost
	api.ServerDetails
}

func (self *SInstance) GetName() string {
	return self.Name
}

func (self *SInstance) GetId() string {
	return self.Id
}

func (self *SInstance) GetGlobalId() string {
	return self.Id
}

func (self *SInstance) GetStatus() string {
	return self.Status
}

func (self *SInstance) Refresh() error {
	ins, err := self.host.zone.region.GetInstance(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, ins)
}

func (self *SInstance) GetCreatedAt() time.Time {
	return self.CreatedAt
}

func (self *SInstance) GetExpiredAt() time.Time {
	return self.ExpiredAt
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetIHostId() string {
	return self.HostId
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := self.host.zone.region.GetDisks("", self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		disks[i].region = self.host.zone.region
		ret = append(ret, &disks[i])
	}
	return ret, nil
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	if len(self.Eip) > 0 {
		eips, err := self.host.zone.region.GetEips(self.Id)
		if err != nil {
			return nil, err
		}
		for i := range eips {
			eips[i].region = self.host.zone.region
			return &eips[i], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	return nil, nil
}

func (self *SInstance) GetVcpuCount() int {
	return self.VcpuCount
}

func (self *SInstance) GetVmemSizeMB() int {
	return self.VmemSize
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

func (self *SInstance) GetOSType() string {
	return self.OsType
}

func (self *SInstance) GetOSName() string {
	return self.OsName
}

func (self *SInstance) GetBios() string {
	return self.Bios
}

func (self *SInstance) GetMachine() string {
	return self.Machine
}

func (self *SInstance) GetInstanceType() string {
	return self.InstanceType
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	ret := []string{}
	for _, sec := range self.Secgroups {
		ret = append(ret, sec.Id)
	}
	return ret, nil
}

func (self *SInstance) GetProjectId() string {
	return self.TenantId
}

func (self *SInstance) AssignSecurityGroup(id string) error {
	params := map[string]interface{}{
		"secgroup_id": id,
	}
	return self.host.zone.region.perform(&modules.Servers, self.Id, "assign-secgroup", params)
}

func (self *SInstance) SetSecurityGroups(ids []string) error {
	params := map[string]interface{}{
		"secgroup_ids": ids,
	}
	return self.host.zone.region.perform(&modules.Servers, self.Id, "set-secgroup", params)
}

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_CLOUDPODS
}

func (self *SInstance) StartVM(ctx context.Context) error {
	return self.host.zone.region.perform(&modules.Servers, self.Id, "start", nil)
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	params := map[string]interface{}{
		"is_force": opts.IsForce,
	}
	return self.host.zone.region.perform(&modules.Servers, self.Id, "stop", params)
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return self.host.zone.region.cli.delete(&modules.Servers, self.Id)
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) RebuildRoot(ctx context.Context, opts *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	params := map[string]interface{}{
		"image_id": opts.ImageId,
		"password": opts.Password,
	}
	diskId := self.DisksInfo[0].Id
	return diskId, self.host.zone.region.perform(&modules.Servers, self.Id, "rebuild-root", params)
}

func (self *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	params := map[string]interface{}{
		"password": password,
	}
	return self.host.zone.region.perform(&modules.Servers, self.Id, "deploy", params)
}

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	params := map[string]interface{}{
		"disk_id": diskId,
	}
	return self.host.zone.region.perform(&modules.Disks, self.Id, "attach-disk", params)
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	params := map[string]interface{}{
		"disk_id": diskId,
	}
	return self.host.zone.region.perform(&modules.Disks, self.Id, "detach-disk", params)
}

func (self *SInstance) CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) MigrateVM(hostId string) error {
	params := map[string]interface{}{
		"prefer_host": hostId,
	}
	return self.host.zone.region.perform(&modules.Servers, self.Id, "migrate", params)
}

func (self *SInstance) LiveMigrateVM(hostId string) error {
	params := map[string]interface{}{
		"prefer_host": hostId,
	}
	return self.host.zone.region.perform(&modules.Servers, self.Id, "live-migrate", params)
}

func (self *SInstance) GetError() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) CreateInstanceSnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudInstanceSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetInstanceSnapshot(idStr string) (cloudprovider.ICloudInstanceSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetInstanceSnapshots() ([]cloudprovider.ICloudInstanceSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SInstance) ResetToInstanceSnapshot(ctx context.Context, idStr string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) SaveImage(opts *cloudprovider.SaveImageOptions) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SInstance) AllocatePublicIpAddress() (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	servers, err := self.zone.region.GetInstances(self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVM{}
	for i := range servers {
		servers[i].host = self
		ret = append(ret, &servers[i])
	}
	return ret, nil
}

func (self *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	ins, err := self.zone.region.GetInstance(id)
	if err != nil {
		return nil, err
	}
	ins.host = self
	return ins, nil
}

func (self *SRegion) GetInstance(id string) (*SInstance, error) {
	ins := &SInstance{}
	return ins, self.cli.get(&modules.Servers, id, nil, ins)
}

func (self *SRegion) GetInstances(hostId string) ([]SInstance, error) {
	params := map[string]interface{}{}
	if len(hostId) > 0 {
		params["host_id"] = hostId
	}
	ret := []SInstance{}
	return ret, self.list(&modules.Servers, params, &ret)
}

func (self *SRegion) CreateInstance(hostId, hypervisor string, opts *cloudprovider.SManagedVMCreateConfig) (*SInstance, error) {
	input := api.ServerCreateInput{}
	input.Name = opts.Name
	input.Description = opts.Description
	input.InstanceType = opts.InstanceType
	input.VcpuCount = opts.Cpu
	input.VmemSize = opts.MemoryMB
	input.Password = opts.Password
	input.LoginAccount = opts.Account
	input.PublicIpBw = opts.PublicIpBw
	input.PublicIpChargeType = string(opts.PublicIpChargeType)
	input.ProjectId = opts.ProjectId
	input.Metadata = opts.Tags
	input.UserData = opts.UserData
	input.PreferHost = hostId
	input.Hypervisor = hypervisor
	if len(input.UserData) > 0 {
		input.EnableCloudInit = true
	}
	input.Secgroups = opts.ExternalSecgroupIds
	if opts.BillingCycle != nil {
		input.Duration = opts.BillingCycle.String()
	}
	input.Disks = append(input.Disks, &api.DiskConfig{
		Index:    0,
		ImageId:  opts.ExternalImageId,
		DiskType: "sys",
		SizeMb:   opts.SysDisk.SizeGB * 1024,
		Backend:  opts.SysDisk.StorageType,
		Storage:  opts.SysDisk.StorageExternalId,
	})
	for idx, disk := range opts.DataDisks {
		input.Disks = append(input.Disks, &api.DiskConfig{
			Index:    idx + 1,
			DiskType: "data",
			SizeMb:   disk.SizeGB * 1024,
			Backend:  disk.StorageType,
			Storage:  disk.StorageExternalId,
		})
	}
	input.Networks = append(input.Networks, &api.NetworkConfig{
		Index:   0,
		Network: opts.ExternalNetworkId,
		Address: opts.IpAddr,
	})
	ins := &SInstance{}
	return ins, self.create(&modules.Servers, input, ins)
}
