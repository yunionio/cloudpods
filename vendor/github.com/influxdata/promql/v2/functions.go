// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package promql

import (
	"math"
)

// Function represents a function of the expression language and is
// used by function nodes.
type Function struct {
	Name       string
	ArgTypes   []ValueType
	Variadic   int
	ReturnType ValueType
}

// Calculate the trend value at the given index i in raw data d.
// This is somewhat analogous to the slope of the trend at the given index.
// The argument "s" is the set of computed smoothed values.
// The argument "b" is the set of computed trend factors.
// The argument "d" is the set of raw input values.
func calcTrendValue(i int, sf, tf, s0, s1, b float64) float64 {
	if i == 0 {
		return b
	}

	x := tf * (s1 - s0)
	y := (1 - tf) * b

	return x + y
}

// linearRegression performs a least-square linear regression analysis on the
// provided SamplePairs. It returns the slope, and the intercept value at the
// provided time.
func linearRegression(samples []Point, interceptTime int64) (slope, intercept float64) {
	var (
		n            float64
		sumX, sumY   float64
		sumXY, sumX2 float64
	)
	for _, sample := range samples {
		x := float64(sample.T-interceptTime) / 1e3
		n += 1.0
		sumY += sample.V
		sumX += x
		sumXY += x * sample.V
		sumX2 += x * x
	}
	covXY := sumXY - sumX*sumY/n
	varX := sumX2 - sumX*sumX/n

	slope = covXY / varX
	intercept = sumY/n - slope*sumX/n
	return slope, intercept
}

