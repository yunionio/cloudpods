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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/monitor/models"
)

type resultHandler interface {
	handle(ctx *EvalContext) error
}

type defaultResultHandler struct {
	notifier *notificationService
}

func newResultHandler() *defaultResultHandler {
	return &defaultResultHandler{
		notifier: newNotificationService(),
	}
}

func (handler *defaultResultHandler) handle(evalCtx *EvalContext) error {
	execErr := ""
	annotationData := jsonutils.NewDict()
	if len(evalCtx.EvalMatches) > 0 {
		annotationData.Add(jsonutils.Marshal(evalCtx.EvalMatches), "evalMatches")
	}

	if evalCtx.Error != nil {
		execErr = evalCtx.Error.Error()
		annotationData.Add(jsonutils.NewString(evalCtx.Error.Error()), "error")
	} else if evalCtx.NoDataFound {
		annotationData.Add(jsonutils.JSONTrue, "noData")
	}
	if evalCtx.shouldUpdateAlertState() {
		log.Infof("New state change, alertId %s, prevState %s, newState %s", evalCtx.Rule.Id, evalCtx.PrevAlertState, evalCtx.Rule.State)
		alert, err := models.AlertManager.GetAlert(evalCtx.Rule.Id)
		if err != nil {
			log.Errorf("get alert %s error: %v", evalCtx.Rule.Id, err)
			return errors.Wrapf(err, "result get alert %s", evalCtx.Rule.Id)
		}
		input := models.AlertSetStateInput{
			State:           evalCtx.Rule.State,
			UpdateStateTime: evalCtx.StartTime,
			ExecutionError:  execErr,
			EvalData:        annotationData,
		}
		if err := alert.SetState(input); err != nil {
			log.Errorf("Failed to set alert %s state: %v", evalCtx.Rule.Name, err)
		} else {
			// StateChanges is used for de duping alert notifications
			// when two servers are raising. This makes sure that the server
			// with the last state change always sends a notification
			evalCtx.Rule.StateChanges = alert.StateChanges

			// Update the last state change of the alert rule in memory
			evalCtx.Rule.LastStateChange = time.Now()
		}
		// TODO: save opslog
	}
	if evalCtx.Error != nil {
		return evalCtx.Error
	}
	if err := handler.notifier.SendIfNeeded(evalCtx); err != nil {
		return err
	}
	return nil
}
