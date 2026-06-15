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

package rockbase

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/imagetools"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SInstance struct {
	multicloud.SInstanceBase
	RockbaseTags

	host *SHost

	osInfo *imagetools.ImageInfo

	UHostId            string    `json:"UHostId"`
	Zone               string    `json:"Zone"`
	LifeCycle          string    `json:"LifeCycle"`
	OSName             string    `json:"OsName"`
	ImageId            string    `json:"ImageId"`
	BasicImageId       string    `json:"BasicImageId"`
	BasicImageName     string    `json:"BasicImageName"`
	Tag                string    `json:"Tag"`
	Name               string    `json:"Name"`
	Remark             string    `json:"Remark"`
	State              string    `json:"State"`
	NetworkState       string    `json:"NetworkState"`
	HostType           string    `json:"HostType"`
	MachineType        string    `json:"MachineType"`
	GpuType            string    `json:"GpuType"`
	StorageType        string    `json:"StorageType"`
	TotalDiskSpace     int       `json:"TotalDiskSpace"`
	DiskSet            []DiskSet `json:"DiskSet"`
	NetCapability      string    `json:"NetCapability"`
	IPSet              []IPSet   `json:"IPSet"`
	SubnetType         string    `json:"SubnetType"`
	ChargeType         string    `json:"ChargeType"`
	ExpireTime         int64     `json:"ExpireTime"`
	AutoRenew          string    `json:"AutoRenew"`
	IsExpire           string    `json:"IsExpire"`
	UHostType          string    `json:"UHostType"`
	OSType             string    `json:"OsType"`
	CreateTime         int64     `json:"CreateTime"`
	CPU                int       `json:"CPU"`
	GPU                int       `json:"GPU"`
	MemoryMB           int       `json:"Memory"`
	TimemachineFeature string    `json:"TimemachineFeature"`
	HotplugFeature     bool      `json:"HotplugFeature"`
	NetCapFeature      bool      `json:"NetCapFeature"`
	BootDiskState      string    `json:"BootDiskState"`
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	secgroups, err := self.GetSecurityGroups()
	if err != nil {
		log.Errorln(err)
	}

	secgroupIds := make([]string, 0)
	for _, secgroup := range secgroups {
		secgroupIds = append(secgroupIds, secgroup.GetId())
	}

	return secgroupIds, nil
}

func (self *SInstance) GetProjectId() string {
	return self.host.zone.region.client.projectId
}

func (self *SInstance) GetError() error {
	return nil
}

type DiskSet struct {
	DiskId    string `json:"DiskId"`
	DiskType  string `json:"DiskType"`
	Drive     string `json:"Drive"`
	IsBoot    bool   `json:"IsBoot"`
	Size      int    `json:"Size"`
	Encrypted string `json:"Encrypted"`
	Type      string `json:"Type"`
}

type IPSet struct {
	Type     string `json:"Type"`
	IP       string `json:"IP"`
	IPId     string `json:"IPId"` // IP资源ID (内网IP无对应的资源ID)
	MAC      string `json:"Mac"`
	VpcId    string `json:"VPCId"`
	SubnetId string `json:"SubnetId"`
}

type SVncInfo struct {
	VNCIP       string `json:"VncIP"`
	VNCPassword string `json:"VncPassword"`
	UHostId     string `json:"UHostId"`
	Action      string `json:"Action"`
	VNCPort     int64  `json:"VncPort"`
}

func (self *SInstance) GetId() string {
	return self.UHostId
}

func (self *SInstance) GetName() string {
	if len(self.Name) == 0 {
		return self.GetId()
	}
	return self.Name
}

func (self *SInstance) GetHostname() string {
	return ""
}

func (self *SInstance) GetGlobalId() string {
	return self.GetId()
}

