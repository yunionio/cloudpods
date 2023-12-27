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

package ctyun

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SInstance struct {
	multicloud.SInstanceBase
	CtyunTags
	multicloud.SBillingBase

	host  *SHost
	image *SImage

	AzName         string
	ExpiredTime    string
	CreatedTime    time.Time
	ProjectId      string
	AttachedVolume []string
	InstanceId     string
	DisplayName    string
	InstanceName   string
	OsType         int
	InstanceStatus string
	OnDemand       bool
	KeypairName    string
	Addresses      []struct {
		VpcName     string
		AddressList []struct {
			Addr    string
			Version int
			Type    string
		}
	}
	SecGroupList []struct {
		SecurityGroupName string
		SecurityGroupId   string
	}
	VipInfoList   []interface{}
	AffinityGroup string
	Image         struct {
		ImageId   string
		ImageName string
	}
	Flavor struct {
		FlavorId     string
		FlavorName   string
		FlavorCPU    int
		FlavorRAM    int
		GpuType      string
		GpuCount     string
		GpuVendor    string
		VideoMemSize string
	}
	ResourceId      string
	UpdatedTime     time.Time
	AvailableDay    int
	ZabbixName      string
	PrivateIP       string
	PrivateIPv6     string
	VipCount        int
	VpcId           string
	VpcName         string
	SubnetIDList    []string
	FixedIPList     []string
	FloatingIP      string
	NetworkCardList []SInstanceNic
}

func (self *SInstance) GetBillingType() string {
	if len(self.ExpiredTime) > 0 {
		return billing_api.BILLING_TYPE_PREPAID
	}
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SInstance) GetCreatedAt() time.Time {
	return self.CreatedTime
}

func (self *SInstance) GetExpiredAt() time.Time {
	if len(self.ExpiredTime) > 0 {
		expire, _ := strconv.Atoi(self.ExpiredTime)
		return time.Unix(int64(expire/1000), 0)
	}
	return time.Time{}
}

func (self *SInstance) GetId() string {
	return self.InstanceId
}

func (self *SInstance) GetName() string {
	return self.DisplayName
}

func (self *SInstance) GetHostname() string {
	return self.InstanceName
}

func (self *SInstance) GetGlobalId() string {
	return self.GetId()
}

func (self *SInstance) GetStatus() string {
	switch self.InstanceStatus {
	case "backingup":
		return api.VM_BACKUP_CREATING
	case "creating", "master_order_creating":
		return api.VM_DEPLOYING
	case "expired", "freezing", "stopped":
		return api.VM_READY
	case "stopping":
		return api.VM_STOPPING
	case "rebuild":
		return api.VM_REBUILD_ROOT
	case "restarting", "starting":
		return api.VM_STARTING
	case "running":
		return api.VM_RUNNING
	default:
		return api.VM_UNKNOWN
	}
}

func (self *SInstance) Refresh() error {
	vm, err := self.host.zone.region.GetInstance(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, vm)
}

