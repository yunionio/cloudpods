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

type QueryResult struct {
	Error       error               `json:"-"`
	ErrorString string              `json:"error,omitempty"`
	RefId       string              `json:"ref_id"`
	Meta        api.QueryResultMeta `json:"meta"`
	Series      api.TimeSeriesSlice `json:"series"`
	Tables      []*Table            `json:"tables"`
	Dataframes  [][]byte            `json:"dataframes"`
}

func NewTimeSeries(
	name string,
	rawName string,
	columns []string,
	points api.TimeSeriesPoints,
	tags map[string]string,
) *api.TimeSeries {
	return &api.TimeSeries{
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

func NewQueryResult() *QueryResult {
	return &QueryResult{
		Series: make(api.TimeSeriesSlice, 0),
	}
}

func FormatRawName(idx int, name string, groupByTags []string, tags map[string]string, diffTagKeys sets.String) string {
	// when group by tag specified
	/*if len(groupByTags) != 0 {
		for key, val := range tags {
			if strings.Contains(strings.Join(groupByTags, ","), key) {
				return val
			}
		}
	}*/

	genHint := func(k, v string) string {
		return fmt.Sprintf("%s=%s", k, v)
	}

	hintNames := sets.NewString()
	hints := sets.NewString()
	tagKeys := sets.NewString()
	for _, tagKey := range api.MEASUREMENT_TAG_KEYWORD {
		tagKeys.Insert(tagKey)
	}
	tagKeys = tagKeys.Union(diffTagKeys)
	for _, tagKey := range tagKeys.List() {
		if tagV, ok := tags[tagKey]; ok {
			gHint := genHint(tagKey, tagV)
			if strings.Contains(tagKey, "name") && !sets.NewString(groupByTags...).Has(tagKey) {
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
