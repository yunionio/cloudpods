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

	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/bitmap"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func parseEvent(es string) (notify.SEvent, error) {
	ess := strings.Split(es, notify.DelimiterInEvent)
	if len(ess) != 2 {
		return notify.SEvent{}, fmt.Errorf("invalid event string %q", es)
	}
	return notify.Event.WithResourceType(ess[0]).WithAction(notify.SAction(ess[1])), nil
}

type SSubscriptionManager struct {
	db.SStandaloneResourceBaseManager
	db.SEnabledResourceBaseManager
}

var SubscriptionManager *SSubscriptionManager

func init() {
	SubscriptionManager = &SSubscriptionManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SSubscription{},
			"subscription_tbl",
			"subscription",
			"subscriptions",
		),
	}
	SubscriptionManager.SetVirtualObject(SubscriptionManager)
}

type SSubscription struct {
	db.SStandaloneResourceBase
	db.SEnabledResourceBase

	Type        string `width:"20" nullable:"false" create:"required" update:"user" list:"user"`
	Resources   uint64 `nullable:"false"`
	Actions     uint32 `nullable:"false"`
	AdvanceDays int    `nullable:"false"`
}

const (
	DefaultResourceCreateDelete   = "resource create or delete"
	DefaultResourceChangeConfig   = "resource change config"
	DefaultResourceUpdate         = "resource update"
	DefaultResourceReleaseDue1Day = "resource release due 1 day"
	DefaultResourceReleaseDue3Day = "resource release due 3 day"
	DefaultScheduledTaskExecute   = "scheduled task execute"
	DefaultScalingPolicyExecute   = "scaling policy execute"
	DefaultSnapshotPolicyExecute  = "snapshot policy execute"
)

