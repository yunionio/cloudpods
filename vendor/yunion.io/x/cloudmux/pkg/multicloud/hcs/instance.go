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

package hcs

import (
	"context"
	"encoding/base64"
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
	"yunion.io/x/pkg/util/cloudinit"
	"yunion.io/x/pkg/util/imagetools"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/apis"
	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei"
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
	OSEXTIPSPortId     string `json:"OS-EXT-IPS:port_id"`
	OSEXTIPSType       string `json:"OS-EXT-IPS:type"`
}

type Flavor struct {
	Disk  string `json:"disk"`
	Vcpus string `json:"vcpus"`
	RAM   string `json:"ram"`
	Id    string `json:"id"`
	Name  string `json:"name"`
}

type Image struct {
	Id string `json:"id"`
}

type VMMetadata struct {
	MeteringImageId           string `json:"metering.image_id"`
	MeteringImagetype         string `json:"metering.imagetype"`
	MeteringOrderId           string `json:"metering.order_id"`
	MeteringResourcespeccode  string `json:"metering.resourcespeccode"`
	ImageName                 string `json:"image_name"`
	OSBit                     string `json:"os_bit"`
	VpcId                     string `json:"vpc_id"`
	MeteringResourcetype      string `json:"metering.resourcetype"`
	CascadedInstanceExtrainfo string `json:"cascaded.instance_extrainfo"`
	OSType                    string `json:"os_type"`
	ChargingMode              string `json:"charging_mode"`
}

type OSExtendedVolumesVolumesAttached struct {
	Device              string `json:"device"`
	BootIndex           string `json:"bootIndex"`
	Id                  string `json:"id"`
	DeleteOnTermination string `json:"delete_on_termination"`
}

type OSSchedulerHints struct {
}

type SecurityGroup struct {
	Name string `json:"name"`
}

type SysTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0094148849.html
// https://support.huaweicloud.com/api-bpconsole/zh-cn_topic_0100166287.html v1.1 支持创建包年/包月的弹性云服务器
type SInstance struct {
	multicloud.SInstanceBase
	huawei.HuaweiTags

	host *SHost

	osInfo *imagetools.ImageInfo

	Id          string                 `json:"id"`
	Name        string                 `json:"name"`
	Addresses   map[string][]IpAddress `json:"addresses"`
	Flavor      Flavor                 `json:"flavor"`
	AccessIPv4  string                 `json:"accessIPv4"`
	AccessIPv6  string                 `json:"accessIPv6"`
	Status      string                 `json:"status"`
	Progress    string                 `json:"progress"`
	HostId      string                 `json:"hostId"`
	Image       Image                  `json:"image"`
	Updated     string                 `json:"updated"`
	Created     time.Time              `json:"created"`
	Metadata    VMMetadata             `json:"metadata"`
	Description string                 `json:"description"`
	Locked      bool                   `json:"locked"`
	ConfigDrive string                 `json:"config_drive"`
	TenantId    string                 `json:"tenant_id"`
	UserId      string                 `json:"user_id"`
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
	OSEXTSRVATTRRamdiskId            string                             `json:"OS-EXT-SRV-ATTR:ramdisk_id"`
	EnterpriseProjectId              string                             `json:"enterprise_project_id"`
	OSEXTSRVATTRUserData             string                             `json:"OS-EXT-SRV-ATTR:user_data"`
	OSSRVUSGLaunchedAt               time.Time                          `json:"OS-SRV-USG:launched_at"`
	OSEXTSRVATTRKernelId             string                             `json:"OS-EXT-SRV-ATTR:kernel_id"`
	OSEXTSRVATTRLaunchIndex          int64                              `json:"OS-EXT-SRV-ATTR:launch_index"`
	HostStatus                       string                             `json:"host_status"`
	OSEXTSRVATTRReservationId        string                             `json:"OS-EXT-SRV-ATTR:reservation_id"`
	OSEXTSRVATTRHostname             string                             `json:"OS-EXT-SRV-ATTR:hostname"`
	OSSRVUSGTerminatedAt             time.Time                          `json:"OS-SRV-USG:terminated_at"`
	SysTags                          []SysTag                           `json:"sys_tags"`
	SecurityGroups                   []SecurityGroup                    `json:"security_groups"`
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
		if attachment.ServerId == server.GetId() && attachment.Device == server.OSEXTSRVATTRRootDeviceName {
			return true
		}
	}

	return false
}

