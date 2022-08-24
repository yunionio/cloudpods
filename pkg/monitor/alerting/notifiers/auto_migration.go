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
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/controller/balancer"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/options"
)

func init() {
	alerting.RegisterNotifier(&alerting.NotifierPlugin{
		Type:    monitor.AlertNotificationTypeAutoMigration,
		Factory: newAutoMigratingNotifier,
		ValidateCreateData: func(cred mcclient.IIdentityProvider, input monitor.NotificationCreateInput) (monitor.NotificationCreateInput, error) {
			settings := new(monitor.NotificationSettingAutoMigration)
			if err := input.Settings.Unmarshal(settings); err != nil {
				return input, errors.Wrap(err, "Unmarshal setting")
			}
			input.Settings = jsonutils.Marshal(settings)
			return input, nil
		},
	})
}

type autoMigrationNotifier struct {
	NotifierBase

	Settings *monitor.NotificationSettingAutoMigration
}

func newAutoMigratingNotifier(conf alerting.NotificationConfig) (alerting.Notifier, error) {
	settings := new(monitor.NotificationSettingAutoMigration)
	if err := conf.Settings.Unmarshal(settings); err != nil {
		return nil, errors.Wrap(err, "unmarshal setting")
	}
	return &autoMigrationNotifier{
		NotifierBase: NewNotifierBase(conf),
		Settings:     settings,
	}, nil
}

func (am *autoMigrationNotifier) getBalancerRules(ctx *alerting.EvalContext, alert *models.SMigrationAlert) (*balancer.Rules, error) {
	if len(ctx.EvalMatches) >= 1 {
		log.Warningf("EvalMatches great than 1, use first one")
	} else {
		return nil, errors.Errorf("Matches not >= 1 %d", len(ctx.EvalMatches))
	}
	match := ctx.EvalMatches[0]
	drv, err := balancer.GetMetricDrivers().Get(alert.GetMetricType())
	if err != nil {
		return nil, errors.Wrap(err, "Get metric driver")
	}
	log.Infof("autoMigrationNotifier for evalMatch: %s", jsonutils.Marshal(match))
	return balancer.NewRules(ctx, match, alert, drv, options.Options.AutoMigrationMustPair)
}

func (am *autoMigrationNotifier) Notify(ctx *alerting.EvalContext, data jsonutils.JSONObject) error {
	if !ctx.Firing {
		// do nothing
		return nil
	}

	alertId := ctx.Rule.Id
	man := models.GetMigrationAlertManager()
	obj, err := man.FetchById(alertId)
	if err != nil {
		return errors.Wrapf(err, "Fetch alert by Id: %q", alertId)
	}
	alert := obj.(*models.SMigrationAlert)

	rules, err := am.getBalancerRules(ctx, alert)
	if err != nil {
		return errors.Wrapf(err, "get balancer rules")
	}

	if err := balancer.DoBalance(ctx.Ctx, auth.GetAdminSession(ctx.Ctx, options.Options.Region), rules, balancer.NewRecorder()); err != nil {
		return errors.Wrap(err, "DoBalance")
	}

	return nil
}
