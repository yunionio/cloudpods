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

package notifiers

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/options"
)

func init() {
	alerting.RegisterNotifier(&alerting.NotifierPlugin{
		Type:    monitor.AlertNotificationTypeAutoScaling,
		Factory: newAutoScalingNotifier,
		ValidateCreateData: func(cred mcclient.IIdentityProvider, input monitor.NotificationCreateInput) (monitor.NotificationCreateInput, error) {
			return input, nil
		},
	})
}

type AutoScalingNotifier struct {
	NotifierBase
	session *mcclient.ClientSession
}

func newAutoScalingNotifier(config alerting.NotificationConfig) (alerting.Notifier, error) {
	return &AutoScalingNotifier{
		NotifierBase: NewNotifierBase(config),
		session:      auth.GetAdminSession(context.Background(), options.Options.Region),
	}, nil
}

func (as *AutoScalingNotifier) Notify(ctx *alerting.EvalContext, data jsonutils.JSONObject) error {
	scalingPolicyId, _ := data.GetString("scaling_policy_id")
	alarmID := ctx.Rule.Id
	params := jsonutils.NewDict()
	params.Set("alarm_id", jsonutils.NewString(alarmID))
	_, err := modules.ScalingPolicy.PerformAction(as.session, scalingPolicyId, "trigger", params)
	if err != nil {
		return errors.Wrap(err, "Request to trigger ScalingPolicy '%s' failed")
	}
	return nil
}

func (as *AutoScalingNotifier) ShouldNotify(ctx context.Context, evalContext *alerting.EvalContext,
	notificationState *models.SAlertnotification) bool {
	if evalContext.NoDataFound {
		return false
	}
	return as.NotifierBase.ShouldNotify(ctx, evalContext, notificationState)
}