func (sm *SSubscriptionManager) InitializeData() error {
	initSNames := sets.NewString(
		DefaultResourceCreateDelete,
		DefaultResourceChangeConfig,
		DefaultResourceUpdate,
		DefaultResourceReleaseDue1Day,
		DefaultResourceReleaseDue3Day,
		DefaultScheduledTaskExecute,
		DefaultScalingPolicyExecute,
		DefaultSnapshotPolicyExecute,
	)
	q := sm.Query()
	subscriptions := make([]SSubscription, 0, initSNames.Len())
	err := db.FetchModelObjects(sm, q, &subscriptions)
	if err != nil {
		return errors.Wrap(err, "unable to FetchModelObjects")
	}
	for i := range subscriptions {
		ss := &subscriptions[i]
		initSNames.Delete(ss.Name)
	}
	ctx := context.Background()
	for _, name := range initSNames.UnsortedList() {
		ss := new(SSubscription)
		ss.Name = name
		switch name {
		case DefaultResourceCreateDelete:
			ss.addResources(
				notify.SUBSCRIPTION_RESOURCE_SERVER,
				notify.SUBSCRIPTION_RESOURCE_SCALINGGROUP,
				notify.SUBSCRIPTION_RESOURCE_IMAGE,
				notify.SUBSCRIPTION_RESOURCE_DISK,
				notify.SUBSCRIPTION_RESOURCE_SNAPSHOT,
				notify.SUBSCRIPTION_RESOURCE_INSTANCESNAPSHOT,
				notify.SUBSCRIPTION_RESOURCE_SNAPSHOTPOLICY,
				notify.SUBSCRIPTION_RESOURCE_NETWORK,
				notify.SUBSCRIPTION_RESOURCE_EIP,
				notify.SUBSCRIPTION_RESOURCE_LOADBALANCER,
				notify.SUBSCRIPTION_RESOURCE_LOADBALANCERACL,
				notify.SUBSCRIPTION_RESOURCE_LOADBALANCERCERTIFICATE,
				notify.SUBSCRIPTION_RESOURCE_BUCKET,
				notify.SUBSCRIPTION_RESOURCE_DBINSTANCE,
				notify.SUBSCRIPTION_RESOURCE_ELASTICCACHE,
			)
			ss.addAction(
				notify.ActionCreate,
				notify.ActionDelete,
				notify.ActionPendingDelete,
			)
			ss.Type = notify.SUBSCRIPTION_TYPE_RESOURCE
		case DefaultResourceChangeConfig:
			ss.addResources(
				notify.SUBSCRIPTION_RESOURCE_SERVER,
				notify.SUBSCRIPTION_RESOURCE_DISK,
				notify.SUBSCRIPTION_RESOURCE_DBINSTANCE,
				notify.SUBSCRIPTION_RESOURCE_ELASTICCACHE,
			)
			ss.addAction(notify.ActionChangeConfig)
			ss.Type = notify.SUBSCRIPTION_TYPE_RESOURCE
		case DefaultResourceUpdate:
			ss.addResources(
				notify.SUBSCRIPTION_RESOURCE_SERVER,
				notify.SUBSCRIPTION_RESOURCE_DISK,
				notify.SUBSCRIPTION_RESOURCE_DBINSTANCE,
				notify.SUBSCRIPTION_RESOURCE_ELASTICCACHE,
			)
			ss.addAction(notify.ActionUpdate)
			ss.Type = notify.SUBSCRIPTION_TYPE_RESOURCE
		case DefaultResourceReleaseDue1Day:
			ss.addResources(
				notify.SUBSCRIPTION_RESOURCE_SERVER,
				notify.SUBSCRIPTION_RESOURCE_DISK,
				notify.SUBSCRIPTION_RESOURCE_EIP,
				notify.SUBSCRIPTION_RESOURCE_LOADBALANCER,
				notify.SUBSCRIPTION_RESOURCE_DBINSTANCE,
				notify.SUBSCRIPTION_RESOURCE_ELASTICCACHE,
			)
			ss.addAction(notify.ActionExpiredRelease)
			ss.Type = notify.SUBSCRIPTION_TYPE_RESOURCE
			ss.AdvanceDays = 1
		case DefaultResourceReleaseDue3Day:
			ss.addResources(
				notify.SUBSCRIPTION_RESOURCE_SERVER,
				notify.SUBSCRIPTION_RESOURCE_DISK,
				notify.SUBSCRIPTION_RESOURCE_EIP,
				notify.SUBSCRIPTION_RESOURCE_LOADBALANCER,
				notify.SUBSCRIPTION_RESOURCE_DBINSTANCE,
				notify.SUBSCRIPTION_RESOURCE_ELASTICCACHE,
			)
			ss.addAction(notify.ActionExpiredRelease)
			ss.Type = notify.SUBSCRIPTION_TYPE_RESOURCE
			ss.AdvanceDays = 3
		case DefaultScheduledTaskExecute:
			ss.addResources(notify.SUBSCRIPTION_RESOURCE_SCHEDULEDTASK)
			ss.addAction(notify.ActionExecute)
			ss.Type = notify.SUBSCRIPTION_TYPE_AUTOMATED_PROCESS
		case DefaultScalingPolicyExecute:
			ss.addResources(notify.SUBSCRIPTION_RESOURCE_SCALINGPOLICY)
			ss.addAction(notify.ActionExecute)
			ss.Type = notify.SUBSCRIPTION_TYPE_AUTOMATED_PROCESS
		case DefaultSnapshotPolicyExecute:
			ss.addResources(notify.SUBSCRIPTION_RESOURCE_SNAPSHOTPOLICY)
			ss.addAction(notify.ActionExecute)
			ss.Type = notify.SUBSCRIPTION_TYPE_AUTOMATED_PROCESS
		}
		err := sm.TableSpec().Insert(ctx, ss)
		if err != nil {
			return errors.Wrapf(err, "unable to insert %s", name)
		}
	}
	return nil
}

func (sm *SSubscriptionManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input notify.SubscriptionListInput) (*sqlchemy.SQuery, error) {
	return q, nil
}

