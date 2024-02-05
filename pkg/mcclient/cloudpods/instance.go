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
	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/mcclient/modules/logger"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	"yunion.io/x/onecloud/pkg/mcclient/modules/webconsole"
)

type SInstance struct {
	multicloud.SInstanceBase
	CloudpodsTags

	host *SHost
	api.ServerDetails
}

func (self *SInstance) GetName() string {
	return self.Name
}

func (self *SInstance) GetHostname() string {
	return self.Hostname
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
	self.DisksInfo = nil
	self.Nics = nil
	self.Secgroups = nil
	self.SubIPs = nil
	self.IsolatedDevices = nil
	self.Cdrom = nil
	self.Floppy = nil
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

func (self *SInstance) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(self.OsType)
}

func (self *SInstance) GetFullOsName() string {
	return self.OsName
}

func (self *SInstance) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(self.Bios)
}

func (ins *SInstance) GetOsDist() string {
	val, ok := ins.Metadata["os_distribution"]
	if ok {
		return val
	}
	return ""
}

func (ins *SInstance) GetOsVersion() string {
	val, ok := ins.Metadata["os_version"]
	if ok {
		return val
	}
	return ""
}

func (ins *SInstance) GetOsLang() string {
	val, ok := ins.Metadata["os_language"]
	if ok {
		return val
	}
	return ""
}

func (ins *SInstance) GetOsArch() string {
	return ins.OsArch
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

func (self *SInstance) SetSecurityGroups(ids []string) error {
	if self.Hypervisor == api.HYPERVISOR_ESXI {
		return nil
	}
	input := api.GuestSetSecgroupInput{}
	input.SecgroupIds = ids
	_, err := self.host.zone.region.perform(&modules.Servers, self.Id, "set-secgroup", input)
	return err
}

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_CLOUDPODS
}

func (self *SInstance) StartVM(ctx context.Context) error {
	if self.Status == api.VM_RUNNING {
		return nil
	}
	_, err := self.host.zone.region.perform(&modules.Servers, self.Id, "start", nil)
	return err
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	if self.Status == api.VM_READY {
		return nil
	}
	input := api.ServerStopInput{}
	input.IsForce = opts.IsForce
	_, err := self.host.zone.region.perform(&modules.Servers, self.Id, "stop", input)
	return err
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	if self.DisableDelete != nil && *self.DisableDelete {
		input := api.ServerUpdateInput{}
		disableDelete := false
		input.DisableDelete = &disableDelete
		self.host.zone.region.cli.update(&modules.Servers, self.Id, input)
	}
	return self.host.zone.region.cli.delete(&modules.Servers, self.Id)
}

func (self *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	if self.Name != input.NAME {
		param := api.ServerUpdateInput{}
		param.Name = input.NAME
		param.Description = input.Description
		self.host.zone.region.cli.update(&modules.Servers, self.Id, input)
		return cloudprovider.WaitMultiStatus(self, []string{api.VM_READY, api.VM_RUNNING}, time.Second*5, time.Minute*3)
	}
	return nil
}

func (self *SInstance) UpdateUserData(userData string) error {
	input := api.ServerUserDataInput{}
	input.UserData = userData
	_, err := self.host.zone.region.perform(&modules.Servers, self.Id, "user-data", input)
	return err
}

func (self *SInstance) RebuildRoot(ctx context.Context, opts *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	input := api.ServerRebuildRootInput{}
	input.ImageId = opts.ImageId
	input.Password = opts.Password
	if len(opts.PublicKey) > 0 {
		keypairId, err := self.host.zone.region.syncKeypair(self.Name, opts.PublicKey)
		if err != nil {
			return "", errors.Wrapf(err, "syncKeypair")
		}
		input.KeypairId = keypairId
	}
	_, err := self.host.zone.region.perform(&modules.Servers, self.Id, "rebuild-root", input)
	if err != nil {
		return "", err
	}
	return self.DisksInfo[0].Id, nil
}

