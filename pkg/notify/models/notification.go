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
	"yunion.io/x/onecloud/pkg/image/policy"
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
	Topic    string `width:"128" nullable:"true" create:"required" search:"user"`
	Priority string `width:"16" nullable:"true" create:"optional" list:"user" get:"user"`
	// swagger:ignore
	Message    string    `create:"required"`
	ReceivedAt time.Time `nullable:"true" list:"user" get:"user"`
	EventId    string    `width:"128" nullable:"true"`
	SendTimes  int
	Tag        string `width:"16" nullable:"true" index:"true" create:"optional"`
}

const (
	SendByContact = "send_by_contact"
)

func (nm *SNotificationManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.NotificationCreateInput) (api.NotificationCreateInput, error) {
	if len(input.Tag) > 0 && !utils.IsInStringArray(input.Tag, []string{api.NOTIFICATION_TAG_ALERT}) {
		return input, httperrors.NewInputParameterError("invalid tag")
	}
	if len(input.Contacts) > 0 {
		if !userCred.IsAllow(rbacutils.ScopeSystem, api.SERVICE_TYPE, nm.KeywordPlural(), policy.PolicyActionPerform, SendByContact) {
			return input, httperrors.NewForbiddenError("only admin can send notification by contact")
		}
		if len(input.Contacts) == 0 {
			input.Contacts = []string{""}
		}
	}
	log.Infof("notify input: %s", jsonutils.Marshal(input))

	// check robot
	if len(input.Robots) > 0 {
		input.ContactType = api.ROBOT
		robots, err := RobotManager.FetchByIdOrNames(ctx, input.Robots...)
		if err != nil {
			return input, errors.Wrap(err, "RobotManager.FetchByIdOrNames")
		}
		idSet := sets.NewString()
		nameSet := sets.NewString()
		for i := range robots {
			idSet.Insert(robots[i].Id)
			nameSet.Insert(robots[i].Name)
		}
		for _, re := range input.Receivers {
			if idSet.Has(re) || nameSet.Has(re) {
				continue
			}
			if !input.IgnoreNonexistentReceiver {
				return input, httperrors.NewInputParameterError("no such robot whose id is %q", re)
			}
		}
		input.Robots = idSet.UnsortedList()
		if len(input.Robots) == 0 {
			return input, httperrors.NewInputParameterError("no valid receiver or contact")
		}
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
			if input.ContactType == api.WEBCONSOLE {
				input.Contacts = append(input.Contacts, re)
			}
			if !input.IgnoreNonexistentReceiver {
				return input, httperrors.NewInputParameterError("no such receiver whose uid is %q", re)
			}
		}
		input.Receivers = idSet.UnsortedList()
		if len(input.Receivers)+len(input.Contacts) == 0 {
			return input, httperrors.NewInputParameterError("no valid receiver or contact")
		}
	}
	nowStr := time.Now().Format("2006-01-02 15:04:05")
	if len(input.Priority) == 0 {
		input.Priority = api.NOTIFICATION_PRIORITY_NORMAL
	}
	// hack
	length := 10
	if len(input.Topic) < 10 {
		length = len(input.Topic)
	}
	name := fmt.Sprintf("%s-%s-%s", input.Topic[:length], input.ContactType, nowStr)
	var err error
	input.Name, err = db.GenerateName(ctx, nm, ownerId, name)
	if err != nil {
		return input, errors.Wrapf(err, "unable to generate name for %s", name)
	}
	log.Infof("after validatecreate input: %s", jsonutils.Marshal(input))
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
	for i := range input.Contacts {
		_, err := ReceiverNotificationManager.CreateContact(ctx, userCred, input.Contacts[i], n.Id)
		if err != nil {
			return errors.Wrap(err, "ReceiverNotificationManager.CreateContact")
		}
	}
	for i := range input.Robots {
		_, err := ReceiverNotificationManager.CreateRobot(ctx, userCred, input.Robots[i], n.Id)
		if err != nil {
			return errors.Wrap(err, "ReceiverNotificationManager.CreateRobot")
		}
	}
	return nil
}

