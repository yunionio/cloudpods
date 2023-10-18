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
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/cloudinit"
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

type TChargeType string

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
	InstanceChargeType TChargeType
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
	userData, err := body.GetString("Result", "UserData")
	if err != nil {
		return "", errors.Wrapf(err, "GetUserData")
	}
	return userData, nil
}

func (region *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	instances, _, err := region.GetInstances("", []string{instanceId}, 1, "")
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return &instances[0], nil
}

func (region *SRegion) GetInstances(zoneId string, ids []string, limit int, token string) ([]SInstance, string, error) {
	if limit > 10 || limit <= 0 {
		limit = 10
	}
	params := make(map[string]string)
	params["MaxResults"] = fmt.Sprintf("%d", limit)
	if len(token) > 0 {
		params["NextToken"] = token
	}
	if len(zoneId) > 0 {
		params["ZoneId"] = zoneId
	}
	if len(ids) > 0 {
		for index, id := range ids {
			key := fmt.Sprintf("InstanceIds.%d", index+1)
			params[key] = id
		}
	}
	body, err := region.ecsRequest("DescribeInstances", params)
	if err != nil {
		return nil, "", errors.Wrapf(err, "GetInstances fail")
	}
	instances := make([]SInstance, 0)
	err = body.Unmarshal(&instances, "Result", "Instances")
	if err != nil {
		return nil, "", errors.Wrapf(err, "Unmarshal details fail")
	}
	nextToken, _ := body.GetString("Result", "NextToken")
	return instances, nextToken, nil
}

func (instance *SInstance) GetIHost() cloudprovider.ICloudHost {
	return instance.host
}

func (instance *SInstance) GetIHostId() string {
	return instance.host.GetGlobalId()
}

func (instance *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	pageNumber := 1
	disks := make([]SDisk, 0)
	for {
		parts, total, err := instance.host.zone.region.GetDisks(instance.InstanceId, "", "", nil, pageNumber, 50)
		if err != nil {
			return nil, err
		}
		disks = append(disks, parts...)
		if len(disks) >= total {
			break
		}
		pageNumber += 1
	}

	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i += 1 {
		store, err := instance.host.zone.getStorageByCategory(disks[i].VolumeType)
		if err != nil {
			return nil, errors.Wrap(err, "getStorageByCategory")
		}
		disks[i].storage = store
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
	networkInterfaces := instance.NetworkInterfaces
	nics := make([]cloudprovider.ICloudNic, 0)
	for _, ni := range networkInterfaces {
		nic := SInstanceNic{
			instance: instance,
			id:       ni.NetworkInterfaceId,
			ipAddr:   ni.PrimaryIpAddress,
			macAddr:  ni.MacAddress,
		}
		nics = append(nics, &nic)
	}
	return nics, nil
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
	ret := []string{}
	for _, net := range instance.NetworkInterfaces {
		ret = append(ret, net.SecurityGroupIds...)
	}
	return ret, nil
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
	// return instance.ExpiredAt
	return time.Time{}
}

func (instance *SInstance) AssignSecurityGroup(secgroupId string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "AssignSecurityGroup")
}

func (instance *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "SetSecurityGroups")
}

func (instance *SInstance) GetError() error {
	return nil
}

func (instance *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	if config.InstanceType == "nil" {
		return errors.Wrapf(cloudprovider.ErrInputParameter, "InstanceType")
	}
	return instance.host.zone.region.ChangeConfig(instance.InstanceId, config.InstanceType)
}

func (instance *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return nil, cloudprovider.ErrNotSupported
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
	for {
		err := instance.host.zone.region.DeleteVM(instance.InstanceId)
		if err != nil {
			if isError(err, "IncorrectInstanceStatus.Initializing") {
				log.Infof("The instance is initializing, try later ...")
				time.Sleep(10 * time.Second)
			} else {
				return errors.Wrapf(err, "DeleteVM fail")
			}
		} else {
			break
		}
	}
	return cloudprovider.WaitDeleted(instance, 10*time.Second, 300*time.Second)
}

func (instance *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	return instance.host.zone.region.UpdateVM(instance.InstanceId, input.NAME, input.Description)
}

