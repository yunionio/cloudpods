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
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	SuggestSysRuleManager       *SSuggestSysRuleManager
	SuggestSysAlertManager      *SSuggestSysAlertManager
	SuggestSysRuleConfigManager *SSuggestSysRuleConfigManager
	InfluxdbShemaManager        *SInfluxdbShemaManager
	SuggestSysAlertCostManager  *SSuggestSysAlertManager
)

func init() {
	SuggestSysRuleManager = NewSuggestSysRuleManager()
	SuggestSysAlertManager = NewSuggestSysAlertManager()
	SuggestSysRuleConfigManager = NewSuggestSysRuleConfigManager()
	InfluxdbShemaManager = NewInfluxdbShemaManager()
	SuggestSysAlertCostManager = NewSuggestSysAlertCostManager()
	for _, m := range []modulebase.IBaseManager{
		SuggestSysRuleManager,
		SuggestSysAlertManager,
		SuggestSysRuleConfigManager,
		InfluxdbShemaManager,
	} {
		modules.Register(m)
	}
}

type SSuggestSysRuleManager struct {
	*modulebase.ResourceManager
}

type SSuggestSysAlertManager struct {
	*modulebase.ResourceManager
}

type SSuggestSysRuleConfigManager struct {
	*modulebase.ResourceManager
}

type SInfluxdbShemaManager struct {
	*modulebase.ResourceManager
}

func NewSuggestSysRuleManager() *SSuggestSysRuleManager {
	man := modules.NewSuggestionManager("suggestsysrule", "suggestsysrules",
		[]string{"id", "name", "type", "enabled", "setting"},
		[]string{})
	return &SSuggestSysRuleManager{
		ResourceManager: &man,
	}
}

func NewSuggestSysAlertManager() *SSuggestSysAlertManager {
	man := modules.NewSuggestionManager("suggestsysalert", "suggestsysalerts",
		[]string{"id", "name", "type", "res_id", "monitor_config"},
		[]string{})
	return &SSuggestSysAlertManager{
		ResourceManager: &man,
	}
}

func NewSuggestSysAlertCostManager() *SSuggestSysAlertManager {
	man := modules.NewSuggestionManager("suggestsysalert", "suggestsysalerts",
		[]string{},
		[]string{})
	return &SSuggestSysAlertManager{
		ResourceManager: &man,
	}
}

func NewSuggestSysRuleConfigManager() *SSuggestSysRuleConfigManager {
	man := modules.NewSuggestionManager("suggestsysruleconfig", "suggestsysruleconfigs",
		[]string{"id", "name", "type", "resource_type", "enabled", "ignore_alert"},
		[]string{"scope", "project_domain", "project"})
	return &SSuggestSysRuleConfigManager{
		ResourceManager: &man,
	}
}

func NewInfluxdbShemaManager() *SInfluxdbShemaManager {
	man := modules.NewSuggestionManager("influxdbshema", "influxdbshemas",
		[]string{},
		[]string{})
	return &SInfluxdbShemaManager{
		ResourceManager: &man,
	}
}
