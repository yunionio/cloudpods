package notifyclient

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
)

var (
	templatesTable        map[string]*template.Template
	notifyClientWorkerMan *appsrv.SWorkerManager
)

func init() {
	notifyClientWorkerMan = appsrv.NewWorkerManager("NotifyClientWorkerManager", 1, 50, false)
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

func getTemplate(topic string, contType string) (*template.Template, error) {
	key := fmt.Sprintf("%s.%s", topic, contType)
	if _, ok := templatesTable[key]; !ok {
		cont, err := getTemplateString(topic, contType, "")
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

func getContent(topic string, contType string, data jsonutils.JSONObject) (string, error) {
	tmpl, err := getTemplate(topic, contType)
	if err != nil {
		return "", err
	}
	buf := strings.Builder{}
	err = tmpl.Execute(&buf, data.Interface())
	if err != nil {
		return "", err
	}
	log.Debugf("notify.getContent %s %s %s %s", topic, contType, data, buf.String())
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

func RawNotify(recipientId string, isGroup bool, channels []notify.TNotifyChannel, priority notify.TNotifyPriority, event string, data jsonutils.JSONObject) {
	log.Infof("notify %s event %s priority %s data %s", recipientId, event, priority, data)
	msg := notify.SNotifyMessage{}
	if isGroup {
		msg.Gid = recipientId
	} else {
		msg.Uid = recipientId
	}
	msg.Priority = priority
	msg.ContactType = channels
	topic, _ := getContent(event, "title", data)
	if len(topic) == 0 {
		topic = event
	}
	msg.Topic = topic
	body, _ := getContent(event, "content", data)
	if len(body) == 0 {
		body = data.String()
	}
	msg.Msg = body
	log.Debugf("send notification %s %s", topic, body)
	notifyClientWorkerMan.Run(func() {
		s := auth.GetAdminSession(context.Background(), consts.GetRegion(), "")
		notify.Notifications.Send(s, msg)
	}, nil, nil)
}

func NotifyNormal(recipientId string, isGroup bool, event string, data jsonutils.JSONObject) {
	RawNotify(recipientId, isGroup,
		[]notify.TNotifyChannel{notify.NotifyByEmail, notify.NotifyByDingTalk},
		notify.NotifyPriorityNormal,
		event, data)
}

func NotifyImportant(recipientId string, isGroup bool, event string, data jsonutils.JSONObject) {
	RawNotify(recipientId, isGroup,
		[]notify.TNotifyChannel{notify.NotifyByEmail, notify.NotifyByDingTalk, notify.NotifyByMobile},
		notify.NotifyPriorityImportant,
		event, data)
}

func NotifyCritical(recipientId string, isGroup bool, event string, data jsonutils.JSONObject) {
	RawNotify(recipientId, isGroup,
		[]notify.TNotifyChannel{notify.NotifyByEmail, notify.NotifyByDingTalk, notify.NotifyByMobile},
		notify.NotifyPriorityCritical,
		event, data)
}

func SystemNotify(event string, data jsonutils.JSONObject) {
	NotifyCritical(auth.AdminCredential().GetProjectId(), true, event, data)
}

func NotifyGeneralSystemError(data jsonutils.JSONObject) {
	SystemNotify(SYSTEM_ERROR, data)
}

type sSystemErrorMsg struct {
	Id     string
	Name   string
	Event  string
	Reason string
}

func NotifySystemError(idstr string, name string, event string, reason string) {
	msg := sSystemErrorMsg{
		Id:     idstr,
		Name:   name,
		Event:  event,
		Reason: reason,
	}
	SystemNotify(SYSTEM_ERROR, jsonutils.Marshal(msg))
}

/*func NotifySystemError(id string, name string, status string, reason string) error {
	log.Errorf("ID: %s Name %s Status %s REASON %s", id, name, status, reason)
	return nil
}*/

func NotifySystemWarning(data jsonutils.JSONObject) {
	SystemNotify(SYSTEM_WARNING, data)
}