func (self *SInstance) GetProjectId() string {
	return self.ProjectId
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	storages, err := self.host.zone.GetStorages()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDisk{}
	for _, diskId := range self.AttachedVolume {
		disk, err := self.host.zone.region.GetDisk(diskId)
		if err != nil {
			return nil, errors.Wrapf(err, "GetDisk %s", diskId)
		}
		find := false
		for i := range storages {
			if disk.DiskType == storages[i].storageType {
				disk.storage = &storages[i]
				find = true
				break
			}
		}
		if !find {
			return nil, fmt.Errorf("failed to found disk storage type %s", disk.DiskType)
		}
		ret = append(ret, disk)
	}
	return ret, nil
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	ret := []cloudprovider.ICloudNic{}
	for i := range self.NetworkCardList {
		self.NetworkCardList[i].instance = self
		ret = append(ret, &self.NetworkCardList[i])
	}
	return ret, nil
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	if len(self.FloatingIP) == 0 {
		return nil, nil
	}
	eips, err := self.host.zone.region.GetEips("ACTIVE")
	if err != nil {
		return nil, err
	}
	for i := range eips {
		if eips[i].EipAddress == self.FloatingIP {
			eips[i].region = self.host.zone.region
			return &eips[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, self.FloatingIP)
}

func (self *SInstance) GetVcpuCount() int {
	return self.Flavor.FlavorCPU
}

func (self *SInstance) GetVmemSizeMB() int {
	return self.Flavor.FlavorRAM
}

func (self *SInstance) GetBootOrder() string {
	return "dcn"
}

func (self *SInstance) GetVga() string {
	return "std"
}

func (self *SInstance) GetVdi() string {
	return "vnc"
}

func (self *SInstance) GetImage() (*SImage, error) {
	if self.image != nil {
		return self.image, nil
	}

	var err error
	self.image, err = self.host.zone.region.GetImage(self.Image.ImageId)
	return self.image, err
}

func (self *SInstance) GetOsType() cloudprovider.TOsType {
	image, err := self.GetImage()
	if err != nil {
		return cloudprovider.OsTypeLinux
	}
	return image.GetOsType()
}

func (self *SInstance) GetFullOsName() string {
	image, err := self.GetImage()
	if err != nil {
		return self.image.ImageName
	}
	return image.GetFullOsName()
}

func (self *SInstance) GetBios() cloudprovider.TBiosType {
	image, err := self.GetImage()
	if err != nil {
		return cloudprovider.BIOS
	}
	return image.GetBios()
}

func (self *SInstance) GetOsArch() string {
	image, err := self.GetImage()
	if err != nil {
		return ""
	}
	return image.GetOsArch()
}

func (self *SInstance) GetOsDist() string {
	image, err := self.GetImage()
	if err != nil {
		return ""
	}
	return image.GetOsDist()
}

func (self *SInstance) GetOsVersion() string {
	image, err := self.GetImage()
	if err != nil {
		return ""
	}
	return image.GetOsVersion()
}

func (self *SInstance) GetOsLang() string {
	image, err := self.GetImage()
	if err != nil {
		return ""
	}
	return image.GetOsLang()
}

func (self *SInstance) GetMachine() string {
	return "pc"
}

func (self *SInstance) GetInstanceType() string {
	return self.Flavor.FlavorName
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	ret := []string{}
	for _, sec := range self.SecGroupList {
		ret = append(ret, sec.SecurityGroupId)
	}
	return ret, nil
}

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	currentIds, err := self.GetSecurityGroupIds()
	if err != nil {
		return errors.Wrap(err, "GetSecurityGroupIds")
	}

	adds := []string{}
	for i := range secgroupIds {
		if !utils.IsInStringArray(secgroupIds[i], currentIds) {
			adds = append(adds, secgroupIds[i])
		}
	}

	for i := range adds {
		err := self.host.zone.region.AssignSecurityGroup(self.GetId(), adds[i])
		if err != nil {
			return errors.Wrap(err, "Instance.SetSecurityGroups")
		}
	}

	removes := []string{}
	for i := range currentIds {
		if !utils.IsInStringArray(currentIds[i], secgroupIds) {
			removes = append(removes, currentIds[i])
		}
	}

	for i := range removes {
		err := self.host.zone.region.UnsignSecurityGroup(self.GetId(), removes[i])
		if err != nil {
			return errors.Wrap(err, "Instance.SetSecurityGroups")
		}
	}

	return nil
}

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_CTYUN
}

func (self *SInstance) StartVM(ctx context.Context) error {
	return self.host.zone.region.StartVM(self.GetId())
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	return self.host.zone.region.StopVM(self.GetId())
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return self.host.zone.region.DeleteVM(self.GetId())
}

func (self *SRegion) UpdateVM(vmId string, opts cloudprovider.SInstanceUpdateOptions) error {
	params := map[string]interface{}{
		"instanceID":  vmId,
		"displayName": opts.NAME,
	}
	_, err := self.post(SERVICE_ECS, "/v4/ecs/update-instance", params)
	return err
}

func (self *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	if self.DisplayName == input.NAME {
		return nil
	}
	return self.host.zone.region.UpdateVM(self.InstanceId, input)
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) RebuildRoot(vmId string, opts *cloudprovider.SManagedVMRebuildRootConfig) error {
	params := map[string]interface{}{
		"instanceID": vmId,
		"password":   opts.Password,
		"imageID":    opts.ImageId,
	}
	_, err := self.post(SERVICE_ECS, "/v4/ecs/rebuild-instance", params)
	return err
}

func (self *SInstance) RebuildRoot(ctx context.Context, opts *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	oldDiskId := self.AttachedVolume[0]
	err := self.host.zone.region.RebuildRoot(self.InstanceId, opts)
	if err != nil {
		return "", err
	}
	cloudprovider.Wait(time.Second*10, time.Minute*5, func() (bool, error) {
		self.Refresh()
		if oldDiskId != self.AttachedVolume[0] {
			return true, nil
		}
		return false, nil
	})
	return self.AttachedVolume[0], nil
}

func (self *SRegion) AttachKeypair(vmId, keyName string) error {
	params := map[string]interface{}{
		"instanceID":  vmId,
		"keyPairName": keyName,
	}
	_, err := self.post(SERVICE_ECS, "/v4/ecs/keypair/attach-instance", params)
	return err
}

