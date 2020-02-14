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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type ServerFilterListInput struct {
	// 以关联主机（ID或Name）过滤列表
	Server string `json:"server"`
	// swagger:ignore
	// Deprecated
	// Filter by guest Id
	ServerId string `json:"server_id" deprecated-by:"server"`
	// swagger:ignore
	// Deprecated
	// Filter by guest Id
	Guest string `json:"guest" deprecated-by:"server"`
	// swagger:ignore
	// Deprecated
	// Filter by guest Id
	GuestId string `json:"guest_id" deprecated-by:"server"`
}

type ServerListInput struct {
	apis.VirtualResourceListInput

	ManagedResourceListInput
	HostFilterListInput
	NetworkFilterListInput
	BillingFilterListInput
	GroupFilterListInput
	SecgroupFilterListInput
	DiskFilterListInput

	// 只列出裸金属主机
	Baremetal *bool `json:"baremetal"`
	// 只列出GPU主机
	Gpu *bool `json:"gpu"`
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
	OsType string `json:"os_type"`
	// 对列表结果按照磁盘进行排序
	// enum: asc,desc
	OrderByDisk string `json:"order_by_disk"`
	// 对主机列表结果按照宿主机名称进行排序
	// enum: asc,desc
	OrderByHost string `json:"order_by_host"`
	// 列出可以挂载指定EIP的主机
	UsableServerForEip string `json:"usable_server_for_eip"`

	// 按主机资源类型进行排序
	// enum: shared,prepaid,dedicated
	ResourceType string `json:"resource_type"`
	// 返回开启主备机功能的主机
	GetBackupGuestsOnHost *bool `json:"get_backup_guests_on_host"`
	// 根据宿主机 SN 过滤
	HostSn string `json:"host_sn"`
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
	CloudproviderInfo

	// details
	// 网络概要
	Networks string `json:"networks"`
	// 磁盘概要
	Disks string `json:"disks"`
	// 磁盘详情
	DisksInfo *jsonutils.JSONArray `json:"disks_info"`
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
	AttachTime time.Time `attach_time`

	// common
	IsPrepaidRecycle bool `json:"is_prepaid_recycle"`

	// 备机所在宿主机名称
	BackupHostName string `json:"backup_host_name"`
	// 北京所在宿主机状态
	BackupHostStatus string `json:"backup_host_status"`
	// 宿主机名称
	Host string `json:"host"`
	// 宿主机SN
	HostSN     string `json:"host_sn"`
	CanRecycle bool   `json:"can_recycle"`
	// 自动释放时间
	AutoDeleteAt time.Time `json:"auto_delete_at"`
	// 标签
	Metadata map[string]string `json:"metadata"`
	// 磁盘数量
	DiskCount int `json:"disk_count"`
	// 是否支持ISO启动
	CdromSupport bool `json:"cdrom_support"`
}