func (instance *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	var keypairName string
	if len(publicKey) > 0 {
		var err error
		keypairName, err = instance.host.zone.region.syncKeypair(publicKey)
		if err != nil {
			return err
		}
	}

	return instance.host.zone.region.DeployVM(instance.InstanceId, name, password, keypairName, deleteKeypair, description)
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
	udata, err := instance.GetUserData()
	if err != nil {
		return "", err
	}

	image, err := instance.host.zone.region.GetImage(desc.ImageId)
	if err != nil {
		return "", errors.Wrapf(err, "GetImage fail")
	}

	keypairName := instance.KeyPairName
	if len(desc.PublicKey) > 0 {
		keypairName, err = instance.host.zone.region.syncKeypair(desc.PublicKey)
		if err != nil {
			return "", fmt.Errorf("RebuildRoot.syncKeypair %s", err)
		}
	}

	userdata := ""
	srcOsType := strings.ToLower(string(instance.GetOsType()))
	destOsType := strings.ToLower(string(image.GetOsType()))
	winOS := strings.ToLower(osprofile.OS_TYPE_WINDOWS)

	cloudconfig := &cloudinit.SCloudConfig{}
	if srcOsType != winOS && len(udata) > 0 {
		_cloudconfig, err := cloudinit.ParseUserDataBase64(udata)
		if err != nil {
			log.Debugf("RebuildRoot invalid instance user data %s", udata)
		} else {
			cloudconfig = _cloudconfig
		}
	}

	if (srcOsType != winOS && destOsType != winOS) || (srcOsType == winOS && destOsType != winOS) {
		// linux/windows to linux
		loginUser := cloudinit.NewUser(api.VM_AWS_DEFAULT_LOGIN_USER)
		loginUser.SudoPolicy(cloudinit.USER_SUDO_NOPASSWD)
		if len(desc.PublicKey) > 0 {
			loginUser.SshKey(desc.PublicKey)
			cloudconfig.MergeUser(loginUser)
		} else if len(desc.Password) > 0 {
			cloudconfig.SshPwauth = cloudinit.SSH_PASSWORD_AUTH_ON
			loginUser.Password(desc.Password)
			cloudconfig.MergeUser(loginUser)
		}

		userdata = cloudconfig.UserDataBase64()
	} else {
		// linux/windows to windows
		data := ""
		if len(desc.Password) > 0 {
			cloudconfig.SshPwauth = cloudinit.SSH_PASSWORD_AUTH_ON
			loginUser := cloudinit.NewUser(api.VM_AWS_DEFAULT_WINDOWS_LOGIN_USER)
			loginUser.SudoPolicy(cloudinit.USER_SUDO_NOPASSWD)
			loginUser.Password(desc.Password)
			cloudconfig.MergeUser(loginUser)
			data = fmt.Sprintf("<powershell>%s</powershell>", cloudconfig.UserDataPowerShell())
		} else {
			if len(udata) > 0 {
				data = fmt.Sprintf("<powershell>%s</powershell>", udata)
			}
		}

		userdata = base64.StdEncoding.EncodeToString([]byte(data))
	}

	diskId, err := instance.host.zone.region.ReplaceSystemDisk(ctx, instance.InstanceId, desc.ImageId, desc.Password, keypairName, userdata)
	if err != nil {
		return "", err
	}

	return diskId, nil
}

func (instance *SInstance) SaveImage(opts *cloudprovider.SaveImageOptions) (cloudprovider.ICloudImage, error) {
	image, err := instance.host.zone.region.SaveImage(instance.InstanceId, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "SaveImage %s", opts.Name)
	}
	return image, nil
}

