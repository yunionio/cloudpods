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
)

type SNotificationGroupManager struct {
	db.SModelBaseManager
}

var NotificationGroupManager *SNotificationGroupManager

func init() {
	NotificationGroupManager = &SNotificationGroupManager{
		SModelBaseManager: db.NewModelBaseManager(
			SNotificationGroup{},
			"notification_group_tbl",
			"notification_group",
			"notification_groups",
		),
	}
	NotificationGroupManager.SetVirtualObject(NotificationGroupManager)
}

// 站内信
type SNotificationGroup struct {
	db.SModelBase

	Id       string `width:"128" nullable:"false" create:"required" list:"user"  index:"true" get:"user"`
	GroupKey string `width:"128" nullable:"false" create:"required" list:"user" get:"user"`
	Title    string
	// swagger:ignore
	Message     string `length:"medium"`
	ReceiverId  string `width:"128" nullable:"false" create:"required" list:"user" get:"user"`
	Body        jsonutils.JSONObject
	Header      jsonutils.JSONObject
	MsgKey      string
	ContactType string `width:"32" nullable:"false" create:"required" list:"user" get:"user"`
	Contact     string `width:"128" nullable:"false" create:"required" list:"user" get:"user"`
	CreatedAt   time.Time
	DomainId    string `width:"128" nullable:"false" create:"required" list:"user" get:"user"`
}

func (ng *SNotificationGroupManager) TaskCreate(ctx context.Context, contactType string, args apis.SendParams) error {
	if contactType == apis.WEBCONSOLE {
		return nil
	}
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
		CreatedAt:   time.Now(),
		DomainId:    args.DomainId,
	}
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
	deleteIds := []string{}
	for _, ng := range ngs {
		msg += fmt.Sprintf("%s %s", ng.Message, joinStr)
		deleteIds = append(deleteIds, fmt.Sprintf("'%s'", ng.Id))
	}

	defer func() {
		_, err = sqlchemy.Exec(fmt.Sprintf("delete from %s where id in (%s) ", ng.TableSpec().Name(), strings.Join(deleteIds, ",")))
		if err != nil {
			log.Errorln("clean notification_groups err:", err)
		}
	}()

	sendParams.Message = msg
	if input.ContactType == apis.EMAIL {
		sendParams.EmailMsg = apis.SEmailMessage{
			Subject: sendParams.Title,
			Body:    msg,
			To:      []string{ngs[0].Contact},
		}
	}
	return sendParams, nil
}
