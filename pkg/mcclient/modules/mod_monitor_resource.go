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

package modules

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

var (
	MonitorResourceManager      *SMonitorResourceManager
	MonitorResourceAlertManager *SMonitorResourceAlertManager
)

type SMonitorResourceManager struct {
	*modulebase.ResourceManager
}

type SMonitorResourceAlertManager struct {
	*modulebase.JointResourceManager
}

func init() {
	MonitorResourceManager = NewMonitorResourceManager()
	MonitorResourceAlertManager = NewAlertResourceAlertManager()

	register(MonitorResourceManager)
	register(MonitorResourceAlertManager)
}

func NewMonitorResourceManager() *SMonitorResourceManager {
	man := NewMonitorV2Manager("monitorresource", "monitorresources",
		[]string{"id", "name", "res_type", "res_id", "alert_state"},
		[]string{})
	return &SMonitorResourceManager{
		ResourceManager: &man,
	}
}

func NewAlertResourceAlertManager() *SMonitorResourceAlertManager {
	man := NewJointMonitorV2Manager("monitorresourcealert", "monitorresourcealerts",
		[]string{"monitor_resource_id", "alert_id", "res_name", "res_type", "alert_name", "alert_state", "send_state", "level",
			"trigger_time", "data"},
		[]string{},
		MonitorResourceManager, CommonAlertManager)
	return &SMonitorResourceAlertManager{&man}
}
