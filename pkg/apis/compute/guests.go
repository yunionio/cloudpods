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
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/billing"
	"yunion.io/x/onecloud/pkg/apis/cloudcommon/db"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/httperrors"
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
	// 只列出透传了 GPU 的主机
	Gpu *bool `json:"gpu"`
	// 只列出透传了 USB 的主机
	Usb *bool `json:"usb"`
	// 自定义 PCI 设备类型
	CustomDevType string `json:"custom_dev_type"`
	// 通用虚拟机
	Normal *bool `json:"normal"`
	// 只列出还有备份机的主机
	Backup *bool `json:"bakcup"`
	// 列出指定类型的主机
	// enum: normal,gpu,usb,backup
	ServerType []string `json:"server_type"`
	// 列出管理安全组为指定安全组的主机
	AdminSecgroup string `json:"admin_security"`
	// 列出Hypervisor为指定值的主机
	// enum: kvm,esxi,baremetal,aliyun,azure,aws,huawei,ucloud,zstack,openstack,google,ctyun,cloudpods,ecloud,jdcloud,remotefile`
	Hypervisor []string `json:"hypervisor"`
	// 列出绑定了弹性IP（EIP）的主机
	WithEip *bool `json:"with_eip"`
	// 列出未绑定弹性IP（EIP）的主机
	WithoutEip *bool `json:"without_eip"`
	// 列出可绑定弹性IP的主机
	EipAssociable *bool `json:"eip_associable"`
	// 列出操作系统为指定值的主机
	// enum: linux,windows,vmware
	OsType []string `json:"os_type"`

	// 对列表结果按照磁盘大小进行排序
	// enum: asc,desc
	OrderByDisk string `json:"order_by_disk"`

	OrderByIp string `json:"order_by_ip"`
	// 根据ip查找机器
	IpAddr string `json:"ip_addr" yunion-deprecated-by:"ip_addrs"`
	// 根据多个ip查找机器
	IpAddrs []string `json:"ip_addrs"`

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

	// swagger: ignore
	Image string `json:"image" yunion-deprecated-by:"image_id"`
	// 关机且停机不收费情况下不允许重装系统
	// 镜像 id
	// required: true
	ImageId string `json:"image_id"`
	// swagger: ignore
	Keypair string `json:"keypair" yunion-deprecated-by:"keypair_id"`
	// 秘钥Id
	KeypairId     string `json:"keypair_id"`
	ResetPassword *bool  `json:"reset_password"`
	Password      string `json:"password"`
	AutoStart     *bool  `json:"auto_start"`
	AllDisks      *bool  `json:"all_disks"`
}

type ServerResumeInput struct {
	apis.Meta
}

type ServerDetails struct {
	apis.VirtualResourceDetails
	apis.EncryptedResourceDetails

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
	// 主备机同步状态
	BackupGuestSyncStatus string `json:"backup_guest_sync_status"`

	// 是否可以回收
	CanRecycle bool `json:"can_recycle"`

	// 自动释放时间
	AutoDeleteAt time.Time `json:"auto_delete_at"`
	// 磁盘数量
	DiskCount int `json:"disk_count"`
	// 是否支持ISO启动
	CdromSupport bool `json:"cdrom_support"`
	//是否支持Floppy启动
	FloppySupport bool `json:"floppy_support"`

	// 磁盘大小
	// example:30720
	DiskSizeMb int64 `json:"disk"`
	// IP地址列表字符串
	// example: 10.165.2.1,172.16.8.1
	IPs string `json:"ips"`
	// VIP
	Vip string `json:"vip"`
	// VIP's eip
	VipEip string `json:"vip_eip"`
	// mac地址信息
	Macs string `json:"macs"`
	// 网卡信息
	Nics []GuestnetworkShortDesc `json:"nics"`

	// 归属VPC
	Vpc string `json:"vpc"`
	// 归属VPC ID
	VpcId string `json:"vpc_id"`
	// Vpc外网访问模式
	VpcExternalAccessMode string `json:"vpc_external_access_mode"`

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
	Cdrom []Cdrom `json:"cdrom"`

	//Floppy信息
	Floppy []Floppy `json:"floppy"`

	// 主机在伸缩组中的状态
	ScalingStatus string `json:"scaling_status"`

	// 伸缩组id
	ScalingGroupId string `json:"scaling_group_id"`

	// 监控上报URL
	MonitorUrl string `json:"monitor_url"`
}

