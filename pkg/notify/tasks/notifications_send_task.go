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
	"fmt"
	"strings"
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
		if !strings.Contains(err.Error(), "no rows in result set") {
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

	for i := range rns {
		receiver, err := rns[i].Receiver()
		if err != nil {
			sendFail(&rns[i], fmt.Sprintf("fail to fetch Receiver: %s", err.Error()))
			continue
		}
		// check receiver enabled
		if !receiver.IsEnabled() {
			sendFail(&rns[i], fmt.Sprintf("disabled receiver"))
			continue
		}
		// check contact enabled
		enabled, err := receiver.IsEnabledContactType(notification.ContactType)
		if err != nil {
			logclient.AddSimpleActionLog(notification, logclient.ACT_SEND_NOTIFICATION, errors.Wrapf(err, "GetEnabledContactTypes"), self.GetUserCred(), false)
			continue
		}
		if notification.ContactType == apis.WEBHOOK {
			notification.ContactType = apis.WEBHOOK_ROBOT
		}
		if receiver.IsRobot() {
			robot := receiver.(*models.SRobot)
			notification.ContactType = fmt.Sprintf("%s-robot", robot.Type)
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

		// contact, err := receiver.GetContact(notification.ContactType)
		// if err != nil {
		// 	logclient.AddSimpleActionLog(notification, logclient.ACT_SEND_NOTIFICATION, errors.Wrapf(err, "GetContact(%s)", notification.ContactType), self.GetUserCred(), false)
		// 	continue
		// }

		// notifyRecv := receiver.GetNotifyReceiver()
		// notifyRecv.Lang, _ = receiver.GetTemplateLang(ctx)
		// notifyRecv.Contact = contact
		// cv := ReceiverSpec{
		// 	receiver:     notifyRecv,
		// 	rNotificaion: &rns[i],
		// }
		// switch notifyRecv.Lang {
		// case "":
		// 	receivers = append(receivers, cv)
		// case apis.TEMPLATE_LANG_EN:
		// 	receiversEn = append(receiversEn, cv)
		// case apis.TEMPLATE_LANG_CN:
		// 	receiversCn = append(receiversCn, cv)
		// }
		// lang, err := receiver.GetTemplateLang(ctx)
		// if err != nil {
		// 	reason := fmt.Sprintf("fail to GetTemplateLang: %s", err.Error())
		// 	sendFail(&rns[i], reason)
		// 	continue
		// }

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

	nn, err := notification.Notification()
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
		p, err := notification.GetTemplate(ctx, event.TopicId, lang, nn)
		if err != nil {
			logclient.AddSimpleActionLog(notification, logclient.ACT_SEND_NOTIFICATION, errors.Wrapf(err, "FillWithTemplate(%s)", lang), self.GetUserCred(), false)
			continue
		}
		if event != nil {
			p.Event = event.Event
		}
		if notification.ContactType != apis.MOBILE {
			switch lang {
			case apis.TEMPLATE_LANG_CN:
				p.Message += "\n来自 " + options.Options.ApiServer
			case apis.TEMPLATE_LANG_EN:
				p.Message += "\nfrom " + options.Options.ApiServer
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
	for i := range receivers {
		if receivers[i].receiver.IsRobot() {
			robot := receivers[i].receiver.(*models.SRobot)
			driver := models.GetDriver(fmt.Sprintf("%s-robot", robot.Type))
			params.Receivers.Contact = robot.Address
			err = driver.Send(params)
			if err != nil {
				fails = append(fails, FailedReceiverSpec{ReceiverSpec: receivers[i], Reason: err.Error()})
			}
		} else if receivers[i].receiver.IsReceiver() {
			receiver := receivers[i].receiver.(*models.SReceiver)
			params.Receivers.Contact, _ = receiver.GetContact(notification.ContactType)
			driver := models.GetDriver(notification.ContactType)
			if notification.ContactType == apis.EMAIL {
				params.EmailMsg = &apis.SEmailMessage{
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
			err = driver.Send(params)
			if err != nil {
				fails = append(fails, FailedReceiverSpec{ReceiverSpec: receivers[i], Reason: err.Error()})
			}
		} else {
			receiver := receivers[i].receiver.(*models.SContact)
			params.Receivers.Contact, _ = receiver.GetContact(notification.ContactType)
			driver := models.GetDriver(notification.ContactType)
			err = driver.Send(params)
			if err != nil {
				fails = append(fails, FailedReceiverSpec{ReceiverSpec: receivers[i], Reason: err.Error()})
			}
		}
	}

	return fails, nil
}
