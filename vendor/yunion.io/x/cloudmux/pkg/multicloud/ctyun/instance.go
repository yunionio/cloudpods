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
	"yunion.io/x/log"
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

	vmDetails *InstanceDetails

	HostID                           string               `json:"hostId"`
	ID                               string               `json:"id"`
	Name                             string               `json:"name"`
	Status                           string               `json:"status"`
	TenantID                         string               `json:"tenant_id"`
	Metadata                         Metadata             `json:"metadata"`
	Image                            Image                `json:"image"`
	Flavor                           FlavorObj            `json:"flavor"`
	Addresses                        map[string][]Address `json:"addresses"`
	UserID                           string               `json:"user_id"`
	Created                          time.Time            `json:"created"`
	DueDate                          *time.Time           `json:"dueDate"`
	SecurityGroups                   []SecurityGroup      `json:"security_groups"`
	OSEXTAZAvailabilityZone          string               `json:"OS-EXT-AZ:availability_zone"`
	OSExtendedVolumesVolumesAttached []Volume             `json:"os-extended-volumes:volumes_attached"`
	MasterOrderId                    string               `json:"masterOrderId"`
}

type InstanceDetails struct {
	HostID     string      `json:"hostId"`
	Name       string      `json:"name"`
	Status     string      `json:"status"`
	PrivateIPS []PrivateIP `json:"privateIps"`
	PublicIPS  []PublicIP  `json:"publicIps"`
	Volumes    []Volume    `json:"volumes"`
	Created    string      `json:"created"`
	FlavorObj  FlavorObj   `json:"flavorObj"`
}

type Address struct {
	Addr               string `json:"addr"`
	OSEXTIPSType       string `json:"OS-EXT-IPS:type"`
	Version            int64  `json:"version"`
	OSEXTIPSMACMACAddr string `json:"OS-EXT-IPS-MAC:mac_addr"`
}

type Image struct {
	Id string `json:"id"`
}

type SecurityGroup struct {
	Name string `json:"name"`
}

type FlavorObj struct {
	Name    string `json:"name"`
	CPUNum  int    `json:"cpuNum"`
	MemSize int    `json:"memSize"`
	ID      string `json:"id"`
}

type PrivateIP struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

type PublicP struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	Bandwidth string `json:"bandwidth"`
}

func (self *SInstance) GetBillingType() string {
	if self.DueDate != nil {
		return billing_api.BILLING_TYPE_PREPAID
	} else {
		return billing_api.BILLING_TYPE_POSTPAID
	}
}

func (self *SInstance) GetCreatedAt() time.Time {
	return self.Created
}

func (self *SInstance) GetExpiredAt() time.Time {
	if self.DueDate == nil {
		return time.Time{}
	}

	return *self.DueDate
}

func (self *SInstance) GetId() string {
	return self.ID
}

func (self *SInstance) GetName() string {
	return self.Name
}

func (self *SInstance) GetHostname() string {
	return self.Name
}

func (self *SInstance) GetGlobalId() string {
	return self.GetId()
}

func (self *SInstance) GetStatus() string {
	switch self.Status {
	case "RUNNING", "ACTIVE":
		return api.VM_RUNNING
	case "RESTARTING", "BUILD", "RESIZE", "VERIFY_RESIZE":
		return api.VM_STARTING
	case "STOPPING", "HARD_REBOOT":
		return api.VM_STOPPING
	case "STOPPED", "SHUTOFF":
		return api.VM_READY
	default:
		return api.VM_UNKNOWN
	}
}

func (self *SInstance) Refresh() error {
	new, err := self.host.zone.region.GetVMById(self.GetId())
	if err != nil {
		return err
	}

	new.host = self.host
	if err != nil {
		return err
	}

	if new.Status == "DELETED" {
		log.Debugf("Instance already terminated.")
		return cloudprovider.ErrNotFound
	}

	// update details
	detail, err := self.host.zone.region.GetVMDetails(self.GetId())
	if err != nil {
		return errors.Wrap(err, "SInstance.Refresh.GetDetails")
	}

	self.vmDetails = detail
	return jsonutils.Update(self, new)
}

func (self *SInstance) IsEmulated() bool {
	return false
}

