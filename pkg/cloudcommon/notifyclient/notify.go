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

package notifyclient

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/i18n"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	npk "yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	templatesTable        map[string]*template.Template
	templatesTableLock    *sync.Mutex
	notifyClientWorkerMan *appsrv.SWorkerManager

	notifyAdminUsers  []string
	notifyAdminGroups []string

	notifyclientI18nTable = i18n.Table{}
)

const (
	SUFFIX = "suffix"
)

func init() {
	notifyClientWorkerMan = appsrv.NewWorkerManager("NotifyClientWorkerManager", 1, 50, false)
	templatesTableLock = &sync.Mutex{}
	templatesTable = make(map[string]*template.Template)

	notifyclientI18nTable.Set(SUFFIX, i18n.NewTableEntry().EN("en").CN("cn"))
}

func getLangSuffix(ctx context.Context) string {
	return notifyclientI18nTable.Lookup(ctx, SUFFIX)
}

func getTemplateString(ctx context.Context, topic string, contType string, channel npk.TNotifyChannel) ([]byte, error) {
	contType = contType + "@" + getLangSuffix(ctx)
	if len(channel) > 0 {
		path := filepath.Join(consts.NotifyTemplateDir, consts.GetServiceType(), contType, fmt.Sprintf("%s.%s", topic, string(channel)))
		cont, err := ioutil.ReadFile(path)
		if err == nil {
			return cont, nil
		}
	}
	path := filepath.Join(consts.NotifyTemplateDir, consts.GetServiceType(), contType, topic)
	return ioutil.ReadFile(path)
}

func getTemplate(ctx context.Context, topic string, contType string, channel npk.TNotifyChannel) (*template.Template, error) {
	key := fmt.Sprintf("%s.%s.%s@%s", topic, contType, channel, getLangSuffix(ctx))
	templatesTableLock.Lock()
	defer templatesTableLock.Unlock()

	if _, ok := templatesTable[key]; !ok {
		cont, err := getTemplateString(ctx, topic, contType, channel)
		if err != nil {
			return nil, err
		}
		tmp := template.New(key)
		tmp.Funcs(template.FuncMap{"unescaped": unescaped})
		tmp, err = tmp.Parse(string(cont))
		if err != nil {
			return nil, err
		}
		templatesTable[key] = tmp
	}
	return templatesTable[key], nil
}

func unescaped(str string) template.HTML {
	return template.HTML(str)
}

func getContent(ctx context.Context, topic string, contType string, channel npk.TNotifyChannel, data jsonutils.JSONObject) (string, error) {
	if channel == npk.NotifyByWebhook {
		return "", nil
	}
	tmpl, err := getTemplate(ctx, topic, contType, channel)
	if err != nil {
		return "", err
	}
	buf := strings.Builder{}
	err = tmpl.Execute(&buf, data.Interface())
	if err != nil {
		return "", err
	}
	// log.Debugf("notify.getContent %s %s %s %s", topic, contType, data, buf.String())
	return buf.String(), nil
}

func NotifyWebhook(ctx context.Context, userCred mcclient.TokenCredential, obj db.IModel, action SAction) {
	ret, err := db.FetchCustomizeColumns(obj.GetModelManager(), ctx, userCred, jsonutils.NewDict(), []interface{}{obj}, stringutils2.SSortedStrings{}, false)
	if err != nil {
		log.Errorf("unable to NotifyWebhook: %v", err)
		return
	}
	if len(ret) == 0 {
		log.Errorf("unable to NotifyWebhook: details of model %q is empty", obj.GetId())
		return
	}
	event := Event.WithAction(action).WithResourceType(obj.GetModelManager())
	msg := jsonutils.NewDict()
	msg.Set("resource_type", jsonutils.NewString(event.ResourceType()))
	msg.Set("action", jsonutils.NewString(event.Action()))
	msg.Set("resource_details", ret[0])
	RawNotifyWithCtx(ctx, []string{}, false, npk.NotifyByWebhook, npk.NotifyPriorityNormal, event.String(), msg)
}

func NotifyWithCtx(ctx context.Context, recipientId []string, isGroup bool, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) {
	notify(ctx, recipientId, isGroup, priority, event, data)
}

func Notify(recipientId []string, isGroup bool, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) {
	notify(context.Background(), recipientId, isGroup, priority, event, data)
}

