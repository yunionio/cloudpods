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
	"database/sql"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	notifyv2 "yunion.io/x/onecloud/pkg/notify"
	"yunion.io/x/onecloud/pkg/notify/oldmodels"
	"yunion.io/x/onecloud/pkg/notify/options"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SNotificationManager struct {
	db.SStatusStandaloneResourceBaseManager
}

var NotificationManager *SNotificationManager
var NotifyService notifyv2.INotifyService

func init() {
	NotificationManager = &SNotificationManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SNotification{},
			"notifications_tbl",
			"notification",
			"notifications",
		),
	}
	NotificationManager.SetVirtualObject(NotificationManager)
}

type SNotification struct {
	db.SStatusStandaloneResourceBase

	ContactType string `width:"16" nullable:"false" create:"required" list:"user" get:"user" index:"true"`
	// swagger:ignore
	Topic    string `width:"128" nullable:"true" create:"required"`
	Priority string `width:"16" nullable:"true" create:"optional" list:"user" get:"user"`
	// swagger:ignore
	Message    string    `create:"required"`
	ReceivedAt time.Time `nullable:"true" list:"user" get:"user"`
	SendTimes  int
}

func (nm *SNotificationManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.NotificationCreateInput) (api.NotificationCreateInput, error) {
	// check contact type enabled
	allContactType, err := ConfigManager.allContactType()
	if err != nil {
		return input, err
	}
	if !utils.IsInStringArray(input.ContactType, allContactType) {
		return input, httperrors.NewInputParameterError("Unconfigured contact type %q", input.ContactType)
	}
	// check uids, rids and contacts
	if len(input.Receivers) == 0 && len(input.Contacts) == 0 {
		return input, httperrors.NewMissingParameterError("receivers | contacts")
	}
	// check receivers
	if len(input.Receivers) > 0 {
		receivers, err := ReceiverManager.FetchByIdOrNames(ctx, input.Receivers...)
		if err != nil {
			return input, errors.Wrap(err, "ReceiverManager.FetchByIDs")
		}
		idSet := sets.NewString()
		nameSet := sets.NewString()
		for i := range receivers {
			idSet.Insert(receivers[i].Id)
			nameSet.Insert(receivers[i].Name)
		}
		for _, re := range input.Receivers {
			if idSet.Has(re) || nameSet.Has(re) {
				continue
			}
			return input, httperrors.NewInputParameterError("no such receiver whose uid is %q", re)
		}
		input.Receivers = idSet.UnsortedList()
	}
	nowStr := time.Now().Format("2006-01-02 15:04:05")
	// hack
	input.Name = fmt.Sprintf("%s(%s)", input.Topic, nowStr)
	return input, nil
}

func (n *SNotification) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	n.ReceivedAt = time.Now()
	n.Id = db.DefaultUUIDGenerator()
	var input api.NotificationCreateInput
	err := data.Unmarshal(&input)
	if err != nil {
		return err
	}
	for i := range input.Receivers {
		_, err := ReceiverNotificationManager.Create(ctx, userCred, input.Receivers[i], n.Id)
		if err != nil {
			return errors.Wrap(err, "ReceiverNotificationManager.Create")
		}
	}
	return nil
}

func (n *SNotification) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	n.SetStatus(userCred, api.NOTIFICATION_STATUS_RECEIVED, "")
	task, err := taskman.TaskManager.NewTask(ctx, "NotificationSendTask", n, userCred, nil, "", "")
	if err != nil {
		log.Errorf("NotificationSendTask newTask error %v", err)
	} else {
		task.ScheduleRun(nil)
	}
}

func (nm *SNotificationManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NotificationDetails {
	rows := make([]api.NotificationDetails, len(objs))

	resRows := nm.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	var err error
	for i := range rows {
		rows[i], err = objs[i].(*SNotification).getMoreDetails(ctx, query, rows[i])
		if err != nil {
			log.Errorf("Notification.getMoreDetails: %v", err)
		}
		rows[i].StatusStandaloneResourceDetails = resRows[i]
	}

	return rows
}

func (n *SNotification) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.NotificationDetails, error) {
	return api.NotificationDetails{}, nil
}

func (n *SNotification) ReceiverNotificationsNotOK() ([]SReceiverNotification, error) {
	rnq := ReceiverNotificationManager.Query().Equals("notification_id", n.Id).NotEquals("status", api.RECEIVER_NOTIFICATION_OK)
	rns := make([]SReceiverNotification, 0, 1)
	err := db.FetchModelObjects(ReceiverNotificationManager, rnq, &rns)
	if err != nil {
		return nil, err
	}
	return rns, nil
}

