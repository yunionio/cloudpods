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

const (
	SEND_STATE_OK     = "ok"
	SEND_STATE_SILENT = "silent"
	SEND_STATE_SHIELD = "shield"
)

type AlertRecordListInput struct {
	apis.Meta

	apis.ScopedResourceBaseListInput
	apis.EnabledResourceBaseListInput
	apis.StatusStandaloneResourceListInput

	AlertId  string `json:"alert_id"`
	Level    string `json:"level"`
	State    string `json:"state"`
	ResType  string `json:"res_type"`
	Alerting bool   `json:"alerting"`
	ResName  string `json:"res_name"`
}

type AlertRecordDetails struct {
	SAlertRecord

	apis.StatusStandaloneResourceDetails
	apis.ScopedResourceBaseInfo

	ResNum      int64  `json:"res_num"`
	AlertName   string `json:"alert_name"`
	TriggerTime time.Time
}

func (self AlertRecordDetails) GetMetricTags() map[string]string {
	ret := map[string]string{
		"id":             self.Id,
		"alert_id":       self.AlertId,
		"alert_name":     self.AlertName,
		"domain_id":      self.DomainId,
		"project_domain": self.ProjectDomain,
		"res_type":       self.ResType,
		"tenant":         self.Tenant,
		"tenant_id":      self.TenantId,
	}
	return ret
}

type AlertRecordCreateInput struct {
	apis.StandaloneResourceCreateInput

	AlertId string `json:"alert_id"`
	// 报警级别
	Level     string             `json:"level"`
	State     string             `json:"state"`
	SendState string             `json:"send_state"`
	ResType   string             `json:"res_type"`
	EvalData  []*EvalMatch       `json:"eval_data"`
	AlertRule []*AlertRecordRule `json:"alert_rule"`
}

type AlertRecordRule struct {
	Metric          string `json:"metric"`
	Database        string `json:"database"`
	Measurement     string `json:"measurement"`
	MeasurementDesc string `json:"measurement_desc"`
	ResType         string `json:"res_type"`
	Field           string `json:"field"`
	FieldDesc       string `json:"field_desc"`
	// 比较运算符, 比如: >, <, >=, <=
	Comparator string `json:"comparator"`
	// 报警阀值
	Threshold     string `json:"threshold"`
	Period        string `json:"period"`
	AlertDuration int64  `json:"alert_duration"`
	ConditionType string `json:"condition_type"`
	// 静默期
	SilentPeriod string `json:"silent_period"`
	Reducer      string `json:"reducer"`
}
