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
	"strconv"
	"strings"
	"time"
)

// DefaultEvalHandler is responsible for evaluating the alert rule.
type DefaultEvalHandler struct {
	alertJobTimeout time.Duration
}

// NewEvalHandler is the `DefaultEvalHandler` constructor.
func NewEvalHandler() *DefaultEvalHandler {
	return &DefaultEvalHandler{
		alertJobTimeout: time.Second * 5,
	}
}

// Eval evaluated the alert rule.
func (e *DefaultEvalHandler) Eval(context *EvalContext) {
	firing := true
	noDataFound := true
	conditionEvals := ""

	for i := 0; i < len(context.Rule.Conditions); i++ {
		condition := context.Rule.Conditions[i]
		cr, err := condition.Eval(context)
		if err != nil {
			context.Error = err
		}

		// break if condition could not be evaluated
		if context.Error != nil {
			break
		}

		if i == 0 {
			firing = cr.Firing
			noDataFound = cr.NoDataFound
		}

		// calculating Firing based on operator
		if cr.Operator == "or" {
			firing = firing || cr.Firing
			noDataFound = noDataFound || cr.NoDataFound
		} else {
			firing = firing && cr.Firing
			noDataFound = noDataFound && cr.NoDataFound
		}

		if i > 0 {
			conditionEvals = "[" + conditionEvals + " " + strings.ToUpper(cr.Operator) + " " + strconv.FormatBool(cr.Firing) + "]"
		} else {
			conditionEvals = strconv.FormatBool(firing)
		}

		context.EvalMatches = append(context.EvalMatches, cr.EvalMatches...)
	}

	context.ConditionEvals = conditionEvals + " = " + strconv.FormatBool(firing)
	context.Firing = firing
	context.NoDataFound = noDataFound
	context.EndTime = time.Now()

	//	elapsedTime := ctx.EndTime.Sub(ctx.StartTime).Nanoseconds() / int64(time.Millisecond)
	//	metrics.MAlertingExecutionTime.Observe(float64(elapsedTime))
}