func NotifyWithContact(ctx context.Context, contacts []string, channel npk.TNotifyChannel, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) {
	p := sNotifyParams{
		contacts: contacts,
		priority: priority,
		channel:  channel,
		event:    event,
		data:     data,
	}
	rawNotify(ctx, p)
}

func notify(ctx context.Context, recipientId []string, isGroup bool, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) {
	switch priority {
	case npk.NotifyPriorityCritical:
		notifyCritical(ctx, recipientId, isGroup, event, data)
	case npk.NotifyPriorityImportant:
		notifyImportant(ctx, recipientId, isGroup, event, data)
	default:
		notifyNormal(ctx, recipientId, isGroup, event, data)
	}
}

func RawNotifyWithCtx(ctx context.Context, recipientId []string, isGroup bool, channel npk.TNotifyChannel, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) {
	rawNotify(ctx, sNotifyParams{
		recipientId: recipientId,
		isGroup:     isGroup,
		channel:     channel,
		priority:    priority,
		event:       event,
		data:        data,
	})
}

func RawNotify(recipientId []string, isGroup bool, channel npk.TNotifyChannel, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) {
	rawNotify(context.Background(), sNotifyParams{
		recipientId: recipientId,
		isGroup:     isGroup,
		channel:     channel,
		priority:    priority,
		event:       event,
		data:        data,
	})
}

// IntelliNotify try to create receiver nonexistent if createReceiver is set to true
func IntelliNotify(ctx context.Context, recipientId []string, isGroup bool, channel npk.TNotifyChannel, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject, createReceiver bool) {
	intelliNotify(ctx, sNotifyParams{
		recipientId:    recipientId,
		isGroup:        isGroup,
		channel:        channel,
		priority:       priority,
		event:          event,
		data:           data,
		createReceiver: createReceiver,
	})
}

const noSuchReceiver = `no such receiver whose uid is '(.*)'`

var noSuchReceiverRegexp = regexp.MustCompile(noSuchReceiver)

func intelliNotify(ctx context.Context, p sNotifyParams) {
	log.Infof("notify %s event %s priority %s", p.recipientId, p.event, p.priority)
	msg := npk.SNotifyMessage{}
	if p.isGroup {
		msg.Gid = p.recipientId
	} else {
		msg.Uid = p.recipientId
	}
	msg.Priority = p.priority
	msg.Contacts = p.contacts
	msg.ContactType = p.channel
	topic, _ := getContent(ctx, p.event, "title", p.channel, p.data)
	if len(topic) == 0 {
		topic = p.event
	}
	msg.Topic = topic
	body, _ := getContent(ctx, p.event, "content", p.channel, p.data)
	if len(body) == 0 {
		body, _ = p.data.GetString()
	}
	msg.Msg = body
	// log.Debugf("send notification %s %s", topic, body)
	notifyClientWorkerMan.Run(func() {
		s := auth.GetAdminSession(context.Background(), consts.GetRegion(), "")
		for {
			err := npk.Notifications.Send(s, msg)
			if err == nil {
				break
			}
			if !p.createReceiver {
				log.Errorf("unable to send notification: %v", err)
				break
			}
			jerr, ok := err.(*httputils.JSONClientError)
			if !ok {
				log.Errorf("unable to send notification: %v", err)
				break
			}
			if jerr.Code > 500 {
				log.Errorf("unable to send notification: %v", err)
				break
			}
			match := noSuchReceiverRegexp.FindStringSubmatch(jerr.Details)
			if match == nil || len(match) <= 1 {
				log.Errorf("unable to send notification: %v", err)
				break
			}
			receiverId := match[1]
			createData := jsonutils.NewDict()
			createData.Set("uid", jsonutils.NewString(receiverId))
			_, err = modules.NotifyReceiver.Create(s, createData)
			if err != nil {
				log.Errorf("try to create receiver %q, but failed: %v", receiverId, err)
				break
			}
			log.Infof("create receiver %q successfully", receiverId)
		}
	}, nil, nil)
}

type sNotifyParams struct {
	recipientId    []string
	isGroup        bool
	contacts       []string
	channel        npk.TNotifyChannel
	priority       npk.TNotifyPriority
	event          string
	data           jsonutils.JSONObject
	createReceiver bool
}

func rawNotify(ctx context.Context, p sNotifyParams) {
	intelliNotify(ctx, p)
}

func NotifyNormal(recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	notifyNormal(context.Background(), recipientId, isGroup, event, data)
}

func NotifyNormalWithCtx(ctx context.Context, recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	notifyNormal(ctx, recipientId, isGroup, event, data)
}