func (n *SNotification) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	if data.Contains("metadata") {
		metadata := make(map[string]interface{})
		err := data.Unmarshal(&metadata, "metadata")
		if err != nil {
			log.Errorf("unable to unmarshal to metadata: %v", err)
		} else {
			n.SetAllMetadata(ctx, metadata, userCred)
		}
	}
	n.SetStatus(userCred, api.NOTIFICATION_STATUS_RECEIVED, "")
	task, err := taskman.TaskManager.NewTask(ctx, "NotificationSendTask", n, userCred, nil, "", "")
	if err != nil {
		log.Errorf("NotificationSendTask newTask error %v", err)
	} else {
		task.ScheduleRun(nil)
	}
}

func (nm *SNotificationManager) AllowPerformEventNotify(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, nm, "event-notify")
}

// TODO: support project and domain
func (nm *SNotificationManager) PerformEventNotify(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.NotificationManagerEventNotifyInput) (api.NotificationManagerEventNotifyOutput, error) {
	log.Infof("default receiverIds: %s", input.ReceiverIds)
	var output api.NotificationManagerEventNotifyOutput
	// check event
	_, err := parseEvent(input.Event)
	if err != nil {
		return output, httperrors.NewInputParameterError("unable to parse event %q", input.Event)
	}
	// contact type
	contactTypes := input.ContactTypes
	cts, err := ConfigManager.allContactType()
	if err != nil {
		return output, errors.Wrap(err, "unable to fetch allContactType")
	}
	if len(contactTypes) == 0 {
		contactTypes = intersection(cts, PersonalConfigContactTypes)
	}

	// receiver
	topics, err := TopicManager.TopicsByEvent(input.Event, input.AdvanceDays)
	if err != nil {
		return output, errors.Wrapf(err, "unable fetch subscriptions by event %q", input.Event)
	}
	if len(topics) == 0 {
		return output, nil
	}
	var receiverIds []string
	for i := range topics {
		receiverIds1, err := SubscriberManager.getReceiversSent(ctx, topics[i].Id, input.ProjectDomainId, input.ProjectId)
		if err != nil {
			return output, errors.Wrap(err, "unable to get receive")
		}
		log.Infof("receiver for topic: %s", receiverIds1)
		receiverIds = append(receiverIds, receiverIds1...)
	}

	// robot
	var robots []string
	for i := range topics {
		_robots, err := SubscriberManager.robot(topics[i].Id, input.ProjectDomainId, input.ProjectId)
		if err != nil {
			if errors.Cause(err) != errors.ErrNotFound {
				return output, errors.Wrapf(err, "unable fetch robot of subscription %q", topics[i].Id)
			}
		} else {
			robots = append(robots, _robots...)
		}
	}
	var webhookRobots []string
	if len(robots) > 0 {
		robots = sets.NewString(robots...).UnsortedList()
		rs, err := RobotManager.FetchByIdOrNames(ctx, robots...)
		if err != nil {
			return output, errors.Wrap(err, "unable to get robots")
		}
		robots, webhookRobots = make([]string, 0, len(rs)), make([]string, 0, 1)
		for i := range rs {
			if rs[i].Type == api.ROBOT_TYPE_WEBHOOK {
				webhookRobots = append(webhookRobots, rs[i].Id)
			} else {
				robots = append(robots, rs[i].Id)
			}
		}
	}

	message := jsonutils.Marshal(input.ResourceDetails).String()

	// append default receiver
	receiverIds = append(receiverIds, input.ReceiverIds...)
	// fillter non-existed receiver
	receivers, err := ReceiverManager.FetchByIdOrNames(ctx, receiverIds...)
	if err != nil {
		return output, errors.Wrap(err, "unable to fetch receivers by ids")
	}
	webconsoleContacts := sets.NewString()
	idSet := sets.NewString()
	for i := range receivers {
		idSet.Insert(receivers[i].Id)
	}
	for _, re := range receiverIds {
		if idSet.Has(re) {
			continue
		}
		webconsoleContacts.Insert(re)
	}
	receiverIds = idSet.UnsortedList()

	// create event
	event, err := EventManager.CreateEvent(ctx, input.Event, message, input.AdvanceDays)
	if err != nil {
		return output, errors.Wrap(err, "unable to create Event")
	}

	// webconsole
	err = nm.create(ctx, userCred, api.WEBCONSOLE, receiverIds, webconsoleContacts.UnsortedList(), input.Priority, event.Id)
	if err != nil {
		output.FailedList = append(output.FailedList, api.FailedElem{
			ContactType: api.WEBCONSOLE,
			Reason:      err.Error(),
		})
	}
	// normal contact type
	for _, ct := range contactTypes {
		if ct == api.MOBILE {
			continue
		}
		err := nm.create(ctx, userCred, ct, receiverIds, nil, input.Priority, event.Id)
		if err != nil {
			output.FailedList = append(output.FailedList, api.FailedElem{
				ContactType: ct,
				Reason:      err.Error(),
			})
		}
	}
	err = nm.createWithWebhookRobots(ctx, userCred, webhookRobots, input.Priority, event.Id)
	if err != nil {
		output.FailedList = append(output.FailedList, api.FailedElem{
			ContactType: api.WEBHOOK,
			Reason:      err.Error(),
		})
	}
	// robot
	err = nm.createWithRobots(ctx, userCred, robots, input.Priority, event.Id)
	if err != nil {
		output.FailedList = append(output.FailedList, api.FailedElem{
			ContactType: api.ROBOT,
			Reason:      err.Error(),
		})
	}
	return output, nil
}

