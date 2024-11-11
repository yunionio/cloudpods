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
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/i18n"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/alerting/notifiers/templates"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/options"
)

const (
	SUFFIX = "onecloudNotifier"

	MOBILE_DEFAULT_TOPIC_CN = "monitor-cn"
	MOBILE_DEFAULT_TOPIC_EN = "monitor-en"

	MOBILE_METER_TOPIC_CN = "meter-cn"
	MOBILE_METER_TOPIC_EN = "meter-en"
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
			if settings.Channel == "" && len(settings.RobotIds) == 0 && len(settings.RoleIds) == 0 {
				return input, httperrors.NewInputParameterError("channel, robot_ids or role_ids is empty")
			}
			ids := make([]string, 0)
			for _, uid := range settings.UserIds {
				ctx := context.Background()
				ctx = context.WithValue(ctx, "alerting", uid)
				obj, err := db.UserCacheManager.FetchUserByIdOrName(ctx, uid)
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
		session:      getAdminSession(),
	}, nil
}

func getAdminSession() *mcclient.ClientSession {
	return auth.GetAdminSession(context.Background(), options.Options.Region)
}

func GetNotifyTemplateConfig(ctx *alerting.EvalContext, isRecoverd bool, matches []*monitor.EvalMatch) monitor.NotificationTemplateConfig {
	return getNotifyTemplateConfigOfLang(ctx, matches, isRecoverd, language.Chinese)
}

func getNotifyTemplateConfigOfLang(ctx *alerting.EvalContext,
	matches []*monitor.EvalMatch, isRecovered bool, lang language.Tag) monitor.NotificationTemplateConfig {
	priority := notify.NotifyPriorityNormal
	levelNormal := "普通"
	levelImportant := "重要"
	levelCritial := "致命"
	msgRecovered := "告警已恢复"
	msgNoData := "暂无数据"
	msgAlerting := "发生告警"

	transMap := map[string]string{
		levelNormal:    "Normal",
		levelImportant: "Important",
		levelCritial:   "Critical",
		msgRecovered:   "Alarm recovered",
		msgNoData:      "No data available",
		msgAlerting:    "Alerting",
	}
	trans := func(input string) string {
		if lang == language.English {
			return transMap[input]
		}
		return input
	}

	level := levelNormal
	switch ctx.Rule.Level {
	case "", "normal":
		priority = notify.NotifyPriorityNormal
	case "important":
		priority = notify.NotifyPriorityImportant
		level = levelNormal
	case "fatal", "critical":
		priority = notify.NotifyPriorityCritical
		level = levelNormal
	}
	topic := fmt.Sprintf("[%s]", trans(level))

	isRecovery := false
	var hintMsg string
	if ctx.Rule.State == monitor.AlertStateOK || isRecovered {
		isRecovery = true
		hintMsg = msgRecovered
	} else if ctx.NoDataFound {
		hintMsg = msgNoData
	} else {
		hintMsg = msgAlerting
	}
	topic = fmt.Sprintf("%s %s %s", topic, ctx.GetRuleTitle(), trans(hintMsg))
	config := ctx.GetNotificationTemplateConfig(matches)
	config.Title = topic
	config.Level = level
	config.Priority = string(priority)
	config.IsRecovery = isRecovery
	return config
}

// Notify sends the alert notification.
func (oc *OneCloudNotifier) Notify(ctx *alerting.EvalContext, _ jsonutils.JSONObject) error {
	log.Infof("Sending alert notification %s to onecloud", ctx.GetRuleTitle())
	userIds := oc.Setting.UserIds
	if len(oc.Setting.RoleIds) != 0 {
		alert, err := models.CommonAlertManager.GetAlert(ctx.Rule.Id)
		if err != nil {
			return errors.Wrapf(err, "Get alert by %s", ctx.Rule.Id)
		}
		scope := alert.GetResourceScope()
		scopeId := ""
		switch scope {
		case rbacscope.ScopeDomain:
			scopeId = alert.GetDomainId()
		case rbacscope.ScopeProject:
			scopeId = alert.GetProjectId()
		}
		roleUserIds, err := getUsersByRoles(oc.Setting.RoleIds, string(scope), scopeId)
		if err != nil {
			return errors.Wrapf(err, "getUsersByRoles with %v:%s:%s", oc.Setting.RoleIds, scope, scopeId)
		}
		userIds = append(userIds, roleUserIds...)
		userIds = sets.NewString(userIds...).List()
	}
	langNotifyGroup, _ := errgroup.WithContext(ctx.Ctx)
	if err := oc.notifyByUserIds(ctx, userIds, langNotifyGroup); err != nil {
		return errors.Wrapf(err, "notifyByUserIds with %v", userIds)
	}

	if len(oc.Setting.RobotIds) != 0 {
		withLangTag := appctx.WithLangTag(context.Background(), language.English)
		langNotifyGroup.Go(func() error {
			return oc.notifyByContextLang(withLangTag, ctx, []string{})
		})
	}
	return langNotifyGroup.Wait()
}