func (self *SInstance) GetId() string {
	return self.Id
}

func (self *SInstance) GetHostname() string {
	return self.OSEXTSRVATTRHostname
}

func (self *SInstance) GetName() string {
	return self.Name
}

func (self *SInstance) GetGlobalId() string {
	return self.Id
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
	ret, err := self.host.zone.region.GetInstance(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, ret)
}

func (self *SInstance) GetInstanceType() string {
	return self.Flavor.Id
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	secgroups, err := self.host.zone.region.GetInstanceSecrityGroups(self.Id)
	if err != nil {
		return nil, err
	}
	ret := []string{}
	for i := range secgroups {
		ret = append(ret, secgroups[i].Id)
	}
	return ret, nil
}

// https://support.huaweicloud.com/api-ecs/ecs_02_1002.html
// key 相同时value不会替换
func (self *SRegion) CreateServerTags(instanceId string, tags map[string]string) error {
	params := map[string]interface{}{
		"action": "create",
	}

	tagsObj := []map[string]string{}
	for k, v := range tags {
		tagsObj = append(tagsObj, map[string]string{"key": k, "value": v})
	}
	params["tags"] = tagsObj

	res := fmt.Sprintf("cloudservers/%s", instanceId)
	return self.perform("ecs", "v1", res, "tags/action", params, nil)
}

// https://support.huaweicloud.com/api-ecs/ecs_02_1003.html
func (self *SRegion) DeleteServerTags(instanceId string, tagsKey []string) error {
	params := map[string]interface{}{
		"action": "delete",
	}
	tagsObj := []map[string]string{}
	for _, k := range tagsKey {
		tagsObj = append(tagsObj, map[string]string{"key": k})
	}
	params["tags"] = tagsObj
	res := fmt.Sprintf("cloudservers/%s", instanceId)
	return self.perform("ecs", "v1", res, "tags/action", params, nil)
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
			return errors.Wrapf(err, "self.host.zone.region.DeleteServerTags(%s,%s)", self.GetId(), deleteTagsKey)
		}
	}
	if len(tags) > 0 {
		err := self.host.zone.region.CreateServerTags(self.GetId(), tags)
		if err != nil {
			return errors.Wrapf(err, "self.host.zone.region.CreateServerTags(%s,%s)", self.GetId(), jsonutils.Marshal(tags).String())
		}
	}
	return nil
}

func (self *SInstance) GetBillingType() string {
	// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0094148849.html
	// charging_mode “0”：按需计费    “1”：按包年包月计费
	if self.Metadata.ChargingMode == "1" {
		return billing_api.BILLING_TYPE_PREPAID
	}
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SInstance) GetCreatedAt() time.Time {
	return self.Created
}

