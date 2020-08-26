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

package alerting

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"yunion.io/x/onecloud/pkg/apis/monitor"
)

type conditionStub struct {
	firing   bool
	operator string
	matches  []*monitor.EvalMatch
	noData   bool
}

func (c *conditionStub) Eval(context *EvalContext) (*ConditionResult, error) {
	return &ConditionResult{Firing: c.firing, EvalMatches: c.matches, Operator: c.operator, NoDataFound: c.noData}, nil
}

func TestAlertingEvaluationHandler(t *testing.T) {
	Convey("Test alert evaluation handler", t, func() {
		handler := NewEvalHandler()

		Convey("Show return triggered with single passing condition", func() {
			ctx := NewEvalContext(context.TODO(), nil, &Rule{
				Conditions: []Condition{&conditionStub{firing: true}},
			})

			handler.Eval(ctx)
			So(ctx.Firing, ShouldEqual, true)
			So(ctx.ConditionEvals, ShouldEqual, "true = true")
		})

		Convey("Show return triggered with single passing conditions2", func() {
			ctx := NewEvalContext(context.TODO(), nil, &Rule{
				Conditions: []Condition{&conditionStub{firing: true, operator: "and"}},
			})

			handler.Eval(ctx)
			So(ctx.Firing, ShouldEqual, true)
			So(ctx.ConditionEvals, ShouldEqual, "true = true")
		})

		Convey("Show return false with not passing asdf", func() {
			ctx := NewEvalContext(context.TODO(), nil, &Rule{
				Conditions: []Condition{
					&conditionStub{firing: true, operator: "and", matches: []*monitor.EvalMatch{{}, {}}},
					&conditionStub{firing: false, operator: "and"},
				}})

			handler.Eval(ctx)
			So(ctx.Firing, ShouldEqual, false)
			So(ctx.ConditionEvals, ShouldEqual, "[true AND false] = false")
		})

		Convey("Show return true if any of condition is passing with OR operator", func() {
			ctx := NewEvalContext(context.TODO(), nil, &Rule{
				Conditions: []Condition{
					&conditionStub{firing: true, operator: "and"},
					&conditionStub{firing: false, operator: "or"},
				},
			})

			handler.Eval(ctx)
			So(ctx.Firing, ShouldEqual, true)
			So(ctx.ConditionEvals, ShouldEqual, "[true OR false] = true")
		})

		Convey("Show return false if any of the condition is failing with AND operator", func() {
			context := NewEvalContext(context.TODO(), nil, &Rule{
				Conditions: []Condition{
					&conditionStub{firing: true, operator: "and"},
					&conditionStub{firing: false, operator: "and"},
				},
			})

			handler.Eval(context)
			So(context.Firing, ShouldEqual, false)
			So(context.ConditionEvals, ShouldEqual, "[true AND false] = false")
		})

		Convey("Show return true if one condition is failing with nested OR operator", func() {
			context := NewEvalContext(context.TODO(), nil, &Rule{
				Conditions: []Condition{
					&conditionStub{firing: true, operator: "and"},
					&conditionStub{firing: true, operator: "and"},
					&conditionStub{firing: false, operator: "or"},
				},
			})

			handler.Eval(context)
			So(context.Firing, ShouldEqual, true)
			So(context.ConditionEvals, ShouldEqual, "[[true AND true] OR false] = true")
		})

		Convey("Show return false if one condition is passing with nested OR operator", func() {
			context := NewEvalContext(context.TODO(), nil, &Rule{
				Conditions: []Condition{
					&conditionStub{firing: true, operator: "and"},
					&conditionStub{firing: false, operator: "and"},
					&conditionStub{firing: false, operator: "or"},
				},
			})

			handler.Eval(context)
			So(context.Firing, ShouldEqual, false)
			So(context.ConditionEvals, ShouldEqual, "[[true AND false] OR false] = false")
		})

		Convey("Show return false if a condition is failing with nested AND operator", func() {
			context := NewEvalContext(context.TODO(), nil, &Rule{
				Conditions: []Condition{
					&conditionStub{firing: true, operator: "and"},
					&conditionStub{firing: false, operator: "and"},
					&conditionStub{firing: true, operator: "and"},
				},
			})

			handler.Eval(context)
			So(context.Firing, ShouldEqual, false)
			So(context.ConditionEvals, ShouldEqual, "[[true AND false] AND true] = false")
		})

		Convey("Show return true if a condition is passing with nested OR operator", func() {
			context := NewEvalContext(context.TODO(), nil, &Rule{
				Conditions: []Condition{
					&conditionStub{firing: true, operator: "and"},
					&conditionStub{firing: false, operator: "or"},
					&conditionStub{firing: true, operator: "or"},
				},
			})

			handler.Eval(context)
			So(context.Firing, ShouldEqual, true)
			So(context.ConditionEvals, ShouldEqual, "[[true OR false] OR true] = true")
		})

		Convey("Should return false if no condition is firing using OR operator", func() {
			context := NewEvalContext(context.TODO(), nil, &Rule{
				Conditions: []Condition{
					&conditionStub{firing: false, operator: "or"},
					&conditionStub{firing: false, operator: "or"},
					&conditionStub{firing: false, operator: "or"},
				},
			})

			handler.Eval(context)
			So(context.Firing, ShouldEqual, false)
			So(context.ConditionEvals, ShouldEqual, "[[false OR false] OR false] = false")
		})

		Convey("Should retuasdfrn no data if one condition has nodata", func() {
			context := NewEvalContext(context.TODO(), nil, &Rule{
				Conditions: []Condition{
					&conditionStub{operator: "or", noData: false},
					&conditionStub{operator: "or", noData: false},
					&conditionStub{operator: "or", noData: false},
				},
			})

			handler.Eval(context)
			So(context.NoDataFound, ShouldBeFalse)
		})

		Convey("Should return no data if one condition has nodata", func() {
			context := NewEvalContext(context.TODO(), nil, &Rule{
				Conditions: []Condition{
					&conditionStub{operator: "and", noData: true},
				},
			})

			handler.Eval(context)
			So(context.Firing, ShouldEqual, false)
			So(context.NoDataFound, ShouldBeTrue)
		})

		Convey("Should return no data if both conditions have no data and using AND", func() {
			context := NewEvalContext(context.TODO(), nil, &Rule{
				Conditions: []Condition{
					&conditionStub{operator: "and", noData: true},
					&conditionStub{operator: "and", noData: false},
				},
			})

			handler.Eval(context)
			So(context.NoDataFound, ShouldBeFalse)
		})

		Convey("Should not return no data if both conditions have no data and using OR", func() {
			ctx := NewEvalContext(context.TODO(), nil, &Rule{
				Conditions: []Condition{
					&conditionStub{operator: "or", noData: true},
					&conditionStub{operator: "or", noData: false},
				},
			})

			handler.Eval(ctx)
			So(ctx.NoDataFound, ShouldBeTrue)
		})
	})
}
