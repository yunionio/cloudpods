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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type SchedtagConfig struct {
	apis.Meta

	Id           string `json:"id"`
	Strategy     string `json:"strategy"`
	Weight       int    `json:"weight"`
	ResourceType string `json:"resource_type"`
}

type NetworkConfig struct {
	apis.Meta

	// 网卡序号, 从0开始
	// required: true
	Index int `json:"index"`

	// 子网名称或ID
	// required: true
	Network string `json:"network"`

	// swagger:ignore
	Wire string `json:"wire"`

	// 内网地址
	Exit    bool `json:"exit"`
	Private bool `json:"private"`

	// 网卡MAC地址
	// requried: false
	Mac string `json:"mac"`

	// 子网内的IPv4地址, 若不指定会安装子网的地址分配策略分配一个IP地址
	// required: false
	Address string `json:"address"`

	// 子网内的IPv6地址
	// required: false
	// swagger:ignore
	Address6 string `json:"address6"`

	// 驱动方式
	// 若指定镜像的网络驱动方式，此参数会被覆盖
	Driver   string `json:"driver"`
	BwLimit  int    `json:"bw_limit"`
	Vip      bool   `json:"vip"`
	Reserved bool   `json:"reserved"`
	NetType  string `json:"net_type"`

	RequireDesignatedIP bool `json:"require_designated_ip"`

	RequireTeaming bool `json:"require_teaming"`
	TryTeaming     bool `json:"try_teaming"`

	StandbyPortCount int `json:"standby_port_count"`
	StandbyAddrCount int `json:"standby_addr_count"`

	// swagger:ignore
	Project string `json:"project_id"`

	// swagger:ignore
	Domain    string            `json:"domain_id"`
	Ifname    string            `json:"ifname"`
	Schedtags []*SchedtagConfig `json:"schedtags"`
}

type AttachNetworkInput struct {
	// 添加的网卡的配置
	// required: true
	Nets []*NetworkConfig `json:"nets"`
}

