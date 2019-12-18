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
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/notify/utils"
)

var workMan *appsrv.SWorkerManager

func init() {
	workMan = appsrv.NewWorkerManager("NotifyWokerManager", 16, 512, false)
}

func Send(notifications []*SNotification, userCred mcclient.TokenCredential, contacts []string) {

	for i := range notifications {
		workMan.Run(func() {
			sendone(context.Background(), userCred, notifications[i], contacts[i])
		}, nil, nil)
	}
}

func sendone(ctx context.Context, userCred mcclient.TokenCredential, notification *SNotification, contact string) {
	err := notification.SetSentAndTime(userCred)
	if err != nil {
		log.Errorf("Change notification's status failed.")
		return
	}
	err = NotifyService.Send(ctx, notification.ContactType, contact, notification.Topic, notification.Msg,
		notification.Priority)
	if err != nil {
		log.Errorf("Send notification failed: %s.", err.Error())
		notification.SetStatus(userCred, NOTIFY_FAIL, err.Error())
	} else {
		log.Debugf("send notification successfully")
		notification.SetStatus(userCred, NOTIFY_OK, "")
	}
}

func RestartService(config map[string]string, serviceName string) {
	workMan.Run(func() {
		NotifyService.RestartService(context.Background(), config, serviceName)
	}, nil, nil)
}

func SendVerifyMessage(ctx context.Context, userCred mcclient.TokenCredential, verify *SVerify,
	contact *SContact) error {
	var (
		err error
		msg string
	)
	processId, token := verify.ID, verify.Token
	if contact.ContactType == "email" {
		emailUrl := strings.Replace(TemplateManager.GetEmailUrl(), "{0}", processId, 1)
		emailUrl = strings.Replace(emailUrl, "{1}", token, 1)

		// get uName
		uName, err := utils.GetUsernameByID(ctx, contact.UID)
		if err != nil || len(uName) == 0 {
			uName = "用户"
		}
		data := struct {
			Name string
			Link string
		}{uName, emailUrl}
		msg = jsonutils.Marshal(data).String()
	} else if contact.ContactType == "mobile" {
		msg = fmt.Sprintf(`{"code": "%s"}`, token)
	} else {
		// todo
		return nil
	}

	err = NotifyService.Send(ctx, contact.ContactType, contact.Contact, "verify", msg, "")
	if err != nil {
		verify.SetStatus(userCred, VERIFICATION_SENT_FAIL, "")
		// set contact's status as "init"
		contact.SetStatus(userCred, CONTACT_INIT, "send verify message failed")
		log.Errorf("Send verify message failed: %s.", err.Error())
		return errors.Wrap(err, "Send Verify Message Failed")
	}
	verify.SetStatus(userCred, VERIFICATION_SENT, "")
	return nil
}

func UpdateDingtalk(uid string) {
	workMan.Run(func() {
		updateDingtalk(context.Background(), uid)
	}, nil, nil)
}

func updateDingtalk(ctx context.Context, uid string) {
	contacts, err := ContactManager.FetchByUIDAndCType(uid, []string{MOBILE, DINGTALK})
	if err != nil {
		log.Errorf("fetch contacts error")
	}
	if len(contacts) == 0 {
		return
	}
	var mobileContact, dingtalkContact *SContact
	for i := range contacts {
		if contacts[i].ContactType == MOBILE {
			mobileContact = &contacts[i]
		} else {
			dingtalkContact = &contacts[i]
		}
	}
	if mobileContact == nil {
		return
	}

	userid, err := NotifyService.ContactByMobile(ctx, mobileContact.Contact, DINGTALK)
	if err != nil {
		log.Errorf("fetch dingtalk userid by mobile failed: %s", err.Error())
	}
	if dingtalkContact != nil {
		dingtalkContact.SetModelManager(ContactManager, dingtalkContact)
		origin := dingtalkContact.Contact
		_, err := db.Update(dingtalkContact, func() error {
			dingtalkContact.Contact = userid
			return nil
		})
		if err != nil {
			log.Errorf("update dingtalk userid %s => %s failed", origin, userid)
		}
		return
	}

	contact := SContact{
		UID:         uid,
		ContactType: DINGTALK,
		Contact:     userid,
		Enabled:     "1",
		VerifiedAt:  time.Now(),
	}
	contact.Status = CONTACT_VERIFIED

	err = ContactManager.TableSpec().InsertOrUpdate(&contact)
	if err != nil {
		log.Errorf("create new dingtalk contact failed")
	}
}
