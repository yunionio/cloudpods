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

package models // import "yunion.io/x/onecloud/pkg/notify/models"

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	apis "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SNotificationGroupManager struct {
	db.SModelBaseManager
}

var NotificationGroupManager *SNotificationGroupManager

func init() {
	NotificationGroupManager = &SNotificationGroupManager{
		SModelBaseManager: db.NewModelBaseManager(
			SNotificationGroup{},
			"notification_groups_tbl",
			"notification_group",
			"notification_groups",
		),
	}
	NotificationGroupManager.SetVirtualObject(NotificationGroupManager)
}

// 站内信
type SNotificationGroup struct {
	db.SModelBase
	Id          string `width:"128" charset:"ascii" primary:"true" list:"user" create:"optional" json:"id"`
	GroupKey    string `width:"128" nullable:"false" create:"required" list:"user" get:"user"`
	Title       string
	Message     string
	ReceiverId  string `width:"128" nullable:"false" create:"required" list:"user" get:"user"`
	Body        jsonutils.JSONObject
	Header      jsonutils.JSONObject
	MsgKey      string
	ContactType string `width:"32" nullable:"false" create:"required" list:"user" get:"user"`
	Contact     string `width:"128" nullable:"false" create:"required" list:"user" get:"user"`
	CreatedAt   time.Time
	TargetTime  time.Time
	GroupTimes  uint
	TopicId     string `width:"128" nullable:"false" create:"required" list:"user" get:"user"`
	DomainId    string `width:"128" nullable:"false" create:"required" list:"user" get:"user"`
}

func (ngm *SNotificationGroupManager) ReSend(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	ngs := []SNotificationGroup{}
	q := ngm.Query()
	q = q.LT("target_time", time.Now())
	q = q.Asc("created_at")
	q.DebugQuery()
	err := db.FetchModelObjects(ngm, q, &ngs)
	if err != nil {
		log.Errorln(errors.Wrap(err, "fetch unsend notification_group"))
		return
	}
	log.Infoln("this is ngs:", len(ngs))
	if len(ngs) == 0 {
		return
	}
	sendKey := map[string]apis.SendParams{}
	for _, ng := range ngs {
		if sendParams, ok := sendKey[ng.TopicId+ng.GroupKey+ng.ReceiverId+ng.ContactType]; ok {
			joinStr := " \n"
			if ng.ContactType == apis.EMAIL {
				joinStr = " <br>"
			}
			sendParams.Message += fmt.Sprintf(" %s %s", joinStr, ng.Message)
			if sendParams.ContactType == apis.EMAIL {
				sendParams.EmailMsg = apis.SEmailMessage{
					Subject: sendParams.Title,
					Body:    sendParams.Message,
					To:      []string{ng.Contact},
				}
			}
			sendKey[ng.TopicId+ng.GroupKey+ng.ReceiverId+ng.ContactType] = sendParams
		} else {
			sendKey[ng.TopicId+ng.GroupKey+ng.ReceiverId+ng.ContactType] = apis.SendParams{
				Body:       ng.Body,
				Header:     ng.Header,
				MsgKey:     ng.MsgKey,
				Title:      ng.Title,
				Message:    ng.Message,
				ReceiverId: ng.ReceiverId,
				Receivers: apis.SNotifyReceiver{
					Contact: ng.Contact,
				},
				EmailMsg: apis.SEmailMessage{
					Subject: sendParams.Title,
					Body:    sendParams.Message,
					To:      []string{ng.Contact},
				},
				DomainId:    ng.DomainId,
				ContactType: ng.ContactType,
			}
		}
	}
	dq := sqlchemy.NewRawQuery("delete from notification_groups_tbl")
	rows, err := dq.Rows()
	if err != nil {
		log.Errorln(errors.Wrap(err, "delete resend"))
		return
	}
	defer rows.Close()
	for _, sendParams := range sendKey {
		driver := GetDriver(sendParams.ContactType)
		err = driver.Send(ctx, sendParams)
		if err != nil {
			log.Errorln("this is resend err:", err)
		}
	}
}