func (self *SInstance) GetProjectId() string {
	return ""
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

// GET http://ctyun-api-url/apiproxy/v3/queryDataDiskByVMId
func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	details, err := self.GetDetails()
	if err != nil {
		return nil, errors.Wrap(err, "SInstance.GetIDisks.GetDetails")
	}

	disks := []SDisk{}
	for i := range details.Volumes {
		volume := details.Volumes[i]
		disk, err := self.host.zone.region.GetDisk(volume.ID)
		if err != nil {
			return nil, errors.Wrap(err, "SInstance.GetIDisks.GetDisk")
		}

		disks = append(disks, *disk)
	}

	for i := 0; i < len(disks); i += 1 {
		// 将系统盘放到第0个位置
		if disks[i].Bootable == "true" {
			_temp := disks[0]
			disks[0] = disks[i]
			disks[i] = _temp
		}
	}

	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := range disks {
		disk := disks[i]
		idisks[i] = &disk
	}

	return idisks, nil
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics, err := self.host.zone.region.GetNics(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "SInstance.GetINics")
	}

	inics := make([]cloudprovider.ICloudNic, len(nics))
	for i := range nics {
		inics[i] = &nics[i]
	}

	return inics, nil
}

// GET http://ctyun-api-urlapiproxy/v3/queryNetworkByVMId
func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	detail, err := self.GetDetails()
	if err != nil {
		return nil, errors.Wrap(err, "SInstance.GetIEIP.GetDetails")
	}

	if len(detail.PublicIPS) > 0 {
		return self.host.zone.region.GetEip(detail.PublicIPS[0].ID)
	}

	return nil, nil
}

func (self *SInstance) GetDetails() (*InstanceDetails, error) {
	if self.vmDetails != nil {
		return self.vmDetails, nil
	}

	detail, err := self.host.zone.region.GetVMDetails(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "SInstance.GetDetails")
	}

	self.vmDetails = detail
	return self.vmDetails, nil
}

func (self *SInstance) GetVcpuCount() int {
	details, err := self.GetDetails()
	if err == nil {
		return details.FlavorObj.CPUNum
	}

	return self.Flavor.CPUNum
}

func (self *SInstance) GetVmemSizeMB() int {
	details, err := self.GetDetails()
	if err == nil {
		return details.FlavorObj.MemSize * 1024
	}

	return self.Flavor.MemSize * 1024
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

	image, err := self.host.zone.region.GetImage(self.Image.Id)
	if err != nil {
		return nil, errors.Wrap(err, "SInstance.GetImage")
	}

	self.image = image
	return self.image, nil
}

func (self *SInstance) GetOsType() cloudprovider.TOsType {
	image, err := self.GetImage()
	if err != nil {
		log.Errorf("SInstance.Image %s", err)
		return cloudprovider.OsTypeLinux
	}
	return image.GetOsType()
}

func (self *SInstance) GetFullOsName() string {
	image, err := self.GetImage()
	if err != nil {
		log.Errorf("SInstance.Image %s", err)
		return ""
	}
	return image.GetFullOsName()
}

func (self *SInstance) GetBios() cloudprovider.TBiosType {
	image, err := self.GetImage()
	if err != nil {
		log.Errorf("SInstance.Image %s", err)
		return cloudprovider.BIOS
	}
	return image.GetBios()
}

func (self *SInstance) GetOsArch() string {
	image, err := self.GetImage()
	if err != nil {
		log.Errorf("SInstance.Image %s", err)
		return ""
	}
	return image.GetOsArch()
}

func (self *SInstance) GetOsDist() string {
	image, err := self.GetImage()
	if err != nil {
		log.Errorf("SInstance.Image %s", err)
		return ""
	}
	return image.GetOsDist()
}

func (self *SInstance) GetOsVersion() string {
	image, err := self.GetImage()
	if err != nil {
		log.Errorf("SInstance.Image %s", err)
		return ""
	}
	return image.GetOsVersion()
}

func (self *SInstance) GetOsLang() string {
	image, err := self.GetImage()
	if err != nil {
		log.Errorf("SInstance.Image %s", err)
		return ""
	}
	return image.GetOsLang()
}

func (self *SInstance) GetMachine() string {
	return "pc"
}

func (self *SInstance) GetInstanceType() string {
	return self.Flavor.ID
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	if len(self.SecurityGroups) == 0 {
		return []string{}, nil
	}

	if len(self.MasterOrderId) > 0 {
		return self.getSecurityGroupIdsByMasterOrderId(self.MasterOrderId)
	}

	secgroups, err := self.host.zone.region.GetSecurityGroups("")
	if err != nil {
		return nil, errors.Wrap(err, "SInstance.GetSecurityGroupIds.GetSecurityGroups")
	}

	names := []string{}
	for i := range self.SecurityGroups {
		names = append(names, self.SecurityGroups[i].Name)
	}

	ids := []string{}
	for i := range secgroups {
		// todo: bugfix 如果安全组重名比较尴尬
		if utils.IsInStringArray(secgroups[i].Name, names) {
			ids = append(ids, secgroups[i].ResSecurityGroupID)
		}
	}

	return ids, nil
}