func (oc *OneCloudNotifier) notifyByUserIds(ctx *alerting.EvalContext, userIds []string, errGrp *errgroup.Group) error {
	langIdsMap, err := GetUserLangIdsMap(userIds)
	if err != nil {
		return errors.Wrapf(err, "GetUserLangIdsMap: %v", userIds)
	}
	for lang, _ := range langIdsMap {
		ids := langIdsMap[lang]
		langTag, _ := language.Parse(lang)
		langStr := i18nTable.LookupByLang(langTag, SUFFIX)
		langContext := appctx.WithLangTag(context.Background(), getLangBystr(langStr))
		errGrp.Go(func() error {
			return oc.notifyByContextLang(langContext, ctx, ids)
		})
	}
	return nil
}

func getUsersByRoles(roleIds []string, roleScope string, scopeId string) ([]string, error) {
	s := getAdminSession()
	return modules.RoleAssignments.GetUserIdsByRolesInScope(s, roleIds, rbacscope.String2Scope(roleScope), scopeId)
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
	errs := []error{}
	if len(evalCtx.EvalMatches) > 0 {
		if err := oc.notifyMatchesByContextLang(ctx, evalCtx, evalCtx.EvalMatches, uids, false); err != nil {
			errs = append(errs, errors.Wrapf(err, "notify alerting matches"))
		}
	}
	if evalCtx.HasRecoveredMatches() {
		if err := oc.notifyMatchesByContextLang(ctx, evalCtx, evalCtx.GetRecoveredMatches(), uids, true); err != nil {
			errs = append(errs, errors.Wrapf(err, "notify recovered matches"))
		}
	}
	return errors.NewAggregate(errs)
}

func (oc *OneCloudNotifier) notifyMatchesByContextLang(
	ctx context.Context,
	evalCtx *alerting.EvalContext,
	matches []*monitor.EvalMatch,
	uids []string,
	isRecoverd bool) error {
	lang := appctx.Lang(ctx)
	config := getNotifyTemplateConfigOfLang(evalCtx, matches, isRecoverd, lang)
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
		Robots:      oc.Setting.RobotIds,
		ContactType: notify.TNotifyChannel(oc.Setting.Channel),
		Topic:       config.Title,
		Priority:    notify.TNotifyPriority(config.Priority),
		Msg:         content,
	}

	factory := new(sendBodyFactory)
	sendImp := factory.newSendnotify(evalCtx, oc, msg, config)

	return sendImp.send()
}

func (oc *OneCloudNotifier) newRemoteMobileContent(config *monitor.NotificationTemplateConfig,
	evalCtx *alerting.EvalContext, lang language.Tag) string {
	db := monitor.METRIC_DATABASE_TELE
	if len(evalCtx.Rule.RuleDescription) != 0 {
		db = evalCtx.Rule.RuleDescription[0].Database
	}
	switch db {
	case monitor.METRIC_DATABASE_METER:
		return oc.newMeterRemoteMobileContent(config, evalCtx, lang)
	default:
		return oc.newDefaultRemoteMobileContent(config, evalCtx, lang)
	}
}

func mobileContent(alertName string, typeStr string) string {
	content := make([][]string, 2)
	content[0] = []string{"alert_name", alertName}
	content[1] = []string{"type", typeStr}
	return jsonutils.Marshal(content).String()
}

func (oc *OneCloudNotifier) newDefaultRemoteMobileContent(config *monitor.NotificationTemplateConfig,
	evalCtx *alerting.EvalContext, lang language.Tag) string {
	typ := ""
	switch lang {
	case language.English:
		typ = "policy"
		config.Title = MOBILE_DEFAULT_TOPIC_EN
	default:
		typ = "告警策略"
		config.Title = MOBILE_DEFAULT_TOPIC_CN
	}
	return mobileContent(evalCtx.Rule.Name, typ)
}