func notifyNormal(ctx context.Context, recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	p := sNotifyParams{
		recipientId: recipientId,
		isGroup:     isGroup,
		event:       event,
		data:        data,
		priority:    npk.NotifyPriorityNormal,
	}
	notifyWithChannel(ctx, p,
		npk.NotifyByEmail,
		npk.NotifyByDingTalk,
		npk.NotifyByFeishu,
		npk.NotifyByWorkwx,
		npk.NotifyByWebConsole,
	)
}

func notifyWithChannel(ctx context.Context, p sNotifyParams, channels ...npk.TNotifyChannel) {
	reps := p.recipientId
	for _, c := range channels {
		p.recipientId = []string{}
		p.contacts = []string{}
		p.channel = c
		if c == npk.NotifyByWebConsole {
			p.contacts = reps
		} else {
			p.recipientId = reps
		}
		rawNotify(ctx, p)
	}

}

func NotifyImportant(recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	notifyImportant(context.Background(), recipientId, isGroup, event, data)
}

func NotifyImportantWithCtx(ctx context.Context, recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	notifyImportant(ctx, recipientId, isGroup, event, data)
}

func notifyImportant(ctx context.Context, recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	p := sNotifyParams{
		recipientId: recipientId,
		isGroup:     isGroup,
		event:       event,
		data:        data,
		priority:    npk.NotifyPriorityNormal,
	}
	notifyWithChannel(ctx, p,
		npk.NotifyByEmail,
		npk.NotifyByDingTalk,
		npk.NotifyByMobile,
		npk.NotifyByWebConsole,
		npk.NotifyByFeishu,
		npk.NotifyByWorkwx,
	)
}

func NotifyCritical(recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	notifyCritical(context.Background(), recipientId, isGroup, event, data)
}

func NotifyCriticalWithCtx(ctx context.Context, recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	notifyCritical(ctx, recipientId, isGroup, event, data)
}

func notifyCritical(ctx context.Context, recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	p := sNotifyParams{
		recipientId: recipientId,
		isGroup:     isGroup,
		event:       event,
		data:        data,
		priority:    npk.NotifyPriorityNormal,
	}
	notifyWithChannel(ctx, p,
		npk.NotifyByEmail,
		npk.NotifyByDingTalk,
		npk.NotifyByMobile,
		npk.NotifyByWebConsole,
		npk.NotifyByFeishu,
		npk.NotifyByWorkwx,
	)
}

// NotifyAllWithoutRobot will send messages via all contacnt type from exclude robot contact type such as dingtalk-robot.
func NotifyAllWithoutRobot(recipientId []string, isGroup bool, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) error {
	return notifyRobot(context.Background(), "no", recipientId, isGroup, priority, event, data)
}

// NotifyAllWithoutRobot will send messages via all contacnt type from exclude robot contact type such as dingtalk-robot.
func NotifyAllWithoutRobotWithCtx(ctx context.Context, recipientId []string, isGroup bool, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) error {
	return notifyRobot(ctx, "no", recipientId, isGroup, priority, event, data)
}

// NotifyRobot will send messages via all robot contact type such as dingtalk-robot.
func NotifyRobot(recipientId []string, isGroup bool, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) error {
	return notifyRobot(context.Background(), "only", recipientId, isGroup, priority, event, data)
}

// NotifyRobot will send messages via all robot contact type such as dingtalk-robot.
func NotifyRobotWithCtx(ctx context.Context, recipientId []string, isGroup bool, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) error {
	return notifyRobot(ctx, "only", recipientId, isGroup, priority, event, data)
}

func notifyRobot(ctx context.Context, robot string, recipientId []string, isGroup bool, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) error {
	s := auth.GetAdminSession(ctx, consts.GetRegion(), "")
	params := jsonutils.NewDict()
	params.Set("robot", jsonutils.NewString(robot))
	result, err := modules.NotifyConfig.PerformClassAction(s, "get-types", params)
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}
	jarray, err := result.Get("types")
	if err != nil {
		return err
	}
	cTypes := jarray.(*jsonutils.JSONArray).GetStringArray()
	for _, ct := range cTypes {
		RawNotifyWithCtx(ctx, recipientId, isGroup, npk.TNotifyChannel(ct), priority, event, data)
	}
	return nil
}

func SystemNotify(priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) {
	systemNotify(context.Background(), priority, event, data)
}

func SystemNotifyWithCtx(ctx context.Context, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) {
	systemNotify(ctx, priority, event, data)
}

