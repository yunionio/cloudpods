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
	"sort"
	"strings"

	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/sets"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/notify/options"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var SubscriberManager *SSubscriberManager

func init() {
	SubscriberManager = &SSubscriberManager{
		SStandaloneAnonResourceBaseManager: db.NewStandaloneAnonResourceBaseManager(
			SSubscriber{},
			"subscriber_tbl",
			"subscriber",
			"subscribers",
		),
	}
	SubscriberManager.SetVirtualObject(SubscriberManager)
}

type SSubscriberManager struct {
	db.SStandaloneAnonResourceBaseManager
	db.SEnabledResourceBaseManager
}

// 消息订阅接收人
type SSubscriber struct {
	db.SStandaloneAnonResourceBase
	db.SEnabledResourceBase

	TopicId                 string `width:"128" charset:"ascii" nullable:"false" index:"true" get:"user" list:"user" create:"required"`
	Type                    string `width:"16" charset:"ascii" nullable:"false" index:"true" get:"user" list:"user" create:"required"`
	Identification          string `width:"128" charset:"ascii" nullable:"false" index:"true"`
	RoleScope               string `width:"8" charset:"ascii" nullable:"false" get:"user" list:"user" create:"optional"`
	ResourceScope           string `width:"8" charset:"ascii" nullable:"false" get:"user" list:"user" create:"required"`
	ResourceAttributionId   string `width:"128" charset:"ascii" nullable:"false" get:"user" list:"user" create:"optional"`
	ResourceAttributionName string `width:"128" charset:"utf8" list:"user" create:"optional"`
	Scope                   string `width:"128" charset:"ascii" nullable:"false" create:"required"`
	DomainId                string `width:"128" charset:"ascii" nullable:"false" create:"optional"`
	// minutes
	GroupTimes uint32 `nullable:"true" list:"user"  update:"user"`
}

func (sm *SSubscriberManager) validateReceivers(ctx context.Context, receivers []string) ([]string, error) {
	rs, err := ReceiverManager.FetchByIdOrNames(ctx, receivers...)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch Receivers")
	}
	reSet := sets.NewString(receivers...)
	reIds := make([]string, len(rs))
	for i := range rs {
		reSet.Delete(rs[i].GetId())
		reSet.Delete(rs[i].GetName())
		reIds[i] = rs[i].GetId()
	}
	if reSet.Len() > 0 {
		return nil, httperrors.NewInputParameterError("receivers %q not found", strings.Join(reSet.UnsortedList(), ", "))
	}
	return reIds, nil
}

func (self *SSubscriber) GetEnabledReceivers() ([]SReceiver, error) {
	q := ReceiverManager.Query().IsTrue("enabled")
	sq := SubscriberReceiverManager.Query().SubQuery()
	q = q.Join(sq, sqlchemy.Equals(q.Field("id"), sq.Field("receiver_id"))).Filter(sqlchemy.Equals(sq.Field("subscriber_id"), self.Id))
	ret := []SReceiver{}
	return ret, db.FetchModelObjects(ReceiverManager, q, &ret)
}

func (self *SSubscriber) GetRobot() (*SRobot, error) {
	robot, err := RobotManager.FetchById(self.Identification)
	if err != nil {
		return nil, errors.Wrapf(err, "RobotManager.FetchById(%s)", self.Identification)
	}
	return robot.(*SRobot), nil
}