func (self *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	input := api.ServerDeployInput{}
	input.Password = opts.Password
	input.DeleteKeypair = opts.DeleteKeypair
	if len(opts.PublicKey) > 0 {
		keypairId, err := self.host.zone.region.syncKeypair(self.Name, opts.PublicKey)
		if err != nil {
			return errors.Wrapf(err, "syncKeypair")
		}
		input.KeypairId = keypairId
	}
	cloudprovider.WaitMultiStatus(self, []string{api.VM_READY, api.VM_RUNNING}, time.Second*5, time.Minute*3)
	_, err := self.host.zone.region.perform(&modules.Servers, self.Id, "deploy", input)
	if err != nil {
		return errors.Wrapf(err, "deploy")
	}
	return cloudprovider.WaitMultiStatus(self, []string{api.VM_READY, api.VM_RUNNING}, time.Second*5, time.Minute*3)
}

func (self *SInstance) ChangeConfig(ctx context.Context, opts *cloudprovider.SManagedVMChangeConfig) error {
	input := api.ServerChangeConfigInput{}
	input.VmemSize = fmt.Sprintf("%dM", opts.MemoryMB)
	input.VcpuCount = &opts.Cpu
	input.InstanceType = opts.InstanceType
	_, err := self.host.zone.region.perform(&modules.Servers, self.Id, "change-config", input)
	return err
}

func (self *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return self.host.zone.region.GetInstanceVnc(self.Id, self.Name)
}

func (self *SRegion) GetInstanceVnc(id, name string) (*cloudprovider.ServerVncOutput, error) {
	s := self.cli.s
	resp, err := webconsole.WebConsole.DoServerConnect(s, id, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "DoServerConnect")
	}
	result := &cloudprovider.ServerVncOutput{
		Protocol:     "cloudpods",
		InstanceId:   id,
		InstanceName: name,
		Hypervisor:   api.HYPERVISOR_CLOUDPODS,
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

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	input := api.ServerAttachDiskInput{}
	input.DiskId = diskId
	_, err := self.host.zone.region.perform(&modules.Servers, self.Id, "attachdisk", input)
	return err
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	input := api.ServerDetachDiskInput{}
	input.DiskId = diskId
	input.KeepDisk = true
	_, err := self.host.zone.region.perform(&modules.Servers, self.Id, "detachdisk", input)
	return err
}

func (self *SInstance) MigrateVM(hostId string) error {
	input := api.GuestMigrateInput{}
	input.PreferHost = hostId
	_, err := self.host.zone.region.perform(&modules.Servers, self.Id, "migrate", input)
	return err
}

func (self *SInstance) LiveMigrateVM(hostId string) error {
	input := api.GuestLiveMigrateInput{}
	input.PreferHost = hostId
	skipCpuCheck := true
	input.SkipCpuCheck = &skipCpuCheck
	_, err := self.host.zone.region.perform(&modules.Servers, self.Id, "live-migrate", input)
	return err
}

