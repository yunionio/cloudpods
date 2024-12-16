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

type SInstanceResp struct {
	Marker        int         `json:"Marker"`
	InstanceCount int         `json:"InstanceCount"`
	RequestID     string      `json:"RequestId"`
	InstancesSet  []SInstance `json:"InstancesSet"`
}

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
	SecurityGroupID string `json:"SecurityGroupId"`
}

type NetworkInterfaceSet struct {
	AllocationId         string                     `json:"AllocationId"`
	NetworkInterfaceID   string                     `json:"NetworkInterfaceId"`
	NetworkInterfaceType string                     `json:"NetworkInterfaceType"`
	VpcID                string                     `json:"VpcId"`
	SubnetID             string                     `json:"SubnetId"`
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
	DiskID             string `json:"DiskId"`
	DiskType           string `json:"DiskType"`
	DiskSize           int    `json:"DiskSize"`
	DeleteWithInstance bool   `json:"DeleteWithInstance"`
	Encrypted          bool   `json:"Encrypted"`
}

type SInstance struct {
	multicloud.SInstanceBase
	SKsTag
	host *SHost

	InstanceID            string                `json:"InstanceId"`
	ProjectID             string                `json:"ProjectId"`
	ShutdownNoCharge      bool                  `json:"ShutdownNoCharge"`
	IsDistributeIpv6      bool                  `json:"IsDistributeIpv6"`
	InstanceName          string                `json:"InstanceName"`
	InstanceType          string                `json:"InstanceType"`
	InstanceConfigure     InstanceConfigure     `json:"InstanceConfigure"`
	ImageID               string                `json:"ImageId"`
	SubnetID              string                `json:"SubnetId"`
	PrivateIPAddress      string                `json:"PrivateIpAddress"`
	InstanceState         InstanceState         `json:"InstanceState"`
	Monitoring            Monitoring            `json:"Monitoring"`
	NetworkInterfaceSet   []NetworkInterfaceSet `json:"NetworkInterfaceSet"`
	SriovNetSupport       string                `json:"SriovNetSupport"`
	IsShowSriovNetSupport bool                  `json:"IsShowSriovNetSupport"`
	CreationDate          time.Time             `json:"CreationDate"`
	AvailabilityZone      string                `json:"AvailabilityZone"`
	AvailabilityZoneName  string                `json:"AvailabilityZoneName"`
	DedicatedUUID         string                `json:"DedicatedUuid"`
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
		part := SInstanceResp{}
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
	for _, instance := range instances {
		if instance.GetGlobalId() == instanceId {
			return &instance, nil
		}
	}
	return nil, errors.Wrapf(err, "instance id:%s", instanceId)
}

func (ins *SInstance) Refresh() error {
	extIns, err := ins.host.zone.region.GetInstance(ins.GetGlobalId())
	if err != nil {
		return errors.Wrap(err, "GetInstance")
	}
	return jsonutils.Update(ins, extIns)
}

func (ins *SInstance) GetTags() (map[string]string, error) {
	tags, err := ins.host.zone.region.ListTags("kec-instance", ins.InstanceID)
	if err != nil {
		return nil, err
	}
	return tags.GetTags(), nil
}

func (ins *SInstance) AssignSecurityGroup(secgroupId string) error {
	return cloudprovider.ErrNotImplemented
}

func (ins *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (ins *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (ins *SInstance) DeleteVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (ins *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
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
	return ins.InstanceID
}

func (ins *SInstance) GetId() string {
	return ins.InstanceID
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
	return ins.ProjectID
}

func (ins *SInstance) GetSecurityGroupIds() ([]string, error) {
	ids := []string{}
	for _, netSet := range ins.NetworkInterfaceSet {
		for _, secgroupSet := range netSet.SecurityGroupSet {
			ids = append(ids, secgroupSet.SecurityGroupID)
		}
	}
	return ids, nil
}

func (ins *SInstance) GetStatus() string {
	switch ins.InstanceState.Name {
	case "active":
		return api.VM_RUNNING
	case "stopped":
		return api.VM_READY
	}
	return ins.InstanceState.Name
}

func (ins *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_KSYUN
}

func (ins *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := ins.host.zone.region.GetDiskByInstanceId(ins.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "getDisks")
	}
	res := []cloudprovider.ICloudDisk{}
	for i := 0; i < len(disks); i++ {
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
	eips, err := ins.host.zone.region.GetEips(eipIds)
	if err != nil {
		return nil, errors.Wrap(err, "get eips")
	}
	if len(eips) == 0 {
		return nil, errors.ErrNotFound
	}
	for _, eip := range eips {
		if utils.IsInStringArray(eip.GetId(), eipIds) {
			eip.region = ins.host.zone.region
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
			Id:       ins.NetworkInterfaceSet[i].SubnetID,
			IpAddr:   ins.NetworkInterfaceSet[i].PrivateIPAddress,
			MacAddr:  ins.NetworkInterfaceSet[i].MacAddress,
		}
		nics = append(nics, &nic)
	}
	return nics, nil
}

func (ins *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	// TODO
	resp, err := ins.host.zone.region.ecsRequest("GetVNCAddress", map[string]string{"InstanceId": ins.InstanceID})
	if err != nil {
		return nil, errors.Wrap(err, "GetVNCAddress")
	}
	temp := struct {
		VNCAddress struct {
			Port string `json:"Port"`
			Host string `json:"Host"`
		}
		Cookies []struct {
			CookieKey   string `json:"CookieKey"`
			CookieValue string `json:"CookieValue"`
		} `json:"Cookies"`
		Domain string `json:"Domain"`
	}{}
	err = resp.Unmarshal(&temp)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal vnc info")
	}
	return nil, errors.ErrNotImplemented
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

func (ins *SInstance) RebuildRoot(ctx context.Context, config *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return "", errors.ErrNotImplemented
}

func (ins *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return errors.ErrNotImplemented
}

func (ins *SInstance) StartVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (ins *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (ins *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotImplemented
}

func (ins *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (ins *SInstance) GetIHost() cloudprovider.ICloudHost {
	return ins.host
}

func (ins *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	return cloudprovider.ErrNotImplemented
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
