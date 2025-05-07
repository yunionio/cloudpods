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
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	npk "yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	notifyClientWorkerMan *appsrv.SWorkerManager

	notifyAdminUsers  []string
	notifyAdminGroups []string

	notifyDBHookResources sync.Map
)

func init() {
	notifyClientWorkerMan = appsrv.NewWorkerManager("NotifyClientWorkerManager", 1, 1024, false)

	// set db notify hook
	db.SetUpdateNotifyHook(func(ctx context.Context, userCred mcclient.TokenCredential, obj db.IModel) {
		_, ok := notifyDBHookResources.Load(obj.KeywordPlural())
		if !ok {
			return
		}
		EventNotify(ctx, userCred, SEventNotifyParam{
			Obj:    obj,
			Action: ActionUpdate,
		})
	})

	db.SetCustomizeNotifyHook(func(ctx context.Context, userCred mcclient.TokenCredential, action string, obj db.IModel, moreDetails jsonutils.JSONObject) {
		_, ok := notifyDBHookResources.Load(obj.KeywordPlural())
		if !ok {
			return
		}
		EventNotify(ctx, userCred, SEventNotifyParam{
			Obj:    obj,
			Action: api.SAction(action),
			ObjDetailsDecorator: func(ctx context.Context, details *jsonutils.JSONDict) {
				if moreDetails != nil {
					details.Set("customize_details", moreDetails)
				}
			},
		})
	})

	db.SetStatusChangedNotifyHook(func(ctx context.Context, userCred mcclient.TokenCredential, oldStatus, newStatus string, obj db.IModel) {
		_, ok := notifyDBHookResources.Load(obj.KeywordPlural())
		if !ok {
			return
		}
		EventNotify(ctx, userCred, SEventNotifyParam{
			Obj:    obj,
			Action: api.ActionStatusChanged,
			ObjDetailsDecorator: func(ctx context.Context, details *jsonutils.JSONDict) {
				details.Set("old_status", jsonutils.NewString(oldStatus))
				details.Set("new_status", jsonutils.NewString(newStatus))
			},
		})
	})
}

