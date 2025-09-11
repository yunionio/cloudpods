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

type SElasticipCreateInput struct {
	apis.VirtualResourceCreateInput

	// 区域名称或Id, 建议使用Id
	// 在指定区域内创建弹性公网ip
	CloudregionId string `json:"cloudregion_id"`
	// swagger:ignore
	Cloudregion string `json:"cloudregion" yunion-deprecated-by:"cloudregion_id"`
	// swagger:ignore
	Region string `json:"region" yunion-deprecated-by:"cloudregion_id"`
	// swagger:ignore
	RegionId string `json:"region_id" yunion-deprecated-by:"cloudregion_id"`

	// swagger:ignore
	Cloudprovider string `json:"cloudprovider" yunion-deprecated-by:"manager_id"`
	// swagger:ignore
	Manager string `json:"manager" yunion-deprecated-by:"manager_id"`
	// 子订阅Id, 建议使用Id
	// 使用指定子订阅创建弹性公网ip
	// 弹性公网ip和虚拟机在同一区域同一子订阅底下，才可以进行绑定操作
	ManagerId string `json:"manager_id"`

	// 计费类型: 流量或带宽
	//
	//
	//
	// | 平台       |    支持类型            |
	// | ---        |    --------            |
	// |Aliyun      | traffic, bandwidth    |
	// |腾讯云      | traffic                |
	// |Azure       | traffic                |
	// |Google      | traffic, bandwidth    |
	// |Ucloud      | traffic                |
	// |Aws         | traffic                |
	// |华为云      | traffic, bandwidth    |
	// |天翼云      | traffic, bandwidth    |
	// |KVM         | 不支持创建            |
	// |VMware      | 不支持创建            |
	// |ZStack      | traffic                |
	// |OpenStack   | traffic                |
	// default: traffic
	// enum: ["traffic", "bandwidth"]
	ChargeType string `json:"charge_type"`

	Mode string `json:"mode"`

	// swagger:ignore
	Network string `json:"network" yunion-deprecated-by:"network_id"`
	// 子网Id
	// 私有云创建此参数必传,例如Openstack, ZStack
	NetworkId string `json:"network_id"`
	// 指定新建EIP的地址
	IpAddr string `json:"ip_addr"`

	// 线路类型
	//
	//
	//
	// | 平台       |    支持类型            |  说明 |
	// | ---        |    --------            | ---   |
	// |Aliyun      | BGP, BGP_PRO(精品线路)           | 部分区域不支持BGP_PRO|
	// default: BGP
	BgpType string `json:"bgp_type"`

	BandwidthMb int `json:"bandwidth"`
}

type ElasticipDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo

	SElasticip

	// 绑定资源名称
	AssociateName string `json:"associate_name"`

	// 虚拟机内网IP
	ServerPrivateIp string `json:"server_private_ip"`
}

func (self ElasticipDetails) GetMetricTags() map[string]string {
	ret := map[string]string{
		"id":             self.Id,
		"name":           self.Name,
		"status":         self.Status,
		"mode":           self.Mode,
		"cloudregion":    self.Cloudregion,
		"cloudregion_id": self.CloudregionId,
		"region_ext_id":  self.RegionExtId,
		"tenant":         self.Project,
		"tenant_id":      self.ProjectId,
		"brand":          self.Brand,
		"domain_id":      self.DomainId,
		"project_domain": self.ProjectDomain,
		"ip_addr":        self.IpAddr,
		"external_id":    self.ExternalId,
	}
	return AppendMetricTags(ret, self.MetadataResourceInfo, self.ProjectizedResourceInfo)
}

type ElasticipSyncstatusInput struct {
}

type ElasticipAssociateInput struct {
	// 待绑定实例Id
	InstanceId string `json:"instance_id"`
	// swagger:ignore
	InstanceExternalId string `json:"instance_external_id"`

	// swagger:ignore
	Instance string `json:"instance" yunion-deprecated-by:"instance_id"`

	// 实例类型
	// enmu: server, natgateway
	// default: server
	InstanceType string `json:"instance_type"`

	// EIP映射的内网IP地址，可选
	IpAddr string `json:"ip_addr"`
}

type ElasticDissociateInput struct {
	// 注意: 若关联到aws NAT网关后，目前没办法解除关联关系
	// 是否解绑后自动删除弹性公网IP
	// default: false
	AutoDelete bool `json:"auto_delete"`
}

type ElasticipRemoteUpdateInput struct {
	// 是否覆盖替换所有标签
	ReplaceTags *bool `json:"replace_tags" help:"replace all remote tags"`
}
