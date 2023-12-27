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

package huawei

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/apis"
	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	InstanceStatusRunning    = "ACTIVE"
	InstanceStatusTerminated = "DELETED"
	InstanceStatusStopped    = "SHUTOFF"
)

type IpAddress struct {
	Version            string `json:"version"`
	Addr               string `json:"addr"`
	OSEXTIPSMACMACAddr string `json:"OS-EXT-IPS-MAC:mac_addr"`
	OSEXTIPSPortID     string `json:"OS-EXT-IPS:port_id"`
	OSEXTIPSType       string `json:"OS-EXT-IPS:type"`
}

type Flavor struct {
	Disk  string `json:"disk"`
	Vcpus string `json:"vcpus"`
	RAM   string `json:"ram"`
	ID    string `json:"id"`
	Name  string `json:"name"`
}

type Image struct {
	ID string `json:"id"`
}

type VMMetadata struct {
	MeteringImageID           string `json:"metering.image_id"`
	MeteringImagetype         string `json:"metering.imagetype"`
	MeteringOrderId           string `json:"metering.order_id"`
	MeteringResourcespeccode  string `json:"metering.resourcespeccode"`
	ImageName                 string `json:"image_name"`
	OSBit                     string `json:"os_bit"`
	VpcID                     string `json:"vpc_id"`
	MeteringResourcetype      string `json:"metering.resourcetype"`
	CascadedInstanceExtrainfo string `json:"cascaded.instance_extrainfo"`
	OSType                    string `json:"os_type"`
	ChargingMode              string `json:"charging_mode"`
}

type OSExtendedVolumesVolumesAttached struct {
	Device              string `json:"device"`
	BootIndex           string `json:"bootIndex"`
	ID                  string `json:"id"`
	DeleteOnTermination string `json:"delete_on_termination"`
}

type OSSchedulerHints struct {
}

type SecurityGroup struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type SysTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SInstance struct {
	multicloud.SInstanceBase
	HuaweiTags

	host *SHost

	image *SImage

	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Addresses   map[string][]IpAddress `json:"addresses"`
	Flavor      Flavor                 `json:"flavor"`
	AccessIPv4  string                 `json:"accessIPv4"`
	AccessIPv6  string                 `json:"accessIPv6"`
	Status      string                 `json:"status"`
	Progress    string                 `json:"progress"`
	HostID      string                 `json:"hostId"`
	Image       Image                  `json:"image"`
	Updated     string                 `json:"updated"`
	Created     time.Time              `json:"created"`
	Metadata    VMMetadata             `json:"metadata"`
	Description string                 `json:"description"`
	Locked      bool                   `json:"locked"`
	ConfigDrive string                 `json:"config_drive"`
	TenantID    string                 `json:"tenant_id"`
	UserID      string                 `json:"user_id"`
	KeyName     string                 `json:"key_name"`

	OSExtendedVolumesVolumesAttached []OSExtendedVolumesVolumesAttached `json:"os-extended-volumes:volumes_attached"`
	OSEXTSTSTaskState                string                             `json:"OS-EXT-STS:task_state"`
	OSEXTSTSPowerState               int64                              `json:"OS-EXT-STS:power_state"`
	OSEXTSTSVMState                  string                             `json:"OS-EXT-STS:vm_state"`
	OSEXTSRVATTRHost                 string                             `json:"OS-EXT-SRV-ATTR:host"`
	OSEXTSRVATTRInstanceName         string                             `json:"OS-EXT-SRV-ATTR:instance_name"`
	OSEXTSRVATTRHypervisorHostname   string                             `json:"OS-EXT-SRV-ATTR:hypervisor_hostname"`
	OSDCFDiskConfig                  string                             `json:"OS-DCF:diskConfig"`
	OSEXTAZAvailabilityZone          string                             `json:"OS-EXT-AZ:availability_zone"`
	OSSchedulerHints                 OSSchedulerHints                   `json:"os:scheduler_hints"`
	OSEXTSRVATTRRootDeviceName       string                             `json:"OS-EXT-SRV-ATTR:root_device_name"`
	OSEXTSRVATTRRamdiskID            string                             `json:"OS-EXT-SRV-ATTR:ramdisk_id"`
	EnterpriseProjectID              string                             `json:"enterprise_project_id"`
	OSEXTSRVATTRUserData             string                             `json:"OS-EXT-SRV-ATTR:user_data"`
	OSSRVUSGLaunchedAt               time.Time                          `json:"OS-SRV-USG:launched_at"`
	OSEXTSRVATTRKernelID             string                             `json:"OS-EXT-SRV-ATTR:kernel_id"`
	OSEXTSRVATTRLaunchIndex          int64                              `json:"OS-EXT-SRV-ATTR:launch_index"`
	HostStatus                       string                             `json:"host_status"`
	OSEXTSRVATTRReservationID        string                             `json:"OS-EXT-SRV-ATTR:reservation_id"`
	OSEXTSRVATTRHostname             string                             `json:"OS-EXT-SRV-ATTR:hostname"`
	OSSRVUSGTerminatedAt             time.Time                          `json:"OS-SRV-USG:terminated_at"`
	SysTags                          []SysTag                           `json:"sys_tags"`
	SecurityGroups                   []SecurityGroup                    `json:"security_groups"`
	EnterpriseProjectId              string
}

