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

func (o *SuggestRuleListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type SuggestRuleShowOptions struct {
	ID string `help:"ID or name of the alert" json:"-"`
}

func (o *SuggestRuleShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *SuggestRuleShowOptions) GetId() string {
	return o.ID
}

type SuggestRuleConfigOptions struct {
	ID       string `help:"ID or name of the alert" json:"-"`
	Period   string `help:"Period of suggest rule e.g. '5s', '1m'"`
	TimeFrom string `help:"TimeFrom of suggest rule e.g. '24h'"`
}

func (o *SuggestRuleConfigOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *SuggestRuleConfigOptions) GetId() string {
	return o.ID
}

type SuggestSysRuleAlertSettingOptions struct {
	Status string `help:"Status of eip_unused rule"`
}

type SuggestRuleCreateOptions struct {
	SuggestSysRuleAlertSettingOptions
	Name    string `help:"Name of the alert"`
	Type    string `help:"Type of suggest rule" choices:"EIP_UNUSED|DISK_UNUSED|LB_UNUSED|SCALE_DOWN"`
	Enabled bool   `help:"Enable rule"`
	Period  string `help:"Period of suggest rule e.g. '5s', '1m'" default:"30s"`
}

func (opt *SuggestRuleCreateOptions) Params() (jsonutils.JSONObject, error) {
	input := new(monitor.SuggestSysRuleCreateInput)
	input.Name = opt.Name
	input.Period = opt.Period
	input.Type = strings.ToUpper(opt.Type)
	input.Enabled = &opt.Enabled
	input.Setting = newSuggestSysAlertSetting(input.Type)
	return input.JSON(input), nil
}

type SuggestRuleUpdateOptions struct {
	SuggestSysRuleAlertSettingOptions
	ID      string `help:"ID or name of the alert" json:"-"`
	Name    string `help:"Name of the alert"`
	Period  string `help:"Period of suggest rule e.g. '5s', '1m'" default:"30s"`
	Enabled bool   `help:"Enable rule"`
	Type    string `help:"Type of suggest rule" choices:"EIP_UNUSED|"`
}

func (opt SuggestRuleUpdateOptions) Params() (jsonutils.JSONObject, error) {
	input := new(monitor.SuggestSysRuleUpdateInput)
	input.Period = opt.Period
	input.Enabled = &opt.Enabled
	input.Name = opt.Name
	input.Setting = newSuggestSysAlertSetting(input.Type)
	return input.JSON(input), nil
}

func newSuggestSysAlertSetting(tp string) *monitor.SSuggestSysAlertSetting {
	setting := new(monitor.SSuggestSysAlertSetting)
	switch monitor.SuggestDriverType(tp) {
	case monitor.EIP_UNUSED:
		setting = &monitor.SSuggestSysAlertSetting{
			EIPUnused: &monitor.EIPUnused{},
		}
	case monitor.DISK_UNUSED:
		setting = &monitor.SSuggestSysAlertSetting{
			DiskUnused: &monitor.DiskUnused{},
		}
	case monitor.LB_UNUSED:
		setting = &monitor.SSuggestSysAlertSetting{
			LBUnused: &monitor.LBUnused{},
		}
	case monitor.SCALE_DOWN:
		scaleRuel := make(monitor.ScaleRule, 0)
		scale := monitor.Scale{
			Database:    "telegraf",
			Measurement: "vm_cpu",
			Operator:    "and",
			Field:       "usage_active",
			EvalType:    ">=",
			Threshold:   50,
			Tag:         "",
			TagVal:      "",
		}
		scaleRuel = append(scaleRuel, scale)
		scale = monitor.Scale{
			Database:    "telegraf",
			Measurement: "vm_diskio",
			Operator:    "or",
			Field:       "read_bps",
			EvalType:    ">=",
			Threshold:   500,
			Tag:         "",
			TagVal:      "",
		}
		scaleRuel = append(scaleRuel, scale)
		setting = &monitor.SSuggestSysAlertSetting{
			ScaleRule: &scaleRuel,
		}

	}
	return setting
}

type SuggestRuleDeleteOptions struct {
	ID string `help:"ID of alert to delete"`
}

func (o *SuggestRuleDeleteOptions) GetId() string {
	return o.ID
}

func (o *SuggestRuleDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}
