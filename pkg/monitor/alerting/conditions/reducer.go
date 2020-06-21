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

package conditions

import (
	"math"
	"sort"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
	"yunion.io/x/onecloud/pkg/monitor/validators"
)

type Reducer interface {
	Reduce(series *tsdb.TimeSeries) *float64
	GetType() string
}

// queryReducer reduces an timeseries to a float
type queryReducer struct {
	// Type is how the timeseries should be reduced.
	// Ex: avg, sum, max, min, count
	Type string
}

func (s *queryReducer) GetType() string {
	return s.Type
}

func (s *queryReducer) Reduce(series *tsdb.TimeSeries) *float64 {
	if len(series.Points) == 0 {
		return nil
	}

	value := float64(0)
	allNull := true

	switch s.Type {
	case "avg":
		validPointsCount := 0
		for _, point := range series.Points {
			if point.IsValid() {
				value += point.Value()
				validPointsCount++
				allNull = false
			}
		}
		if validPointsCount > 0 {
			value = value / float64(validPointsCount)
		}
	case "sum":
		for _, point := range series.Points {
			if point.IsValid() {
				value += point.Value()
				allNull = false
			}
		}
	case "min":
		value = math.MaxFloat64
		for _, point := range series.Points {
			if point.IsValid() {
				allNull = false
				if value > point.Value() {
					value = point.Value()
				}
			}
		}
	case "max":
		value = -math.MaxFloat64
		for _, point := range series.Points {
			if point.IsValid() {
				allNull = false
				if value < point.Value() {
					value = point.Value()
				}
			}
		}
	case "count":
		value = float64(len(series.Points))
		allNull = false
	case "last":
		points := series.Points
		for i := len(points) - 1; i >= 0; i-- {
			if points[i].IsValid() {
				value = points[i].Value()
				allNull = false
				break
			}
		}
	case "median":
		var values []float64
		for _, v := range series.Points {
			if v.IsValid() {
				allNull = false
				values = append(values, v.Value())
			}
		}
		if len(values) >= 1 {
			sort.Float64s(values)
			length := len(values)
			if length%2 == 1 {
				value = values[(length-1)/2]
			} else {
				value = (values[(length/2)-1] + values[length/2]) / 2
			}
		}
	case "diff":
		allNull, value = calculateDiff(series, allNull, value, diff)
	case "percent_diff":
		allNull, value = calculateDiff(series, allNull, value, percentDiff)
	case "count_non_null":
		for _, v := range series.Points {
			if v.IsValid() {
				value++
			}
		}

		if value > 0 {
			allNull = false
		}
	}

	if allNull {
		return nil
	}

	return &value
}

func newSimpleReducer(t string) *queryReducer {
	return &queryReducer{Type: t}
}

func calculateDiff(series *tsdb.TimeSeries, allNull bool, value float64, fn func(float64, float64) float64) (bool, float64) {
	var (
		points = series.Points
		first  float64
		i      int
	)
	// get the newest point
	for i = len(points) - 1; i >= 0; i-- {
		if points[i].IsValid() {
			allNull = false
			first = points[i].Value()
			break
		}
	}
	if i >= 1 {
		// get the oldest point
		for i := 0; i < len(points); i++ {
			if points[i].IsValid() {
				allNull = false
				val := fn(first, points[i].Value())
				value = math.Abs(val)
				break
			}
		}
	}
	return allNull, value
}

var diff = func(newest, oldest float64) float64 {
	return newest - oldest
}

var percentDiff = func(newest, oldest float64) float64 {
	return (newest - oldest) / oldest * 100
}

func NewAlertReducer(cond *monitor.Condition) (Reducer, error) {
	if len(cond.Operators) == 0 {
		return newSimpleReducer(cond.Type), nil
	}

	if utils.IsInStringArray(cond.Operators[0], validators.CommonAlertReducerFieldOpts) {
		return newMathReducer(cond)
	}

	return nil, errors.Wrapf(errors.Error("reducer operator is ilegal"), "operator: %s", cond.Operators[0])
}
