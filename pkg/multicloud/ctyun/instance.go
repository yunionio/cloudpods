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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type SInstance struct {
	multicloud.SInstanceBase

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
	Created                          int64                `json:"created"`
	DueDate                          int64                `json:"dueDate"`
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
	if self.DueDate > 0 {
		return billing_api.BILLING_TYPE_PREPAID
	} else {
		return billing_api.BILLING_TYPE_POSTPAID
	}
}

func (self *SInstance) GetCreatedAt() time.Time {
	return time.Unix(self.Created/1000, 0)
}

func (self *SInstance) GetExpiredAt() time.Time {
	if self.DueDate == 0 {
		return time.Time{}
	}

	return time.Unix(self.DueDate/1000, 0)
}

func (self *SInstance) GetId() string {
	return self.ID
}

func (self *SInstance) GetName() string {
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

	return jsonutils.Update(self, new)
}

func (self *SInstance) IsEmulated() bool {
	return false
}

func (self *SInstance) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	lowerOs := self.GetOSType()
	if strings.HasPrefix(lowerOs, "win") {
		lowerOs = "win"
	}
	priceKey := fmt.Sprintf("%s::%s::%s", self.host.zone.region.GetId(), self.GetInstanceType(), lowerOs)
	data.Add(jsonutils.NewString(priceKey), "price_key")
	data.Add(jsonutils.NewString(self.host.zone.GetGlobalId()), "zone_ext_id")
	return data
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

func (self *SInstance) GetOSType() string {
	image, err := self.GetImage()
	if err != nil {
		log.Errorf("SInstance.GetOSType %s", err)
		return ""
	}

	return image.OSType
}

func (self *SInstance) GetOSName() string {
	image, err := self.GetImage()
	if err != nil {
		log.Errorf("SInstance.GetOSName %s", err)
		return ""
	}

	return image.Name
}

func (self *SInstance) GetBios() string {
	return "BIOS"
}

func (self *SInstance) GetMachine() string {
	return "pc"
}

func (self *SInstance) GetInstanceType() string {
	return self.Flavor.ID
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	if len(self.SecurityGroups) == 0 {
		return nil, nil
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
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_CTYUN
}

func (self *SInstance) StartVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) StopVM(ctx context.Context, isForce bool) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) RebuildRoot(ctx context.Context, imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return cloudprovider.ErrNotImplemented
}

// http://ctyun-api-url/apiproxy/v3/queryVncUrl
func (self *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	url, err := self.host.zone.region.GetInstanceVNCUrl(self.GetId())
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(url), "url")
	ret.Add(jsonutils.NewString("ctyun"), "protocol")
	ret.Add(jsonutils.NewString(self.GetId()), "instance_id")
	return ret, nil
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotImplemented
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

func (self *SRegion) CreateInstance(zoneId, name, imageId, volumetype, flavorRef, vpcid, subnetId, secGroupId, adminPass string) error {
	rootParams := jsonutils.NewDict()
	rootParams.Set("volumetype", jsonutils.NewString(volumetype))

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
	// todo: fix me
	serverParams.Set("osType", jsonutils.NewString("Linux"))
	serverParams.Set("vpcid", jsonutils.NewString(vpcid))
	serverParams.Set("security_groups", secgroupParams)
	serverParams.Set("nics", nicParams)
	serverParams.Set("adminPass", jsonutils.NewString(adminPass))
	serverParams.Set("count", jsonutils.NewString("1"))
	serverParams.Set("extendparam", extParams)

	vmParams := jsonutils.NewDict()
	vmParams.Set("server", serverParams)

	params := map[string]jsonutils.JSONObject{
		"createVMInfo": vmParams,
	}

	_, err := self.client.DoPost("/apiproxy/v3/ondemand/createVM", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.CreateInstance.DoPost")
	}

	return nil
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
