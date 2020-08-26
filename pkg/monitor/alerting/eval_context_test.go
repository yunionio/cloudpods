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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"yunion.io/x/onecloud/pkg/apis/monitor"
)

func TestStateIsUpdatedWhenNeeded(t *testing.T) {
	ctx := NewEvalContext(context.TODO(), nil, &Rule{Conditions: []Condition{&conditionStub{firing: true}}})

	t.Run("ok -> alerting", func(t *testing.T) {
		ctx.PrevAlertState = monitor.AlertStateOK
		ctx.Rule.State = monitor.AlertStateAlerting

		if !ctx.shouldUpdateAlertState() {
			t.Fatalf("expected should updated to be true")
		}
	})

	t.Run("ok -> ok", func(t *testing.T) {
		ctx.PrevAlertState = monitor.AlertStateOK
		ctx.Rule.State = monitor.AlertStateOK

		if ctx.shouldUpdateAlertState() {
			t.Fatalf("expected should updated to be false")
		}
	})
}

func TestGetStateFromEvalContext(t *testing.T) {
	tcs := []struct {
		name     string
		expected monitor.AlertStateType
		applyFn  func(ec *EvalContext)
	}{
		{
			name:     "ok -> alerting",
			expected: monitor.AlertStateAlerting,
			applyFn: func(ec *EvalContext) {
				ec.Firing = true
				ec.PrevAlertState = monitor.AlertStateOK
			},
		},
		{
			name:     "ok -> error(alerting)",
			expected: monitor.AlertStateAlerting,
			applyFn: func(ec *EvalContext) {
				ec.PrevAlertState = monitor.AlertStateOK
				ec.Error = errors.New("test error")
				ec.Rule.ExecutionErrorState = monitor.ExecutionErrorSetAlerting
			},
		},
		{
			name:     "ok -> pending. since its been firing for less than FOR",
			expected: monitor.AlertStatePending,
			applyFn: func(ec *EvalContext) {
				ec.PrevAlertState = monitor.AlertStateOK
				ec.Firing = true
				ec.Rule.LastStateChange = time.Now().Add(-time.Minute * 2)
				ec.Rule.For = time.Minute * 5
			},
		},
		{
			name:     "ok -> pending. since it has to be pending longer than FOR and prev state is ok",
			expected: monitor.AlertStatePending,
			applyFn: func(ec *EvalContext) {
				ec.PrevAlertState = monitor.AlertStateOK
				ec.Firing = true
				ec.Rule.LastStateChange = time.Now().Add(-(time.Hour * 5))
				ec.Rule.For = time.Minute * 2
			},
		},
		{
			name:     "pending -> alerting. since its been firing for more than FOR and prev state is pending",
			expected: monitor.AlertStateAlerting,
			applyFn: func(ec *EvalContext) {
				ec.PrevAlertState = monitor.AlertStatePending
				ec.Firing = true
				ec.Rule.LastStateChange = time.Now().Add(-(time.Hour * 5))
				ec.Rule.For = time.Minute * 2
			},
		},
		{
			name:     "alerting -> alerting. should not update regardless of FOR",
			expected: monitor.AlertStateAlerting,
			applyFn: func(ec *EvalContext) {
				ec.PrevAlertState = monitor.AlertStateAlerting
				ec.Firing = true
				ec.Rule.LastStateChange = time.Now().Add(-time.Minute * 5)
				ec.Rule.For = time.Minute * 2
			},
		},
		{
			name:     "ok -> ok. should not update regardless of FOR",
			expected: monitor.AlertStateOK,
			applyFn: func(ec *EvalContext) {
				ec.PrevAlertState = monitor.AlertStateOK
				ec.Rule.LastStateChange = time.Now().Add(-time.Minute * 5)
				ec.Rule.For = time.Minute * 2
			},
		},
		{
			name:     "ok -> error(keep_last)",
			expected: monitor.AlertStateOK,
			applyFn: func(ec *EvalContext) {
				ec.PrevAlertState = monitor.AlertStateOK
				ec.Error = errors.New("test error")
				ec.Rule.ExecutionErrorState = monitor.ExecutionErrorKeepState
			},
		},
		{
			name:     "pending -> error(keep_last)",
			expected: monitor.AlertStatePending,
			applyFn: func(ec *EvalContext) {
				ec.PrevAlertState = monitor.AlertStatePending
				ec.Error = errors.New("test error")
				ec.Rule.ExecutionErrorState = monitor.ExecutionErrorKeepState
			},
		},
		{
			name:     "ok -> no_data(alerting)",
			expected: monitor.AlertStateAlerting,
			applyFn: func(ec *EvalContext) {
				ec.PrevAlertState = monitor.AlertStateOK
				ec.Rule.NoDataState = monitor.NoDataSetAlerting
				ec.NoDataFound = true
			},
		},
		{
			name:     "ok -> no_data(keep_last)",
			expected: monitor.AlertStateOK,
			applyFn: func(ec *EvalContext) {
				ec.PrevAlertState = monitor.AlertStateOK
				ec.Rule.NoDataState = monitor.NoDataKeepState
				ec.NoDataFound = true
			},
		},
		{
			name:     "pending -> no_data(keep_last)",
			expected: monitor.AlertStatePending,
			applyFn: func(ec *EvalContext) {
				ec.PrevAlertState = monitor.AlertStatePending
				ec.Rule.NoDataState = monitor.NoDataKeepState
				ec.NoDataFound = true
			},
		},
		{
			name:     "pending -> no_data(alerting) with for duration have not passed",
			expected: monitor.AlertStatePending,
			applyFn: func(ec *EvalContext) {
				ec.PrevAlertState = monitor.AlertStatePending
				ec.Rule.NoDataState = monitor.NoDataSetAlerting
				ec.NoDataFound = true
				ec.Rule.For = time.Minute * 5
				ec.Rule.LastStateChange = time.Now().Add(-time.Minute * 2)
			},
		},
		{
			name:     "pending -> no_data(alerting) should set alerting since time passed FOR",
			expected: monitor.AlertStateAlerting,
			applyFn: func(ec *EvalContext) {
				ec.PrevAlertState = monitor.AlertStatePending
				ec.Rule.NoDataState = monitor.NoDataSetAlerting
				ec.NoDataFound = true
				ec.Rule.For = time.Minute * 2
				ec.Rule.LastStateChange = time.Now().Add(-time.Minute * 5)
			},
		},
		{
			name:     "pending -> error(alerting) with for duration have not passed ",
			expected: monitor.AlertStatePending,
			applyFn: func(ec *EvalContext) {
				ec.PrevAlertState = monitor.AlertStatePending
				ec.Rule.ExecutionErrorState = monitor.ExecutionErrorSetAlerting
				ec.Error = errors.New("test error")
				ec.Rule.For = time.Minute * 5
				ec.Rule.LastStateChange = time.Now().Add(-time.Minute * 2)
			},
		},
		{
			name:     "pending -> error(alerting) should set alerting since time passed FOR",
			expected: monitor.AlertStateAlerting,
			applyFn: func(ec *EvalContext) {
				ec.PrevAlertState = monitor.AlertStatePending
				ec.Rule.ExecutionErrorState = monitor.ExecutionErrorSetAlerting
				ec.Error = errors.New("test error")
				ec.Rule.For = time.Minute * 2
				ec.Rule.LastStateChange = time.Now().Add(-time.Minute * 5)
			},
		},
	}

	for _, tc := range tcs {
		evalContext := NewEvalContext(context.Background(), nil, &Rule{Conditions: []Condition{&conditionStub{firing: true}}})

		tc.applyFn(evalContext)
		newState := evalContext.GetNewState()
		assert.Equal(t, tc.expected, newState, "failed: %s \n expected '%s' have '%s'\n", tc.name, tc.expected, string(newState))
	}
}
