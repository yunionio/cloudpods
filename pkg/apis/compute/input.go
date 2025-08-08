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

/*
type RegionalResourceCreateInput struct {
	Cloudregion   string `json:"cloudregion"`
	CloudregionId string `json:"cloudregion_id"`
}

type ManagedResourceInput struct {
	Manager string `json:"manager"`
	ManagerId string `json:"manager_id"`
}
*/

type DeletePreventableCreateInput struct {
	//删除保护,创建的资源默认不允许删除
	//default: true
	DisableDelete *bool `json:"disable_delete"`
}

type KeypairListInput struct {
	apis.UserResourceListInput
	apis.SharableResourceBaseListInput

	// 加密类型
	// example: RSA
	Scheme []string `json:"scheme"`

	// 指纹信息
	// example: 1d:3a:83:4a:a1:f3:75:97:ec:d1:ef:f8:3f:a7:5d:9e
	Fingerprint []string `json:"fingerprint"`
}

type ExternalProjectListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput

	ManagedResourceListInput
}

type RouteTableListInput struct {
	apis.StatusInfrasResourceBaseListInput
	apis.ExternalizedResourceBaseListInput

	VpcFilterListInput

	// filter by type
	Type []string `json:"type"`
}

type SnapshotPolicyCacheListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	ManagedResourceListInput
	RegionalFilterListInput
	SnapshotPolicyFilterListInput

	// filter by snapshotpolicy Id or Name
	//Snapshotpolicy string `json:"snapshotpolicy"`
}

type NetworkInterfaceListInput struct {
	apis.StatusInfrasResourceBaseListInput
	apis.ExternalizedResourceBaseListInput

	ManagedResourceListInput
	RegionalFilterListInput

	// MAC地址
	Mac []string `json:"mac"`
	// 绑定资源类型
	AssociateType []string `json:"associate_type"`
	// 绑定资源Id
	AssociateId []string `json:"associate_id"`
}

type BaremetalagentListInput struct {
	apis.StandaloneResourceListInput
	ZonalFilterListInput

	// 以状态过滤
	Status []string `json:"status"`
	// 以IP地址过滤
	AccessIp []string `json:"access_ip"`
	// 以AgentType过滤
	AgentType []string `json:"agent_type"`
}

type DynamicschedtagListInput struct {
	apis.StandaloneResourceListInput
	SchedtagFilterListInput

	// filter by enabled status
	Enabled *bool `json:"enabled"`
}

type SchedpolicyListInput struct {
	apis.StandaloneResourceListInput
	SchedtagFilterListInput

	//
	Strategy []string `json:"strategy"`

	//
	Enabled *bool `json:"enabled"`
}

type GuestTemplateFilterListInput struct {
	// 主机镜像
	GuestTemplateId string `json:"guest_template_id"`
	// swagger:ignore
	// Deprecated
	GuestTemplate string `json:"guest_template" yunion-deprecated-by:"guest_template_id"`
}

type ServiceCatalogListInput struct {
	apis.SharableVirtualResourceListInput

	GuestTemplateFilterListInput
}

type SnapshotPolicyListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	ManagedResourceListInput
	RegionalFilterListInput

	// 按绑定的磁盘数量排序
	// pattern:asc|desc
	OrderByBindDiskCount string `json:"order_by_bind_disk_count"`
	// 按类型过滤
	Type string `json:"type"`
}

type HostnameInput struct {
	// 主机名
	// 点号（.）和短横线（-）不能作为 HostName 的首尾字符，不能连续使用
	// 字符长度2-60个字符
	// Windows: 字符长度2-15, 允许大小写英文字母, 数字和短横线, 不支持点号（.），不能全是数字
	// 若输入为空，则会根据资源名称自动生成主机名
	// 输入不为空则会自动剔除不符合规则的字符, 并进行校验
	// 若长度大于允许的最大长度，会自动截取
	// required: false
	Hostname string `json:"hostname"`
}
