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

type SAlertRecordManager struct {
	*modulebase.ResourceManager
}

var (
	AlertRecordManager       *SAlertRecordManager
	AlertRecordShieldManager *SAlertRecordManager
)

func init() {
	AlertRecordManager = NewAlertRecordManager()
	AlertRecordShieldManager = NewAlertRecordShieldManager()
	register(AlertRecordManager)
	register(AlertRecordShieldManager)
}

func NewAlertRecordManager() *SAlertRecordManager {
	man := NewMonitorV2Manager("alertrecord", "alertrecords",
		[]string{"id", "alert_name", "res_type", "level", "state", "res_num", "eval_data"},
		[]string{})
	return &SAlertRecordManager{
		ResourceManager: &man,
	}
}

func NewAlertRecordShieldManager() *SAlertRecordManager {
	man := NewMonitorV2Manager("alertrecordshield", "alertrecordshields",
		[]string{"id", "res_name", "alert_name", "res_type", "alert_id", "start_time", "end_time"},
		[]string{})
	return &SAlertRecordManager{
		ResourceManager: &man,
	}
}
