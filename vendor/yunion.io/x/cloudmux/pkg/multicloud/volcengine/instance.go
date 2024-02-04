// Copyright 2023 Yunion
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

package volcengine

import (
	"context"
	"fmt"
	"strings"
	"time"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/imagetools"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/utils"
)

const (
	InstanceStatusCreating   = "CREATING"
	InstanceStatusRunning    = "RUNNING"
	InstanceStatusStopping   = "STOPPING"
	InstanceStatusStopped    = "STOPPED"
	InstanceStatusRebooting  = "REBOOTING"
	InstanceStatusStarting   = "STARTING"
	InstanceStatusRebuilding = "REBUILDING"
	InstanceStatusResizing   = "RESIZING"
	InstanceStatusError      = "ERROR"
	InstanceStatusDeleting   = "DELETING"
)

type SSecurityGroupIds []string

type SRdmaIPAddress []string

type SInstance struct {
	multicloud.SInstanceBase
	VolcEngineTags

	host *SHost

	osInfo *imagetools.ImageInfo

	CreatedAt          time.Time
	UpdatedAt          time.Time
	InstanceId         string
	ZoneId             string
	ImageId            string
	Status             string
	InstanceName       string
	Description        string
	Hostname           string
	VpcId              string
	InstanceTypeId     string
	Cpus               int
	MemorySize         int
	OsName             string
	OsType             string
	NetworkInterfaces  []SNetworkInterface
	RdmaIpAddress      SRdmaIPAddress
	KeyPairName        string
	KeyPairId          string
	InstanceChargeType string
	StoppedMode        string
	SpotStrategy       string
	DeploymentSetId    string
	EipAddress         SEipAddress
	ExpiredAt          time.Time
	Uuid               string
	ProjectName        string
}

func billingCycle2Params(bc *billing.SBillingCycle, params map[string]string) error {
	if bc.GetMonths() > 0 {
		params["PeriodUnit"] = "Month"
		params["Period"] = fmt.Sprintf("%d", bc.GetMonths())
	} else if bc.GetWeeks() > 0 {
		params["PeriodUnit"] = "Week"
		params["Period"] = fmt.Sprintf("%d", bc.GetWeeks())
		// renew by week is not currently supported
		return fmt.Errorf("invalid renew time period %s", bc.String())
	} else {
		return fmt.Errorf("invalid renew time period %s", bc.String())
	}
	return nil
}

func (instance *SInstance) UpdatePassword(passwd string) error {
	params := make(map[string]string)
	params["Password"] = passwd
	return instance.host.zone.region.modifyInstanceAttribute(instance.InstanceId, params)
}

func (instance *SInstance) UpdateUserData(userData string) error {
	params := make(map[string]string)
	params["UserData"] = userData
	return instance.host.zone.region.modifyInstanceAttribute(instance.InstanceId, params)
}

func (instance *SInstance) GetUserData() (string, error) {
	params := make(map[string]string)
	params["InstanceId"] = instance.InstanceId
	body, err := instance.host.zone.region.ecsRequest("DescribeUserData", params)
	if err != nil {
		return "", errors.Wrapf(err, "GetUserData")
	}
	userData, err := body.GetString("UserData")
	if err != nil {
		return "", errors.Wrapf(err, "GetUserData")
	}
	return userData, nil
}