type Floppy struct {
	Ordinal int    `json:"ordinal"`
	Detail  string `json:"detail"`
}

type Cdrom struct {
	Ordinal   int    `json:"ordinal"`
	Detail    string `json:"detail"`
	BootIndex int8   `json:"boot_index"`
}

func (self ServerDetails) GetMetricTags() map[string]string {
	ret := map[string]string{
		"id":                  self.Id,
		"res_type":            "guest",
		"is_vm":               "true",
		"paltform":            self.Hypervisor,
		"host":                self.Host,
		"host_id":             self.HostId,
		"vm_id":               self.Id,
		"vm_name":             self.Name,
		"zone":                self.Zone,
		"zone_id":             self.ZoneId,
		"zone_ext_id":         self.ZoneExtId,
		"os_type":             self.OsType,
		"status":              self.Status,
		"cloudregion":         self.Cloudregion,
		"cloudregion_id":      self.CloudregionId,
		"region_ext_id":       self.RegionExtId,
		"tenant":              self.Project,
		"tenant_id":           self.ProjectId,
		"brand":               self.Brand,
		"vm_scaling_group_id": self.ScalingGroupId,
		"domain_id":           self.DomainId,
		"project_domain":      self.ProjectDomain,
		"account":             self.Account,
		"account_id":          self.AccountId,
		"external_id":         self.ExternalId,
	}
	for k, v := range self.Metadata {
		if strings.HasPrefix(k, db.USER_TAG_PREFIX) {
			ret[k] = v
		}
	}
	return ret
}

func (self ServerDetails) GetMetricPairs() map[string]string {
	ret := map[string]string{
		"vcpu_count": fmt.Sprintf("%d", self.VcpuCount),
		"vmem_size":  fmt.Sprintf("%d", self.VmemSize),
		"disk":       fmt.Sprintf("%d", self.DiskSizeMb),
	}
	return ret
}

// GuestDiskInfo describe the information of disk on the guest.
type GuestDiskInfo struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	FsFormat    string `json:"fs,omitempty"`
	DiskType    string `json:"disk_type"`
	Index       int8   `json:"index"`
	BootIndex   int8   `json:"boot_index"`
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
	StorageId   string `json:"storage_id"`
}

func (self GuestDiskInfo) ShortDesc() string {
	fs := ""
	if len(self.ImageId) > 0 {
		fs = "root"
	} else if len(self.FsFormat) > 0 {
		fs = self.FsFormat
	} else {
		fs = "none"
	}
	return fmt.Sprintf("disk%d:%dM/%s/%s/%s/%s/%s", self.Index, self.SizeMb,
		self.DiskFormat, self.Driver, self.CacheMode, self.AioMode, fs)
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
	// 若公有云本身不支持自动续费, 则在本地周期(默认三小时)检查快过期虚拟机并进行续费,续费周期根据设置，请避免使用特殊的计费周期，避免续费失败
	AutoRenew bool `json:"auto_renew"`
	// 续费周期
	// example: 1Y, 1M, 1W
	// default: 1M
	// 腾讯云仅支持1M
	// 阿里云支持 1, 2, 3Y, 1, 2, 3, 6, 12M, 1, 2, 3, 4W
	Duration string `json:"duration"`
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
	// swagger: ignore
	PreferHost   string `json:"prefer_host" yunion-deprecated-by:"prefer_host_id"`
	PreferHostId string `json:"prefer_host_id"`
	AutoStart    bool   `json:"auto_start"`
	IsRescueMode bool   `json:"rescue_mode"`
}