var functions = map[string]*Function{
	"abs": {
		Name:       "abs",
		ArgTypes:   []ValueType{ValueTypeVector},
		ReturnType: ValueTypeVector,
	},
	"absent": {
		Name:       "absent",
		ArgTypes:   []ValueType{ValueTypeVector},
		ReturnType: ValueTypeVector,
	},
	"avg_over_time": {
		Name:       "avg_over_time",
		ArgTypes:   []ValueType{ValueTypeMatrix},
		ReturnType: ValueTypeVector,
	},
	"ceil": {
		Name:       "ceil",
		ArgTypes:   []ValueType{ValueTypeVector},
		ReturnType: ValueTypeVector,
	},
	"changes": {
		Name:       "changes",
		ArgTypes:   []ValueType{ValueTypeMatrix},
		ReturnType: ValueTypeVector,
	},
	"clamp_max": {
		Name:       "clamp_max",
		ArgTypes:   []ValueType{ValueTypeVector, ValueTypeScalar},
		ReturnType: ValueTypeVector,
	},
	"clamp_min": {
		Name:       "clamp_min",
		ArgTypes:   []ValueType{ValueTypeVector, ValueTypeScalar},
		ReturnType: ValueTypeVector,
	},
	"count_over_time": {
		Name:       "count_over_time",
		ArgTypes:   []ValueType{ValueTypeMatrix},
		ReturnType: ValueTypeVector,
	},
	"days_in_month": {
		Name:       "days_in_month",
		ArgTypes:   []ValueType{ValueTypeVector},
		Variadic:   1,
		ReturnType: ValueTypeVector,
	},
	"day_of_month": {
		Name:       "day_of_month",
		ArgTypes:   []ValueType{ValueTypeVector},
		Variadic:   1,
		ReturnType: ValueTypeVector,
	},
	"day_of_week": {
		Name:       "day_of_week",
		ArgTypes:   []ValueType{ValueTypeVector},
		Variadic:   1,
		ReturnType: ValueTypeVector,
	},
	"delta": {
		Name:       "delta",
		ArgTypes:   []ValueType{ValueTypeMatrix},
		ReturnType: ValueTypeVector,
	},
	"deriv": {
		Name:       "deriv",
		ArgTypes:   []ValueType{ValueTypeMatrix},
		ReturnType: ValueTypeVector,
	},
	"exp": {
		Name:       "exp",
		ArgTypes:   []ValueType{ValueTypeVector},
		ReturnType: ValueTypeVector,
	},
	"floor": {
		Name:       "floor",
		ArgTypes:   []ValueType{ValueTypeVector},
		ReturnType: ValueTypeVector,
	},
	"histogram_quantile": {
		Name:       "histogram_quantile",
		ArgTypes:   []ValueType{ValueTypeScalar, ValueTypeVector},
		ReturnType: ValueTypeVector,
	},
	"holt_winters": {
		Name:       "holt_winters",
		ArgTypes:   []ValueType{ValueTypeMatrix, ValueTypeScalar, ValueTypeScalar},
		ReturnType: ValueTypeVector,
	},
	"hour": {
		Name:       "hour",
		ArgTypes:   []ValueType{ValueTypeVector},
		Variadic:   1,
		ReturnType: ValueTypeVector,
	},
	"idelta": {
		Name:       "idelta",
		ArgTypes:   []ValueType{ValueTypeMatrix},
		ReturnType: ValueTypeVector,
	},
	"increase": {
		Name:       "increase",
		ArgTypes:   []ValueType{ValueTypeMatrix},
		ReturnType: ValueTypeVector,
	},
	"irate": {
		Name:       "irate",
		ArgTypes:   []ValueType{ValueTypeMatrix},
		ReturnType: ValueTypeVector,
	},
	"label_replace": {
		Name:       "label_replace",
		ArgTypes:   []ValueType{ValueTypeVector, ValueTypeString, ValueTypeString, ValueTypeString, ValueTypeString},
		ReturnType: ValueTypeVector,
	},
	"label_join": {
		Name:       "label_join",
		ArgTypes:   []ValueType{ValueTypeVector, ValueTypeString, ValueTypeString, ValueTypeString},
		Variadic:   -1,
		ReturnType: ValueTypeVector,
	},
	"ln": {
		Name:       "ln",
		ArgTypes:   []ValueType{ValueTypeVector},
		ReturnType: ValueTypeVector,
	},
	"log10": {
		Name:       "log10",
		ArgTypes:   []ValueType{ValueTypeVector},
		ReturnType: ValueTypeVector,
	},
	"log2": {
		Name:       "log2",
		ArgTypes:   []ValueType{ValueTypeVector},
		ReturnType: ValueTypeVector,
	},
	"max_over_time": {
		Name:       "max_over_time",
		ArgTypes:   []ValueType{ValueTypeMatrix},
		ReturnType: ValueTypeVector,
	},
	"min_over_time": {
		Name:       "min_over_time",
		ArgTypes:   []ValueType{ValueTypeMatrix},
		ReturnType: ValueTypeVector,
	},
	"minute": {
		Name:       "minute",
		ArgTypes:   []ValueType{ValueTypeVector},
		Variadic:   1,
		ReturnType: ValueTypeVector,
	},
	"month": {
		Name:       "month",
		ArgTypes:   []ValueType{ValueTypeVector},
		Variadic:   1,
		ReturnType: ValueTypeVector,
	},
	"predict_linear": {
		Name:       "predict_linear",
		ArgTypes:   []ValueType{ValueTypeMatrix, ValueTypeScalar},
		ReturnType: ValueTypeVector,
	},
	"quantile_over_time": {
		Name:       "quantile_over_time",
		ArgTypes:   []ValueType{ValueTypeScalar, ValueTypeMatrix},
		ReturnType: ValueTypeVector,
	},
	"rate": {
		Name:       "rate",
		ArgTypes:   []ValueType{ValueTypeMatrix},
		ReturnType: ValueTypeVector,
	},
	"resets": {
		Name:       "resets",
		ArgTypes:   []ValueType{ValueTypeMatrix},
		ReturnType: ValueTypeVector,
	},
	"round": {
		Name:       "round",
		ArgTypes:   []ValueType{ValueTypeVector, ValueTypeScalar},
		Variadic:   1,
		ReturnType: ValueTypeVector,
	},
	"scalar": {
		Name:       "scalar",
		ArgTypes:   []ValueType{ValueTypeVector},
		ReturnType: ValueTypeScalar,
	},
	"sort": {
		Name:       "sort",
		ArgTypes:   []ValueType{ValueTypeVector},
		ReturnType: ValueTypeVector,
	},
	"sort_desc": {
		Name:       "sort_desc",
		ArgTypes:   []ValueType{ValueTypeVector},
		ReturnType: ValueTypeVector,
	},
	"sqrt": {
		Name:       "sqrt",
		ArgTypes:   []ValueType{ValueTypeVector},
		ReturnType: ValueTypeVector,
	},
	"stddev_over_time": {
		Name:       "stddev_over_time",
		ArgTypes:   []ValueType{ValueTypeMatrix},
		ReturnType: ValueTypeVector,
	},
	"stdvar_over_time": {
		Name:       "stdvar_over_time",
		ArgTypes:   []ValueType{ValueTypeMatrix},
		ReturnType: ValueTypeVector,
	},
	"sum_over_time": {
		Name:       "sum_over_time",
		ArgTypes:   []ValueType{ValueTypeMatrix},
		ReturnType: ValueTypeVector,
	},
	"time": {
		Name:       "time",
		ArgTypes:   []ValueType{},
		ReturnType: ValueTypeScalar,
	},
	"timestamp": {
		Name:       "timestamp",
		ArgTypes:   []ValueType{ValueTypeVector},
		ReturnType: ValueTypeVector,
	},
	"vector": {
		Name:       "vector",
		ArgTypes:   []ValueType{ValueTypeScalar},
		ReturnType: ValueTypeVector,
	},
	"year": {
		Name:       "year",
		ArgTypes:   []ValueType{ValueTypeVector},
		Variadic:   1,
		ReturnType: ValueTypeVector,
	},
}

// getFunction returns a predefined Function object for the given name.
func getFunction(name string) (*Function, bool) {
	function, ok := functions[name]
	return function, ok
}

type vectorByValueHeap Vector

func (s vectorByValueHeap) Len() int {
	return len(s)
}

func (s vectorByValueHeap) Less(i, j int) bool {
	if math.IsNaN(s[i].V) {
		return true
	}
	return s[i].V < s[j].V
}

func (s vectorByValueHeap) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s *vectorByValueHeap) Push(x interface{}) {
	*s = append(*s, *(x.(*Sample)))
}

func (s *vectorByValueHeap) Pop() interface{} {
	old := *s
	n := len(old)
	el := old[n-1]
	*s = old[0 : n-1]
	return el
}

type vectorByReverseValueHeap Vector

func (s vectorByReverseValueHeap) Len() int {
	return len(s)
}

func (s vectorByReverseValueHeap) Less(i, j int) bool {
	if math.IsNaN(s[i].V) {
		return true
	}
	return s[i].V > s[j].V
}

func (s vectorByReverseValueHeap) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s *vectorByReverseValueHeap) Push(x interface{}) {
	*s = append(*s, *(x.(*Sample)))
}

func (s *vectorByReverseValueHeap) Pop() interface{} {
	old := *s
	n := len(old)
	el := old[n-1]
	*s = old[0 : n-1]
	return el
}
