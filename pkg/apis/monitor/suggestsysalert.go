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

package monitor

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/compute"
)

type SuggestSysAlertListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput
	compute.ManagedResourceListInput

	//监控规则type：Rule Type
	Type  string `json:"type"`
	ResId string `json:"res_id"`
}

type SuggestSysAlertCreateInput struct {
	apis.VirtualResourceCreateInput

	Enabled       *bool                    `json:"enabled"`
	MonitorConfig *SSuggestSysAlertSetting `json:"monitor_config"`

	//转换成ResId
	ResID string `json:"res_id"`
	Type  string `json:"type"`
	//Problem jsonutils.JSONObject `json:"problem"`
	Suggest string `json:"suggest"`
	Action  string `json:"action"`

	RuleAt time.Time `json:"rule_at"`
}

type SuggestSysAlertDetails struct {
	apis.VirtualResourceDetails
	compute.CloudregionResourceInfo
	RuleName string `json:"rule_name"`
	ShowName string `json:"show_name"`
	ResType  string `json:"res_type"`
	Suggest  string `json:"suggest"`
	Brand    string `json:"brand"`
	Account  string `json:"account"`
}

type SuggestSysAlertUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput

	Enabled       *bool                    `json:"enabled"`
	MonitorConfig *SSuggestSysAlertSetting `json:"monitor_config"`

	//转换成ResId
	ResID string `json:"res_id"`
	Type  string `json:"type"`
	//Problem jsonutils.JSONObject `json:"problem"`
	Suggest string `json:"suggest"`
	Action  string `json:"action"`

	RuleAt time.Time `json:"rule_at"`
}

type SuggestAlertIngoreInput struct {
	apis.ScopedResourceCreateInput
}