// 实例状态，枚举值：
// >初始化: Initializing;
// >启动中: Starting;
// > 运行中: Running;
// > 关机中: Stopping;
// >关机: Stopped
// >安装失败: Install Fail;
// >重启中: Rebooting
func (self *SInstance) GetStatus() string {
	switch self.State {
	case "Running":
		return api.VM_RUNNING
	case "Stopped":
		return api.VM_READY
	case "Rebooting":
		return api.VM_STOPPING
	case "Initializing":
		return api.VM_INIT
	case "Starting":
		return api.VM_STARTING
	case "Stopping":
		return api.VM_STOPPING
	case "Install Fail":
		return api.VM_CREATE_FAILED
	default:
		return api.VM_UNKNOWN
	}
}

func (self *SInstance) Refresh() error {
	vm, err := self.host.zone.region.GetInstance(self.GetId())
	if err != nil {
		return err
	}
	self.DiskSet = nil
	self.IPSet = nil
	self.osInfo = nil
	return jsonutils.Update(self, vm)
}

// 计费模式，枚举值为： Year，按年付费； Month，按月付费； Dynamic，按需付费（需开启权限）；
func (self *SInstance) GetBillingType() string {
	switch self.ChargeType {
	case "Year", "Month":
		return billing_api.BILLING_TYPE_PREPAID
	default:
		return billing_api.BILLING_TYPE_POSTPAID
	}
}

func (self *SInstance) GetCreatedAt() time.Time {
	return time.Unix(self.CreateTime, 0)
}

func (self *SInstance) GetExpiredAt() time.Time {
	if strings.EqualFold(self.ChargeType, "Year") || strings.EqualFold(self.ChargeType, "Month") {
		return time.Unix(self.ExpireTime, 0)
	}
	return time.Time{}
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetLocalDisk(diskId, storageType string, sizeGB int, isBoot bool) SDisk {
	diskType := ""
	if isBoot {
		diskType = "SystemDisk"
	}

	disk := SDisk{
		SDisk:      multicloud.SDisk{},
		Status:     "Available",
		UHostId:    self.GetId(),
		Name:       diskId,
		Zone:       self.host.zone.GetId(),
		DiskType:   diskType,
		UDiskId:    diskId,
		UHostName:  self.GetName(),
		CreateTime: self.CreateTime,
		SizeGB:     sizeGB,
	}

	disk.storage = &SStorage{zone: self.host.zone, storageType: storageType}
	return disk
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	localDisks := make([]SDisk, 0)
	disks := []SDisk{}
	for _, disk := range self.DiskSet {
		if utils.IsInStringArray(disk.DiskType, []string{api.STORAGE_ROCKBASE_LOCAL_NORMAL, api.STORAGE_ROCKBASE_LOCAL_SSD}) {
			localDisks = append(localDisks, self.GetLocalDisk(disk.DiskId, disk.DiskType, disk.Size, disk.IsBoot))
			continue
		}
		disk, err := self.host.zone.region.GetDisk(disk.DiskId)
		if err != nil {
			return nil, err
		}
		disks = append(disks, *disk)
	}

	disks = append(disks, localDisks...)
	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i += 1 {
		if disks[i].storage == nil {
			var category string
			if strings.Contains(disks[i].DiskType, "SSD") {
				category = api.STORAGE_ROCKBASE_CLOUD_SSD
			} else {
				category = api.STORAGE_ROCKBASE_CLOUD_NORMAL
			}
			storage, err := self.host.zone.getStorageByCategory(category)
			if err != nil {
				return nil, err
			}
			disks[i].storage = storage
		}
		idisks[i] = &disks[i]
		// 将系统盘放到第0个位置
		if disks[i].GetDiskType() == api.DISK_TYPE_SYS {
			idisks[0], idisks[i] = idisks[i], idisks[0]
		}
	}

	return idisks, nil
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics := make([]cloudprovider.ICloudNic, 0)

	for _, ip := range self.IPSet {
		if len(ip.SubnetId) == 0 {
			continue
		}

		nic := SInstanceNic{
			instance: self,
			ipAddr:   ip.IP,
			macAddr:  ip.MAC,
		}
		nics = append(nics, &nic)
	}

	return nics, nil
}

// 国际: Internation，BGP: BGP，内网: Private
func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	for _, ip := range self.IPSet {
		if len(ip.IPId) > 0 {
			eip, err := self.host.zone.region.GetEip(ip.IPId)
			if err != nil {
				return nil, errors.Wrapf(err, "GetEip %s", ip.IPId)
			}

			return eip, nil
		}
	}

	return nil, nil
}

