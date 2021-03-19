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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/bitmap"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func parseEvent(es string) (notify.SEvent, error) {
	ess := strings.Split(es, notify.DelimiterInEvent)
	if len(ess) != 2 {
		return notify.SEvent{}, fmt.Errorf("invalid event string %q", es)
	}
	return notify.Event.WithResourceType(ess[0]).WithAction(notify.SAction(ess[1])), nil
}

type STopicManager struct {
	db.SStandaloneResourceBaseManager
	db.SEnabledResourceBaseManager
}

var TopicManager *STopicManager

func init() {
	TopicManager = &STopicManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			STopic{},
			"topic_tbl",
			"topic",
			"topics",
		),
	}
	TopicManager.SetVirtualObject(TopicManager)
}

type STopic struct {
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

func (sm *STopicManager) InitializeData() error {
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
	topics := make([]STopic, 0, initSNames.Len())
	err := db.FetchModelObjects(sm, q, &topics)
	if err != nil {
		return errors.Wrap(err, "unable to FetchModelObjects")
	}
	for i := range topics {
		t := &topics[i]
		initSNames.Delete(t.Name)
	}
	ctx := context.Background()
	for _, name := range initSNames.UnsortedList() {
		t := new(STopic)
		t.Name = name
		t.Enabled = tristate.True
		switch name {
		case DefaultResourceCreateDelete:
			t.addResources(
				notify.TOPIC_RESOURCE_SERVER,
				notify.TOPIC_RESOURCE_SCALINGGROUP,
				notify.TOPIC_RESOURCE_IMAGE,
				notify.TOPIC_RESOURCE_DISK,
				notify.TOPIC_RESOURCE_SNAPSHOT,
				notify.TOPIC_RESOURCE_INSTANCESNAPSHOT,
				notify.TOPIC_RESOURCE_SNAPSHOTPOLICY,
				notify.TOPIC_RESOURCE_NETWORK,
				notify.TOPIC_RESOURCE_EIP,
				notify.TOPIC_RESOURCE_LOADBALANCER,
				notify.TOPIC_RESOURCE_LOADBALANCERACL,
				notify.TOPIC_RESOURCE_LOADBALANCERCERTIFICATE,
				notify.TOPIC_RESOURCE_BUCKET,
				notify.TOPIC_RESOURCE_DBINSTANCE,
				notify.TOPIC_RESOURCE_ELASTICCACHE,
			)
			t.addAction(
				notify.ActionCreate,
				notify.ActionDelete,
				notify.ActionPendingDelete,
			)
			t.Type = notify.TOPIC_TYPE_RESOURCE
		case DefaultResourceChangeConfig:
			t.addResources(
				notify.TOPIC_RESOURCE_SERVER,
				notify.TOPIC_RESOURCE_DISK,
				notify.TOPIC_RESOURCE_DBINSTANCE,
				notify.TOPIC_RESOURCE_ELASTICCACHE,
			)
			t.addAction(notify.ActionChangeConfig)
			t.Type = notify.TOPIC_TYPE_RESOURCE
		case DefaultResourceUpdate:
			t.addResources(
				notify.TOPIC_RESOURCE_SERVER,
				notify.TOPIC_RESOURCE_DISK,
				notify.TOPIC_RESOURCE_DBINSTANCE,
				notify.TOPIC_RESOURCE_ELASTICCACHE,
			)
			t.addAction(notify.ActionUpdate)
			t.addAction(notify.ActionRebuildRoot)
			t.addAction(notify.ActionResetPassword)
			t.addAction(notify.ActionChangeIpaddr)
			t.Type = notify.TOPIC_TYPE_RESOURCE
		case DefaultResourceReleaseDue1Day:
			t.addResources(
				notify.TOPIC_RESOURCE_SERVER,
				notify.TOPIC_RESOURCE_DISK,
				notify.TOPIC_RESOURCE_EIP,
				notify.TOPIC_RESOURCE_LOADBALANCER,
				notify.TOPIC_RESOURCE_DBINSTANCE,
				notify.TOPIC_RESOURCE_ELASTICCACHE,
			)
			t.addAction(notify.ActionExpiredRelease)
			t.Type = notify.TOPIC_TYPE_RESOURCE
			t.AdvanceDays = 1
		case DefaultResourceReleaseDue3Day:
			t.addResources(
				notify.TOPIC_RESOURCE_SERVER,
				notify.TOPIC_RESOURCE_DISK,
				notify.TOPIC_RESOURCE_EIP,
				notify.TOPIC_RESOURCE_LOADBALANCER,
				notify.TOPIC_RESOURCE_DBINSTANCE,
				notify.TOPIC_RESOURCE_ELASTICCACHE,
			)
			t.addAction(notify.ActionExpiredRelease)
			t.Type = notify.TOPIC_TYPE_RESOURCE
			t.AdvanceDays = 3
		case DefaultScheduledTaskExecute:
			t.addResources(notify.TOPIC_RESOURCE_SCHEDULEDTASK)
			t.addAction(notify.ActionExecute)
			t.Type = notify.TOPIC_TYPE_AUTOMATED_PROCESS
		case DefaultScalingPolicyExecute:
			t.addResources(notify.TOPIC_RESOURCE_SCALINGPOLICY)
			t.addAction(notify.ActionExecute)
			t.Type = notify.TOPIC_TYPE_AUTOMATED_PROCESS
		case DefaultSnapshotPolicyExecute:
			t.addResources(notify.TOPIC_RESOURCE_SNAPSHOTPOLICY)
			t.addAction(notify.ActionExecute)
			t.Type = notify.TOPIC_TYPE_AUTOMATED_PROCESS
		}
		err := sm.TableSpec().Insert(ctx, t)
		if err != nil {
			return errors.Wrapf(err, "unable to insert %s", name)
		}
	}
	return nil
}

func (sm *STopicManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input notify.TopicListInput) (*sqlchemy.SQuery, error) {
	return sm.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StandaloneResourceListInput)
}