func (self *SInstance) getSecurityGroupIdsByMasterOrderId(orderId string) ([]string, error) {
	orders, err := self.host.zone.region.GetOrder(self.MasterOrderId)
	if err != nil {
		return nil, errors.Wrap(err, "SInstance.GetSecurityGroupIds.GetOrder")
	}

	if len(orders) == 0 {
		return nil, nil
	}

	for i := range orders {
		secgroups := orders[i].ResourceConfigMap.SecurityGroups
		if len(secgroups) > 0 {
			ids := []string{}
			for j := range secgroups {
				ids = append(ids, secgroups[j].ID)
			}

			return ids, nil
		}
	}

	return nil, nil
}

func (self *SInstance) AssignSecurityGroup(secgroupId string) error {
	return self.host.zone.region.AssignSecurityGroup(self.GetId(), secgroupId)
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
	err := self.host.zone.region.StartVM(self.GetId())
	if err != nil {
		return errors.Wrap(err, "Instance.StartVM")
	}

	err = cloudprovider.WaitStatus(self, api.VM_RUNNING, 5*time.Second, 300*time.Second)
	if err != nil {
		return errors.Wrap(err, "Instance.StartVM.WaitStatus")
	}
	return nil
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	err := self.host.zone.region.StopVM(self.GetId())
	if err != nil {
		return errors.Wrap(err, "Instance.StopVM")
	}

	err = cloudprovider.WaitStatus(self, api.VM_READY, 5*time.Second, 300*time.Second)
	if err != nil {
		return errors.Wrap(err, "Instance.StopVM.WaitStatus")
	}
	return nil
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	err := self.host.zone.region.DeleteVM(self.GetId())
	if err != nil {
		return errors.Wrap(err, "SInstance.DeleteVM")
	}

	err = cloudprovider.WaitDeleted(self, 10*time.Second, 180*time.Second)
	if err != nil {
		return errors.Wrap(err, "Instance.DeleteVM.WaitDeleted")
	}
	return nil
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) RebuildRoot(ctx context.Context, config *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	currentImage, err := self.GetImage()
	if err != nil {
		return "", errors.Wrap(err, "Instance.RebuildRoot")
	}

	publicKeyName := ""
	if len(config.PublicKey) > 0 {
		publicKeyName, err = self.host.zone.region.syncKeypair(config.PublicKey)
		if err != nil {
			return "", errors.Wrap(err, "Instance.RebuildRoot.syncKeypair")
		}
	}

	jobId := ""
	if currentImage.GetId() != config.ImageId {
		jobId, err = self.host.zone.region.SwitchVMOs(self.GetId(), config.Password, publicKeyName, config.ImageId)
		if err != nil {
			return "", errors.Wrap(err, "SInstance.RebuildRoot.SwitchVMOs")
		}
	} else {
		jobId, err = self.host.zone.region.RebuildVM(self.GetId(), config.Password, publicKeyName)
		if err != nil {
			return "", errors.Wrap(err, "SInstance.RebuildRoot.RebuildVM")
		}
	}

	err = cloudprovider.Wait(10*time.Second, 1800*time.Second, func() (b bool, err error) {
		statusJson, err := self.host.zone.region.GetJob(jobId)
		if err != nil {
			if strings.Contains(err.Error(), "job fail") {
				return false, err
			}

			return false, nil
		}

		if status, _ := statusJson.GetString("status"); status == "SUCCESS" {
			return true, nil
		} else if status == "FAILED" {
			return false, fmt.Errorf("RebuildRoot job %s failed", jobId)
		} else {
			return false, nil
		}
	})
	if err != nil {
		return "", errors.Wrap(err, "Instance.RebuildRoot.Wait")
	}

	err = self.Refresh()
	if err != nil {
		return "", err
	}

	idisks, err := self.GetIDisks()
	if err != nil {
		return "", err
	}

	if len(idisks) == 0 {
		return "", fmt.Errorf("server %s has no volume attached.", self.GetId())
	}

	return idisks[0].GetId(), nil
}

