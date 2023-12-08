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
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"yunion.io/x/onecloud/pkg/apis/monitor"
)

func TestSimpleReducer(t *testing.T) {
	Convey("Test simple reducer by calculating", t, func() {

		Convey("sum", func() {
			result := testReducer("sum", 1, 2, 3)
			So(result, ShouldEqual, float64(6))
		})

		Convey("min", func() {
			result := testReducer("min", 3, 2, 1)
			So(result, ShouldEqual, float64(1))
		})

		Convey("max", func() {
			result := testReducer("max", 1, 2, 3)
			So(result, ShouldEqual, float64(3))
		})

		Convey("count", func() {
			result := testReducer("count", 1, 2, 3000)
			So(result, ShouldEqual, float64(3))
		})

		Convey("last", func() {
			result := testReducer("last", 1, 2, 3000)
			So(result, ShouldEqual, float64(3000))
		})

		Convey("median odd amount of numbers", func() {
			result := testReducer("median", 1, 2, 3000)
			So(result, ShouldEqual, float64(2))
		})

		Convey("median even amount of numbers", func() {
			result := testReducer("median", 1, 2, 4, 3000)
			So(result, ShouldEqual, float64(3))
		})

		Convey("median with one values", func() {
			result := testReducer("median", 1)
			So(result, ShouldEqual, float64(1))
		})

		Convey("median should ignore null values", func() {
			reducer := newSimpleReducerByType("median")
			series := &monitor.TimeSeries{
				Name: "test time series",
			}

			series.Points = append(series.Points, monitor.NewTimePoint(nil, 1))
			series.Points = append(series.Points, monitor.NewTimePoint(nil, 2))
			series.Points = append(series.Points, monitor.NewTimePoint(nil, 3))
			series.Points = append(series.Points, monitor.NewTimePointByVal(1, 4))
			series.Points = append(series.Points, monitor.NewTimePointByVal(2, 5))
			series.Points = append(series.Points, monitor.NewTimePointByVal(3, 6))

			result, _ := reducer.Reduce(series)
			So(result, ShouldNotBeNil)
			So(*result, ShouldEqual, 2)
		})

		Convey("avg", func() {
			result := testReducer("avg", 1, 2, 3)
			So(result, ShouldEqual, float64(2))
		})

		Convey("count_non_null", func() {
			Convey("with null values and real values", func() {
				reducer := newSimpleReducerByType("count_non_null")
				series := &monitor.TimeSeries{
					Name: "test time series",
				}

				series.Points = append(series.Points, monitor.NewTimePoint(nil, 1))
				series.Points = append(series.Points, monitor.NewTimePoint(nil, 2))
				series.Points = append(series.Points, monitor.NewTimePointByVal(3, 3))
				series.Points = append(series.Points, monitor.NewTimePointByVal(4, 4))
				reduce, _ := reducer.Reduce(series)
				So(reduce, ShouldNotBeNil)
				So(*reduce, ShouldEqual, 2)
			})

			Convey("with null values", func() {
				reducer := newSimpleReducerByType("count_non_null")
				series := &monitor.TimeSeries{
					Name: "test time series",
				}

				series.Points = append(series.Points, monitor.NewTimePoint(nil, 1))
				series.Points = append(series.Points, monitor.NewTimePoint(nil, 2))
				reduce, _ := reducer.Reduce(series)
				So(reduce, ShouldBeNil)
			})
		})

		Convey("avg of number values and null values should ignore nulls", func() {
			reduer := newSimpleReducerByType("avg")
			series := &monitor.TimeSeries{
				Name: "test time series",
			}

			series.Points = append(series.Points, monitor.NewTimePoint(nil, 1))
			series.Points = append(series.Points, monitor.NewTimePoint(nil, 2))
			series.Points = append(series.Points, monitor.NewTimePoint(nil, 3))
			series.Points = append(series.Points, monitor.NewTimePointByVal(3, 4))
			reduce, _ := reduer.Reduce(series)
			So(*reduce, ShouldEqual, 3)
		})

		Convey("diff one point", func() {
			result := testReducer("diff", 30)
			So(result, ShouldEqual, float64(0))
		})

		Convey("diff two points", func() {
			result := testReducer("diff", 30, 40)
			So(result, ShouldEqual, float64(10))
		})

		Convey("diff three points", func() {
			result := testReducer("diff", 30, 40, 40)
			So(result, ShouldEqual, float64(10))
		})

		Convey("diff with only nulls", func() {
			reducer := newSimpleReducerByType("diff")
			series := &monitor.TimeSeries{
				Name: "test time serie",
			}

			series.Points = append(series.Points, monitor.NewTimePoint(nil, 1))
			series.Points = append(series.Points, monitor.NewTimePoint(nil, 2))
			reduce, _ := reducer.Reduce(series)
			So(reduce, ShouldBeNil)
		})

		Convey("percent_diff one point", func() {
			result := testReducer("percent_diff", 40)
			So(result, ShouldEqual, float64(0))
		})

		Convey("percent_diff two points", func() {
			result := testReducer("percent_diff", 30, 40)
			So(result, ShouldEqual, float64(33.33333333333333))
		})

		Convey("percent_diff three points", func() {
			result := testReducer("percent_diff", 30, 40, 40)
			So(result, ShouldEqual, float64(33.33333333333333))
		})

		Convey("percent_diff with only nulls", func() {
			reducer := newSimpleReducerByType("percent_diff")
			series := &monitor.TimeSeries{
				Name: "test time serie",
			}

			series.Points = append(series.Points, monitor.NewTimePoint(nil, 1))
			series.Points = append(series.Points, monitor.NewTimePoint(nil, 2))
			reduce, _ := reducer.Reduce(series)
			So(reduce, ShouldBeNil)
		})
	})
}

func testReducer(reducerType string, datapoints ...float64) float64 {
	reducer := newSimpleReducerByType(reducerType)
	serires := &monitor.TimeSeries{
		Name: "test time series",
	}

	for idx := range datapoints {
		val := datapoints[idx]
		serires.Points = append(serires.Points, monitor.NewTimePoint(&val, 1234134))
	}
	reduce, _ := reducer.Reduce(serires)
	return *reduce
}