func (sm *SSubscriptionManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []notify.SubscriptionDetails {
	sRows := sm.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	rows := make([]notify.SubscriptionDetails, len(objs))
	for i := range rows {
		rows[i].StandaloneResourceDetails = sRows[i]
		ss := objs[i].(*SSubscription)
		rows[i].Resources = ss.getResources()
		srs, err := ss.subscriptionReceiverDiss()
		if err != nil {
			log.Errorf("unable to get subscriptionReceivers: %v", err)
		}
		for j := range srs {
			sr := &srs[j]
			switch sr.ReceiverType {
			case ReceiverNormal:
				rows[i].Receivers.Receivers = append(rows[i].Receivers.Receivers, notify.IDAndName{
					ID:   sr.Receiver,
					Name: sr.ReceiverName,
				})
			case ReceiverRole:
				rows[i].Receivers.ReceivingRoles = append(rows[i].Receivers.ReceivingRoles, notify.ReceivingRoleIDAndName{
					IDAndName: notify.IDAndName{
						ID:   sr.Receiver,
						Name: sr.RoleName,
					},
					Scope: sr.RoleScope,
				})
			case ReceiverDingtalkRobot, ReceiverFeishuRobot, ReceiverWorkwxRobot:
				rows[i].Robot = sr.ReceiverType
			case ReceiverWeebhook:
				rows[i].Webhook = sr.ReceiverType
			}
		}
	}
	return rows
}

type SSubscriptionReceiverDis struct {
	SSubscriptionReceiver
	ReceiverName string `json:"receiver_name"`
	RoleName     string `json:"role_name"`
}

func (s *SSubscription) subscriptionReceiverDiss() ([]SSubscriptionReceiverDis, error) {
	q := SubscriptionReceiverManager.Query().Equals("subscription_id", s.Id)
	rq := ReceiverManager.Query("id", "name").SubQuery()
	roq := db.RoleCacheManager.Query("id", "name").SubQuery()
	q = q.LeftJoin(rq, sqlchemy.Equals(q.Field("receiver"), rq.Field("id")))
	q = q.LeftJoin(roq, sqlchemy.Equals(q.Field("receiver"), roq.Field("id")))
	// It looks strange, but the order of append cannot be changed
	q.AppendField(q.QueryFields()...)
	q.AppendField(rq.Field("name", "receiver_name"))
	q.AppendField(roq.Field("name", "role_name"))
	srs := make([]SSubscriptionReceiverDis, 0)
	err := q.All(&srs)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch All")
	}
	return srs, nil
}

func (sm *SSubscriptionManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewForbiddenError("prohibit creation")
}

func (ss *SSubscription) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return input, httperrors.NewForbiddenError("update prohibited")
}

func (ss *SSubscription) ValidateDeleteCondition(ctx context.Context) error {
	return httperrors.NewForbiddenError("prohibit deletion")
}

func (ss *SSubscription) AllowPerformSetReceiver(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, ss, "set-receiver")
}

func (ss *SSubscription) PerformSetReceiver(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input notify.SubscriptionSetReceiverInput) (jsonutils.JSONObject, error) {
	// check input
	errgo, _ := errgroup.WithContext(ctx)
	// check receiving roles
	validScopes := []string{string(rbacutils.ScopeSystem), string(rbacutils.ScopeDomain), string(rbacutils.ScopeProject)}
	for i := range input.ReceivingRoles {
		role := input.ReceivingRoles[i]
		index := i
		errgo.Go(func() error {
			if len(role.Scope) == 0 {
				return httperrors.NewInputParameterError("empty scope for role %q", role.Role)
			}
			if !utils.IsInStringArray(role.Scope, validScopes) {
				return httperrors.NewInputParameterError("invalid scope %q for role %q, need %s, %s or %s", role.Scope, role.Role, rbacutils.ScopeSystem, rbacutils.ScopeDomain, rbacutils.ScopeProject)
			}
			roleCache, err := db.RoleCacheManager.FetchRoleByIdOrName(ctx, role.Role)
			if err != nil {
				return errors.Wrapf(err, "unable find role %q", role)
			}
			input.ReceivingRoles[index].Role = roleCache.GetId()
			return nil
		})
	}
	err := errgo.Wait()
	if err != nil {
		return nil, err
	}
	receivers, err := ReceiverManager.FetchByIdOrNames(ctx, input.Receivers...)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch Receivers")
	}
	reSet := sets.NewString(input.Receivers...)
	reIds := make([]string, len(receivers))
	for i := range receivers {
		reSet.Delete(receivers[i].GetId())
		reSet.Delete(receivers[i].GetName())
		reIds[i] = receivers[i].GetId()
	}
	if reSet.Len() > 0 {
		return nil, httperrors.NewInputParameterError("receivers %q not found", strings.Join(reSet.UnsortedList(), ", "))
	}
	input.Receivers = reIds

	// deal with subscriptionReceivers
	srs, err := SubscriptionReceiverManager.findReceivers(ss.Id, ReceiverNormal, ReceiverRole)
	if err != nil {
		return nil, errors.Wrap(err, "unable to findReceivers")
	}
	reSet = sets.NewString(input.Receivers...)
	reRoleSet := make(map[notify.ReceivingRole]struct{})
	for _, role := range input.ReceivingRoles {
		reRoleSet[role] = struct{}{}
	}
	for i := range srs {
		rs := &srs[i]
		switch rs.ReceiverType {
		case ReceiverNormal:
			if !reSet.Has(rs.Receiver) {
				err := rs.Delete(ctx, userCred)
				if err != nil {
					return nil, errors.Wrapf(err, "unable to delete receiver %s", rs.Receiver)
				}
			}
			reSet.Delete(rs.Receiver)
		case ReceiverRole:
			role := rs.receivingRole()
			if _, ok := reRoleSet[role]; !ok {
				err := rs.Delete(ctx, userCred)
				if err != nil {
					return nil, errors.Wrapf(err, "unable to delete receiver %s", rs.Receiver)
				}
			}
			delete(reRoleSet, role)
		}
	}
	for _, re := range reSet.UnsortedList() {
		_, err := SubscriptionReceiverManager.create(ctx, ss.Id, re, ReceiverNormal, "")
		if err != nil {
			return nil, errors.Wrapf(err, "unable to create receiver %s", re)
		}
	}
	for role := range reRoleSet {
		_, err := SubscriptionReceiverManager.create(ctx, ss.Id, role.Role, ReceiverRole, role.Scope)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to create role %s", role)
		}
	}

	return nil, nil
}

