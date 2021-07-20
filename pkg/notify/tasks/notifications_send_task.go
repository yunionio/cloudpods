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

	apis "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/notify/models"
	rpcapi "yunion.io/x/onecloud/pkg/notify/rpc/apis"
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
		self.taskFailed(ctx, notification, "fail to fetch ReceiverNotifications", true)
		return
	}
	notification.SetStatus(self.UserCred, apis.NOTIFICATION_STATUS_SENDING, "")

	failedRecord := make([]string, 0)
	sendFail := func(rn *models.SReceiverNotification, reason string) {
		rn.AfterSend(ctx, false, reason)
		failedRecord = append(failedRecord, fmt.Sprintf("%s: %s", rn.ReceiverID, reason))
	}

	// build contactMap
	receivers := make([]ReceiverSpec, 0, 0)
	receiversEn := make([]ReceiverSpec, 0, len(rns)/2)
	receiversCn := make([]ReceiverSpec, 0, len(rns)/2)
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
			sendFail(&rns[i], fmt.Sprintf("IsEnabledContactType error for receiver: %s", err.Error()))
			continue
		}
		if !enabled {
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
		// 	reason := fmt.Sprintf("fail to fetch contact: %s", err.Error())
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

	var contactLen int
	for lang, receivers := range map[string][]ReceiverSpec{
		"":                    receivers,
		apis.TEMPLATE_LANG_CN: receiversCn,
		apis.TEMPLATE_LANG_EN: receiversEn,
	} {
		if len(receivers) == 0 {
			continue
		}

		// send
		nn, err := notification.Notification()
		if err != nil {
			self.taskFailed(ctx, notification, err.Error(), false)
		}
		p, err := notification.TemplateStore().FillWithTemplate(ctx, lang, nn)
		if err != nil {
			self.taskFailed(ctx, notification, err.Error(), false)
		}
		// set status before send
		now := time.Now()
		for _, rn := range receivers {
			rn.rNotificaion.BeforeSend(ctx, now)
		}

		contactLen += len(receivers)

		// send
		fds, err := self.batchSend(ctx, notification.ContactType, receivers, p)
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

func (self *NotificationSendTask) batchSend(ctx context.Context, contactType string, receivers []ReceiverSpec, params rpcapi.SendParams) (fails []FailedReceiverSpec, err error) {
	log.Infof("contactType: %s, receivers: %s, params: %s", contactType, receivers, jsonutils.Marshal(params))
	if contactType != apis.ROBOT && contactType != apis.WEBHOOK {
		return self._batchSend(ctx, contactType, receivers, func(res []*rpcapi.SReceiver) ([]*rpcapi.FailedRecord, error) {
			return models.NotifyService.BatchSend(ctx, contactType, rpcapi.BatchSendParams{
				Receivers:      res,
				Title:          params.Title,
				Message:        params.Message,
				Priority:       params.Priority,
				RemoteTemplate: params.RemoteTemplate,
			})
		})
	}
	robots := make(map[string][]ReceiverSpec)
	for i := range receivers {
		robot := receivers[i].receiver.(*models.SRobot)
		robots[robot.Type] = append(robots[robot.Type], receivers[i])
	}
	for rType, robots := range robots {
		_fails, err := self._batchSend(ctx, contactType, robots, func(res []*rpcapi.SReceiver) ([]*rpcapi.FailedRecord, error) {
			return models.NotifyService.SendRobotMessage(ctx, rType, res, params.Title, params.Message)
		})
		if err != nil {
			for i := range robots {
				fails = append(fails, FailedReceiverSpec{
					ReceiverSpec: robots[i],
					Reason:       err.Error(),
				})
			}
		}
		fails = append(fails, _fails...)
	}
	return fails, nil
}

func (self *NotificationSendTask) _batchSend(ctx context.Context, contactType string, receivers []ReceiverSpec, send func([]*rpcapi.SReceiver) ([]*rpcapi.FailedRecord, error)) (fails []FailedReceiverSpec, err error) {
	rpcReceivers := make([]*rpcapi.SReceiver, len(receivers))
	rpc2Receiver := make(map[DomainContact]ReceiverSpec, len(receivers))
	for i := range receivers {
		contact, err := receivers[i].receiver.GetContact(contactType)
		if err != nil {
			fails = append(fails, FailedReceiverSpec{
				ReceiverSpec: receivers[i],
				Reason:       fmt.Sprintf("fail to fetch contact: %s", err.Error()),
			})
			continue
		}
		rpcReceivers[i] = &rpcapi.SReceiver{
			DomainId: receivers[i].receiver.GetDomainId(),
			Contact:  contact,
		}
		rpc2Receiver[DomainContact{
			DomainId: receivers[i].receiver.GetDomainId(),
			Contact:  contact,
		}] = receivers[i]
	}
	fds, err := send(rpcReceivers)
	if err != nil {
		return nil, err
	}
	// check result
	for _, fd := range fds {
		dc := DomainContact{
			DomainId: fd.Receiver.DomainId,
			Contact:  fd.Receiver.Contact,
		}
		receiver := rpc2Receiver[dc]
		fails = append(fails, FailedReceiverSpec{
			ReceiverSpec: receiver,
			Reason:       fd.Reason,
		})
	}
	return
}
