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

package tasks

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	identity "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/onecloud/pkg/notify/options"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type TopicMessageSendTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(TopicMessageSendTask{})
}

func (topicMessageSendTask *TopicMessageSendTask) taskFailed(ctx context.Context, topic *models.STopic, err error) {
	logclient.AddActionLogWithContext(ctx, topic, logclient.ACT_SEND_NOTIFICATION, err, topicMessageSendTask.UserCred, false)
	topicMessageSendTask.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (topicMessageSendTask *TopicMessageSendTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	failedReasons := []string{}
	topic := obj.(*models.STopic)
	input := api.NotificationManagerEventNotifyInput{}
	topicMessageSendTask.GetParams().Unmarshal(&input)

	message := jsonutils.Marshal(input.ResourceDetails).String()
	sevent := api.Event.WithAction(input.Action).WithResourceType(input.ResourceType)
	event, err := models.EventManager.CreateEvent(ctx, sevent.String(), topic.Id, message, string(input.Action), string(input.ResourceType), input.AdvanceDays)
	if err != nil {
		topicMessageSendTask.taskFailed(ctx, topic, errors.Wrap(err, "unable to create Event"))
		return
	}
	snotification := api.SsNotification{
		Topic:       topic.Id,
		Message:     event.Message,
		Event:       sevent,
		AdvanceDays: event.AdvanceDays,
	}
	n := models.SNotification{
		Priority:  input.Priority,
		EventId:   event.GetId(),
		TopicType: topic.Type,
	}
	n.Id = db.DefaultUUIDGenerator()
	for _, contact := range input.ContactTypes {
		n.ContactType = contact
		err = models.NotificationManager.TableSpec().Insert(ctx, &n)
		if err != nil {
			topicMessageSendTask.taskFailed(ctx, topic, errors.Wrap(err, "notifications insert"))
			return
		}
	}

	if err != nil {
		topicMessageSendTask.taskFailed(ctx, topic, errors.Wrap(err, "unable to fetch receivers by ids"))
		return
	}

	needWebconsole := false

	// 本地模板
	send, _ := models.LocalTemplateManager.FillWithTemplate(ctx, api.TEMPLATE_LANG_CN, snotification)
	// 远程模板（短信）
	remoteSend, _ := models.TemplateManager.FillWithTemplate(ctx, api.TEMPLATE_LANG_CN, snotification)
	send.Event = event.Event
	remoteSend.Event = event.Event
	websocketDriver := models.GetDriver(api.WEBSOCKET)
	err = websocketDriver.Send(send)
	if err != nil {
		topicMessageSendTask.taskFailed(ctx, topic, errors.Wrapf(err, "websocket send"))
	}
	scribers, err := topic.GetEnabledSubscribers(input.ProjectDomainId, input.ProjectId)
	if err != nil {
		topicMessageSendTask.taskFailed(ctx, topic, errors.Wrapf(err, "GetSubscribers"))
		return
	}

	robots := map[string]*models.SRobot{}
	userIds := []string{}
	for i := range scribers {
		switch scribers[i].Type {
		case api.SUBSCRIBER_TYPE_RECEIVER:
			recvs, err := scribers[i].GetEnabledReceivers()
			if err != nil {
				log.Errorf("scribers[%d].GetEnabledReceivers err :%s", i, err)
				return
			}
			// ids := []string{}
			err = sendByReceivers(recvs, []string{}, n, send, remoteSend, needWebconsole)
			if err != nil {
				topicMessageSendTask.taskFailed(ctx, topic, errors.Wrap(err, "sendByReceivers"))
				return
			}
		case api.SUBSCRIBER_TYPE_ROLE:
			query := jsonutils.NewDict()
			query.Set("roles", jsonutils.NewStringArray([]string{scribers[i].Identification}))
			query.Set("effective", jsonutils.JSONTrue)
			if scribers[i].RoleScope == api.SUBSCRIBER_SCOPE_DOMAIN {
				query.Set("project_domain_id", jsonutils.NewString(scribers[i].ResourceAttributionId))
			} else if scribers[i].RoleScope == api.SUBSCRIBER_SCOPE_PROJECT {
				query.Add(jsonutils.NewString(scribers[i].ResourceAttributionId), "scope", "project", "id")
			}
			s := auth.GetAdminSession(ctx, options.Options.Region)
			ret, err := identity.RoleAssignments.List(s, query)
			if err != nil {
				logclient.AddActionLogWithContext(ctx, topic, logclient.ACT_SEND_NOTIFICATION, errors.Wrapf(err, "RoleAssignments.List"), topicMessageSendTask.UserCred, false)
				continue
			}
			users := []struct {
				User struct {
					Id string
				}
			}{}
			jsonutils.Update(&users, ret.Data)
			for _, user := range users {
				userIds = append(userIds, user.User.Id)
			}
			recvs, err := models.ReceiverManager.FetchEnableReceiversByIdOrNames(ctx, userIds...)
			if err != nil {
				topicMessageSendTask.taskFailed(ctx, topic, errors.Wrap(err, "FetchEnableReceiversByIdOrNames"))
				return
			}
			err = sendByReceivers(recvs, []string{}, n, send, remoteSend, false)
			if err != nil {
				topicMessageSendTask.taskFailed(ctx, topic, errors.Wrap(err, "sendByReceivers"))
				return
			}
		case api.SUBSCRIBER_TYPE_ROBOT:
			robot, err := scribers[i].GetRobot()
			if err != nil {
				logclient.AddActionLogWithContext(ctx, topic, logclient.ACT_SEND_NOTIFICATION, errors.Wrapf(err, "GetRobot"), topicMessageSendTask.UserCred, false)
				continue
			}
			if !robot.Enabled.Bool() {
				continue
			}
			robots[robot.Id] = robot
			robotType := ""
			switch robot.Type {
			case api.FEISHU:
				robotType = api.FEISHU_ROBOT
			case api.DINGTALK:
				robotType = api.DINGTALK_ROBOT
			case api.WORKWX:
				robotType = api.WORKWX_ROBOT
			case api.WEBHOOK:
				robotType = api.WEBHOOK_ROBOT
			}
			n.ContactType = robotType
			err = models.NotificationManager.TableSpec().Insert(ctx, &n)
			if err != nil {
				failedReasons = append(failedReasons, err.Error())
			}
			session := auth.GetAdminSession(ctx, options.Options.Region)
			rn := models.SReceiverNotification{
				NotificationID: n.Id,
				ReceiverType:   api.RECEIVER_TYPE_ROBOT,
				ReceiverID:     robot.Id,
				Status:         api.RECEIVER_NOTIFICATION_RECEIVED,
				SendBy:         session.GetUserId(),
			}

			models.ReceiverNotificationManager.TableSpec().Insert(ctx, &rn)
			driver := models.GetDriver(robotType)
			if driver == nil {
				log.Errorln(robotType)
			}
			send.Receivers = api.SNotifyReceiver{Contact: robot.Address}
			err = driver.Send(send)
			if err != nil {
				log.Errorln("this is err:", err)
			}
		}
	}
	if len(failedReasons) > 0 {
		reason := strings.Join(failedReasons, "; ")
		topicMessageSendTask.taskFailed(ctx, topic, errors.Error(reason))
		return
	}
	logclient.AddActionLogWithContext(ctx, topic, logclient.ACT_SEND_NOTIFICATION, jsonutils.Marshal(input), topicMessageSendTask.UserCred, true)
	topicMessageSendTask.SetStageComplete(ctx, nil)
}

func sendByReceivers(recvs []models.SReceiver, receiverIds []string, n models.SNotification, send, remoteSend api.SendParams, needWebconsole bool) error {
	ctx := context.Background()
	failedReasons := []string{}
	ids := []string{}
	idFailedMap := make(map[string][]string)
	for i, recv := range recvs {
		// 检查是否发送email
		if recvs[i].EnabledEmail == tristate.True {
			if recvs[i].VerifiedEmail == tristate.True {
				if recvs[i].EnabledEmail == tristate.True {
					driver := models.GetDriver(api.EMAIL)
					send.Receivers.Contact = recvs[i].Email
					err := driver.Send(send)
					if err != nil {
						failedReasons = append(failedReasons, errors.Wrapf(err, "email send").Error())
					}
				}
			} else {
				log.Errorln(errors.Errorf("email has no verified: %s,receiver name: %s", recvs[i].Email, recvs[i].Name))
			}
		}
		// 检查是否发送短信
		if recvs[i].EnabledMobile == tristate.True {
			if recvs[i].VerifiedMobile == tristate.True {
				if recvs[i].EnabledMobile == tristate.True {
					driver := models.GetDriver(api.MOBILE)
					send.Receivers.Contact = recvs[i].Mobile
					err := driver.Send(remoteSend)
					if err != nil {
						failedReasons = append(failedReasons, errors.Wrapf(err, "mobile send").Error())
					}
				}
			} else {
				log.Errorln(errors.Errorf("sms has no verified: %s,receiver name: %s", recvs[i].Mobile, recvs[i].Name))
			}
		}
		idFailedMap[recv.Id] = append(idFailedMap[recv.Id], failedReasons...)
		ids = append(ids, recvs[i].Id)
	}
	session := auth.GetAdminSession(ctx, options.Options.Region)
	for _, receiverId := range receiverIds {
		rn := models.SReceiverNotification{
			NotificationID: n.Id,
			Status:         api.RECEIVER_NOTIFICATION_RECEIVED,
			SendBy:         session.GetUserId(),
		}
		if utils.IsInStringArray(receiverId, ids) {
			rn.ReceiverType = api.RECEIVER_TYPE_USER
			rn.ReceiverID = receiverId
		} else {
			rn.ReceiverType = api.RECEIVER_TYPE_CONTACT
			rn.Contact = receiverId
		}
		models.ReceiverNotificationManager.TableSpec().Insert(ctx, &rn)
	}
	rm := &models.SReceiverManager{}
	// 从subcontacts表中获取数据并发送
	subContactsMap, err := rm.FetchSubContacts(ids)
	if err != nil {
		return errors.Wrap(err, "rm.FetchSubContacts")
	}
	for id, subContacts := range subContactsMap {
		for _, subContact := range subContacts {
			n.Topic = send.Topic
			n.Status = api.NOTIFICATION_STATUS_SENDING
			if subContact.Enabled == tristate.False {
				continue
			}
			n.ContactType = subContact.Type
			models.NotificationManager.TableSpec().Insert(context.Background(), &n)
			driver := models.GetDriver(subContact.Type)
			send.Receivers = api.SNotifyReceiver{
				Contact: subContact.Contact,
			}
			err = driver.Send(send)
			if err != nil {
				failedReasons = append(failedReasons, errors.Wrapf(err, "content type:%s,receiver:%s", subContact.Type, subContact.ReceiverID).Error())
			}
		}
		if len(idFailedMap[id]) == 0 {

		}
	}
	return nil
}

func createReceiverNotification(receiverIds []string, recvs []models.SReceiver, n models.SNotification) {

}
