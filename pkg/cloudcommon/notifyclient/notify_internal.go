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
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/language"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	npk "yunion.io/x/onecloud/pkg/mcclient/modules/notify"
)

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

func notifyGeneralSystemError(ctx context.Context, data jsonutils.JSONObject) {
	systemNotify(ctx, npk.NotifyPriorityCritical, SYSTEM_ERROR, data)
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

func systemNotify(ctx context.Context, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) {
	systemNotifyWithTemplateFuncs(ctx, priority, event, data, nil)
}

func systemNotifyWithTemplateFuncs(ctx context.Context, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject, templateFuncs template.FuncMap) {
	// userId
	notifyWithTemplateFuncs(ctx, notifyAdminUsers, false, priority, event, data, templateFuncs)

	// groupId
	notifyWithTemplateFuncs(ctx, notifyAdminGroups, true, priority, event, data, templateFuncs)
}

func notifyAll(ctx context.Context, recipientId []string, isGroup bool, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) error {
	return notifyAllWithTemplateFuncs(ctx, recipientId, isGroup, priority, event, data, nil)
}

func notifyAllWithTemplateFuncs(ctx context.Context, recipientId []string, isGroup bool, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject, templateFuncs template.FuncMap) error {
	s, err := AdminSessionGenerator(ctx, consts.GetRegion())
	if err != nil {
		return err
	}
	params := jsonutils.NewDict()
	result, err := npk.NotifyReceiver.PerformClassAction(s, "get-types", params)
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
		RawNotifyWithCtxAndTemplateFuncs(ctx, recipientId, isGroup, npk.TNotifyChannel(ct), priority, event, data, templateFuncs)
	}
	return nil
}

type sTarget struct {
	reIds    []string
	contacts []string
	robots   []string
}

func langRobot(ctx context.Context, robots []string) (map[language.Tag]*sTarget, error) {
	contextLang := appctx.Lang(ctx)
	robotLang, err := getRobotLang(robots)
	if err != nil {
		return nil, err
	}
	ret := make(map[language.Tag]*sTarget)
	for robot, langStr := range robotLang {
		lang, err := language.Parse(langStr)
		if err != nil {
			log.Errorf("can't parse %s to language.Tag: %v", langStr, err)
			lang = contextLang
		}
		t := ret[lang]
		if t == nil {
			ret[lang] = &sTarget{}
			t = ret[lang]
		}
		t.robots = append(t.robots, robot)
	}
	return ret, nil
}