// charging_mode “0”：按需计费  “1”：按包年包月计费
func (self *SInstance) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	attached := self.OSExtendedVolumesVolumesAttached
	disks := make([]SDisk, 0)
	for _, vol := range attached {
		disk, err := self.host.zone.region.GetDisk(vol.Id)
		if err != nil {
			return nil, err
		}
		disk.storage = &SStorage{zone: self.host.zone, storageType: disk.VolumeType}
		disks = append(disks, *disk)
	}

	ret := []cloudprovider.ICloudDisk{}
	for i := 0; i < len(disks); i += 1 {
		disks[i].region = self.host.zone.region
		if isBootDisk(self, &disks[i]) {
			ret = append([]cloudprovider.ICloudDisk{&disks[i]}, ret...)
		} else {
			ret = append(ret, &disks[i])
		}
	}
	return ret, nil
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics := make([]cloudprovider.ICloudNic, 0)
	for _, ipAddresses := range self.Addresses {
		for _, ipAddress := range ipAddresses {
			if ipAddress.OSEXTIPSType == "fixed" {
				nic := SInstanceNic{
					instance: self,
					ipAddr:   ipAddress.Addr,
					macAddr:  ipAddress.OSEXTIPSMACMACAddr,
				}
				nics = append(nics, &nic)
			}
		}
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
		eips[0].region = self.host.zone.region
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

func (self *SInstance) GetOSArch() string {
	if len(self.Image.Id) > 0 {
		image, err := self.host.zone.region.GetImage(self.Image.Id)
		if err == nil {
			return image.GetOsArch()
		}

		log.Debugf("GetOSArch.GetImage %s: %s", self.Image.Id, err)
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

func (self *SInstance) GetOSName() string {
	return self.Metadata.ImageName
}

func (i *SInstance) getNormalizedOsInfo() *imagetools.ImageInfo {
	if i.osInfo == nil {
		osInfo := imagetools.NormalizeImageInfo(i.Metadata.ImageName, "", i.Metadata.OSType, "", "")
		i.osInfo = &osInfo
	}
	return i.osInfo
}

func (self *SInstance) GetBios() cloudprovider.TBiosType {
	return cloudprovider.BIOS
}

func (i *SInstance) GetOsDist() string {
	return i.getNormalizedOsInfo().OsDistro
}

func (i *SInstance) GetOsVersion() string {
	return i.getNormalizedOsInfo().OsVersion
}

func (i *SInstance) GetOsLang() string {
	return i.getNormalizedOsInfo().OsLang
}

func (self *SInstance) GetFullOsName() string {
	return self.Metadata.ImageName
}

func (self *SInstance) GetMachine() string {
	return "pc"
}

func (self *SInstance) GetOsArch() string {
	image, err := self.host.zone.region.GetImage(self.Image.Id)
	if err == nil {
		return image.GetOsArch()
	}
	flavor, err := self.host.zone.region.GetICloudSku(self.Flavor.Id)
	if err == nil {
		return flavor.GetCpuArch()
	}

	t := self.GetInstanceType()
	if len(t) > 0 {
		if strings.HasPrefix(t, "k") {
			return apis.OS_ARCH_AARCH64
		}
	}

	return apis.OS_ARCH_X86
}

func (self *SInstance) AssignSecurityGroup(secgroupId string) error {
	return self.SetSecurityGroups([]string{secgroupId})
}

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	ports, err := self.host.zone.region.GetPorts(self.Id)
	if err != nil {
		return errors.Wrapf(err, "GetPorts")
	}
	for i := range ports {
		return self.host.zone.region.SetSecurityGroups(secgroupIds, ports[i].Id)
	}
	return nil
}

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_HCS
}

func (self *SInstance) StartVM(ctx context.Context) error {
	if self.Status == InstanceStatusRunning {
		return nil
	}

	timeout := 300 * time.Second
	interval := 15 * time.Second

	startTime := time.Now()
	for time.Now().Sub(startTime) < timeout {
		err := self.Refresh()
		if err != nil {
			return err
		}

		if self.GetStatus() == api.VM_RUNNING {
			return nil
		} else if self.GetStatus() == api.VM_READY {
			err := self.host.zone.region.StartVM(self.GetId())
			if err != nil {
				return err
			}
		}
		time.Sleep(interval)
	}
	return cloudprovider.ErrTimeout
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	if self.Status == InstanceStatusStopped {
		return nil
	}

	if self.Status == InstanceStatusTerminated {
		log.Debugf("Instance already terminated.")
		return nil
	}

	err := self.host.zone.region.StopVM(self.GetId(), opts.IsForce)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, api.VM_READY, 10*time.Second, 300*time.Second) // 5mintues
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return self.host.zone.region.DeleteVM(self.GetId())
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	return self.host.zone.region.UpdateVM(self.GetId(), name)
}