func (self *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	if len(password) == 0 {
		return cloudprovider.ErrNotSupported
	}

	// 只支持重置密码
	return self.host.zone.region.ResetVMPassword(self.GetId(), password)
}

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	jobId, err := self.host.zone.region.ChangeVMConfig(self.GetId(), config.InstanceType)
	if err != nil {
		return errors.Wrap(err, "Instance.ChangeConfig")
	}

	err = cloudprovider.Wait(10*time.Second, 1800*time.Second, func() (b bool, err error) {
		statusJson, err := self.host.zone.region.GetJob(jobId)
		if err != nil {
			if strings.Contains(err.Error(), "job fail") {
				return false, err
			}

			return false, nil
		}

		if status, _ := statusJson.GetString("status"); status == "SUCCESS" {
			return true, nil
		} else if status == "FAILED" {
			return false, fmt.Errorf("ChangeConfig job %s failed", jobId)
		} else {
			return false, nil
		}
	})
	if err != nil {
		return errors.Wrap(err, "Instance.ChangeConfig.Wait")
	}

	return nil
}

// http://ctyun-api-url/apiproxy/v3/queryVncUrl
func (self *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	url, err := self.host.zone.region.GetInstanceVNCUrl(self.GetId())
	if err != nil {
		return nil, err
	}
	ret := &cloudprovider.ServerVncOutput{
		Url:        url,
		Protocol:   "ctyun",
		InstanceId: self.GetId(),
		Hypervisor: api.HYPERVISOR_CTYUN,
	}
	return ret, nil
}

func (self *SInstance) NextDeviceName() (string, error) {
	details, err := self.GetDetails()
	if err != nil {
		return "", errors.Wrap(err, "SInstance.NextDeviceName.GetDetails")
	}

	disks := []*SDisk{}
	for i := range details.Volumes {
		disk, err := self.host.zone.region.GetDisk(details.Volumes[i].ID)
		if err != nil {
			return "", errors.Wrap(err, "SInstance.NextDeviceName.GetDisk")
		}

		disks = append(disks, disk)
	}

	prefix := "s"
	if len(disks) > 0 && strings.Contains(disks[0].GetMountpoint(), "/vd") {
		prefix = "v"
	}

	currents := []string{}
	for _, disk := range disks {
		currents = append(currents, strings.ToLower(disk.GetMountpoint()))
	}

	for i := 0; i < 25; i++ {
		device := fmt.Sprintf("/dev/%sd%s", prefix, string([]byte{byte(98 + i)}))
		if ok, _ := utils.InStringArray(device, currents); !ok {
			return device, nil
		}
	}

	return "", fmt.Errorf("disk devicename out of index, current deivces: %s", currents)
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	device, err := self.NextDeviceName()
	if err != nil {
		return errors.Wrap(err, "Instance.AttachDisk.NextDeviceName")
	}

	_, err = self.host.zone.region.AttachDisk(self.GetId(), diskId, device)
	if err != nil {
		return errors.Wrap(err, "Instance.AttachDisk")
	}

	disk, err := self.host.zone.region.GetDisk(diskId)
	if err != nil {
		return errors.Wrap(err, "AttachDisk.GetDisk")
	}

	err = cloudprovider.WaitStatusWithDelay(disk, api.DISK_READY, 10*time.Second, 5*time.Second, 180*time.Second)
	if err != nil {
		return errors.Wrap(err, "Instance.DetachDisk.WaitStatusWithDelay")
	}

	if disk.Status != "in-use" {
		return errors.Wrap(fmt.Errorf("disk status %s", disk.Status), "Instance.DetachDisk.Status")
	}

	return nil
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	disk, err := self.host.zone.region.GetDisk(diskId)
	if err != nil {
		return errors.Wrap(err, "DetachDisk.Wait")
	}

	if len(disk.Attachments) == 0 {
		return errors.Wrap(err, "Instance.DetachDisk")
	}

	_, err = self.host.zone.region.DetachDisk(self.GetId(), diskId, disk.Attachments[0].Device)
	if err != nil {
		return errors.Wrap(err, "Instance.DetachDisk")
	}

	err = cloudprovider.WaitStatusWithDelay(disk, api.DISK_READY, 10*time.Second, 5*time.Second, 180*time.Second)
	if err != nil {
		return errors.Wrap(err, "Instance.DetachDisk.WaitStatusWithDelay")
	}

	if disk.Status != "available" {
		return errors.Wrap(fmt.Errorf("disk status %s", disk.Status), "Instance.DetachDisk.Status")
	}

	return nil
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

type SDiskDetails struct {
	HostID     string      `json:"hostId"`
	Name       string      `json:"name"`
	Status     string      `json:"status"`
	PrivateIPS []PrivateIP `json:"privateIps"`
	PublicIPS  []PublicIP  `json:"publicIps"`
	Volumes    []Volume    `json:"volumes"`
	Created    string      `json:"created"`
	FlavorObj  FlavorObj   `json:"flavorObj"`
}

type PublicIP struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	Bandwidth string `json:"bandwidth"`
}