func (sm *SSubscriberManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.SubscriberCreateInput) (api.SubscriberCreateInput, error) {
	var err error
	// permission check
	sSystem, sDomain := string(rbacscope.ScopeSystem), string(rbacscope.ScopeDomain)
	switch input.Scope {
	case sSystem:
		allow := db.IsAdminAllowCreate(userCred, sm)
		if allow.Result.IsDeny() {
			return input, httperrors.NewForbiddenError("The scope %s and the role of the operator do not match", input.Scope)
		}
	case sDomain:
		allow := db.IsDomainAllowCreate(userCred, sm)
		if allow.Result.IsDeny() {
			return input, httperrors.NewForbiddenError("The scope %s and the role of the operator do not match", input.Scope)
		}
	default:
		return input, httperrors.NewInputParameterError("unknown scope %s", input.Scope)
	}
	input.StandaloneAnonResourceCreateInput, err = sm.SStandaloneAnonResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneAnonResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBaseManager.ValidateCreateData")
	}
	// check topic
	t, err := TopicManager.FetchById(input.TopicID)
	if err != nil {
		return input, errors.Wrapf(err, "unable to fetch topic %s", input.TopicID)
	}
	// check resource scope
	if !utils.IsInStringArray(input.ResourceScope, []string{api.SUBSCRIBER_SCOPE_SYSTEM, api.SUBSCRIBER_SCOPE_DOMAIN, api.SUBSCRIBER_SCOPE_PROJECT}) {
		return input, httperrors.NewInputParameterError("unknown resource_scope %q", input.ResourceScope)
	}
	// resource Attribution Id
	var domainId string
	switch input.ResourceScope {
	case api.SUBSCRIBER_SCOPE_SYSTEM:
		input.ResourceAttributionId = ""
		input.DomainId = ""
	case api.SUBSCRIBER_SCOPE_PROJECT:
		tenant, err := db.TenantCacheManager.FetchTenantById(ctx, input.ResourceAttributionId)
		if err != nil {
			return input, errors.Wrapf(err, "unable to fetch project %s", input.ResourceAttributionId)
		}
		domainId = tenant.DomainId
		input.DomainId = domainId
		input.ResourceAttributionId = tenant.GetId()
		input.ResourceAttributionName = tenant.GetName()
	case api.SUBSCRIBER_SCOPE_DOMAIN:
		tenant, err := db.TenantCacheManager.FetchDomainByIdOrName(ctx, input.ResourceAttributionId)
		if err != nil {
			return input, errors.Wrapf(err, "unable to fetch domain %s", input.ResourceAttributionId)
		}
		domainId = tenant.Id
		input.DomainId = domainId
		input.ResourceAttributionId = tenant.Id
		input.ResourceAttributionName = tenant.Name
	}
	if input.Scope == sDomain && domainId != userCred.GetProjectDomainId() {
		return input, httperrors.NewForbiddenError("domain %s admin can't create subscriber for domain %s", userCred.GetProjectDomainId(), domainId)
	}

	var checkQuery *sqlchemy.SQuery
	input.TopicID = t.GetId()
	switch input.Type {
	case api.SUBSCRIBER_TYPE_RECEIVER:
		reIds, err := sm.validateReceivers(ctx, input.Receivers)
		if err != nil {
			return input, err
		}
		input.Receivers = reIds
	case api.SUBSCRIBER_TYPE_ROLE:
		if input.RoleScope == "" {
			input.RoleScope = input.ResourceScope
		}
		roleCache, err := db.RoleCacheManager.FetchRoleByIdOrName(ctx, input.Role)
		if err != nil {
			return input, errors.Wrapf(err, "unable find role %q", input.Role)
		}
		input.Role = roleCache.GetId()
		checkQuery = sm.Query().Equals("topic_id", input.TopicID).Equals("type", api.SUBSCRIBER_TYPE_ROLE).Equals("resource_scope", input.ResourceScope).Equals("identification", input.Role).Equals("role_scope", input.RoleScope)
	case api.SUBSCRIBER_TYPE_ROBOT:
		robot, err := RobotManager.FetchByIdOrName(ctx, userCred, input.Robot)
		if errors.Cause(err) == sql.ErrNoRows {
			return input, httperrors.NewInputParameterError("robot %q not found", input.Robot)
		}
		if err != nil {
			return input, errors.Wrapf(err, "unable to fetch robot %q", input.Robot)
		}
		input.Robot = robot.GetId()
		checkQuery = sm.Query().Equals("type", api.SUBSCRIBER_TYPE_ROLE).Equals("topic_id", input.TopicID).Equals("resource_scope", input.ResourceScope).Equals("identification", input.Robot)
	default:
		return input, httperrors.NewInputParameterError("unkown type %q", input.Type)
	}
	// check type+resourceScope+identification
	if checkQuery != nil {
		count, err := checkQuery.CountWithError()
		if err != nil {
			return input, errors.Wrap(err, "unable to count")
		}
		if count > 0 {
			return input, httperrors.NewForbiddenError("repeated with existing subscribers")
		}
	}
	if input.GroupTimes != nil {
		if *input.GroupTimes < 0 {
			return input, httperrors.NewInputParameterError("invalidate group_times %d", input.GroupTimes)
		}
	}
	return input, nil
}

