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
	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	WAF_ACTION_ALLOW      = "Allow"
	WAF_ACTION_BLOCK      = "Block"
	WAF_ACTION_PREVENTION = "Prevention"
	WAF_ACTION_DETECTION  = "Detection"

	WAF_STATUS_AVAILABLE     = compute.WAF_STATUS_AVAILABLE
	WAF_STATUS_DELETING      = compute.WAF_STATUS_DELETING
	WAF_STATUS_DELETE_FAILED = "delete_failed"
	WAF_STATUS_CREATING      = "creating"
	WAF_STATUS_CREATE_FAILED = compute.WAF_STATUS_CREATE_FAILED
	WAF_STATUS_UPDATING      = compute.WAF_STATUS_UPDATING
	WAF_STATUS_UNKNOWN       = "unknown"
)

type WafInstanceCreateInput struct {
	apis.EnabledStatusInfrasResourceBaseCreateInput

	// 阿里云CNAME介入回源地址,支持IP和域名,域名仅支持输入一个
	// 此参数和cloud_resources两者必须指定某一个
	SourceIps cloudprovider.WafSourceIps `json:"source_ips"`

	// 关联云资源列表
	// 阿里云要求输入此参数或source_ips
	CloudResources []cloudprovider.SCloudResource

	CloudregionResourceInput
	CloudproviderResourceInput

	Type cloudprovider.TWafType

	DefaultAction *cloudprovider.DefaultAction
}

type WafInstanceDetails struct {
	apis.EnabledStatusInfrasResourceBaseDetails
	ManagedResourceInfo
	CloudregionResourceInfo
	SWafInstance

	Rules []SWafRule
}

type WafInstanceListInput struct {
	apis.EnabledStatusInfrasResourceBaseListInput
	apis.ExternalizedResourceBaseListInput

	ManagedResourceListInput
	RegionalFilterListInput
}

type WafSyncstatusInput struct {
}

type WafDeleteRuleInput struct {
	WafRuleId string
}
