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
	"time"

	"github.com/pkg/errors"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	apis "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type NotificationSendTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(NotificationSendTask{})
}

func (self *NotificationSendTask) taskFailed(ctx context.Context, notification *models.SNotification, err error) {
	notification.SetStatus(self.UserCred, apis.NOTIFICATION_STATUS_FAILED, err.Error())
	notification.AddOne()
	logclient.AddActionLogWithContext(ctx, notification, logclient.ACT_SEND_NOTIFICATION, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *NotificationSendTask) taskComplete(ctx context.Context, notification *models.SNotification) {
	self.SetStageComplete(ctx, nil)
}

type DomainContact struct {
	DomainId string
	Contact  string
}

type ReceiverSpec struct {
	receiver     api.SNotifyReceiver
	rNotificaion *models.SReceiverNotification
}

func (self *NotificationSendTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	notification := obj.(*models.SNotification)
	if notification.Status == apis.NOTIFICATION_STATUS_OK {
		self.taskComplete(ctx, notification)
		return
	}
	rns, err := notification.ReceiverNotificationsNotOK()
	if err != nil {
		self.taskFailed(ctx, notification, errors.Wrapf(err, "ReceiverNotificationsNotOK"))
		return
	}
	notification.SetStatus(self.UserCred, apis.NOTIFICATION_STATUS_SENDING, "")

	_receivers, err := notification.GetNotOKReceivers()
	if err != nil {
		self.taskFailed(ctx, notification, errors.Wrapf(err, "GetNotOKReceivers"))
		return
	}

	// build contactMap
	receivers := make([]ReceiverSpec, 0, 0)
	receiversEn := make([]ReceiverSpec, 0, len(rns)/2)
	receiversCn := make([]ReceiverSpec, 0, len(rns)/2)
	for i := range _receivers {
		receiver := receivers[i]
		// check contact enabled
		enabled, err := receiver.GetEnabledContactTypes()
		if err != nil {
			logclient.AddSimpleActionLog(notification, logclient.ACT_SEND_NOTIFICATION, errors.Wrapf(err, "GetEnabledContactTypes"), self.GetUserCred(), false)
			continue
		}
		driver := models.GetDriver(notification.ContactType)
		if !driver.IsSystemConfigContactType() && !utils.IsInStringArray(notification.ContactType, enabled) {
			continue
		}
		// check contact verified
		verified, err := receiver.GetVerifiedContactTypes()
		if err != nil {
			logclient.AddSimpleActionLog(notification, logclient.ACT_SEND_NOTIFICATION, errors.Wrapf(err, "GetVerifiedContactTypes"), self.GetUserCred(), false)
			continue
		}
		if !utils.IsInStringArray(notification.ContactType, verified) {
			continue
		}

		contact, err := receiver.GetContact(notification.ContactType)
		if err != nil {
			logclient.AddSimpleActionLog(notification, logclient.ACT_SEND_NOTIFICATION, errors.Wrapf(err, "GetContact(%s)", notification.ContactType), self.GetUserCred(), false)
			continue
		}
		notifyRecv := receiver.GetNotifyReceiver()
		notifyRecv.Lang, _ = receiver.GetTemplateLang(ctx)
		notifyRecv.Contact = contact
		cv := ReceiverSpec{
			receiver:     notifyRecv,
			rNotificaion: &rns[i],
		}
		switch notifyRecv.Lang {
		case "":
			receivers = append(receivers, cv)
		case apis.TEMPLATE_LANG_EN:
			receiversEn = append(receiversEn, cv)
		case apis.TEMPLATE_LANG_CN:
			receiversCn = append(receiversCn, cv)
		}
	}

	nn, err := notification.Notification()
	if err != nil {
		self.taskFailed(ctx, notification, errors.Wrapf(err, "Notification"))
		return
	}

	for lang, receivers := range map[string][]ReceiverSpec{
		"":                    receivers,
		apis.TEMPLATE_LANG_CN: receiversCn,
		apis.TEMPLATE_LANG_EN: receiversEn,
	} {
		// send
		p, err := notification.FillWithTemplate(ctx, lang, nn)
		if err != nil {
			logclient.AddSimpleActionLog(notification, logclient.ACT_SEND_NOTIFICATION, errors.Wrapf(err, "FillWithTemplate(%s)", lang), self.GetUserCred(), false)
			continue
		}
		// set status before send
		now := time.Now()
		for _, rn := range receivers {
			rn.rNotificaion.BeforeSend(ctx, now)
		}

		// send
		self.batchSend(ctx, notification, receivers, p)
	}
	notification.SetStatus(self.UserCred, apis.NOTIFICATION_STATUS_OK, "")
	logclient.AddActionLogWithContext(ctx, notification, logclient.ACT_SEND_NOTIFICATION, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

type FailedReceiverSpec struct {
	ReceiverSpec
	Reason string
}

func (self *NotificationSendTask) batchSend(ctx context.Context, notification *models.SNotification, receivers []ReceiverSpec, params apis.SendParams) (fails []FailedReceiverSpec, err error) {
	//driver := models.GetDriver(notification.ContactType)
	/*
		log.Debugf("contactType: %s, receivers: %s, params: %s", contactType, jsonutils.Marshal(receivers), jsonutils.Marshal(params))
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
	*/
	return fails, nil
}

/*
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
*/