func (s *SSubscriber) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	s.SStandaloneAnonResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	var input api.SubscriberCreateInput
	_ = data.Unmarshal(&input)
	if s.Type == api.SUBSCRIBER_TYPE_RECEIVER {
		err := s.SetReceivers(ctx, input.Receivers)
		if err != nil {
			logclient.AddActionLogWithContext(ctx, s, logclient.ACT_CREATE, err.Error(), userCred, false)
			_, err := db.Update(s, func() error {
				s.SetEnabled(false)
				return nil
			})
			if err != nil {
				log.Errorf("unable to enable subscriber: %v", err)
			}
		}
	}
	logclient.AddActionLogWithContext(ctx, s, logclient.ACT_CREATE, "", userCred, true)
	return
}

func (s *SSubscriber) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	err := s.SStandaloneAnonResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
	if err != nil {
		return errors.Wrap(err, "SVirtualResourceBase.CustomizeCreate")
	}
	var input api.SubscriberCreateInput
	_ = data.Unmarshal(&input)
	switch input.Type {
	case api.SUBSCRIBER_TYPE_RECEIVER:
	case api.SUBSCRIBER_TYPE_ROBOT:
		s.Identification = input.Robot
	case api.SUBSCRIBER_TYPE_ROLE:
		s.Identification = input.Role
	}
	s.Enabled = tristate.True
	return nil
}

func (s *SSubscriber) PerformChange(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SubscriberChangeInput) (jsonutils.JSONObject, error) {
	if s.Scope == string(rbacscope.ScopeSystem) {
		if !db.IsAdminAllowUpdate(ctx, userCred, s) {
			return nil, httperrors.NewForbiddenError("")
		}
	} else {
		if !db.IsDomainAllowUpdate(ctx, userCred, s) {
			return nil, httperrors.NewForbiddenError("")
		}
		if s.DomainId != userCred.GetProjectDomainId() {
			return nil, httperrors.NewForbiddenError("")
		}
	}
	switch s.Type {
	case api.SUBSCRIBER_TYPE_RECEIVER:
		err := s.SetReceivers(ctx, input.Receivers)
		if err != nil {
			log.Errorf("unable to set receivers %s", input.Receivers)
		}
	case api.SUBSCRIBER_TYPE_ROBOT:
		_, err := db.Update(s, func() error {
			s.Identification = input.Robot
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "unable to update subscriber")
		}
	case api.SUBSCRIBER_TYPE_ROLE:
		_, err := db.Update(s, func() error {
			s.Identification = input.Role
			if input.RoleScope != "" {
				s.RoleScope = input.RoleScope
			}
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "unable to update subscriber")
		}
	}
	if input.GroupTimes != nil {
		_, err := db.Update(s, func() error {
			s.GroupTimes = *input.GroupTimes
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "unable to update subscriber group_times")
		}
	}

	return nil, nil
}

