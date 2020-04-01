package notifiers

import (
	"context"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/monitor/options"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/models"
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
	session         *mcclient.ClientSession
}

func newAutoScalingNotifier(config alerting.NotificationConfig) (alerting.Notifier, error) {
	return &AutoScalingNotifier{
		NotifierBase:    NewNotifierBase(config),
		session:         auth.GetAdminSession(context.Background(), options.Options.Region, ""),
	}, nil
}

func (as *AutoScalingNotifier) Notify(ctx *alerting.EvalContext, data jsonutils.JSONObject) error {
	scalingPolicyId, _ := data.GetString("scaling_policy_id")
	alarmID := ctx.Rule.Id
	params := jsonutils.NewDict()
	params.Set("alarm_id", jsonutils.NewString(alarmID))
	_, err := modules.ScalingPolicy.PerformAction(as.session, scalingPolicyId, "trigger", params)
	if err != nil {
		return errors.Wrap(err, "Request to trigger ScalingPolicy '%s' failed", )
	}
	return nil
}

func (as *AutoScalingNotifier) ShouldNotify(ctx context.Context, evalContext *alerting.EvalContext, notificationState *models.SAlertnotification) bool {
	if evalContext.Rule.State == monitor.AlertStateOK || evalContext.NoDataFound {
		return false
	}
	return true
}
