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

import "yunion.io/x/pkg/util/sets"

type ReducerType string

const (
	REDUCER_AVG            ReducerType = "avg"
	REDUCER_SUM            ReducerType = "sum"
	REDUCER_MIN            ReducerType = "min"
	REDUCER_MAX            ReducerType = "max"
	REDUCER_COUNT          ReducerType = "count"
	REDUCER_LAST           ReducerType = "last"
	REDUCER_MEDIAN         ReducerType = "median"
	REDUCER_DIFF           ReducerType = "diff"
	REDUCER_PERCENT_DIFF   ReducerType = "percent_diff"
	REDUCER_COUNT_NON_NULL ReducerType = "count_non_null"
	REDUCER_PERCENTILE     ReducerType = "percentile"
)

var ValidateReducerTypes = sets.NewString()

func init() {
	for _, rt := range []ReducerType{REDUCER_AVG, REDUCER_SUM, REDUCER_MIN,
		REDUCER_MAX, REDUCER_COUNT, REDUCER_LAST, REDUCER_MEDIAN, REDUCER_DIFF,
		REDUCER_PERCENT_DIFF, REDUCER_COUNT_NON_NULL, REDUCER_PERCENTILE} {
		ValidateReducerTypes.Insert(string(rt))
	}
}

type ReducedResult struct {
	Reducer Condition `json:"reducer"`
	Result  []float64 `json:"result"`
}
