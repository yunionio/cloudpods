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
	"fmt"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/mcclient/modules/logger"
	"yunion.io/x/onecloud/pkg/mcclient/modules/webconsole"
)

type SInstance struct {
	multicloud.SInstanceBase
	CloudpodsTags

	host *SHost
	api.ServerDetails
}

func (vm *SInstance) GetName() string {
	return vm.Name
}

func (vm *SInstance) GetHostname() string {
	return vm.Hostname
}

func (vm *SInstance) GetId() string {
	return vm.Id
}

func (vm *SInstance) GetGlobalId() string {
	return vm.Id
}

func (vm *SInstance) GetStatus() string {
	return vm.Status
}

func (vm *SInstance) Refresh() error {
	ins, err := vm.host.zone.region.GetInstance(vm.Id)
	if err != nil {
		return err
	}
	vm.DisksInfo = nil
	vm.Nics = nil
	vm.Secgroups = nil
	vm.SubIPs = nil
	vm.IsolatedDevices = nil
	vm.Cdrom = nil
	vm.Floppy = nil
	return jsonutils.Update(vm, ins)
}

func (vm *SInstance) GetCreatedAt() time.Time {
	return vm.CreatedAt
}

func (vm *SInstance) GetExpiredAt() time.Time {
	return vm.ExpiredAt
}

func (vm *SInstance) GetIHost() cloudprovider.ICloudHost {
	return vm.host
}

func (vm *SInstance) GetIHostId() string {
	return vm.HostId
}

func (vm *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := vm.host.zone.region.GetDisks("", vm.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		disks[i].region = vm.host.zone.region
		ret = append(ret, &disks[i])
	}
	return ret, nil
}

func (vm *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	if len(vm.Eip) > 0 {
		eips, err := vm.host.zone.region.GetEips(vm.Id)
		if err != nil {
			return nil, err
		}
		for i := range eips {
			eips[i].region = vm.host.zone.region
			return &eips[i], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	return nil, nil
}

func (vm *SInstance) GetVcpuCount() int {
	return vm.VcpuCount
}

func (vm *SInstance) GetVmemSizeMB() int {
	return vm.VmemSize
}

func (vm *SInstance) GetBootOrder() string {
	return vm.BootOrder
}

func (vm *SInstance) GetVga() string {
	return vm.Vga
}

func (vm *SInstance) GetVdi() string {
	return vm.Vdi
}

func (vm *SInstance) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(vm.OsType)
}

func (vm *SInstance) GetFullOsName() string {
	return vm.OsName
}

func (vm *SInstance) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(vm.Bios)
}

func (vm *SInstance) GetOsDist() string {
	val, ok := vm.Metadata["os_distribution"]
	if ok {
		return val
	}
	return ""
}

func (vm *SInstance) GetOsVersion() string {
	val, ok := vm.Metadata["os_version"]
	if ok {
		return val
	}
	return ""
}

func (vm *SInstance) GetOsLang() string {
	val, ok := vm.Metadata["os_language"]
	if ok {
		return val
	}
	return ""
}

func (vm *SInstance) GetOsArch() string {
	return vm.OsArch
}

func (vm *SInstance) GetMachine() string {
	return vm.Machine
}

func (vm *SInstance) GetInstanceType() string {
	return vm.InstanceType
}

func (vm *SInstance) GetSecurityGroupIds() ([]string, error) {
	ret := []string{}
	for _, sec := range vm.Secgroups {
		ret = append(ret, sec.Id)
	}
	return ret, nil
}

func (vm *SInstance) GetProjectId() string {
	return vm.TenantId
}

func (vm *SInstance) SetSecurityGroups(ids []string) error {
	if vm.Hypervisor == api.HYPERVISOR_ESXI {
		return nil
	}
	input := api.GuestSetSecgroupInput{}
	input.SecgroupIds = ids
	_, err := vm.host.zone.region.perform(&modules.Servers, vm.Id, "set-secgroup", input)
	return err
}

func (vm *SInstance) GetHypervisor() string {
	return vm.Hypervisor
}

func (vm *SInstance) StartVM(ctx context.Context) error {
	if vm.Status == api.VM_RUNNING {
		return nil
	}
	_, err := vm.host.zone.region.perform(&modules.Servers, vm.Id, "start", nil)
	return err
}

func (vm *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	if vm.Status == api.VM_READY {
		return nil
	}
	input := api.ServerStopInput{}
	input.IsForce = opts.IsForce
	_, err := vm.host.zone.region.perform(&modules.Servers, vm.Id, "stop", input)
	return err
}

