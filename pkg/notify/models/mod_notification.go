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
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/notify/utils"
)

type SNotificationManager struct {
	SStatusStandaloneResourceBaseManager
}

var NotificationManager *SNotificationManager

func init() {
	NotificationManager = &SNotificationManager{
		SStatusStandaloneResourceBaseManager: NewStatusStandaloneResourceBaseManager(
			SNotification{},
			"notify_t_notification",
			"notification",
			"notifications",
		),
	}
	NotificationManager.SetVirtualObject(NotificationManager)
}

type SNotification struct {
	SStatusStandaloneResourceBase

	UID         string    `width:"128" nullable:"false" create:"required"`
	ContactType string    `width:"16" nullable:"false" create:"required"`
	Topic       string    `width:"128" nullable:"false" create:"optional"`
	Priority    string    `width:"16" nullable:"false" create:"optional"`
	Msg         string    `create:"required"`
	ReceivedAt  time.Time `nullable:"false"`
	SendAt      time.Time `nullable:"false"`
	SendBy      string    `width:"128" nullable:"false"`
}

func (self *SNotificationManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SNotificationManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SNotificationManager) InitializeData() error {
	sql := fmt.Sprintf("update %s set updated_at=update_at, deleted=is_deleted", self.TableSpec().Name())
	q := sqlchemy.NewRawQuery(sql, "")
	q.Row()
	return nil
}

func (self *SNotificationManager) BatchCreate(ctx context.Context, data jsonutils.JSONObject, contacts []SContact) ([]string, error) {
	userCred := policy.FetchUserCredential(ctx)
	ownerID, err := utils.FetchOwnerId(ctx, NotificationManager, userCred, jsonutils.JSONNull)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	msg, _ := data.GetString("msg")
	priority, _ := data.GetString("priority")
	topic, _ := data.GetString("topic")
	createFailed, createSuccess, contactSuccess := make([]string, 0), make([]*SNotification, 0, len(contacts)/2), make([]string, 0, len(contacts)/2)

	for i := range contacts {
		createData := map[string]string{
			"uid":          contacts[i].ID,
			"contact_type": contacts[i].ContactType,
			"topic":        topic,
			"priority":     priority,
			"msg":          msg,
			"send_by":      userCred.GetUserId(),
			"status":       NOTIFY_UNSENT,
		}
		model, err := db.DoCreate(self, ctx, userCred, jsonutils.JSONNull, jsonutils.Marshal(createData), ownerID)
		if err != nil {
			createFailed = append(createFailed, contacts[i].ID)
		} else {
			createSuccess = append(createSuccess, model.(*SNotification))
			contactSuccess = append(contactSuccess, contacts[i].Contact)
		}
	}
	go send(createSuccess, userCred, contactSuccess)
	if len(createFailed) != 0 {
		errInfo := new(bytes.Buffer)
		errInfo.WriteString("notifications whose uid are ")
		for i := range createFailed {
			errInfo.WriteString(createFailed[i])
			errInfo.WriteString(", ")
		}
		errInfo.Truncate(errInfo.Len() - 2)
		errInfo.WriteString("created failed.")
		log.Errorf(errInfo.String())
		return nil, errors.Error("Not all notifications were sent successfully")
	}
	notificationIDs := make([]string, len(createSuccess))
	for i := range createSuccess {
		notificationIDs[i] = createSuccess[i].ID
	}
	return notificationIDs, nil
}

func (self *SNotificationManager) FetchNotOK(lastTime time.Time) ([]SNotification, error) {
	q := self.Query()
	q.Filter(sqlchemy.AND(sqlchemy.GE(q.Field("created_at"), lastTime), sqlchemy.NotEquals(q.Field("status"), NOTIFY_UNSENT)))
	records := make([]SNotification, 0, 10)
	err := db.FetchModelObjects(self, q, &records)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func send(notifications []*SNotification, userCred mcclient.TokenCredential, contacts []string) {
	var wg sync.WaitGroup
	sendone := func(notification *SNotification, contact string) {
		err := notification.SetSentAndTime(userCred)
		if err != nil {
			log.Errorf("Change notification's status failed.")
			return
		}
		err = RpcService.Send(notification.ContactType, contact, notification.Topic, notification.Msg, notification.Priority)
		if err != nil {
			log.Errorf("Send notification failed because that %s.", err.Error())
			notification.SetStatus(userCred, NOTIFY_FAIL, err.Error())
		} else {
			notification.SetStatus(userCred, NOTIFY_OK, "")
		}
		wg.Done()
	}
	for i := range notifications {
		wg.Add(1)
		go sendone(notifications[i], contacts[i])
	}
	wg.Wait()
}

func (self *SNotification) SetSentAndTime(userCred mcclient.TokenCredential) error {
	status := NOTIFY_SENT
	if self.Status == status {
		return nil
	}
	oldStatus := self.Status
	_, err := db.Update(self, func() error {
		self.Status = status
		self.SendAt = time.Now()
		return nil
	})
	if err != nil {
		return err
	}
	reason := "sent notification"
	if userCred != nil {
		notes := fmt.Sprintf("%s=>%s", oldStatus, status)
		if len(reason) > 0 {
			notes = fmt.Sprintf("%s: %s", notes, reason)
		}
		db.OpsLog.LogEvent(self, db.ACT_UPDATE_STATUS, notes, userCred)
	}
	return nil
}

func (self *SNotification) SetStatusWithoutUserCred(status string) error {
	_, err := db.Update(self, func() error {
		self.Status = status
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
func sendWithoutUserCred(notifications []SNotification) {
	var wg sync.WaitGroup
	sendone := func(notification SNotification) {
		// Get contact
		contact, err := ContactManager.FetchByUIDAndCType(notification.UID, []string{notification.ContactType})
		if err != nil {
			return
		}
		if len(contact) == 0 {
			return
		}
		// sent_at update todo
		notification.SetStatusWithoutUserCred(NOTIFY_SENT)
		err = RpcService.Send(notification.ContactType, contact[0].Contact, notification.Topic, notification.Msg, notification.Priority)
		if err == nil {
			return
		}
		if err != nil {
			log.Errorf("Send notification failed because that %s.", err.Error())
			notification.SetStatusWithoutUserCred(NOTIFY_FAIL)
		} else {
			notification.SetStatusWithoutUserCred(NOTIFY_OK)
		}
		wg.Done()
	}
	for i := range notifications {
		wg.Add(1)
		go sendone(notifications[i])
	}
	wg.Wait()
}

func ReSend(minutes int) {
	scope := time.Duration(minutes) * time.Minute
	for {
		select {
		case <-time.After(scope / 2):
			//lastTime := time.Now().Add(-scope)
			//q := NotificationManager.Query()
			notifications, err := NotificationManager.FetchNotOK(time.Now().Add(-scope))
			if err != nil {
				break
			}
			sendWithoutUserCred(notifications)
		}
	}
}
