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

package baidu

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

type SInstance struct {
	multicloud.SInstanceBase
	SBaiduTag
	host *SHost

	Id                    string
	Name                  string
	RoleName              string
	Hostname              string
	InstanceType          string
	Spec                  string
	Status                string
	Desc                  string
	CreatedFrom           string
	PaymentTiming         string
	CreateTime            time.Time
	InternalIP            string
	PublicIP              string
	CPUCount              int
	IsomerismCard         string
	CardCount             string
	NpuVideoMemory        string
	MemoryCapacityInGB    int
	LocalDiskSizeInGB     int
	ImageId               string
	ImageName             string
	ImageType             string
	PlacementPolicy       string
	SubnetId              string
	VpcId                 string
	HostId                string
	SwitchId              string
	RackId                string
	DeploysetId           string
	ZoneName              string
	DedicatedHostId       string
	OsVersion             string
	OsArch                string
	OsName                string
	HosteyeType           string
	NicInfo               NicInfo
	DeletionProtection    int
	Ipv6                  string
	Volumes               []SDisk
	NetworkCapacityInMbps int
}

func (region *SRegion) GetInstances(zoneName string, instanceIds []string) ([]SInstance, error) {
	params := url.Values{}
	if len(zoneName) > 0 {
		params.Set("zoneName", zoneName)
	}
	if len(instanceIds) > 0 {
		params.Set("instanceIds", strings.Join(instanceIds, ","))
	}
	ret := []SInstance{}
	for {
		resp, err := region.bccList("v2/instance", params)
		if err != nil {
			return nil, errors.Wrap(err, "list instance")
		}
		part := struct {
			NextMarker string
			Instances  []SInstance
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Instances...)
		if len(part.NextMarker) == 0 {
			break
		}
		params.Set("marker", part.NextMarker)
	}
	return ret, nil
}

func (region *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	resp, err := region.bccList(fmt.Sprintf("v2/instance/%s", instanceId), nil)
	if err != nil {
		return nil, errors.Wrap(err, "show instance")
	}
	ret := &SInstance{}
	err = resp.Unmarshal(ret, "instance")
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal instance")
	}
	return ret, nil
}

func (ins *SInstance) AssignSecurityGroup(secgroupId string) error {
	return ins.host.zone.region.AssignSecurityGroup(ins.Id, secgroupId)
}

func (region *SRegion) AssignSecurityGroup(vmId string, secgroupId string) error {
	params := url.Values{}
	params.Set("bind", "")
	body := map[string]interface{}{
		"securityGroupId": secgroupId,
	}
	_, err := region.bccUpdate(fmt.Sprintf("v2/instance/%s", vmId), params, body)
	return err
}

func (region *SRegion) RevokeSecurityGroup(vmId string, secgroupId string) error {
	params := url.Values{}
	params.Set("unbind", "")
	body := map[string]interface{}{
		"securityGroupId": secgroupId,
	}
	_, err := region.bccUpdate(fmt.Sprintf("v2/instance/%s", vmId), params, body)
	return err
}

func (ins *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return ins.host.zone.region.AttachDisk(ins.Id, diskId)
}

func (region *SRegion) AttachDisk(vmId string, diskId string) error {
	params := url.Values{}
	params.Set("attach", "")
	body := map[string]interface{}{
		"instanceId": vmId,
	}
	_, err := region.bccUpdate(fmt.Sprintf("v2/volume/%s", diskId), params, body)
	return err
}

func (ins *SInstance) ChangeConfig(ctx context.Context, opts *cloudprovider.SManagedVMChangeConfig) error {
	return ins.host.zone.region.ChangeVmConfig(ins.Id, opts.InstanceType)
}

func (region *SRegion) ChangeVmConfig(vmId string, instanceType string) error {
	params := url.Values{}
	params.Set("clientToken", utils.GenRequestId(20))
	params.Set("resize", "")
	body := map[string]interface{}{
		"spec": instanceType,
	}
	_, err := region.bccUpdate(fmt.Sprintf("v2/instance/%s", vmId), params, body)
	return err
}

func (ins *SInstance) DeleteVM(ctx context.Context) error {
	return ins.host.zone.region.DeleteVm(ins.Id)
}