type DiskConfig struct {
	apis.Meta

	// 挂载到虚拟机的磁盘顺序, -1代表不挂载任何虚拟机
	// default: -1
	Index int `json:"index"`

	// 镜像ID,通过镜像创建磁盘,创建虚拟机时第一块磁盘需要指定此参数
	// required: false
	ImageId string `json:"image_id"`

	// 快照ID,通过快照创建磁盘,此参数必须加上 'snapshot-' 前缀
	// example: snapshot-3140cecb-ccc4-4865-abae-3a5ba8c69d9b
	// requried: false
	SnapshotId string `json:"snapshot_id"`

	// 磁盘类型
	// enum: sys, data, swap
	DiskType string `json:"disk_type"`

	Schedtags []*SchedtagConfig `json:"schedtags"`

	// 磁盘大小,单位Mb
	// 若创建裸金属服务器是, -1代表自动向后扩展存储
	// requried:true
	SizeMb int `json:"size"`

	// 文件系统,仅kvm支持自动格式化磁盘,私有云和公有云此参数不会生效
	// enum: swap, ext2, ext3, ext4, xfs, ntfs, fat, hfsplus
	// requried: false
	Fs string `json:"fs"`

	// 磁盘存储格式
	// enum: qcow2, raw, docker, iso, vmdk, vmdkflatver1, vmdkflatver2, vmdkflat, vmdksparse, vmdksparsever1, vmdksparsever2, vmdksepsparse vhd
	// requried: false
	Format string `json:"format"`

	// 磁盘驱动方式
	// enum: virtio, ide, scsi, sata, pvscsi
	// requried: false
	Driver string `json:"driver"`

	// 磁盘缓存模式
	// enum: writeback, none, writethrough
	// requried: false
	Cache string `json:"cache"`

	// 挂载点,必须以 '/' 开头,例如 /opt 仅KVM此参数有效
	// requried: false
	Mountpoint string `json:"mountpoint"`

	// 操作系统CPU架构
	// required: false
	OsArch string `json:"os_arch"`

	//后端存储类型,若指定了存储ID,此参数会根据存储设置,若不指定，则作为调度的一个参考
	//
	//
	//
	//| 平台			|  存储类型				|	中文		|	数据盘		|	系统盘			|	可否扩容	|	可否单独创建	|	扩容步长	|	说明	|
	//|	----			|	----				|	----		|	-----		|	-----			|	-------		|	-----------		|	-------		|	-----	|
	//|	Esxi			|local					|本地盘			|1-500GB		|30~500GB			|	是			|	否				|	1G			|			|
	//|	Esxi			|nas					|NAS盘			|30-3072GB		|30~500GB			|	是			|	否				|	1G			|			|
	//|	Esxi			|vsan					|VSAN盘			|30-3072GB		|30~500GB			|	是			|	否				|	1G			|			|
	//|	KVM				|local					|本地盘盘		|1-500GB		|30~500GB			|	是			|	否				|	1G			|			|
	//|	KVM				|rbd					|云硬盘			|1-3072GB		|30~500GB			|	是			|	是				|	1G			|			|
	//|	Azure			|standard_lrs			|标准 HDD		|1-4095GB		|30~4095GB			|	是			|	是				|	1G			|			|
	//|	Azure			|standardssd_lrs		|标准 SSD		|1-4095GB		|30~4095GB			|	是			|	是				|	1G			|			|
	//|	Azure			|premium_lrs			|高级SSD		|1-4095GB		|30~4095GB			|	是			|	是				|	1G			|			|
	//|	AWS				|gp2					|通用型SSD		|1-16384GB		|20~16384GB			|	是			|	是				|	1G			|			|
	//|	AWS				|io1					|预配置 IOPS SSD|4-16384GB		|20-16384GB			|	是			|	是				|	1G			|			|
	//|	AWS				|st1					|吞吐优化HDD	|500-16384GB	|不支持				|	是			|	是				|	1G			|			|
	//|	AWS				|sc1					|Cold HDD		|500-16384GB	|不支持				|	是			|	是				|	1G			|			|
	//|	AWS				|standard				|磁介质			|1-1024GB		|20-1024GB			|	是			|	是				|	1G			|			|
	//|	腾讯云			|cloud_ssd				|SSD云硬盘		|10-16000GB		|50~500GB			|	是			|	是				|	10G			|			|
	//|	腾讯云			|cloud_basic			|普通云硬盘		|10-16000GB		|50~500GB			|	是			|	是				|	10G			|			|
	//|	腾讯云			|cloud_preium			|高性能云硬盘	|10-16000GB		|50~1024GB			|	是			|	是				|	10G			|			|
	//|	腾讯云			|local_basic			|普通本地盘		|10-1600GB		|不支持				|	否			|	否				|				|			|
	//|	腾讯云			|local_ssd				|SSD本地盘		|10-7000GB		|部分区域套餐支持	|	否			|	否				|				|			|
	//|	腾讯云			|local_pro				|HDD本地盘		|跟随套餐		|部分区域套餐支持	|	否			|	否				|				|			|
	//|	华为云或天翼云	|SSD					|超高IO云硬盘	|10-32768GB		|40~1024GB			|	是			|	是				|	1G			|			|
	//|	华为云或天翼云	|SAS					|高IO云硬盘		|10-32768GB		|40~1024GB			|	是			|	是				|	1G			|			|
	//|	华为云或天翼云	|SATA					|普通IO云硬盘	|10-32768GB		|40~1024GB			|	是			|	是				|	1G			|			|
	//|	OpenStack		|nova					|nova			|不支持			|30-500GB			|	否			|	否				|				|			|
	//|	OpenStack		|自定义					|...			|无限制			|无限制				|	是			|	是				|	1G			|			|
	//|	Ucloud			|CLOUD_NORMAL			|普通云盘		|20-8000GB		|不支持				|	是			|	是				|	1G			|			|
	//|	Ucloud			|CLOUD_SSD				|SSD云盘		|20-4000GB		|20-500GB			|	是			|	是				|	1G			|			|
	//|	Ucloud			|LOCAL_NORMAL			|普通本地盘		|				|					|	是			|	是				|	1G			|			|
	//|	Ucloud			|LOCAL_SSD				|SSD本地盘		|				|					|	是			|	是				|	1G			|			|
	//|	Ucloud			|EXCLUSIVE_LOCAL_DISK	|独享本地盘		|				|					|	是			|	是				|	1G			|			|
	//|	ZStack			|localstorage			|本地盘			|				|					|	是			|	是				|	1G			|			|
	//|	ZStack			|ceph					|云硬盘			|				|					|	是			|	是				|	1G			|			|
	//|	Google			|local-ssd				|本地SSD暂存盘	|375GB			|不支持				|	否			|	否				|				|	跟随实例创建，一次最多添加8个		|
	//|	Google			|pd-standard			|标准永久性磁盘	|10-65536GB		|10-65536GB			|	是			|	是				|	1G			|			|
	//|	Google			|pd-ssd					|SSD永久性磁盘	|10-65536GB		|10-65536GB			|	是			|	是				|	1G			|			|
	Backend string `json:"backend"`

	//介质类型
	//rotate: 机械盘
	//ssd: 固态硬盘
	//hybird: 混合盘
	//emum: [rotate, ssd, hybrid]
	//default: hybird
	Medium string `json:"medium"`

	//swagger:ignore
	ImageProperties map[string]string `json:"image_properties"`

	//存储ID, 指定存储后，磁盘会在此存储上创建
	//存储列表可以参数 storage 列表API
	//required: false
	Storage string `json:"storage_id"`

	//swagger:ignore
	DiskId string `json:"disk_id"`
}