type Volume struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Type     string `json:"type"`
	Size     string `json:"size"`
	Name     string `json:"name"`
	Bootable bool   `json:"bootable"`
}

func (self *SRegion) GetVMDetails(vmId string) (*InstanceDetails, error) {
	params := map[string]string{
		"regionId": self.GetId(),
		"vmId":     vmId,
	}

	resp, err := self.client.DoGet("/apiproxy/v3/ondemand/queryVMDetail", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetVMDetails.DoGet")
	}

	details := &InstanceDetails{}
	err = resp.Unmarshal(details, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetVMDetails.Unmarshal")
	}

	return details, nil
}

type SVncInfo struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

func (self *SRegion) GetInstanceVNCUrl(vmId string) (string, error) {
	params := map[string]string{
		"regionId": self.GetId(),
		"vmId":     vmId,
	}

	resp, err := self.client.DoGet("/apiproxy/v3/queryVncUrl", params)
	if err != nil {
		return "", errors.Wrap(err, "")
	}

	ret := SVncInfo{}
	err = resp.Unmarshal(&ret, "returnObj", "console")
	if err != nil {
		return "", errors.Wrap(err, "")
	}

	return ret.URL, nil
}

/*
创建主机接口目前没有绑定密钥的参数选项，不支持绑定密码。
但是重装系统接口支持绑定密钥
*/
func (self *SRegion) CreateInstance(zoneId, name, imageId, osType, flavorRef, vpcid, subnetId, secGroupId, adminPass, volumetype string, volumeSize int, dataDisks []cloudprovider.SDiskInfo) (string, error) {
	rootParams := jsonutils.NewDict()
	rootParams.Set("volumetype", jsonutils.NewString(volumetype))
	if volumeSize > 0 {
		rootParams.Set("size", jsonutils.NewInt(int64(volumeSize)))
	}

	nicParams := jsonutils.NewArray()
	nicParam := jsonutils.NewDict()
	nicParam.Set("subnet_id", jsonutils.NewString(subnetId))
	nicParams.Add(nicParam)

	secgroupParams := jsonutils.NewArray()
	secgroupParam := jsonutils.NewDict()
	secgroupParam.Set("id", jsonutils.NewString(secGroupId))
	secgroupParams.Add(secgroupParam)

	extParams := jsonutils.NewDict()
	extParams.Set("regionID", jsonutils.NewString(self.GetId()))

	serverParams := jsonutils.NewDict()
	serverParams.Set("availability_zone", jsonutils.NewString(zoneId))
	serverParams.Set("name", jsonutils.NewString(name))
	serverParams.Set("imageRef", jsonutils.NewString(imageId))
	serverParams.Set("root_volume", rootParams)
	serverParams.Set("flavorRef", jsonutils.NewString(flavorRef))
	serverParams.Set("osType", jsonutils.NewString(osType))
	serverParams.Set("vpcid", jsonutils.NewString(vpcid))
	serverParams.Set("security_groups", secgroupParams)
	serverParams.Set("nics", nicParams)
	serverParams.Set("adminPass", jsonutils.NewString(adminPass))
	serverParams.Set("count", jsonutils.NewString("1"))
	serverParams.Set("extendparam", extParams)

	if dataDisks != nil && len(dataDisks) > 0 {
		dataDisksParams := jsonutils.NewArray()
		for i := range dataDisks {
			dataDiskParams := jsonutils.NewDict()
			dataDiskParams.Set("volumetype", jsonutils.NewString(dataDisks[i].StorageType))
			dataDiskParams.Set("size", jsonutils.NewInt(int64(dataDisks[i].SizeGB)))
			dataDisksParams.Add(dataDiskParams)
		}

		serverParams.Set("data_volumes", dataDisksParams)
	}

	vmParams := jsonutils.NewDict()
	vmParams.Set("server", serverParams)

	params := map[string]jsonutils.JSONObject{
		"createVMInfo": vmParams,
	}

	resp, err := self.client.DoPost("/apiproxy/v3/ondemand/createVM", params)
	if err != nil {
		return "", errors.Wrap(err, "SRegion.CreateInstance.DoPost")
	}

	var ok bool
	err = resp.Unmarshal(&ok, "returnObj", "status")
	if !ok {
		msg, _ := resp.GetString("returnObj", "message")
		return "", errors.Wrap(fmt.Errorf(msg), "SRegion.CreateInstance.JobFailed")
	}

	var jobId string
	err = resp.Unmarshal(&jobId, "returnObj", "data")
	if err != nil {
		return "", errors.Wrap(err, "SRegion.CreateInstance.Unmarshal")
	}

	return jobId, nil
}