// https://support.huaweicloud.com/usermanual-ecs/zh-cn_topic_0032380449.html
// 创建云服务器过程中注入用户数据。支持注入文本、文本文件或gzip文件。
// 注入内容，需要进行base64格式编码。注入内容（编码之前的内容）最大长度32KB。
// 对于Linux弹性云服务器，adminPass参数传入时，user_data参数不生效。
func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0067876349.html 使用原镜像重装
// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0067876971.html 更换系统盘操作系统
// 不支持调整系统盘大小
func (self *SInstance) RebuildRoot(ctx context.Context, desc *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	publicKeyName := ""
	var err error
	if len(desc.PublicKey) > 0 {
		publicKeyName, err = self.host.zone.region.syncKeypair(desc.PublicKey)
		if err != nil {
			return "", err
		}
	}

	image, err := self.host.zone.region.GetImage(desc.ImageId)
	if err != nil {
		return "", errors.Wrap(err, "SInstance.RebuildRoot.GetImage")
	}

	// Password存在的情况下，windows 系统直接使用密码
	if strings.ToLower(image.Platform) == strings.ToLower(osprofile.OS_TYPE_WINDOWS) && len(desc.Password) > 0 {
		publicKeyName = ""
	}

	userData, err := updateUserData(self.OSEXTSRVATTRUserData, image.OSVersion, desc.Account, desc.Password, desc.PublicKey)
	if err != nil {
		return "", errors.Wrap(err, "SInstance.RebuildRoot.updateUserData")
	}

	if self.Metadata.MeteringImageId == desc.ImageId {
		err = self.host.zone.region.RebuildRoot(ctx, self.UserId, self.GetId(), desc.Password, publicKeyName, userData)
		if err != nil {
			return "", err
		}
	} else {
		err = self.host.zone.region.ChangeRoot(ctx, self.UserId, self.GetId(), desc.ImageId, desc.Password, publicKeyName, userData)
		if err != nil {
			return "", err
		}
	}

	err = self.Refresh()
	if err != nil {
		return "", errors.Wrapf(err, "Refresh")
	}

	idisks, err := self.GetIDisks()
	if err != nil {
		return "", errors.Wrapf(err, "GetIDisks")
	}

	if len(idisks) == 0 {
		return "", fmt.Errorf("server %s has no volume attached.", self.GetId())
	}

	return idisks[0].GetId(), nil
}

func (self *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	return self.host.zone.region.DeployVM(self.GetId(), name, password, publicKey, deleteKeypair, description)
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
			instanceTypes = append(instanceTypes, flavor.Id)
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

// todo:// 返回jsonobject感觉很诡异。不能直接知道内部细节
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
		return errors.Wrap(err, "Instance.AttachDisk.NextDeviceName")
	}

	err = self.host.zone.region.AttachDisk(self.GetId(), diskId, device)
	if err != nil {
		return errors.Wrap(err, "Instance.AttachDisk.AttachDisk")
	}

	return cloudprovider.Wait(5*time.Second, 60*time.Second, func() (bool, error) {
		disk, err := self.host.zone.region.GetDisk(diskId)
		if err != nil {
			log.Debugf("Instance.AttachDisk.GetDisk %s", err)
			return false, nil
		}

		if disk.Status == "in-use" {
			return true, nil
		}

		return false, nil
	})
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	err := self.host.zone.region.DetachDisk(self.GetId(), diskId)
	if err != nil {
		return errors.Wrap(err, "Instance.DetachDisk")
	}

	return cloudprovider.Wait(5*time.Second, 60*time.Second, func() (bool, error) {
		disk, err := self.host.zone.region.GetDisk(diskId)
		if err != nil {
			log.Debugf("Instance.DetachDisk.GetDisk %s", err)
			return false, nil
		}

		if disk.Status == "available" {
			return true, nil
		}

		return false, nil
	})
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0094148850.html
func (self *SRegion) GetInstances(ip string) ([]SInstance, error) {
	params := url.Values{}
	if len(ip) > 0 {
		params.Set("ip", ip)
	}
	params.Set("offset", "1")
	ret := []SInstance{}
	return ret, self.list("ecs", "v1", "cloudservers/detail", params, &ret)
}

