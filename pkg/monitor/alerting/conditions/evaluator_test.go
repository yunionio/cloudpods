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

func evalutorScenario(typ string, params []float64, reducedValue float64) bool {
	evaluator, err := NewAlertEvaluator(&monitor.Condition{Type: typ, Params: params})
	So(err, ShouldBeNil)

	return evaluator.Eval(&reducedValue)
}

func TestEvalutors(t *testing.T) {
	Convey("greater than", t, func() {
		So(evalutorScenario("gt", []float64{1}, 3), ShouldBeTrue)
		So(evalutorScenario("gt", []float64{3}, 1), ShouldBeFalse)
	})

	Convey("less than", t, func() {
		So(evalutorScenario("lt", []float64{1}, 3), ShouldBeFalse)
		So(evalutorScenario("lt", []float64{3}, 1), ShouldBeTrue)
	})

	Convey("within_range", t, func() {
		So(evalutorScenario("within_range", []float64{1, 100}, 3), ShouldBeTrue)
		So(evalutorScenario("within_range", []float64{1, 100}, 300), ShouldBeFalse)
		So(evalutorScenario("within_range", []float64{100, 1}, 3), ShouldBeTrue)
		So(evalutorScenario("within_range", []float64{100, 1}, 300), ShouldBeFalse)
	})

	Convey("outside_range", t, func() {
		So(evalutorScenario("outside_range", []float64{1, 100}, 1000), ShouldBeTrue)
		So(evalutorScenario("outside_range", []float64{1, 100}, 50), ShouldBeFalse)
		So(evalutorScenario("outside_range", []float64{100, 1}, 1000), ShouldBeTrue)
		So(evalutorScenario("outside_range", []float64{100, 1}, 50), ShouldBeFalse)
	})

	Convey("no_value", t, func() {
		Convey("should be false if series have values", func() {
			So(evalutorScenario("no_value", nil, 50), ShouldBeFalse)
		})

		Convey("should be true when the series have no value", func() {
			evaluator, err := NewAlertEvaluator(&monitor.Condition{Type: "no_value"})
			So(err, ShouldBeNil)
			So(evaluator.Eval(nil), ShouldBeTrue)
		})
	})
}