func (self *SRegion) DetachKeypair(vmId, keyName string) error {
	params := map[string]interface{}{
		"instanceID":  vmId,
		"keyPairName": keyName,
	}
	_, err := self.post(SERVICE_ECS, "/v4/ecs/keypair/detach-instance", params)
	return err
}

func (self *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	if len(opts.Password) > 0 {
		return self.host.zone.region.ResetVMPassword(self.GetId(), opts.Password)
	}
	if len(opts.PublicKey) > 0 {
		keypair, err := self.host.zone.region.syncKeypair(opts.Password)
		if err != nil {
			return errors.Wrapf(err, "syncKeypair")
		}
		return self.host.zone.region.AttachKeypair(self.InstanceId, keypair.KeyPairName)
	}
	if opts.DeleteKeypair && len(self.KeypairName) > 0 {
		return self.host.zone.region.DetachKeypair(self.InstanceId, self.KeypairName)
	}
	return nil
}

func (self *SRegion) ChangeVMConfig(id, instanceType string) error {
	skus, err := self.GetServerSkus("")
	if err != nil {
		return errors.Wrapf(err, "GetServerSkus")
	}
	for i := range skus {
		if skus[i].FlavorName == instanceType {
			instanceType = skus[i].FlavorId
			break
		}
	}
	params := map[string]interface{}{
		"instanceID":  id,
		"clientToken": utils.GenRequestId(20),
		"flavorID":    instanceType,
	}
	_, err = self.post(SERVICE_ECS, "/v4/ecs/update-flavor-spec", params)
	return err
}

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return self.host.zone.region.ChangeVMConfig(self.GetId(), config.InstanceType)
}

func (self *SRegion) GetInstanceVnc(vmId string) (string, error) {
	params := map[string]interface{}{
		"instanceID": vmId,
	}
	resp, err := self.list(SERVICE_ECS, "/v4/ecs/vnc/details", params)
	if err != nil {
		return "", err
	}
	return resp.GetString("returnObj", "token")
}

func (self *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	url, err := self.host.zone.region.GetInstanceVnc(self.InstanceId)
	if err != nil {
		return nil, err
	}
	protocol := "ctyun"
	if strings.HasPrefix(url, "wss") {
		protocol = "vnc"
	}
	ret := &cloudprovider.ServerVncOutput{
		Url:        url,
		Protocol:   protocol,
		InstanceId: self.GetId(),
		Hypervisor: api.HYPERVISOR_CTYUN,
	}
	return ret, nil
}

func (self *SRegion) AttachDisk(id, diskId string) error {
	params := map[string]interface{}{
		"diskID":     diskId,
		"instanceID": id,
	}
	_, err := self.post(SERVICE_EBS, "/v4/ebs/attach-ebs", params)
	return err
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return self.host.zone.region.AttachDisk(self.InstanceId, diskId)
}

func (self *SRegion) DetachDisk(id, diskId string) error {
	params := map[string]interface{}{
		"diskID":     diskId,
		"instanceID": id,
	}
	_, err := self.post(SERVICE_EBS, "/v4/ebs/detach-ebs", params)
	return err
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return self.host.zone.region.DetachDisk(self.InstanceId, diskId)
}

func (self *SInstance) Renew(bc billing.SBillingCycle) error {
	_, err := self.host.zone.region.RenewVM(self.GetId(), &bc)
	if err != nil {
		return errors.Wrap(err, "Instance.Renew.RenewVM")
	}

	return nil
}

func (self *SInstance) GetError() error {
	return nil
}

