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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/notify/cache"
	_interface "yunion.io/x/onecloud/pkg/notify/interface"
	"yunion.io/x/onecloud/pkg/notify/options"
	"yunion.io/x/onecloud/pkg/notify/utils"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SNotificationManager struct {
	SStatusStandaloneResourceBaseManager
}

var NotificationManager *SNotificationManager
var NotifyService _interface.INotifyService

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

func (self *SNotificationManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeUser
}

func (self *SNotificationManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeUser
}

func (self *SNotificationManager) FetchOwnerId(ctx context.Context,
	data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {

	return db.FetchUserInfo(ctx, data)
}

func (self *SNotificationManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider,
	scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		if scope == rbacutils.ScopeUser {
			if len(owner.GetUserId()) > 0 {
				q = q.Equals("uid", owner.GetUserId())
			}
		}
	}
	return q
}

type SNotification struct {
	SStatusStandaloneResourceBase

	UID         string    `width:"128" nullable:"false" create:"required"`
	ContactType string    `width:"16" nullable:"false" create:"required" list:"user" index:"true"`
	Topic       string    `width:"128" nullable:"true" create:"optional" list:"user"`
	Priority    string    `width:"16" nullable:"true" create:"optional" list:"user"`
	Msg         string    `create:"required"`
	ReceivedAt  time.Time `nullable:"true" list:"user" create:"optional"`
	SendAt      time.Time `nullable:"false"`
	SendBy      string    `width:"128" nullable:"false"`
	// ClusterID identify message with same topic, msg, priority
	ClusterID string `width:"128" charset:"ascii" primary:"true" create:"optional"`
}

type UserDetail struct {
	Status     string
	Name       string
	ReceivedAt time.Time
}

func (self *SNotification) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) *jsonutils.JSONDict {

	// collect user infos
	scopeStr, err := query.GetString("scope")
	if err != nil {
		scopeStr = "system"
	}
	scope := rbacutils.TRbacScope(scopeStr)

	var userDetails []UserDetail
	if scope.HigherEqual(rbacutils.ScopeSystem) {
		// fetch users from database
		userDetails, err = NotificationManager.fetchUserDetailByClusterID(ctx, self.ClusterID)
		if err != nil {
			log.Errorf(err.Error())
		}

	} else {
		userDetail := UserDetail{
			Status:     self.Status,
			Name:       userCred.GetUserId(),
			ReceivedAt: self.ReceivedAt,
		}
		name, err := utils.GetUsernameByID(ctx, self.UID)
		if err == nil && len(name) != 0 {
			userDetail.Name = name
		}
		userDetails = []UserDetail{userDetail}
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.Marshal(userDetails), "user_list")
	return ret
}

func (self *SNotification) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	userDetail := UserDetail{
		Status:     self.Status,
		Name:       self.UID,
		ReceivedAt: self.ReceivedAt,
	}
	name, err := utils.GetUsernameByID(ctx, self.UID)
	if err == nil && len(name) != 0 {
		userDetail.Name = name
	}
	ret := jsonutils.NewDict()
	data := jsonutils.Marshal([]UserDetail{userDetail})
	ret.Add(data, "user_list")
	return ret, nil
}

func (self *SNotificationManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SNotificationManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

type sUpdate struct {
	ID          string
	UID         string
	Topic       string
	Priority    string
	ContactType string
}

func (self *SNotificationManager) InitializeData() error {
	scope := time.Duration(options.Options.InitNotificationScope) * time.Hour
	time := time.Now().Add(-scope)
	q := self.Query("id", "uid", "topic", "priority", "contact_type").GE("created_at",
		time).Desc("received_at").Equals("contact_type", "webconsole")
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("cluster_id")), sqlchemy.IsEmpty(q.Field("cluster_id"))))
	rows, err := q.Rows()
	if err != nil {
		return err
	}
	updates, update := make([]sUpdate, 0, 10), sUpdate{}
	for rows.Next() {
		err := rows.Scan(&update.ID, &update.UID, &update.Topic, &update.Priority, &update.ContactType)
		if err == nil {
			updates = append(updates, update)
		}
	}
	log.Debugf("this is total %d updates", len(updates))

	// updates is too little
	//if len(updates) < 100 {
	//	updates = updates[:0]
	//	q := self.Query("id", "uid", "topic", "priority", "contact_type").Desc("received_at").Equals("contact_type",
	//		"webconsole").Limit(500)
	//	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("cluster_id")), sqlchemy.IsEmpty(q.Field("cluster_id"))))
	//	rows, err := q.Rows()
	//	if err != nil {
	//		return err
	//	}
	//	for rows.Next() {
	//		err := rows.Scan(&update.ID, &update.UID, &update.Topic, &update.Priority, &update.ContactType)
	//		if err == nil {
	//			updates = append(updates, update)
	//		}
	//	}
	//	log.Debugf("this is total %d updates", len(updates))
	//}

	cache := make([]string, 0, 10)
	if len(updates) > 0 {
		cache = append(cache, updates[0].ID)
	}
	for i := 1; i < len(updates); i++ {
		if updates[i].Topic == updates[i-1].Topic && updates[i].Priority == updates[i-1].Priority {

			cache = append(cache, updates[i].ID)
			continue
		}
		err = self.syncDatabase(cache)
		if err != nil {
			return errors.Wrap(err, "exec sql error")
		}
		cache = cache[:0]
		if i < len(updates)-1 {
			cache = append(cache, updates[i].ID)
		}
	}
	if len(cache) == 0 {
		return nil
	}
	err = self.syncDatabase(cache)
	if err != nil {
		return errors.Wrap(err, "exec sql error")
	}

	return nil
}