// vm & nic job
func (self *SRegion) GetJob(jobId string) (jsonutils.JSONObject, error) {
	params := map[string]string{
		"regionId": self.GetId(),
		"jobId":    jobId,
	}

	resp, err := self.client.DoGet("/apiproxy/v3/queryJobStatus", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetJob.DoGet")
	}

	ret := jsonutils.NewDict()
	err = resp.Unmarshal(&ret, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetJob.Unmarshal")
	}

	return ret, nil
}

// 查询云硬盘备份JOB状态信息
func (self *SRegion) GetVbsJob(jobId string) (jsonutils.JSONObject, error) {
	params := map[string]string{
		"regionId": self.GetId(),
		"jobId":    jobId,
	}

	resp, err := self.client.DoGet("/apiproxy/v3/ondemand/queryVbsJob", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetVbsJob.DoGet")
	}

	ret := jsonutils.NewDict()
	err = resp.Unmarshal(&ret, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetVbsJob.Unmarshal")
	}

	return ret, nil
}

// 查询云硬盘JOB状态信息
func (self *SRegion) GetVolumeJob(jobId string) (jsonutils.JSONObject, error) {
	params := map[string]string{
		"regionId": self.GetId(),
		"jobId":    jobId,
	}

	resp, err := self.client.DoGet("/apiproxy/v3/queryVolumeJob", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetVolumeJob.DoGet")
	}

	ret := jsonutils.NewDict()
	err = resp.Unmarshal(&ret, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetVolumeJob.Unmarshal")
	}

	return ret, nil
}

// POST http://ctyun-api-url/apiproxy/v3/addSecurityGroup 绑定安全组
func (self *SRegion) AssignSecurityGroup(vmId, securityGroupRuleId string) error {
	securityParams := jsonutils.NewDict()
	securityParams.Set("regionId", jsonutils.NewString(self.GetId()))
	securityParams.Set("vmId", jsonutils.NewString(vmId))
	securityParams.Set("securityGroupRuleId", jsonutils.NewString(securityGroupRuleId))

	params := map[string]jsonutils.JSONObject{
		"securityGroup": securityParams,
	}

	_, err := self.client.DoPost("/apiproxy/v3/addSecurityGroup", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.AssignSecurityGroup.DoPost")
	}

	return nil
}

// POST http://ctyun-api-url/apiproxy/v3/removeSecurityGroup 解绑安全组
func (self *SRegion) UnsignSecurityGroup(vmId, securityGroupRuleId string) error {
	securityParams := jsonutils.NewDict()
	securityParams.Set("regionId", jsonutils.NewString(self.GetId()))
	securityParams.Set("vmId", jsonutils.NewString(vmId))
	securityParams.Set("securityGroupRuleId", jsonutils.NewString(securityGroupRuleId))

	params := map[string]jsonutils.JSONObject{
		"securityGroup": securityParams,
	}

	_, err := self.client.DoPost("/apiproxy/v3/removeSecurityGroup", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.UnsignSecurityGroup.DoPost")
	}

	return nil
}

func (self *SRegion) StartVM(vmId string) error {
	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"vmId":     jsonutils.NewString(vmId),
	}

	_, err := self.client.DoPost("/apiproxy/v3/ondemand/startVM", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.StartVm.DoPost")
	}

	return nil
}

func (self *SRegion) StopVM(vmId string) error {
	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"vmId":     jsonutils.NewString(vmId),
	}

	_, err := self.client.DoPost("/apiproxy/v3/ondemand/stopVM", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.StopVM.DoPost")
	}

	return nil
}