func (self *SInstance) GetDetails() (*api.ServerDetails, error) {
	ret := &api.ServerDetails{}
	err := self.host.zone.region.cli.get(&modules.Servers, self.Id, nil, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SInstance) VMSetStatus(status string) error {
	input := apis.PerformStatusInput{}
	input.Status = status
	_, err := self.host.zone.region.perform(&modules.Servers, self.Id, "status", input)
	return err
}

func (self *SInstance) GetError() error {
	if utils.IsInStringArray(self.Status, []string{api.VM_DISK_FAILED, api.VM_SCHEDULE_FAILED, api.VM_NETWORK_FAILED}) {
		return fmt.Errorf("vm create failed with status %s", self.Status)
	}
	if self.Status == api.VM_DEPLOY_FAILED {
		params := map[string]interface{}{"obj_id": self.Id, "success": false}
		actions := []apis.OpsLogDetails{}
		self.host.zone.region.list(&logger.Actions, params, &actions)
		if len(actions) > 0 {
			return fmt.Errorf(actions[0].Notes)
		}
		return fmt.Errorf("vm create failed with status %s", self.Status)
	}
	return nil
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
	return self.host.zone.region.SaveImage(self.Id, opts.Name, opts.Notes)
}

func (self *SRegion) SaveImage(id, imageName, notes string) (*SImage, error) {
	input := api.ServerSaveImageInput{}
	input.GenerateName = imageName
	input.Notes = notes
	resp, err := self.perform(&modules.Servers, id, "save-image", input)
	if err != nil {
		return nil, err
	}
	imageId, err := resp.GetString("image_id")
	if err != nil {
		return nil, err
	}
	caches, err := self.GetStoragecaches()
	if err != nil {
		return nil, errors.Wrapf(err, "GetStoragecaches")
	}
	if len(caches) == 0 {
		return nil, fmt.Errorf("no storage cache found")
	}
	caches[0].region = self
	image, err := self.GetImage(imageId)
	if err != nil {
		return nil, err
	}
	image.cache = &caches[0]
	return image, nil
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
	input := api.ServerCreateInput{
		ServerConfigs: &api.ServerConfigs{},
	}
	input.Name = opts.Name
	input.Hostname = opts.Hostname
	input.Description = opts.Description
	input.InstanceType = opts.InstanceType
	input.VcpuCount = opts.Cpu
	input.VmemSize = opts.MemoryMB
	input.Password = opts.Password
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
		DiskType: api.DISK_TYPE_SYS,
		SizeMb:   opts.SysDisk.SizeGB * 1024,
		Backend:  opts.SysDisk.StorageType,
		Storage:  opts.SysDisk.StorageExternalId,
	})
	for idx, disk := range opts.DataDisks {
		input.Disks = append(input.Disks, &api.DiskConfig{
			Index:    idx + 1,
			DiskType: api.DISK_TYPE_DATA,
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

type SMetricData struct {
	Id    string    `json:"id"`
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

func (cli *SCloudpodsClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	brandArr := []string{"OneCloud", "VMware"}
	metrics := []SMetricData{}
	usefulResourceType := []cloudprovider.TResourceType{cloudprovider.METRIC_RESOURCE_TYPE_HOST, cloudprovider.METRIC_RESOURCE_TYPE_SERVER}
	isUse, _ := utils.InArray(opts.ResourceType, usefulResourceType)
	if !isUse {
		return nil, nil
	}
	for i := 0; i < len(brandArr); i++ {
		params := map[string]interface{}{
			"metric_name": opts.MetricType,
			"start_time":  opts.StartTime,
			"end_time":    opts.EndTime,
			"tag_pairs": map[string]interface{}{
				"brand": brandArr[i],
			},
		}
		onecloudObj, err := monitor.UnifiedMonitorManager.Get(cli.s, "simple-query", jsonutils.Marshal(params))
		if err != nil {
			return nil, err
		}
		tmp := []SMetricData{}
		onecloudObj.Unmarshal(&tmp, "values")
		metrics = append(metrics, tmp...)
	}

	idWithMetric := map[string][]cloudprovider.MetricValue{}
	for _, v := range metrics {
		if _, isExist := idWithMetric[v.Id]; !isExist {
			idWithMetric[v.Id] = []cloudprovider.MetricValue{{
				Timestamp: v.Time,
				Value:     v.Value}}
		} else {
			idWithMetric[v.Id] = append(idWithMetric[v.Id], cloudprovider.MetricValue{
				Timestamp: v.Time,
				Value:     v.Value})
		}
	}

	res := []cloudprovider.MetricValues{}
	for _, metric := range metrics {
		res = append(res, cloudprovider.MetricValues{
			Id:         metric.Id,
			MetricType: opts.MetricType,
			Values:     idWithMetric[metric.Id],
		})
	}
	return res, nil
}

func (self *SInstance) CreateDisk(ctx context.Context, opts *cloudprovider.GuestDiskCreateOptions) (string, error) {
	diskIds := []string{}
	for _, disk := range self.DisksInfo {
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
	_, err := self.host.zone.region.perform(&modules.Servers, self.Id, "createdisk", input)
	if err != nil {
		return "", err
	}
	ret := ""
	cloudprovider.Wait(time.Second*3, time.Minute*3, func() (bool, error) {
		err = self.Refresh()
		if err != nil {
			return false, errors.Wrapf(err, "Refresh")
		}

		for _, disk := range self.DisksInfo {
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
