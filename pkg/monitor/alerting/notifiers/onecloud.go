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
	"strings"

	"golang.org/x/sync/errgroup"
	"golang.org/x/text/language"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/i18n"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/alerting/notifiers/templates"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/options"
)

const (
	SUFFIX = "onecloudNotifier"

	MOBILE_TOPIC_CN = "monitor-cn"
	MOBILE_TOPIC_EN = "monitor-en"
)

var (
	i18nTable = i18n.Table{}
	i18nEnTry = i18n.NewTableEntry().EN("en").CN("cn")
)

func init() {
	i18nTable.Set(SUFFIX, i18nEnTry)

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

func GetNotifyTemplateConfigOfEN(ctx *alerting.EvalContext) monitor.NotificationTemplateConfig {
	priority := notify.NotifyPriorityNormal
	level := "Normal"
	switch ctx.Rule.Level {
	case "", "normal":
		priority = notify.NotifyPriorityNormal
	case "important":
		priority = notify.NotifyPriorityImportant
		level = "Important"
	case "fatal", "critical":
		priority = notify.NotifyPriorityCritical
		level = "Critical"
	}
	topic := fmt.Sprintf("[%s]", level)

	isRecovery := false
	if ctx.Rule.State == monitor.AlertStateOK {
		isRecovery = true
		topic = fmt.Sprintf("%s %s Alarm recovered", topic, ctx.GetRuleTitle())
	} else if ctx.NoDataFound {
		topic = fmt.Sprintf("%s %s No data available", topic, ctx.GetRuleTitle())
	} else {
		topic = fmt.Sprintf("%s %s Alarm", topic, ctx.GetRuleTitle())
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
	langIdsMap, err := GetUserLangIdsMap(oc.Setting.UserIds)
	if err != nil {
		return errors.Wrapf(err, "OneCloudNotifier getIds:%s userLang err", oc.Setting.UserIds)
	}
	langNotifyGroup, _ := errgroup.WithContext(ctx.Ctx)
	for lang, _ := range langIdsMap {
		ids := langIdsMap[lang]
		langTag, _ := language.Parse(lang)
		langStr := i18nTable.LookupByLang(langTag, SUFFIX)
		langContext := i18n.WithLangTag(context.Background(), getLangBystr(langStr))
		langNotifyGroup.Go(func() error {
			return oc.notifyByContextLang(langContext, ctx, ids)
		})
	}
	return langNotifyGroup.Wait()
}

func getLangBystr(str string) language.Tag {
	for lang, val := range i18nEnTry {
		if val == str {
			return lang
		}
	}
	return language.English
}

func (oc *OneCloudNotifier) notifyByContextLang(ctx context.Context, evalCtx *alerting.EvalContext, uids []string) error {
	var config monitor.NotificationTemplateConfig
	lang := i18n.Lang(ctx)
	switch lang {
	case language.English:
		config = GetNotifyTemplateConfigOfEN(evalCtx)
	default:
		config = GetNotifyTemplateConfig(evalCtx)
	}
	oc.filterMatchTagsForConfig(&config, ctx)

	contentConfig := oc.buildContent(config)

	var content string
	var err error
	switch oc.Setting.Channel {
	case string(notify.NotifyByEmail):
		content, err = contentConfig.GenerateEmailMarkdown()
	case string(notify.NotifyByMobile):
		content = oc.newRemoteMobileContent(&config, evalCtx, lang)
	default:
		content, err = contentConfig.GenerateMarkdown()
	}
	if err != nil {
		return errors.Wrap(err, "build content")
	}

	msg := notify.SNotifyMessage{
		Uid:         uids,
		ContactType: notify.TNotifyChannel(oc.Setting.Channel),
		Topic:       config.Title,
		Priority:    notify.TNotifyPriority(config.Priority),
		Msg:         content,
	}

	factory := new(sendBodyFactory)
	sendImp := factory.newSendnotify(oc, msg, config)

	return sendImp.send()
}

func (oc *OneCloudNotifier) newRemoteMobileContent(config *monitor.NotificationTemplateConfig,
	evalCtx *alerting.EvalContext, lang language.Tag) string {
	switch lang {
	case language.English:
		config.Title = MOBILE_TOPIC_EN
	default:
		config.Title = MOBILE_TOPIC_CN
	}
	content := jsonutils.NewDict()
	content.Set("alert_name", jsonutils.NewString(evalCtx.Rule.Name))
	return content.String()
}

func GetUserLangIdsMap(ids []string) (map[string][]string, error) {
	session := auth.GetAdminSession(context.Background(), "", "")
	langIdsMap := make(map[string][]string)
	params := jsonutils.NewDict()
	params.Set("filter", jsonutils.NewString(fmt.Sprintf("id.in(%s)", strings.Join(ids, ","))))
	params.Set("details", jsonutils.JSONFalse)
	params.Set("scope", jsonutils.NewString("system"))
	params.Set("system", jsonutils.JSONTrue)
	ret, err := modules.UsersV3.List(session, params)
	if err != nil {
		return nil, err
	}
	for i := range ret.Data {
		id, _ := ret.Data[i].GetString("id")
		langStr, _ := ret.Data[i].GetString("lang")
		if _, ok := langIdsMap[langStr]; ok {
			langIdsMap[langStr] = append(langIdsMap[langStr], id)
			continue
		}
		langIdsMap[langStr] = []string{id}
	}
	return langIdsMap, nil
}

var (
	companyInfo models.SCompanyInfo
)

func (oc *OneCloudNotifier) filterMatchTagsForConfig(config *monitor.NotificationTemplateConfig, ctx context.Context) {
	sCompanyInfo, err := models.GetCompanyInfo(ctx)
	if err != nil {
		log.Errorf("GetCompanyInfo error:%#v", err)
		return
	}
	for i, _ := range config.Matches {
		if val, ok := config.Matches[i].Tags[hostconsts.TELEGRAF_TAG_KEY_BRAND]; ok {
			if val == hostconsts.TELEGRAF_TAG_ONECLOUD_BRAND {
				config.Matches[i].Tags[hostconsts.TELEGRAF_TAG_KEY_BRAND] = sCompanyInfo.Name
			}
		}
	}
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
	case string(notify.NotifyByMobile):
		mobile := new(sendMobileImpl)
		mobile.sendnotifyBase = def
		return mobile
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
	notifyclient.RawNotifyWithCtx(s.Ctx, s.msg.Uid, false, notify.TNotifyChannel(s.Setting.Channel),
		notify.TNotifyPriority(s.msg.Priority),
		"DEFAULT",
		jsonutils.Marshal(&s.config))
	return nil
	//return notify.Notifications.Send(s.session, s.msg)
}

type sendUserImpl struct {
	*sendnotifyBase
}

func (s *sendUserImpl) send() error {
	return notifyclient.NotifyAllWithoutRobotWithCtx(s.Ctx, s.msg.Uid, false, notify.TNotifyPriority(s.msg.Priority),
		"DEFAULT", jsonutils.Marshal(&s.config))
}

type sendSysImpl struct {
	*sendnotifyBase
}

func (s *sendSysImpl) send() error {
	notifyclient.SystemNotifyWithCtx(s.Ctx, notify.TNotifyPriority(s.msg.Priority), "DEFAULT",
		jsonutils.Marshal(&s.config))
	return nil
}

type sendMobileImpl struct {
	*sendnotifyBase
}

func (s *sendMobileImpl) send() error {
	msgObj, err := jsonutils.ParseString(s.msg.Msg)
	if err != nil {
		return err
	}
	notifyclient.RawNotifyWithCtx(s.Ctx, s.msg.Uid, false, notify.TNotifyChannel(s.Setting.Channel),
		notify.TNotifyPriority(s.msg.Priority),
		s.msg.Topic,
		msgObj)
	return nil
}