type GuestLiveMigrateInput struct {
	// swagger: ignore
	PreferHost string `json:"prefer_host" yunion-deprecated-by:"prefer_host_id"`
	// 指定期望的迁移目标宿主机
	PreferHostId string `json:"prefer_host_id"`
	// 是否跳过CPU检查，默认要做CPU检查
	SkipCpuCheck *bool `json:"skip_cpu_check"`
	// 是否跳过kernel检查
	SkipKernelCheck *bool `json:"skip_kernel_check"`
	// 是否启用 tls
	EnableTLS *bool `json:"enable_tls"`

	// 迁移带宽限制
	MaxBandwidthMb *int64 `json:"max_bandwidth_mb"`
	// 快速完成，内存同步一定周期后调整 downtime
	QuicklyFinish         *bool `json:"quickly_finish"`
	KeepDestGuestOnFailed *bool `json:"keep_dest_guest_on_failed"`
}

type GuestSetSecgroupInput struct {
	// 安全组Id列表
	// 实例必须处于运行,休眠或者关机状态
	//
	//
	// | 平台         | 最多绑定安全组数量    |
	// |-------------|-------------------    |
	// | Azure       | 1                    |
	// | VMware      | 不支持安全组            |
	// | Baremetal   | 不支持安全组            |
	// | ZStack         | 1                    |
	// | 其他         | 5                    |
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
	// | 平台        | 最多绑定安全组数量    |
	// |-------------|-------------------    |
	// | Azure       | 1                    |
	// | VMware      | 不支持安全组            |
	// | Baremetal   | 不支持安全组            |
	// | ZStack      | 1                    |
	// | 其他        | 5                    |
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

	// 弹性IP映射的内网IP地址，可选
	IpAddr string `json:"ip_addr"`
}

type ServerCreateEipInput struct {
	// 计费方式，traffic or bandwidth
	ChargeType string `json:"charge_type"`

	// Bandwidth
	Bandwidth int64 `json:"bandwidth"`

	// bgp_type
	BgpType string `json:"bgp_type"`

	// auto_dellocate
	AutoDellocate *bool `json:"auto_dellocate"`

	// 弹性IP映射的内网IP地址，可选
	IpAddr string `json:"ip_addr"`
}

type ServerDissociateEipInput struct {
	// 是否自动释放
	AudoDelete *bool `json:"auto_delete"`
}

type ServerResetInput struct {
	InstanceSnapshot string `json:"instance_snapshot"`
	// 自动启动
	AutoStart *bool `json:"auto_start"`
	// 恢复内存
	WithMemory bool `json:"with_memory"`
}

type ServerStopInput struct {
	// 是否强制关机
	IsForce bool `json:"is_force"`

	// 是否关机停止计费, 若平台不支持停止计费，此参数无作用
	// 目前仅阿里云，腾讯云此参数生效
	StopCharging bool `json:"stop_charging"`
}

type ServerSaveImageInput struct {
	// 镜像名称
	Name         string
	GenerateName string
	Notes        string
	IsPublic     *bool
	// 镜像格式
	Format string

	// 保存镜像后是否自动启动,若实例状态为运行中,则会先关闭实例
	// 公有云若支持开机保存镜像，此参数则不生效
	// default: false
	AutoStart bool
	// swagger: ignore
	Restart bool

	// swagger: ignore
	OsType string

	// swagger: ignore
	OsArch string

	// swagger: ignore
	ImageId string
}

type ServerSaveGuestImageInput struct {
	imageapi.GuestImageCreateInputBase

	// 保存镜像后是否自动启动
	AutoStart *bool `json:"auto_start"`
}

type ServerDeleteInput struct {
	// 是否越过回收站直接删除
	// default: false
	OverridePendingDelete bool

	// 是否仅删除本地资源
	// default: false
	Purge bool

	// 是否删除快照
	// default: false
	DeleteSnapshots bool

	// 是否删除关联的EIP
	// default: false
	DeleteEip bool

	// 是否删除关联的数据盘
	// default: false
	DeleteDisks bool
}

