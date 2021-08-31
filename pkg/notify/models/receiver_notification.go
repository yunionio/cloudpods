package models

import (
	"context"
	"time"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var ReceiverNotificationManager *SReceiverNotificationManager

func init() {
	db.InitManager(func() {
		ReceiverNotificationManager = &SReceiverNotificationManager{
			SJointResourceBaseManager: db.NewJointResourceBaseManager(
				SReceiverNotification{},
				"receivernotification_tbl",
				"receivernotification",
				"receivernotifications",
				NotificationManager,
				ReceiverManager,
			),
		}
		ReceiverNotificationManager.SetVirtualObject(ReceiverNotificationManager)
	})
}

type SReceiverNotificationManager struct {
	db.SJointResourceBaseManager
}

const (
	ReceiverIdDefault = "default"
)

// +onecloud:swagger-gen-ignore
type SReceiverNotification struct {
	db.SJointResourceBase

	ReceiverID     string `width:"128" charset:"ascii" nullable:"false" index:"true"`
	NotificationID string `width:"128" charset:"ascii" nullable:"false" index:"true"`
	// ignore if ReceiverID is not empty or default
	Contact      string `width:"128" index:"true"`
	SendAt       time.Time
	SendBy       string `width:"128"`
	Status       string `width:"36" charset:"ascii"`
	FailedReason string `width:"1024"`
}

func (self *SReceiverNotificationManager) InitializeData() error {
	return dataCleaning(self.TableSpec().Name())
}

func (rnm *SReceiverNotificationManager) Create(ctx context.Context, userCred mcclient.TokenCredential, receiverID, notificationID string) (*SReceiverNotification, error) {
	rn := &SReceiverNotification{
		ReceiverID:     receiverID,
		NotificationID: notificationID,
		Status:         api.RECEIVER_NOTIFICATION_RECEIVED,
		SendBy:         userCred.GetUserId(),
	}
	return rn, rnm.TableSpec().Insert(ctx, rn)
}

func (rnm *SReceiverNotificationManager) GetMasterFieldName() string {
	return "notification_id"
}

func (rnm *SReceiverNotificationManager) GetSlaveFieldName() string {
	return "receiver_id"
}

func (rnm *SReceiverNotificationManager) CreateWithoutReceiver(ctx context.Context, userCred mcclient.TokenCredential, contact, notificationID string) (*SReceiverNotification, error) {
	rn := &SReceiverNotification{
		NotificationID: notificationID,
		ReceiverID:     ReceiverIdDefault,
		Contact:        contact,
		Status:         api.RECEIVER_NOTIFICATION_RECEIVED,
		SendBy:         userCred.GetUserId(),
	}
	return rn, rnm.TableSpec().Insert(ctx, rn)
}

func (rn *SReceiverNotification) Receiver() (*SReceiver, error) {
	q := ReceiverManager.Query().Equals("id", rn.ReceiverID)
	var receiver SReceiver
	err := q.First(&receiver)
	if err != nil {
		return nil, err
	}
	receiver.SetModelManager(ReceiverManager, &receiver)
	return &receiver, nil
}

func (rn *SReceiverNotification) BeforeSend(ctx context.Context, sendTime time.Time) error {
	if sendTime.IsZero() {
		sendTime = time.Now()
	}
	_, err := db.Update(rn, func() error {
		rn.SendAt = sendTime
		rn.Status = api.RECEIVER_NOTIFICATION_SENT
		return nil
	})
	return err
}

func (rn *SReceiverNotification) AfterSend(ctx context.Context, success bool, reason string) error {
	_, err := db.Update(rn, func() error {
		if success {
			rn.Status = api.RECEIVER_NOTIFICATION_OK
		} else {
			rn.Status = api.RECEIVER_NOTIFICATION_FAIL
			rn.FailedReason = reason
		}
		return nil
	})
	return err
}