func (oc *OneCloudNotifier) newMeterRemoteMobileContent(config *monitor.NotificationTemplateConfig,
	evalCtx *alerting.EvalContext, lang language.Tag) string {
	typ := ""
	switch lang {
	case language.English:
		config.Title = MOBILE_DEFAULT_TOPIC_EN
		typ = "budget"
	default:
		typ = "预算"
		config.Title = MOBILE_DEFAULT_TOPIC_CN
	}
	customizeConfig := new(monitor.MeterCustomizeConfig)
	evalCtx.Rule.CustomizeConfig.Unmarshal(customizeConfig)
	return mobileContent(customizeConfig.Name, typ)
}

func GetUserLangIdsMap(ids []string) (map[string][]string, error) {
	if len(ids) == 0 {
		return map[string][]string{}, nil
	}
	session := getAdminSession()
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

func (f *sendBodyFactory) newSendnotify(evalCtx *alerting.EvalContext, notifier *OneCloudNotifier,
	message notify.SNotifyMessage,
	config monitor.NotificationTemplateConfig) Isendnotify {
	def := new(sendnotifyBase)
	def.OneCloudNotifier = notifier
	def.evalCtx = *evalCtx
	def.msg = message
	def.config = config
	// 系统内置报警处理
	if len(notifier.Setting.UserIds) == 0 && len(notifier.Setting.RobotIds) == 0 {
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
	case string(notify.NotifyByRobot):
		robot := new(sendRobotImpl)
		robot.sendnotifyBase = def
		return robot
	default:
		return def
	}
}

type Isendnotify interface {
	send() error
	execNotifyFunc() error
}

type sendnotifyBase struct {
	*OneCloudNotifier
	evalCtx alerting.EvalContext
	msg     notify.SNotifyMessage
	config  monitor.NotificationTemplateConfig
}

func (s *sendnotifyBase) send() error {
	return SendNotifyInfo(s, s)
	//return notify.Notifications.Send(s.session, s.msg)
}

func (s *sendnotifyBase) execNotifyFunc() error {
	notifyclient.RawNotifyWithCtx(s.Ctx, s.msg.Uid, false, notify.TNotifyChannel(s.Setting.Channel),
		notify.TNotifyPriority(s.msg.Priority),
		"DEFAULT",
		jsonutils.Marshal(&s.config))
	return nil
}

type sendUserImpl struct {
	*sendnotifyBase
}

func (s *sendUserImpl) send() error {
	return SendNotifyInfo(s.sendnotifyBase, s)
}

func (s *sendUserImpl) execNotifyFunc() error {
	return notifyclient.NotifyAllWithoutRobotWithCtx(
		s.Ctx, s.msg.Uid, false, s.msg.Priority,
		"DEFAULT", jsonutils.Marshal(&s.config))
}

type sendSysImpl struct {
	*sendnotifyBase
}

func (s *sendSysImpl) send() error {
	return SendNotifyInfo(s.sendnotifyBase, s)
}

func (s *sendSysImpl) execNotifyFunc() error {
	notifyclient.SystemNotifyWithCtx(s.Ctx, s.msg.Priority, "DEFAULT",
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
		s.msg.Priority,
		s.msg.Topic,
		msgObj)
	return nil
}

type sendRobotImpl struct {
	*sendnotifyBase
}

func (s *sendRobotImpl) send() error {
	return SendNotifyInfo(s.sendnotifyBase, s)
}

func (s *sendRobotImpl) execNotifyFunc() error {
	return notifyclient.NotifyRobotWithCtx(s.Ctx, s.msg.Robots, s.msg.Priority,
		"DEFAULT", jsonutils.Marshal(&s.config))
}

func SendNotifyInfo(base *sendnotifyBase, imp Isendnotify) error {
	tmpMatches := base.config.Matches
	batch := 100
	for i := 0; i < len(tmpMatches); i += batch {
		split := i + batch
		if split > len(tmpMatches) {
			split = len(tmpMatches)
		}
		base.config.Matches = tmpMatches[i:split]
		base.config.ResourceName = base.evalCtx.GetResourceNameOfMatches(base.config.Matches)
		err := imp.execNotifyFunc()
		if err != nil {
			return err
		}

	}
	return nil
}
