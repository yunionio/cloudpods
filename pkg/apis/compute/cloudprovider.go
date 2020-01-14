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
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type CloudproviderInfo struct {
	// 云平台
	// example: Google
	Provider string `json:"provider,omitempty"`

	// 云平台品牌
	// example: Google
	Brand string `json:"brand,omitempty"`

	// 云账号名称
	// example: google-account
	Account string `json:"account,omitempty"`

	// 云账号Id
	// example: 4d3c8979-9dd0-439b-8d78-36fe1ab1666c
	AccountId string `json:"account_id,omitempty"`

	// 子订阅名称
	// example: google-account
	Manager string `json:"manager,omitempty"`

	// 子订阅Id
	// example: fa4aaf88-aed8-422d-84e7-56dea533b364
	ManagerId string `json:"manager_id,omitempty"`

	// 子订阅所在项目名称
	// example: system
	ManagerProject string `json:"manager_project,omitempty"`
	// 子订阅所在项目Id
	// example: 4d3c8979-9dd0-439b-8d78-36fe1ab1666c
	ManagerProjectId string `json:"manager_project_id,omitempty"`

	// 子订阅所在域名称
	// example: Default
	ManagerDomain string `json:"manager_domain,omitempty"`

	// 子订阅所在域Id
	// example: default
	ManagerDomainId string `json:"manager_domain_id,omitempty"`

	// 区域名称
	// example: 腾讯云 华南地区(广州)
	Region string `json:"region,omitempty"`
	// 区域Id
	// example: 6151c89b-77f2-4d43-8ef9-cd03d604a16b
	RegionId string `json:"region_id,omitempty"`
	// 区域Id
	// example: 6151c89b-77f2-4d43-8ef9-cd03d604a16b
	CloudregionId string `json:"cloudregion_id,omitempty"`
	// 区域外部Id
	// Qcloud/ap-guangzhou
	RegionExternalId string `json:"region_external_id,omitempty"`
	// 区域外部Id(不携带平台信息)
	// example: ap-guangzhou
	RegionExtId string `json:"region_ext_id,omitempty"`

	// 可用区名称
	// example: 腾讯云 广州四区
	Zone string `json:"zone,omitempty"`
	// 可用区Id
	// example: 336ac6d2-b80d-43bb-86d5-1ebf474da8d4
	ZoneId string `json:"zone_id,omitempty"`
	// 可用区外部Id
	// example: ap-guangzhou-4
	ZoneExtId string `json:"zone_ext_id,omitempty"`

	// 云环境
	// example: public
	CloudEnv string `json:"cloud_env,omitempty"`
}

type CloudproviderDetails struct {
	apis.StandaloneResourceDetails
	SCloudprovider

	// 云账号名称
	// example: google-account
	Cloudaccount string `json:"cloudaccount"`
	// 子订阅同步状态
	SyncStatus2 string `json:"sync_status2"`
	// 支持服务列表
	Capabilities []string `json:"capabilities"`

	// 云主机数量
	// example: 1
	GuestCount int `json:"guest_count"`
	// 宿主机数量
	// example: 2
	HostCount int `json:"host_count"`
	// 虚拟私有网络数量
	// example: 4
	VpcCount int `json:"vpc_count"`
	// 块存储梳理
	// example: 4
	StorageCount int `json:"storage_count"`
	// 存储缓存数量
	// example: 1
	StorageCacheCount int `json:"storagecache_count"`
	// 弹性公网IP数量
	// example: 12
	EipCount int `json:"eip_count"`
	// 快照数量
	// example: 0
	SnapshotCount int `json:"snapshot_count"`
	// 负载均衡器数量
	// example: 2
	LoadbalancerCount int `json:"loadbalancer_count"`
	// 项目数量
	ProjectCount int `json:"project_count"`
	// 同步区域数量
	SyncRegionCount int `json:"sync_region_count"`

	// 子订阅品牌信息
	Brand string `json:"brand"`
}

