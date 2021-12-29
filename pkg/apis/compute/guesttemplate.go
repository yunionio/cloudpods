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

type GuestTemplateInput struct {
	// description: the content of guest template
	// required: true
	Content *jsonutils.JSONDict `json:"content"`

	VmemSize      int    `json:"vmem_size"`
	VcpuCount     int    `json:"vcpu_count"`
	OsType        string `json:"os_type"`
	Hypervisor    string `json:"hypervisor"`
	CloudregionId string `json:"cloudregion_id"`
	VpcId         string `json:"vpc_id"`
	ImageType     string `json:"image_type"`
	ImageId       string `json:"image_id"`
	InstanceType  string `json:"instance_type"`
	BillingType   string `json:"billing_type"`
}

type GuestTemplateCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	GuestTemplateInput
}

type GuestTemplateUpdateInput struct {
	apis.SharableVirtualResourceBaseUpdateInput

	GuestTemplateInput
}

type GuestTemplateDetails struct {
	apis.SharableVirtualResourceDetails
	CloudregionResourceInfo
	VpcResourceInfo
	SGuestTemplate

	Secgroups []string `json:"secgroups"`
	Zone      string   `json:"zone"`
	ZoneId    string   `json:"zone_id"`
	Brand     string   `json:"brand"`

	ConfigInfo GuestTemplateConfigInfo `json:"config_info"`
}

type GuestTemplateListInput struct {
	apis.SharableVirtualResourceListInput
	RegionalFilterListInput
	VpcFilterListInput
	BillingType string `json:"billing_type"`
	Brand       string `json:"brand"`
}

type GuestTemplateConfigInfo struct {
	Metadata             map[string]string      `json:"metadata"`
	Secgroup             string                 `json:"secgroup"`
	Sku                  GuestTemplateSku       `json:"sku"`
	Disks                []GuestTemplateDisk    `json:"disks"`
	Keypair              string                 `json:"keypair"`
	Nets                 []GuestTemplateNetwork `json:"nets"`
	IsolatedDeviceConfig []IsolatedDeviceConfig `json:"isolated_device_config"`
	Image                string                 `json:"image"`
	ResetPassword        bool                   `json:"reset_password"`
}

type GuestTemplateDisk struct {
	Backend  string `json:"backend"`
	DiskType string `json:"disk_type"`
	Index    int    `json:"index"`
	SizeMb   int    `json:"size_mb"`
}

type GuestTemplateNetwork struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	GuestIpStart string `json:"guest_ip_start"`
	GuestIpEnd   string `json:"guest_ip_end"`
	VlanId       int    `json:"vlan_id"`
	VpcId        string `json:"vpc_id"`
	VpcName      string `json:"vpc_name"`
}

type GuestTemplateSku struct {
	Name                 string `json:"name"`
	CpuCoreCount         int    `json:"cpu_core_count"`
	MemorySizeMb         int    `json:"memory_size_mb"`
	InstanceTypeCategory string `json:"instance_type_category"`
	InstanceTypeFamily   string `json:"instance_type_family"`
}

type GuestTemplatePublicInput struct {
	apis.Meta

	// description: the scope about public operator
	// required: true
	// example: system
	Scope string `json:"scope"`
}

type GuestTemplateValidateInput struct {
	apis.Meta

	// description: the hypervisor about guest template
	Hypervisor string `json:"hypervisor"`

	CloudregionId string `json:"cloudregion_id"`
}

type GuestTemplateResourceInfo struct {
	// 主机模板名称
	GuestTemplate string `json:"guest_template"`

	// 主机模板ID
	GuestTemplateId string `json:"guest_template_id"`
}