func (self *SInstance) GetVcpuCount() int {
	return self.CPU
}

func (self *SInstance) GetVmemSizeMB() int {
	return self.MemoryMB
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

func (ins *SInstance) getNormalizedOsInfo() *imagetools.ImageInfo {
	if ins.osInfo == nil {
		osInfo := imagetools.NormalizeImageInfo(ins.OSName, "", ins.OSType, "", "")
		ins.osInfo = &osInfo
	}
	return ins.osInfo
}

func (ins *SInstance) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(ins.getNormalizedOsInfo().OsType)
}

func (ins *SInstance) GetFullOsName() string {
	return ins.OSName
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

func (self *SInstance) GetMachine() string {
	return "pc"
}

func (self *SInstance) instanceTypeHostPrefix() string {
	if self.GPU > 0 && len(self.GpuType) > 0 {
		return self.GpuType
	}
	if len(self.UHostType) > 0 {
		return self.UHostType
	}
	if len(self.MachineType) > 0 {
		return self.MachineType
	}
	return self.HostType
}

func (self *SInstance) GetInstanceType() string {
	memGB := self.MemoryMB / 1024
	return formatInstanceSpec(self.instanceTypeHostPrefix(), self.CPU, memGB, self.GPU)
}

// https://docs.ucloud.cn/api/unet-api/grant_firewall
func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	if len(secgroupIds) == 0 {
		return fmt.Errorf("SetSecurityGroups secgroup id should not be empty")
	} else if len(secgroupIds) > 1 {
		return fmt.Errorf("SetSecurityGroups only allowed to assign one secgroup id. %d given", len(secgroupIds))
	}

	return self.host.zone.region.assignSecurityGroups(self.GetId(), secgroupIds[0])
}

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_ROCKBASE
}

func (self *SInstance) StartVM(ctx context.Context) error {
	err := self.host.zone.region.StartVM(self.GetId())
	if err != nil {
		return err
	}
	err = cloudprovider.WaitStatusWithDelay(self, api.VM_RUNNING, 10*time.Second, 10*time.Second, 600*time.Second)
	if err != nil {
		return errors.Wrap(err, "StartVM")
	}

	return nil
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	err := self.host.zone.region.StopVM(self.GetId())
	if err != nil {
		return err
	}
	err = cloudprovider.WaitStatusWithDelay(self, api.VM_READY, 10*time.Second, 10*time.Second, 600*time.Second)
	if err != nil {
		return errors.Wrap(err, "StopVM")
	}

	return nil
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return self.host.zone.region.DeleteVM(self.GetId())
}

