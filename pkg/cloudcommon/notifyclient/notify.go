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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/language"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

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

	notifyclientI18nTable                        = i18n.Table{}
	AdminSessionGenerator SAdminSessionGenerator = getAdminSesion
	UserLangFetcher       SUserLangFetcher       = getUserLang
	topicWithTemplateSet                         = &sync.Map{}
	checkTemplates        bool
)

type SAdminSessionGenerator func(ctx context.Context, region string, apiVersion string) (*mcclient.ClientSession, error)
type SUserLangFetcher func(uids []string) (map[string]string, error)

func getAdminSesion(ctx context.Context, region string, apiVersion string) (*mcclient.ClientSession, error) {
	return auth.GetAdminSession(ctx, region, apiVersion), nil
}

func getUserLang(uids []string) (map[string]string, error) {
	s, err := AdminSessionGenerator(context.Background(), consts.GetRegion(), "")
	if err != nil {
		return nil, err
	}
	uidLang := make(map[string]string)
	if len(uids) > 0 {
		params := jsonutils.NewDict()
		params.Set("filter", jsonutils.NewString(fmt.Sprintf("id.in(%s)", strings.Join(uids, ","))))
		params.Set("details", jsonutils.JSONFalse)
		params.Set("scope", jsonutils.NewString("system"))
		params.Set("system", jsonutils.JSONTrue)
		ret, err := modules.UsersV3.List(s, params)
		if err != nil {
			return nil, err
		}
		for i := range ret.Data {
			id, _ := ret.Data[i].GetString("id")
			langStr, _ := ret.Data[i].GetString("lang")
			uidLang[id] = langStr
		}
	}
	return uidLang, nil
}

const (
	SUFFIX = "suffix"
)

func init() {
	notifyClientWorkerMan = appsrv.NewWorkerManager("NotifyClientWorkerManager", 1, 50, false)
	templatesTableLock = &sync.Mutex{}
	templatesTable = make(map[string]*template.Template)

	notifyclientI18nTable.Set(SUFFIX, i18n.NewTableEntry().EN("en").CN("cn"))
}

func hasTemplateOfTopic(topic string) bool {
	if checkTemplates {
		_, ok := topicWithTemplateSet.Load(topic)
		return ok
	}
	path := filepath.Join(consts.NotifyTemplateDir, consts.GetServiceType(), "content@cn")
	fileInfoList, err := ioutil.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			checkTemplates = true
			return false
		}
		log.Errorf("unable to read dir %s", path)
		return false
	}
	for i := range fileInfoList {
		topicWithTemplateSet.Store(fileInfoList[i].Name(), nil)
	}
	checkTemplates = true
	_, ok := topicWithTemplateSet.Load(topic)
	return ok
}

