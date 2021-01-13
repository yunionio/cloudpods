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
	"yunion.io/x/onecloud/pkg/notify"
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

	// split rns
	var (
		rnsWithReceiver    []*models.SReceiverNotification
		rnsWithoutReceiver []*models.SReceiverNotification
	)

	for i := range rns {
		if rns[i].ReceiverID == models.ReceiverIdDefault {
			rnsWithoutReceiver = append(rnsWithoutReceiver, &rns[i])
		} else {
			rnsWithReceiver = append(rnsWithReceiver, &rns[i])
		}
	}

	failedRecord := make([]string, 0)
	sendFail := func(rn *models.SReceiverNotification, reason string) {
		rn.AfterSend(ctx, false, reason)
		failedRecord = append(failedRecord, fmt.Sprintf("%s: %s", rn.ReceiverID, reason))
	}

	// build contactMap
	contactMap := make(map[string]*models.SReceiverNotification)
	contactMapEn := make(map[string]*models.SReceiverNotification)
	contactmapCn := make(map[string]*models.SReceiverNotification)
	for i := range rnsWithReceiver {
		if len(rnsWithReceiver[i].ReceiverID) == 0 {
			contactMap[rnsWithReceiver[i].Contact] = rnsWithReceiver[i]
			continue
		}
		receiver, err := rnsWithReceiver[i].Receiver()
		if err != nil {
			sendFail(rnsWithReceiver[i], fmt.Sprintf("fail to fetch Receiver: %s", err.Error()))
			continue
		}
		// check receiver enabled
		if receiver.Enabled.IsFalse() {
			sendFail(rnsWithReceiver[i], fmt.Sprintf("disabled receiver"))
			continue
		}
		// check contact enabled
		enabled, err := receiver.IsEnabledContactType(notification.ContactType)
		if err != nil {
			sendFail(rnsWithReceiver[i], fmt.Sprintf("IsEnabledContactType error for receiver: %s", err.Error()))
			continue
		}
		if !enabled {
			sendFail(rnsWithReceiver[i], fmt.Sprintf("disabled contactType %q", notification.ContactType))
			continue
		}

		// check contact verified
		verified, err := receiver.IsVerifiedContactType(notification.ContactType)
		if err != nil {
			sendFail(rnsWithReceiver[i], fmt.Sprintf("IsVerifiedContactType error for receiver: %s", err.Error()))
			continue
		}
		if !verified {
			sendFail(rnsWithReceiver[i], fmt.Sprintf("unverified contactType %q", notification.ContactType))
			continue
		}

		contact, err := receiver.GetContact(notification.ContactType)
		if err != nil {
			reason := fmt.Sprintf("fail to fetch contact: %s", err.Error())
			sendFail(rnsWithReceiver[i], reason)
			continue
		}
		lang, err := receiver.GetTemplateLang(ctx)
		if err != nil {
			reason := fmt.Sprintf("fail to GetTemplateLang: %s", err.Error())
			sendFail(rnsWithReceiver[i], reason)
			continue
		}
		switch lang {
		case "":
			contactMap[contact] = rnsWithReceiver[i]
		case apis.TEMPLATE_LANG_EN:
			contactMapEn[contact] = rnsWithReceiver[i]
		case apis.TEMPLATE_LANG_CN:
			contactmapCn[contact] = rnsWithReceiver[i]
		}
	}

	for i := range rnsWithoutReceiver {
		contactMap[rnsWithoutReceiver[i].Contact] = rnsWithoutReceiver[i]
	}

	var contactLen int
	for lang, contactMap := range map[string]map[string]*models.SReceiverNotification{
		"":                    contactMap,
		apis.TEMPLATE_LANG_CN: contactmapCn,
		apis.TEMPLATE_LANG_EN: contactMapEn,
	} {
		if len(contactMap) == 0 {
			continue
		}
		// set status before send
		now := time.Now()
		contacts := make([]string, 0, len(contactMap))
		for c, rn := range contactMap {
			rn.BeforeSend(ctx, now)
			contacts = append(contacts, c)
		}

		contactLen += len(contacts)

		p := notify.SBatchSendParams{
			Contacts:    contacts,
			ContactType: notification.ContactType,
			Topic:       notification.Topic,
			Message:     notification.Message,
			Priority:    notification.Priority,
			Lang:        lang,
		}
		// send
		fds, err := models.NotifyService.BatchSend(ctx, p)
		if err != nil {
			for _, rn := range contactMap {
				sendFail(rn, err.Error())
			}
			continue
		}
		// check result
		for _, fd := range fds {
			rn := contactMap[fd.Contact]
			sendFail(rn, err.Error())
			delete(contactMap, fd.Contact)
		}
		// after send for successful notify
		for _, rn := range contactMap {
			rn.AfterSend(ctx, true, "")
		}
	}
	if len(failedRecord) > 0 && len(failedRecord) == contactLen {
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