func (self *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	return self.host.zone.region.UpdateVM(self.GetId(), input.NAME)
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

// https://docs.ucloud.cn/api/uhost-api/reinstall_uhost_instance
func (self *SInstance) RebuildRoot(ctx context.Context, desc *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	err := self.host.zone.region.RebuildRoot(self.GetId(), desc.ImageId, desc.Password)
	if err != nil {
		return "", err
	}

	err = cloudprovider.WaitStatusWithDelay(self, api.VM_RUNNING, 10*time.Second, 15*time.Second, 300*time.Second)
	if err != nil {
		return "", errors.Wrap(err, "RebuildRoot")
	}

	for _, disk := range self.DiskSet {
		if strings.EqualFold(disk.Type, "SystemDisk") || strings.EqualFold(disk.Type, "Boot") {
			return disk.DiskId, nil
		}
	}
	return "", errors.Wrapf(cloudprovider.ErrNotFound, "SystemDisk not found")
}

func (self *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	if len(opts.PublicKey) > 0 {
		return fmt.Errorf("DeployVM not support assign ssh keypair")
	}

	if opts.DeleteKeypair {
		return fmt.Errorf("DeployVM not support delete ssh keypair")
	}

	if self.GetStatus() != api.VM_READY {
		return fmt.Errorf("DeployVM instance status %s , expected %s.", self.GetStatus(), api.VM_READY)
	}

	if len(opts.Password) > 0 {
		err := self.host.zone.region.ResetVMPasswd(self.GetId(), opts.Password)
		if err != nil {
			return err
		}
	}

	err := cloudprovider.WaitStatus(self, api.VM_READY, 10*time.Second, 300*time.Second)
	if err != nil {
		return errors.Wrap(err, "DeployVM")
	}

	return nil
}

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	if len(config.InstanceType) > 0 {
		return self.ChangeConfig2(ctx, config.InstanceType)
	}
	return self.host.zone.region.ResizeVM(self.GetId(), config.Cpu, config.MemoryMB)
}

func (self *SInstance) ChangeConfig2(ctx context.Context, instanceType string) error {
	i, err := ParseInstanceType(instanceType)
	if err != nil {
		return err
	}

	return self.host.zone.region.ResizeVM(self.GetId(), i.CPU, i.MemoryMB)
}

func (self *SInstance) GetTags() (map[string]string, error) {
	return self.host.zone.region.GetResourceTags(self.GetId())
}

func (self *SInstance) SetTags(tags map[string]string, replace bool) error {
	return self.host.zone.region.SetResourceTags(self.GetId(), tags, replace)
}

func (self *SInstance) SaveImage(opts *cloudprovider.SaveImageOptions) (cloudprovider.ICloudImage, error) {
	image, err := self.host.zone.region.SaveImage(self.GetId(), opts)
	if err != nil {
		return nil, errors.Wrapf(err, "SaveImage")
	}
	return image, nil
}

func (self *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return self.host.zone.region.GetInstanceVNCUrl(self.GetId())
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return self.host.zone.region.AttachDisk(self.host.zone.GetId(), self.GetId(), diskId)
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	err := self.host.zone.region.DetachDisk(self.host.zone.GetId(), self.GetId(), diskId)
	if err != nil {
		return err
	}

	disk, err := self.host.zone.region.GetDisk(diskId)
	if err != nil {
		return err
	}

	disk.storage = &SStorage{zone: self.host.zone, storageType: disk.GetStorageType()}
	err = cloudprovider.WaitStatusWithDelay(disk, api.DISK_READY, 10*time.Second, 10*time.Second, 60*time.Second)
	if err != nil {
		return errors.Wrap(err, "DetachDisk")
	}

	return nil
}

func (self *SInstance) Renew(bc billing.SBillingCycle) error {
	// return self.host.zone.region
	return self.host.zone.region.RenewInstance(self.GetId(), bc)
}

func (self *SInstance) GetSecurityGroups() ([]SSecurityGroup, error) {
	return self.host.zone.region.GetSecurityGroups("", self.GetId(), "")
}

// https://docs.ucloud.cn/api/uhost-api/get_uhost_instance_vnc_info
func (self *SRegion) GetInstanceVNCUrl(instanceId string) (*cloudprovider.ServerVncOutput, error) {
	params := NewRockbaseParams()
	params.Set("UHostId", instanceId)
	vnc := SVncInfo{}
	err := self.DoAction("GetUHostInstanceVncInfo", params, &vnc)
	if err != nil {
		return nil, err
	}

	ret := &cloudprovider.ServerVncOutput{
		Host:       vnc.VNCIP,
		Port:       vnc.VNCPort,
		Password:   vnc.VNCPassword,
		Hypervisor: api.HYPERVISOR_ROCKBASE,
		Protocol:   "vnc",
		InstanceId: instanceId,
	}
	return ret, nil
}

