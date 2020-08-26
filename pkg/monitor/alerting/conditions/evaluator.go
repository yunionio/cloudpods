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
	"fmt"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/validators"
)

// AlertEvaluator evaluates the reduced value of a timeserie.
// Returning true if a timeseries is violating the condition
// ex: ThresholdEvaluator, NoValueEvaluator, RangeEvaluator
type AlertEvaluator interface {
	Eval(reducedValue *float64) bool
	String() string
}

type noValueEvaluator struct{}

func (e *noValueEvaluator) Eval(reducedValue *float64) bool {
	return reducedValue == nil
}

func (e *noValueEvaluator) String() string {
	return "no_data"
}

type thresholdEvaluator struct {
	Type      string
	Threshold float64
}

func newThresholdEvaluator(cond *monitor.Condition) (*thresholdEvaluator, error) {
	defaultEval := &thresholdEvaluator{
		Type:      cond.Type,
		Threshold: cond.Params[0],
	}
	return defaultEval, nil
}

func (e *thresholdEvaluator) Eval(reducedValue *float64) bool {
	if reducedValue == nil {
		return false
	}

	val := *reducedValue
	switch e.Type {
	case "gt":
		return val > e.Threshold
	case "lt":
		return val < e.Threshold
	}

	return false
}

func (e *thresholdEvaluator) String() string {
	var op string
	switch e.Type {
	case "gt":
		op = ">"
	case "lt":
		op = "<"
	}
	return fmt.Sprintf("%s %.2f", op, e.Threshold)
}

type rangedEvaluator struct {
	Type  string
	Lower float64
	Upper float64
}

func newRangedEvaluator(cond *monitor.Condition) (*rangedEvaluator, error) {
	if len(cond.Params) == 0 {
		return nil, errors.Wrap(validators.ErrMissingParameterThreshold, "RangedEvaluator parameter is empty")
	}
	if len(cond.Params) == 1 {
		return nil, errors.Wrap(validators.ErrMissingParameterThreshold, "RangedEvaluator parameter second parameter is missing")
	}

	rangedEval := &rangedEvaluator{
		Type:  cond.Type,
		Lower: cond.Params[0],
		Upper: cond.Params[1],
	}
	return rangedEval, nil
}

func (e *rangedEvaluator) Eval(reducedValue *float64) bool {
	if reducedValue == nil {
		return false
	}
	val := *reducedValue
	switch e.Type {
	case "within_range":
		return (e.Lower < val && e.Upper > val) || (e.Upper < val && e.Lower > val)
	case "outside_range":
		return (e.Upper < val && e.Lower < val) || (e.Upper > val && e.Lower > val)
	}
	return false
}

func (e *rangedEvaluator) String() string {
	return fmt.Sprintf("%s [%.2f, %.2f]", e.Type, e.Lower, e.Upper)
}

// NewAlertEvaluator is a factory function for returning
// an `AlertEvaluator` depending on the input condition.
func NewAlertEvaluator(cond *monitor.Condition) (AlertEvaluator, error) {
	typ := cond.Type
	if typ == "" {
		return nil, validators.ErrMissingParameterType
	}

	if utils.IsInStringArray(typ, validators.EvaluatorDefaultTypes) {
		return newThresholdEvaluator(cond)
	}
	if utils.IsInStringArray(typ, validators.EvaluatorRangedTypes) {
		return newRangedEvaluator(cond)
	}

	if typ == "no_value" {
		return &noValueEvaluator{}, nil
	}

	return nil, errors.Wrapf(validators.ErrInvalidEvaluatorType, "type: %s", typ)
}