func (sm *SSubscriberManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.SubscriberListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = sm.SStandaloneAnonResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StandaloneAnonResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = sm.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, input.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	sSystem, sDomain := string(rbacscope.ScopeSystem), string(rbacscope.ScopeDomain)
	if input.Scope == "" {
		input.Scope = sSystem
	}
	switch input.Scope {
	case sSystem:
		allow := db.IsAdminAllowList(userCred, sm)
		if allow.Result.IsDeny() {
			return nil, httperrors.NewForbiddenError("")
		}
	case sDomain:
		allow := db.IsDomainAllowList(userCred, sm)
		if allow.Result.IsDeny() {
			return nil, httperrors.NewForbiddenError("")
		}
		q = q.Equals("domain_id", userCred.GetProjectDomainId())
	default:
		return nil, httperrors.NewInputParameterError("unkown scope %s", input.Scope)
	}
	if input.TopicID != "" {
		q = q.Equals("topic_id", input.TopicID)
	}
	if input.Type != "" {
		q = q.Equals("type", input.Type)
	}
	if input.ResourceScope != "" {
		q = q.Equals("resource_scope", input.ResourceScope)
	}
	return q, nil
}

func (sm *SSubscriberManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.SubscriberDetails {
	var err error
	vRows := sm.SStandaloneAnonResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	rows := make([]api.SubscriberDetails, len(objs))
	for i := range rows {
		rows[i].StandaloneAnonResourceDetails = vRows[i]
		s := objs[i].(*SSubscriber)
		switch s.Type {
		case api.SUBSCRIBER_TYPE_RECEIVER:
			rows[i].Receivers, err = s.receiverIdentifications()
			if err != nil {
				log.Errorf("unable to get receiverIdentifications for subscriber %q: %v", s.Id, err)
			}
		case api.SUBSCRIBER_TYPE_ROBOT:
			rows[i].Robot, err = s.robotIdentification()
			if err != nil {
				log.Errorf("unable get robotIdentification for subscriber %q: %v", s.Id, err)
			}
		case api.SUBSCRIBER_TYPE_ROLE:
			rows[i].Role, err = s.roleIdentification(ctx)
			if err != nil {
				log.Errorf("unable to get roleIdentification for subscriber %q: %v", s.Id, err)
			}
		}
	}
	return rows
}

func (s *SSubscriber) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	err := s.SStandaloneAnonResourceBase.CustomizeDelete(ctx, userCred, query, data)
	if err != nil {
		return err
	}
	if s.Scope == string(rbacscope.ScopeSystem) {
		if !db.IsAdminAllowDelete(ctx, userCred, s) {
			return httperrors.NewForbiddenError("")
		}
	} else {
		if !db.IsDomainAllowDelete(ctx, userCred, s) {
			return httperrors.NewForbiddenError("")
		}
		if s.DomainId != userCred.GetProjectDomainId() {
			return httperrors.NewForbiddenError("")
		}
	}
	return nil
}

