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
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	proxyapi "yunion.io/x/onecloud/pkg/apis/cloudcommon/proxy"
)

type ManagedResourceInfo struct {
	CloudaccountResourceInfo

	// 云账号Id
	// example: 4d3c8979-9dd0-439b-8d78-36fe1ab1666c
	AccountId string `json:"account_id,omitempty"`

	// 云订阅名称
	// example: google-account
	Manager string `json:"manager,omitempty"`

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
}

type SCloudproviderUsage struct {
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
	// 数据库实例数量
	// example: 2
	DBInstanceCount int `json:"dbinstance_count"`
	// 弹性缓存实例数量
	// example: 2
	ElasticcacheCount int `json:"elasticcache_count"`
	// 项目数量
	ProjectCount int `json:"project_count"`
	// 同步区域数量
	SyncRegionCount int `json:"sync_region_count"`
}

func (usage *SCloudproviderUsage) IsEmpty() bool {
	if usage.HostCount > 0 {
		return false
	}
	if usage.VpcCount > 0 {
		return false
	}
	if usage.StorageCount > 0 {
		return false
	}
	if usage.StorageCacheCount > 0 {
		return false
	}
	if usage.EipCount > 0 {
		return false
	}
	if usage.SnapshotCount > 0 {
		return false
	}
	if usage.LoadbalancerCount > 0 {
		return false
	}
	/*if usage.ProjectCount > 0 {
		return false
	}
	if usage.SyncRegionCount > 0 {
		return false
	}*/
	return true
}

type CloudproviderDetails struct {
	apis.EnabledStatusStandaloneResourceDetails
	apis.ProjectizedResourceInfo

	ProxySetting proxyapi.SProxySetting `json:"proxy_setting"`

	SCloudprovider

	// 云账号名称
	// example: google-account
	Cloudaccount string `json:"cloudaccount"`
	// 子订阅同步状态
	SyncStatus2 string `json:"sync_status2"`
	// 支持服务列表
	Capabilities []string `json:"capabilities"`

	SCloudproviderUsage

	// 子订阅品牌信息
	Brand string `json:"brand"`

	ReadOnly bool `json:"read_only"`

	ProjectMappingResourceInfo

	// 上次同步耗时
	LastSyncCost string
}

// 云订阅输入参数
type CloudproviderResourceInput struct {
	// 列出关联指定云订阅(ID或Name)的资源
	CloudproviderId string `json:"cloudprovider_id"`
	// List objects belonging to the cloud provider
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Manager string `json:"manager" yunion-deprecated-by:"cloudprovider_id"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	ManagerId string `json:"manager_id" yunion-deprecated-by:"cloudprovider_id"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Cloudprovider string `json:"cloudprovider" yunion-deprecated-by:"cloudprovider_id"`
}

type CloudproviderResourceListInput struct {
	// 列出关联指定云订阅(ID或Name)的资源
	CloudproviderId []string `json:"cloudprovider_id"`
	// List objects belonging to the cloud provider
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Manager string `json:"manager" yunion-deprecated-by:"cloudprovider_id"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	ManagerId string `json:"manager_id" yunion-deprecated-by:"cloudprovider_id"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Cloudprovider string `json:"cloudprovider" yunion-deprecated-by:"cloudprovider_id"`
}

type ManagedResourceListInput struct {
	apis.DomainizedResourceListInput
	CloudenvResourceListInput

	CloudproviderResourceListInput

	// 列出关联指定云账号(ID或Name)的资源
	CloudaccountId []string `json:"cloudaccount_id"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Cloudaccount []string `json:"cloudaccount" yunion-deprecated-by:"cloudaccount_id"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Account []string `json:"account" yunion-deprecated-by:"cloudaccount_id"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	AccountId []string `json:"account_id" yunion-deprecated-by:"cloudaccount_id"`

	// 过滤资源，是否为非OneCloud内置私有云管理的资源
	// default: false
	IsManaged *bool `json:"is_managed"`

	// 以云账号名称排序
	// pattern:asc|desc
	OrderByAccount string `json:"order_by_account"`

	// 以云订阅名称排序
	// pattern:asc|desc
	OrderByManager string `json:"order_by_manager"`
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
	apis.ProjectizedResourceListInput

	apis.ExternalizedResourceBaseListInput

	UsableResourceListInput

	CloudregionResourceInput

	ZoneResourceInput

	CapabilityListInput

	SyncableBaseResourceListInput

	ReadOnly *bool `json:"read_only"`

	// 账号健康状态
	HealthStatus []string `json:"health_status"`

	// filter by host schedtag
	HostSchedtagId string `json:"host_schedtag_id"`
}

func (input *CapabilityListInput) AfterUnmarshal() {
	if input.HasObjectStorage != nil && *input.HasObjectStorage && !utils.IsInStringArray(cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE, input.Capability) {
		input.Capability = append(input.Capability, cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE)
	}
}

type SyncableBaseResourceListInput struct {
	// 同步状态
	SyncStatus []string `json:"sync_status"`
}

type CloudproviderUpdateInput struct {
	apis.EnabledStatusStandaloneResourceBaseUpdateInput
}

type CloudproviderCreateInput struct {
}

type CloudproviderGetStorageClassInput struct {
	CloudregionResourceInput
}

type CloudproviderGetStorageClassOutput struct {
	// 对象存储存储类型
	StorageClasses []string `json:"storage_classes"`
}

type CloudproviderGetCannedAclInput struct {
	CloudregionResourceInput
}

type CloudproviderGetCannedAclOutput struct {
	// Bucket支持的预置ACL列表
	BucketCannedAcls []string `json:"bucket_canned_acls"`
	// Object支持的预置ACL列表
	ObjectCannedAcls []string `json:"object_canned_acls"`
}

type CloudproviderSync struct {
	// 指定区域启用或禁用同步
	// default: false
	Enabled bool `json:"enabled"`
	// 指定区域信息
	CloudregionIds []string `json:"cloudregion_ids"`
}