func getTemplateString(suffix string, topic string, contType string, channel npk.TNotifyChannel) ([]byte, error) {
	contType = contType + "@" + suffix
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

func getTemplate(suffix string, topic string, contType string, channel npk.TNotifyChannel) (*template.Template, error) {
	key := fmt.Sprintf("%s.%s.%s@%s", topic, contType, channel, suffix)
	templatesTableLock.Lock()
	defer templatesTableLock.Unlock()

	if _, ok := templatesTable[key]; !ok {
		cont, err := getTemplateString(suffix, topic, contType, channel)
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

func getContent(suffix string, topic string, contType string, channel npk.TNotifyChannel, data jsonutils.JSONObject) (string, error) {
	if channel == npk.NotifyByWebhook {
		return "", nil
	}
	tmpl, err := getTemplate(suffix, topic, contType, channel)
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

func NotifyWithTag(ctx context.Context, params SNotifyParams) {
	p := sNotifyParams{
		recipientId:               params.RecipientId,
		isGroup:                   params.IsGroup,
		event:                     params.Event,
		data:                      params.Data,
		priority:                  params.Priority,
		tag:                       params.Tag,
		metadata:                  params.Metadata,
		ignoreNonexistentReceiver: params.IgnoreNonexistentReceiver,
	}
	notifyWithChannel(ctx, p,
		npk.NotifyByEmail,
		npk.NotifyByMobile,
		npk.NotifyByDingTalk,
		npk.NotifyByFeishu,
		npk.NotifyByWorkwx,
		npk.NotifyByWebConsole,
	)
}

type SNotifyParams struct {
	RecipientId               []string
	IsGroup                   bool
	Priority                  npk.TNotifyPriority
	Event                     string
	Data                      jsonutils.JSONObject
	Tag                       string
	Metadata                  map[string]interface{}
	IgnoreNonexistentReceiver bool
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

type sTarget struct {
	reIds    []string
	contacts []string
}

func lang(ctx context.Context, contactType npk.TNotifyChannel, reIds []string, contacts []string) (map[language.Tag]*sTarget, error) {
	contextLang := i18n.Lang(ctx)
	langMap := make(map[language.Tag]*sTarget)
	insertReid := func(lang language.Tag, id string) {
		t := langMap[lang]
		if t == nil {
			t = &sTarget{}
			langMap[lang] = t
		}
		t.reIds = append(t.reIds, id)
	}
	insertContact := func(lang language.Tag, id string) {
		t := langMap[lang]
		if t == nil {
			t = &sTarget{}
			langMap[lang] = t
		}
		t.contacts = append(t.contacts, id)
	}

	uids := append([]string{}, reIds...)
	if contactType == npk.NotifyByWebConsole {
		uids = append(uids, contacts...)
	}

	uidLang, err := UserLangFetcher(uids)
	if err != nil {
		return nil, errors.Wrap(err, "unable to feth UserLang")
	}
	insert := func(id string, insertFunc func(language.Tag, string)) {
		langStr := uidLang[id]
		if len(langStr) == 0 {
			insertFunc(contextLang, id)
			return
		}
		lang, err := language.Parse(langStr)
		if err != nil {
			log.Errorf("can't parse %s to language.Tag: %v", langStr, err)
			insertFunc(contextLang, id)
			return
		}
		insertFunc(lang, id)
	}
	for _, reid := range reIds {
		insert(reid, insertReid)
	}
	if contactType == npk.NotifyByWebConsole {
		for _, contact := range contacts {
			insert(contact, insertContact)
		}
	} else {
		for _, cs := range contacts {
			insertContact(contextLang, cs)
		}
	}
	return langMap, nil
}

func genMsgViaLang(ctx context.Context, p sNotifyParams) ([]npk.SNotifyMessage, error) {
	reIds := make([]string, 0)
	s, err := AdminSessionGenerator(context.Background(), consts.GetRegion(), "")
	if err != nil {
		return nil, err
	}
	if p.isGroup {
		// fetch uid
		uidSet := sets.NewString()
		for _, gid := range p.recipientId {
			users, err := modules.Groups.GetUsers(s, gid, nil)
			if err != nil {
				return nil, errors.Wrapf(err, "Groups.GetUsers for group %q", gid)
			}
			for i := range users.Data {
				id, _ := users.Data[i].GetString("id")
				uidSet.Insert(id)
			}
		}
		for _, uid := range uidSet.UnsortedList() {
			reIds = append(reIds, uid)
		}
	} else {
		reIds = p.recipientId
	}

	if !hasTemplateOfTopic(p.event) {
		msg := npk.SNotifyMessage{}
		msg.Uid = reIds
		msg.Priority = p.priority
		msg.Contacts = p.contacts
		msg.ContactType = p.channel
		msg.Topic = p.event
		msg.Msg = p.data.String()
		msg.Tag = p.tag
		msg.Metadata = p.metadata
		msg.IgnoreNonexistentReceiver = p.ignoreNonexistentReceiver
		return []npk.SNotifyMessage{msg}, nil
	}

	langMap, err := lang(ctx, p.channel, reIds, p.contacts)
	if err != nil {
		return nil, err
	}

	msgs := make([]npk.SNotifyMessage, 0, len(langMap))
	for lang, t := range langMap {
		suffix := notifyclientI18nTable.LookupByLang(lang, SUFFIX)
		msg := npk.SNotifyMessage{}
		msg.Uid = t.reIds
		msg.Priority = p.priority
		msg.Contacts = t.contacts
		msg.ContactType = p.channel
		topic, _ := getContent(suffix, p.event, "title", p.channel, p.data)
		if len(topic) == 0 {
			topic = p.event
		}
		msg.Topic = topic
		body, _ := getContent(suffix, p.event, "content", p.channel, p.data)
		if len(body) == 0 {
			body, _ = p.data.GetString()
		}
		msg.Msg = body
		msg.Tag = p.tag
		msg.Metadata = p.metadata
		msg.IgnoreNonexistentReceiver = p.ignoreNonexistentReceiver
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

func intelliNotify(ctx context.Context, p sNotifyParams) {
	log.Infof("recipientId: %v, contacts: %v, event %s priority %s", p.recipientId, p.contacts, p.event, p.priority)
	msgs, err := genMsgViaLang(ctx, p)
	if err != nil {
		log.Errorf("unable send notification: %v", err)
	}
	for i := range msgs {
		msg := msgs[i]
		notifyClientWorkerMan.Run(func() {
			s, err := AdminSessionGenerator(context.Background(), consts.GetRegion(), "")
			if err != nil {
				log.Errorf("fail to get session: %v", err)
			}
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
	// log.Debugf("send notification %s %s", topic, body)
}

type sNotifyParams struct {
	recipientId               []string
	isGroup                   bool
	contacts                  []string
	channel                   npk.TNotifyChannel
	priority                  npk.TNotifyPriority
	event                     string
	data                      jsonutils.JSONObject
	createReceiver            bool
	tag                       string
	metadata                  map[string]interface{}
	ignoreNonexistentReceiver bool
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
		p.recipientId = reps
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
	s, err := AdminSessionGenerator(ctx, consts.GetRegion(), "")
	if err != nil {
		return err
	}
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
	s, err := AdminSessionGenerator(ctx, region, "v1")
	if err != nil {
		log.Errorf("unable to get admin session: %v", err)
	}

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