// https://docs.ucloud.cn/api/unet-api/grant_firewall
func (self *SRegion) assignSecurityGroups(instanceId string, secgroupId string) error {
	params := NewRockbaseParams()
	params.Set("FWId", secgroupId)
	params.Set("ResourceType", "uhost")
	params.Set("ResourceId", instanceId)

	return self.DoAction("GrantFirewall", params, nil)
}

// https://docs.ucloud.cn/api/uhost-api/start_uhost_instance
func (self *SRegion) StartVM(instanceId string) error {
	params := NewRockbaseParams()
	params.Set("UHostId", instanceId)

	return self.DoAction("StartUHostInstance", params, nil)
}

// https://docs.ucloud.cn/api/uhost-api/stop_uhost_instance
func (self *SRegion) StopVM(instanceId string) error {
	params := NewRockbaseParams()
	params.Set("UHostId", instanceId)

	return self.DoAction("StopUHostInstance", params, nil)
}

// https://docs.ucloud.cn/api/uhost-api/terminate_uhost_instance
func (self *SRegion) DeleteVM(instanceId string) error {
	params := NewRockbaseParams()
	params.Set("UHostId", instanceId)
	params.Set("Destroy", 1) // 跳过回收站，直接删除
	params.Set("ReleaseUDisk", true)

	return self.DoAction("TerminateUHostInstance", params, nil)
}

// https://docs.ucloud.cn/api/uhost-api/modify_uhost_instance_name
func (self *SRegion) UpdateVM(instanceId, name string) error {
	params := NewRockbaseParams()
	params.Set("UHostId", instanceId)
	params.Set("Name", name)

	return self.DoAction("ModifyUHostInstanceName", params, nil)
}

// ChargeType : Dynamic(按需)/Month(按月)/Year(按年)
func (self *SRegion) RenewInstance(instanceId string, bc billing.SBillingCycle) error {
	params := NewRockbaseParams()
	params.Set("ResourceId", instanceId)
	params.Set("ResourceType", "Host")

	if bc.GetMonths() >= 10 && bc.GetMonths() < 12 {
		params.Set("ChargeType", "Year")
		params.Set("Quantity", 1)
	} else if bc.GetYears() >= 1 {
		params.Set("ChargeType", "Year")
		params.Set("Quantity", bc.GetYears())
	} else {
		params.Set("ChargeType", "Month")
		params.Set("Quantity", bc.GetMonths())
	}

	return self.DoAction("CreateRenew", params, nil)
}

// https://docs.ucloud.cn/api/uhost-api/reset_uhost_instance_password
// 该操作需要UHost实例处于关闭状态。
func (self *SRegion) ResetVMPasswd(instanceId, password string) error {
	params := NewRockbaseParams()
	params.Set("UHostId", instanceId)
	params.Set("Password", base64.StdEncoding.EncodeToString([]byte(password)))

	return self.DoAction("ResetUHostInstancePassword", params, nil)
}

// https://docs.ucloud.cn/api/uhost-api/reinstall_uhost_instance
// （密码格式使用BASE64编码；LoginMode不可变更）
func (self *SRegion) RebuildRoot(instanceId, imageId, password string) error {
	params := NewRockbaseParams()
	params.Set("UHostId", instanceId)
	params.Set("Password", base64.StdEncoding.EncodeToString([]byte(password)))
	params.Set("ImageId", imageId)

	return self.DoAction("ReinstallUHostInstance", params, nil)
}

// https://docs.ucloud.cn/api/uhost-api/resize_uhost_instance
func (self *SRegion) ResizeVM(instanceId string, cpu, memoryMB int) error {
	params := NewRockbaseParams()
	params.Set("UHostId", instanceId)
	params.Set("CPU", cpu)
	params.Set("Memory", memoryMB)

	return self.DoAction("ResizeUHostInstance", params, nil)
}
