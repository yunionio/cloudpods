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
	METRIC_TAG   = "TAG"
	METRIC_FIELD = "FIELD"

	METRIC_VM_ID    = "vm_id"
	METRIC_OSS_ID   = "oss_id"
	METRIC_RDS_ID   = "rds_id"
	METRIC_REDIS_ID = "redis_id"
)

var PROPERTY_TYPE = []string{"databases", "measurements", "metric-measurement"}

var FilterSuggestRuleMeasureMentMap = map[SuggestDriverType]string{
	SCALE_DOWN:         "vm",
	REDIS_UNREASONABLE: "dcs",
	RDS_UNREASONABLE:   "rds",
	OSS_UNREASONABLE:   "oss",
}

var METRIC_ATTRI = []string{METRIC_TAG, METRIC_FIELD}

type InfluxMeasurement struct {
	apis.Meta
	Database               string
	Measurement            string
	MeasurementDisplayName string
	ResType                string
	TagKey                 []string
	TagValue               map[string][]string
	FieldKey               []string
	FieldDescriptions      map[string]MetricFieldDetail
	Unit                   []string
}

type SuggestSysRuleListInput struct {
	apis.StandaloneResourceListInput
	apis.EnabledResourceBaseListInput
}

type SuggestSysRuleCreateInput struct {
	apis.StandaloneResourceCreateInput

	// 查询指标周期
	Period         string                   `json:"period"`
	TimeFrom       string                   `json:"time_from"`
	Type           string                   `json:"type"`
	Enabled        *bool                    `json:"enabled"`
	Setting        *SSuggestSysAlertSetting `json:"setting"`
	IgnoreTimeFrom *bool                    `json:"ignore_time_from"`
}

type SuggestSysRuleUpdateInput struct {
	apis.Meta

	// 查询指标周期
	Period       string                   `json:"period"`
	Name         string                   `json:"name"`
	Type         string                   `json:"type"`
	Setting      *SSuggestSysAlertSetting `json:"setting"`
	Enabled      *bool                    `json:"enabled"`
	ExecTime     time.Time                `json:"exec_time"`
	IgnorePeriod *bool                    `json:"ignore_period"`
}

type SuggestSysRuleDetails struct {
	apis.StandaloneResourceDetails
	CommonAlertMetricDetails []*CommonAlertMetricDetails `json:"common_alert_metric_details"`

	ID      string                   `json:"id"`
	Name    string                   `json:"name"`
	Setting *SSuggestSysAlertSetting `json:"setting"`
	Enabled bool                     `json:"enabled"`
}

type SSuggestSysAlertSetting struct {
	EIPUnused  *EIPUnused  `json:"eip_unused"`
	DiskUnused *DiskUnused `json:"disk_unused"`
	LBUnused   *LBUnused   `json:"lb_unused"`
	ScaleRule  *ScaleRule  `json:"scale_rule"`
}

type EIPUnused struct {
}

type DiskUnused struct {
}

type LBUnused struct {
}

type ScaleRule []Scale

type Scale struct {
	Database    string `json:"database"`
	Measurement string `json:"measurement"`
	//rule operator rule [and|or]
	Operator  string  `json:"operator"`
	Field     string  `json:"field"`
	EvalType  string  `json:"eval_type"`
	Threshold float64 `json:"threshold"`
	Tag       string  `json:"tag"`
	TagVal    string  `json:"tag_val"`
}

type ScaleEvalMatch struct {
	EvalMatch
	ResourceId map[string]string `json:"resource_id"`
}