func (ss *SSubscription) AllowPerformSetRobot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, ss, "set-robot")
}

func (ss *SSubscription) PerformSetRobot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input notify.SubscriptionSetRobotInput) (jsonutils.JSONObject, error) {
	err := ss.setSingleReceiver(ctx, input.Robot, ReceiverDingtalkRobot, ReceiverFeishuRobot, ReceiverWorkwxRobot)
	if errors.Cause(err) == errors.ErrNotFound {
		return nil, httperrors.NewInputParameterError("unkown robot %q", input.Robot)
	}
	return nil, err
}

func (ss *SSubscription) AllowPerformSetWebhook(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, ss, "set-webhook")
}

func (ss *SSubscription) PerformSetWebhook(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input notify.SubscriptionSetWebhookInput) (jsonutils.JSONObject, error) {
	err := ss.setSingleReceiver(ctx, input.Webhook, ReceiverWeebhook)
	if errors.Cause(err) == errors.ErrNotFound {
		return nil, httperrors.NewInputParameterError("unkown webhook %q", input.Webhook)
	}
	return nil, err
}

func (ss *SSubscription) setSingleReceiver(ctx context.Context, re string, reTypes ...string) error {
	receiverRobot := reTypes
	if !utils.IsInStringArray(re, receiverRobot) {
		return errors.ErrNotFound
	}
	srs, err := SubscriptionReceiverManager.findReceivers(ss.Id, receiverRobot...)
	if err != nil {
		return errors.Wrap(err, "unable to findReceivers")
	}
	if len(srs) == 0 {
		// create one
		_, err := SubscriptionReceiverManager.create(ctx, ss.Id, "", re, "")
		return err
	}
	if len(srs) > 1 {
		return fmt.Errorf("multi robot receiver")
	}
	// update
	sr := &srs[0]
	_, err = db.Update(sr, func() error {
		sr.ReceiverType = re
		return nil
	})
	return err
}

func (s *SSubscription) addResources(resources ...string) {
	for _, resource := range resources {
		v := converter.resourceValue(resource)
		if v < 0 {
			continue
		}
		s.Resources += 1 << v
	}
}

func (s *SSubscription) addAction(actions ...notify.SAction) {
	for _, action := range actions {
		v := converter.actionValue(action)
		if v < 0 {
			continue
		}
		s.Actions += 1 << v
	}
}

func (s *SSubscription) getResources() []string {
	vs := bitmap.Uint64ToIntArray(s.Resources)
	resources := make([]string, 0, len(vs))
	for _, v := range vs {
		resources = append(resources, converter.resource(v))
	}
	return resources
}

func (s *SSubscription) getActions() []notify.SAction {
	vs := bitmap.Uint2IntArray(s.Actions)
	actions := make([]notify.SAction, 0, len(vs))
	for _, v := range vs {
		actions = append(actions, converter.action(v))
	}
	return actions
}