// region
func (region *SRegion) CreateInstance(
	name string,
	hostname string,
	imageId string,
	instanceType string,
	securityGroupId string,
	zoneId string,
	desc string,
	passwd string,
	disks []SDisk,
	networkID string,
	ipAddr string,
	keypair string,
	userData string,
	bc *billing.SBillingCycle,
	projectId string,
	tags map[string]string,
) (string, error) {
	params := make(map[string]string)
	params["RegionId"] = region.RegionId
	params["ImageId"] = imageId
	params["InstanceType"] = instanceType
	params["ZoneId"] = zoneId
	params["InstanceName"] = name
	params["ProjectName"] = projectId
	if len(hostname) > 0 {
		params["HostName"] = hostname
	}
	params["Description"] = desc
	if len(passwd) > 0 {
		params["Password"] = passwd
	} else {
		params["KeepImageCredential"] = "True"
	}
	if len(keypair) > 0 {
		params["KeyPairName"] = keypair
	}

	if len(userData) > 0 {
		params["UserData"] = userData
	}

	if len(tags) > 0 {
		tagIdx := 1
		for k, v := range tags {
			params[fmt.Sprintf("Tag.%d.Key", tagIdx)] = k
			params[fmt.Sprintf("Tag.%d.Value", tagIdx)] = v
			tagIdx += 1
		}
	}

	if len(disks) > 0 {
		for idx, disk := range disks {
			diskIdx := idx + 1
			params[fmt.Sprintf("Volumes.%d.Size", diskIdx)] = fmt.Sprintf("%d", disk.Size)
			params[fmt.Sprintf("Volumes.%d.VolumeType", diskIdx)] = disk.VolumeType
		}
	}

	params["NetworkInterfaces.1.SubnetId"] = ipAddr
	// currently only support binding the first NetworkInterface securitygroup
	params["NetworkInterfaces.1.SecurityGroupIds.1"] = securityGroupId

	if bc != nil {
		params["InstanceChargeType"] = "PrePaid"
		err := billingCycle2Params(bc, params)
		if err != nil {
			return "", err
		}
		if bc.AutoRenew {
			params["AutoRenew"] = "true"
			params["AutoRenewPeriod"] = "1"
		} else {
			params["AutoRenew"] = "False"
		}
	} else {
		params["InstanceChargeType"] = "PostPaid"
		params["SpotStrategy"] = "NoSpot"
	}

	params["ClientToken"] = utils.GenRequestId(20)

	body, err := region.ecsRequest("CreateInstance", params)
	if err != nil {
		return "", errors.Wrapf(err, "CreateInstance fail")
	}
	instanceId, _ := body.GetString("InstanceId")
	return instanceId, nil
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
	status, err := region.GetInstanceStatus(instanceId)
	if err != nil {
		return errors.Wrapf(err, "Fail to get instance status on DeleteVM")
	}
	log.Debugf("Instance status on delete is %s", status)
	if status != InstanceStatusStopped {
		log.Warningf("DeleteVM: vm status is %s expect %s", status, InstanceStatusStopped)
	}
	return region.doDeleteVM(instanceId)
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

func (region *SRegion) doDeleteVM(instanceId string) error {
	return region.instanceOperation(instanceId, "DeleteInstance", nil)
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

func (region *SRegion) DeployVM(instanceId string, name string, password string, keypairName string, deleteKeypair bool, description string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}

	if deleteKeypair {
		err = region.DetachKeyPair(instanceId, instance.KeyPairName)
		if err != nil {
			return err
		}
	}

	if len(keypairName) > 0 {
		err = region.AttachKeypair(instanceId, keypairName)
		if err != nil {
			return err
		}
	}

	params := make(map[string]string)

	if len(password) > 0 {
		params["Password"] = password
	}

	if len(name) > 0 && instance.InstanceName != name {
		params["InstanceName"] = name
	}

	if len(description) > 0 && instance.Description != description {
		params["Description"] = description
	}

	if len(params) > 0 {
		return region.modifyInstanceAttribute(instanceId, params)
	} else {
		return nil
	}
}

func (region *SRegion) DetachDisk(instanceId string, diskId string) error {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	params["VolumeId"] = diskId
	log.Infof("Detach instance %s disk %s", instanceId, diskId)
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

func (region *SRegion) ReplaceSystemDisk(ctx context.Context, instanceId string, imageId string, passwd string, keypairName string, userdata string) (string, error) {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	params["ImageId"] = imageId
	if len(passwd) > 0 {
		params["Password"] = passwd
	} else {
		params["KeepImageCredential"] = "True"
	}
	if len(keypairName) > 0 {
		params["KeyPairName"] = keypairName
	}
	_, err := region.ecsRequest("ReplaceSystemVolume", params)
	if err != nil {
		return "", err
	}
	// volcengine does not return volumeId
	return "", nil
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
	imageId, err := body.GetString("Result", "IamgeId")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	image, err := region.GetImage(imageId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetImage %s", imageId)
	}
	return image, nil
}