type ManagedResourceListInput struct {
	// 列出关联指定云订阅(ID或Name)的资源
	Cloudprovider string `json:"cloudprovider"`
	// List objects belonging to the cloud provider
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Manager string `json:"manager" deprecated-by:"cloudprovider"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	ManagerId string `json:"manager_id" deprecated-by:"cloudprovider"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	CloudproviderId string `json:"cloudprovider_id" deprecated-by:"cloudprovider"`

	// 列出关联指定云账号(ID或Name)的资源
	Cloudaccount string `json:"cloudaccount"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	CloudaccountId string `json:"cloudaccount_id" deprecated-by:"cloudaccount"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Account string `json:"account" deprecated-by:"cloudaccount"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	AccountId string `json:"account_id" deprecated-by:"cloudaccount"`

	// 列出指定云平台的资源，支持的云平台如下
	//
	// | Provider  | 开始支持版本 | 平台                                |
	// |-----------|------------|-------------------------------------|
	// | OneCloud  | 0.0        | OneCloud内置私有云，包括KVM和裸金属管理 |
	// | VMware    | 1.2        | VMware vCenter                      |
	// | OpenStack | 2.6        | OpenStack M版本以上私有云             |
	// | ZStack    | 2.10       | ZStack私有云                         |
	// | Aliyun    | 2.0        | 阿里云                               |
	// | Aws       | 2.3        | Amazon AWS                          |
	// | Azure     | 2.2        | Microsoft Azure                     |
	// | Google    | 2.13       | Google Cloud Platform               |
	// | Qcloud    | 2.3        | 腾讯云                               |
	// | Huawei    | 2.5        | 华为公有云                           |
	// | Ucloud    | 2.7        | UCLOUD                               |
	// | Ctyun     | 2.13       | 天翼云                               |
	// | S3        | 2.11       | 通用s3对象存储                        |
	// | Ceph      | 2.11       | Ceph对象存储                         |
	// | Xsky      | 2.11       | XSKY启明星辰Ceph对象存储              |
	//
	// enum: OneCloud,VMware,Aliyun,Qcloud,Azure,Aws,Huawei,OpenStack,Ucloud,ZStack,Google,Ctyun,S3,Ceph,Xsky"
	Providers []string `json:"providers"`
	// swagger:ignore
	// Deprecated
	Provider []string `json:"provider" deprecated-by:"providers"`

	// 列出指定云平台品牌的资源，一般来说brand和provider相同，除了以上支持的provider之外，还支持以下band
	//
	// |   Brand  | Provider | 说明        |
	// |----------|----------|------------|
	// | DStack   | ZStack   | 滴滴云私有云 |
	//
	Brands []string `json:"brands"`
	// swagger:ignore
	// Deprecated
	Brand []string `json:"brand" deprecated-by:"brands"`

	// 列出指定云环境的资源，支持云环境如下：
	//
	// | CloudEnv  | 说明   |
	// |-----------|--------|
	// | public    | 公有云  |
	// | private   | 私有云  |
	// | onpremise | 本地IDC |
	//
	// enum: public,private,onpremise
	CloudEnv string `json:"cloud_env"`

	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	PublicCloud bool `json:"public_cloud"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	IsPublic bool `json:"is_public"`

	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	PrivateCloud bool `json:"private_cloud"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	IsPrivate bool `json:"is_private"`

	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	IsOnPremise bool `json:"is_on_premise"`

	// 过滤资源，是否为非OneCloud内置私有云管理的资源
	// default: false
	IsManaged bool `json:"is_managed"`
}

func (input *ManagedResourceListInput) AfterUnmarshal() {
	if len(input.CloudEnv) > 0 {
		return
	}
	if input.PublicCloud || input.IsPublic {
		input.CloudEnv = CLOUD_ENV_PUBLIC_CLOUD
	} else if input.PrivateCloud || input.IsPrivate {
		input.CloudEnv = CLOUD_ENV_PRIVATE_CLOUD
	} else if input.IsOnPremise {
		input.CloudEnv = CLOUD_ENV_ON_PREMISE
	}
}

type CapabilityListInput struct {
	// 根据该云平台的功能对云账号或云订阅进行过滤
	Capability []string `json:"capability"`
	// swagger:ignore
	// Deprecated
	// filter by HasObjectStorage
	HasObjectStorage *bool `json:"has_object_storage"`
}

type CloudproviderListInput struct {
	apis.EnabledStatusStandaloneResourceListInput

	ManagedResourceListInput

	UsableResourceListInput

	CapabilityListInput
}

func (input *CapabilityListInput) AfterUnmarshal() {
	if input.HasObjectStorage != nil && *input.HasObjectStorage && !utils.IsInStringArray(cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE, input.Capability) {
		input.Capability = append(input.Capability, cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE)
	}
}
