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
)

type SuggestSysRuleListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput
}

type SuggestSysRuleCreateInput struct {
	apis.VirtualResourceCreateInput

	// 查询指标周期
	Period  string                   `json:"period"`
	Type    string                   `json:"type"`
	Enabled *bool                    `json:"enabled"`
	Setting *SSuggestSysAlertSetting `json:"setting"`
}

type SuggestSysRuleUpdateInput struct {
	apis.Meta

	// 查询指标周期
	Period   string                   `json:"period"`
	Name     string                   `json:"name"`
	Type     string                   `json:"type"`
	Setting  *SSuggestSysAlertSetting `json:"setting"`
	Enabled  *bool                    `json:"enabled"`
	ExecTime time.Time                `json:"exec_time"`
}

type SuggestSysRuleDetails struct {
	apis.VirtualResourceDetails

	ID      string                   `json:"id"`
	Name    string                   `json:"name"`
	Setting *SSuggestSysAlertSetting `json:"setting"`
	Enabled bool                     `json:"enabled"`
}

type SSuggestSysAlertSetting struct {
	EIPUnused  *EIPUnused  `json:"eip_unused"`
	DiskUnused *DiskUnused `json:"disk_unused"`
	LBUnused   *LBUnused   `json:"lb_unused"`
}

type EIPUnused struct {
	//Status string `json:"status"`
}

type DiskUnused struct {
}

type LBUnused struct {
}
