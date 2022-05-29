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
	"yunion.io/x/onecloud/pkg/util/ansible"
)

type LoadbalancerAgentDeployInput struct {
	apis.Meta

	Host         ansible.Host
	DeployMethod string
}

const (
	DeployMethodYum  = "yum"
	DeployMethodCopy = "copy"
)

type LoadbalancerListenerListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	LoadbalancerFilterListInput
	// filter by acl
	LoadbalancerAclResourceInput

	// filter by backend_group
	BackendGroup string `json:"backend_group"`

	ListenerType []string `json:"listener_type"`
	ListenerPort []int    `json:"listener_port"`

	Scheduler []string `json:"scheduler"`

	Certificate []string `json:"certificate_id"`

	SendProxy []string `json:"send_proxy"`

	AclStatus []string `json:"acl_status"`
	AclType   []string `json:"acl_type"`
}

type LoadbalancerListenerRuleListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	LoadbalancerListenerFilterListInput

	// filter by backend_group
	BackendGroup string `json:"backend_group"`

	// 默认转发策略，目前只有aws用到其它云都是false
	IsDefault *bool `json:"is_default"`

	Domain []string `json:"domain"`
	Path   []string `json:"path"`
}

type LoadbalancerListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.DeletePreventableResourceBaseListInput

	VpcFilterListInput
	ZonalFilterListBase
	NetworkFilterListBase

	// filter by cluster
	Cluster string `json:"cluster"`

	Address []string `json:"address"`
	// 地址类型
	AddressType []string `json:"address_type"`
	// 网络类型
	NetworkType []string `json:"network_type"`
	// 计费类型
	ChargeType []string `json:"charge_type"`
	// 套餐名称
	LoadbalancerSpec []string `json:"loadbalancer_spec"`

	// filter for EIP
	WithEip                  *bool  `json:"with_eip"`
	WithoutEip               *bool  `json:"without_eip"`
	EipAssociable            *bool  `json:"eip_associable"`
	UsableLoadbalancerForEip string `json:"usable_loadbalancer_for_eip"`
}

type LoadbalancerAgentListInput struct {
	apis.StandaloneResourceListInput
	LoadbalancerClusterFilterListInput

	Version []string `json:"version"`
	IP      []string `json:"ip"`
	HaState []string `json:"ha_state"`
}

type LoadbalancerCertificateListInput struct {
	apis.SharableVirtualResourceListInput
	apis.ExternalizedResourceBaseListInput

	UsableResourceListInput
	RegionalFilterListInput
	ManagedResourceListInput

	CommonName              []string `json:"common_name"`
	SubjectAlternativeNames []string `json:"subject_alternative_names"`
}

type LoadbalancerBackendListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput

	LoadbalancerBackendGroupFilterListInput

	// filter by backend server
	Backend string `json:"backend"`

	// filter by backend group
	// BackendGroup string `json:"backend_group"`

	BackendType []string `json:"backend_type"`
	BackendRole []string `json:"backend_role"`
	Address     []string `json:"address"`

	SendProxy []string `json:"send_proxy"`
	Ssl       []string `json:"ssl"`
}

type LoadbalancerBackendGroupListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput

	LoadbalancerFilterListInput

	// filter LoadbalancerBackendGroup with no reference
	NoRef *bool `json:"no_ref"`

	Type []string `json:"type"`
}

type LoadbalancerClusterListInput struct {
	apis.StandaloneResourceListInput

	ZonalFilterListInput
	WireFilterListBase
}

type LoadbalancerAclListInput struct {
	apis.SharableVirtualResourceListInput
	apis.ExternalizedResourceBaseListInput

	ManagedResourceListInput
	RegionalFilterListInput

	//
	Fingerprint string `json:"fingerprint"`
}

