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

package compute

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/billing"
)

type ServerListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.DeletePreventableResourceBaseListInput
	apis.MultiArchResourceBaseListInput

	HostFilterListInput

	NetworkFilterListInput `yunion-ambiguous-prefix:"vpc_"`

	billing.BillingResourceListInput

	GroupFilterListInput
	SecgroupFilterListInput
	//DiskFilterListInput `yunion-ambiguous-prefix:"storage_"`
	ScalingGroupFilterListInput

	// 只列出裸金属主机
	Baremetal *bool `json:"baremetal"`
	// 只列出GPU主机
	Gpu *bool `json:"gpu"`
	// 只列出还有备份机的主机
	Backup *bool `json:"bakcup"`
	// 列出指定类型的主机
	// enum: normal,gpu,backup
	ServerType string `json:"server_type"`
	// 列出管理安全组为指定安全组的主机
	AdminSecgroup string `json:"admin_security"`
	// 列出Hypervisor为指定值的主机
	// enum: kvm,esxi,baremetal,aliyun,azure,aws,huawei,ucloud,zstack,openstack,google,ctyun"`
	Hypervisor []string `json:"hypervisor"`
	// 列出绑定了弹性IP（EIP）的主机
	WithEip *bool `json:"with_eip"`
	// 列出未绑定弹性IP（EIO）的主机
	WithoutEip *bool `json:"without_eip"`
	// 列出操作系统为指定值的主机
	// enum: linux,windows,vmware
	OsType []string `json:"os_type"`

	// 对列表结果按照磁盘进行排序
	// enum: asc,desc
	// OrderByDisk string `json:"order_by_disk"`

	// 根据ip查找机器
	IpAddr string `json:"ip_addr"`

	// 列出可以挂载指定EIP的主机
	UsableServerForEip string `json:"usable_server_for_eip"`

	// 列出可以挂载磁盘的主机
	AttachableServersForDisk string `json:"attachable_servers_for_disk"`
	// Deprecated
	// 列出可以挂载磁盘的主机
	Disk string `json:"disk" yunion-deprecated-by:"attachable_servers_for_disk"`

	// 按主机资源类型进行排序
	// enum: shared,prepaid,dedicated
	ResourceType string `json:"resource_type"`
	// 返回该宿主机上的所有虚拟机，包括备份机
	GetAllGuestsOnHost string `json:"get_all_guests_on_host"`

	// 根据宿主机 SN 过滤
	// HostSn string `json:"host_sn"`

	VcpuCount []int `json:"vcpu_count"`

	VmemSize []int `json:"vmem_size"`

	BootOrder []string `json:"boot_order"`

	Vga []string `json:"vga"`

	Vdi []string `json:"vdi"`

	Machine []string `json:"machine"`

	Bios []string `json:"bios"`

	SrcIpCheck *bool `json:"src_ip_check"`

	SrcMacCheck *bool `json:"src_mac_check"`

	InstanceType []string `json:"instance_type"`

	// 是否调度到宿主机上
	WithHost *bool `json:"with_host"`
}

func (input *ServerListInput) AfterUnmarshal() {
	if input.Baremetal != nil && *input.Baremetal {
		input.Hypervisor = append(input.Hypervisor, HYPERVISOR_BAREMETAL)
	}
}

type ServerRebuildRootInput struct {
	apis.Meta

	// 镜像名称
	Image string `json:"image"`
	// 镜像 id
	// required: true
	ImageId       string `json:"image_id"`
	Keypair       string `json:"keypair"`
	KeypairId     string `json:"keypair_id"`
	ResetPassword *bool  `json:"reset_password"`
	Password      string `json:"password"`
	AutoStart     *bool  `json:"auto_start"`
	AllDisks      *bool  `json:"all_disks"`
}

func (i ServerRebuildRootInput) GetImageName() string {
	if len(i.Image) > 0 {
		return i.Image
	}
	if len(i.ImageId) > 0 {
		return i.ImageId
	}
	return ""
}

func (i ServerRebuildRootInput) GetKeypairName() string {
	if len(i.Keypair) > 0 {
		return i.Keypair
	}
	if len(i.KeypairId) > 0 {
		return i.KeypairId
	}
	return ""
}

type ServerResumeInput struct {
	apis.Meta
}