type ServerDetachnetworkInput struct {
	// 是否保留IP地址(ip地址会进入到预留ip)
	Reserve bool `json:"reserve"`
	// 通过IP子网地址, 优先级最高
	NetId string `json:"net_id"`
	// 通过IP解绑网卡, 优先级高于mac
	IpAddr string `json:"ip_addr"`
	// 通过Mac解绑网卡, 优先级低于ip_addr
	Mac string `json:"mac"`
}

type ServerMigrateForecastInput struct {
	PreferHostId string `json:"prefer_host_id"`
	// Deprecated
	PreferHost      string `json:"prefer_host" yunion-deprecated-by:"prefer_host_id"`
	LiveMigrate     bool   `json:"live_migrate"`
	SkipCpuCheck    bool   `json:"skip_cpu_check"`
	SkipKernelCheck bool   `json:"skip_kernel_check"`
	ConvertToKvm    bool   `json:"convert_to_kvm"`
	IsRescueMode    bool   `json:"is_rescue_mode"`
}

type ServerResizeDiskInput struct {
	// swagger: ignore
	Disk string `json:"disk" yunion-deprecated-by:"disk_id"`
	// 磁盘Id
	DiskId string `json:"disk_id"`

	DiskResizeInput
}

type ServerMigrateNetworkInput struct {
	// Source network Id
	Src string `json:"src"`
	// Destination network Id
	Dest string `json:"dest"`
}

type ServerDeployInput struct {
	apis.Meta

	// swagger: ignore
	Keypair string `json:"keypair" yunion-deprecated-by:"keypair_id"`
	// 秘钥Id
	KeypairId string `json:"keypair_id"`

	// 清理指定公钥
	// 若指定的秘钥Id和虚拟机的秘钥Id不相同, 则清理旧的公钥
	DeletePublicKey string `json:"delete_public_key"`
	// 解绑当前虚拟机秘钥, 并清理公钥信息
	DeleteKeypair bool `json:"__delete_keypair__"`
	// 生成随机密码, 优先级低于password
	ResetPassword bool `json:"reset_password"`
	// 重置指定密码
	Password string `json:"password"`
	// 部署完成后是否自动启动
	// 若虚拟机重置密码后需要重启生效，并且当前虚拟机状态为running, 此参数默认为true
	// 若虚拟机状态为ready, 指定此参数后，部署完成后，虚拟机会自动启动
	AutoStart bool `json:"auto_start"`
	// swagger: ignore
	Restart bool `json:"restart"`

	// swagger: ignore
	DeployConfigs []*DeployConfig `json:"deploy_configs"`
	// swagger: ignore
	DeployTelegraf bool `json:"deploy_telegraf"`
}

type ServerUserDataInput struct {
	UserData string `json:"user_data"`
}

type ServerAttachDiskInput struct {
	DiskId string `json:"disk_id"`

	BootIndex *int8 `json:"boot_index"`
}

type ServerDetachDiskInput struct {
	// 磁盘Id，若磁盘未挂载在虚拟机上，不返回错误
	DiskId string `json:"disk_id"`
	// 是否保留磁盘
	// default: false
	KeepDisk bool `json:"keep_disk"`
}

type ServerChangeConfigInput struct {
	// 关机且停机不收费情况下不允许调整配置
	// 实例类型, 优先级高于vcpu_count和vmem_size
	InstanceType string `json:"instance_type"`
	// swagger: ignore
	Sku string `json:"sku" yunion-deprecated-by:"instance_type"`
	// swagger: ignore
	Flavor string `json:"flavor" yunion-deprecated-by:"instance_type"`

	// cpu大小
	VcpuCount int `json:"vcpu_count"`
	// 内存大小, 1024M, 1G
	VmemSize string `json:"vmem_size"`

	// 调整完配置后是否自动启动
	AutoStart bool `json:"auto_start"`

	Disks []DiskConfig `json:"disks"`
}

type ServerUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput

	HostnameInput

	// 删除保护开关
	DisableDelete *bool `json:"disable_delete"`
	// 启动顺序
	BootOrder *string `json:"boot_order"`
	// 关机执行操作
	ShutdownBehavior *string `json:"shutdown_behavior"`
	Vga              *string `json:"vga"`
	Vdi              *string `json:"vdi"`
	Machine          *string `json:"machine"`
	Bios             *string `json:"bios"`

	SrcIpCheck  *bool `json:"src_ip_check"`
	SrcMacCheck *bool `json:"src_mac_check"`

	SshPort int `json:"ssh_port"`

	// swagger: ignore
	ProgressMbps float32 `json:"progress_mbps"`
}

type GuestJsonDesc struct {
	Name           string `json:"name"`
	Hostname       string `json:"hostname"`
	Description    string `json:"description"`
	UUID           string `json:"uuid"`
	Mem            int    `json:"mem"`
	Cpu            int    `json:"cpu"`
	Vga            string `json:"vga"`
	Vdi            string `json:"vdi"`
	Machine        string `json:"machine"`
	Bios           string `json:"bios"`
	BootOrder      string `json:"boot_order"`
	SrcIpCheck     bool   `json:"src_ip_check"`
	SrcMacCheck    bool   `json:"src_mac_check"`
	IsMaster       *bool  `json:"is_master"`
	IsSlave        *bool  `json:"is_slave"`
	IsVolatileHost bool   `json:"is_volatile_host"`
	HostId         string `json:"host_id"`

	IsolatedDevices []*IsolatedDeviceJsonDesc `json:"isolated_devices"`

	Domain string `json:"domain"`

	Nics  []*GuestnetworkJsonDesc `json:"nics"`
	Disks []*GuestdiskJsonDesc    `json:"disks"`

	Cdrom  *GuestcdromJsonDesc   `json:"cdrom"`
	Cdroms []*GuestcdromJsonDesc `json:"cdroms"`

	Floppys []*GuestfloppyJsonDesc `json:"floppys"`

	Tenant        string `json:"tenant"`
	TenantId      string `json:"tenant_id"`
	DomainId      string `json:"domain_id"`
	ProjectDomain string `json:"project_domain"`

	Keypair string `json:"keypair"`
	Pubkey  string `json:"pubkey"`

	NetworkRoles []string `json:"network_roles"`

	Secgroups          []*SecgroupJsonDesc `json:"secgroups"`
	SecurityRules      string              `json:"security_rules"`
	AdminSecurityRules string              `json:"admin_security_rules"`

	ExtraOptions jsonutils.JSONObject `json:"extra_options"`

	Kvm string `json:"kvm"`

	Zone   string `json:"zone"`
	ZoneId string `json:"zone_id"`

	OsName string `json:"os_name"`

	Metadata       map[string]string `json:"metadata"`
	UserData       string            `json:"user_data"`
	PendingDeleted bool              `json:"pending_deleted"`

	ScalingGroupId string `json:"scaling_group_id"`

	// baremetal
	DiskConfig  jsonutils.JSONObject    `json:"disk_config"`
	NicsStandby []*GuestnetworkJsonDesc `json:"nics_standby"`

	// esxi
	InstanceSnapshotInfo struct {
		InstanceSnapshotId string `json:"instance_snapshot_id"`
		InstanceId         string `json:"instance_id"`
	} `json:"instance_snapshot_info"`

	EncryptKeyId string `json:"encrypt_key_id,omitempty"`

	IsDaemon bool `json:"is_daemon"`
}

type ServerSetBootIndexInput struct {
	// key index, value boot_index
	Disks map[string]int8 `json:"disks"`
	// key ordinal, value boot_index
	Cdroms map[string]int8 `json:"cdroms"`
}

type ServerChangeStorageInput struct {
	TargetStorageId string `json:"target_storage_id"`
	KeepOriginDisk  bool   `json:"keep_origin_disk"`
}

