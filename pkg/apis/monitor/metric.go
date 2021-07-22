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

import "yunion.io/x/onecloud/pkg/apis"

const (
	METRIC_RES_TYPE_GUEST        = "guest"
	METRIC_RES_TYPE_HOST         = "host"
	METRIC_RES_TYPE_REDIS        = "redis"
	METRIC_RES_TYPE_OSS          = "oss"
	METRIC_RES_TYPE_RDS          = "rds"
	METRIC_RES_TYPE_CLOUDACCOUNT = "cloudaccount"
	METRIC_RES_TYPE_AGENT        = "agent"
	METRIC_RES_TYPE_TENANT       = "tenant"
	METRIC_RES_TYPE_DOMAIN       = "domain"
	METRIC_RES_TYPE_STORAGE      = "storage"
	METRIC_RES_TYPE_JENKINS      = "jenkins"

	METRIC_UNIT_PERCENT = "%"
	METRIC_UNIT_BPS     = "bps"
	METRIC_UNIT_MBPS    = "Mbps"
	METRIC_UNIT_BYTEPS  = "Bps"
	METRIC_UNIT_CPS     = "cps"
	METRIC_UNIT_COUNT   = "count"
	METRIC_UNIT_MS      = "ms"
	METRIC_UNIT_BYTE    = "byte"
	METRIC_UNIT_RMB     = "RMB"
	METRIC_UNIT_MB      = "Mb"

	METRIC_DATABASE_TELE  = "telegraf"
	METRIC_DATABASE_METER = "meter_db"
)

var (
	MetricResType = []string{METRIC_RES_TYPE_GUEST, METRIC_RES_TYPE_HOST, METRIC_RES_TYPE_REDIS, METRIC_RES_TYPE_OSS,
		METRIC_RES_TYPE_RDS, METRIC_RES_TYPE_CLOUDACCOUNT}
	MetricUnit = []string{METRIC_UNIT_PERCENT, METRIC_UNIT_BPS, METRIC_UNIT_MBPS, METRIC_UNIT_BYTEPS, "count/s",
		METRIC_UNIT_COUNT, METRIC_UNIT_MS, METRIC_UNIT_BYTE, METRIC_UNIT_RMB}
	ResTypeScoreMap = map[string]float64{
		METRIC_RES_TYPE_GUEST:        1,
		METRIC_RES_TYPE_AGENT:        1.1,
		METRIC_RES_TYPE_HOST:         2,
		METRIC_RES_TYPE_OSS:          3,
		METRIC_RES_TYPE_RDS:          4,
		METRIC_RES_TYPE_REDIS:        5,
		METRIC_RES_TYPE_CLOUDACCOUNT: 6,
		METRIC_RES_TYPE_STORAGE:      7,
	}
)

type MetricMeasurementCreateInput struct {
	apis.StandaloneResourceCreateInput
	apis.EnabledBaseResourceCreateInput

	ResType     string `json:"res_type"`
	DisplayName string `json:"display_name"`
	Database    string `json:"database"`
	Score       int    `json:"score"`
}

type MetricMeasurementUpdateInput struct {
	apis.StandaloneResourceBaseUpdateInput

	ResType     string `json:"res_type"`
	DisplayName string `json:"display_name"`
}

type MetricMeasurementListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.EnabledResourceBaseListInput
	apis.ScopedResourceBaseListInput

	ResType     string `json:"res_type"`
	DisplayName string `json:"display_name"`
}

type MetricFieldCreateInput struct {
	apis.StandaloneResourceCreateInput

	DisplayName string `json:"display_name"`
	Unit        string `json:"unit"`
	ValueType   string `json:"value_type"`
	Score       int    `json:"score"`
}

type MetricFieldUpdateInput struct {
	apis.StandaloneResourceBaseUpdateInput

	Id          string
	DisplayName string `json:"display_name"`
	Unit        string `json:"unit"`
	ValueType   string `json:"value_type"`
	Score       int    `json:"score"`
}

type MetricFieldListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.EnabledResourceBaseListInput
	apis.ScopedResourceBaseListInput

	DisplayName string `json:"display_name"`
	Unit        string `json:"unit"`
	Scope       string `json:"scope"`
}

type MetricCreateInput struct {
	Measurement  MetricMeasurementCreateInput `json:"measurement"`
	MetricFields []MetricFieldCreateInput     `json:"metric_fields"`
	Scope        string                       `json:"scope"`
}

type MetricUpdateInput struct {
	apis.Meta

	Measurement  MetricMeasurementUpdateInput `json:"measurement"`
	MetricFields []MetricFieldUpdateInput     `json:"metric_fields"`
	Scope        string                       `json:"scope"`
}

type MetricListInput struct {
	apis.Meta

	Measurement  MetricMeasurementListInput `json:"measurement"`
	MetricFields MetricFieldListInput       `json:"metric_fields"`
	Scope        string                     `json:"scope"`
}

type MetricDetails struct {
	apis.StatusStandaloneResourceDetails
	apis.ScopedResourceBaseInfo

	MetricFields []MetricFieldDetail `json:"metric_fields"`
}

type MetricFieldDetail struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Unit        string `json:"unit"`
	Id          string `json:"id"`
}
