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

package ksyun

import (
	"context"
	"fmt"
	"time"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/imagetools"
	"yunion.io/x/pkg/utils"
)

type InstanceConfigure struct {
	Vcpu         int    `json:"VCPU"`
	Gpu          int    `json:"GPU"`
	MemoryGb     int    `json:"MemoryGb"`
	DataDiskGb   int    `json:"DataDiskGb"`
	RootDiskGb   int    `json:"RootDiskGb"`
	DataDiskType string `json:"DataDiskType"`
	Vgpu         string `json:"VGPU"`
}

type InstanceState struct {
	Name      string `json:"Name"`
	OnMigrate bool   `json:"OnMigrate"`
	CostTime  string `json:"CostTime"`
	TimeStamp string `json:"TimeStamp"`
}

type Monitoring struct {
	State string `json:"State"`
}

type GroupSet struct {
	GroupID string `json:"GroupId"`
}

type InstanceSecurityGroupSet struct {
	SecurityGroupId string `json:"SecurityGroupId"`
}

type NetworkInterfaceSet struct {
	AllocationId         string                     `json:"AllocationId"`
	NetworkInterfaceId   string                     `json:"NetworkInterfaceId"`
	NetworkInterfaceType string                     `json:"NetworkInterfaceType"`
	VpcId                string                     `json:"VpcId"`
	SubnetId             string                     `json:"SubnetId"`
	MacAddress           string                     `json:"MacAddress"`
	PrivateIPAddress     string                     `json:"PrivateIpAddress"`
	GroupSet             []GroupSet                 `json:"GroupSet"`
	SecurityGroupSet     []InstanceSecurityGroupSet `json:"SecurityGroupSet"`
	NetworkInterfaceName string                     `json:"NetworkInterfaceName"`
}

type SystemDisk struct {
	DiskType string `json:"DiskType"`
	DiskSize int    `json:"DiskSize"`
}

type DataDisks struct {
	DiskId             string `json:"DiskId"`
	DiskType           string `json:"DiskType"`
	DiskSize           int    `json:"DiskSize"`
	DeleteWithInstance bool   `json:"DeleteWithInstance"`
	Encrypted          bool   `json:"Encrypted"`
}

type SInstance struct {
	multicloud.SInstanceBase
	SKsyunTags
	host   *SHost
	region *SRegion

	InstanceId            string                `json:"InstanceId"`
	ProjectId             string                `json:"ProjectId"`
	ShutdownNoCharge      bool                  `json:"ShutdownNoCharge"`
	IsDistributeIpv6      bool                  `json:"IsDistributeIpv6"`
	InstanceName          string                `json:"InstanceName"`
	InstanceType          string                `json:"InstanceType"`
	InstanceConfigure     InstanceConfigure     `json:"InstanceConfigure"`
	ImageID               string                `json:"ImageId"`
	SubnetId              string                `json:"SubnetId"`
	PrivateIPAddress      string                `json:"PrivateIpAddress"`
	InstanceState         InstanceState         `json:"InstanceState"`
	Monitoring            Monitoring            `json:"Monitoring"`
	NetworkInterfaceSet   []NetworkInterfaceSet `json:"NetworkInterfaceSet"`
	SriovNetSupport       string                `json:"SriovNetSupport"`
	IsShowSriovNetSupport bool                  `json:"IsShowSriovNetSupport"`
	CreationDate          time.Time             `json:"CreationDate"`
	AvailabilityZone      string                `json:"AvailabilityZone"`
	AvailabilityZoneName  string                `json:"AvailabilityZoneName"`
	DedicatedUuid         string                `json:"DedicatedUuid"`
	ProductType           int                   `json:"ProductType"`
	ProductWhat           int                   `json:"ProductWhat"`
	LiveUpgradeSupport    bool                  `json:"LiveUpgradeSupport"`
	ChargeType            string                `json:"ChargeType"`
	SystemDisk            SystemDisk            `json:"SystemDisk"`
	HostName              string                `json:"HostName"`
	UserData              string                `json:"UserData"`
	Migration             int                   `json:"Migration"`
	DataDisks             []DataDisks           `json:"DataDisks"`
	VncSupport            bool                  `json:"VncSupport"`
	Platform              string                `json:"Platform"`
	ServiceEndTime        time.Time             `json:"ServiceEndTime"`
}

func (region *SRegion) GetInstances(zoneName string, instanceIds []string) ([]SInstance, error) {
	projectIds, err := region.client.GetSplitProjectIds()
	if err != nil {
		return nil, err
	}
	ret := []SInstance{}
	for _, projects := range projectIds {
		part, err := region.getInstances(zoneName, instanceIds, projects)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part...)
	}
	return ret, nil
}