func (ng *SNotificationGroupManager) TaskCreate(ctx context.Context, contactType string, args apis.SendParams) error {
	if contactType == apis.WEBCONSOLE {
		return nil
	}
	now := time.Now()
	insertNotificationGroup := SNotificationGroup{
		Id:          db.DefaultUUIDGenerator(),
		ContactType: contactType,
		Body:        args.Body,
		Header:      args.Header,
		MsgKey:      args.MsgKey,
		ReceiverId:  args.ReceiverId,
		Title:       args.Title,
		Message:     args.Message,
		GroupKey:    args.GroupKey,
		Contact:     args.Receivers.Contact,
		CreatedAt:   now,
		TargetTime:  now.Add(time.Duration(args.GroupTimes) * time.Minute),
		GroupTimes:  args.GroupTimes,
		DomainId:    args.DomainId,
	}
	insertNotificationGroup.GetId()
	if contactType == apis.EMAIL {
		insertNotificationGroup.Title = args.EmailMsg.Subject
		insertNotificationGroup.Message = args.EmailMsg.Body
		insertNotificationGroup.Contact = args.EmailMsg.To[0]
	}
	return NotificationGroupManager.TableSpec().Insert(ctx, &insertNotificationGroup)
}

func (ng *SNotificationGroupManager) TaskSend(ctx context.Context, input apis.SNotificationGroupSearchInput) (*apis.SendParams, error) {
	q := ng.Query()
	q = q.Between("created_at", input.StartTime, input.EndTime)
	q = q.Equals("group_key", input.GroupKey)
	q = q.Equals("receiver_id", input.ReceiverId)
	q = q.Equals("contact_type", input.ContactType)
	q = q.Equals("domain_id", input.DomainId)
	q = q.Asc("created_at")
	ngs := []SNotificationGroup{}
	err := db.FetchModelObjects(ng, q, &ngs)
	if err != nil {
		return nil, errors.Wrap(err, "fetch notification groups")
	}
	if len(ngs) <= 1 {
		return nil, errors.Wrapf(errors.ErrNotFound, "notification groups just found :%d", len(ngs))
	}
	sendParams := &apis.SendParams{
		Body:       ngs[0].Body,
		Header:     ngs[0].Header,
		MsgKey:     ngs[0].MsgKey,
		Title:      ngs[0].Title,
		ReceiverId: ngs[0].ReceiverId,
		Receivers: apis.SNotifyReceiver{
			Contact: ngs[0].Contact,
		},
		DomainId: ngs[0].DomainId,
	}
	msg := ""
	joinStr := " \n"
	sendParams.Message = msg
	if input.ContactType == apis.EMAIL {
		joinStr = " <br>"
	}
	ids := []string{}
	for _, ng := range ngs {
		msg += fmt.Sprintf("%s %s", ng.Message, joinStr)
		ids = append(ids, fmt.Sprintf("'%s'", ng.Id))
	}
	sendParams.Message = msg
	if input.ContactType == apis.EMAIL {
		sendParams.EmailMsg = apis.SEmailMessage{
			Subject: sendParams.Title,
			Body:    msg,
			To:      []string{ngs[0].Contact},
		}
	}
	defer func(ids []string) {
		sqlStr := fmt.Sprintf(
			"delete from %s  where id in (%s)",
			NotificationGroupManager.TableSpec().Name(),
			strings.Join(ids, ","),
		)
		dq := sqlchemy.NewRawQuery(sqlStr)
		rows, err := dq.Rows()
		if err != nil {
			log.Errorln("unable to delete:", NotificationGroupManager.TableSpec().Name())
			return
		}
		rows.Close()
		log.Infof("delete expired data in %q successfully", NotificationGroupManager.TableSpec().Name())
	}(ids)
	return sendParams, nil
}
