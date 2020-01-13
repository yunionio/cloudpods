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