func (nm *SNotificationManager) createWithWebhookRobots(ctx context.Context, userCred mcclient.TokenCredential, webhookRobotIds []string, priority, eventId string) error {
	if len(webhookRobotIds) == 0 {
		return nil
	}
	n := &SNotification{
		ContactType: api.WEBHOOK,
		Priority:    priority,
		ReceivedAt:  time.Now(),
		EventId:     eventId,
	}
	n.Id = db.DefaultUUIDGenerator()
	for i := range webhookRobotIds {
		_, err := ReceiverNotificationManager.CreateRobot(ctx, userCred, webhookRobotIds[i], n.Id)
		if err != nil {
			return errors.Wrap(err, "ReceiverNotificationManager.CreateRobot")
		}
	}
	err := nm.TableSpec().Insert(ctx, n)
	if err != nil {
		return errors.Wrap(err, "unable to insert Notification")
	}
	n.SetModelManager(nm, n)
	task, err := taskman.TaskManager.NewTask(ctx, "NotificationSendTask", n, userCred, nil, "", "")
	if err != nil {
		log.Errorf("NotificationSendTask newTask error %v", err)
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (nm *SNotificationManager) createWithRobots(ctx context.Context, userCred mcclient.TokenCredential, robotIds []string, priority, eventId string) error {
	if len(robotIds) == 0 {
		return nil
	}
	n := &SNotification{
		ContactType: api.ROBOT,
		Priority:    priority,
		ReceivedAt:  time.Now(),
		EventId:     eventId,
	}
	n.Id = db.DefaultUUIDGenerator()
	for i := range robotIds {
		_, err := ReceiverNotificationManager.CreateRobot(ctx, userCred, robotIds[i], n.Id)
		if err != nil {
			return errors.Wrap(err, "ReceiverNotificationManager.CreateRobot")
		}
	}
	err := nm.TableSpec().Insert(ctx, n)
	if err != nil {
		return errors.Wrap(err, "unable to insert Notification")
	}
	n.SetModelManager(nm, n)
	task, err := taskman.TaskManager.NewTask(ctx, "NotificationSendTask", n, userCred, nil, "", "")
	if err != nil {
		log.Errorf("NotificationSendTask newTask error %v", err)
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (nm *SNotificationManager) create(ctx context.Context, userCred mcclient.TokenCredential, contactType string, receiverIds, contacts []string, priority, eventId string) error {
	if len(receiverIds)+len(contacts) == 0 {
		log.Infof("%s: no send", contactType)
		return nil
	}

	n := &SNotification{
		ContactType: contactType,
		Priority:    priority,
		ReceivedAt:  time.Now(),
		EventId:     eventId,
	}
	n.Id = db.DefaultUUIDGenerator()
	err := nm.TableSpec().Insert(ctx, n)
	if err != nil {
		return errors.Wrap(err, "unable to insert Notification")
	}
	for i := range receiverIds {
		_, err := ReceiverNotificationManager.Create(ctx, userCred, receiverIds[i], n.Id)
		if err != nil {
			return errors.Wrap(err, "ReceiverNotificationManager.Create")
		}
	}
	for i := range contacts {
		_, err := ReceiverNotificationManager.CreateContact(ctx, userCred, contacts[i], n.Id)
		if err != nil {
			return errors.Wrap(err, "ReceiverNotificationManager.CreateContact")
		}
	}
	log.Infof("start NotificationSendTask for %s", contactType)
	n.SetModelManager(nm, n)
	task, err := taskman.TaskManager.NewTask(ctx, "NotificationSendTask", n, userCred, nil, "", "")
	if err != nil {
		log.Errorf("NotificationSendTask newTask error %v", err)
	} else {
		task.ScheduleRun(nil)
	}
	return nil
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
		rows[i], err = objs[i].(*SNotification).getMoreDetails(ctx, userCred, query, rows[i])
		if err != nil {
			log.Errorf("Notification.getMoreDetails: %v", err)
		}
		rows[i].StatusStandaloneResourceDetails = resRows[i]
	}

	return rows
}

func (n *SNotification) ReceiverNotificationsNotOK() ([]SReceiverNotification, error) {
	rnq := ReceiverNotificationManager.Query().Equals("notification_id", n.Id).NotEquals("status", api.RECEIVER_NOTIFICATION_OK)
	rns := make([]SReceiverNotification, 0, 1)
	err := db.FetchModelObjects(ReceiverNotificationManager, rnq, &rns)
	if err == sql.ErrNoRows {
		return []SReceiverNotification{}, nil
	}
	if err != nil {
		return nil, err
	}
	return rns, nil
}

func (n *SNotification) ReceiveDetails(userCred mcclient.TokenCredential, scope string) ([]api.ReceiveDetail, error) {
	RQ := ReceiverManager.Query("id", "name")
	q := ReceiverNotificationManager.Query("receiver_id", "notification_id", "contact", "send_at", "send_by", "status", "failed_reason").Equals("notification_id", n.Id)
	s := rbacutils.TRbacScope(scope)

	switch s {
	case rbacutils.ScopeSystem:
		subRQ := RQ.SubQuery()
		q.AppendField(subRQ.Field("name", "receiver_name"))
		q = q.LeftJoin(subRQ, sqlchemy.OR(sqlchemy.Equals(q.Field("receiver_id"), subRQ.Field("id")), sqlchemy.Equals(q.Field("contact"), subRQ.Field("id"))))
	case rbacutils.ScopeDomain:
		subRQ := RQ.Equals("domain_id", userCred.GetDomainId()).SubQuery()
		q.AppendField(subRQ.Field("name", "receiver_name"))
		q = q.Join(subRQ, sqlchemy.OR(sqlchemy.Equals(q.Field("receiver_id"), subRQ.Field("id")), sqlchemy.Equals(q.Field("contact"), subRQ.Field("id"))))
	default:
		subRQ := RQ.Equals("id", userCred.GetUserId()).SubQuery()
		q.AppendField(subRQ.Field("name", "receiver_name"))
		q = q.Join(subRQ, sqlchemy.OR(sqlchemy.Equals(q.Field("receiver_id"), subRQ.Field("id")), sqlchemy.Equals(q.Field("contact"), subRQ.Field("id"))))
	}
	ret := make([]api.ReceiveDetail, 0, 2)
	err := q.All(&ret)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		log.Errorf("SQuery.All: %v", err)
		return nil, err
	}
	return ret, nil
}

func (n *SNotification) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, out api.NotificationDetails) (api.NotificationDetails, error) {
	// get title adn content
	lang := getLangSuffix(ctx)
	nn, err := n.Notification()
	if err != nil {
		return out, err
	}
	p, err := n.TemplateStore().FillWithTemplate(ctx, lang, nn)
	if err != nil {
		return out, err
	}
	out.Title = p.Title
	out.Content = p.Message

	scope, _ := query.GetString("scope")
	// get receive details
	out.ReceiveDetails, err = n.ReceiveDetails(userCred, scope)
	if err != nil {
		return out, err
	}
	return out, nil
}

func (n *SNotification) Notification() (notifyv2.SNotification, error) {
	if n.EventId == "" {
		return notifyv2.SNotification{
			ContactType: n.ContactType,
			Topic:       n.Topic,
			Message:     n.Message,
		}, nil
	}
	event, err := EventManager.GetEvent(n.EventId)
	if err != nil {
		return notifyv2.SNotification{}, err
	}
	e, _ := parseEvent(event.Event)
	return notifyv2.SNotification{
		ContactType: n.ContactType,
		Topic:       n.Topic,
		Message:     event.Message,
		Event:       e,
		AdvanceDays: event.AdvanceDays,
	}, nil
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
		subRNq := RNq.Join(subRq, sqlchemy.OR(sqlchemy.Equals(RNq.Field("receiver_id"), subRq.Field("id")), sqlchemy.Equals(RNq.Field("contact"), subRq.Field("id")))).SubQuery()
		q = q.Join(subRNq, sqlchemy.Equals(q.Field("id"), subRNq.Field("notification_id")))
	case rbacutils.ScopeProject, rbacutils.ScopeUser:
		sq := ReceiverNotificationManager.Query("notification_id")
		subq := sq.Filter(sqlchemy.OR(sqlchemy.Equals(sq.Field("receiver_id"), owner.GetUserId()), sqlchemy.Equals(sq.Field("contact"), owner.GetUserId()))).SubQuery()
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
	return dataCleaning(self.TableSpec().Name())
}

func dataCleaning(tableName string) error {
	now := time.Now()
	monthsDaysAgo := now.AddDate(0, -1, 0).Format("2006-01-02 15:04:05")
	sqlStr := fmt.Sprintf(
		"update %s set deleted = 1 where deleted = 0 and created_at < '%s'",
		tableName,
		monthsDaysAgo,
	)
	q := sqlchemy.NewRawQuery(sqlStr)
	_, err := q.Rows()
	if err != nil {
		return errors.Wrapf(err, "unable to delete expired data in %q", tableName)
	}
	log.Infof("delete expired data in %q successfully", tableName)
	return nil
}

func (self *SNotificationManager) dataMigration() error {
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
	if len(input.Tag) > 0 {
		q = q.Equals("tag", input.Tag)
	}
	return q, nil
}

func (nm *SNotificationManager) ReSend(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	timeLimit := time.Now().Add(-time.Duration(options.Options.ReSendScope) * time.Second * 2).Format("2006-01-02 15:04:05")
	q := nm.Query().GT("created_at", timeLimit).In("status", []string{api.NOTIFICATION_STATUS_FAILED, api.NOTIFICATION_STATUS_PART_OK}).LT("send_times", options.Options.MaxSendTimes)
	ns := make([]SNotification, 0, 2)
	err := db.FetchModelObjects(nm, q, &ns)
	if err != nil {
		log.Errorf("fail to FetchModelObjects: %v", err)
		return
	}
	log.Infof("need to resend total %d notifications", len(ns))
	for i := range ns {
		task, err := taskman.TaskManager.NewTask(ctx, "NotificationSendTask", &ns[i], userCred, nil, "", "")
		if err != nil {
			log.Errorf("NotificationSendTask newTask error %v", err)
		} else {
			task.ScheduleRun(nil)
		}
	}
}

func (n *SNotification) TemplateStore() notifyv2.ITemplateStore {
	if len(n.EventId) == 0 || n.ContactType == api.MOBILE {
		return TemplateManager
	}
	return LocalTemplateManager
}
