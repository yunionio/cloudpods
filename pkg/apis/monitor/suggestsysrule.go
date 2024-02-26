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

var METRIC_ATTRI = []string{METRIC_TAG, METRIC_FIELD}

type InfluxMeasurement struct {
	apis.Meta
	Database               string
	Measurement            string
	MeasurementDisplayName string
	ResType                string
	Score                  int
	TagKey                 []string
	TagValue               map[string][]string
	FieldKey               []string
	FieldDescriptions      map[string]MetricFieldDetail
	Unit                   []string
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
