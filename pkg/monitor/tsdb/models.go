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

package tsdb

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"yunion.io/x/pkg/util/sets"

	api "yunion.io/x/onecloud/pkg/apis/monitor"
)

type TsdbQuery struct {
	TimeRange *TimeRange
	Queries   []*Query
	Debug     bool
}

type Query struct {
	RefId string
	api.MetricQuery
	DataSource    DataSource
	MaxDataPoints int64
	IntervalMs    int64
}

type Response struct {
	Results map[string]*QueryResult `json:"results"`
	Message string                  `json:"message,omitempty"`
}

type QueryResultMeta struct {
	RawQuery string `json:"raw_query"`
}

type QueryResult struct {
	Error       error           `json:"-"`
	ErrorString string          `json:"error,omitempty"`
	RefId       string          `json:"ref_id"`
	Meta        QueryResultMeta `json:"meta"`
	Series      TimeSeriesSlice `json:"series"`
	Tables      []*Table        `json:"tables"`
	Dataframes  [][]byte        `json:"dataframes"`
}

type TimeSeries struct {
	// RawName is used to frontend displaying the curve name
	RawName string            `json:"raw_name"`
	Columns []string          `json:"columns"`
	Name    string            `json:"name"`
	Points  TimeSeriesPoints  `json:"points"`
	Tags    map[string]string `json:"tags,omitempty"`
}

func NewTimeSeries(
	name string,
	rawName string,
	columns []string,
	points TimeSeriesPoints,
	tags map[string]string,
) *TimeSeries {
	return &TimeSeries{
		RawName: rawName,
		Columns: columns,
		Name:    name,
		Points:  points,
		Tags:    tags,
	}
}

type Table struct {
	Columns []TableColumn `json:"columns"`
	Rows    []RowValues   `json:"rows"`
}

type TableColumn struct {
	Text string `json:"text"`
}

type RowValues []interface{}
type TimePoint []interface{}
type TimeSeriesPoints []TimePoint
type TimeSeriesSlice []*TimeSeries

func NewQueryResult() *QueryResult {
	return &QueryResult{
		Series: make(TimeSeriesSlice, 0),
	}
}

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

func FormatRawName(idx int, name string, groupByTags []string, tags map[string]string) string {
	// when group by tag specified
	if len(groupByTags) != 0 {
		for key, val := range tags {
			if strings.Contains(strings.Join(groupByTags, ","), key) {
				return val
			}
		}
	}

	genHint := func(k, v string) string {
		return fmt.Sprintf("%s=%s", k, v)
	}

	hintNames := sets.NewString()
	hints := sets.NewString()
	for _, tagKey := range api.MEASUREMENT_TAG_KEYWORD {
		if tagV, ok := tags[tagKey]; ok {
			gHint := genHint(tagKey, tagV)
			if strings.Contains(tagKey, "name") {
				hintNames.Insert(gHint)
			} else {
				hints.Insert(gHint)
			}
		}
	}

	if len(hints) == 0 {
		// try id
		for key, val := range tags {
			if strings.Contains(key, "id") && len(val) != 0 {
				hints.Insert(genHint(key, val))
			}
		}
	}

	if len(hints) == 0 {
		// if hints is empty at last, return index hint
		return fmt.Sprintf("unknown-%d-%s", idx, name)
	}
	sortNL := hintNames.List()
	sort.Strings(sortNL)
	sortL := hints.List()
	sort.Strings(sortL)
	sortNL = append(sortNL, sortL...)
	return fmt.Sprintf("{%s}", strings.Join(sortNL, ","))
}
