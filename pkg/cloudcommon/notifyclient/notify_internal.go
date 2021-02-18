package notifyclient

import (
	"context"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/language"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/i18n"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	npk "yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/util/httputils"
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
	// userId
	notify(ctx, notifyAdminUsers, false, priority, event, data)

	// groupId
	notify(ctx, notifyAdminGroups, true, priority, event, data)
}

func notifyRobot(ctx context.Context, robot string, recipientId []string, isGroup bool, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) error {
	s, err := AdminSessionGenerator(ctx, consts.GetRegion(), "")
	if err != nil {
		return err
	}
	params := jsonutils.NewDict()
	params.Set("robot", jsonutils.NewString(robot))
	result, err := modules.NotifyReceiver.PerformClassAction(s, "get-types", params)
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
		langSuffix := notifyclientI18nTable.LookupByLang(lang, suffix)
		msg := npk.SNotifyMessage{}
		msg.Uid = t.reIds
		msg.Priority = p.priority
		msg.Contacts = t.contacts
		msg.ContactType = p.channel
		topic, _ := getContent(langSuffix, p.event, "title", p.channel, p.data)
		if len(topic) == 0 {
			topic = p.event
		}
		msg.Topic = topic
		body, _ := getContent(langSuffix, p.event, "content", p.channel, p.data)
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

const noSuchReceiver = `no such receiver whose uid is '(.*)'`

var noSuchReceiverRegexp = regexp.MustCompile(noSuchReceiver)

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
