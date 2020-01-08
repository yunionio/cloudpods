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

type ServerListInput struct {
	apis.VirtualResourceListInput

	ManagedResourceListInput
	HostResourceListInput
	NetworkResourceListInput
	BillingResourceListInput

	// Disk ID or Name
	Disk string `json:"disk"`
	// Show baremetal servers
	Baremetal *bool `json:"baremetal"`
	// Show gpu servers
	Gpu *bool `json:"gpu"`
	// Secgroup ID or Name
	Secgroup string `json:"secgroup"`
	// AdminSecgroup ID or Name
	AdminSecgroup string `json:"admin_security"`
	// Show server of hypervisor choices:"kvm|esxi|container|baremetal|aliyun|azure|aws|huawei|ucloud|zstack|openstack"`
	Hypervisor []string `json:"hypervisor"`
	// Show Servers with EIP
	WithEip *bool `json:"with_eip"`
	// Show Servers without EIP
	WithoutEip *bool `json:"without_eip"`
	// OS Type choices:"linux|windows|vmware"`
	OsType string `json:"os_type"`
	// Order by disk size choices:"asc|desc"
	OrderByDisk string `json:"order_by_disk"`
	// Order by host name choices:"asc|desc"
	OrderByHost string `json:"order_by_host"`
	// Vpc id or name
	Vpc string `json:"vpc"`
	// Eip id or name
	UsableServerForEip string `json:"usable_server_for_eip"`
	// Show Servers without user metadata
	WithoutUserMeta *bool `json:"without_user_meta"`
	// Instance Group ID or Name
	Group string `json:"group"`
	// deprecated: true
	// Filter by instance group Id
	GroupId string `json:"group_id"`
	// Resource type choices:"shared|prepaid|dedicated"
	ResourceType string `json:"resource_type"`
	// return backup guests on a host
	GetBackupGuestsOnHost *bool `json:"get_backup_guests_on_host"`
}

func (input ServerListInput) HypervisorList() []string {
	ret := make([]string, 0)
	if input.Baremetal != nil && *input.Baremetal {
		ret = append(ret, HYPERVISOR_BAREMETAL)
	}
	if len(input.Hypervisor) > 0 {
		ret = append(ret, input.Hypervisor...)
	}
	return ret
}

func (input ServerListInput) GroupStr() string {
	if len(input.Group) > 0 {
		return input.Group
	}
	if len(input.GroupId) > 0 {
		return input.GroupId
	}
	return ""
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