func (s *SSubscriber) receiverIdentifications() ([]api.Identification, error) {
	srSubq := SubscriberReceiverManager.Query().Equals("subscriber_id", s.Id).SubQuery()
	rq := ReceiverManager.Query("id", "name")
	rq = rq.Join(srSubq, sqlchemy.Equals(srSubq.Field("receiver_id"), rq.Field("id")))
	var ret []api.Identification
	err := rq.All(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *SSubscriber) robotIdentification() (api.Identification, error) {
	var ret api.Identification
	q := RobotManager.Query("id", "name").Equals("id", s.Identification)
	err := q.First(&ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func (s *SSubscriber) roleIdentification(ctx context.Context) (api.Identification, error) {
	var ret api.Identification
	roleCache, err := db.RoleCacheManager.FetchRoleById(ctx, s.Identification)
	if err != nil {
		return ret, errors.Wrapf(err, "unable to find role %q", s.Identification)
	}
	ret.ID = s.Identification
	ret.Name = roleCache.Name
	return ret, nil
}

func (srm *SSubscriberManager) robot(tid, projectDomainId, projectId string) (map[string]uint32, error) {
	srs, err := srm.findSuitableOnes(tid, projectDomainId, projectId, api.SUBSCRIBER_TYPE_ROBOT)
	if err != nil {
		return nil, err
	}
	robotIds := make(map[string]uint32)
	for i := range srs {
		// robotIds[i] = srs[i].Identification
		robotIds[srs[i].Identification] = srs[i].GroupTimes
	}
	return robotIds, nil
}

func (srm *SSubscriberManager) findSuitableOnes(tid, projectDomainId, projectId string, types ...string) ([]SSubscriber, error) {
	q := srm.Query().Equals("topic_id", tid).IsTrue("enabled")
	q = q.Filter(sqlchemy.OR(
		sqlchemy.AND(
			sqlchemy.Equals(q.Field("resource_scope"), api.SUBSCRIBER_SCOPE_PROJECT),
			sqlchemy.Equals(q.Field("resource_attribution_id"), projectId),
		),
		sqlchemy.AND(
			sqlchemy.Equals(q.Field("resource_scope"), api.SUBSCRIBER_SCOPE_DOMAIN),
			sqlchemy.Equals(q.Field("resource_attribution_id"), projectDomainId),
		),
		sqlchemy.Equals(q.Field("resource_scope"), api.SUBSCRIBER_SCOPE_SYSTEM),
	))
	switch len(types) {
	case 0:
	case 1:
		q = q.Equals("type", types[0])
	default:
		q = q.In("type", types)
	}
	srs := make([]SSubscriber, 0, 1)
	err := db.FetchModelObjects(srm, q, &srs)
	if err != nil {
		return nil, err
	}
	return srs, nil
}

// TODO: Use cache to increase speed
func (srm *SSubscriberManager) getReceiversSent(ctx context.Context, tid string, projectDomainId string, projectId string) (map[string]uint32, error) {
	srs, err := srm.findSuitableOnes(tid, projectDomainId, projectId, api.SUBSCRIBER_TYPE_RECEIVER, api.SUBSCRIBER_TYPE_ROLE)
	if err != nil {
		return nil, err
	}
	// 接受人-聚合时间
	receivers := make(map[string]uint32)
	// 角色-接受人
	roleMap := make(map[string][]string, 3)
	// 接受角色-接受人-聚合时间
	receivermap := make(map[string]map[string]uint32, 3)
	// 聚合时间
	roleGroupTimes := 0
	for _, sr := range srs {
		if sr.Type == api.SUBSCRIBER_TYPE_RECEIVER {
			rIds, err := sr.getReceivers()
			if err != nil {
				return nil, errors.Wrap(err, "unable to get receivers")
			}
			for _, receiveId := range rIds {
				// receivers = append(receivers, api.SReceiverWithGroupTimes{ReceiverId: receiveId, GroupTimes: sr.GroupTimes})
				receivers[receiveId] = sr.GroupTimes
			}
		} else if sr.Type == api.SUBSCRIBER_TYPE_ROLE {
			roleGroupTimes = int(sr.GroupTimes)
			roleMap[sr.RoleScope] = append(roleMap[sr.RoleScope], sr.Identification)
			receivermap[sr.RoleScope] = map[string]uint32{}
		}
	}
	errgo, _ := errgroup.WithContext(ctx)
	for _scope, _roles := range roleMap {
		scope, roles := _scope, _roles
		errgo.Go(func() error {
			query := jsonutils.NewDict()
			query.Set("roles", jsonutils.NewStringArray(roles))
			query.Set("effective", jsonutils.JSONTrue)
			switch scope {
			case api.SUBSCRIBER_SCOPE_SYSTEM:
			case api.SUBSCRIBER_SCOPE_DOMAIN:
				if len(projectDomainId) == 0 {
					return fmt.Errorf("need projectDomainId")
				}
				query.Set("project_domain_id", jsonutils.NewString(projectDomainId))
			case api.SUBSCRIBER_SCOPE_PROJECT:
				if len(projectId) == 0 {
					return fmt.Errorf("need projectId")
				}
				query.Add(jsonutils.NewString(projectId), "scope", "project", "id")
			}
			s := auth.GetAdminSession(ctx, "")
			listRet, err := modules.RoleAssignments.List(s, query)
			if err != nil {
				return errors.Wrap(err, "unable to list RoleAssignments")
			}
			for i := range listRet.Data {
				ras := listRet.Data[i]
				user, err := ras.Get("user")
				if err == nil {
					id, err := user.GetString("id")
					if err != nil {
						return errors.Wrap(err, "unable to get user.id from result of RoleAssignments.List")
					}
					if _, ok := receivermap[scope][id]; !ok {
						receivermap[scope][id] = uint32(roleGroupTimes)
					}
				}
			}
			return nil
		})
	}
	err = errgo.Wait()
	if err != nil {
		return nil, err
	}

	for _, res := range receivermap {
		for receive, time := range res {
			if t, ok := receivers[receive]; !ok || t == 0 {
				receivers[receive] = time
			}
		}
	}
	// de-duplication
	return receivers, nil
}

func (sr *SSubscriber) getReceivers() ([]string, error) {
	srrs, err := SubscriberReceiverManager.getBySubscriberId(sr.Id)
	if err != nil {
		return nil, err
	}
	rIds := make([]string, len(srrs))
	for i := range srrs {
		rIds[i] = srrs[i].ReceiverId
	}
	return rIds, nil
}

func (sr *SSubscriber) SetReceivers(ctx context.Context, receiverIds []string) error {
	srrs, err := SubscriberReceiverManager.getBySubscriberId(sr.Id)
	if err != nil {
		return errors.Wrapf(err, "unable to get SRReceiver by Subscriber %s", sr.Id)
	}
	dbReceivers := make([]string, len(srrs))
	for i := range srrs {
		dbReceivers[i] = srrs[i].ReceiverId
	}
	var addReceivers, rmReceivers []string
	sort.Strings(dbReceivers)
	sort.Strings(receiverIds)
	for i, j := 0, 0; i < len(dbReceivers) || j < len(receiverIds); {
		switch {
		case i == len(dbReceivers):
			addReceivers = append(addReceivers, receiverIds[j])
			j++
		case j == len(receiverIds):
			rmReceivers = append(rmReceivers, dbReceivers[i])
			i++
		case dbReceivers[i] > receiverIds[j]:
			addReceivers = append(addReceivers, receiverIds[j])
			j++
		case dbReceivers[i] < receiverIds[j]:
			rmReceivers = append(rmReceivers, dbReceivers[i])
			i++
		case dbReceivers[i] == receiverIds[j]:
			i++
			j++
		}
	}
	// add
	for _, rId := range addReceivers {
		_, err := SubscriberReceiverManager.create(ctx, sr.Id, rId)
		if err != nil {
			return errors.Wrapf(err, "unable to connect subscription receiver %q with receiver %q", sr.Id, rId)
		}
	}
	for _, rId := range rmReceivers {
		err := SubscriberReceiverManager.delete(sr.Id, rId)
		if err != nil {
			return errors.Wrapf(err, "unable to disconnect subscription receiver %q with receiver %q", sr.Id, rId)
		}
	}
	return nil
}

func (s *SSubscriber) PerformSetReceiver(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SubscriberSetReceiverInput) (jsonutils.JSONObject, error) {
	reIds, err := SubscriberManager.validateReceivers(ctx, input.Receivers)
	if err != nil {
		return nil, err
	}
	return nil, s.SetReceivers(ctx, reIds)
}

func (s *SSubscriber) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(s, ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (s *SSubscriber) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(s, ctx, userCred, false)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (sm *SSubscriberManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := sm.SStandaloneAnonResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	switch field {
	case "resource_scope":
		return sm.Query("resource_scope").Distinct(), nil
	case "type":
		return sm.Query("type").Distinct(), nil
	}
	return q, nil
}

var defaultNotifyTopics = []string{
	DefaultServerPanicked,
	DefaultServiceAbnormal,
	DefaultNetOutOfSync,
	DefaultMysqlOutOfSync,
	DefaultActionLogExceedCount,
}

func (sm *SSubscriberManager) InitializeData() error {
	ctx := context.Background()
	session := auth.GetAdminSession(ctx, options.Options.Region)
	// 获取系统管理员角色id
	params := map[string]interface{}{
		"project_domain": "default",
	}
	role, err := identity.RolesV3.Get(session, "admin", jsonutils.Marshal(params))
	if err != nil {
		return errors.Wrap(err, "identity.RolesV3.List")
	}
	roleId, _ := role.GetString("id")
	q := TopicManager.Query()
	q = q.Filter(sqlchemy.OR(sqlchemy.In(q.Field("name"), defaultNotifyTopics)))
	topics := []STopic{}
	err = db.FetchModelObjects(TopicManager, q, &topics)
	if err != nil {
		return errors.Wrap(err, "FetchModelObjects topic")
	}
	for _, topic := range topics {
		q := sm.Query()
		q = q.Equals("topic_id", topic.Id)
		q = q.Equals("type", api.SUBSCRIBER_TYPE_ROLE)
		q = q.Equals("identification", roleId)
		count, err := q.CountWithError()
		if err != nil {
			return errors.Wrap(err, "CountWithError")
		}
		if count != 0 {
			continue
		}

		subscriber := SSubscriber{}
		subscriber.Type = api.SUBSCRIBER_TYPE_ROLE
		subscriber.Identification = roleId
		subscriber.TopicId = topic.Id
		subscriber.Scope = api.SUBSCRIBER_SCOPE_SYSTEM
		subscriber.ResourceScope = api.SUBSCRIBER_SCOPE_SYSTEM
		subscriber.Enabled = tristate.True
		sm.TableSpec().Insert(ctx, &subscriber)
	}
	return nil
}

// 根据接受人ID获取订阅
func getSubscriberByReceiverId(receiverId string, showDisabled bool) ([]SSubscriber, error) {
	results := []SSubscriber{}

	tempRes := []SSubscriber{}
	// q1 根据接受人ID查找(优先)
	q1 := SubscriberManager.Query()
	q1 = q1.Equals("type", api.SUBSCRIBER_TYPE_RECEIVER)
	srq := SubscriberReceiverManager.Query().Equals("receiver_id", receiverId)
	srsq := srq.SubQuery()
	if !showDisabled {
		q1 = q1.Equals("enabled", true)
	}
	q1.Join(srsq, sqlchemy.Equals(q1.Field("id"), srsq.Field("subscriber_id")))
	err := db.FetchModelObjects(SubscriberManager, q1, &tempRes)
	if err != nil {
		return nil, errors.Wrap(err, "fetch receiver")
	}
	results = append(results, tempRes...)
	roleArr := []string{}
	// 获取当前接受人所有角色
	s := auth.GetAdminSession(context.Background(), options.Options.Region)
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("system"), "scope")
	query.Add(jsonutils.NewString("user"), "resource")
	query.Add(jsonutils.NewBool(true), "details")
	query.Add(jsonutils.NewString("project"), "group_by")
	query.Add(jsonutils.NewBool(true), "effective")
	resp, err := identity.RoleAssignments.GetProjectRole(s, receiverId, query)
	if err != nil {
		return nil, errors.Wrap(err, "UserCacheManager.FetchUserByIdOrName")
	}
	dataArr, _ := resp.GetArray("data")
	for _, data := range dataArr {
		groupArr, _ := data.GetArray("groups")
		for _, group := range groupArr {
			rolesArr, _ := group.GetArray("roles")
			for _, role := range rolesArr {
				roleId, _ := role.GetString("id")
				roleArr = append(roleArr, roleId)
			}
		}
	}
	// q2 根据角色查找
	q2 := SubscriberManager.Query()
	q2 = q2.Equals("type", api.SUBSCRIBER_TYPE_ROLE)
	if !showDisabled {
		q2 = q2.Equals("enabled", true)
	}
	q2 = q2.In("identification", roleArr)
	tempRes = []SSubscriber{}
	err = db.FetchModelObjects(SubscriberManager, q2, &tempRes)
	if err != nil {
		return nil, errors.Wrap(err, "fetch role")
	}
	results = append(results, tempRes...)
	return results, nil
}
