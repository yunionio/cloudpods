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

package suggestsysdrivers

import (
	"yunion.io/x/onecloud/pkg/apis/monitor"
)

type baseDriver struct {
	driverType   monitor.SuggestDriverType
	resourceType monitor.MonitorResourceType
	action       monitor.SuggestDriverAction
	suggest      monitor.MonitorSuggest
	defRule      monitor.SuggestSysRuleCreateInput
}

func newBaseDriver(
	drvType monitor.SuggestDriverType,
	resType monitor.MonitorResourceType,
	action monitor.SuggestDriverAction,
	suggest monitor.MonitorSuggest,
	defRule monitor.SuggestSysRuleCreateInput,
) *baseDriver {
	return &baseDriver{
		driverType:   drvType,
		resourceType: resType,
		action:       action,
		suggest:      suggest,
		defRule:      defRule,
	}
}

func (drv baseDriver) GetType() monitor.SuggestDriverType {
	return drv.driverType
}

func (drv baseDriver) GetResourceType() monitor.MonitorResourceType {
	return drv.resourceType
}

func (drv baseDriver) GetAction() monitor.SuggestDriverAction {
	return drv.action
}

func (drv baseDriver) GetSuggest() monitor.MonitorSuggest {
	return drv.suggest
}

func (drv baseDriver) GetDefaultRule() monitor.SuggestSysRuleCreateInput {
	return drv.defRule
}
