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
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
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
	SubscriberManager.SetVirtualObject(ReceiverNotificationManager)
}

type SSubscriberManager struct {
	db.SStandaloneAnonResourceBaseManager
	db.SEnabledResourceBaseManager
}

type SSubscriber struct {
	db.SStandaloneAnonResourceBase
	db.SEnabledResourceBase

	TopicID               string `width:"128" charset:"ascii" nullable:"false" index:"true" get:"user" list:"user" create:"required"`
	Type                  string `width:"16" charset:"ascii" nullable:"false" index:"true" get:"user" list:"user" create:"required"`
	Identification        string `width:"128" charset:"ascii" nullable:"false" index:"true"`
	RoleScope             string `width:"8" charset:"ascii" nullable:"false" get:"user" list:"user" create:"optional"`
	ResourceScope         string `width:"8" charset:"ascii" nullable:"false" get:"user" list:"user" create:"required"`
	ResourceAttributionId string `width:"128" charset:"ascii" nullable:"false" get:"user" list:"user" create:"optional"`
	Scope                 string `width:"128" charset:"ascii" nullable:"false" create:"required"`
	DomainId              string `width:"128" charset:"ascii" nullable:"false" create:"optional"`
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

func (sm *SSubscriberManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.SubscriberCreateInput) (api.SubscriberCreateInput, error) {
	log.Infof("before deal: %s", jsonutils.Marshal(input))
	var err error
	// permission check
	sSystem, sDomain := string(rbacutils.ScopeSystem), string(rbacutils.ScopeDomain)
	switch input.Scope {
	case sSystem:
		allow := db.IsAdminAllowCreate(userCred, sm)
		if !allow {
			return input, httperrors.NewForbiddenError("The scope %s and the role of the operator do not match", input.Scope)
		}
	case sDomain:
		allow := db.IsDomainAllowCreate(userCred, sm)
		if !allow {
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
		tenant, err := db.TenantCacheManager.FetchTenantByIdOrName(ctx, input.ResourceAttributionId)
		if err != nil {
			return input, errors.Wrapf(err, "unable to fetch project %s", input.ResourceAttributionId)
		}
		domainId = tenant.DomainId
		input.DomainId = domainId
		input.ResourceAttributionId = tenant.GetId()
	case api.SUBSCRIBER_SCOPE_DOMAIN:
		tenant, err := db.TenantCacheManager.FetchDomainByIdOrName(ctx, input.ResourceAttributionId)
		if err != nil {
			return input, errors.Wrapf(err, "unable to fetch domain %s", input.ResourceAttributionId)
		}
		domainId = tenant.DomainId
		input.DomainId = domainId
		input.ResourceAttributionId = tenant.DomainId
	}
	if input.Scope == sDomain && domainId != userCred.GetDomainId() {
		return input, httperrors.NewForbiddenError("domain %s admin can't create subscriber for domain %s", userCred.GetDomainId(), domainId)
	}
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
	case api.SUBSCRIBER_TYPE_ROBOT:
		robot, err := RobotManager.FetchByIdOrName(userCred, input.Robot)
		if errors.Cause(err) == sql.ErrNoRows {
			return input, httperrors.NewInputParameterError("robot %q not found", input.Robot)
		}
		if err != nil {
			return input, errors.Wrapf(err, "unable to fetch robot %q", input.Robot)
		}
		input.Robot = robot.GetId()
	default:
		return input, httperrors.NewInputParameterError("unkown type %q", input.Type)
	}
	log.Infof("after deal input: %s", jsonutils.Marshal(input))
	return input, nil
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
		err := s.SetReceivers(ctx, input.Receivers)
		if err != nil {
			return errors.Wrapf(err, "unable to set connect receivers with subscriber %s", s.Id)
		}
	case api.SUBSCRIBER_TYPE_ROBOT:
		s.Identification = input.Robot
	case api.SUBSCRIBER_TYPE_ROLE:
		s.Identification = input.Role
	}
	s.Enabled = tristate.True
	return nil
}