func (ins *SInstance) Refresh() error {
	vm, err := ins.host.zone.region.GetInstance(ins.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(ins, vm)
}

func (region *SRegion) DeleteVm(id string) error {
	params := url.Values{}
	body := map[string]interface{}{
		"relatedReleaseFlag":    true,
		"deleteCdsSnapshotFlag": true,
		"deleteRelatedEnisFlag": true,
		"bccRecycleFlag":        false,
	}
	_, err := region.bccPost(fmt.Sprintf("v2/instance/%s", id), params, body)
	return err
}

func (ins *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return ins.host.zone.region.DetachDisk(ins.Id, diskId)
}

func (region *SRegion) DetachDisk(vmId string, diskId string) error {
	params := url.Values{}
	params.Set("detach", "")
	body := map[string]interface{}{
		"instanceId": vmId,
	}
	_, err := region.bccUpdate(fmt.Sprintf("v2/volume/%s", diskId), params, body)
	return err
}

func (ins *SInstance) GetBios() cloudprovider.TBiosType {
	return ""
}

func (ins *SInstance) GetBootOrder() string {
	return "bcd"
}

func (ins *SInstance) GetError() error {
	return nil
}

func (ins *SInstance) GetFullOsName() string {
	return ""
}

func (ins *SInstance) GetGlobalId() string {
	return ins.Id
}

func (ins *SInstance) GetId() string {
	return ins.Id
}

func (ins *SInstance) GetInstanceType() string {
	return ins.Spec
}

func (ins *SInstance) GetMachine() string {
	return "pc"
}

func (ins *SInstance) GetHostname() string {
	return ins.Hostname
}

func (ins *SInstance) GetName() string {
	return ins.Name
}

func (ins *SInstance) GetOsArch() string {
	return getOsArch(ins.OsArch)
}

func (ins *SInstance) GetOsDist() string {
	return ""
}

func (ins *SInstance) GetOsLang() string {
	return ins.OsName
}

func (ins *SInstance) GetOsType() cloudprovider.TOsType {
	if strings.Contains(strings.ToLower(ins.OsName), "windows") {
		return cloudprovider.OsTypeWindows
	}
	return cloudprovider.OsTypeLinux
}

func (ins *SInstance) GetOsVersion() string {
	return ins.OsVersion
}

func (ins *SInstance) GetProjectId() string {
	return ""
}

func (ins *SInstance) GetSecurityGroupIds() ([]string, error) {
	return ins.NicInfo.SecurityGroups, nil
}

func (ins *SInstance) GetStatus() string {
	switch ins.Status {
	case "Running":
		return api.VM_RUNNING
	case "Stopped":
		return api.VM_READY
	case "Stopping":
		return api.VM_STOPPING
	case "Starting":
		return api.VM_STARTING
	default:
		return strings.ToLower(ins.Status)
	}
}

func (ins *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_BAIDU
}

func (ins *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := ins.host.zone.region.GetDisks("", "", ins.Id)
	if err != nil {
		return nil, err
	}
	storages, err := ins.host.zone.region.GetStorageTypes(ins.host.zone.ZoneName)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		for j := range storages {
			if isMatchStorageType(disks[i].StorageType, storages[j].StorageType) {
				storages[i].zone = ins.host.zone
				disks[i].storage = &storages[i]
				ret = append(ret, &disks[i])
				break
			}
		}
	}
	return ret, nil
}

