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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/alerting/notifiers/templates"
	"yunion.io/x/onecloud/pkg/monitor/options"
)

func init() {
	alerting.RegisterNotifier(&alerting.NotifierPlugin{
		Type:    monitor.AlertNotificationTypeOneCloud,
		Factory: newOneCloudNotifier,
		ValidateCreateData: func(cred mcclient.IIdentityProvider, input monitor.NotificationCreateInput) (monitor.NotificationCreateInput, error) {
			settings := new(monitor.NotificationSettingOneCloud)
			if err := input.Settings.Unmarshal(settings); err != nil {
				return input, errors.Wrap(err, "unmarshal setting")
			}
			if settings.Channel == "" {
				return input, httperrors.NewInputParameterError("channel is empty")
			}
			ids := make([]string, 0)
			for _, uid := range settings.UserIds {
				obj, err := db.UserCacheManager.FetchUserByIdOrName(context.TODO(), uid)
				if err != nil {
					return input, errors.Wrapf(err, "fetch setting uid %s", uid)
				}
				ids = append(ids, obj.GetId())
			}
			settings.UserIds = ids
			input.Settings = jsonutils.Marshal(settings)
			return input, nil
		},
	})
}

// OneCloudNotifier is responsible for sending
// alert notifications over onecloud notify service.
type OneCloudNotifier struct {
	NotifierBase
	Setting *monitor.NotificationSettingOneCloud
	session *mcclient.ClientSession
}

func newOneCloudNotifier(config alerting.NotificationConfig) (alerting.Notifier, error) {
	setting := new(monitor.NotificationSettingOneCloud)
	if err := config.Settings.Unmarshal(setting); err != nil {
		return nil, errors.Wrapf(err, "unmarshal onecloud setting %s", config.Settings)
	}
	return &OneCloudNotifier{
		NotifierBase: NewNotifierBase(config),
		Setting:      setting,
		session:      auth.GetAdminSession(context.Background(), options.Options.Region, ""),
	}, nil
}

func GetNotifyTemplateConfig(ctx *alerting.EvalContext) monitor.NotificationTemplateConfig {
	priority := notify.NotifyPriorityNormal
	level := "普通"
	switch ctx.Rule.Level {
	case "", "normal":
		priority = notify.NotifyPriorityNormal
	case "important":
		priority = notify.NotifyPriorityImportant
		level = "重要"
	case "fatal", "critical":
		priority = notify.NotifyPriorityCritical
		level = "致命"
	}
	topic := fmt.Sprintf("[%s]", level)

	isRecovery := false
	if ctx.Rule.State == monitor.AlertStateOK {
		isRecovery = true
		topic = fmt.Sprintf("%s %s 告警已恢复", topic, ctx.GetRuleTitle())
	} else if ctx.NoDataFound {
		topic = fmt.Sprintf("%s %s 暂无数据", topic, ctx.GetRuleTitle())
	} else {
		topic = fmt.Sprintf("%s %s 发生告警", topic, ctx.GetRuleTitle())
	}
	config := ctx.GetNotificationTemplateConfig()
	config.Title = topic
	config.Level = level
	config.Priority = string(priority)
	config.IsRecovery = isRecovery
	return config
}

// Notify sends the alert notification.
func (oc *OneCloudNotifier) Notify(ctx *alerting.EvalContext, _ jsonutils.JSONObject) error {
	log.Infof("Sending alert notification %s to onecloud", ctx.GetRuleTitle())
	config := GetNotifyTemplateConfig(ctx)
	contentConfig := oc.buildContent(config)

	var content string
	var err error
	switch oc.Setting.Channel {
	case string(notify.NotifyByEmail):
		content, err = contentConfig.GenerateEmailMarkdown()
	default:
		content, err = contentConfig.GenerateMarkdown()
	}
	if err != nil {
		return errors.Wrap(err, "build content")
	}

	msg := notify.SNotifyMessage{
		Uid:         oc.Setting.UserIds,
		ContactType: notify.TNotifyChannel(oc.Setting.Channel),
		Topic:       config.Title,
		Priority:    notify.TNotifyPriority(config.Priority),
		Msg:         content,
	}

	factory := new(sendBodyFactory)
	sendImp := factory.newSendnotify(oc, msg, config)

	return sendImp.send()
}

func (oc *OneCloudNotifier) buildContent(config monitor.NotificationTemplateConfig) *templates.TemplateConfig {
	return templates.NewTemplateConfig(config)
}

type sendBodyFactory struct {
}

func (f *sendBodyFactory) newSendnotify(notifier *OneCloudNotifier, message notify.SNotifyMessage,
	config monitor.NotificationTemplateConfig) Isendnotify {
	def := new(sendnotifyBase)
	def.OneCloudNotifier = notifier
	def.msg = message
	def.config = config
	if len(notifier.Setting.UserIds) == 0 {
		sys := new(sendSysImpl)
		sys.sendnotifyBase = def
		return sys
	}
	switch notifier.Setting.Channel {
	case monitor.DEFAULT_SEND_NOTIFY_CHANNEL:
		user := new(sendUserImpl)
		user.sendnotifyBase = def
		return user
	default:
		return def
	}
}

type Isendnotify interface {
	send() error
}

type sendnotifyBase struct {
	*OneCloudNotifier
	msg    notify.SNotifyMessage
	config monitor.NotificationTemplateConfig
}

func (s *sendnotifyBase) send() error {
	return notify.Notifications.Send(s.session, s.msg)
}

type sendUserImpl struct {
	*sendnotifyBase
}

func (s *sendUserImpl) send() error {
	return notifyclient.NotifyAllWithoutRobot(s.Setting.UserIds, false, notify.TNotifyPriority(s.msg.Priority),
		"DEFAULT", jsonutils.Marshal(&s.config))
}

type sendSysImpl struct {
	*sendnotifyBase
}

func (s *sendSysImpl) send() error {
	notifyclient.SystemNotify(notify.TNotifyPriority(s.msg.Priority), s.msg.Topic,
		jsonutils.NewString(s.msg.Msg))
	return nil
}
