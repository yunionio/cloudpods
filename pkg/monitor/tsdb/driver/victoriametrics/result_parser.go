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

package victoriametrics

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/influxdata/promql/v2/pkg/labels"
	"github.com/zexi/influxql-to-metricsql/converter/translator"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

func newMapId(input map[string]string, ignoreKeys ...string) string {
	keys := make([]string, 0)
	ignoreKS := sets.NewString(ignoreKeys...)
	for key := range input {
		if ignoreKS.Has(key) {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]string, len(keys))
	for i, key := range keys {
		pair := fmt.Sprintf("%s->%s", key, input[key])
		pairs[i] = pair
	}
	return strings.Join(pairs, ",")
}

type points struct {
	id      string
	columns []string
	values  []ResponseDataResultValue
	tags    map[string]string
}

func (p *points) add(op *points) error {
	if len(op.columns) != 1 {
		return errors.Errorf("input points' columns are %#v, which length isn't equal 1", op.columns)
	}
	p.columns = append(p.columns, op.columns[0])
	// merge values
	for i, val := range p.values {
		oVal := op.values[i]
		valTime := val[0]
		oValTime := oVal[0]
		if valTime != oValTime {
			return errors.Errorf("value time %v != other value time %v", valTime, oValTime)
		}
		if len(oVal) != 2 {
			return errors.Errorf("input values' are %#v, which length isn't equal 2", oVal)
		}
		val = append(val, oVal[1])
		p.values[i] = val
	}
	return nil
}

func (p *points) isEqual(op *points) bool {
	if p.id != op.id {
		return false
	}
	return reflect.DeepEqual(p.columns, op.columns) && reflect.DeepEqual(p.tags, op.tags) && reflect.DeepEqual(p.values, op.values)
}

func newPointsByResult(result ResponseDataResult, sameTimes sets.String) (*points, error) {
	tags := result.Metric
	column, ok := tags[translator.UNION_RESULT_NAME]
	if !ok {
		return nil, errors.Errorf("result tags %#v don't contain key %s", tags, translator.UNION_RESULT_NAME)
	}
	for _, ignoreKey := range []string{
		translator.UNION_RESULT_NAME,
		labels.MetricName,
	} {
		delete(tags, ignoreKey)
	}
	values := result.Values
	id := newMapId(tags)
	filterValues := []ResponseDataResultValue{}
	for _, val := range values {
		valTime := fmt.Sprintf("%s", val[0])
		if sameTimes.Has(valTime) {
			tmpVal := val
			filterValues = append(filterValues, tmpVal)
		}
	}
	return &points{
		id:      id,
		columns: []string{column},
		values:  filterValues,
		tags:    tags,
	}, nil
}

func newPointsByResults(results []ResponseDataResult) ([]*points, error) {
	uniq := make(map[string]*points, 0)
	ret := make([]*points, 0)

	var sameTimes sets.String = nil
	for _, result := range results {
		resultTime := sets.NewString()
		for _, v := range result.Values {
			cTime := fmt.Sprintf("%v", v[0])
			resultTime.Insert(cTime)
		}
		if sameTimes == nil {
			sameTimes = resultTime
		} else {
			sameTimes = sameTimes.Intersection(resultTime)
		}
	}

	for _, result := range results {
		p, err := newPointsByResult(result, sameTimes)
		if err != nil {
			return nil, errors.Wrapf(err, "new points by result: %#v", result)
		}
		if ep, ok := uniq[p.id]; ok {
			if err := ep.add(p); err != nil {
				return nil, errors.Wrapf(err, "add point %#v", p)
			}
		} else {
			uniq[p.id] = p
			ret = append(ret, p)
		}
	}
	return ret, nil
}

func transPointsToSeries(points []*points, query *tsdb.Query) monitor.TimeSeriesSlice {
	var result monitor.TimeSeriesSlice
	for _, point := range points {
		result = append(result, transPointToSeries(point, query)...)
	}
	return result
}

func transValuesToTSDBPoints(vals []ResponseDataResultValue) monitor.TimeSeriesPoints {
	var points monitor.TimeSeriesPoints
	for _, val := range vals {
		point, err := parseTimepoint(val)
		if err != nil {
			log.Errorf("parseTimepoint: %#v", val)
		} else {
			points = append(points, point)
		}
	}
	return points
}

func reviseTags(tags map[string]string) map[string]string {
	ret := make(map[string]string)
	for key, val := range tags {
		val_ := strings.ReplaceAll(val, "+", " ")
		ret[key] = val_
	}
	return ret
}

func transPointToSeries(p *points, query *tsdb.Query) monitor.TimeSeriesSlice {
	var result monitor.TimeSeriesSlice

	points := transValuesToTSDBPoints(p.values)
	tags := reviseTags(p.tags)
	metricName := strings.Join(p.columns, ",")
	ts := tsdb.NewTimeSeries(metricName, formatRawName(0, metricName, query, tags, nil), append(p.columns, "time"), points, tags)
	result = append(result, ts)
	return result
}
