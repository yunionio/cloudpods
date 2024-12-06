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
	"strconv"
	"time"
)

var (
	UNIFIED_MONITOR_FIELD_OPT_TYPE   = []string{"Aggregations", "Selectors"}
	UNIFIED_MONITOR_GROUPBY_OPT_TYPE = []string{"time", "tag", "fill"}
	UNIFIED_MONITOR_FIELD_OPT_VALUE  = map[string][]string{
		"Aggregations": {"MEAN", "SUM", "MAX", "MIN"}, // {"COUNT", "DISTINCT", "INTEGRAL", "MEAN", "MEDIAN", "MODE", "STDDEV", "SUM"},
		"Selectors":    {"BOTTOM", "FIRST", "LAST", "MAX", "MIN", "TOP"},
	}
	UNIFIED_MONITOR_GROUPBY_OPT_VALUE = map[string][]string{
		"fill": {"linear", "none", "previous", "0"},
	}

	MEASUREMENT_TAG_KEYWORD = map[string]string{
		METRIC_RES_TYPE_HOST:         "host",
		METRIC_RES_TYPE_GUEST:        "vm_name",
		METRIC_RES_TYPE_REDIS:        "redis_name",
		METRIC_RES_TYPE_RDS:          "rds_name",
		METRIC_RES_TYPE_OSS:          "oss_name",
		METRIC_RES_TYPE_CLOUDACCOUNT: "cloudaccount_name",
		METRIC_RES_TYPE_STORAGE:      "storage_name",
		METRIC_RES_TYPE_AGENT:        "vm_name",
	}
	MEASUREMENT_TAG_ID = map[string]string{
		METRIC_RES_TYPE_HOST:         "host_id",
		METRIC_RES_TYPE_GUEST:        "vm_id",
		METRIC_RES_TYPE_AGENT:        "vm_id",
		METRIC_RES_TYPE_REDIS:        "redis_id",
		METRIC_RES_TYPE_RDS:          "rds_id",
		METRIC_RES_TYPE_OSS:          "oss_id",
		METRIC_RES_TYPE_CLOUDACCOUNT: "cloudaccount_id",
		METRIC_RES_TYPE_TENANT:       "tenant_id",
		METRIC_RES_TYPE_DOMAIN:       "domain_id",
		METRIC_RES_TYPE_STORAGE:      "storage_id",
	}
	AlertReduceFunc = map[string]string{
		"avg":          "average value",
		"sum":          "Summation",
		"min":          "minimum value",
		"max":          "Maximum",
		"count":        "count value",
		"last":         "Latest value",
		"median":       "median",
		"diff":         "The difference between the latest value and the oldest value. The judgment basis value must be legal",
		"percent_diff": "The difference between the new value and the old value,based on the percentage of the old value",
	}
)

func GetMeasurementTagIdKeyByResType(resType string) string {
	return MEASUREMENT_TAG_ID[resType]
}

func GetMeasurementTagIdKeyByResTypeWithDefault(resType string) string {
	tagId := GetMeasurementTagIdKeyByResType(resType)
	if len(tagId) == 0 {
		tagId = "host_id"
	}
	return tagId
}

func GetMeasurementResourceId(tags map[string]string, resType string) string {
	return tags[GetMeasurementTagIdKeyByResType(resType)]
}

func GetResourceIdFromTagWithDefault(tags map[string]string, resType string) string {
	tagId := GetMeasurementTagIdKeyByResTypeWithDefault(resType)
	return tags[tagId]
}

type MetricFunc struct {
	FieldOptType  []string            `json:"field_opt_type"`
	FieldOptValue map[string][]string `json:"field_opt_value"`
	GroupOptType  []string            `json:"group_opt_type"`
	GroupOptValue map[string][]string `json:"group_opt_value"`
}

type SimpleQueryInput struct {
	// 资源Id, 可以不填, 代表查询指定监控的所有监控数据
	Id string `json:"id"`
	// 查询指定数据库
	// default: telegraf
	Database string `json:"database"`
	// 监控指标: https://github.com/codelinz/cloudpods/blob/monitor/pkg/cloudprovider/metrics.go
	MetricName string `json:"metric_name"`
	// 开始时间
	StartTime time.Time `json:"start_time"`
	// 结束时间
	EndTime time.Time `json:"end_time"`
	// 指定标签
	Tags map[string]string `json:"tag_pairs"`
}

type SimpleQueryOutput struct {
	Id    string    `json:"id"`
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

type MetricsQueryResult struct {
	SeriesTotal   int64
	Series        TimeSeriesSlice
	Metas         []QueryResultMeta
	ReducedResult *ReducedResult
}

type TimeSeriesPoints []TimePoint

type TimeSeriesSlice []*TimeSeries

type TimeSeries struct {
	// RawName is used to frontend displaying the curve name
	RawName string            `json:"raw_name"`
	Columns []string          `json:"columns"`
	Name    string            `json:"name"`
	Points  TimeSeriesPoints  `json:"points"`
	Tags    map[string]string `json:"tags,omitempty"`
}

type TimePoint []interface{}

func NewTimePoint(value *float64, timestamp float64) TimePoint {
	return TimePoint{value, timestamp}
}

func NewTimePointByVal(value float64, timestamp float64) TimePoint {
	return NewTimePoint(&value, timestamp)
}

func (p TimePoint) IsValid() bool {
	if val, ok := p[0].(*float64); ok && val != nil {
		return true
	}
	return false
	//return p[0].(*float64) != nil
}

func (p TimePoint) IsValids() bool {
	for i := 0; i < len(p)-1; i++ {
		if p[i] == nil {
			return false
		}
		if p[i].(*float64) == nil {
			return false
		}
	}
	return true
}

func (p TimePoint) Value() float64 {
	return *(p[0].(*float64))
}

func (p TimePoint) Timestamp() float64 {
	return p[len(p)-1].(float64)
}

func (p TimePoint) Values() []float64 {
	values := make([]float64, 0)
	for i := 0; i < len(p)-1; i++ {
		values = append(values, *(p[i].(*float64)))
	}
	return values
}

func (p TimePoint) PointValueStr() []string {
	arrStr := make([]string, 0)
	for i := 0; i < len(p)-1; i++ {
		if p[i] == nil {
			arrStr = append(arrStr, "")
		}
		if fval, ok := p[i].(*float64); ok {
			arrStr = append(arrStr, strconv.FormatFloat((*fval), 'f', -1, 64))
			continue
		}
		if ival, ok := p[i].(*int64); ok {
			arrStr = append(arrStr, strconv.FormatInt((*ival), 64))
			continue
		}
		arrStr = append(arrStr, p[i].(string))
	}
	return arrStr
}

func NewTimeSeriesPointsFromArgs(values ...float64) TimeSeriesPoints {
	points := make(TimeSeriesPoints, 0)

	for i := 0; i < len(values); i += 2 {
		points = append(points, NewTimePoint(&values[i], values[i+1]))
	}

	return points
}

type QueryResultMeta struct {
	RawQuery           string  `json:"raw_query"`
	ResultReducerValue float64 `json:"result_reducer_value"`
}

const ConditionTypeMetricQuery = "metricquery"