func (n *SNotification) ReceiveDetails() ([]api.ReceiveDetail, error) {
	subRQ := ReceiverManager.Query("id", "name").SubQuery()
	q := ReceiverNotificationManager.Query("receiver_id", "notification_id", "contact", "send_at", "send_by", "status", "failed_reason").Equals("notification_id", n.Id)
	q.AppendField(subRQ.Field("name", "receiver_name"))
	q = q.Join(subRQ, sqlchemy.Equals(q.Field("receiver_id"), subRQ.Field("id")))
	ret := make([]api.ReceiveDetail, 0, 2)
	err := q.All(&ret)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		log.Errorf("SQuery.All: %v", err)
		return nil, err
	}
	return ret, nil
}

func (n *SNotification) getMoreDetails(ctx context.Context, query jsonutils.JSONObject, out api.NotificationDetails) (api.NotificationDetails, error) {
	// get title adn content
	p, err := TemplateManager.NotifyFilter(n.ContactType, n.Topic, n.Message)
	if err != nil {
		return out, err
	}
	out.Title = p.Title
	out.Content = p.Message
	// get receive details
	out.ReceiveDetails, err = n.ReceiveDetails()
	if err != nil {
		return out, err
	}
	return out, nil
}

func (nm *SNotificationManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	if !data.Contains("contacts") {
		return db.IsAdminAllowCreate(userCred, nm)
	}
	return true
}

func (nm *SNotificationManager) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (nm *SNotificationManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeUser
}

func (nm *SNotificationManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeSystem
}

func (nm *SNotificationManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return db.FetchUserInfo(ctx, data)
}

func (nm *SNotificationManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner == nil {
		return q
	}
	switch scope {
	case rbacutils.ScopeDomain:
		subRq := ReceiverManager.Query("id").Equals("domain_id", owner.GetDomainId()).SubQuery()
		RNq := ReceiverNotificationManager.Query("notification_id", "receiver_id")
		subRNq := RNq.Join(subRq, sqlchemy.Equals(RNq.Field("receiver_id"), subRq.Field("id"))).SubQuery()
		q = q.Join(subRNq, sqlchemy.Equals(q.Field("id"), subRNq.Field("notification_id")))
	case rbacutils.ScopeProject, rbacutils.ScopeUser:
		subq := ReceiverNotificationManager.Query("notification_id").Equals("receiver_id", owner.GetUserId()).SubQuery()
		q = q.Join(subq, sqlchemy.Equals(q.Field("id"), subq.Field("notification_id")))
	}
	return q
}

func (n *SNotification) AddOne() error {
	_, err := db.Update(n, func() error {
		n.SendTimes += 1
		return nil
	})
	return err
}

const (
	NOTIFY_RECEIVED = "received"  // Received a task about sending a notification
	NOTIFY_SENT     = "sent"      // Nofity module has sent notification, but result unkown
	NOTIFY_OK       = "sent_ok"   // Notification was sent successfully
	NOTIFY_FAIL     = "sent_fail" // That sent a notification is failed
	NOTIFY_REMOVED  = "removed"
)

func (self *SNotificationManager) singleRowLineQuery(sqlStr string, dest ...interface{}) error {
	q := sqlchemy.NewRawQuery(sqlStr)
	rows, err := q.Rows()
	if err != nil {
		return errors.Wrap(err, "q.Rows")
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(dest...)
		if err != nil {
			return errors.Wrap(err, "rows.Scan")
		}
		return nil
	}
	return sql.ErrNoRows
}

