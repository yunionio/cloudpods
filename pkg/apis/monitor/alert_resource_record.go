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

const (
	ALERT_RESOURCE_RECORD_SHIELD_KEY   = "send_state"
	ALERT_RESOURCE_RECORD_SHIELD_VALUE = "hide"
)

type AlertResourceRecordCreateInput struct {
	apis.Meta
	apis.SStandaloneResourceBase

	EvalData      EvalMatch `json:"eval_data"`
	AlertTime     time.Time `json:"alert_time"`
	ResName       string    `json:"res_name"`
	ResType       string    `json:"res_type"`
	Brand         string    `json:"brand"`
	TriggerVal    string    `json:"trigger_val"`
	AlertRecordId string    `json:"alert_record_id"`
	AlertId       string    `json:"alert_id"`
	SendState     string    `json:"send_state"`
}

type AlertResourceRecordDetails struct {
	apis.StatusStandaloneResourceDetails
	apis.ScopedResourceBaseInfo

	AlertName string               `json:"alert_name"`
	AlertRule jsonutils.JSONObject `json:"alert_rule"`
	ResType   string               `json:"res_type"`
	Level     string               `json:"level"`
	State     string               `json:"state"`
}
