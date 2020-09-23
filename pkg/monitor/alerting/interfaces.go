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
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/notifydrivers"
)

type evalHandler interface {
	Eval(ctx *EvalContext)
}

type scheduler interface {
	Tick(time time.Time, execQueue chan *Job)
	Update(rules []*Rule)
}

// ConditionResult is the result of a condition evaluation.
type ConditionResult struct {
	Firing             bool
	NoDataFound        bool
	Operator           string
	EvalMatches        []*monitor.EvalMatch
	AlertOkEvalMatches []*monitor.EvalMatch
}

// Condition is responsible for evaluating an alert condition.
type Condition interface {
	Eval(result *EvalContext) (*ConditionResult, error)
}

type Notifier interface {
	notifydrivers.Notifier

	Notify(evalContext *EvalContext, params jsonutils.JSONObject) error

	// ShouldNotify checks this evaluation should send an alert notification
	ShouldNotify(ctx context.Context, evalContext *EvalContext, notificationState *models.SAlertnotification) bool
}