func AddNotifyDBHookResources(keywordPlurals ...string) {
	for _, kp := range keywordPlurals {
		notifyDBHookResources.Store(kp, true)
	}
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

func NotifyNormal(recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	notifyNormal(context.Background(), recipientId, isGroup, event, data)
}

func NotifyNormalWithCtx(ctx context.Context, recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	notifyNormal(ctx, recipientId, isGroup, event, data)
}

func NotifyImportant(recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	notifyImportant(context.Background(), recipientId, isGroup, event, data)
}

func NotifyImportantWithCtx(ctx context.Context, recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	notifyImportant(ctx, recipientId, isGroup, event, data)
}

func NotifyCritical(recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	notifyCritical(context.Background(), recipientId, isGroup, event, data)
}

func NotifyCriticalWithCtx(ctx context.Context, recipientId []string, isGroup bool, event string, data jsonutils.JSONObject) {
	notifyCritical(ctx, recipientId, isGroup, event, data)
}

// NotifyAllWithoutRobot will send messages via all contacnt type from exclude robot contact type such as dingtalk-robot.
func NotifyAllWithoutRobot(recipientId []string, isGroup bool, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) error {
	return notifyAll(context.Background(), recipientId, isGroup, priority, event, data)
}

// NotifyAllWithoutRobot will send messages via all contacnt type from exclude robot contact type such as dingtalk-robot.
func NotifyAllWithoutRobotWithCtx(ctx context.Context, recipientId []string, isGroup bool, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) error {
	return notifyAll(ctx, recipientId, isGroup, priority, event, data)
}

// NotifyRobot will send messages via all robot contact type such as dingtalk-robot.
func NotifyRobot(robotIds []string, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) error {
	return NotifyRobotWithCtx(context.Background(), robotIds, priority, event, data)
}

// NotifyRobot will send messages via all robot contact type such as dingtalk-robot.
func NotifyRobotWithCtx(ctx context.Context, robotIds []string, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) error {
	rawNotify(ctx, sNotifyParams{
		robots:   robotIds,
		channel:  npk.NotifyByRobot,
		priority: priority,
		event:    event,
		data:     data,
	})
	return nil
}

func SystemNotify(priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) {
	systemNotify(context.Background(), priority, event, data)
}

func SystemNotifyWithCtx(ctx context.Context, priority npk.TNotifyPriority, event string, data jsonutils.JSONObject) {
	systemNotify(ctx, priority, event, data)
}

func NotifyGeneralSystemError(data jsonutils.JSONObject) {
	notifyGeneralSystemError(context.Background(), data)
}

func NotifyGeneralSystemErrorWithCtx(ctx context.Context, data jsonutils.JSONObject) {
	notifyGeneralSystemError(ctx, data)
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

func NotifyWebhook(ctx context.Context, userCred mcclient.TokenCredential, obj db.IModel, action api.SAction) {
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

type SEventMessage struct {
	ResourceType    string              `json:"resource_type"`
	Action          string              `json:"action"`
	ResourceDetails *jsonutils.JSONDict `json:"resource_details"`
}

type SEventNotifyParam struct {
	Obj                 db.IModel
	ResourceType        string
	Action              api.SAction
	IsFail              bool
	ObjDetailsDecorator func(context.Context, *jsonutils.JSONDict)
	AdvanceDays         int
}

type eventTask struct {
	params api.NotificationManagerEventNotifyInput
}

func (t *eventTask) Dump() string {
	return fmt.Sprintf("eventTask params: %v", t.params)
}

func (t *eventTask) Run() {
	s, err := AdminSessionGenerator(context.Background(), "")
	if err != nil {
		log.Errorf("unable to get admin session: %v", err)
		return
	}
	_, err = npk.Notification.PerformClassAction(s, "event-notify", jsonutils.Marshal(t.params))
	if err != nil {
		log.Errorf("unable to EventNotify: %s", err)
	}
}

func EventNotify(ctx context.Context, userCred mcclient.TokenCredential, ep SEventNotifyParam) {
	var objDetails *jsonutils.JSONDict
	if ep.Action == ActionDelete || ep.Action == ActionSyncDelete {
		objDetails = jsonutils.Marshal(ep.Obj).(*jsonutils.JSONDict)
	} else {
		ret, err := db.FetchCustomizeColumns(ep.Obj.GetModelManager(), ctx, userCred, jsonutils.NewDict(), []interface{}{ep.Obj}, nil, false)
		if err != nil {
			log.Errorf("unable to FetchCustomizeColumns: %v", err)
			return
		}
		if len(ret) == 0 {
			log.Errorf("unable to FetchCustomizeColumns: details of model %q is empty", ep.Obj.GetId())
			return
		}
		objDetails = ret[0]
	}
	if ep.ObjDetailsDecorator != nil {
		ep.ObjDetailsDecorator(ctx, objDetails)
	}
	rt := ep.ResourceType
	if len(rt) == 0 {
		rt = ep.Obj.GetModelManager().Keyword()
	}
	event := api.Event.WithAction(ep.Action).WithResourceType(rt)
	if ep.IsFail {
		event = event.WithResult(api.ResultFailed)
	}
	var (
		projectId       string
		projectDomainId string
	)
	ownerId := ep.Obj.GetOwnerId()
	if ownerId != nil {
		projectId = ownerId.GetProjectId()
		projectDomainId = ownerId.GetProjectDomainId()
	}
	params := api.NotificationManagerEventNotifyInput{
		ReceiverIds:     []string{userCred.GetUserId()},
		ResourceDetails: objDetails,
		Event:           event.String(),
		AdvanceDays:     ep.AdvanceDays,
		Priority:        string(npk.NotifyPriorityNormal),
		ProjectId:       projectId,
		ProjectDomainId: projectDomainId,
		ResourceType:    ep.ResourceType,
		Action:          ep.Action,
	}
	EventNotify2(params)
}

func EventNotify2(params api.NotificationManagerEventNotifyInput) {
	t := eventTask{
		params: params,
	}
	notifyClientWorkerMan.Run(&t, nil, nil)
}

func EventNotifyServiceAbnormal(ctx context.Context, userCred mcclient.TokenCredential, service, method, path string, body jsonutils.JSONObject, err error) {
	event := api.Event.WithAction(api.ActionServiceAbnormal).WithResourceType(api.TOPIC_RESOURCE_SERVICE)
	obj := jsonutils.NewDict()
	if body != nil {
		obj.Set("body", jsonutils.NewString(body.PrettyString()))
	}
	obj.Set("method", jsonutils.NewString(method))
	obj.Set("path", jsonutils.NewString(path))
	obj.Set("error", jsonutils.NewString(err.Error()))
	obj.Set("service_name", jsonutils.NewString(service))
	params := api.NotificationManagerEventNotifyInput{
		ReceiverIds:     []string{userCred.GetUserId()},
		ResourceDetails: obj,
		Event:           event.String(),
		AdvanceDays:     0,
		Priority:        string(npk.NotifyPriorityNormal),
		ResourceType:    api.TOPIC_RESOURCE_SERVICE,
		Action:          api.ActionServiceAbnormal,
	}
	t := eventTask{
		params: params,
	}
	notifyClientWorkerMan.Run(&t, nil, nil)
}

func systemEventNotify(ctx context.Context, action api.SAction, resType string, result api.SResult, priority string, obj *jsonutils.JSONDict) {
	event := api.Event.WithAction(action).WithResourceType(resType).WithResult(result)
	params := api.NotificationManagerEventNotifyInput{
		ReceiverIds:     []string{},
		ResourceDetails: obj,
		Event:           event.String(),
		Priority:        priority,
	}
	EventNotify2(params)
}

func SystemEventNotify(ctx context.Context, action api.SAction, resType string, obj *jsonutils.JSONDict) {
	systemEventNotify(ctx, action, resType, api.ResultSucceed, string(npk.NotifyPriorityNormal), obj)
}

func SystemExceptionNotify(ctx context.Context, action api.SAction, resType string, obj *jsonutils.JSONDict) {
	systemEventNotify(ctx, action, resType, api.ResultSucceed, string(npk.NotifyPriorityCritical), obj)
}

func SystemExceptionNotifyWithResult(ctx context.Context, action api.SAction, resType string, result api.SResult, obj *jsonutils.JSONDict) {
	systemEventNotify(ctx, action, resType, result, string(npk.NotifyPriorityCritical), obj)
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

func FetchNotifyAdminRecipients(ctx context.Context, region string, users []string, groups []string) {
	s, err := AdminSessionGenerator(ctx, region)
	if err != nil {
		log.Errorf("unable to get admin session: %v", err)
	}

	notifyAdminUsers = make([]string, 0)
	for _, u := range users {
		uId, err := getIdentityId(s, u, &identity.UsersV3)
		if err != nil {
			log.Warningf("fetch user %s fail: %s", u, err)
		} else {
			notifyAdminUsers = append(notifyAdminUsers, uId)
		}
	}
	notifyAdminGroups = make([]string, 0)
	for _, g := range groups {
		gId, err := getIdentityId(s, g, &identity.Groups)
		if err != nil {
			log.Warningf("fetch group %s fail: %s", g, err)
		} else {
			notifyAdminGroups = append(notifyAdminGroups, gId)
		}
	}
}

func NotifyVmIntegrity(ctx context.Context, name string) {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(name), "name")
	SystemExceptionNotifyWithResult(ctx, api.ActionChecksumTest, api.TOPIC_RESOURCE_VM_INTEGRITY_CHECK, api.ResultFailed, data)
}
