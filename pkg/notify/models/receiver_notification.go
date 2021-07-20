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

package models

import (
	"context"
	"net/http"
	"time"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/appsrv"
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
	Contact      string    `width:"128" nullable:"false" index:"true"`
	ReceiverType string    `width:"16"`
	SendAt       time.Time `nullable:"false"`
	SendBy       string    `width:"128" nullable:"false"`
	Status       string    `width:"36" charset:"ascii"`
	FailedReason string    `width:"1024"`
}

func (self *SReceiverNotificationManager) InitializeData() error {
	return dataCleaning(self.TableSpec().Name())
}

func (rnm *SReceiverNotificationManager) Create(ctx context.Context, userCred mcclient.TokenCredential, receiverID, notificationID string) (*SReceiverNotification, error) {
	rn := &SReceiverNotification{
		ReceiverID:     receiverID,
		NotificationID: notificationID,
		ReceiverType:   api.RECEIVER_TYPE_USER,
		Status:         api.RECEIVER_NOTIFICATION_RECEIVED,
		SendBy:         userCred.GetUserId(),
	}
	return rn, rnm.TableSpec().Insert(ctx, rn)
}

func (rnm *SReceiverNotificationManager) CreateRobot(ctx context.Context, userCred mcclient.TokenCredential, RobotID, notificationID string) (*SReceiverNotification, error) {
	rn := &SReceiverNotification{
		ReceiverID:     RobotID,
		NotificationID: notificationID,
		ReceiverType:   api.RECEIVER_TYPE_ROBOT,
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

func (rnm *SReceiverNotificationManager) CreateContact(ctx context.Context, userCred mcclient.TokenCredential, contact, notificationID string) (*SReceiverNotification, error) {
	rn := &SReceiverNotification{
		NotificationID: notificationID,
		ReceiverType:   api.RECEIVER_TYPE_CONTACT,
		Contact:        contact,
		Status:         api.RECEIVER_NOTIFICATION_RECEIVED,
		SendBy:         userCred.GetUserId(),
	}
	return rn, rnm.TableSpec().Insert(ctx, rn)
}

func (rnm *SReceiverNotificationManager) SetHandlerProcessTimeout(info *appsrv.SHandlerInfo, r *http.Request) time.Duration {
	if r.Method == http.MethodGet && len(r.URL.Query().Get("export_keys")) > 0 {
		return time.Hour * 2
	}
	return -time.Second
}

// func (rn *SReceiverNotification) Receiver() (*SReceiver, error) {
// 	q := ReceiverManager.Query().Equals("id", rn.ReceiverID)
// 	var receiver SReceiver
// 	err := q.First(&receiver)
// 	if err != nil {
// 		return nil, err
// 	}
// 	receiver.SetModelManager(ReceiverManager, &receiver)
// 	return &receiver, nil
// }

func (rn *SReceiverNotification) receiver() (*SReceiver, error) {
	q := ReceiverManager.Query().Equals("id", rn.ReceiverID)
	var receiver SReceiver
	err := q.First(&receiver)
	if err != nil {
		return nil, err
	}
	receiver.SetModelManager(ReceiverManager, &receiver)
	return &receiver, nil
}

func (rn *SReceiverNotification) robot() (*SRobot, error) {
	q := RobotManager.Query().Equals("id", rn.ReceiverID)
	var robot SRobot
	err := q.First(&robot)
	if err != nil {
		return nil, err
	}
	robot.SetModelManager(RobotManager, &robot)
	return &robot, nil
}

func (rn *SReceiverNotification) Receiver() (IReceiver, error) {
	switch rn.ReceiverType {
	case api.RECEIVER_TYPE_USER:
		return rn.receiver()
	case api.RECEIVER_TYPE_CONTACT:
		return &SContact{contact: rn.Contact}, nil
	case api.RECEIVER_TYPE_ROBOT:
		return rn.robot()
	default:
		// compatible
		if rn.ReceiverID != "" && rn.ReceiverID != ReceiverIdDefault {
			return rn.receiver()
		}
		return &SContact{contact: rn.Contact}, nil
	}
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

type IReceiver interface {
	IsEnabled() bool
	GetDomainId() string
	IsEnabledContactType(string) (bool, error)
	IsVerifiedContactType(string) (bool, error)
	GetContact(string) (string, error)
	GetTemplateLang(context.Context) (string, error)
}

type SReceiverBase struct {
}

func (s SReceiverBase) IsEnabled() bool {
	return true
}

func (s SReceiverBase) GetDomainId() string {
	return ""
}

func (s SReceiverBase) IsEnabledContactType(_ string) (bool, error) {
	return true, nil
}

func (s SReceiverBase) IsVerifiedContactType(_ string) (bool, error) {
	return true, nil
}

func (s SReceiverBase) GetTemplateLang(ctx context.Context) (string, error) {
	return "", nil
}

type SContact struct {
	SReceiverBase
	contact string
}

func (s *SContact) GetContact(_ string) (string, error) {
	return s.contact, nil
}