func (sm *STopicManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []notify.TopicDetails {
	sRows := sm.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	rows := make([]notify.TopicDetails, len(objs))
	for i := range rows {
		rows[i].StandaloneResourceDetails = sRows[i]
		ss := objs[i].(*STopic)
		rows[i].Resources = ss.getResources()
	}
	return rows
}

type SSubscriberDis struct {
	SSubscriber
	ReceiverName string `json:"receiver_name"`
	RoleName     string `json:"role_name"`
}

func (s *STopic) subscriptionReceiverDiss() ([]SSubscriberDis, error) {
	q := SubscriberManager.Query().Equals("subscription_id", s.Id)
	rq := ReceiverManager.Query("id", "name").SubQuery()
	roq := db.RoleCacheManager.Query("id", "name").SubQuery()
	q = q.LeftJoin(rq, sqlchemy.Equals(q.Field("receiver"), rq.Field("id")))
	q = q.LeftJoin(roq, sqlchemy.Equals(q.Field("receiver"), roq.Field("id")))
	// It looks strange, but the order of append cannot be changed
	q.AppendField(q.QueryFields()...)
	q.AppendField(rq.Field("name", "receiver_name"))
	q.AppendField(roq.Field("name", "role_name"))
	srs := make([]SSubscriberDis, 0)
	err := q.All(&srs)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch All")
	}
	return srs, nil
}

func (sm *STopicManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewForbiddenError("prohibit creation")
}

func (ss *STopic) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return input, httperrors.NewForbiddenError("update prohibited")
}

func (ss *STopic) ValidateDeleteCondition(ctx context.Context) error {
	return httperrors.NewForbiddenError("prohibit deletion")
}

func (s *STopic) addResources(resources ...string) {
	for _, resource := range resources {
		v := converter.resourceValue(resource)
		if v < 0 {
			continue
		}
		s.Resources += 1 << v
	}
}

func (s *STopic) addAction(actions ...notify.SAction) {
	for _, action := range actions {
		v := converter.actionValue(action)
		if v < 0 {
			continue
		}
		s.Actions += 1 << v
	}
}

func (s *STopic) getResources() []string {
	vs := bitmap.Uint64ToIntArray(s.Resources)
	resources := make([]string, 0, len(vs))
	for _, v := range vs {
		resources = append(resources, converter.resource(v))
	}
	return resources
}

func (s *STopic) getActions() []notify.SAction {
	vs := bitmap.Uint2IntArray(s.Actions)
	actions := make([]notify.SAction, 0, len(vs))
	for _, v := range vs {
		actions = append(actions, converter.action(v))
	}
	return actions
}

func (sm *STopicManager) TopicsByEvent(eventStr string, advanceDays int) ([]STopic, error) {
	event, err := parseEvent(eventStr)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse event %q", event)
	}
	resourceV := converter.resourceValue(event.ResourceType())
	actionV := converter.actionValue(event.Action())
	q := sm.Query().Equals("advance_days", advanceDays)
	q = q.Filter(sqlchemy.GT(sqlchemy.AND_Val("", q.Field("resources"), 1<<resourceV), 0))
	q = q.Filter(sqlchemy.GT(sqlchemy.AND_Val("", q.Field("actions"), 1<<actionV), 0))
	var topics []STopic
	err = db.FetchModelObjects(sm, q, &topics)
	if err != nil {
		q.DebugQuery()
		return nil, errors.Wrap(err, "unable to FetchModelObjects")
	}
	return topics, nil
}

func init() {
	converter = &sConverter{
		resourceValueMap: make(map[string]int, 5),
		resourceList:     make([]string, 0, 5),
		actionList:       make([]notify.SAction, 0, 5),
		actionValueMap:   make(map[notify.SAction]int, 5),
	}
	converter.registerResource(
		notify.TOPIC_RESOURCE_SERVER,
		notify.TOPIC_RESOURCE_SCALINGGROUP,
		notify.TOPIC_RESOURCE_SCALINGPOLICY,
		notify.TOPIC_RESOURCE_IMAGE,
		notify.TOPIC_RESOURCE_DISK,
		notify.TOPIC_RESOURCE_SNAPSHOT,
		notify.TOPIC_RESOURCE_INSTANCESNAPSHOT,
		notify.TOPIC_RESOURCE_SNAPSHOTPOLICY,
		notify.TOPIC_RESOURCE_NETWORK,
		notify.TOPIC_RESOURCE_EIP,
		notify.TOPIC_RESOURCE_SECGROUP,
		notify.TOPIC_RESOURCE_LOADBALANCER,
		notify.TOPIC_RESOURCE_LOADBALANCERACL,
		notify.TOPIC_RESOURCE_LOADBALANCERCERTIFICATE,
		notify.TOPIC_RESOURCE_BUCKET,
		notify.TOPIC_RESOURCE_DBINSTANCE,
		notify.TOPIC_RESOURCE_ELASTICCACHE,
		notify.TOPIC_RESOURCE_SCHEDULEDTASK,
	)
	converter.registerAction(
		notify.ActionCreate,
		notify.ActionDelete,
		notify.ActionPendingDelete,
		notify.ActionUpdate,
		notify.ActionRebuildRoot,
		notify.ActionResetPassword,
		notify.ActionChangeConfig,
		notify.ActionExpiredRelease,
		notify.ActionExecute,
		notify.ActionChangeIpaddr,
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