type IsolatedDeviceConfig struct {
	Index   int    `json:"index"`
	Id      string `json:"id"`
	DevType string `json:"dev_type"`
	Model   string `json:"model"`
	Vendor  string `json:"vendor"`
}

type BaremetalDiskConfig struct {
	//Index int `json:"index"`
	// disk type
	Type string `json:"type"` // ssd / rotate
	// raid config
	Conf         string  `json:"conf"`  // raid配置
	Count        int64   `json:"count"` // 连续几块
	Range        []int64 `json:"range"` // 指定几块
	Splits       string  `json:"splits"`
	Size         []int64 `json:"size"` //
	Adapter      *int    `json:"adapter,omitempty"`
	Driver       string  `json:"driver"`
	Cachedbadbbu *bool   `json:"cachedbadbbu,omitempty"`
	Strip        *int64  `json:"strip,omitempty"`
	RA           *bool   `json:"ra,omitempty"`
	WT           *bool   `json:"wt,omitempty"`
	Direct       *bool   `json:"direct,omitempty"`
}

type ServerConfigs struct {
	// 调度使用指定的云账号
	PreferManager string `json:"prefer_manager_id"`

	// 调度到指定区域,优先级低于prefer_zone_id
	PreferRegion string `json:"prefer_region_id"`

	// 调度到指定可用区,优先级低于prefer_host_id
	PreferZone string `json:"prefer_zone_id"`

	// 调度使用指定二层网络, 优先级低于prefer_host_id
	PreferWire string `json:"prefer_wire_id"`

	// 调度使用指定宿主机
	PreferHost string `json:"prefer_host_id"`

	// 主机高可用时，将备机调度到指定宿主机, 此参数仅对KVM生效
	PreferBackupHost string `json:"prefer_backup_host"`

	// 虚拟化技术或平台
	//
	//
	//
	// |hypervisor	|	技术或平台	|
	// |-------		|	----------	|
	// |kvm			|	本地私有云	|
	// |esxi		|	VMWare		|
	// |baremetal	|	裸金属		|
	// |aliyun		|	阿里云		|
	// |aws			|	亚马逊		|
	// |qcloud		|	腾讯云		|
	// |azure		|	微软云		|
	// |huawei		|	华为云		|
	// |openstack	|	OpenStack	|
	// |ucloud		|	Ucloud		|
	// |zstack		|	ZStack		|
	// |google		|	谷歌云		|
	// |ctyun		|	天翼云		|
	// default: kvm
	Hypervisor string `json:"hypervisor"`

	// 包年包月资源池
	// swagger:ignore
	// emum: shared, prepaid, dedicated
	ResourceType string `json:"resource_type"`

	// 套餐名称, 此参数优先级高于vcpu_count和vmem_size
	// 套餐可以通过 serverskus 列表获取
	// esxi, openstack, zstack等私有云都统一使用provider为OneCloud的套餐
	// 公有云使用各自的套餐
	InstanceType string `json:"instance_type"`

	// swagger:ignore
	// Deprecated
	// alias for InstanceType
	Sku string `json:"sku" yunion-deprecated-by:"instance_type"`

	// 虚拟机高可用(创建备机)
	// default: false
	// required: false
	Backup bool `json:"backup"`

	// 创建虚拟机数量
	// default: 1
	Count int `json:"count"`

	// 磁盘列表,第一块磁盘为系统盘,需要指定image_id
	// 若指定主机快照，此参数可以为空
	// required: true
	Disks []*DiskConfig `json:"disks"`

	// 指定主机网络
	// required: false
	Networks []*NetworkConfig `json:"nets"`

	// 调度标签
	// required: false
	Schedtags []*SchedtagConfig `json:"schedtags"`

	// 透传设备列表
	// required: false
	IsolatedDevices []*IsolatedDeviceConfig `json:"isolated_devices"`

	// 裸金属磁盘配置列表
	BaremetalDiskConfigs []*BaremetalDiskConfig `json:"baremetal_disk_configs"`

	// 主机组列表, 参数可以是主机组名称或ID,建议使用ID
	InstanceGroupIds []string `json:"groups"`

	// DEPRECATE
	Suggestion bool `json:"suggestion"`
}

