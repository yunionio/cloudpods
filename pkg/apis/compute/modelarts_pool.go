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

const (
	MODELARTS_POOL_STATUS_RUNNING              = "running"
	MODELARTS_POOL_STATUS_ABNORMAL             = "abnormal"
	MODELARTS_POOL_STATUS_CREATING             = "creating"
	MODELARTS_POOL_STATUS_CREATE_FAILED        = "create_failed"
	MODELARTS_POOL_STATUS_DELETING             = "deleting"
	MODELARTS_POOL_STATUS_DELETE_FAILED        = "delete_failed"
	MODELARTS_POOL_STATUS_CHANGE_CONFIG        = "change_config"
	MODELARTS_POOL_STATUS_CHANGE_CONFIG_FAILED = "change_config_failed"
	MODELARTS_POOL_STATUS_ERROR                = "error"
	MODELARTS_POOL_STATUS_UNKNOWN              = "unknown"
	MODELARTS_POOL_STATUS_TIMEOUT              = "timeout"
)

type ModelartsPoolCreateInput struct {
	apis.VirtualResourceCreateInput
	DeletePreventableCreateInput

	CloudregionResourceInput
	CloudproviderResourceInput

	NodeCount int    `json:"node_count"`
	Cidr      string `json:"cidr"`
}

type ModelartsPoolUpdateInput struct {
	apis.StatusInfrasResourceBaseCreateInput
	CloudproviderResourceInput
	WorkType string `json:"work_type"`
}

// 资源返回详情
type ModelartsPoolDetails struct {
	apis.SVirtualResourceBase
	apis.VirtualResourceDetails

	apis.SExternalizedResourceBase
	SBillingResourceBase
	ManagedResourceInfo
	CloudregionResourceInfo
}

func (self ModelartsPoolDetails) GetMetricTags() map[string]string {
	ret := map[string]string{
		"id":                  self.Id,
		"modelarts_pool_id":   self.Id,
		"modelarts_pool_name": self.Name,
		"status":              self.Status,
		"tenant_id":           self.ProjectId,
		"brand":               self.Brand,
		"domain_id":           self.DomainId,
		"account_id":          self.AccountId,
		"account":             self.Account,
		"external_id":         self.ExternalId,
	}

	return AppendMetricTags(ret, self.MetadataResourceInfo, self.ProjectizedResourceInfo)
}

type ModelartsPoolListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	ManagedResourceListInput
	RegionalFilterListInput
	apis.DeletePreventableResourceBaseListInput
}

type ModelartsPoolSyncstatusInput struct {
}

type ModelartsPoolChangeConfigInput struct {
	NodeCount int
}