func (region *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	instances, err := region.GetInstances("", []string{instanceId})
	if err != nil {
		return nil, err
	}
	for i := range instances {
		if instances[i].InstanceId == instanceId {
			return &instances[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, instanceId)
}

func (region *SRegion) GetInstances(zoneId string, ids []string) ([]SInstance, error) {
	params := make(map[string]string)
	params["MaxResults"] = "100"
	if len(zoneId) > 0 {
		params["ZoneId"] = zoneId
	}
	for index, id := range ids {
		key := fmt.Sprintf("InstanceIds.%d", index+1)
		params[key] = id
	}
	ret := []SInstance{}
	for {
		resp, err := region.ecsRequest("DescribeInstances", params)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeInstances")
		}
		part := struct {
			Instances []SInstance
			NextToken string
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Instances...)
		if len(part.NextToken) == 0 || len(part.Instances) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
}

func (instance *SInstance) GetIHost() cloudprovider.ICloudHost {
	return instance.host
}

func (instance *SInstance) GetIHostId() string {
	return instance.host.GetGlobalId()
}

func (instance *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := instance.host.zone.region.GetDisks(instance.InstanceId, "", "", nil)
	if err != nil {
		return nil, err
	}

	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i += 1 {
		storage := &SStorage{zone: instance.host.zone, storageType: disks[i].VolumeType}
		disks[i].storage = storage
		idisks[i] = &disks[i]
	}
	return idisks, nil
}

func (instance *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	if len(instance.EipAddress.AllocationId) > 0 {
		return instance.host.zone.region.GetEip(instance.EipAddress.AllocationId)
	}
	for _, nic := range instance.NetworkInterfaces {
		if len(nic.AssociatedElasticIp.EipAddress) > 0 {
			eip := SEipAddress{region: instance.host.zone.region}
			eip.region = instance.host.zone.region
			eip.EipAddress = nic.AssociatedElasticIp.EipAddress
			eip.InstanceId = instance.InstanceId
			eip.AllocationId = instance.InstanceId
			return &eip, nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (instance *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	ret := []cloudprovider.ICloudNic{}
	nics, err := instance.host.zone.region.GetNetworkInterfaces("", instance.InstanceId)
	if err != nil {
		return nil, err
	}
	for i := range nics {
		nics[i].region = instance.host.zone.region
		ret = append(ret, &nics[i])
	}
	return ret, nil
}

func (instance *SInstance) GetId() string {
	return instance.InstanceId
}

func (instance *SInstance) GetName() string {
	if len(instance.InstanceName) > 0 {
		return instance.InstanceName
	}
	return instance.InstanceId
}

func (instance *SInstance) GetHostname() string {
	return instance.Hostname
}

func (instance *SInstance) GetGlobalId() string {
	return instance.InstanceId
}

func (instance *SInstance) GetInstanceType() string {
	return instance.InstanceTypeId
}

func (instance *SInstance) GetSecurityGroupIds() ([]string, error) {
	nics, err := instance.host.zone.region.GetNetworkInterfaces("", instance.InstanceId)
	if err != nil {
		return nil, err
	}
	for _, nic := range nics {
		if len(nic.SecurityGroupIds) > 0 {
			return nic.SecurityGroupIds, nil
		}
	}
	return []string{}, nil
}

func (instance *SInstance) GetVcpuCount() int {
	return instance.Cpus
}

func (instance *SInstance) GetVmemSizeMB() int {
	return instance.MemorySize
}

func (instance *SInstance) GetBootOrder() string {
	return "dcn"
}

func (instance *SInstance) GetVga() string {
	return "std"
}

func (instance *SInstance) GetVdi() string {
	return "vnc"
}

func (ins *SInstance) getNormalizedOsInfo() *imagetools.ImageInfo {
	if ins.osInfo == nil {
		osInfo := imagetools.NormalizeImageInfo(ins.OsName, "", ins.OsType, "", "")
		ins.osInfo = &osInfo
	}
	return ins.osInfo
}

func (instance *SInstance) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(osprofile.NormalizeOSType(instance.OsType))
}

func (instance *SInstance) GetFullOsName() string {
	return instance.OsName
}

func (ins *SInstance) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(ins.getNormalizedOsInfo().OsBios)
}

func (ins *SInstance) GetOsArch() string {
	return ins.getNormalizedOsInfo().OsArch
}

func (ins *SInstance) GetOsDist() string {
	return ins.getNormalizedOsInfo().OsDistro
}

func (ins *SInstance) GetOsVersion() string {
	return ins.getNormalizedOsInfo().OsVersion
}

func (ins *SInstance) GetOsLang() string {
	return ins.getNormalizedOsInfo().OsLang
}

func (instance *SInstance) GetMachine() string {
	return "pc"
}

func (instance *SInstance) GetStatus() string {
	switch instance.Status {
	case InstanceStatusRunning:
		return api.VM_RUNNING
	case InstanceStatusStarting:
		return api.VM_STARTING
	case InstanceStatusStopping:
		return api.VM_STOPPING
	case InstanceStatusStopped:
		return api.VM_READY
	case InstanceStatusDeleting:
		return api.VM_DELETING
	default:
		return api.VM_UNKNOWN
	}
}

func (instance *SInstance) Refresh() error {
	ins, err := instance.host.zone.region.GetInstance(instance.InstanceId)
	if err != nil {
		return err
	}
	return jsonutils.Update(instance, ins)
}

func (instance *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_VOLCENGINE
}

func (instance *SInstance) GetCreatedAt() time.Time {
	return instance.CreatedAt
}

func (instance *SInstance) GetExpiredAt() time.Time {
	if instance.InstanceChargeType != "PostPaid" {
		return instance.ExpiredAt
	}
	return time.Time{}
}

func (instance *SInstance) GetBillingType() string {
	if instance.InstanceChargeType == "PostPaid" {
		return billing_api.BILLING_TYPE_POSTPAID
	}
	return billing_api.BILLING_TYPE_PREPAID
}

func (instance *SInstance) SetSecurityGroups(secgroupIds []string) error {
	for _, nic := range instance.NetworkInterfaces {
		return instance.host.zone.region.ModifyNetworkInterfaceAttributes(nic.NetworkInterfaceId, secgroupIds)
	}
	return nil
}

func (self *SRegion) ModifyNetworkInterfaceAttributes(id string, secgroupIds []string) error {
	params := map[string]string{
		"NetworkInterfaceId": id,
	}
	for i, id := range secgroupIds {
		params[fmt.Sprintf("SecurityGroupIds.%d", i+1)] = id
	}
	_, err := self.vpcRequest("ModifyNetworkInterfaceAttributes", params)
	return err
}

func (instance *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	if config.InstanceType == "nil" {
		return errors.Wrapf(cloudprovider.ErrInputParameter, "InstanceType")
	}
	return instance.host.zone.region.ChangeConfig(instance.InstanceId, config.InstanceType)
}

func (instance *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	url, err := instance.host.zone.region.DescribeInstanceVncUrl(instance.InstanceId)
	if err != nil {
		return nil, err
	}
	protocol := api.HYPERVISOR_VOLCENGINE
	ret := &cloudprovider.ServerVncOutput{
		Url:          strings.TrimPrefix(url, "wss://"),
		Protocol:     protocol,
		InstanceId:   instance.InstanceId,
		Region:       instance.host.zone.region.RegionId,
		InstanceName: instance.InstanceName,
		Hypervisor:   api.HYPERVISOR_VOLCENGINE,
	}
	return ret, nil
}

func (self *SRegion) DescribeInstanceVncUrl(id string) (string, error) {
	params := map[string]string{
		"InstanceId": id,
	}
	resp, err := self.ecsRequest("DescribeInstanceVncUrl", params)
	if err != nil {
		return "", err
	}
	return resp.GetString("VncUrl")
}

func (instance *SInstance) StartVM(ctx context.Context) error {
	err := instance.host.zone.region.StartVM(instance.InstanceId)
	return err
}

func (instance *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	err := instance.host.zone.region.StopVM(instance.InstanceId, opts.IsForce, opts.StopCharging)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(instance, api.VM_READY, 10*time.Second, 300*time.Second)
}

func (instance *SInstance) DeleteVM(ctx context.Context) error {
	return instance.host.zone.region.DeleteVM(instance.InstanceId)
}

func (instance *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	return instance.host.zone.region.UpdateVM(instance.InstanceId, input.NAME, input.Description)
}

func (instance *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	return instance.host.zone.region.DeployVM(instance.InstanceId, opts)
}

func (instance *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return instance.host.zone.region.AttachDisk(instance.InstanceId, diskId)
}

func (instance *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.RetryOnError(
		func() error {
			return instance.host.zone.region.DetachDisk(instance.InstanceId, diskId)
		},
		[]string{
			`"Code":"InvalidOperation.Conflict"`,
		},
		4)
}

func (instance *SInstance) GetProjectId() string {
	return instance.ProjectName
}

func (instance *SInstance) RebuildRoot(ctx context.Context, desc *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (instance *SInstance) SaveImage(opts *cloudprovider.SaveImageOptions) (cloudprovider.ICloudImage, error) {
	image, err := instance.host.zone.region.SaveImage(instance.InstanceId, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "SaveImage %s", opts.Name)
	}
	return image, nil
}

// region
func (region *SRegion) CreateInstance(zoneId string, opts *cloudprovider.SManagedVMCreateConfig) (string, error) {
	params := make(map[string]string)
	params["RegionId"] = region.RegionId
	params["ImageId"] = opts.ExternalImageId
	params["InstanceType"] = opts.InstanceType
	params["ZoneId"] = zoneId
	params["InstanceName"] = opts.Name
	if len(opts.ProjectId) > 0 {
		params["ProjectName"] = opts.ProjectId
	}
	if len(opts.Hostname) > 0 {
		params["HostName"] = opts.Hostname
	}
	params["Description"] = opts.Description
	if len(opts.Password) > 0 {
		params["Password"] = opts.Password
	}
	if len(opts.KeypairName) > 0 {
		params["KeyPairName"] = opts.KeypairName
	}
	if len(opts.Password) == 0 && len(opts.KeypairName) == 0 {
		params["KeepImageCredential"] = "True"
	}

	if len(opts.UserData) > 0 {
		params["UserData"] = opts.UserData
	}

	tagIdx := 1
	for k, v := range opts.Tags {
		params[fmt.Sprintf("Tags.%d.Key", tagIdx)] = k
		params[fmt.Sprintf("Tags.%d.Value", tagIdx)] = v
		tagIdx += 1
	}

	params["Volumes.1.Size"] = fmt.Sprintf("%d", opts.SysDisk.SizeGB)
	params["Volumes.1.VolumeType"] = opts.SysDisk.StorageType

	for idx, disk := range opts.DataDisks {
		params[fmt.Sprintf("Volumes.%d.Size", idx+2)] = fmt.Sprintf("%d", disk.SizeGB)
		params[fmt.Sprintf("Volumes.%d.VolumeType", idx+2)] = disk.StorageType
	}

	params["NetworkInterfaces.1.SubnetId"] = opts.ExternalNetworkId
	if len(opts.IpAddr) > 0 {
		//params["NetworkInterfaces.1.IpAddr"] = opts.IpAddr
	}
	for idx, id := range opts.ExternalSecgroupIds {
		params[fmt.Sprintf("NetworkInterfaces.1.SecurityGroupIds.%d", idx+1)] = id
	}

	params["InstanceChargeType"] = "PostPaid"
	params["SpotStrategy"] = "NoSpot"
	if opts.BillingCycle != nil {
		params["InstanceChargeType"] = "PrePaid"
		err := billingCycle2Params(opts.BillingCycle, params)
		if err != nil {
			return "", err
		}
		params["AutoRenew"] = "False"
		if opts.BillingCycle.AutoRenew {
			params["AutoRenew"] = "true"
			params["AutoRenewPeriod"] = "1"
		}
	}

	params["ClientToken"] = utils.GenRequestId(20)

	resp, err := region.ecsRequest("RunInstances", params)
	if err != nil {
		return "", errors.Wrapf(err, "RunInstances")
	}
	ids := []string{}
	err = resp.Unmarshal(&ids, "InstanceIds")
	if err != nil {
		return "", err
	}
	for _, id := range ids {
		return id, nil
	}
	return "", errors.Wrapf(cloudprovider.ErrNotFound, "after created")
}

func (region *SRegion) RenewInstance(instanceId string, bc billing.SBillingCycle) error {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	err := billingCycle2Params(&bc, params)
	if err != nil {
		return err
	}
	params["ClientToken"] = utils.GenRequestId(20)
	_, err = region.ecsRequest("RenewInstance", params)
	if err != nil {
		return errors.Wrapf(err, "RenewInstance fail")
	}
	return nil
}

func (region *SRegion) ChangeConfig(instanceId string, instanceTypeId string) error {
	params := make(map[string]string)
	params["InstanceTypeId"] = instanceTypeId
	return region.instanceOperation(instanceId, "ModifyInstanceSpec", params)
}

func (region *SRegion) StartVM(instanceId string) error {
	status, err := region.GetInstanceStatus(instanceId)
	if err != nil {
		return errors.Wrapf(err, "Fail to get instance status on StartVM")
	}
	if status != InstanceStatusStopped {
		return errors.Wrapf(cloudprovider.ErrInvalidStatus, "StartVM: vm status is %s expect %s", status, InstanceStatusStopped)
	}
	return region.doStartVM(instanceId)
}

func (region *SRegion) StopVM(instanceId string, isForce, stopCharging bool) error {
	status, err := region.GetInstanceStatus(instanceId)
	if err != nil {
		return errors.Wrapf(err, "Fail to get instance status on StopVM")
	}
	if status == InstanceStatusStopped {
		return nil
	}
	if status != InstanceStatusRunning {
		return errors.Wrapf(cloudprovider.ErrInvalidStatus, "StartVM: vm status is %s expect %s", status, InstanceStatusRunning)
	}
	return region.doStopVM(instanceId, isForce, stopCharging)
}

func (region *SRegion) DeleteVM(instanceId string) error {
	return region.instanceOperation(instanceId, "DeleteInstance", nil)
}

func (region *SRegion) doStartVM(instanceId string) error {
	return region.instanceOperation(instanceId, "StartInstance", nil)
}

func (region *SRegion) doStopVM(instanceId string, isForce, stopCharging bool) error {
	params := make(map[string]string)
	if isForce {
		params["ForceStop"] = "true"
	} else {
		params["ForceStop"] = "false"
	}
	params["StoppedMode"] = "KeepCharging"
	if stopCharging {
		params["StoppedMode"] = "StopCharging"
	}
	return region.instanceOperation(instanceId, "StopInstance", params)
}

func (region *SRegion) modifyInstanceAttribute(instanceId string, params map[string]string) error {
	return region.instanceOperation(instanceId, "ModifyInstanceAttribute", params)
}

func (region *SRegion) UpdateVM(instanceId string, name, description string) error {
	params := make(map[string]string)
	params["InstanceName"] = name
	params["Description"] = description
	return region.modifyInstanceAttribute(instanceId, params)
}

func (region *SRegion) DeployVM(instanceId string, opts *cloudprovider.SInstanceDeployOptions) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}

	if opts.DeleteKeypair {
		err = region.DetachKeyPair(instanceId, instance.KeyPairName)
		if err != nil {
			return err
		}
	}

	if len(opts.PublicKey) > 0 {
		keypairName, err := instance.host.zone.region.syncKeypair(opts.PublicKey)
		if err != nil {
			return err
		}
		err = region.AttachKeypair(instanceId, keypairName)
		if err != nil {
			return err
		}
	}

	if len(opts.Password) > 0 {
		params := make(map[string]string)
		params["Password"] = opts.Password
		return region.modifyInstanceAttribute(instanceId, params)
	}

	return nil
}

func (region *SRegion) DetachDisk(instanceId string, diskId string) error {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	params["VolumeId"] = diskId
	_, err := region.storageRequest("DetachVolume", params)
	if err != nil {
		return errors.Wrap(err, "DetachDisk")
	}
	return nil
}

func (region *SRegion) AttachDisk(instanceId string, diskId string) error {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	params["VolumeId"] = diskId
	_, err := region.storageRequest("AttachVolume", params)
	if err != nil {
		return errors.Wrapf(err, "AttachDisk %s to %s fail", diskId, instanceId)
	}
	return nil
}

func (region *SRegion) SaveImage(instanceId string, opts *cloudprovider.SaveImageOptions) (*SImage, error) {
	params := map[string]string{
		"InstanceId":  instanceId,
		"ImageName":   opts.Name,
		"Description": opts.Notes,
		"ClientToken": utils.GenRequestId(20),
	}
	body, err := region.ecsRequest("CreateImage", params)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateImage")
	}
	imageId, err := body.GetString("ImageId")
	if err != nil {
		return nil, errors.Wrapf(err, "get imageId")
	}
	cloudprovider.Wait(time.Second*3, time.Minute, func() (bool, error) {
		_, err := region.GetImage(imageId)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	return region.GetImage(imageId)
}