func NewServerConfigs() *ServerConfigs {
	return &ServerConfigs{
		Disks:                make([]*DiskConfig, 0),
		Networks:             make([]*NetworkConfig, 0),
		Schedtags:            make([]*SchedtagConfig, 0),
		IsolatedDevices:      make([]*IsolatedDeviceConfig, 0),
		BaremetalDiskConfigs: make([]*BaremetalDiskConfig, 0),
		InstanceGroupIds:     make([]string, 0),
	}
}

type DeployConfig struct {
	Action  string `json:"action"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

type ServerCreateInput struct {
	apis.VirtualResourceCreateInput
	DeletePreventableCreateInput

	*ServerConfigs

	// 虚拟机内存大小,单位Mb,若未指定instance_type,此参数为必传项
	VmemSize int `json:"vmem_size"`

	// 虚拟机Cpu大小,若未指定instance_type,此参数为必传项
	// default: 1
	VcpuCount int `json:"vcpu_count"`

	// 用户自定义启动脚本
	// required: false
	UserData string `json:"user_data"`

	// swagger:ignore
	// Deprecated
	Keypair string `json:"keypair" yunion-deprecated-by:"keypair_id"`

	// 秘钥对Id
	// required: false
	KeypairId string `json:"keypair_id"`

	// 密码
	// 要求: 密码长度 >= 20, 至少包含一个数字一个小写字母一个大小字母及特殊字符~`!@#$%^&*()-_=+[]{}|:';\",./<>?中的一个
	// requried: false
	Password string `json:"password"`

	// 登录账户
	// required: false
	LoginAccount string `json:"login_account"`

	// 使用ISO光盘启动, 仅KVM平台支持
	// required: false
	Cdrom string `json:"cdrom"`

	// enum: cirros, vmware, qxl, std
	// default: std
	Vga string `json:"vga"`

	// 远程连接协议
	// enum: vnc, spice
	// default: vnc
	Vdi string `json:"vdi"`

	// BIOS类型, 若镜像是Windows，并且支持UEFI,则自动会设置为UEFI
	// emulate: BIOS, UEFI
	Bios string `json:"bios"`

	// 启动顺序
	// c: cdrome
	// d: disk
	// n: network
	// example: cnd
	// default: cdn
	BootOrder string `json:"boot_order"`

	// 启用cloud-init,需要镜像装有cloud-init服务
	// default: false
	EnableCloudInit bool `json:"enable_cloud_init"`

	// 随机密码, 若指定password参数,此参数不生效
	// 若值为false并且password为空,则表示保留镜像密码
	ResetPassword *bool `json:"reset_password"`

	// 关机后执行的操作
	// terminate: 关机后自动删除
	// emum: stop, terminate
	// default: stop
	ShutdownBehavior string `json:"shutdown_behavior"`

	// 创建后自动启动
	// 部分云创建后会自动启动例如: 腾讯云, AWS, OpenStack, ZStack, Ucloud, Huawei, Azure, 天翼云
	// default: false
	AutoStart     bool            `json:"auto_start"`
	DeployConfigs []*DeployConfig `json:"deploy_configs"`

	// 包年包月时长
	//
	//
	// |平台				|是否支持	|
	// |----				|-------	|
	// |KVM					|否			|
	// |ESxi				|否			|
	// |OpenStack			|否			|
	// |ZStack				|否			|
	// |Google				|否			|
	// |Azure				|否			|
	// |AWS					|否			|
	// |腾讯云				|是			|
	// |Aliyun				|是			|
	// |Ucloud				|是			|
	// |Huawei				|是			|
	// |天翼云				|是			|
	Duration string `json:"duration"`

	// 是否自动续费
	// default: false
	AutoRenew bool `json:"auto_renew"`

	// swagger:ignore
	AutoPrepaidRecycle bool `json:"auto_prepaid_recycle,omitfalse"`

	// 弹性公网IP带宽
	// 指定此参数后会创建新的弹性公网IP并绑定到新建的虚拟机
	// 此参数优先级低于public_ip
	EipBw int `json:"eip_bw,omitzero"`
	// 弹性公网IP线路类型
	EipBgpType string `json:"eip_bgp_type,omitzero"`
	// 弹性公网IP计费类型
	EipChargeType string `json:"eip_charge_type,omitempty"`
	// 是否跟随主机删除而自动释放
	EipAutoDellocate bool `json:"eip_auto_dellocate,omitempty"`

	// 弹性公网IP名称或ID
	// 绑定已有弹性公网IP, 此参数会限制虚拟机再谈下公网IP所在的区域创建
	// required: false
	Eip string `json:"eip,omitempty"`

	// 公网IP带宽(单位MB)
	// 若指定此参数则忽略eip相关参数
	// 私有云不支持此参数
	//
	//
	// |平台				|支持范围	|
	// |----				|-------	|
	// |腾讯云				|按量计费1-100, 包年包月1-200 |
	PublicIpBw int `json:"public_ip_bw,omitzero"`
	// 公网IP计费类型
	// 默认按流量计费
	//
	//
	// |类别					|说明	|
	// |----					|-------	|
	// |traffic					|按流量计费|
	// |bandwidth				|按带宽计费|
	PublicIpChargeType string `json:"public_ip_charge_type,omitempty"`

	// 使用主机快照创建虚拟机, 主机快照不会重置密码及秘钥信息
	// 使用主机快照创建的虚拟机将沿用之前的密码秘钥及安全组信息
	// required: false
	InstanceSnapshotId string `json:"instance_snapshot_id,omitempty"`

	// 安全组Id, 此参数会和secgroups参数合并
	SecgroupId string `json:"secgrp_id"`
	// 安全组Id列表
	Secgroups []string `json:"secgroups"`

	// swagger:ignore
	OsType string `json:"os_type"`
	// swagger:ignore
	OsArch string `json:"os_arch"`
	// swagger:ignore
	DisableUsbKbd bool `json:"disable_usb_kbd"`
	// swagger:ignore
	OsProfile jsonutils.JSONObject `json:"__os_profile__"`
	// swagger:ignore
	BillingType string `json:"billing_type"`
	// swagger:ignore
	BillingCycle string `json:"billing_cycle"`

	// swagger:ignore
	// Deprecated
	// 此参数等同于 hypervisor=baremetal
	Baremetal bool `json:"baremetal"`

	// Used to store BaremetalConvertHypervisorTaskId
	ParentTaskId string `json:"__parent_task_id,omitempty"`

	// 指定系统盘默认存储类型, 如果指定宿主机
	// swagger:ignore
	DefaultStorageType string `json:"default_storage_type,omitempty"`

	// 指定用于新建主机的主机镜像ID
	GuestImageID string `json:"guest_image_id"`
}

