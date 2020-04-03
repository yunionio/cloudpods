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
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type SuggestRuleListOptions struct {
	options.BaseListOptions
}

type SuggestRuleShowOptions struct {
	ID string `help:"ID or name of the alert" json:"-"`
}

type SuggestSysRuleAlertSettingOptions struct {
	Status string `help:"Status of eip_unused rule"`
}

type SuggestRuleCreateOptions struct {
	SuggestSysRuleAlertSettingOptions
	Name    string `help:"Name of the alert"`
	Type    string `help:"Type of suggest rule" choices:"EIP_UNUSED|"`
	Enabled bool   `help:"Enable rule"`
	Period  string `help:"Period of suggest rule e.g. '5s', '1m'" default:"30s""`
}

func (opt SuggestRuleCreateOptions) Params() (jsonutils.JSONObject, error) {
	input := new(monitor.SuggestSysRuleCreateInput)
	input.Name = opt.Name
	input.Period = opt.Period
	input.Type = strings.ToUpper(opt.Type)
	input.Enabled = &opt.Enabled
	if input.Type == monitor.EIP_UN_USED {
		input.Setting = &monitor.SSuggestSysAlertSetting{
			EIPUnused: &monitor.EIPUnused{
				Status: opt.Status,
			},
		}
	}
	return input.JSON(input), nil
}

type SuggestRuleUpdateOptions struct {
	SuggestSysRuleAlertSettingOptions
	ID      string `help:"ID or name of the alert" json:"-"`
	Period  string `help:"Period of suggest rule e.g. '5s', '1m'" default:"30s""`
	Enabled bool   `help:"Enable rule"`
	Type    string `help:"Type of suggest rule" choices:"EIP_UNUSED|"`
}

func (opt SuggestRuleUpdateOptions) Params() (jsonutils.JSONObject, error) {
	input := new(monitor.SuggestSysRuleUpdateInput)
	input.Type = opt.Type
	input.Period = opt.Period
	input.Enabled = &opt.Enabled
	if input.Type == monitor.EIP_UN_USED {
		input.Setting = &monitor.SSuggestSysAlertSetting{
			EIPUnused: &monitor.EIPUnused{
				Status: opt.Status,
			},
		}
	}
	return input.JSON(input), nil
}

type SuggestRuleDeleteOptions struct {
	ID []string `help:"ID of alert to delete"`
}