func (vm *SInstance) DeleteVM(ctx context.Context) error {
	if vm.DisableDelete != nil && *vm.DisableDelete {
		input := api.ServerUpdateInput{}
		disableDelete := false
		input.DisableDelete = &disableDelete
		vm.host.zone.region.cli.update(&modules.Servers, vm.Id, input)
	}
	return vm.host.zone.region.cli.delete(&modules.Servers, vm.Id)
}

func (vm *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	if vm.Name != input.NAME {
		param := api.ServerUpdateInput{}
		param.Name = input.NAME
		param.Description = input.Description
		vm.host.zone.region.cli.update(&modules.Servers, vm.Id, input)
		return cloudprovider.WaitMultiStatus(vm, []string{api.VM_READY, api.VM_RUNNING}, time.Second*5, time.Minute*3)
	}
	return nil
}

func (vm *SInstance) UpdateUserData(userData string) error {
	input := api.ServerUserDataInput{}
	input.UserData = userData
	_, err := vm.host.zone.region.perform(&modules.Servers, vm.Id, "user-data", input)
	return err
}

func (vm *SInstance) RebuildRoot(ctx context.Context, opts *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	input := api.ServerRebuildRootInput{}
	input.ImageId = opts.ImageId
	input.Password = opts.Password
	if len(opts.PublicKey) > 0 {
		keypairId, err := vm.host.zone.region.syncKeypair(vm.Name, opts.PublicKey)
		if err != nil {
			return "", errors.Wrapf(err, "syncKeypair")
		}
		input.KeypairId = keypairId
	}
	_, err := vm.host.zone.region.perform(&modules.Servers, vm.Id, "rebuild-root", input)
	if err != nil {
		return "", err
	}
	return vm.DisksInfo[0].Id, nil
}

func (vm *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	input := api.ServerDeployInput{}
	input.Password = opts.Password
	input.DeleteKeypair = opts.DeleteKeypair
	if len(opts.PublicKey) > 0 {
		keypairId, err := vm.host.zone.region.syncKeypair(vm.Name, opts.PublicKey)
		if err != nil {
			return errors.Wrapf(err, "syncKeypair")
		}
		input.KeypairId = keypairId
	}
	cloudprovider.WaitMultiStatus(vm, []string{api.VM_READY, api.VM_RUNNING}, time.Second*5, time.Minute*3)
	_, err := vm.host.zone.region.perform(&modules.Servers, vm.Id, "deploy", input)
	if err != nil {
		return errors.Wrapf(err, "deploy")
	}
	timeout := time.Minute * 3
	if vm.Hypervisor == api.HYPERVISOR_BAREMETAL {
		timeout = time.Minute * 10
	}
	return cloudprovider.WaitMultiStatus(vm, []string{api.VM_READY, api.VM_RUNNING}, time.Second*5, timeout)
}

func (vm *SInstance) ChangeConfig(ctx context.Context, opts *cloudprovider.SManagedVMChangeConfig) error {
	input := api.ServerChangeConfigInput{}
	input.VmemSize = fmt.Sprintf("%dM", opts.MemoryMB)
	input.VcpuCount = &opts.Cpu
	input.InstanceType = opts.InstanceType
	_, err := vm.host.zone.region.perform(&modules.Servers, vm.Id, "change-config", input)
	return err
}

func (vm *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return vm.host.zone.region.GetInstanceVnc(vm.Id, vm.Name)
}

func (region *SRegion) GetInstanceVnc(id, name string) (*cloudprovider.ServerVncOutput, error) {
	s := region.cli.s
	resp, err := webconsole.WebConsole.DoServerConnect(s, id, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "DoServerConnect")
	}
	result := &cloudprovider.ServerVncOutput{
		Protocol:     "cloudpods",
		InstanceId:   id,
		InstanceName: name,
		Hypervisor:   api.HYPERVISOR_DEFAULT,
	}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	resp, err = identity.ServicesV3.GetSpecific(s, "common", "config", nil)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSpecific")
	}
	result.ApiServer, _ = resp.GetString("config", "default", "api_server")
	return result, nil
}

func (vm *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	input := api.ServerAttachDiskInput{}
	input.DiskId = diskId
	_, err := vm.host.zone.region.perform(&modules.Servers, vm.Id, "attachdisk", input)
	return err
}

func (vm *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	input := api.ServerDetachDiskInput{}
	input.DiskId = diskId
	input.KeepDisk = true
	_, err := vm.host.zone.region.perform(&modules.Servers, vm.Id, "detachdisk", input)
	return err
}