type ServerDetails struct {
	apis.VirtualResourceDetails

	SGuest

	HostResourceInfo

	// details
	// 网络概要
	Networks string `json:"networks"`
	// 磁盘概要
	Disks string `json:"disks"`

	// 磁盘详情
	DisksInfo []GuestDiskInfo `json:"disks_info"`
	// 虚拟机Ip列表
	VirtualIps string `json:"virtual_ips"`
	// 安全组规则
	SecurityRules string `json:"security_rules"`
	// 操作系统名称
	OsName string `json:"os_name"`
	// 操作系统类型
	OsType string `json:"os_type"`
	// 系统管理员可见的安全组规则
	AdminSecurityRules string `json:"admin_security_rules"`

	// list
	AttachTime time.Time `json:"attach_time"`

	// common
	IsPrepaidRecycle bool `json:"is_prepaid_recycle"`

	// 备份主机所在宿主机名称
	BackupHostName string `json:"backup_host_name"`
	// 备份主机所在宿主机状态
	BackupHostStatus string `json:"backup_host_status"`

	// 是否可以回收
	CanRecycle bool `json:"can_recycle"`

	// 自动释放时间
	AutoDeleteAt time.Time `json:"auto_delete_at"`
	// 磁盘数量
	DiskCount int `json:"disk_count"`
	// 是否支持ISO启动
	CdromSupport bool `json:"cdrom_support"`

	// 磁盘大小
	// example:30720
	DiskSizeMb int64 `json:"disk"`
	// IP地址列表字符串
	// example: 10.165.2.1,172.16.8.1
	IPs string `json:"ips"`
	// mac地址信息
	Macs string `json:"macs"`
	// 网卡信息
	Nics []GuestnetworkShortDesc `json:"nics"`

	// 归属VPC
	Vpc string `json:"vpc"`
	// 归属VPC ID
	VpcId string `json:"vpc_id"`

	// 关联安全组列表
	Secgroups []apis.StandaloneShortDesc `json:"secgroups"`
	// 关联主安全组
	Secgroup string `json:"secgroup"`

	// 浮动IP
	Eip string `json:"eip"`
	// 浮动IP类型
	EipMode string `json:"eip_mode"`

	// 密钥对
	Keypair string `json:"keypair"`

	// 直通设备（GPU）列表
	IsolatedDevices []SIsolatedDevice `json:"isolated_devices"`
	// 是否支持GPU
	IsGpu bool `json:"is_gpu"`

	// Cdrom信息
	Cdrom string `json:"cdrom,allowempty"`

	// 主机在伸缩组中的状态
	ScalingStatus string `json:"scaling_status"`

	// 伸缩组id
	ScalingGroupId string `json:"scaling_group_id"`
}

// GuestDiskInfo describe the information of disk on the guest.
type GuestDiskInfo struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	FsFormat    string `json:"fs,omitempty"`
	DiskType    string `json:"disk_type"`
	Index       int8   `json:"index"`
	SizeMb      int    `json:"size"`
	DiskFormat  string `json:"disk_format"`
	Driver      string `json:"driver"`
	CacheMode   string `json:"cache_mode"`
	AioMode     string `json:"aio_mode"`
	MediumType  string `json:"medium_type"`
	StorageType string `json:"storage_type"`
	Iops        int    `json:"iops"`
	Bps         int    `json:"bps"`
	ImageId     string `json:"image_id,omitempty"`
	Image       string `json:"image,omitemtpy"`
}

type GuestJointResourceDetails struct {
	apis.VirtualJointResourceBaseDetails

	// 云主机名称
	Guest string `json:"guest"`
	// 云主机名称
	Server string `json:"server"`
}

type GuestJointsListInput struct {
	apis.VirtualJointResourceBaseListInput

	ServerFilterListInput
}

type GuestResourceInfo struct {
	// 虚拟机名称
	Guest string `json:"guest"`

	// 虚拟机状态
	GuestStatus string `json:"guest_status"`

	// 宿主机ID
	HostId string `json:"host_id"`

	HostResourceInfo
}

type ServerResourceInput struct {
	// 主机（ID或Name）
	ServerId string `json:"server_id"`
	// swagger:ignore
	// Deprecated
	// Filter by guest Id
	Server string `json:"server" yunion-deprecated-by:"server_id"`
	// swagger:ignore
	// Deprecated
	// Filter by guest Id
	Guest string `json:"guest" yunion-deprecated-by:"server_id"`
	// swagger:ignore
	// Deprecated
	// Filter by guest Id
	GuestId string `json:"guest_id" yunion-deprecated-by:"server_id"`
}

