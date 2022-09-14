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
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	CommonAlerts *SCommonAlertManager
)

type SCommonAlertManager struct {
	*modulebase.ResourceManager
}

func init() {
	CommonAlerts = NewCommonAlertManager()
	for _, m := range []modulebase.IBaseManager{
		CommonAlerts,
	} {
		modules.Register(m)
	}
}

func NewCommonAlertManager() *SCommonAlertManager {
	man := modules.NewMonitorV2Manager("commonalert", "commonalerts",
		[]string{"id", "name", "enabled", "level", "alert_type", "period", "recipients", "channel"},
		[]string{})
	return &SCommonAlertManager{
		ResourceManager: &man,
	}
}

func (m *SCommonAlertManager) DoCreate(s *mcclient.ClientSession, config *AlertConfig, bi *monitor.CommonAlertCreateBaseInput) (jsonutils.JSONObject, error) {
	input := config.ToCommonAlertCreateInput(bi)
	log.Errorf("======create json: %s", input.JSON(input).PrettyString())
	return m.Create(s, input.JSON(input))
}