func (vm *SInstance) MigrateVM(hostId string) error {
	input := api.GuestMigrateInput{}
	input.PreferHost = hostId
	input.PreferHostId = hostId
	_, err := vm.host.zone.region.perform(&modules.Servers, vm.Id, "migrate", input)
	return err
}

func (vm *SInstance) LiveMigrateVM(hostId string) error {
	input := api.GuestLiveMigrateInput{}
	input.PreferHost = hostId
	input.PreferHostId = hostId
	skipCheck := true
	input.SkipCpuCheck = &skipCheck
	input.SkipKernelCheck = &skipCheck
	_, err := vm.host.zone.region.perform(&modules.Servers, vm.Id, "live-migrate", input)
	return err
}

func (vm *SInstance) GetDetails() (*api.ServerDetails, error) {
	ret := &api.ServerDetails{}
	err := vm.host.zone.region.cli.get(&modules.Servers, vm.Id, nil, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (vm *SInstance) VMSetStatus(status string) error {
	input := apis.PerformStatusInput{}
	input.Status = status
	_, err := vm.host.zone.region.perform(&modules.Servers, vm.Id, "status", input)
	return err
}

func (vm *SInstance) GetError() error {
	if utils.IsInStringArray(vm.Status, []string{api.VM_DISK_FAILED, api.VM_SCHEDULE_FAILED, api.VM_NETWORK_FAILED}) {
		return fmt.Errorf("vm create failed with status %s", vm.Status)
	}
	if vm.Status == api.VM_DEPLOY_FAILED {
		params := map[string]interface{}{"obj_id": vm.Id, "success": false}
		actions := []apis.OpsLogDetails{}
		vm.host.zone.region.list(&logger.Actions, params, &actions)
		if len(actions) > 0 {
			return fmt.Errorf("%s", actions[0].Notes)
		}
		return fmt.Errorf("vm create failed with status %s", vm.Status)
	}
	return nil
}

func (vm *SInstance) CreateInstanceSnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudInstanceSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (vm *SInstance) GetInstanceSnapshot(idStr string) (cloudprovider.ICloudInstanceSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (vm *SInstance) GetInstanceSnapshots() ([]cloudprovider.ICloudInstanceSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (vm *SInstance) ResetToInstanceSnapshot(ctx context.Context, idStr string) error {
	return cloudprovider.ErrNotImplemented
}

func (vm *SInstance) SaveImage(opts *cloudprovider.SaveImageOptions) (cloudprovider.ICloudImage, error) {
	return vm.host.zone.region.SaveImage(vm.Id, opts.Name, opts.Notes)
}

func (region *SRegion) SaveImage(id, imageName, notes string) (*SImage, error) {
	input := api.ServerSaveImageInput{}
	input.GenerateName = imageName
	input.Notes = notes
	resp, err := region.perform(&modules.Servers, id, "save-image", input)
	if err != nil {
		return nil, err
	}
	imageId, err := resp.GetString("image_id")
	if err != nil {
		return nil, err
	}
	caches, err := region.GetStoragecaches()
	if err != nil {
		return nil, errors.Wrapf(err, "GetStoragecaches")
	}
	if len(caches) == 0 {
		return nil, fmt.Errorf("no storage cache found")
	}
	caches[0].region = region
	image, err := region.GetImage(imageId)
	if err != nil {
		return nil, err
	}
	image.cache = &caches[0]
	return image, nil
}

func (vm *SInstance) AllocatePublicIpAddress() (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (host *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	servers, err := host.zone.region.GetInstances(host.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVM{}
	for i := range servers {
		servers[i].host = host
		ret = append(ret, &servers[i])
	}
	return ret, nil
}

func (host *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	ins, err := host.zone.region.GetInstance(id)
	if err != nil {
		return nil, err
	}
	ins.host = host
	return ins, nil
}

func (region *SRegion) GetInstance(id string) (*SInstance, error) {
	ins := &SInstance{}
	return ins, region.cli.get(&modules.Servers, id, nil, ins)
}

func (region *SRegion) GetInstances(hostId string) ([]SInstance, error) {
	params := map[string]interface{}{}
	if len(hostId) > 0 {
		params["host_id"] = hostId
	}
	params["filter"] = "hypervisor.in('kvm', 'baremetal', 'pod')"
	ret := []SInstance{}
	return ret, region.list(&modules.Servers, params, &ret)
}

func (region *SRegion) CreateInstance(hostId, hypervisor string, opts *cloudprovider.SManagedVMCreateConfig) (*SInstance, error) {
	input := api.ServerCreateInput{
		ServerConfigs: &api.ServerConfigs{},
	}
	input.GenerateName = opts.Name
	input.Hostname = opts.Hostname
	input.Description = opts.Description
	input.InstanceType = opts.InstanceType
	input.VcpuCount = opts.Cpu
	input.VmemSize = opts.MemoryMB
	input.Password = opts.Password
	if len(input.Password) == 0 {
		resetPasswd := false
		input.ResetPassword = &resetPasswd
	}
	input.PublicIpBw = opts.PublicIpBw
	input.PublicIpChargeType = billing_api.TNetChargeType(opts.PublicIpChargeType)
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
	image, err := region.GetImage(opts.ExternalImageId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetImage")
	}
	imageId := opts.ExternalImageId
	if image.DiskFormat == "iso" {
		input.Cdrom = opts.ExternalImageId
		imageId = ""
	}
	sysDisk := &api.DiskConfig{
		Index:    0,
		ImageId:  imageId,
		DiskType: api.DISK_TYPE_SYS,
		SizeMb:   opts.SysDisk.SizeGB * 1024,
		Backend:  opts.SysDisk.StorageType,
		Storage:  opts.SysDisk.StorageExternalId,
	}
	if len(opts.SysDisk.Driver) > 0 {
		sysDisk.Driver = opts.SysDisk.Driver
	}
	if len(opts.SysDisk.CacheMode) > 0 {
		sysDisk.Cache = opts.SysDisk.CacheMode
	}
	input.Disks = append(input.Disks, sysDisk)
	for idx, disk := range opts.DataDisks {
		dataDisk := &api.DiskConfig{
			Index:    idx + 1,
			DiskType: api.DISK_TYPE_DATA,
			SizeMb:   disk.SizeGB * 1024,
			Backend:  disk.StorageType,
			Storage:  disk.StorageExternalId,
		}
		if len(disk.Driver) > 0 {
			dataDisk.Driver = disk.Driver
		}
		if len(disk.CacheMode) > 0 {
			dataDisk.Cache = disk.CacheMode
		}
		input.Disks = append(input.Disks, dataDisk)
	}
	input.IsolatedDevices = []*api.IsolatedDeviceConfig{}
	for _, dev := range input.IsolatedDevices {
		devConfig := &api.IsolatedDeviceConfig{
			Id: dev.Id,
		}
		input.IsolatedDevices = append(input.IsolatedDevices, devConfig)
	}
	input.Networks = append(input.Networks, &api.NetworkConfig{
		Index:   0,
		Network: opts.ExternalNetworkId,
		Address: opts.IpAddr,
	})
	ins := &SInstance{}
	return ins, region.create(&modules.Servers, input, ins)
}

func (vm *SInstance) CreateDisk(ctx context.Context, opts *cloudprovider.GuestDiskCreateOptions) (string, error) {
	diskIds := []string{}
	for _, disk := range vm.DisksInfo {
		diskIds = append(diskIds, disk.Id)
	}
	input := jsonutils.Marshal(map[string]interface{}{
		"disks": []map[string]interface{}{
			{
				"size":          opts.SizeMb,
				"storage_id":    opts.StorageId,
				"preallocation": opts.Preallocation,
			},
		},
	})
	_, err := vm.host.zone.region.perform(&modules.Servers, vm.Id, "createdisk", input)
	if err != nil {
		return "", err
	}
	ret := ""
	cloudprovider.Wait(time.Second*3, time.Minute*3, func() (bool, error) {
		err = vm.Refresh()
		if err != nil {
			return false, errors.Wrapf(err, "Refresh")
		}

		for _, disk := range vm.DisksInfo {
			if !utils.IsInStringArray(disk.Id, diskIds) {
				ret = disk.Id
				return true, nil
			}
		}
		return false, nil
	})
	if len(ret) > 0 {
		return ret, nil
	}
	return "", errors.Wrapf(cloudprovider.ErrNotFound, "after disk created")
}

func (vm *SInstance) GetIsolateDeviceIds() ([]string, error) {
	devs, err := vm.host.zone.region.GetIsolatedDevices("", vm.Id)
	if err != nil {
		return nil, err
	}
	ret := []string{}
	for i := range devs {
		ret = append(ret, devs[i].GetGlobalId())
	}
	return ret, nil
}

func (vm *SInstance) GetContainers() ([]cloudprovider.ICloudContainer, error) {
	containers, err := vm.host.zone.region.GetContainers(vm.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudContainer{}
	for i := range containers {
		containers[i].region = vm.host.zone.region
		ret = append(ret, &containers[i])
	}
	return ret, nil
}