func (input *ServerCreateInput) AfterUnmarshal() {
	if input.Baremetal {
		input.Hypervisor = HYPERVISOR_BAREMETAL
	}
}

type ServerCloneInput struct {
	apis.Meta

	Name      string `json:"name"`
	AutoStart bool   `json:"auto_start"`

	EipBw         int    `json:"eip_bw,omitzero"`
	EipChargeType string `json:"eip_charge_type,omitempty"`
	Eip           string `json:"eip,omitempty"`

	PreferHostId string `json:"prefer_host_id"`
}

type ServerDeployInput struct {
	apis.Meta

	Id string

	Keypair       string          `json:"keypair"`
	DeleteKeypair *bool           `json:"__delete_keypair__"`
	DeployConfigs []*DeployConfig `json:"deploy_configs"`
	ResetPassword *bool           `json:"reset_password"`
	Password      string          `json:"password"`
	AutoStart     *bool           `json:"auto_start"`
}

type GuestBatchMigrateRequest struct {
	apis.Meta

	GuestIds []string `json:"guest_ids"`

	PreferHostId string `json:"prefer_host_id"`
	// Deprecated
	// swagger:ignore
	PreferHost string `json:"prefer_host" yunion-deprecated-by:"prefer_host_id"`
}

type GuestBatchMigrateParams struct {
	Id          string
	LiveMigrate bool
	RescueMode  bool
	OldStatus   string
}

type HostLoginInfo struct {
	apis.Meta

	Username string `json:"username"`
	Password string `json:"password"`
	Ip       string `json:"ip"`
}
