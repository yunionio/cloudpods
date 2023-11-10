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

type DataSourceConfig struct {
	Id     string
	Name   string
	Driver string
	Config interface{}
}

type MetricResource struct {
	// Type is the metric resource type. e.g: host, vm, lbinstance
	Type string `json:"type"`
	// ConfigId is the data source config id
	ConfigId string `json:"config_id"`
}

type Metric struct {
	Resource    MetricResource `json:"resource"`
	Measurement string         `json:"measurement"`
	Field       string         `json:"field"`
	DisplayName string         `json:"displayname"`
}

type TimeSeries struct {
	Results []TimeSeriesResult `json:"results"`
}

type TimeSeriesResult struct {
	Series []TimeSeriesRow `json:"series"`
}

type TimeSeriesRow struct {
	Metric  Metric            `json:"metric"`
	Tags    map[string]string `json:"tags,omitempty"`
	Columns []string          `json:"columns,omitempty"`
	// Value item is a point with timestamp and value
	Values [][]interface{} `json:"values,omitempty"`
}

type MetricRequest struct {
	// The start time for the query
	From string `json:"from"`
	// An end time for the query
	To      string         `json:"to"`
	Queries []*MetricQuery `json:"queries"`
	Debug   bool           `json:"debug"`
}

type MetricQueryTag struct {
	Key       string `json:"key"`
	Operator  string `json:"operator"`
	Value     string `json:"value"`
	Condition string `json:"condition"`
}

type MetricQueryPart struct {
	Type   string   `json:"type"`
	Params []string `json:"params"`
}

type MetricQuerySelect []MetricQueryPart

func NewMetricQuerySelect(parts ...MetricQueryPart) MetricQuerySelect {
	return parts
}

type MetricQuery struct {
	Alias        string              `json:"alias"`
	Tz           string              `json:"tz"`
	Database     string              `json:"database"`
	Measurement  string              `json:"measurement"`
	Tags         []MetricQueryTag    `json:"tags"`
	GroupBy      []MetricQueryPart   `json:"group_by"`
	Selects      []MetricQuerySelect `json:"select"`
	Interval     string              `json:"interval"`
	Policy       string              `json:"policy"`
	ResultFormat string              `json:"result_format"`
}

type AlertConditionCombiner string

type Condition struct {
	Type      string    `json:"type"`
	Params    []float64 `json:"params"`
	Operators []string  `json:"operators"`
}
