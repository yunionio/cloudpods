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

import "fmt"

type Condition struct {
	Type      string    `json:"type"`
	Params    []float64 `json:"params"`
	Operators []string  `json:"operators"`
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

func NewMetricQueryPartField(fieldName string) MetricQueryPart {
	return MetricQueryPart{
		Type:   "field",
		Params: []string{fieldName},
	}
}

func NewMetricQueryPartAS(alias string) MetricQueryPart {
	return MetricQueryPart{
		Type:   "alias",
		Params: []string{alias},
	}
}

func NewMetricQueryPartMath(op string, val string) MetricQueryPart {
	return MetricQueryPart{
		Type:   "math",
		Params: []string{fmt.Sprintf("%s %s", op, val)},
	}
}

func NewMetricQueryPartFunc(funcName string) MetricQueryPart {
	return MetricQueryPart{
		Type: funcName,
	}
}

func NewMetricQueryPartMean() MetricQueryPart {
	return NewMetricQueryPartFunc("mean")
}

func NewMetricQueryPartCount() MetricQueryPart {
	return NewMetricQueryPartFunc("count")
}

func NewMetricQueryPartDistinct() MetricQueryPart {
	return NewMetricQueryPartFunc("distinct")
}

func NewMetricQueryPartSum() MetricQueryPart {
	return NewMetricQueryPartFunc("sum")
}

func NewMetricQueryPartMin() MetricQueryPart {
	return NewMetricQueryPartFunc("min")
}

func NewMetricQueryPartMax() MetricQueryPart {
	return NewMetricQueryPartFunc("max")
}

func NewMetricQueryPartLast() MetricQueryPart {
	return NewMetricQueryPartFunc("last")
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
