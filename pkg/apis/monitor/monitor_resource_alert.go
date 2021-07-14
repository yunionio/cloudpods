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

	"yunion.io/x/onecloud/pkg/apis"
)

type MonitorResourceJointListInput struct {
	MonitorResourceId string  `json:"monitor_resource_id"`
	AlertId           string  `json:"alert_id"`
	JointId           []int64 `json:"joint_id"`
}

type MonitorResourceJointCreateInput struct {
	apis.Meta
	MonitorResourceId string `json:"monitor_resource_id"`
	AlertId           string `json:"alert_id"`

	AlertRecordId string    `width:"36" charset:"ascii" list:"user"  update:"user"`
	AlertState    string    `width:"18" charset:"ascii" list:"user"  update:"user"`
	TriggerTime   time.Time `list:"user"  update:"user" json:"trigger_time"`
	Data          EvalMatch `json:"data"`
}
