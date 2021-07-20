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

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

type mathReducer struct {
	// Type is how the timeseries should be reduced.
	// Ex: avg, sum, max, min, count
	Type   string
	Opt    string
	Params []float64
}

func (s *mathReducer) GetParams() []float64 {
	return s.Params
}

func (s *mathReducer) GetType() string {
	return s.Type
}

func (s *mathReducer) Reduce(series *tsdb.TimeSeries) (*float64, []string) {
	if len(series.Points) == 0 {
		return nil, nil
	}

	value := float64(0)
	allNull := true
	valArr := make([]string, 0)
	switch s.Type {
	case "avg":
		validPointsCount := 0
		for _, point := range series.Points {
			if point.IsValids() {
				values := point.Values()
				tem, err := s.mathValue(values)
				if err != nil {
					return nil, valArr
				}
				value += tem
				validPointsCount++
				allNull = false
			}
		}
		if validPointsCount > 0 {
			value = value / float64(validPointsCount)
		}
	case "sum":
		for _, point := range series.Points {
			if point.IsValids() {
				values := point.Values()
				tem, err := s.mathValue(values)
				if err != nil {
					return nil, valArr
				}
				value += tem
				allNull = false
			}
		}
	case "min":
		value = math.MaxFloat64
		for _, point := range series.Points {
			if point.IsValids() {
				allNull = false
				values := point.Values()
				tem, err := s.mathValue(values)
				if err != nil {
					return nil, valArr
				}
				if value > tem {
					value = tem
				}
			}
		}
	case "max":
		value = -math.MaxFloat64
		for _, point := range series.Points {
			if point.IsValids() {
				allNull = false
				values := point.Values()
				tem, err := s.mathValue(values)
				if err != nil {
					return nil, valArr
				}
				if value < tem {
					value = tem
				}
			}
		}
	case "count":
		value = float64(len(series.Points))
		allNull = false
	case "last":
		points := series.Points
		for i := len(points) - 1; i >= 0; i-- {
			if points[i].IsValids() {
				values := points[i].Values()
				tem, err := s.mathValue(values)
				if err != nil {
					return nil, valArr
				}
				value = tem
				allNull = false
				break
			}
		}
	case "median":
		var values []float64
		for _, point := range series.Points {
			if point.IsValids() {
				allNull = false
				values := point.Values()
				tem, err := s.mathValue(values)
				if err != nil {
					return nil, valArr
				}
				values = append(values, tem)
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
			if v.IsValids() {
				value++
			}
		}

		if value > 0 {
			allNull = false
		}
	}

	if allNull {
		return nil, valArr
	}

	return &value, valArr
}

func (reducer *mathReducer) mathValue(values []float64) (float64, error) {
	value := float64(0)
	switch reducer.Opt {
	case "/":
		if len(values) < 2 {
			return value, errors.Errorf("point values length is too short")
		}
		if values[1] == 0 {
			return 1, nil
		}
		return values[0] / values[1], nil

	}
	return value, errors.Errorf("the reducer operator:%s is ilegal", reducer.Opt)
}

func newMathReducer(cond *monitor.Condition) (*mathReducer, error) {
	return &mathReducer{
		Type:   cond.Type,
		Opt:    cond.Operators[0],
		Params: cond.Params,
	}, nil

}