func lang(ctx context.Context, contactType npk.TNotifyChannel, reIds []string, contacts []string) (map[language.Tag]*sTarget, error) {
	contextLang := appctx.Lang(ctx)
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

func isEmptyRecipients(recipientId, robots []string) bool {
	var recvs []string
	for _, c := range recipientId {
		if len(c) > 0 {
			recvs = append(recvs, c)
		}
	}
	for _, robot := range robots {
		if len(robot) > 0 {
			recvs = append(recvs, robot)
		}
	}
	return len(recvs) == 0
}

func genMsgViaLang(ctx context.Context, p sNotifyParams) ([]npk.SNotifyMessage, error) {
	reIds := make([]string, 0)
	s, err := AdminSessionGenerator(context.Background(), consts.GetRegion())
	if err != nil {
		return nil, err
	}
	if p.isGroup {
		// fetch uid
		uidSet := sets.NewString()
		for _, gid := range p.recipientId {
			users, err := identity.Groups.GetUsers(s, gid, nil)
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

	if isEmptyRecipients(p.recipientId, p.robots) {
		return nil, errors.Wrap(errors.ErrEmpty, "empty receipients")
	}

	if !hasTemplateOfTopic(p.event) {
		msg := npk.SNotifyMessage{}
		msg.Uid = reIds
		msg.Priority = p.priority
		msg.Robots = p.robots
		msg.Contacts = p.contacts
		msg.ContactType = p.channel
		msg.Topic = p.event
		msg.Msg = p.data.String()
		msg.Tag = p.tag
		msg.Metadata = p.metadata
		msg.IgnoreNonexistentReceiver = p.ignoreNonexistentReceiver
		return []npk.SNotifyMessage{msg}, nil
	}

	var langMap map[language.Tag]*sTarget

	if p.channel == npk.NotifyByRobot {
		langMap, err = langRobot(ctx, p.robots)
		if err != nil {
			return nil, err
		}
	} else {
		langMap, err = lang(ctx, p.channel, reIds, p.contacts)
		if err != nil {
			return nil, err
		}
	}

	msgs := make([]npk.SNotifyMessage, 0, len(langMap))
	for lang, t := range langMap {
		langSuffix := notifyclientI18nTable.LookupByLang(lang, suffix)
		msg := npk.SNotifyMessage{}
		msg.Uid = t.reIds
		msg.Priority = p.priority
		msg.Robots = p.robots
		msg.Contacts = t.contacts
		msg.ContactType = p.channel
		topic, err := getContent(langSuffix, p.event, "title", p.channel, p.data, p.templateFuncs)
		if err != nil {
			log.Warningf("get title error: %s", err)
		}
		if len(topic) == 0 {
			topic = p.event
		}
		msg.Topic = topic
		body, err := getContent(langSuffix, p.event, "content", p.channel, p.data, p.templateFuncs)
		if err != nil {
			log.Errorf("get content error: %s", err)
		}
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

type sNotifyParams struct {
	recipientId               []string
	robots                    []string
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
	templateFuncs             template.FuncMap
}

// newSNotifyParams 创建 sNotifyParams 结构体，event 和 data 是必需参数
func newSNotifyParams(event string, data jsonutils.JSONObject) sNotifyParams {
	return sNotifyParams{
		event: event,
		data:  data,
	}
}

// withRecipientAndPriority 设置 recipientId、isGroup、priority 和 templateFuncs（用于 notifyCritical/Important/Normal）
func (p sNotifyParams) withRecipientAndPriority(recipientId []string, isGroup bool, priority npk.TNotifyPriority, templateFuncs template.FuncMap) sNotifyParams {
	p.recipientId = recipientId
	p.isGroup = isGroup
	p.priority = priority
	p.templateFuncs = templateFuncs
	return p
}

// withRecipientChannelAndPriority 设置 recipientId、isGroup、channel、priority 和 templateFuncs（用于 RawNotify）
func (p sNotifyParams) withRecipientChannelAndPriority(recipientId []string, isGroup bool, channel npk.TNotifyChannel, priority npk.TNotifyPriority, templateFuncs template.FuncMap) sNotifyParams {
	p.recipientId = recipientId
	p.isGroup = isGroup
	p.channel = channel
	p.priority = priority
	p.templateFuncs = templateFuncs
	return p
}

// withRobotChannelAndPriority 设置 robots、channel、priority 和 templateFuncs（用于 NotifyRobot）
func (p sNotifyParams) withRobotChannelAndPriority(robots []string, channel npk.TNotifyChannel, priority npk.TNotifyPriority, templateFuncs template.FuncMap) sNotifyParams {
	p.robots = robots
	p.channel = channel
	p.priority = priority
	p.templateFuncs = templateFuncs
	return p
}

// withContactChannelAndPriority 设置 contacts、channel 和 priority（用于 NotifyWithContact）
func (p sNotifyParams) withContactChannelAndPriority(contacts []string, channel npk.TNotifyChannel, priority npk.TNotifyPriority) sNotifyParams {
	p.contacts = contacts
	p.channel = channel
	p.priority = priority
	return p
}

// withRecipientId 设置 recipientId
func (p sNotifyParams) withRecipientId(recipientId []string) sNotifyParams {
	p.recipientId = recipientId
	return p
}

// withRobots 设置 robots
func (p sNotifyParams) withRobots(robots []string) sNotifyParams {
	p.robots = robots
	return p
}

// withIsGroup 设置 isGroup
func (p sNotifyParams) withIsGroup(isGroup bool) sNotifyParams {
	p.isGroup = isGroup
	return p
}

// withContacts 设置 contacts
func (p sNotifyParams) withContacts(contacts []string) sNotifyParams {
	p.contacts = contacts
	return p
}

// withChannel 设置 channel
func (p sNotifyParams) withChannel(channel npk.TNotifyChannel) sNotifyParams {
	p.channel = channel
	return p
}

// withPriority 设置 priority
func (p sNotifyParams) withPriority(priority npk.TNotifyPriority) sNotifyParams {
	p.priority = priority
	return p
}

// withCreateReceiver 设置 createReceiver
func (p sNotifyParams) withCreateReceiver(createReceiver bool) sNotifyParams {
	p.createReceiver = createReceiver
	return p
}

// withTag 设置 tag
func (p sNotifyParams) withTag(tag string) sNotifyParams {
	p.tag = tag
	return p
}

// withMetadata 设置 metadata
func (p sNotifyParams) withMetadata(metadata map[string]interface{}) sNotifyParams {
	p.metadata = metadata
	return p
}

// withIgnoreNonexistentReceiver 设置 ignoreNonexistentReceiver
func (p sNotifyParams) withIgnoreNonexistentReceiver(ignoreNonexistentReceiver bool) sNotifyParams {
	p.ignoreNonexistentReceiver = ignoreNonexistentReceiver
	return p
}

// withTemplateFuncs 设置 templateFuncs
func (p sNotifyParams) withTemplateFuncs(templateFuncs template.FuncMap) sNotifyParams {
	p.templateFuncs = templateFuncs
	return p
}

func rawNotify(ctx context.Context, p sNotifyParams) {
	intelliNotify(ctx, p)
}

func notifyCritical(ctx context.Context, recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	notifyCriticalWithTemplateFuncs(ctx, recipientId, isGroup, event, data, nil)
}

func notifyCriticalWithTemplateFuncs(ctx context.Context, recipientId []string, isGroup bool, event string, data jsonutils.JSONObject, templateFuncs template.FuncMap) {
	p := newSNotifyParams(event, data).
		withRecipientAndPriority(recipientId, isGroup, npk.NotifyPriorityNormal, templateFuncs)
	notifyWithChannel(ctx, p,
		npk.NotifyByEmail,
		npk.NotifyByDingTalk,
		npk.NotifyByMobile,
		npk.NotifyByWebConsole,
		npk.NotifyByFeishu,
		npk.NotifyByWorkwx,
	)
}

func notifyImportant(ctx context.Context, recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	notifyImportantWithTemplateFuncs(ctx, recipientId, isGroup, event, data, nil)
}

func notifyImportantWithTemplateFuncs(ctx context.Context, recipientId []string, isGroup bool, event string, data jsonutils.JSONObject, templateFuncs template.FuncMap) {
	p := newSNotifyParams(event, data).
		withRecipientAndPriority(recipientId, isGroup, npk.NotifyPriorityNormal, templateFuncs)
	notifyWithChannel(ctx, p,
		npk.NotifyByEmail,
		npk.NotifyByDingTalk,
		npk.NotifyByMobile,
		npk.NotifyByWebConsole,
		npk.NotifyByFeishu,
		npk.NotifyByWorkwx,
	)
}

func notifyNormal(ctx context.Context, recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	notifyNormalWithTemplateFuncs(ctx, recipientId, isGroup, event, data, nil)
}

func notifyNormalWithTemplateFuncs(ctx context.Context, recipientId []string, isGroup bool, event string, data jsonutils.JSONObject, templateFuncs template.FuncMap) {
	p := newSNotifyParams(event, data).
		withRecipientAndPriority(recipientId, isGroup, npk.NotifyPriorityNormal, templateFuncs)
	notifyWithChannel(ctx, p,
		npk.NotifyByEmail,
		npk.NotifyByDingTalk,
		npk.NotifyByFeishu,
		npk.NotifyByWorkwx,
		npk.NotifyByWebConsole,
	)
}

func notifyWithChannel(ctx context.Context, p sNotifyParams, channels ...npk.TNotifyChannel) {
	for _, c := range channels {
		np := p
		np.channel = c
		rawNotify(ctx, np)
	}
}

const noSuchReceiver = `no such receiver whose uid is '(.*)'`

var noSuchReceiverRegexp = regexp.MustCompile(noSuchReceiver)

type notifyTask struct {
	ctx            context.Context
	msg            npk.SNotifyMessage
	createReceiver bool
}

func (t *notifyTask) Dump() string {
	return fmt.Sprintf("msg: %v createReceiver: %v", t.msg, t.createReceiver)
}

func (t *notifyTask) Run() {
	s, err := AdminSessionGenerator(t.ctx, consts.GetRegion())
	if err != nil {
		log.Errorf("fail to get session: %v", err)
	}
	for {
		err := npk.Notifications.Send(s, t.msg)
		if err == nil {
			break
		}
		if !t.createReceiver {
			log.Errorf("unable to send notification to contacts %s uid %s gid %s @%s: %s", strings.Join(t.msg.Contacts, ","), strings.Join(t.msg.Uid, ","), strings.Join(t.msg.Gid, ","), t.msg.ContactType, jsonutils.Marshal(err))
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
		_, err = npk.NotifyReceiver.Create(s, createData)
		if err != nil {
			log.Errorf("try to create receiver %q, but failed: %v", receiverId, err)
			break
		}
		log.Infof("create receiver %q successfully", receiverId)
	}
}

func intelliNotify(ctx context.Context, p sNotifyParams) {
	if isEmptyRecipients(p.recipientId, p.robots) {
		return
	}
	log.Infof("recipientId: %v, robots: %v, contacts: %v, event %s priority %s", p.recipientId, p.robots, p.contacts, p.event, p.priority)
	msgs, err := genMsgViaLang(ctx, p)
	if err != nil {
		log.Errorf("unable send notification: %v", err)
	}
	for i := range msgs {
		t := notifyTask{
			ctx:            ctx,
			createReceiver: p.createReceiver,
			msg:            msgs[i],
		}
		notifyClientWorkerMan.Run(&t, nil, nil)
	}
	// log.Debugf("send notification %s %s", topic, body)
}

func notify(ctx context.Context, recipientId []string, isGroup bool, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) {
	notifyWithTemplateFuncs(ctx, recipientId, isGroup, priority, event, data, nil)
}

func notifyWithTemplateFuncs(ctx context.Context, recipientId []string, isGroup bool, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject, templateFuncs template.FuncMap) {
	switch priority {
	case npk.NotifyPriorityCritical:
		notifyCriticalWithTemplateFuncs(ctx, recipientId, isGroup, event, data, templateFuncs)
	case npk.NotifyPriorityImportant:
		notifyImportantWithTemplateFuncs(ctx, recipientId, isGroup, event, data, templateFuncs)
	default:
		notifyNormalWithTemplateFuncs(ctx, recipientId, isGroup, event, data, templateFuncs)
	}
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
			domainId, err = identity.Domains.GetId(s, domain, nil)
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