func (self *SRegion) GetInstance(id string) (*SInstance, error) {
	ret := &SInstance{}
	res := fmt.Sprintf("cloudservers/%s", id)
	return ret, self.get("ecs", "v1", res, ret)
}
func (self *SRegion) CreateInstance(name string, imageId string, instanceType string, subnetId string,
	securityGroupId string, vpcId string, zoneId string, desc string, sysDisk cloudprovider.SDiskInfo, dataDisks []cloudprovider.SDiskInfo, ipAddr string,
	keypair string, passwd string, userData string, bc *billing.SBillingCycle, projectId string, tags map[string]string) (*SInstance, error) {
	diskParams := []map[string]interface{}{}
	for _, disk := range dataDisks {
		diskParams = append(diskParams, map[string]interface{}{
			"volumetype": disk.StorageType,
			"size":       disk.SizeGB,
		})
	}
	params := map[string]interface{}{
		"availability_zone": zoneId,
		"name":              name,
		"flavorRef":         instanceType,
		"imageRef":          imageId,
		"description":       desc,
		"count":             1,
		"tenancy":           "0",
		"region_id":         self.Id,
		"operate_type":      "apply",
		"service_type":      "ecs",
		"tenantId":          self.client.projectId,
		"project_id":        self.client.projectId,
		"nics": []map[string]interface{}{
			{
				"subnet_id":  subnetId,
				"ip_address": ipAddr,
				"binding:profile": map[string]interface{}{
					"disable_security_groups": false,
				},
			},
		},
		"security_groups": []map[string]interface{}{
			{
				"id": securityGroupId,
			},
		},
		"vpcid":    vpcId,
		"metadata": map[string]string{},
		"root_volume": map[string]interface{}{
			"volumetype": sysDisk.StorageType,
			"size":       sysDisk.SizeGB,
		},
		"data_volumes": diskParams,
		"extendparam": map[string]interface{}{
			"enterprise_project_id": projectId,
			"regionID":              self.Id,
		},
	}

	if len(keypair) > 0 {
		params["key_name"] = keypair
	} else {
		params["adminPass"] = passwd
	}

	if len(userData) > 0 {
		params["user_data"] = userData
	}
	tagParams := []map[string]interface{}{}
	for k, v := range tags {
		tagParams = append(tagParams, map[string]interface{}{
			"key":   k,
			"value": v,
		})
	}
	params["server_tags"] = tagParams
	body := map[string]interface{}{
		"server": params,
	}
	job := &SJob{}
	err := self.client.create("ecs", "v1.1", self.Id, "cloudservers", body, job)
	if err != nil {
		return nil, err
	}
	for _, sub := range job.Entities.SubJobs {
		if len(sub.Entities.ServerId) > 0 {
			return self.GetInstance(sub.Entities.ServerId)
		}
	}
	return nil, fmt.Errorf("no server id returned with job %s", jsonutils.Marshal(job))
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0067161717.html
func (self *SRegion) SetSecurityGroups(secgroupIds []string, portId string) error {
	res := fmt.Sprintf("ports/%s", portId)
	params := map[string]interface{}{
		"port": map[string]interface{}{
			"security_groups": secgroupIds,
		},
	}
	return self.update("vpc", "v2.0", res, params)
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212207.html
func (self *SRegion) StartVM(id string) error {
	params := map[string]interface{}{
		"os-start": map[string]string{},
	}
	res := fmt.Sprintf("servers/%s", id)
	return self.ecsPerform(res, "action", params, nil)
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212651.html
func (self *SRegion) StopVM(id string, isForce bool) error {
	params := map[string]interface{}{
		"os-stop": map[string]string{},
	}
	res := fmt.Sprintf("servers/%s", id)
	return self.ecsPerform(res, "action", params, nil)
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212679.html
// 只删除主机。弹性IP和数据盘需要单独删除
func (self *SRegion) DeleteVM(id string) error {
	params := map[string]interface{}{
		"delete_publicip": false,
		"delete_volume":   false,
		"servers": []struct {
			Id string
		}{
			{Id: id},
		},
	}
	return self.perform("ecs", "v1", "cloudservers", "delete", params, nil)
}

func (self *SRegion) UpdateVM(instanceId, name string) error {
	params := map[string]interface{}{
		"server": map[string]string{
			"name": name,
		},
	}
	return self.update("ecs", "v2.1", "servers/"+instanceId, params)
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0067876349.html
// 返回job id
func (self *SRegion) RebuildRoot(ctx context.Context, userId, instanceId, passwd, publicKeyName, userData string) error {
	reInstall := map[string]interface{}{}
	if len(publicKeyName) > 0 {
		reInstall["keyname"] = publicKeyName
	} else if len(passwd) > 0 {
		reInstall["adminpass"] = passwd
	} else {
		return fmt.Errorf("both password and publicKey are empty.")
	}
	if len(userId) > 0 {
		reInstall["userid"] = userId
	}
	metadata := map[string]interface{}{}
	if len(userData) > 0 {
		metadata["user_data"] = userData
	}
	reInstall["metadata"] = metadata

	params := map[string]interface{}{
		"os-reinstall": reInstall,
	}
	return self.ecsPerform("cloudservers/"+instanceId, "reinstallos", params, nil)
}

func (self *SRegion) ChangeRoot(ctx context.Context, userId, instanceId, imageId, passwd, publicKeyName, userData string) error {
	osChange := map[string]interface{}{}
	if len(publicKeyName) > 0 {
		osChange["keyname"] = publicKeyName
	} else if len(passwd) > 0 {
		osChange["adminpass"] = passwd
	} else {
		return fmt.Errorf("both password and publicKey are empty.")
	}
	if len(userId) > 0 {
		osChange["userid"] = userId
	}
	metadata := map[string]interface{}{}
	if len(userData) > 0 {
		metadata["user_data"] = userData
	}
	osChange["metadata"] = metadata
	osChange["imageid"] = imageId
	params := map[string]interface{}{
		"os-change": osChange,
	}
	return self.ecsPerform("cloudservers/"+instanceId, "changeos", params, nil)
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212692.html
// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0110109377.html
// 一键式重置密码 需要安装安装一键式重置密码插件 https://support.huaweicloud.com/usermanual-ecs/zh-cn_topic_0068095385.html
// 目前不支持直接重置密钥
func (self *SRegion) DeployVM(instanceId string, name string, password string, keypairName string, deleteKeypair bool, description string) error {
	if len(name) > 0 {
		err := self.UpdateVM(instanceId, name)
		if err != nil {
			return err
		}
	}

	if len(password) > 0 {
		params := map[string]interface{}{
			"reset-password": map[string]interface{}{
				"new_password": password,
			},
		}
		res := fmt.Sprintf("cloudservers/%s/os-reset-password", instanceId)
		return self.update("ecs", "v1", res, params)
	}
	return nil
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212653.html
func (self *SRegion) ChangeVMConfig(instanceId string, instanceType string) error {
	params := map[string]interface{}{
		"resize": map[string]interface{}{
			"flavorRef": instanceType,
		},
	}
	return self.perform("ecs", "v1.1", "cloudservers/"+instanceId, "resize", params, nil)
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0142763126.html 微版本2.6及以上?
// https://support.huaweicloud.com/api-ecs/ecs_02_0208.html
func (self *SRegion) GetInstanceVNCUrl(instanceId string) (*cloudprovider.ServerVncOutput, error) {
	params := map[string]interface{}{
		"remote_console": map[string]interface{}{
			"type":     "novnc",
			"protocol": "vnc",
		},
	}
	ret := struct {
		RemoteConsole cloudprovider.ServerVncOutput
	}{}
	err := self.perform("ecs", "v1", "cloudservers/"+instanceId, "remote_console", params, &ret)
	if err != nil {
		return nil, err
	}
	ret.RemoteConsole.Hypervisor = api.HYPERVISOR_HCS
	ret.RemoteConsole.Protocol = "hcs"
	return &ret.RemoteConsole, nil
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0065817702.html
func (self *SRegion) GetInstanceSecrityGroups(instanceId string) ([]SSecurityGroup, error) {
	ret := []SSecurityGroup{}
	return ret, self.get("ecs", "v2.1", "servers/"+instanceId+"/os-security-groups", &ret)
}

func (self *SInstance) GetProjectId() string {
	return self.EnterpriseProjectId
}

func (self *SInstance) GetError() error {
	return nil
}

func updateUserData(userData, osVersion, username, password, publicKey string) (string, error) {
	winOS := strings.ToLower(osprofile.OS_TYPE_WINDOWS)
	osVersion = strings.ToLower(osVersion)
	config := &cloudinit.SCloudConfig{}
	if len(userData) > 0 {
		if strings.Contains(osVersion, winOS) {
			if _config, err := cloudinit.ParseUserDataBase64(userData); err == nil {
				config = _config
			} else {
				log.Debugf("updateWindowsUserData invalid userdata %s", userData)
			}
		} else {
			if _config, err := cloudinit.ParseUserDataBase64(userData); err == nil {
				config = _config
			} else {
				return "", fmt.Errorf("updateLinuxUserData invalid userdata %s", userData)
			}
		}
	}

	user := cloudinit.NewUser(username)
	config.RemoveUser(user)
	config.DisableRoot = 0
	if len(password) > 0 {
		config.SshPwauth = cloudinit.SSH_PASSWORD_AUTH_ON
		user.Password(password)
		config.MergeUser(user)
	}

	if len(publicKey) > 0 {
		user.SshKey(publicKey)
		config.MergeUser(user)
	}

	if strings.Contains(osVersion, winOS) {
		userData, err := updateWindowsUserData(config.UserDataPowerShell(), osVersion, username, password)
		if err != nil {
			return "", errors.Wrap(err, "updateUserData.updateWindowsUserData")
		}
		return userData, nil
	} else {
		return config.UserDataBase64(), nil
	}
}

func updateWindowsUserData(userData string, osVersion string, username, password string) (string, error) {
	// Windows Server 2003, Windows Vista, Windows Server 2008, Windows Server 2003 R2, Windows Server 2000, Windows Server 2012, Windows Server 2003 with SP1, Windows 8
	oldVersions := []string{"2000", "2003", "2008", "2012", "Vista"}
	isOldVersion := false
	for i := range oldVersions {
		if strings.Contains(osVersion, oldVersions[i]) {
			isOldVersion = true
		}
	}

	shells := ""
	if isOldVersion {
		shells += fmt.Sprintf("rem cmd\n")
		if username == "Administrator" {
			shells += fmt.Sprintf("net user %s %s\n", username, password)
		} else {
			shells += fmt.Sprintf("net user %s %s  /add\n", username, password)
			shells += fmt.Sprintf("net localgroup administrators %s  /add\n", username)
		}

		shells += fmt.Sprintf("net user %s /active:yes", username)
	} else {
		if !strings.HasPrefix(userData, "#ps1") {
			shells = fmt.Sprintf("#ps1\n%s", userData)
		}
	}

	return base64.StdEncoding.EncodeToString([]byte(shells)), nil
}

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
	job := &SJob{}
	err := self.perform("ims", "v2", "cloudimages", "action", params, job)
	if err != nil {
		return nil, err
	}
	for _, id := range job.GetIds() {
		image, err := self.GetImage(id)
		if err != nil {
			return nil, errors.Wrapf(err, "GetImage(%s)", id)
		}
		image.cache = &SStoragecache{region: self}
		return image, nil
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, jsonutils.Marshal(job).String())
}

func (self *SInstance) SaveImage(opts *cloudprovider.SaveImageOptions) (cloudprovider.ICloudImage, error) {
	image, err := self.host.zone.region.SaveImage(self.Id, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "SaveImage")
	}
	return image, nil
}