func (self *SNotificationManager) syncDatabase(ids []string) error {

	sql := "update %s set updated_at=update_at, deleted=is_deleted, cluster_id='%s' where id in %s"

	newUid := DefaultUUIDGenerator()

	buffer := new(strings.Builder)
	buffer.WriteString("(")
	for _, id := range ids {
		buffer.WriteString("'")
		buffer.WriteString(id)
		buffer.WriteString("', ")
	}
	newSql := fmt.Sprintf(sql, self.TableSpec().Name(), newUid, buffer.String()[:buffer.Len()-2]+")")
	q := sqlchemy.NewRawQuery(newSql)
	rows, err := q.Rows()
	defer rows.Close()
	return err
}

func (self *SNotificationManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {

	// no domainID for now
	scopeStr, err := query.GetString("scope")
	if err != nil {
		scopeStr = "system"
	}
	scope := rbacutils.TRbacScope(scopeStr)

	if !scope.HigherEqual(rbacutils.ScopeSystem) {
		q = q.Equals("uid", userCred.GetUserId())
	}

	q = q.GroupBy("cluster_id")
	return q, nil
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

	now, clusterId := time.Now(), DefaultUUIDGenerator()
	for i := range contacts {
		createData := map[string]interface{}{
			"uid":          contacts[i].UID,
			"contact_type": contacts[i].ContactType,
			"topic":        topic,
			"priority":     priority,
			"msg":          msg,
			"received_at":  now,
			"send_by":      userCred.GetUserId(),
			"status":       NOTIFY_RECEIVED,
			"cluster_id":   clusterId,
		}
		model, err := db.DoCreate(self, ctx, userCred, jsonutils.JSONNull, jsonutils.Marshal(createData), ownerID)
		if err != nil {
			createFailed = append(createFailed, contacts[i].ID)
		} else {
			createSuccess = append(createSuccess, model.(*SNotification))
			contactSuccess = append(contactSuccess, contacts[i].Contact)
		}
	}
	Send(createSuccess, userCred, contactSuccess)
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

func (self *SNotificationManager) fetchUserDetailByClusterID(ctx context.Context, clusterID string) ([]UserDetail,
	error) {
	q := self.Query("uid", "status", "received_at").Equals("cluster_id", clusterID)
	row, err := q.Rows()
	if err != nil {
		return nil, err
	}
	ret := make([]UserDetail, 0)
	userIds := make([]string, 0)
	var userId, status string
	var receviedAt time.Time
	for row.Next() {
		err := row.Scan(&userId, &status, &receviedAt)
		if err != nil {
			return nil, errors.Wrap(err, "sql.row parse error")
		}
		userIds = append(userIds, userId)
		ret = append(ret, UserDetail{
			Status:     status,
			Name:       userId,
			ReceivedAt: receviedAt,
		})
	}

	userMap, err := cache.UserCacheManager.FetchUsersByIDs(ctx, userIds)
	if err != nil {
		return nil, errors.Wrap(err, "fetch users by ids failed")
	}
	for i := range ret {
		if user, ok := userMap[ret[i].Name]; ok {
			ret[i].Name = user.Name
		}
	}

	return ret, nil
}

func (self *SNotificationManager) FetchNotOK(lastTime time.Time) ([]SNotification, error) {
	q := self.Query()
	q.Filter(sqlchemy.AND(sqlchemy.GE(q.Field("created_at"), lastTime), sqlchemy.NotEquals(q.Field("status"), NOTIFY_OK)))
	records := make([]SNotification, 0, 10)
	err := db.FetchModelObjects(self, q, &records)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (self *SNotification) SetStatus(userCred mcclient.TokenCredential, status string, reason string) error {
	if self.Status == status {
		return nil
	}
	oldStatus := self.Status
	_, err := db.Update(self, func() error {
		self.Status = status
		return nil
	})
	if err != nil {
		return err
	}
	if userCred != nil {
		notes := fmt.Sprintf("%s=>%s", oldStatus, status)
		if len(reason) > 0 {
			notes = fmt.Sprintf("%s: %s", notes, reason)
		}
		db.OpsLog.LogEvent(self, db.ACT_UPDATE_STATUS, notes, userCred)
	}
	return nil
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
	// limit the number of Concurrency
	Max := 10
	limit := make(chan struct{}, Max)
	sendone := func(notification SNotification) {
		defer func() {
			<-limit
		}()
		// Get contact
		contact, err := ContactManager.FetchByUIDAndCType(notification.UID, []string{notification.ContactType})
		if err != nil {
			log.Debugf("fail to fetch contacts with uid '%s' in ReSend Cron Job", notification.UID)
			return
		}
		if len(contact) == 0 {
			return
		}
		// sent_at update todo
		notification.SetStatusWithoutUserCred(NOTIFY_SENT)
		err = NotifyService.Send(context.Background(), notification.ContactType, contact[0].Contact, notification.Topic,
			notification.Msg,
			notification.Priority)
		if err == nil {
			return
		}
		if err != nil {
			log.Errorf("Send notification failed in ReSend Cron Job: %s.", err.Error())
			notification.SetStatusWithoutUserCred(NOTIFY_FAIL)
		} else {
			notification.SetStatusWithoutUserCred(NOTIFY_OK)
		}
	}
	for i := range notifications {
		limit <- struct{}{}
		go sendone(notifications[i])
	}
	// wait all finish
	for i := 0; i < Max; i++ {
		limit <- struct{}{}
	}
}

func ReSend(seconds int) {
	scope := time.Duration(seconds) * time.Second
	notifications, err := NotificationManager.FetchNotOK(time.Now().Add(-scope))
	if err != nil {
		return
	}
	log.Debugf("Start to resend message with a total of %d", len(notifications))
	sendWithoutUserCred(notifications)
}