func (s *SSubscriber) AllowPerformChange(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (s *SSubscriber) PerformChange(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SubscriberChangeInput) (jsonutils.JSONObject, error) {
	if s.Scope == string(rbacutils.ScopeSystem) {
		if !db.IsAdminAllowUpdate(userCred, s) {
			return nil, httperrors.NewForbiddenError("")
		}
	} else {
		if !db.IsDomainAllowUpdate(userCred, s) {
			return nil, httperrors.NewForbiddenError("")
		}
		if s.DomainId != userCred.GetDomainId() {
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
		s.Identification = input.Robot
	case api.SUBSCRIBER_TYPE_ROLE:
		s.Identification = input.Role
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
	sSystem, sDomain := string(rbacutils.ScopeSystem), string(rbacutils.ScopeDomain)
	if input.Scope == "" {
		input.Scope = sSystem
	}
	switch input.Scope {
	case sSystem:
		allow := db.IsAdminAllowList(userCred, sm)
		if !allow {
			return nil, httperrors.NewForbiddenError("")
		}
	case sDomain:
		allow := db.IsAdminAllowList(userCred, sm)
		if !allow {
			return nil, httperrors.NewForbiddenError("")
		}
		q = q.Equals("domain_id", userCred.GetDomainId())
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
			rows[i].Role, err = s.roleIdentification()
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
	if s.Scope == string(rbacutils.ScopeSystem) {
		if !db.IsAdminAllowDelete(userCred, s) {
			return httperrors.NewForbiddenError("")
		}
	} else {
		if !db.IsDomainAllowDelete(userCred, s) {
			return httperrors.NewForbiddenError("")
		}
		if s.DomainId != userCred.GetDomainId() {
			return httperrors.NewForbiddenError("")
		}
	}
	return nil
}

func (s *SSubscriber) receiverIdentifications() ([]api.Identification, error) {
	srSubq := SubscriberReceiverManager.Query().Equals("subscription_id", s.Id).SubQuery()
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

func (s *SSubscriber) roleIdentification() (api.Identification, error) {
	var ret api.Identification
	q := db.RoleCacheManager.Query("id", "name").Equals("id", s.Identification)
	err := q.First(&ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func (srm *SSubscriberManager) robot(tid, projectDomainId, projectId string) ([]string, error) {
	srs, err := srm.findSuitableOnes(tid, projectDomainId, projectId, api.SUBSCRIBER_TYPE_ROBOT)
	if err != nil {
		return nil, err
	}
	robotIds := make([]string, len(srs))
	for i := range srs {
		robotIds[i] = srs[i].Identification
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
	q.DebugQuery()
	srs := make([]SSubscriber, 0, 1)
	err := db.FetchModelObjects(srm, q, &srs)
	if err != nil {
		return nil, err
	}
	return srs, nil
}

// TODO: Use cache to increase speed
func (srm *SSubscriberManager) getReceiversSent(ctx context.Context, tid string, projectDomainId string, projectId string) ([]string, error) {
	srs, err := srm.findSuitableOnes(tid, projectDomainId, projectId, api.SUBSCRIBER_TYPE_RECEIVER, api.SUBSCRIBER_TYPE_ROLE)
	if err != nil {
		return nil, err
	}
	receivers := make([]string, 0, len(srs))
	roleMap := make(map[string][]string, 3)
	receivermap := make(map[string]*[]string, 3)
	for _, sr := range srs {
		if sr.Type == api.SUBSCRIBER_TYPE_RECEIVER {
			rIds, err := sr.getReceivers()
			if err != nil {
				return nil, errors.Wrap(err, "unable to get receivers")
			}
			receivers = append(receivers, rIds...)
		} else if sr.Type == api.SUBSCRIBER_TYPE_ROLE {
			roleMap[sr.RoleScope] = append(roleMap[sr.RoleScope], sr.Identification)
			receivermap[sr.RoleScope] = &[]string{}
		}
	}
	errgo, _ := errgroup.WithContext(ctx)
	for _scope, _roles := range roleMap {
		scope, roles := _scope, _roles
		receivers := receivermap[scope]
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
			s := auth.GetAdminSession(ctx, "", "")
			log.Debugf("query for role-assignments: %s", query.String())
			listRet, err := modules.RoleAssignments.List(s, query)
			if err != nil {
				return errors.Wrap(err, "unable to list RoleAssignments")
			}
			log.Debugf("return value for role-assignments: %s", jsonutils.Marshal(listRet))
			for i := range listRet.Data {
				ras := listRet.Data[i]
				user, err := ras.Get("user")
				if err == nil {
					id, err := user.GetString("id")
					if err != nil {
						return errors.Wrap(err, "unable to get user.id from result of RoleAssignments.List")
					}
					*receivers = append(*receivers, id)
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
		receivers = append(receivers, *res...)
	}
	// de-duplication
	return sets.NewString(receivers...).UnsortedList(), nil
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
			rmReceivers = append(rmReceivers, dbReceivers[i])
			i++
		case dbReceivers[i] < receiverIds[j]:
			addReceivers = append(addReceivers, receiverIds[j])
			j++
		case dbReceivers[i] == receiverIds[j]:
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

func (s *SSubscriber) AllowPerformSetReceiver(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, s, "set-receiver")
}

func (s *SSubscriber) PerformSetReceiver(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SubscriberSetReceiverInput) (jsonutils.JSONObject, error) {
	reIds, err := SubscriberManager.validateReceivers(ctx, input.Receivers)
	if err != nil {
		return nil, err
	}
	return nil, s.SetReceivers(ctx, reIds)
}

func (s *SSubscriber) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, s, "enable")
}

func (s *SSubscriber) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(s, ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (s *SSubscriber) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, s, "disable")
}

func (s *SSubscriber) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(s, ctx, userCred, false)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}
