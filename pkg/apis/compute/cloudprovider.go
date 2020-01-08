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
	// List objects belonging to the cloud provider
	Cloudprovider string `json:"cloudprovider"`
	// List objects belonging to the cloud provider
	// swagger: ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Manager string `json:"manager"`
	// swagger: ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	ManagerId string `json:"manager_id"`
	// swagger: ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	CloudproviderId string `json:"cloudprovider_id"`

	// List objects belonging to the cloud account
	Cloudaccount string `json:"cloudaccount"`
	// swagger: ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	CloudaccountId string `json:"cloudaccount_id"`
	// swagger: ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Account string `json:"account"`
	// swagger: ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	AccountId string `json:"account_id"`

	// List objects from the providers, choices:"OneCloud|VMware|Aliyun|Qcloud|Azure|Aws|Huawei|OpenStack|Ucloud|ZStack|Google"
	Providers []string `json:"provider"`

	// List objects belonging to brands
	Brands []string `json:"brand"`

	// enum: public,private,onpremise
	CloudEnv string `json:"cloud_env"`

	// swagger: ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	PublicCloud bool `json:"public_cloud"`
	// swagger: ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	IsPublic bool `json:"is_public"`

	// swagger: ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	PrivateCloud bool `json:"private_cloud"`
	// swagger: ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	IsPrivate bool `json:"is_private"`

	// swagger: ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	IsOnPremise bool `json:"is_on_premise"`

	// List objects managed by external providers
	// default: false
	IsManaged bool `json:"is_managed"`
}

func (input ManagedResourceListInput) CloudaccountStr() string {
	if len(input.Cloudaccount) > 0 {
		return input.Cloudaccount
	}
	if len(input.CloudaccountId) > 0 {
		return input.CloudaccountId
	}
	if len(input.Account) > 0 {
		return input.Account
	}
	if len(input.AccountId) > 0 {
		return input.AccountId
	}
	return ""
}

func (input ManagedResourceListInput) CloudproviderStr() string {
	if len(input.Cloudprovider) > 0 {
		return input.Cloudprovider
	}
	if len(input.CloudproviderId) > 0 {
		return input.CloudproviderId
	}
	if len(input.Manager) > 0 {
		return input.Manager
	}
	if len(input.ManagerId) > 0 {
		return input.ManagerId
	}
	return ""
}

func (input ManagedResourceListInput) CloudEnvStr() string {
	if len(input.CloudEnv) > 0 {
		return input.CloudEnv
	}
	if input.PublicCloud || input.IsPublic {
		return CLOUD_ENV_PUBLIC_CLOUD
	}
	if input.PrivateCloud || input.IsPrivate {
		return CLOUD_ENV_PRIVATE_CLOUD
	}
	if input.IsOnPremise {
		return CLOUD_ENV_ON_PREMISE
	}
	return ""
}

type CapabilityListInput struct {
	// filter by cloudprovider capability
	Capability []string `json:"capability"`
	// swagger: ignore
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

func (input CapabilityListInput) CapabilityList() []string {
	if input.HasObjectStorage != nil && *input.HasObjectStorage && !utils.IsInStringArray(cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE, input.Capability) {
		input.Capability = append(input.Capability, cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE)
	}
	return input.Capability
}
