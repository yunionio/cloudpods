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
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
)

var (
	templatesTable        map[string]*template.Template
	templatesTableLock    *sync.Mutex
	notifyClientWorkerMan *appsrv.SWorkerManager

	notifyAdminUsers  []string
	notifyAdminGroups []string
)

func init() {
	notifyClientWorkerMan = appsrv.NewWorkerManager("NotifyClientWorkerManager", 1, 50, false)
	templatesTableLock = &sync.Mutex{}
	templatesTable = make(map[string]*template.Template)
}

func getTemplateString(topic string, contType string, channel notify.TNotifyChannel) ([]byte, error) {
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

func getTemplate(topic string, contType string, channel notify.TNotifyChannel) (*template.Template, error) {
	key := fmt.Sprintf("%s.%s.%s", topic, contType, channel)
	templatesTableLock.Lock()
	defer templatesTableLock.Unlock()

	if _, ok := templatesTable[key]; !ok {
		cont, err := getTemplateString(topic, contType, channel)
		if err != nil {
			return nil, err
		}
		tmp, err := template.New(key).Parse(string(cont))
		if err != nil {
			return nil, err
		}
		templatesTable[key] = tmp
	}
	return templatesTable[key], nil
}

func getContent(topic string, contType string, channel notify.TNotifyChannel, data jsonutils.JSONObject) (string, error) {
	tmpl, err := getTemplate(topic, contType, channel)
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

func Notify(recipientId string, isGroup bool, priority notify.TNotifyPriority, event string, data jsonutils.JSONObject) {
	switch priority {
	case notify.NotifyPriorityCritical:
		NotifyCritical(recipientId, isGroup, event, data)
	case notify.NotifyPriorityImportant:
		NotifyImportant(recipientId, isGroup, event, data)
	default:
		NotifyNormal(recipientId, isGroup, event, data)
	}
}

func RawNotify(recipientId string, isGroup bool, channel notify.TNotifyChannel, priority notify.TNotifyPriority, event string, data jsonutils.JSONObject) {
	log.Infof("notify %s event %s priority %s", recipientId, event, priority)
	msg := notify.SNotifyMessage{}
	if isGroup {
		msg.Gid = recipientId
	} else {
		msg.Uid = recipientId
	}
	msg.Priority = priority
	msg.ContactType = channel
	topic, _ := getContent(event, "title", channel, data)
	if len(topic) == 0 {
		topic = event
	}
	msg.Topic = topic
	body, _ := getContent(event, "content", channel, data)
	if len(body) == 0 {
		body = data.String()
	}
	msg.Msg = body
	// log.Debugf("send notification %s %s", topic, body)
	notifyClientWorkerMan.Run(func() {
		s := auth.GetAdminSession(context.Background(), consts.GetRegion(), "")
		notify.Notifications.Send(s, msg)
	}, nil, nil)
}

func NotifyNormal(recipientId string, isGroup bool, event string, data jsonutils.JSONObject) {
	for _, c := range []notify.TNotifyChannel{
		notify.NotifyByEmail,
		notify.NotifyByDingTalk,
		notify.NotifyByWebConsole,
	} {
		RawNotify(recipientId, isGroup,
			c,
			notify.NotifyPriorityNormal,
			event, data)
	}
}

func NotifyImportant(recipientId string, isGroup bool, event string, data jsonutils.JSONObject) {
	for _, c := range []notify.TNotifyChannel{
		notify.NotifyByEmail,
		notify.NotifyByDingTalk,
		notify.NotifyByMobile,
		notify.NotifyByWebConsole,
	} {
		RawNotify(recipientId, isGroup,
			c,
			notify.NotifyPriorityImportant,
			event, data)
	}
}

func NotifyCritical(recipientId string, isGroup bool, event string, data jsonutils.JSONObject) {
	for _, c := range []notify.TNotifyChannel{
		notify.NotifyByEmail,
		notify.NotifyByDingTalk,
		notify.NotifyByMobile,
		notify.NotifyByWebConsole,
	} {
		RawNotify(recipientId, isGroup,
			c,
			notify.NotifyPriorityCritical,
			event, data)
	}
}

func SystemNotify(priority notify.TNotifyPriority, event string, data jsonutils.JSONObject) {
	for _, uid := range notifyAdminUsers {
		Notify(uid, false, priority, event, data)
	}
	for _, gid := range notifyAdminGroups {
		Notify(gid, true, priority, event, data)
	}
}

func NotifyGeneralSystemError(data jsonutils.JSONObject) {
	SystemNotify(notify.NotifyPriorityCritical, SYSTEM_ERROR, data)
}

type SSystemEventMsg struct {
	Id      string
	Name    string
	Event   string
	Reason  string
	Created time.Time
}

func NotifySystemError(idstr string, name string, event string, reason string) {
	msg := SSystemEventMsg{
		Id:      idstr,
		Name:    name,
		Event:   event,
		Reason:  reason,
		Created: time.Now(),
	}
	SystemNotify(notify.NotifyPriorityCritical, SYSTEM_ERROR, jsonutils.Marshal(msg))
}

func NotifySystemWarning(idstr string, name string, event string, reason string) {
	msg := SSystemEventMsg{
		Id:      idstr,
		Name:    name,
		Event:   event,
		Reason:  reason,
		Created: time.Now(),
	}
	SystemNotify(notify.NotifyPriorityImportant, SYSTEM_WARNING, jsonutils.Marshal(msg))
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
