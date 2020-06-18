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

package models

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	//存储初始化的内容，同时起到默认配置的作用。
	suggestSysRuleDrivers = make(map[monitor.SuggestDriverType]ISuggestSysRuleDriver, 0)
)

type ISuggestSysRuleDriver interface {
	GetType() monitor.SuggestDriverType
	GetResourceType() monitor.MonitorResourceType
	GetAction() monitor.SuggestDriverAction
	GetSuggest() monitor.MonitorSuggest
	//validate on create
	ValidateSetting(input *monitor.SSuggestSysAlertSetting) error

	//method call for cronjob
	DoSuggestSysRule(ctx context.Context, userCred mcclient.TokenCredential, isStart bool)
	Run(instance *monitor.SSuggestSysAlertSetting)

	//resolve thing for the rule
	StartResolveTask(ctx context.Context, userCred mcclient.TokenCredential, suggestSysAlert *SSuggestSysAlert,
		params *jsonutils.JSONDict) error
	Resolve(data *SSuggestSysAlert) error
}

func RegisterSuggestSysRuleDrivers(drvs ...ISuggestSysRuleDriver) {
	for _, drv := range drvs {
		suggestSysRuleDrivers[drv.GetType()] = drv
	}
}

func GetSuggestSysRuleDrivers() map[monitor.SuggestDriverType]ISuggestSysRuleDriver {
	return suggestSysRuleDrivers
}