func (sm *SSubscriptionManager) SubsciptionByEvent(eventStr string, advanceDays int) ([]SSubscription, error) {
	event, err := parseEvent(eventStr)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse event %q", event)
	}
	resourceV := converter.resourceValue(event.ResourceType())
	actionV := converter.actionValue(event.Action())
	q := sm.Query().Equals("advance_days", advanceDays)
	q = q.Filter(sqlchemy.GT(sqlchemy.AND_Val("", q.Field("resources"), 1<<resourceV), 0))
	q = q.Filter(sqlchemy.GT(sqlchemy.AND_Val("", q.Field("actions"), 1<<actionV), 0))
	var subscriptions []SSubscription
	err = db.FetchModelObjects(sm, q, &subscriptions)
	if err != nil {
		q.DebugQuery()
		return nil, errors.Wrap(err, "unable to FetchModelObjects")
	}
	return subscriptions, nil
}

func init() {
	converter = &sConverter{
		resourceValueMap: make(map[string]int, 5),
		resourceList:     make([]string, 0, 5),
		actionList:       make([]notify.SAction, 0, 5),
		actionValueMap:   make(map[notify.SAction]int, 5),
	}
	converter.registerResource(
		notify.SUBSCRIPTION_RESOURCE_SERVER,
		notify.SUBSCRIPTION_RESOURCE_SCALINGGROUP,
		notify.SUBSCRIPTION_RESOURCE_SCALINGPOLICY,
		notify.SUBSCRIPTION_RESOURCE_IMAGE,
		notify.SUBSCRIPTION_RESOURCE_DISK,
		notify.SUBSCRIPTION_RESOURCE_SNAPSHOT,
		notify.SUBSCRIPTION_RESOURCE_INSTANCESNAPSHOT,
		notify.SUBSCRIPTION_RESOURCE_SNAPSHOTPOLICY,
		notify.SUBSCRIPTION_RESOURCE_NETWORK,
		notify.SUBSCRIPTION_RESOURCE_EIP,
		notify.SUBSCRIPTION_RESOURCE_SECGROUP,
		notify.SUBSCRIPTION_RESOURCE_LOADBALANCER,
		notify.SUBSCRIPTION_RESOURCE_LOADBALANCERACL,
		notify.SUBSCRIPTION_RESOURCE_LOADBALANCERCERTIFICATE,
		notify.SUBSCRIPTION_RESOURCE_BUCKET,
		notify.SUBSCRIPTION_RESOURCE_DBINSTANCE,
		notify.SUBSCRIPTION_RESOURCE_ELASTICCACHE,
		notify.SUBSCRIPTION_RESOURCE_SCHEDULEDTASK,
	)
	converter.registerAction(
		notify.ActionCreate,
		notify.ActionDelete,
		notify.ActionPendingDelete,
		notify.ActionUpdate,
		notify.ActionChangeConfig,
		notify.ActionExpiredRelease,
		notify.ActionExecute,
	)
}

var converter *sConverter

type sConverter struct {
	resourceValueMap map[string]int
	resourceList     []string
	actionValueMap   map[notify.SAction]int
	actionList       []notify.SAction
}

func (rc *sConverter) registerResource(resources ...string) {
	for _, resource := range resources {
		if _, ok := rc.resourceValueMap[resource]; ok {
			return
		}
		rc.resourceList = append(rc.resourceList, resource)
		rc.resourceValueMap[resource] = len(rc.resourceList) - 1
	}
}

func (rc *sConverter) registerAction(actions ...notify.SAction) {
	for _, action := range actions {
		if _, ok := rc.actionValueMap[action]; ok {
			return
		}
		rc.actionList = append(rc.actionList, action)
		rc.actionValueMap[action] = len(rc.actionList) - 1
	}
}

func (rc *sConverter) resourceValue(resource string) int {
	v, ok := rc.resourceValueMap[resource]
	if !ok {
		return -1
	}
	return v
}

func (rc *sConverter) resource(resourceValue int) string {
	if resourceValue < 0 || resourceValue >= len(rc.resourceList) {
		return ""
	}
	return rc.resourceList[resourceValue]
}

func (rc *sConverter) actionValue(action notify.SAction) int {
	v, ok := rc.actionValueMap[action]
	if !ok {
		return -1
	}
	return v
}

func (rc *sConverter) action(actionValue int) notify.SAction {
	if actionValue < 0 || actionValue >= len(rc.actionList) {
		return notify.SAction("")
	}
	return rc.actionList[actionValue]
}