func (self *SRegion) CreateInstance(zoneId string, opts *cloudprovider.SManagedVMCreateConfig) (string, error) {
	image, err := self.GetImage(opts.ExternalImageId)
	if err != nil {
		return "", errors.Wrapf(err, "GetImage %s", opts.ExternalImageId)
	}
	if image.DiskSize > opts.SysDisk.SizeGB {
		opts.SysDisk.SizeGB = image.DiskSize
	}
	if opts.SysDisk.SizeGB < 40 {
		opts.SysDisk.SizeGB = 40
	}
	skus, err := self.GetServerSkus("")
	if err != nil {
		return "", errors.Wrapf(err, "GetServerSkus")
	}
	for i := range skus {
		if skus[i].FlavorName == opts.InstanceType {
			opts.InstanceType = skus[i].FlavorId
			break
		}
	}
	disks := []map[string]interface{}{}
	for _, disk := range opts.DataDisks {
		disks = append(disks, map[string]interface{}{
			"diskName": disk.Name,
			"diskSize": disk.SizeGB,
			"diskType": disk.StorageType,
		})
	}
	nets := map[string]interface{}{
		"subnetID": opts.ExternalNetworkId,
		"isMaster": true,
	}
	if len(opts.IpAddr) > 0 {
		nets["fixedIP"] = opts.IpAddr
	}
	imageType, _ := map[string]int{
		"private":   0,
		"public":    1,
		"shared":    2,
		"safe":      3,
		"community": 4,
	}[image.Visibility]
	params := map[string]interface{}{
		"clientToken":     utils.GenRequestId(20),
		"userPassword":    opts.Password,
		"imageID":         opts.ExternalImageId,
		"imageType":       imageType,
		"userData":        opts.UserData,
		"instanceName":    opts.Hostname,
		"displayName":     opts.Name,
		"flavorID":        opts.InstanceType,
		"onDemand":        true,
		"extIP":           "0",
		"vpcID":           opts.ExternalVpcId,
		"bootDiskType":    opts.SysDisk.StorageType,
		"bootDiskSize":    opts.SysDisk.SizeGB,
		"secGroupList":    opts.ExternalSecgroupIds,
		"azName":          zoneId,
		"dataDiskList":    disks,
		"networkCardList": []map[string]interface{}{nets},
	}

	if len(opts.PublicKey) > 0 {
		keypair, err := self.syncKeypair(opts.PublicKey)
		if err != nil {
			return "", errors.Wrapf(err, "syncKeypair")
		}
		params["keyPairID"] = keypair.KeyPairId
	}

	if opts.BillingCycle != nil {
		params["onDemand"] = false
		if opts.BillingCycle.AutoRenew {
			params["autoRenewStatus"] = "true"
		}
		if opts.BillingCycle.GetYears() > 0 {
			params["cycleType"] = "YEAR"
			params["cycleCount"] = opts.BillingCycle.GetYears()
		} else if opts.BillingCycle.GetMonths() > 0 {
			params["cycleType"] = "MONTH"
			params["cycleCount"] = opts.BillingCycle.GetMonths()
		}
	}

	resp, err := self.post(SERVICE_ECS, "/v4/ecs/create-instance", params)
	if err != nil {
		return "", err
	}

	orderId, err := resp.GetString("returnObj", "masterOrderID")
	if err != nil {
		return "", err
	}

	return self.GetResourceId(orderId)
}

func (self *SRegion) GetResourceId(orderId string) (string, error) {
	params := map[string]interface{}{
		"masterOrderId": orderId,
	}
	for i := 0; i < 10; i++ {
		resp, err := self.list(SERVICE_ECS, "/v4/order/query-uuid", params)
		if err != nil {
			return "", err
		}
		ids := []string{}
		resp.Unmarshal(&ids, "returnObj", "resourceUUID")
		for i := range ids {
			return ids[i], nil
		}
		time.Sleep(time.Second * 10)
	}
	return "", errors.Wrapf(cloudprovider.ErrNotFound, "failed get uuid by order id: %s", orderId)
}

func (self *SRegion) AssignSecurityGroup(vmId, groupId string) error {
	params := map[string]interface{}{
		"securityGroupID": groupId,
		"instanceID":      vmId,
		"action":          "joinSecurityGroup",
	}
	_, err := self.post(SERVICE_ECS, "/v4/ecs/vpc/join-security-group", params)
	return err
}

func (self *SRegion) UnsignSecurityGroup(vmId, groupId string) error {
	params := map[string]interface{}{
		"securityGroupID": groupId,
		"instanceID":      vmId,
		"action":          "joinSecurityGroup",
	}
	_, err := self.post(SERVICE_ECS, "/v4/ecs/vpc/leave-security-group", params)
	return err
}

func (self *SRegion) StartVM(id string) error {
	params := map[string]interface{}{
		"instanceID": id,
	}
	_, err := self.post(SERVICE_ECS, "/v4/ecs/start-instance", params)
	return err
}

func (self *SRegion) StopVM(id string) error {
	params := map[string]interface{}{
		"instanceID": id,
	}
	_, err := self.post(SERVICE_ECS, "/v4/ecs/stop-instance", params)
	return err
}

func (self *SRegion) DeleteVM(id string) error {
	params := map[string]interface{}{
		"instanceID":  id,
		"clientToken": utils.GenRequestId(20),
	}
	_, err := self.post(SERVICE_ECS, "/v4/ecs/unsubscribe-instance", params)
	return err
}

func (self *SRegion) RenewVM(vmId string, bc *billing.SBillingCycle) ([]string, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) ResetVMPassword(id, password string) error {
	params := map[string]interface{}{
		"instanceID":  id,
		"newPassword": password,
	}
	_, err := self.post(SERVICE_ECS, "/v4/ecs/reset-password", params)
	return err
}