func compareSet(currentSet []string, newSet []string) (add []string, remove []string, keep []string) {
	sort.Strings(currentSet)
	sort.Strings(newSet)

	i, j := 0, 0
	for i < len(currentSet) || j < len(newSet) {
		if i < len(currentSet) && j < len(newSet) {
			if currentSet[i] == newSet[j] {
				keep = append(keep, currentSet[i])
				i += 1
				j += 1
			} else if currentSet[i] < newSet[j] {
				remove = append(remove, currentSet[i])
				i += 1
			} else {
				add = append(add, newSet[j])
				j += 1
			}
		} else if i >= len(currentSet) {
			add = append(add, newSet[j])
			j += 1
		} else if j >= len(newSet) {
			remove = append(remove, currentSet[i])
			i += 1
		}
	}

	return add, remove, keep
}

// 启动盘 != 系统盘(必须是启动盘且挂载在root device上)
func isBootDisk(server *SInstance, disk *SDisk) bool {
	if disk.GetDiskType() != api.DISK_TYPE_SYS {
		return false
	}

	for _, attachment := range disk.Attachments {
		if attachment.ServerID == server.GetId() && attachment.Device == server.OSEXTSRVATTRRootDeviceName {
			return true
		}
	}

	return false
}

func (self *SInstance) GetId() string {
	return self.ID
}

func (self *SInstance) GetHostname() string {
	return self.OSEXTSRVATTRHostname
}

func (self *SInstance) GetName() string {
	return self.Name
}

func (self *SInstance) GetGlobalId() string {
	return self.ID
}

func (self *SInstance) GetStatus() string {
	switch self.Status {
	case "ACTIVE":
		return api.VM_RUNNING
	case "MIGRATING", "REBUILD", "BUILD", "RESIZE", "VERIFY_RESIZE": // todo: pending ?
		return api.VM_STARTING
	case "REBOOT", "HARD_REBOOT":
		return api.VM_STOPPING
	case "SHUTOFF":
		return api.VM_READY
	default:
		return api.VM_UNKNOWN
	}
}

func (ins *SInstance) GetPowerStates() string {
	switch ins.OSEXTSTSPowerState {
	case 1:
		return api.VM_POWER_STATES_ON
	default:
		return api.VM_POWER_STATES_OFF
	}
}

func (self *SInstance) Refresh() error {
	vm, err := self.host.zone.region.GetInstance(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, vm)
}

