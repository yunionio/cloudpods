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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type MonitorResourceJointListInput struct {
	apis.JointResourceBaseListInput
	apis.VirtualResourceListInput
	MonitorResourceId string  `json:"monitor_resource_id"`
	AlertId           string  `json:"alert_id"`
	JointId           []int64 `json:"joint_id"`
	Alerting          bool    `json:"alerting"`
	AlertState        string  `json:"alert_state"`
	SendState         string  `json:"send_state"`
	ResType           string  `json:"res_type"`
	Metric            string  `json:"metric"`
	ResName           string  `json:"res_name"`
	AlertName         string  `json:"alert_name"`
	Level             string  `json:"level"`
	// 查询所有状态
	AllState bool `json:"all_state"`
}

type MonitorResourceJointCreateInput struct {
	apis.Meta
	MonitorResourceId string `json:"monitor_resource_id"`
	AlertId           string `json:"alert_id"`
	Metric            string `json:"metric"`

	AlertRecordId string    `width:"36" charset:"ascii" list:"user"  update:"user"`
	AlertState    string    `width:"18" charset:"ascii" list:"user"  update:"user"`
	TriggerTime   time.Time `list:"user"  update:"user" json:"trigger_time"`
	Data          EvalMatch `json:"data"`
}

type MonitorResourceJointDetails struct {
	ResName     string               `json:"res_name"`
	ResId       string               `json:"res_id"`
	ResType     string               `json:"res_type"`
	AlertName   string               `json:"alert_name"`
	AlertRule   jsonutils.JSONObject `json:"alert_rule"`
	Level       string               `json:"level"`
	SendState   string               `json:"send_state"`
	State       string               `json:"state"`
	IsSetShield bool                 `json:"is_set_shield"`
}
