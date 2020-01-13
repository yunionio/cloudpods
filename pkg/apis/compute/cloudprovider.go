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

type CloudproviderDetails struct {
	Provider         string `json:"provider,omitempty"`
	Brand            string `json:"brand,omitempty"`
	Account          string `json:"account,omitempty"`
	AccountId        string `json:"account_id,omitempty"`
	Manager          string `json:"manager,omitempty"`
	ManagerId        string `json:"manager_id,omitempty"`
	ManagerProject   string `json:"manager_project,omitempty"`
	ManagerProjectId string `json:"manager_project_id,omitempty"`
	ManagerDomain    string `json:"manager_domain,omitempty"`
	ManagerDomainId  string `json:"manager_domain_id,omitempty"`
	Region           string `json:"region,omitempty"`
	RegionId         string `json:"region_id,omitempty"`
	CloudregionId    string `json:"cloudregion_id,omitempty"`
	RegionExternalId string `json:"region_external_id,omitempty"`
	RegionExtId      string `json:"region_ext_id,omitempty"`
	Zone             string `json:"zone,omitempty"`
	ZoneId           string `json:"zone_id,omitempty"`
	ZoneExtId        string `json:"zone_ext_id,omitempty"`
	CloudEnv         string `json:"cloud_env,omitempty"`
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
	// enum: OneCloud,VMware,Aliyun,Qcloud,Azure,Aws,Huawei,OpenStack,Ucloud,ZStack,Google,Ctyun,S3,Ceph,Xsky"
	Providers []string `json:"providers"`
	// swagger:ignore
	// Deprecated
	Provider []string `json:"provider" deprecated-by:"providers"`

	// 列出指定云平台品牌的资源
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