func (self *SRegion) DeleteVM(vmId string) error {
	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"vmId":     jsonutils.NewString(vmId),
	}

	_, err := self.client.DoPost("/apiproxy/v3/ondemand/deleteVM", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.DeleteVM.DoPost")
	}

	return nil
}

func (self *SRegion) RestartVM(vmId string) error {
	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"vmId":     jsonutils.NewString(vmId),
		"type":     jsonutils.NewString("SOFT"),
	}

	_, err := self.client.DoPost("/apiproxy/v3/ondemand/restartVM", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.RestartVM.DoPost")
	}

	return nil
}

func (self *SRegion) SwitchVMOs(vmId, adminPass, keyName, imageRef string) (string, error) {
	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"vmId":     jsonutils.NewString(vmId),
		"imageRef": jsonutils.NewString(imageRef),
	}

	if len(keyName) > 0 {
		params["keyName"] = jsonutils.NewString(keyName)
	} else if len(adminPass) > 0 {
		params["adminPass"] = jsonutils.NewString(adminPass)
	} else {
		return "", errors.Wrap(fmt.Errorf("require public key or password"), "SRegion.SwitchVMOs")
	}

	resp, err := self.client.DoPost("/apiproxy/v3/ondemand/switchSys", params)
	if err != nil {
		return "", errors.Wrap(err, "SRegion.SwitchVMOs.DoPost")
	}

	var ok bool
	err = resp.Unmarshal(&ok, "returnObj", "status")
	if !ok {
		msg, _ := resp.GetString("returnObj", "message")
		return "", errors.Wrap(fmt.Errorf(msg), "SRegion.SwitchVMOs.JobFailed")
	}

	var jobId string
	err = resp.Unmarshal(&jobId, "returnObj", "data")
	if err != nil {
		return "", errors.Wrap(err, "SRegion.SwitchVMOs.Unmarshal")
	}

	return jobId, nil
}

func (self *SRegion) RebuildVM(vmId, adminPass, keyName string) (string, error) {
	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"vmId":     jsonutils.NewString(vmId),
	}

	if len(keyName) > 0 {
		params["keyName"] = jsonutils.NewString(keyName)
	} else if len(adminPass) > 0 {
		params["adminPass"] = jsonutils.NewString(adminPass)
	} else {
		return "", errors.Wrap(fmt.Errorf("require public key or password"), "SRegion.RebuildVM")
	}

	resp, err := self.client.DoPost("/apiproxy/v3/ondemand/reInstallSys", params)
	if err != nil {
		return "", errors.Wrap(err, "SRegion.RebuildVM.DoPost")
	}

	var ok bool
	err = resp.Unmarshal(&ok, "returnObj", "status")
	if !ok {
		msg, _ := resp.GetString("returnObj", "message")
		return "", errors.Wrap(fmt.Errorf(msg), "SRegion.RebuildVM.JobFailed")
	}

	var jobId string
	err = resp.Unmarshal(&jobId, "returnObj", "data")
	if err != nil {
		return "", errors.Wrap(err, "SRegion.RebuildVM.Unmarshal")
	}

	return jobId, nil
}

func (self *SRegion) AttachDisk(vmId, volumeId, device string) (string, error) {
	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"volumeId": jsonutils.NewString(volumeId),
		"vmId":     jsonutils.NewString(vmId),
		"device":   jsonutils.NewString(device),
	}

	resp, err := self.client.DoPost("/apiproxy/v3/ondemand/attachVolume", params)
	if err != nil {
		return "", errors.Wrap(err, "SRegion.AttachDisk.DoPost")
	}

	var ok bool
	err = resp.Unmarshal(&ok, "returnObj", "status")
	if !ok {
		msg, _ := resp.GetString("returnObj", "message")
		return "", errors.Wrap(fmt.Errorf(msg), "SRegion.AttachDisk.JobFailed")
	}

	var jobId string
	err = resp.Unmarshal(&jobId, "returnObj", "data")
	if err != nil {
		return "", errors.Wrap(err, "SRegion.AttachDisk.Unmarshal")
	}

	return jobId, nil
}

