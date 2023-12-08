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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	UnifiedMonitorManager *SUnifiedMonitorManager
)

type SUnifiedMonitorManager struct {
	*modulebase.ResourceManager
}

func init() {
	UnifiedMonitorManager = NewUnifiedMonitorManager()
	for _, m := range []modulebase.IBaseManager{
		UnifiedMonitorManager,
	} {
		modules.Register(m)
	}
}

func NewUnifiedMonitorManager() *SUnifiedMonitorManager {
	man := modules.NewMonitorV2Manager("unifiedmonitor", "unifiedmonitors",
		[]string{},
		[]string{})
	return &SUnifiedMonitorManager{
		ResourceManager: &man,
	}
}

func (m *SUnifiedMonitorManager) PerformQuery(s *mcclient.ClientSession, input *monitor.MetricQueryInput) (jsonutils.JSONObject, error) {
	return m.PerformClassAction(s, "query", jsonutils.Marshal(input))
}