type ServerChangeStorageInternalInput struct {
	ServerChangeStorageInput
	Disks        []string `json:"disks"`
	GuestRunning bool     `json:"guest_running"`
	DiskCount    int      `json:"disk_count"`
}

type ServerChangeDiskStorageInput struct {
	DiskId          string `json:"disk_id"`
	TargetStorageId string `json:"target_storage_id"`
	KeepOriginDisk  bool   `json:"keep_origin_disk"`
}

type ServerChangeDiskStorageInternalInput struct {
	ServerChangeDiskStorageInput
	StorageId      string             `json:"storage_id"`
	TargetDiskId   string             `json:"target_disk_id"`
	DiskFormat     string             `json:"disk_format"`
	GuestRunning   bool               `json:"guest_running"`
	TargetDiskDesc *GuestdiskJsonDesc `json:"target_disk_desc"`

	// clone progress
	CompletedDiskCount int `json:"completed_disk_count"`
	CloneDiskCount     int `json:"disk_count"`
}

type ServerSetExtraOptionInput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (o ServerSetExtraOptionInput) Validate() error {
	if len(o.Key) == 0 {
		return errors.Wrap(httperrors.ErrBadRequest, "empty key")
	}
	if len(o.Value) == 0 {
		return errors.Wrap(httperrors.ErrBadRequest, "empty value")
	}
	return nil
}

type ServerDelExtraOptionInput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (o ServerDelExtraOptionInput) Validate() error {
	if len(o.Key) == 0 {
		return errors.Wrap(httperrors.ErrBadRequest, "empty key")
	}
	return nil
}

type ServerSnapshotAndCloneInput struct {
	ServerCreateSnapshotParams

	// number of cloned servers
	// 数量
	Count *int `json:"count"`

	// Whether auto start the cloned server
	// 是否自动启动
	AutoStart *bool `json:"auto_start"`

	// Whether delete instance snapshot automatically
	// 是否自动删除主机快照
	AutoDeleteInstanceSnapshot *bool `json:"auto_delete_instance_snapshot"`

	// ignore
	InstanceSnapshotId string `json:"instance_snapshot_id"`
}

type ServerInstanceSnapshot struct {
	ServerCreateSnapshotParams
	WithMemory bool `json:"with_memory"`
}

type ServerCreateSnapshotParams struct {
	Name         string `json:"name"`
	GenerateName string `json:"generate_name"`
}

type ServerCPUSetInput struct {
	// Specifies the CPUs that tasks in this cgroup are permitted to access.
	CPUS []int `json:"cpus"`
}

type ServerCPUSetResp struct{}

type ServerCPUSetRemoveInput struct{}

type ServerCPUSetRemoveResp struct {
	Done  bool   `json:"done"`
	Error string `json:"error"`
}

type ServerGetCPUSetCoresInput struct{}

type ServerGetCPUSetCoresResp struct {
	PinnedCores   []int `json:"pinned_cores"`
	HostCores     []int `json:"host_cores"`
	HostUsedCores []int `json:"host_used_cores"`
}

type ServerMonitorInput struct {
	COMMAND string
	QMP     bool
}

type ServerQemuInfo struct {
	Version string `json:"version"`
	Cmdline string `json:"cmdline"`
}

type ServerQgaSetPasswordInput struct {
	Username string
	Password string
}

type ServerQgaCommandInput struct {
	Command string
}

type ServerSetPasswordInput struct {
	Username string
	Password string

	// deploy params
	ResetPassword bool
	AutoStart     bool
}

type ServerInsertVfdInput struct {
	FloppyOrdinal int64  `json:"floppy_ordinal"`
	ImageId       string `json:"image_id"`
}

type ServerEjectVfdInput struct {
	FloppyOrdinal int64  `json:"floppy_ordinal"`
	ImageId       string `json:"image_id"`
}

type ServerSetLiveMigrateParamsInput struct {
	MaxBandwidthMB  *int64
	DowntimeLimitMS *int64
}