func (region *SRegion) getInstances(zoneName string, instanceIds []string, projectIds []string) ([]SInstance, error) {
	instances := []SInstance{}
	params := map[string]string{
		"MaxResults": "1000",
		"Marker":     "0",
	}
	if len(zoneName) > 0 {
		params["Filter.1.Name"] = "availability-zone-name"
		params["Filter.1.Value.1"] = zoneName
	}
	for i, v := range instanceIds {
		params[fmt.Sprintf("InstanceId.%d", i+1)] = v
	}
	for i, v := range projectIds {
		params[fmt.Sprintf("ProjectId.%d", i+1)] = v
	}
	for {
		resp, err := region.ecsRequest("DescribeInstances", params)
		if err != nil {
			return nil, errors.Wrap(err, "list instance")
		}
		part := struct {
			InstancesSet  []SInstance `json:"InstancesSet"`
			Marker        int         `json:"Marker"`
			InstanceCount int         `json:"InstanceCount"`
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal instances")
		}
		instances = append(instances, part.InstancesSet...)
		if len(instances) >= part.InstanceCount {
			break
		}
		params["Marker"] = fmt.Sprintf("%d", part.Marker)
	}

	return instances, nil
}

func (region *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	instances, err := region.GetInstances("", []string{instanceId})
	if err != nil {
		return nil, errors.Wrap(err, "GetInstances")
	}
	for i := range instances {
		if instances[i].GetGlobalId() == instanceId {
			return &instances[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s", instanceId)
}

func (ins *SInstance) Refresh() error {
	extIns, err := ins.getRegion().GetInstance(ins.GetGlobalId())
	if err != nil {
		return errors.Wrap(err, "GetInstance")
	}
	return jsonutils.Update(ins, extIns)
}

func (ins *SInstance) GetTags() (map[string]string, error) {
	tags, err := ins.getRegion().ListTags("kec-instance", ins.InstanceId)
	if err != nil {
		return nil, err
	}
	return tags.GetTags(), nil
}

func (ins *SInstance) getRegion() *SRegion {
	if ins.region != nil {
		return ins.region
	}
	return ins.host.zone.region
}

func (ins *SInstance) AssignSecurityGroup(secgroupId string) error {
	groupIds, err := ins.GetSecurityGroupIds()
	if err != nil {
		return err
	}
	groupIds = append(groupIds, secgroupId)
	return ins.SetSecurityGroups(groupIds)
}

func (ins *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return ins.getRegion().AttachDisk(ins.InstanceId, diskId)
}

func (region *SRegion) AttachDisk(instanceId, diskId string) error {
	params := map[string]string{
		"VolumeId":           diskId,
		"InstanceId":         instanceId,
		"DeleteWithInstance": "true",
	}
	_, err := region.ebsRequest("AttachVolume", params)
	return err
}

func (ins *SInstance) ChangeConfig(ctx context.Context, opts *cloudprovider.SManagedVMChangeConfig) error {
	return ins.getRegion().ChangeConfig(ins.InstanceId, opts)
}

func (region *SRegion) ChangeConfig(instanceId string, opts *cloudprovider.SManagedVMChangeConfig) error {
	params := map[string]string{
		"InstanceId":   instanceId,
		"InstanceType": opts.InstanceType,
	}
	_, err := region.ecsRequest("ModifyInstanceType", params)
	return err
}

func (ins *SInstance) DeleteVM(ctx context.Context) error {
	return ins.getRegion().DeleteVM(ins.InstanceId)
}

func (region *SRegion) DeleteVM(instanceId string) error {
	params := map[string]string{
		"InstanceId.1": instanceId,
		"ForceDelete":  "true",
	}
	_, err := region.ecsRequest("TerminateInstances", params)
	return err
}

func (ins *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return ins.getRegion().DetachDisk(ins.InstanceId, diskId)
}

func (region *SRegion) DetachDisk(instanceId, diskId string) error {
	params := map[string]string{
		"VolumeId": diskId,
	}
	if len(instanceId) > 0 {
		params["InstanceId"] = instanceId
	}
	_, err := region.ecsRequest("DetachVolume", params)
	return err
}

func (ins *SInstance) GetBios() cloudprovider.TBiosType {
	return ""
}

func (ins *SInstance) GetBootOrder() string {
	return ""
}

func (ins *SInstance) GetError() error {
	return nil
}

func (ins *SInstance) GetFullOsName() string {
	return ""
}

func (ins *SInstance) GetGlobalId() string {
	return ins.InstanceId
}

func (ins *SInstance) GetId() string {
	return ins.InstanceId
}

func (ins *SInstance) GetInstanceType() string {
	return ins.InstanceType
}

func (ins *SInstance) GetMachine() string {
	return "pc"
}

func (ins *SInstance) GetHostname() string {
	return ins.HostName
}

func (ins *SInstance) GetName() string {
	return ins.InstanceName
}

func (ins *SInstance) GetOsArch() string {
	return ""
}

func (ins *SInstance) GetOsDist() string {
	return ""
}

func (ins *SInstance) GetOsLang() string {
	return ""
}

func (ins *SInstance) GetOsType() cloudprovider.TOsType {
	imageInfo := imagetools.NormalizeImageInfo("", "", "", ins.Platform, "")
	return cloudprovider.TOsType(imageInfo.OsType)
}

func (ins *SInstance) GetOsVersion() string {
	return ""
}

func (ins *SInstance) GetProjectId() string {
	return ins.ProjectId
}

func (ins *SInstance) GetSecurityGroupIds() ([]string, error) {
	ids := []string{}
	for _, netSet := range ins.NetworkInterfaceSet {
		for _, secgroupSet := range netSet.SecurityGroupSet {
			ids = append(ids, secgroupSet.SecurityGroupId)
		}
	}
	return ids, nil
}

func (ins *SInstance) GetStatus() string {
	switch ins.InstanceState.Name {
	case "block_device_mapping", "scheduling":
		return api.VM_DEPLOYING
	case "active":
		return api.VM_RUNNING
	case "stopping":
		return api.VM_STOPPING
	case "stopped":
		return api.VM_READY
	}
	return ins.InstanceState.Name
}

func (ins *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_KSYUN
}

func (ins *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := ins.getRegion().GetDiskByInstanceId(ins.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "getDisks")
	}
	res := []cloudprovider.ICloudDisk{}
	storages, err := ins.host.zone.GetStorages()
	if err != nil {
		return nil, errors.Wrap(err, "GetStorages")
	}
	for i := 0; i < len(disks); i++ {
		for j := range storages {
			if disks[i].VolumeType == storages[j].StorageType {
				disks[i].storage = &storages[j]
				break
			}
		}
		if disks[i].storage == nil {
			return nil, fmt.Errorf("failed to found disk storage type %s", disks[i].VolumeType)
		}
		res = append(res, &disks[i])
	}
	return res, nil
}

func (ins *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	eipIds := []string{}
	for _, set := range ins.NetworkInterfaceSet {
		eipIds = append(eipIds, set.AllocationId)
	}
	if len(eipIds) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	eips, err := ins.getRegion().GetEips(eipIds)
	if err != nil {
		return nil, errors.Wrap(err, "get eips")
	}
	if len(eips) == 0 {
		return nil, errors.ErrNotFound
	}
	for _, eip := range eips {
		if utils.IsInStringArray(eip.GetId(), eipIds) {
			eip.region = ins.getRegion()
			return &eip, nil
		}
	}
	return nil, errors.Wrapf(err, "instanceId id:%s", ins.GetGlobalId())
}

func (ins *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics := []cloudprovider.ICloudNic{}
	for i := 0; i < len(ins.NetworkInterfaceSet); i++ {
		nic := SInstanceNic{
			Instance: ins,
			Id:       ins.NetworkInterfaceSet[i].SubnetId,
			IpAddr:   ins.NetworkInterfaceSet[i].PrivateIPAddress,
			MacAddr:  ins.NetworkInterfaceSet[i].MacAddress,
		}
		nics = append(nics, &nic)
	}
	return nics, nil
}

func (ins *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	vnc, err := ins.getRegion().GetVNCInfo(ins.InstanceId)
	if err != nil {
		return nil, errors.Wrap(err, "GetVNCInfo")
	}
	ret := &cloudprovider.ServerVncOutput{
		Url:        fmt.Sprintf("http://kec.console.ksyun.com/kec/connect/vnc?VncUrl=%s", vnc),
		Protocol:   "ksyun",
		InstanceId: ins.InstanceId,
		Hypervisor: api.HYPERVISOR_KSYUN,
	}
	return ret, nil
}

/*
{
"RequestId":
"91f6c5ea-f0b0-4d1a-9f87-6154f1823442003",
"VncUrl":
"ws://tjwqone.vnc.ksyun.com:80/websockify?token=u4-394839af-a920-4394-bfe3-f99b88c87fa1"
}
*/

func (region *SRegion) GetVNCInfo(instanceId string) (string, error) {
	resp, err := region.ecsRequest("DescribeInstanceVncUrl", map[string]string{"InstanceId": instanceId})
	if err != nil {
		return "", errors.Wrap(err, "GetVNCAddress")
	}
	return resp.GetString("VncUrl")
}

func (ins *SInstance) GetVcpuCount() int {
	return ins.InstanceConfigure.Vcpu
}

func (ins *SInstance) GetVmemSizeMB() int {
	return ins.InstanceConfigure.MemoryGb * 1024
}

func (ins *SInstance) GetVdi() string {
	return ""
}

func (ins *SInstance) GetVga() string {
	return ""
}

func (ins *SInstance) RebuildRoot(ctx context.Context, opts *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	err := ins.getRegion().RebuildRoot(ins.InstanceId, opts)
	if err != nil {
		return "", err
	}
	disks, err := ins.GetIDisks()
	if err != nil {
		return "", err
	}
	if len(disks) == 0 {
		return "", fmt.Errorf("server %s has no volume attached.", ins.GetId())
	}
	return disks[0].GetGlobalId(), nil
}

func (region *SRegion) RebuildRoot(instanceId string, opts *cloudprovider.SManagedVMRebuildRootConfig) error {
	params := map[string]string{
		"InstanceId": instanceId,
		"ImageId":    opts.ImageId,
	}
	if len(opts.Password) > 0 {
		params["InstancePassword"] = opts.Password
	}
	/*
		if len(opts.PublicKey) > 0 {
			params["KeyPairName"] = opts.PublicKey
		}
	*/
	if len(opts.UserData) > 0 {
		params["UserData"] = opts.UserData
	}
	_, err := region.ecsRequest("ModifyInstanceImage", params)
	return err
}

func (ins *SInstance) SetSecurityGroups(secgroupIds []string) error {
	nicId := ""
	for _, netSet := range ins.NetworkInterfaceSet {
		if netSet.NetworkInterfaceType == "primary" {
			nicId = netSet.NetworkInterfaceId
			break
		}
	}
	return ins.getRegion().SetSecurityGroups(secgroupIds, ins.InstanceId, nicId, ins.SubnetId)
}

func (ins *SInstance) StartVM(ctx context.Context) error {
	return ins.getRegion().StartVM(ins.InstanceId)
}

func (ins *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	return ins.getRegion().StopVM(ins.InstanceId, opts.IsForce, opts.StopCharging)
}

func (region *SRegion) StartVM(instanceId string) error {
	params := map[string]string{
		"InstanceId.1": instanceId,
	}
	_, err := region.ecsRequest("StartInstances", params)
	if err != nil {
		return errors.Wrap(err, "StartInstances")
	}
	return nil
}

func (region *SRegion) StopVM(instanceId string, force, stopCharging bool) error {
	params := map[string]string{
		"InstanceId.1": instanceId,
		"StoppedMode":  "KeepCharging",
	}
	if stopCharging {
		params["StoppedMode"] = "StopCharging"
	}
	if force {
		params["ForceStop"] = "true"
	}
	_, err := region.ecsRequest("StopInstances", params)
	if err != nil {
		return errors.Wrap(err, "StopInstances")
	}
	return nil
}

func (ins *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

func (ins *SInstance) UpdateVM(ctx context.Context, opts cloudprovider.SInstanceUpdateOptions) error {
	return ins.getRegion().UpdateVM(ins.InstanceId, opts)
}

func (region *SRegion) UpdateVM(instanceId string, opts cloudprovider.SInstanceUpdateOptions) error {
	params := map[string]string{
		"InstanceId": instanceId,
	}
	if len(opts.NAME) > 0 {
		params["InstanceName"] = opts.NAME
	}
	if len(opts.HostName) > 0 {
		params["HostName"] = opts.HostName
	}
	_, err := region.ecsRequest("ModifyInstanceAttribute", params)
	return err
}

func (ins *SInstance) GetIHost() cloudprovider.ICloudHost {
	return ins.host
}

func (ins *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	return ins.getRegion().DeployVM(ins.InstanceId, opts)
}

func (region *SRegion) DeployVM(instanceId string, opts *cloudprovider.SInstanceDeployOptions) error {
	params := map[string]string{
		"InstanceId": instanceId,
	}
	if len(opts.Password) > 0 {
		params["InstancePassword"] = opts.Password
		params["RestartMode"] = "Restart"
	}
	/*
			if len(opts.PublicKey) > 0 {
				params["KeyPairName"] = opts.PublicKey
			}
		if len(opts.UserData) > 0 {
			params["UserData"] = opts.UserData
		}
	*/
	_, err := region.ecsRequest("ModifyInstanceAttribute", params)
	return err
}

func (ins *SInstance) GetBillingType() string {
	if ins.ChargeType == "Monthly" {
		return billing_api.BILLING_TYPE_PREPAID
	}
	return billing_api.BILLING_TYPE_POSTPAID
}

func (ins *SInstance) GetCreatedAt() time.Time {
	return ins.CreationDate
}

func (ins *SInstance) GetExpiredAt() time.Time {
	return ins.ServiceEndTime
}