func (self *SNotificationManager) InitializeData() error {
	// check
	sqlStr := fmt.Sprintf(
		"select count(*) as total from (select cluster_id from %s where status='%s' and contact_type='webconsole' group by cluster_id) as cluster",
		oldmodels.NotificationManager.TableSpec().Name(),
		NOTIFY_REMOVED,
	)
	var count int
	err := self.singleRowLineQuery(sqlStr, &count)
	if err != nil {
		return err
	}
	if count >= options.Options.MaxSyncNotification {
		return nil
	}

	limitTimeStr := time.Now().Add(time.Duration(-30) * time.Hour * 24).Format("2006-01-02 15:04:05")

	// get min received_at
	var minReceivedAt time.Time
	sqlStr = fmt.Sprintf(
		"select min(received_at) as min_received_at from (select received_at from %s where received_at > '%s' and contact_type = 'webconsole' group by cluster_id order by received_at desc limit %d) as cluster",
		oldmodels.NotificationManager.TableSpec().Name(),
		limitTimeStr,
		options.Options.MaxSyncNotification,
	)
	err = self.singleRowLineQuery(sqlStr, &minReceivedAt)
	if err != nil {
		return err
	}
	log.Infof("minReceivedAt: %s", minReceivedAt)

	ctx := context.Background()
	q := oldmodels.NotificationManager.Query().Equals("contact_type", api.WEBCONSOLE).GT("received_at", minReceivedAt).NotEquals("status", NOTIFY_REMOVED)
	n := q.Count()
	log.Infof("total %d notifications to sync", n)
	oldNotifications := make([]oldmodels.SNotification, 0, n)
	err = db.FetchModelObjects(oldmodels.NotificationManager, q, &oldNotifications)
	if err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}

	// build cluster=>Notification
	cnMap := make(map[string][]*oldmodels.SNotification)
	for i := range oldNotifications {
		clusterId := oldNotifications[i].ClusterID
		if _, ok := cnMap[clusterId]; !ok {
			cnMap[clusterId] = make([]*oldmodels.SNotification, 0, 2)
		}
		cnMap[clusterId] = append(cnMap[clusterId], &oldNotifications[i])
	}

	for _, oldNotifications := range cnMap {
		oldNotificaion := oldNotifications[0]
		newNotification := SNotification{
			ContactType: oldNotificaion.ContactType,
			Topic:       oldNotificaion.Topic,
			Priority:    oldNotificaion.Priority,
			Message:     oldNotificaion.Msg,
			ReceivedAt:  oldNotificaion.ReceivedAt,
		}
		newNotification.Id = db.DefaultUUIDGenerator()
		statusMap := make(map[string]int, 4)
		for _, oldNotificaion := range oldNotifications {
			rn := SReceiverNotification{
				ReceiverID:     oldNotificaion.UID,
				NotificationID: newNotification.Id,
				SendAt:         oldNotificaion.SendAt,
				SendBy:         oldNotificaion.SendBy,
				Status:         oldNotificaion.Status,
			}
			if rn.Status == NOTIFY_SENT {
				rn.Status = api.NOTIFICATION_STATUS_SENDING
			}
			statusMap[rn.Status] += 1
			err := ReceiverNotificationManager.TableSpec().Insert(ctx, &rn)
			if err != nil {
				return errors.Wrap(err, "TableSpec().Insert")
			}
		}
		switch {
		case statusMap[api.RECEIVER_NOTIFICATION_OK] == len(oldNotifications):
			newNotification.Status = api.NOTIFICATION_STATUS_OK
		case statusMap[api.RECEIVER_NOTIFICATION_RECEIVED] == len(oldNotifications):
			newNotification.Status = api.NOTIFICATION_STATUS_RECEIVED
		case statusMap[api.RECEIVER_NOTIFICATION_FAIL] == len(oldNotifications):
			newNotification.Status = api.NOTIFICATION_STATUS_FAILED
		case statusMap[api.RECEIVER_NOTIFICATION_FAIL] == 0 && statusMap[api.RECEIVER_NOTIFICATION_SENT] > 0:
			newNotification.Status = api.NOTIFICATION_STATUS_SENDING
		default:
			newNotification.Status = api.NOTIFICATION_STATUS_PART_OK
		}
		err := self.TableSpec().InsertOrUpdate(ctx, &newNotification)
		if err != nil {
			return errors.Wrap(err, "TableSpec().InsertOrUpdate")
		}

		// mark removed
		for _, oldNotificaion := range oldNotifications {
			_, err := db.Update(oldNotificaion, func() error {
				oldNotificaion.Status = NOTIFY_REMOVED
				return nil
			})
			if err != nil {
				return errors.Wrap(err, "Delete")
			}
		}
	}
	return nil
}

// 通知消息列表
func (nm *SNotificationManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.NotificationListInput) (*sqlchemy.SQuery, error) {
	q, err := nm.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	if len(input.ContactType) > 0 {
		q = q.Equals("contact_type", input.ContactType)
	}
	if len(input.ReceiverId) > 0 {
		subq := ReceiverNotificationManager.Query("notification_id").Equals("receiver_id", input.ReceiverId).SubQuery()
		q = q.Join(subq, sqlchemy.Equals(q.Field("id"), subq.Field("notification_id")))
	}
	return q, nil
}

func (nm *SNotificationManager) ReSend(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	q := nm.Query().NotEquals("status", api.NOTIFICATION_STATUS_OK).LT("send_times", options.Options.MaxSendTimes)
	ns := make([]SNotification, 0, 2)
	err := db.FetchModelObjects(nm, q, &ns)
	if err != nil {
		log.Errorf("fail to FetchModelObjects: %v", err)
		return
	}
	for i := range ns {
		task, err := taskman.TaskManager.NewTask(ctx, "NotificationSendTask", &ns[i], userCred, nil, "", "")
		if err != nil {
			log.Errorf("NotificationSendTask newTask error %v", err)
		} else {
			task.ScheduleRun(nil)
		}
	}
}