type LoadbalancerDetails struct {
	apis.VirtualResourceDetails

	ManagedResourceInfo

	LoadbalancerClusterResourceInfo

	VpcResourceInfoBase
	CloudregionResourceInfo
	ZoneResourceInfoBase
	Zone1ResourceInfoBase
	NetworkResourceInfoBase

	SLoadbalancer

	// 公网IP地址
	Eip string `json:"eip"`

	EipId string `json:"eip_id"`

	// 公网IP地址类型: 弹性、非弹性
	// example: public_ip
	EipMode string `json:"eip_mode"`

	// 后端服务器组名称
	BackendGroup string `json:"backend_group"`
}

type LoadbalancerResourceInfo struct {
	// 负载均衡名称
	Loadbalancer string `json:"loadbalancer"`

	// VPC ID
	VpcId string `json:"vpc_id"`

	VpcResourceInfo

	// 可用区ID
	ZoneId string `json:"zone_id"`

	ZoneResourceInfo

	// cloud provider info
	ManagedResourceInfo
}

type LoadbalancerResourceInput struct {
	// 负载均衡名称
	LoadbalancerId string `json:"loadbalancer_id"`

	// swagger:ignore
	// Deprecated
	Loadbalancer string `json:"loadbalancer" yunion-deprecated-by:"loadbalancer_id"`
}

type LoadbalancerFilterListInput struct {
	VpcFilterListInput

	ZonalFilterListBase

	LoadbalancerResourceInput

	// 以负载均衡名称排序
	OrderByLoadbalancer string `json:"order_by_loadbalancer"`
}

type LoadbalancerCreateInput struct {
	apis.VirtualResourceCreateInput

	// IP地址
	Address string `json:"address"`
	// 地址类型
	AddressType string `json:"address_type"`
	// 网络类型
	NetworkType string `json:"network_type"`

	// 负载均衡集群Id
	ClusterId string `json:"cluster_id"`

	// 计费类型
	ChargeType string `json:"charge_type"`

	// 出口带宽
	EgressMbps int `json:"egress_mbps"`

	// 套餐名称
	LoadbalancerSpec string `json:"loadbalancer_spec"`

	// 弹性公网IP带宽
	// 指定此参数后会创建新的弹性公网IP并绑定到新建的负载均衡
	EipBw int `json:"eip_bw,omitzero"`
	// 弹性公网IP线路类型
	EipBgpType string `json:"eip_bgp_type,omitzero"`
	// 弹性公网IP计费类型
	EipChargeType string `json:"eip_charge_type,omitempty"`
	// 是否跟随主机删除而自动释放
	EipAutoDellocate bool `json:"eip_auto_dellocate,omitempty"`

	// EIP ID
	Eip string `json:"eip"`

	// LB的其他配置信息
	LBInfo jsonutils.JSONObject `json:"lb_info"`

	// 从可用区1
	// required: false
	Zone1 string `json:"zone_1"`

	// SLoadbalancer

	VpcResourceInput
	// Vpc         string `json:"vpc"`
	ZoneResourceInput
	// Zone        string `json:"zone"`
	CloudregionResourceInput
	// Cloudregion string `json:"cloudregion"`
	NetworkResourceInput
	// Network     string `json:"network"`
	CloudproviderResourceInput
	// Manager     string `json:"manager"`
}

type LoadbalancerRemoteUpdateInput struct {
	// 是否覆盖替换所有标签
	ReplaceTags *bool `json:"replace_tags" help:"replace all remote tags"`
}

type LoadbalancerAssociateEipInput struct {
	// 弹性公网IP的ID
	EipId string `json:"eip_id"`

	// 弹性IP映射的内网IP地址，可选
	IpAddr string `json:"ip_addr"`
}

type LoadbalancerCreateEipInput struct {
	// 计费方式，traffic or bandwidth
	ChargeType string `json:"charge_type"`

	// Bandwidth
	Bandwidth int64 `json:"bandwidth"`

	// bgp_type
	BgpType string `json:"bgp_type"`

	// auto_dellocate
	AutoDellocate *bool `json:"auto_dellocate"`

	// 弹性IP映射的内网IP地址，可选
	// IpAddr string `json:"ip_addr"`
}

type LoadbalancerDissociateEipInput struct {
	// 是否自动释放
	AudoDelete *bool `json:"auto_delete"`
}