func systemNotify(ctx context.Context, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) {
	// userId
	notify(ctx, notifyAdminUsers, false, priority, event, data)

	// groupId
	notify(ctx, notifyAdminGroups, true, priority, event, data)
}

func NotifyGeneralSystemError(data jsonutils.JSONObject) {
	notifyGeneralSystemError(context.Background(), data)
}

func NotifyGeneralSystemErrorWithCtx(ctx context.Context, data jsonutils.JSONObject) {
	notifyGeneralSystemError(ctx, data)
}

func notifyGeneralSystemError(ctx context.Context, data jsonutils.JSONObject) {
	systemNotify(ctx, npk.NotifyPriorityCritical, SYSTEM_ERROR, data)
}

type SSystemEventMsg struct {
	Id      string
	Name    string
	Event   string
	Reason  string
	Created time.Time
}

func NotifySystemError(idstr string, name string, event string, reason string) {
	notifySystemError(context.Background(), idstr, name, event, reason)
}

func NotifySystemErrorWithCtx(ctx context.Context, idstr string, name string, event string, reason string) {
	notifySystemError(ctx, idstr, name, event, reason)
}

func notifySystemError(ctx context.Context, idstr string, name string, event string, reason string) {
	msg := SSystemEventMsg{
		Id:      idstr,
		Name:    name,
		Event:   event,
		Reason:  reason,
		Created: time.Now(),
	}
	systemNotify(ctx, npk.NotifyPriorityCritical, SYSTEM_ERROR, jsonutils.Marshal(msg))
}

func NotifyError(ctx context.Context, userCred mcclient.TokenCredential, idstr, name, event, reason string) {
	msg := SSystemEventMsg{
		Id:      idstr,
		Name:    name,
		Event:   event,
		Reason:  reason,
		Created: time.Now(),
	}
	notify(ctx, []string{userCred.GetUserId()}, false, npk.NotifyPriorityCritical, SYSTEM_ERROR, jsonutils.Marshal(msg))
}

func NotifySystemWarning(idstr string, name string, event string, reason string) {
	notifySystemWarning(context.Background(), idstr, name, event, reason)
}

func NotifySystemWarningWithCtx(ctx context.Context, idstr string, name string, event string, reason string) {
	notifySystemWarning(ctx, idstr, name, event, reason)
}

func notifySystemWarning(ctx context.Context, idstr string, name string, event string, reason string) {
	msg := SSystemEventMsg{
		Id:      idstr,
		Name:    name,
		Event:   event,
		Reason:  reason,
		Created: time.Now(),
	}
	systemNotify(ctx, npk.NotifyPriorityImportant, SYSTEM_WARNING, jsonutils.Marshal(msg))
}

func parseIdName(idName string) (string, string) {
	pos := strings.Index(idName, "\\")
	if pos > 0 {
		return idName[:pos], idName[pos+1:]
	} else {
		return "", idName
	}
}

var (
	domainCache = make(map[string]string)
)

func getIdentityId(s *mcclient.ClientSession, idName string, manager modulebase.Manager) (string, error) {
	domain, idName := parseIdName(idName)

	query := jsonutils.NewDict()
	if len(domain) > 0 {
		domainId, ok := domainCache[domain]
		if !ok {
			var err error
			domainId, err = modules.Domains.GetId(s, domain, nil)
			if err != nil {
				log.Errorf("fail to find domainId for domain %s: %s", domain, err)
				return "", err
			}
			domainCache[domain] = domainId
		}
		query.Add(jsonutils.NewString(domainId), "domain_id")
	}
	return manager.GetId(s, idName, query)
}

func FetchNotifyAdminRecipients(ctx context.Context, region string, users []string, groups []string) {
	s := auth.GetAdminSession(ctx, region, "v1")

	notifyAdminUsers = make([]string, 0)
	for _, u := range users {
		uId, err := getIdentityId(s, u, &modules.UsersV3)
		if err != nil {
			log.Warningf("fetch user %s fail: %s", u, err)
		} else {
			notifyAdminUsers = append(notifyAdminUsers, uId)
		}
	}
	notifyAdminGroups = make([]string, 0)
	for _, g := range groups {
		gId, err := getIdentityId(s, g, &modules.Groups)
		if err != nil {
			log.Warningf("fetch group %s fail: %s", g, err)
		} else {
			notifyAdminGroups = append(notifyAdminGroups, gId)
		}
	}
}