func (self *SRegion) DetachDisk(vmId, volumeId, device string) (string, error) {
	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"volumeId": jsonutils.NewString(volumeId),
		"vmId":     jsonutils.NewString(vmId),
		"device":   jsonutils.NewString(device),
	}

	resp, err := self.client.DoPost("/apiproxy/v3/ondemand/uninstallVolume", params)
	if err != nil {
		return "", errors.Wrap(err, "SRegion.DetachDisk.DoPost")
	}

	var ok bool
	err = resp.Unmarshal(&ok, "returnObj", "status")
	if !ok {
		msg, _ := resp.GetString("returnObj", "message")
		return "", errors.Wrap(fmt.Errorf(msg), "SRegion.DetachDisk.JobFailed")
	}

	var jobId string
	err = resp.Unmarshal(&jobId, "returnObj", "data")
	if err != nil {
		return "", errors.Wrap(err, "SRegion.DetachDisk.Unmarshal")
	}

	return jobId, nil
}

func (self *SRegion) ChangeVMConfig(vmId, flavorId string) (string, error) {
	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"vmId":     jsonutils.NewString(vmId),
		"flavorId": jsonutils.NewString(flavorId),
	}

	resp, err := self.client.DoPost("/apiproxy/v3/ondemand/upgradeVM", params)
	if err != nil {
		return "", errors.Wrap(err, "SRegion.ChangeVMConfig.DoPost")
	}

	var ok bool
	err = resp.Unmarshal(&ok, "returnObj", "status")
	if !ok {
		msg, _ := resp.GetString("returnObj", "message")
		return "", errors.Wrap(fmt.Errorf(msg), "SRegion.ChangeVMConfig.JobFailed")
	}

	var jobId string
	err = resp.Unmarshal(&jobId, "returnObj", "data")
	if err != nil {
		return "", errors.Wrap(err, "SRegion.ChangeVMConfig.Unmarshal")
	}

	return jobId, nil
}

// POST http://ctyun-api-url/apiproxy/v3/order/placeRenewOrder 续订
func (self *SRegion) RenewVM(vmId string, bc *billing.SBillingCycle) ([]string, error) {
	if bc == nil {
		return nil, errors.Wrap(fmt.Errorf("SBillingCycle is nil"), "Region.RenewVM")
	}

	resourcePackage := jsonutils.NewDict()
	month := bc.GetMonths()
	switch {
	case month <= 11:
		resourcePackage.Set("cycleCount", jsonutils.NewString(strconv.Itoa(month)))
		resourcePackage.Set("cycleType", jsonutils.NewString("3"))
	case month == 12:
		resourcePackage.Set("cycleCount", jsonutils.NewString("1"))
		resourcePackage.Set("cycleType", jsonutils.NewString("5"))
	case month == 24:
		resourcePackage.Set("cycleCount", jsonutils.NewString("1"))
		resourcePackage.Set("cycleType", jsonutils.NewString("6"))
	case month == 36:
		resourcePackage.Set("cycleCount", jsonutils.NewString("1"))
		resourcePackage.Set("cycleType", jsonutils.NewString("7"))
	default:
		return nil, errors.Wrap(fmt.Errorf("unsupported month duration %d. expected 1~11, 12, 24, 36", month), "Region.RenewVM")
	}

	vmIds := jsonutils.NewArray()
	vmIds.Add(jsonutils.NewString(vmId))
	resourcePackage.Set("resourceIds", vmIds)

	params := map[string]jsonutils.JSONObject{
		"resourceDetailJson": resourcePackage,
	}

	resp, err := self.client.DoPost("/apiproxy/v3/order/placeRenewOrder", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.RenewVM.DoPost")
	}

	var ok bool
	err = resp.Unmarshal(&ok, "returnObj", "submitted")
	if !ok {
		msg, _ := resp.GetString("returnObj", "message")
		return nil, errors.Wrap(fmt.Errorf(msg), "SRegion.RenewVM.JobFailed")
	}

	type OrderPlacedEventsElement struct {
		ErrorMessage string `json:"errorMessage"`
		Submitted    bool   `json:"submitted"`
		NewOrderID   string `json:"newOrderId"`
		NewOrderNo   string `json:"newOrderNo"`
		TotalPrice   int64  `json:"totalPrice"`
	}

	orders := []OrderPlacedEventsElement{}
	err = resp.Unmarshal(&orders, "returnObj", "orderPlacedEvents")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.RenewVM.Unmarshal")
	}

	orderIds := []string{}
	for i := range orders {
		orderIds = append(orderIds, orders[i].NewOrderID)
	}

	return orderIds, nil
}

func (self *SRegion) ResetVMPassword(vmId, password string) error {
	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"vmId":     jsonutils.NewString(vmId),
		"password": jsonutils.NewString(password),
	}

	_, err := self.client.DoPost("/apiproxy/v3/resetVmPassword", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.ResetVMPassword.DoPost")
	}

	return nil
}
