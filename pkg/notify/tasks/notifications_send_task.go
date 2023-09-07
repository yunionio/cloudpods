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
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	apis "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/onecloud/pkg/notify/options"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type NotificationSendTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(NotificationSendTask{})
}

func (self *NotificationSendTask) taskFailed(ctx context.Context, notification *models.SNotification, reason string, all bool) {
	log.Errorf("fail to send notification %q", notification.GetId())
	if all {
		notification.SetStatus(self.UserCred, apis.NOTIFICATION_STATUS_FAILED, reason)
	} else {
		notification.SetStatus(self.UserCred, apis.NOTIFICATION_STATUS_PART_OK, reason)
	}
	notification.AddOne()
	logclient.AddActionLogWithContext(ctx, notification, logclient.ACT_SEND_NOTIFICATION, reason, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(reason))
}

type DomainContact struct {
	DomainId string
	Contact  string
}

type ReceiverSpec struct {
	receiver     models.IReceiver
	rNotificaion *models.SReceiverNotification
}

var notificationSendMap sync.Map
var notificationGroupLock sync.Mutex

func init() {
	notificationGroupLock = sync.Mutex{}
}
func (self *NotificationSendTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	notification := obj.(*models.SNotification)
	if notification.Status == apis.NOTIFICATION_STATUS_OK {
		self.SetStageComplete(ctx, nil)
		return
	}
	rns, err := notification.ReceiverNotificationsNotOK()
	if err != nil {
		self.taskFailed(ctx, notification, errors.Wrapf(err, "ReceiverNotificationsNotOK").Error(), true)
		return
	}
	event, err := models.EventManager.GetEvent(notification.EventId)
	if err != nil {
		if err != sql.ErrNoRows {
			self.taskFailed(ctx, notification, errors.Wrapf(err, "GetEvent").Error(), true)
			return
		}
	}
	notification.SetStatus(self.UserCred, apis.NOTIFICATION_STATUS_SENDING, "")

	// build contactMap
	receivers := make([]ReceiverSpec, 0)
	receiversEn := make([]ReceiverSpec, 0, len(rns)/2)
	receiversCn := make([]ReceiverSpec, 0, len(rns)/2)

	failedRecord := make([]string, 0)
	sendFail := func(rn *models.SReceiverNotification, reason string) {
		rn.AfterSend(ctx, false, reason)
		failedRecord = append(failedRecord, fmt.Sprintf("%s: %s", rn.ReceiverID, reason))
	}
	robotUseTemplate := false
	for i := range rns {
		receiver, err := rns[i].Receiver()
		if err != nil {
			sendFail(&rns[i], fmt.Sprintf("fail to fetch Receiver: %s", err.Error()))
			continue
		}
		// check receiver enabled
		if !receiver.IsEnabled() {
			sendFail(&rns[i], "disabled receiver")
			continue
		}
		// check contact enabled
		if notification.ContactType == apis.WEBHOOK {
			notification.ContactType = apis.WEBHOOK_ROBOT
		}
		if receiver.IsRobot() {
			robot := receiver.(*models.SRobot)
			notification.ContactType = fmt.Sprintf("%s-robot", robot.Type)
			robotUseTemplate = robot.UseTemplate.Bool()
		}
		enabled, err := receiver.IsEnabledContactType(notification.ContactType)
		if err != nil {
			logclient.AddSimpleActionLog(notification, logclient.ACT_SEND_NOTIFICATION, errors.Wrapf(err, "GetEnabledContactTypes"), self.GetUserCred(), false)
			continue
		}
		driver := models.GetDriver(notification.ContactType)
		if driver == nil || !enabled {
			sendFail(&rns[i], fmt.Sprintf("disabled contactType %q", notification.ContactType))
			continue
		}

		// check contact verified
		verified, err := receiver.IsVerifiedContactType(notification.ContactType)
		if err != nil {
			sendFail(&rns[i], fmt.Sprintf("IsVerifiedContactType error for receiver: %s", err.Error()))
			continue
		}
		if !verified {
			sendFail(&rns[i], fmt.Sprintf("unverified contactType %q", notification.ContactType))
			continue
		}
		lang, err := receiver.GetTemplateLang(ctx)
		if err != nil {
			reason := fmt.Sprintf("fail to GetTemplateLang: %s", err.Error())
			sendFail(&rns[i], reason)
			continue
		}
		switch lang {
		case "":
			receivers = append(receivers, ReceiverSpec{
				receiver:     receiver,
				rNotificaion: &rns[i],
			})
		case apis.TEMPLATE_LANG_EN:
			receiversEn = append(receiversEn, ReceiverSpec{
				receiver:     receiver,
				rNotificaion: &rns[i],
			})
		case apis.TEMPLATE_LANG_CN:
			receiversCn = append(receiversCn, ReceiverSpec{
				receiver:     receiver,
				rNotificaion: &rns[i],
			})
		}
	}

	nn, err := notification.Notification(robotUseTemplate)
	if err != nil {
		self.taskFailed(ctx, notification, errors.Wrapf(err, "Notification").Error(), true)
		return
	}
	for lang, receivers := range map[string][]ReceiverSpec{
		"":                    receivers,
		apis.TEMPLATE_LANG_CN: receiversCn,
		apis.TEMPLATE_LANG_EN: receiversEn,
	} {
		if len(receivers) == 0 {
			log.Warningf("no receiver to send, skip ...")
			continue
		}
		// send
		topicId := ""
		if event != nil {
			topicId = event.TopicId
		}
		p, err := notification.GetTemplate(ctx, topicId, lang, nn)
		if err != nil {
			logclient.AddSimpleActionLog(notification, logclient.ACT_SEND_NOTIFICATION, errors.Wrapf(err, "FillWithTemplate(%s)", lang), self.GetUserCred(), false)
			continue
		}
		if event != nil {
			p.Event = event.Event
		}
		if notification.ContactType != apis.MOBILE && notification.ContactType != apis.WEBHOOK_ROBOT {
			switch lang {
			case apis.TEMPLATE_LANG_CN:
				p.Message += "\n来自 " + options.Options.ApiServer
				tz, _ := time.LoadLocation(options.Options.TimeZone)
				p.Message += "\n发生于 " + time.Now().In(tz).Format("2006-01-02 15:04:05")
			case apis.TEMPLATE_LANG_EN:
				p.Message += "\nfrom " + options.Options.ApiServer
				p.Message += "\nat " + time.Now().In(time.UTC).Format("2006-01-02 15:04:05")
			}
		}

		p.DomainId = self.UserCred.GetDomainId()
		// set status before send
		now := time.Now()
		for _, rn := range receivers {
			rn.rNotificaion.BeforeSend(ctx, now)
		}
		// send
		fds, err := self.batchSend(ctx, notification, receivers, p)
		if err != nil {
			for _, r := range receivers {
				sendFail(r.rNotificaion, err.Error())
			}
			continue
		}
		// check result
		failedRnIds := make(map[int64]struct{}, 0)
		for _, fd := range fds {
			sendFail(fd.rNotificaion, fd.Reason)
			failedRnIds[fd.rNotificaion.RowId] = struct{}{}
		}
		// after send for successful notify
		for _, r := range receivers {
			if _, ok := failedRnIds[r.rNotificaion.RowId]; ok {
				continue
			}
			r.rNotificaion.AfterSend(ctx, true, "")
		}
	}
	if len(failedRecord) > 0 && len(failedRecord) >= len(rns) {
		self.taskFailed(ctx, notification, strings.Join(failedRecord, "; "), true)
		return
	}
	if len(failedRecord) > 0 {
		self.taskFailed(ctx, notification, strings.Join(failedRecord, "; "), false)
		return
	}
	notification.SetStatus(self.UserCred, apis.NOTIFICATION_STATUS_OK, "")
	logclient.AddActionLogWithContext(ctx, notification, logclient.ACT_SEND_NOTIFICATION, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

type FailedReceiverSpec struct {
	ReceiverSpec
	Reason string
}

func (notificationSendTask *NotificationSendTask) batchSend(ctx context.Context, notification *models.SNotification, receivers []ReceiverSpec, params apis.SendParams) (fails []FailedReceiverSpec, err error) {
	if notification.ContactType == apis.WEBCONSOLE {
		return
	}
	for i := range receivers {
		if receivers[i].receiver.IsRobot() {
			robot := receivers[i].receiver.(*models.SRobot)
			driver := models.GetDriver(fmt.Sprintf("%s-robot", robot.Type))
			params.Receivers.Contact = robot.Address
			params.Header = robot.Header
			params.Body = robot.Body
			params.MsgKey = robot.MsgKey
			params.GroupTimes = uint(receivers[i].rNotificaion.GroupTimes)
			err = driver.Send(ctx, params)
			if err != nil {
				fails = append(fails, FailedReceiverSpec{ReceiverSpec: receivers[i], Reason: err.Error()})
			}
		} else if receivers[i].receiver.IsReceiver() {
			receiver := receivers[i].receiver.(*models.SReceiver)
			params.Receivers.Contact, _ = receiver.GetContact(notification.ContactType)
			params.GroupTimes = uint(receivers[i].rNotificaion.GroupTimes)
			driver := models.GetDriver(notification.ContactType)
			if notification.ContactType == apis.EMAIL {
				params.EmailMsg = apis.SEmailMessage{
					To:      []string{receiver.Email},
					Subject: params.Title,
					Body:    params.Message,
				}
			}
			if notification.ContactType == apis.MOBILE {
				mobileArr := strings.Split(receiver.Mobile, " ")
				mobile := strings.Join(mobileArr, "")
				params.Receivers.Contact = mobile
			}
			params.ReceiverId = receiver.Id
			if len(params.GroupKey) > 0 && params.GroupTimes > 0 {
				notificationGroupLock.Lock()
				if _, ok := notificationSendMap.Load(params.GroupKey + receiver.Id + notification.ContactType); ok {
					err = models.NotificationGroupManager.TaskCreate(ctx, notification.ContactType, params)
				} else {
					err = models.NotificationGroupManager.TaskCreate(ctx, notification.ContactType, params)
					if err != nil {
						fails = append(fails, FailedReceiverSpec{ReceiverSpec: receivers[i], Reason: err.Error()})
					}
					err = driver.Send(ctx, params)
					createTimeTicker(ctx, driver, params, receiver.Id, notification.ContactType)
				}
				notificationGroupLock.Unlock()
			} else {
				err = driver.Send(ctx, params)
			}
			if err != nil {
				fails = append(fails, FailedReceiverSpec{ReceiverSpec: receivers[i], Reason: err.Error()})
			}
		} else {
			receiver := receivers[i].receiver.(*models.SContact)
			params.Receivers.Contact, _ = receiver.GetContact(notification.ContactType)
			driver := models.GetDriver(notification.ContactType)
			err = driver.Send(ctx, params)
			if err != nil {
				fails = append(fails, FailedReceiverSpec{ReceiverSpec: receivers[i], Reason: err.Error()})
			}
		}
	}
	return fails, nil
}

func createTimeTicker(ctx context.Context, driver models.ISenderDriver, params apis.SendParams, receiverId, contactType string) {
	// 创建一个计时器，每秒触发一次
	// params.GroupTimes = 5
	timer := time.NewTicker(time.Duration(params.GroupTimes) * time.Minute)
	notificationSendMap.Store(params.GroupKey+receiverId+contactType, apis.SNotificationGroupSearchInput{
		GroupKey:    params.GroupKey,
		ReceiverId:  receiverId,
		ContactType: contactType,
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(time.Duration(params.GroupTimes) * time.Minute),
	})

	// 启动一个goroutine来处理计时器触发的事件
	go func() {
		for {
			// 等待计时器触发的事件
			<-timer.C
			// 处理计时器触发的事件
			arrValue, ok := notificationSendMap.Load(params.GroupKey + receiverId + contactType)
			if !ok {
				return
			}
			input := arrValue.(apis.SNotificationGroupSearchInput)
			// 组装聚合后的消息
			sendParams, err := models.NotificationGroupManager.TaskSend(ctx, input)
			if err != nil {
				log.Errorln("TaskSend err:", err)
				return
			}
			driverT := models.GetDriver(contactType)
			driverT.Send(ctx, *sendParams)
			notificationSendMap.Delete(params.GroupKey + receiverId + contactType)
		}
	}()
}