type ServerFilterListInput struct {
	HostFilterListInput

	ServerResourceInput

	// 以主机名称排序
	OrderByServer string `json:"order_by_server"`
}

type GuestJointBaseUpdateInput struct {
	apis.VirtualJointResourceBaseUpdateInput
}

type GuestPublicipToEipInput struct {
	// 转换完成后是否自动启动
	// default: false
	AutoStart bool `json:"auto_start"`
}

type GuestAutoRenewInput struct {

	// 设置自动续费
	// default: false
	// 自动续费分为本地和云上两种模式
	// 若公有云本身支持自动续费功能, 则使用云上设置
	// 若公有云本身不支持自动续费, 则在本地周期(默认三小时)检查快过期虚拟机并进行续费一个月
	AutoRenew bool `json:"auto_renew"`
}

type ConvertEsxiToKvmInput struct {
	apis.Meta

	// target hypervisor
	TargetHypervisor string `json:"target_hypervisor"`
	// 指定转换的宿主机
	PreferHost string `json:"prefer_host"`
}

type GuestSaveToTemplateInput struct {
	// The name of guest template
	Name string `json:"name"`
	// The generate name of guest template
	GenerateName string `json:"generate_name"`
}

type GuestSyncFixNicsInput struct {
	// 需要修正的IP地址列表
	Ip []string `json:"ip"`
}

type GuestMigrateInput struct {
	PreferHost   string `json:"prefer_host"`
	AutoStart    bool   `json:"auto_start"`
	IsRescueMode bool   `json:"rescue_mode"`
}

type GuestLiveMigrateInput struct {
	PreferHost string `json:"prefer_host"`
}

type GuestSetSecgroupInput struct {
	// 安全组Id列表
	// 实例必须处于运行,休眠或者关机状态
	//
	//
	// | 平台		 | 最多绑定安全组数量	|
	// |-------------|-------------------	|
	// | Azure       | 1					|
	// | VMware      | 不支持安全组			|
	// | Baremetal   | 不支持安全组			|
	// | ZStack	     | 1					|
	// | 其他	     | 5					|
	SecgroupIds []string `json:"secgroup_ids"`
}

type GuestRevokeSecgroupInput struct {
	// 安全组Id列表
	// 实例必须处于运行,休眠或者关机状态
	SecgroupIds []string `json:"secgroup_ids"`
}

type GuestAssignSecgroupInput struct {
	// 安全组Id
	// 实例必须处于运行,休眠或者关机状态
	SecgroupId string `json:"secgroup_id"`

	// swagger:ignore
	// Deprecated
	Secgrp string `json:"secgrp" yunion-deprecated-by:"secgroup_id"`

	// swagger:ignore
	// Deprecated
	Secgroup string `json:"secgroup" yunion-deprecated-by:"secgroup_id"`
}

type GuestAddSecgroupInput struct {
	// 安全组Id列表
	// 实例必须处于运行,休眠或者关机状态
	//
	//
	// | 平台		 | 最多绑定安全组数量	|
	// |-------------|-------------------	|
	// | Azure       | 1					|
	// | VMware      | 不支持安全组			|
	// | Baremetal   | 不支持安全组			|
	// | ZStack	     | 1					|
	// | 其他	     | 5					|
	SecgroupIds []string `json:"secgroup_ids"`
}

type ServerRemoteUpdateInput struct {
	// 是否覆盖替换所有标签
	ReplaceTags *bool `json:"replace_tags" help:"replace all remote tags"`
}

type ServerAssociateEipInput struct {
	// swagger:ignore
	// Deprecated
	Eip string `json:"eip" yunion-deprecated-by:"eip_id"`
	// 弹性公网IP的ID
	EipId string `json:"eip_id"`
}

type ServerDissociateEipInput struct {
	// 是否自动释放
	AudoDelete *bool `json:"auto_delete"`
}

type ServerResetInput struct {
	InstanceSnapshot string `json:"instance_snapshot"`
	// 自动启动
	AutoStart *bool `json:"auto_start"`
}

type ServerStopInput struct {
	// 是否强制关机
	IsForce bool `json:"is_force"`

	// 是否关机停止计费, 若平台不支持停止计费，此参数无作用
	// 目前仅阿里云，腾讯云此参数生效
	StopCharging bool `json:"stop_charging"`
}
