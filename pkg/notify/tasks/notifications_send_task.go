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

	// sort out what needs to be sent
	failedRecord := make([]string, 0)
	sendFail := func(rn *models.SReceiverNotification, reason string) {
		rn.AfterSend(ctx, false, reason)
		failedRecord = append(failedRecord, fmt.Sprintf("%s: %s", rn.ReceiverID, reason))
	}

	// build contactMap
	contactMap := make(map[string]*models.SReceiverNotification)
	for i := range rns {
		if len(rns[i].ReceiverID) == 0 {
			contactMap[rns[i].Contact] = &rns[i]
			continue
		}
		receiver, err := rns[i].Receiver()
		if err != nil {
			sendFail(&rns[i], fmt.Sprintf("fail to fetch Receiver: %s", err.Error()))
			continue
		}
		// check receiver enabled
		if receiver.Enabled.IsFalse() {
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

		contact, err := receiver.GetContact(notification.ContactType)
		if err != nil {
			reason := fmt.Sprintf("fail to fetch contact: %s", err.Error())
			sendFail(&rns[i], reason)
			continue
		}
		contactMap[contact] = &rns[i]
	}

	if len(contactMap) == 0 {
		self.taskFailed(ctx, notification, strings.Join(failedRecord, "; "), true)
	}

	// set status before send
	now := time.Now()
	contacts := make([]string, 0, len(contactMap))
	for c, rn := range contactMap {
		rn.BeforeSend(ctx, now)
		contacts = append(contacts, c)
	}

	// send
	ret, err := models.NotifyService.BatchSend(ctx, contacts, notification.ContactType, notification.Topic, notification.Message, notification.Priority)
	if err != nil {
		for _, rn := range contactMap {
			rn.AfterSend(ctx, false, err.Error())
		}
		failedRecord = append(failedRecord, fmt.Sprintf("others: %s", err.Error()))
		self.taskFailed(ctx, notification, strings.Join(failedRecord, "; "), true)
		return
	}

	// check result
	for _, fd := range ret {
		rn := contactMap[fd.Contact]
		rn.AfterSend(ctx, false, fd.Reason)
		failedRecord = append(failedRecord, fmt.Sprintf("%s: %s", rn.ReceiverID, fd.Reason))
	}
	if len(failedRecord) == len(contacts) {
		self.taskFailed(ctx, notification, strings.Join(failedRecord, "; "), true)
		return
	}
	if len(failedRecord) > 0 {
		self.taskFailed(ctx, notification, strings.Join(failedRecord, "; "), false)
		return
	}
	log.Infof("successfully send notification %q", notification.GetId())
	notification.SetStatus(self.UserCred, apis.NOTIFICATION_STATUS_OK, "")
	logclient.AddActionLogWithContext(ctx, notification, logclient.ACT_SEND_NOTIFICATION, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