func (ins *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	eips, err := ins.host.zone.region.GetEips(ins.Id)
	if err != nil {
		return nil, err
	}
	for i := range eips {
		eips[i].region = ins.host.zone.region
		return &eips[i], nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (ins *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	return []cloudprovider.ICloudNic{&ins.NicInfo}, nil
}

func (ins *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	url, err := ins.host.zone.region.GetInstanceVnc(ins.Id)
	if err != nil {
		return nil, err
	}
	return &cloudprovider.ServerVncOutput{
		Url:        url,
		Protocol:   "baidu",
		InstanceId: ins.Id,
		Hypervisor: api.HYPERVISOR_BAIDU,
		OsName:     string(ins.GetOsType()),
	}, nil
}

func (ins *SInstance) GetVcpuCount() int {
	return ins.CPUCount
}

func (ins *SInstance) GetVmemSizeMB() int {
	return ins.MemoryCapacityInGB * 1024
}

func (ins *SInstance) GetVdi() string {
	return ""
}

func (ins *SInstance) GetVga() string {
	return ""
}

func (ins *SInstance) RebuildRoot(ctx context.Context, opts *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	disks, err := ins.host.zone.region.GetDisks("", "", ins.Id)
	if err != nil {
		return "", err
	}
	systemDiskId := ""
	for _, disk := range disks {
		if disk.IsSystemVolume {
			systemDiskId = disk.Id
		}
	}
	err = ins.host.zone.region.RebuildRoot(ins.Id, opts)
	if err != nil {
		return "", err
	}
	return systemDiskId, nil
}

func (region *SRegion) RebuildRoot(vmId string, opts *cloudprovider.SManagedVMRebuildRootConfig) error {
	params := url.Values{}
	params.Set("rebuild", "")
	body := map[string]interface{}{
		"imageId":  opts.ImageId,
		"userData": opts.UserData,
	}
	var err error
	if len(opts.PublicKey) > 0 {
		keypair, err := region.SyncKeypair(fmt.Sprintf("auto-generate-%d", time.Now().Unix()), opts.PublicKey)
		if err != nil {
			return err
		}
		body["keypairId"] = keypair.KeypairId
	} else if len(opts.Password) > 0 {
		body["adminPass"], err = AesECBEncryptHex(region.client.accessKeySecret, opts.Password)
		if err != nil {
			return err
		}
	}
	_, err = region.bccUpdate(fmt.Sprintf("v2/instance/%s", vmId), params, body)
	return err
}

func (ins *SInstance) SetSecurityGroups(secgroupIds []string) error {
	current, err := ins.GetSecurityGroupIds()
	if err != nil {
		return errors.Wrapf(err, "GetSecurityGroupIds")
	}
	for _, secgroupId := range secgroupIds {
		if !utils.IsInStringArray(secgroupId, current) {
			err = ins.host.zone.region.AssignSecurityGroup(ins.Id, secgroupId)
			if err != nil {
				return errors.Wrapf(err, "AssignSecurityGroup %s", secgroupId)
			}
		}
	}
	for _, secgroupId := range current {
		if !utils.IsInStringArray(secgroupId, secgroupIds) {
			err = ins.host.zone.region.RevokeSecurityGroup(ins.Id, secgroupId)
			if err != nil {
				return errors.Wrapf(err, "RevokeSecurityGroup %s", secgroupId)
			}
		}
	}
	return nil
}

func (ins *SInstance) StartVM(ctx context.Context) error {
	return ins.host.zone.region.StartVM(ins.Id)
}

func (region *SRegion) StartVM(vmId string) error {
	params := url.Values{}
	params.Set("start", "")
	_, err := region.bccUpdate(fmt.Sprintf("v2/instance/%s", vmId), params, nil)
	return err
}

func (ins *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	return ins.host.zone.region.StopVM(ins.Id, opts)
}

func (region *SRegion) StopVM(vmId string, opts *cloudprovider.ServerStopOptions) error {
	params := url.Values{}
	params.Set("stop", "")
	body := map[string]interface{}{
		"forceStop":        opts.IsForce,
		"stopWithNoCharge": opts.StopCharging,
	}
	_, err := region.bccUpdate(fmt.Sprintf("v2/instance/%s", vmId), params, body)
	return err
}

func (ins *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

func (ins *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	if len(input.HostName) > 0 {
		err := ins.host.zone.region.UpdateVmHostname(ins.Id, input.HostName)
		if err != nil {
			return err
		}
	}
	if len(input.Description) > 0 {
		err := ins.host.zone.region.UpdateVmDesc(ins.Id, input.Description)
		if err != nil {
			return err
		}
	}
	if len(input.NAME) > 0 {
		err := ins.host.zone.region.UpdateVmAttr(ins.Id, input.NAME)
		if err != nil {
			return err
		}
	}
	return nil
}

func (region *SRegion) UpdateVmAttr(vmId string, name string) error {
	params := url.Values{}
	params.Set("modifyAttribute", "")
	body := map[string]interface{}{
		"name": name,
	}
	_, err := region.bccUpdate(fmt.Sprintf("v2/instance/%s", vmId), params, body)
	return err
}

func (region *SRegion) UpdateVmDesc(vmId string, desc string) error {
	params := url.Values{}
	params.Set("modifyDesc", "")
	body := map[string]interface{}{
		"desc": desc,
	}
	_, err := region.bccUpdate(fmt.Sprintf("v2/instance/%s", vmId), params, body)
	return err
}

func (region *SRegion) UpdateVmHostname(vmId string, hostname string) error {
	params := url.Values{}
	params.Set("changeHostname", "")
	body := map[string]interface{}{
		"reboot":   false,
		"hostname": hostname,
	}
	_, err := region.bccUpdate(fmt.Sprintf("v2/instance/%s", vmId), params, body)
	return err
}

func (ins *SInstance) GetIHost() cloudprovider.ICloudHost {
	return ins.host
}

func (region *SRegion) UpdateVmPassword(vmId string, passwd string) error {
	params := url.Values{}
	params.Set("changePass", "")
	body := map[string]interface{}{
		"adminPass": passwd,
	}
	_, err := region.bccUpdate(fmt.Sprintf("v2/instance/%s", vmId), params, body)
	return err
}

func (ins *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	if len(opts.Password) > 0 {
		return ins.host.zone.region.UpdateVmPassword(ins.Id, opts.Password)
	}
	return nil
}

func (region *SRegion) GetInstanceVnc(vmId string) (string, error) {
	params := url.Values{}
	resp, err := region.bccList(fmt.Sprintf("v2/instance/%s/vnc", vmId), params)
	if err != nil {
		return "", err
	}
	return resp.GetString("vncUrl")
}