func (self *SInstance) GetInstanceType() string {
	return self.Flavor.ID
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	ret := []string{}
	for _, sec := range self.SecurityGroups {
		ret = append(ret, sec.Id)
	}
	return ret, nil
}

// key 相同时value不会替换
// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=BatchCreateServerTags
func (self *SRegion) CreateServerTags(instanceId string, tags map[string]string) error {
	params := map[string]interface{}{
		"action": "create",
	}
	tagsObj := []map[string]string{}
	for k, v := range tags {
		tagsObj = append(tagsObj, map[string]string{"key": k, "value": v})
	}
	params["tags"] = tagsObj
	_, err := self.post(SERVICE_ECS, fmt.Sprintf("cloudservers/%s/tags/action", instanceId), params)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=BatchDeleteServerTags
func (self *SRegion) DeleteServerTags(instanceId string, tagsKey []string) error {
	params := map[string]interface{}{
		"action": "delete",
	}
	tagsObj := []map[string]string{}
	for _, k := range tagsKey {
		tagsObj = append(tagsObj, map[string]string{"key": k})
	}
	params["tags"] = tagsObj
	_, err := self.post(SERVICE_ECS, fmt.Sprintf("cloudservers/%s/tags/action", instanceId), params)
	return err
}

func (self *SInstance) SetTags(tags map[string]string, replace bool) error {
	existedTags, err := self.GetTags()
	if err != nil {
		return errors.Wrap(err, "self.GetTags()")
	}
	deleteTagsKey := []string{}
	for k := range existedTags {
		if replace {
			deleteTagsKey = append(deleteTagsKey, k)
		} else {
			if _, ok := tags[k]; ok {
				deleteTagsKey = append(deleteTagsKey, k)
			}
		}
	}
	if len(deleteTagsKey) > 0 {
		err := self.host.zone.region.DeleteServerTags(self.GetId(), deleteTagsKey)
		if err != nil {
			return errors.Wrapf(err, "DeleteServerTags(%s,%s)", self.GetId(), deleteTagsKey)
		}
	}
	if len(tags) > 0 {
		err := self.host.zone.region.CreateServerTags(self.GetId(), tags)
		if err != nil {
			return errors.Wrapf(err, "CreateServerTags(%s,%s)", self.GetId(), jsonutils.Marshal(tags).String())
		}
	}
	return nil
}

func (self *SInstance) GetBillingType() string {
	if self.Metadata.ChargingMode == "1" {
		return billing_api.BILLING_TYPE_PREPAID
	}
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SInstance) GetCreatedAt() time.Time {
	return self.Created
}

func (self *SInstance) GetDescription() string {
	return self.Description
}

func (self *SInstance) GetExpiredAt() time.Time {
	orders, err := self.host.zone.region.client.GetOrderResources()
	if err != nil {
		return time.Time{}
	}
	order, ok := orders[self.ID]
	if ok {
		return order.ExpireTime
	}
	return time.Time{}
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	attached := self.OSExtendedVolumesVolumesAttached
	disks := make([]SDisk, 0)
	for _, vol := range attached {
		disk, err := self.host.zone.region.GetDisk(vol.ID)
		if err != nil {
			return nil, err
		}

		disks = append(disks, *disk)
	}

	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i += 1 {
		storage, err := self.host.zone.getStorageByCategory(disks[i].VolumeType)
		if err != nil {
			return nil, err
		}
		disks[i].storage = storage
		idisks[i] = &disks[i]
		// 将系统盘放到第0个位置
		if isBootDisk(self, &disks[i]) {
			_temp := idisks[0]
			idisks[0] = &disks[i]
			idisks[i] = _temp
		}
	}
	return idisks, nil
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	ret := map[string]*SInstanceNic{}
	for _, ipAddresses := range self.Addresses {
		for _, ipAddress := range ipAddresses {
			if ipAddress.OSEXTIPSType == "fixed" && ipAddress.Version == "4" {
				_, ok := ret[ipAddress.OSEXTIPSMACMACAddr]
				if !ok {
					ret[ipAddress.OSEXTIPSMACMACAddr] = &SInstanceNic{
						instance: self,
						ipAddr:   ipAddress.Addr,
						macAddr:  ipAddress.OSEXTIPSMACMACAddr,
						subAddrs: []string{},
					}
				} else {
					addrs := append(ret[ipAddress.OSEXTIPSMACMACAddr].subAddrs, ipAddress.Addr)
					ret[ipAddress.OSEXTIPSMACMACAddr].subAddrs = addrs
				}
			}
		}
	}
	nics := make([]cloudprovider.ICloudNic, 0)
	for mac := range ret {
		nics = append(nics, ret[mac])
	}
	return nics, nil
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	ips := make([]string, 0)
	for _, addresses := range self.Addresses {
		for _, address := range addresses {
			if address.OSEXTIPSType != "fixed" && !strings.HasPrefix(address.Addr, "100.") {
				ips = append(ips, address.Addr)
			}
		}
	}

	if len(ips) == 0 {
		return nil, nil
	}

	eips, err := self.host.zone.region.GetEips("", ips)
	if err != nil {
		return nil, err
	}
	if len(eips) > 0 {
		return &eips[0], nil
	}
	return nil, nil
}

func (self *SInstance) GetVcpuCount() int {
	cpu, _ := strconv.Atoi(self.Flavor.Vcpus)
	return cpu
}

func (self *SInstance) GetVmemSizeMB() int {
	mem, _ := strconv.Atoi(self.Flavor.RAM)
	return int(mem)
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

func (i *SInstance) getImage() *SImage {
	if i.image == nil && len(i.Image.ID) > 0 {
		image, err := i.host.zone.region.GetImage(i.Image.ID)
		if err == nil {
			i.image = image
		} else {
			log.Debugf("GetOSArch.GetImage %s: %s", i.Image.ID, err)
		}
	}
	return i.image
}

func (self *SInstance) GetOsArch() string {
	img := self.getImage()
	if img != nil {
		return img.GetOsArch()
	}

	t := self.GetInstanceType()
	if len(t) > 0 {
		if strings.HasPrefix(t, "k") {
			return apis.OS_ARCH_AARCH64
		}
	}

	return apis.OS_ARCH_X86
}

func (self *SInstance) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(osprofile.NormalizeOSType(self.Metadata.OSType))
}

func (self *SInstance) GetFullOsName() string {
	return self.Metadata.ImageName
}

func (self *SInstance) GetBios() cloudprovider.TBiosType {
	img := self.getImage()
	if img != nil {
		return img.GetBios()
	}
	return cloudprovider.BIOS
}

func (self *SInstance) GetOsDist() string {
	img := self.getImage()
	if img != nil {
		return img.GetOsDist()
	}
	return ""
}

func (self *SInstance) GetOsVersion() string {
	img := self.getImage()
	if img != nil {
		return img.GetOsVersion()
	}
	return ""
}

func (self *SInstance) GetOsLang() string {
	img := self.getImage()
	if img != nil {
		return img.GetOsLang()
	}
	return ""
}

func (self *SInstance) GetMachine() string {
	return "pc"
}

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	ids, err := self.GetSecurityGroupIds()
	if err != nil {
		return err
	}

	add, remove, _ := compareSet(ids, secgroupIds)
	for _, id := range add {
		err = self.host.zone.region.assignSecurityGroup(self.GetId(), id)
		if err != nil {
			return err
		}
	}
	for _, id := range remove {
		err := self.host.zone.region.unassignSecurityGroups(self.GetId(), id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_HUAWEI
}

func (self *SInstance) StartVM(ctx context.Context) error {
	return self.host.zone.region.StartVM(self.GetId())
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	return self.host.zone.region.StopVM(self.GetId(), opts.IsForce)
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return self.host.zone.region.DeleteVM(self.GetId())
}

func (self *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	return self.host.zone.region.UpdateVM(self.GetId(), input)
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=ChangeServerOsWithoutCloudInit
func (self *SRegion) RebuildRoot(instanceId string, opts *cloudprovider.SManagedVMRebuildRootConfig) (jsonutils.JSONObject, error) {
	var err error

	keyName := ""
	if len(opts.PublicKey) > 0 {
		keyName, err = self.syncKeypair(opts.PublicKey)
		if err != nil {
			return nil, err
		}
	}
	params := map[string]interface{}{
		"imageid": opts.ImageId,
	}
	if len(opts.Password) > 0 {
		params["adminpass"] = opts.Password
	}
	if len(keyName) > 0 {
		params["keyname"] = keyName
	}
	return self.post(SERVICE_ECS, fmt.Sprintf("cloudservers/%s/changeos", instanceId), map[string]interface{}{"os-change": params})
}

func (self *SInstance) RebuildRoot(ctx context.Context, opts *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	resp, err := self.host.zone.region.RebuildRoot(self.ID, opts)
	if err != nil {
		return "", errors.Wrapf(err, "change os")
	}
	jobId, err := resp.GetString("job_id")
	if err != nil {
		return "", errors.Wrapf(err, "get job_id")
	}

	err = self.host.zone.region.waitTaskStatus(SERVICE_ECS, jobId, TASK_SUCCESS, 15*time.Second, 900*time.Second)
	if err != nil {
		return "", errors.Wrapf(err, "wait task")
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

func (self *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	return self.host.zone.region.DeployVM(self.GetId(), opts)
}

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	instanceTypes := []string{}
	if len(config.InstanceType) > 0 {
		instanceTypes = []string{config.InstanceType}
	} else {
		flavors, err := self.host.zone.region.GetMatchInstanceTypes(config.Cpu, config.MemoryMB, self.OSEXTAZAvailabilityZone)
		if err != nil {
			return errors.Wrapf(err, "GetMatchInstanceTypes")
		}
		for _, flavor := range flavors {
			instanceTypes = append(instanceTypes, flavor.ID)
		}
	}
	var err error
	for _, instanceType := range instanceTypes {
		err = self.host.zone.region.ChangeVMConfig(self.GetId(), instanceType)
		if err != nil {
			log.Warningf("ChangeVMConfig %s for %s error: %v", self.GetId(), instanceType, err)
		} else {
			return cloudprovider.WaitStatusWithDelay(self, api.VM_READY, 15*time.Second, 15*time.Second, 180*time.Second)
		}
	}
	if err != nil {
		return errors.Wrapf(err, "ChangeVMConfig")
	}
	return fmt.Errorf("Failed to change vm config, specification not supported")
}

func (self *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return self.host.zone.region.GetInstanceVNCUrl(self.GetId())
}

func (self *SInstance) NextDeviceName() (string, error) {
	prefix := "s"
	if strings.Contains(self.OSEXTSRVATTRRootDeviceName, "/vd") {
		prefix = "v"
	}

	currents := []string{}
	for _, item := range self.OSExtendedVolumesVolumesAttached {
		currents = append(currents, strings.ToLower(item.Device))
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
		return errors.Wrap(err, "NextDeviceName")
	}

	return self.host.zone.region.AttachDisk(self.GetId(), diskId, device)
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	for _, disk := range self.OSExtendedVolumesVolumesAttached {
		if disk.ID == diskId {
			return self.host.zone.region.DetachDisk(self.GetId(), diskId)
		}
	}
	return nil
}

func (self *SInstance) Renew(bc billing.SBillingCycle) error {
	return self.host.zone.region.RenewInstance(self.GetId(), bc)
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=ListServersDetails
func (self *SRegion) GetInstances(ip string) ([]SInstance, error) {
	params := url.Values{}
	params.Set("limit", "1000")
	if len(ip) > 0 {
		params.Set("ip", ip)
	}
	ret := []SInstance{}
	for {
		resp, err := self.list(SERVICE_ECS, "cloudservers/detail", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Servers []SInstance
			Count   int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Servers...)
		if len(ret) >= part.Count || len(part.Servers) == 0 {
			break
		}
		params.Set("offset", fmt.Sprintf("%d", len(ret)))
	}
	return ret, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=ShowServer
func (self *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	resp, err := self.list(SERVICE_ECS, "cloudservers/"+instanceId, nil)
	if err != nil {
		return nil, err
	}
	ret := &SInstance{}
	err = resp.Unmarshal(ret, "server")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=CreateServers
func (self *SRegion) CreateInstance(keypair, zoneId string, opts *cloudprovider.SManagedVMCreateConfig) (string, error) {
	secgroups := []map[string]interface{}{}
	for _, id := range opts.ExternalSecgroupIds {
		secgroups = append(secgroups, map[string]interface{}{
			"id": id,
		})
	}
	dataDisks := []map[string]interface{}{}
	for _, disk := range opts.DataDisks {
		dataDisks = append(dataDisks, map[string]interface{}{
			"volumetype": disk.StorageType,
			"size":       disk.SizeGB,
		})
	}
	extendparam := map[string]interface{}{
		"regionID":     self.getId(),
		"chargingMode": POST_PAID,
	}
	if len(opts.ProjectId) > 0 {
		extendparam["enterprise_project_id"] = opts.ProjectId
	}

	// billing type
	if opts.BillingCycle != nil {
		extendparam["chargingMode"] = PRE_PAID
		extendparam["isAutoRenew"] = opts.BillingCycle.AutoRenew
		extendparam["isAutoPay"] = true
		if opts.BillingCycle.GetMonths() <= 9 {
			extendparam["periodNum"] = opts.BillingCycle.GetMonths()
			extendparam["periodType"] = "month"
		} else {
			extendparam["periodNum"] = opts.BillingCycle.GetYears()
			extendparam["periodType"] = "year"
		}
	}

	params := map[string]interface{}{
		"availability_zone": zoneId,
		"name":              opts.Name,
		"flavorRef":         opts.InstanceType,
		"imageRef":          opts.ExternalImageId,
		"count":             1,
		"nics": []map[string]interface{}{
			{
				"subnet_id":  opts.ExternalNetworkId,
				"ip_address": opts.IpAddr,
			},
		},
		"security_groups": secgroups,
		"vpcid":           opts.ExternalVpcId,
		"root_volume": map[string]interface{}{
			"volumetype": opts.SysDisk.StorageType,
			"size":       opts.SysDisk.SizeGB,
		},
		"data_volumes": dataDisks,
		"extendparam":  extendparam,
	}

	if len(keypair) > 0 {
		params["key_name"] = keypair
	} else {
		params["adminPass"] = opts.Password
	}

	if len(opts.UserData) > 0 {
		params["user_data"] = opts.UserData
	}

	if len(opts.Tags) > 0 {
		serverTags := []map[string]interface{}{}
		for k, v := range opts.Tags {
			serverTags = append(serverTags, map[string]interface{}{
				"key":   k,
				"value": v,
			})
		}
		params["server_tags"] = serverTags
	}

	resp, err := self.post(SERVICE_ECS_V1_1, "cloudservers", map[string]interface{}{"server": params})
	if err != nil {
		return "", err
	}

	ret := struct {
		JobId     string
		OrderId   string
		ServerIds []string `json:"serverIds"`
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return "", err
	}
	for _, id := range ret.ServerIds {
		return id, nil
	}
	if len(ret.JobId) > 0 {
		ids, err := self.GetAllSubTaskEntityIDs(SERVICE_ECS, ret.JobId)
		if err != nil {
			return "", errors.Wrapf(err, "GetAllSubTaskEntityIDs(%s)", ret.JobId)
		}
		for _, id := range ids {
			return id, nil
		}
	}
	return "", fmt.Errorf("create server return empyte response %s", resp.String())
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=NovaAssociateSecurityGroup
func (self *SRegion) assignSecurityGroup(instanceId, secgroupId string) error {
	_, err := self.post(SERVICE_ECS_V2_1, fmt.Sprintf("servers/%s/action", instanceId), map[string]interface{}{
		"addSecurityGroup": map[string]interface{}{
			"name": secgroupId,
		},
	})
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=NovaDisassociateSecurityGroup
func (self *SRegion) unassignSecurityGroups(instanceId, secgroupId string) error {
	_, err := self.post(SERVICE_ECS_V2_1, fmt.Sprintf("servers/%s/action", instanceId), map[string]interface{}{
		"removeSecurityGroup": map[string]interface{}{
			"name": secgroupId,
		},
	})
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=NovaStartServer
func (self *SRegion) StartVM(instanceId string) error {
	params := map[string]interface{}{
		"os-start": map[string]string{},
	}
	_, err := self.post(SERVICE_ECS_V2_1, fmt.Sprintf("servers/%s/action", instanceId), params)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=NovaStopServer
func (self *SRegion) StopVM(instanceId string, isForce bool) error {
	stopType := "SOFT"
	if isForce {
		stopType = "HARD"
	}
	params := map[string]interface{}{
		"os-stop": map[string]string{
			"type": stopType,
		},
	}
	_, err := self.post(SERVICE_ECS_V2_1, fmt.Sprintf("servers/%s/action", instanceId), params)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=DeleteServers
func (self *SRegion) DeleteVM(instanceId string) error {
	params := map[string]interface{}{
		"servers": []map[string]interface{}{
			{
				"id": instanceId,
			},
		},
	}
	_, err := self.post(SERVICE_ECS, "cloudservers/delete", params)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=UpdateServer
func (self *SRegion) UpdateVM(instanceId string, input cloudprovider.SInstanceUpdateOptions) error {
	params := map[string]interface{}{
		"server": map[string]interface{}{
			"name":        input.NAME,
			"description": input.Description,
		},
	}
	_, err := self.put(SERVICE_ECS, "cloudservers/"+instanceId, params)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=BatchResetServersPassword
func (self *SRegion) BatchResetServersPassword(instanceId string, password string) error {
	params := map[string]interface{}{
		"new_password": password,
		"servers": []map[string]interface{}{
			{
				"id": instanceId,
			},
		},
	}
	_, err := self.put(SERVICE_ECS, "cloudservers/os-reset-passwords", params)
	if err != nil {
		return errors.Wrapf(err, "reset password")
	}
	return nil
}

func (self *SRegion) DeployVM(instanceId string, opts *cloudprovider.SInstanceDeployOptions) error {
	if len(opts.Password) > 0 {
		err := self.BatchResetServersPassword(instanceId, opts.Password)
		if err != nil {
			return errors.Wrapf(err, "BatchResetServersPassword")
		}
	}
	return nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=ResizeServer
func (self *SRegion) ChangeVMConfig(instanceId string, instanceType string) error {
	params := map[string]interface{}{
		"resize": map[string]interface{}{
			"flavorRef": instanceType,
		},
	}
	_, err := self.post(SERVICE_ECS_V1_1, fmt.Sprintf("cloudservers/%s/resize", instanceId), params)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=ShowServerRemoteConsole
func (self *SRegion) GetInstanceVNCUrl(instanceId string) (*cloudprovider.ServerVncOutput, error) {
	params := map[string]interface{}{
		"remote_console": map[string]interface{}{
			"type":     "novnc",
			"protocol": "vnc",
		},
	}
	resp, err := self.post(SERVICE_ECS, fmt.Sprintf("cloudservers/%s/remote_console", instanceId), params)
	if err != nil {
		return nil, err
	}
	result := &cloudprovider.ServerVncOutput{
		Hypervisor: api.HYPERVISOR_HUAWEI,
	}
	resp.Unmarshal(result, "remote_console")
	result.Protocol = api.HYPERVISOR_HUAWEI
	return result, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=AttachServerVolume
func (self *SRegion) AttachDisk(instanceId string, diskId string, device string) error {
	params := map[string]interface{}{
		"volumeAttachment": map[string]interface{}{
			"volumeId": diskId,
			"device":   device,
		},
	}
	_, err := self.post(SERVICE_ECS, fmt.Sprintf("cloudservers/%s/attachvolume", instanceId), params)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=DetachServerVolume
func (self *SRegion) DetachDisk(instanceId string, diskId string) error {
	_, err := self.delete(SERVICE_ECS, fmt.Sprintf("cloudservers/%s/detachvolume/%s", instanceId, diskId))
	//volume a2091934-2669-4fca-8eb4-a950c1836b3c is not in server 49b053d2-f798-432f-af55-76eb6ef2c769 attach volume list => 磁盘已经被卸载了
	if err != nil && strings.Contains(err.Error(), fmt.Sprintf("is not in server")) && strings.Contains(err.Error(), fmt.Sprintf("attach volume list")) {
		return nil
	}
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/BSS/doc?api=RenewalResources
func (self *SRegion) RenewInstance(instanceId string, bc billing.SBillingCycle) error {
	params := map[string]interface{}{
		"resource_ids":  []string{instanceId},
		"expire_policy": 3,
		"is_auto_pay":   1,
	}

	month := int64(bc.GetMonths())
	year := int64(bc.GetYears())

	if month >= 1 && month <= 11 {
		params["period_type"] = 2
		params["period_num"] = month
	} else if year >= 1 && year <= 3 {
		params["period_type"] = 3
		params["period_num"] = year
	} else {
		return fmt.Errorf("invalid renew period %d month,must be 1~11 month or 1~3 year", month)
	}
	_, err := self.post(SERVICE_BSS, "orders/subscriptions/resources/renew", params)
	return err
}

func (self *SInstance) GetProjectId() string {
	return self.EnterpriseProjectId
}

func (self *SInstance) GetError() error {
	return nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IMS/doc?api=CreateImage
func (self *SRegion) SaveImage(instanceId string, opts *cloudprovider.SaveImageOptions) (*SImage, error) {
	params := map[string]interface{}{
		"name":        opts.Name,
		"instance_id": instanceId,
	}
	if len(opts.Notes) > 0 {
		params["description"] = func() string {
			opts.Notes = strings.ReplaceAll(opts.Notes, "<", "")
			opts.Notes = strings.ReplaceAll(opts.Notes, ">", "")
			opts.Notes = strings.ReplaceAll(opts.Notes, "\n", "")
			if len(opts.Notes) > 1024 {
				opts.Notes = opts.Notes[:1024]
			}
			return opts.Notes
		}()
	}
	resp, err := self.post(SERVICE_IMS, "cloudimages/action", params)
	if err != nil {
		return nil, errors.Wrapf(err, "crate image")
	}
	jobId, err := resp.GetString("job_id")
	if err != nil {
		return nil, errors.Wrapf(err, "get job_id")
	}
	err = self.waitTaskStatus(SERVICE_IMS_V1, jobId, TASK_SUCCESS, 15*time.Second, 10*time.Minute)
	if err != nil {
		return nil, errors.Wrapf(err, "waitTaskStatus")
	}
	imageId, err := self.GetTaskEntityID(SERVICE_IMS, jobId, "image_id")
	if err != nil {
		return nil, errors.Wrapf(err, "GetTaskEntityID")
	}
	image, err := self.GetImage(imageId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetImage(%s)", imageId)
	}
	image.storageCache = self.getStoragecache()
	return image, nil
}

func (self *SInstance) SaveImage(opts *cloudprovider.SaveImageOptions) (cloudprovider.ICloudImage, error) {
	image, err := self.host.zone.region.SaveImage(self.ID, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "SaveImage")
	}
	return image, nil
}
