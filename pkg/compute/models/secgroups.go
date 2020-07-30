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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSecurityGroupManager struct {
	db.SSharableVirtualResourceBaseManager
}

var SecurityGroupManager *SSecurityGroupManager

func init() {
	SecurityGroupManager = &SSecurityGroupManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SSecurityGroup{},
			"secgroups_tbl",
			"secgroup",
			"secgroups",
		),
	}
	SecurityGroupManager.NameLength = 128
	SecurityGroupManager.NameRequireAscii = true
	SecurityGroupManager.SetVirtualObject(SecurityGroupManager)
}

const (
	SECURITY_GROUP_SEPARATOR = ";"
)

type SSecurityGroup struct {
	db.SSharableVirtualResourceBase
	IsDirty bool `nullable:"false" default:"false"`
}

// 安全组列表
func (manager *SSecurityGroupManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.SecgroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
	}

	if len(input.Equals) > 0 {
		_secgroup, err := manager.FetchByIdOrName(userCred, input.Equals)
		if err != nil {
			return nil, httperrors.NewInputParameterError("Failed fetching secgroup %s", input.Equals)
		}
		secgroup := _secgroup.(*SSecurityGroup)
		inAllowList := secgroup.GetInAllowList()
		outAllowList := secgroup.GetOutAllowList()
		sq := manager.Query().NotEquals("id", secgroup.Id)
		secgroups := []SSecurityGroup{}
		err = db.FetchModelObjects(manager, sq, &secgroups)
		if err != nil {
			return nil, err
		}
		secgroupIds := []string{}
		for i := 0; i < len(secgroups); i++ {
			_inAllowList := secgroups[i].GetInAllowList()
			if !inAllowList.Equals(_inAllowList) {
				continue
			}
			_outAllowList := secgroups[i].GetOutAllowList()
			if !outAllowList.Equals(_outAllowList) {
				continue
			}
			secgroupIds = append(secgroupIds, secgroups[i].Id)
		}
		q = q.In("id", secgroupIds)
	}
	serverStr := input.ServerId
	if len(serverStr) > 0 {
		guest, _, err := ValidateGuestResourceInput(userCred, input.ServerResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateGuestResourceInput")
		}
		serverId := guest.GetId()
		filters := []sqlchemy.ICondition{}
		filters = append(filters, sqlchemy.In(q.Field("id"), GuestManager.Query("secgrp_id").Equals("id", serverId).SubQuery()))
		filters = append(filters, sqlchemy.In(q.Field("id"), GuestsecgroupManager.Query("secgroup_id").Equals("guest_id", serverId).SubQuery()))

		isAdmin := false
		admin := (input.ServerFilterListInput.Admin != nil && *input.ServerFilterListInput.Admin)
		if consts.IsRbacEnabled() {
			allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
			if allowScope == rbacutils.ScopeSystem || allowScope == rbacutils.ScopeDomain {
				isAdmin = true
			}
		} else if userCred.HasSystemAdminPrivilege() && admin {
			isAdmin = true
		}

		if isAdmin {
			filters = append(filters, sqlchemy.In(q.Field("id"), GuestManager.Query("admin_secgrp_id").Equals("id", serverId).SubQuery()))
		}
		q = q.Filter(sqlchemy.OR(filters...))
	}

	if len(input.Ip) > 0 || len(input.Ports) > 0 {
		sq := SecurityGroupRuleManager.Query("secgroup_id")
		if len(input.Ip) > 0 {
			sq = sq.Like("cidr", input.Ip+"%")
		}
		if len(input.Ports) > 0 {
			sq = sq.Equals("ports", input.Ports)
		}
		if utils.IsInStringArray(input.Direction, []string{"in", "out"}) {
			sq = sq.Equals("direction", input.Direction)
		}
		q = q.In("id", sq.SubQuery())
	}

	return q, nil
}

func (manager *SSecurityGroupManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.SecgroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.OrderByExtraFields")
	}

	orderByCache := input.OrderByCacheCnt
	if sqlchemy.SQL_ORDER_ASC.Equals(orderByCache) || sqlchemy.SQL_ORDER_DESC.Equals(orderByCache) {
		caches := SecurityGroupCacheManager.Query().SubQuery()
		cacheQ := caches.Query(
			caches.Field("secgroup_id"),
			sqlchemy.COUNT("cache_cnt"),
		)
		cacheSQ := cacheQ.GroupBy(caches.Field("secgroup_id")).SubQuery()
		q = q.LeftJoin(cacheSQ, sqlchemy.Equals(q.Field("id"), cacheSQ.Field("secgroup_id")))
		if sqlchemy.SQL_ORDER_ASC.Equals(orderByCache) {
			q = q.Asc(cacheSQ.Field("cache_cnt"))
		} else {
			q = q.Desc(cacheSQ.Field("cache_cnt"))
		}
	}
	orderByGuest := input.OrderByGuestCnt
	if sqlchemy.SQL_ORDER_ASC.Equals(orderByGuest) || sqlchemy.SQL_ORDER_DESC.Equals(orderByGuest) {
		guests := GuestManager.Query().SubQuery()
		guestsecgroups := GuestsecgroupManager.Query().SubQuery()
		q1 := guests.Query(guests.Field("id").Label("guest_id"),
			guests.Field("secgrp_id").Label("secgroup_id"))
		q2 := guestsecgroups.Query(guestsecgroups.Field("guest_id"),
			guestsecgroups.Field("secgroup_id"))
		uq := sqlchemy.Union(q1, q2)
		uQ := uq.Query(
			uq.Field("secgroup_id"),
			sqlchemy.COUNT("guest_cnt", uq.Field("guest_id")),
		)
		sq := uQ.GroupBy(uq.Field("secgroup_id")).SubQuery()

		q = q.LeftJoin(sq, sqlchemy.Equals(q.Field("id"), sq.Field("secgroup_id")))

		if sqlchemy.SQL_ORDER_ASC.Equals(orderByGuest) {
			q = q.Asc(sq.Field("guest_cnt"))
		} else {
			q = q.Desc(sq.Field("guest_cnt"))
		}

	}

	return q, nil
}

func (manager *SSecurityGroupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SSecurityGroup) GetGuestsQuery() *sqlchemy.SQuery {
	guests := GuestManager.Query().SubQuery()
	return guests.Query().Filter(
		sqlchemy.OR(
			sqlchemy.Equals(guests.Field("secgrp_id"), self.Id),
			sqlchemy.Equals(guests.Field("admin_secgrp_id"), self.Id),
			sqlchemy.In(guests.Field("id"), GuestsecgroupManager.Query("guest_id").Equals("secgroup_id", self.Id).SubQuery()),
		),
	).Filter(sqlchemy.NotIn(guests.Field("hypervisor"), []string{api.HYPERVISOR_CONTAINER, api.HYPERVISOR_BAREMETAL, api.HYPERVISOR_ESXI}))
}

func (self *SSecurityGroup) GetGuestsCount() (int, error) {
	return self.GetGuestsQuery().CountWithError()
}

func (self *SSecurityGroup) GetGuests() []SGuest {
	guests := []SGuest{}
	q := self.GetGuestsQuery()
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("GetGuests fail %s", err)
		return nil
	}
	return guests
}

func (self *SSecurityGroup) GetSecgroupCacheQuery() *sqlchemy.SQuery {
	return SecurityGroupCacheManager.Query().Equals("secgroup_id", self.Id)
}

func (self *SSecurityGroup) GetSecgroupCacheCount() (int, error) {
	return self.GetSecgroupCacheQuery().CountWithError()
}

func (self *SSecurityGroup) getDesc() jsonutils.JSONObject {
	desc := jsonutils.NewDict()
	desc.Add(jsonutils.NewString(self.Name), "name")
	desc.Add(jsonutils.NewString(self.Id), "id")
	//desc.Add(jsonutils.NewString(self.getSecurityRuleString("")), "security_rules")
	return desc
}

func (manager *SSecurityGroupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.SecgroupDetails {
	rows := make([]api.SecgroupDetails, len(objs))

	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	secgroupIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.SecgroupDetails{
			SharableVirtualResourceDetails: virtRows[i],
		}
		secgroup := objs[i].(*SSecurityGroup)
		secgroupIds[i] = secgroup.Id
	}

	caches := []SSecurityGroupCache{}
	q := SecurityGroupCacheManager.Query().In("secgroup_id", secgroupIds)
	err := db.FetchModelObjects(SecurityGroupCacheManager, q, &caches)
	if err != nil {
		log.Errorf("db.FetchModelObjects error: %v", err)
		return rows
	}

	cacheMaps := map[string]int{}
	for i := range caches {
		if _, ok := cacheMaps[caches[i].SecgroupId]; !ok {
			cacheMaps[caches[i].SecgroupId] = 0
		}
		cacheMaps[caches[i].SecgroupId]++
	}

	guests := []SGuest{}
	q = GuestManager.Query()
	q = q.Filter(sqlchemy.OR(
		sqlchemy.In(q.Field("secgrp_id"), secgroupIds),
		sqlchemy.In(q.Field("admin_secgrp_id"), secgroupIds),
	))

	ownerId, queryScope, err := db.FetchCheckQueryOwnerScope(ctx, userCred, query, GuestManager, policy.PolicyActionList, true)
	if err != nil {
		log.Errorf("FetchCheckQueryOwnerScope error: %v", err)
		return rows
	}

	q = GuestManager.FilterByOwner(q, ownerId, queryScope)
	err = db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("db.FetchModelObjects error: %v", err)
		return rows
	}

	adminGuestMaps := map[string]int{}
	systemGuestMaps := map[string]int{}
	normalGuestMaps := map[string]int{}
	for i := range guests {
		if guests[i].IsSystem {
			if _, ok := systemGuestMaps[guests[i].SecgrpId]; !ok {
				systemGuestMaps[guests[i].SecgrpId] = 0
			}
			systemGuestMaps[guests[i].SecgrpId]++
		} else {
			if _, ok := normalGuestMaps[guests[i].SecgrpId]; !ok {
				normalGuestMaps[guests[i].SecgrpId] = 0
			}
			normalGuestMaps[guests[i].SecgrpId]++
		}
		if len(guests[i].AdminSecgrpId) > 0 {
			if _, ok := adminGuestMaps[guests[i].AdminSecgrpId]; !ok {
				adminGuestMaps[guests[i].AdminSecgrpId] = 0
			}
			adminGuestMaps[guests[i].AdminSecgrpId]++
		}
	}

	sq := GuestManager.Query("id")
	sq = GuestManager.FilterByOwner(sq, ownerId, queryScope)

	guestSecgroups := []SGuestsecgroup{}
	q = GuestsecgroupManager.Query().In("secgroup_id", secgroupIds).In("guest_id", sq.SubQuery())
	err = db.FetchModelObjects(GuestsecgroupManager, q, &guestSecgroups)
	if err != nil {
		log.Errorf("db.FetchModelObjects error: %v", err)
		return rows
	}

	for i := range guestSecgroups {
		if _, ok := normalGuestMaps[guestSecgroups[i].SecgroupId]; !ok {
			normalGuestMaps[guestSecgroups[i].SecgroupId] = 0
		}
		normalGuestMaps[guestSecgroups[i].SecgroupId]++
	}

	rules := []SSecurityGroupRule{}
	q = SecurityGroupRuleManager.Query().In("secgroup_id", secgroupIds)
	err = db.FetchModelObjects(SecurityGroupRuleManager, q, &rules)
	if err != nil {
		log.Errorf("db.FetchModelObjects error: %v", err)
		return rows
	}
	ruleMaps := map[string][]SSecurityGroupRule{}
	for i := range rules {
		if _, ok := ruleMaps[rules[i].SecgroupId]; !ok {
			ruleMaps[rules[i].SecgroupId] = []SSecurityGroupRule{}
		}
		ruleMaps[rules[i].SecgroupId] = append(ruleMaps[rules[i].SecgroupId], rules[i])
	}
	for i := range rows {
		rules, ok := ruleMaps[secgroupIds[i]]
		if !ok {
			continue
		}
		_rules := []api.SSecurityGroupRule{}
		_inRules := []api.SSecurityGroupRule{}
		_outRules := []api.SSecurityGroupRule{}
		for j := range rules {
			rule := api.SSecurityGroupRule{
				Id:          rules[j].Id,
				Priority:    rules[j].Priority,
				Protocol:    rules[j].Protocol,
				Ports:       rules[j].Ports,
				Direction:   rules[j].Direction,
				CIDR:        rules[j].CIDR,
				Action:      rules[j].Action,
				Description: rules[j].Description,
			}
			_rules = append(_rules, rule)
			switch rule.Direction {
			case secrules.DIR_IN:
				_inRules = append(_inRules, rule)
			case secrules.DIR_OUT:
				_outRules = append(_outRules, rule)
			}
		}
		rows[i].Rules = _rules
		rows[i].InRules = _inRules
		rows[i].OutRules = _outRules
		rows[i].CacheCnt, _ = cacheMaps[secgroupIds[i]]
		rows[i].GuestCnt, _ = normalGuestMaps[secgroupIds[i]]
		rows[i].AdminGuestCnt, _ = adminGuestMaps[secgroupIds[i]]
		rows[i].SystemGuestCnt, _ = systemGuestMaps[secgroupIds[i]]
	}

	return rows
}

func (self *SSecurityGroup) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.SecgroupDetails, error) {
	return api.SecgroupDetails{}, nil
}

func (manager *SSecurityGroupManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.SSecgroupCreateInput,
) (api.SSecgroupCreateInput, error) {
	var err error

	// TODO: check set pending quota
	input.Status = api.SECGROUP_STATUS_READY

	for i, rule := range input.Rules {
		err = rule.Check()
		if err != nil {
			return input, httperrors.NewInputParameterError("rule %d is invalid: %s", i, err)
		}
	}

	input.SharableVirtualResourceCreateInput, err = manager.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, err
	}

	return input, nil
}

func (self *SSecurityGroup) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	input := &api.SSecgroupCreateInput{}
	data.Unmarshal(input)

	for _, r := range input.Rules {
		rule := &SSecurityGroupRule{
			Priority:    int64(r.Priority),
			Protocol:    r.Protocol,
			Ports:       r.Ports,
			Direction:   r.Direction,
			CIDR:        r.CIDR,
			Action:      r.Action,
			Description: r.Description,
		}
		rule.SecgroupId = self.Id

		SecurityGroupRuleManager.TableSpec().Insert(ctx, rule)
	}
}

func (manager *SSecurityGroupManager) FetchSecgroupById(secId string) *SSecurityGroup {
	if len(secId) > 0 {
		secgrp, _ := manager.FetchById(secId)
		if secgrp != nil {
			return secgrp.(*SSecurityGroup)
		}
	}
	return nil
}

func (self *SSecurityGroup) getSecurityRules(direction string) (rules []SSecurityGroupRule) {
	secgrouprules := SecurityGroupRuleManager.Query().SubQuery()
	sql := secgrouprules.Query().Filter(sqlchemy.Equals(secgrouprules.Field("secgroup_id"), self.Id)).Desc("priority")
	if len(direction) > 0 && utils.IsInStringArray(direction, []string{"in", "out"}) {
		sql = sql.Equals("direction", direction)
	}
	if err := db.FetchModelObjects(SecurityGroupRuleManager, sql, &rules); err != nil {
		log.Errorf("GetGuests fail %s", err)
		return
	}
	return
}

func (self *SSecurityGroup) GetSecRules(direction string) []secrules.SecurityRule {
	rules := make([]secrules.SecurityRule, 0)
	for _, _rule := range self.getSecurityRules(direction) {
		//这里没必要拆分为单个单个的端口,到公有云那边适配
		rule, err := _rule.toRule()
		if err != nil {
			log.Errorln(err)
			continue
		}
		rules = append(rules, *rule)
	}
	return rules
}

func (self *SSecurityGroup) getSecurityRuleString(direction string) string {
	secgrouprules := self.getSecurityRules(direction)
	var rules []string
	for _, rule := range secgrouprules {
		rules = append(rules, rule.String())
	}
	return strings.Join(rules, SECURITY_GROUP_SEPARATOR)
}

func totalSecurityGroupCount(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) (int, error) {
	q := SecurityGroupManager.Query()

	switch scope {
	case rbacutils.ScopeSystem:
		// do nothing
	case rbacutils.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	case rbacutils.ScopeProject:
		q = q.Equals("tenant_id", ownerId.GetProjectId())
	}

	return q.CountWithError()
}

func (self *SSecurityGroup) AllowPerformUncacheSecgroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "uncache-secgroup")
}

func (self *SSecurityGroup) PerformUncacheSecgroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	cacheV := validators.NewModelIdOrNameValidator("secgroupcache", "secgroupcache", nil)
	err := cacheV.Validate(data.(*jsonutils.JSONDict))
	if err != nil {
		return nil, err
	}
	cache := cacheV.Model.(*SSecurityGroupCache)
	return nil, cache.StartSecurityGroupCacheDeleteTask(ctx, userCred, "")
}

func (self *SSecurityGroup) AllowPerformCacheSecgroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "cache-secgroup")
}

func (self *SSecurityGroup) PerformCacheSecgroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	vpcV := validators.NewModelIdOrNameValidator("vpc", "vpc", nil)
	err := vpcV.Validate(data.(*jsonutils.JSONDict))
	if err != nil {
		return nil, err
	}
	vpc := vpcV.Model.(*SVpc)
	if len(vpc.ExternalId) == 0 {
		return nil, httperrors.NewInputParameterError("vpc %s(%s) is not a managed resouce", vpc.Name, vpc.Id)
	}

	region, err := vpc.GetRegion()
	if err != nil {
		return nil, err
	}
	classic, _ := data.Bool("classic")
	if classic && !region.GetDriver().IsSupportClassicSecurityGroup() {
		return nil, httperrors.NewInputParameterError("Not support cache classic security group")
	}

	return nil, self.StartSecurityGroupCacheTask(ctx, userCred, vpc.Id, classic, "")
}

func (self *SSecurityGroup) StartSecurityGroupCacheTask(ctx context.Context, userCred mcclient.TokenCredential, vpcId string, classic bool, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(vpcId), "vpc_id")
	params.Add(jsonutils.NewBool(classic), "classic")
	task, err := taskman.TaskManager.NewTask(ctx, "SecurityGroupCacheTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SSecurityGroup) AllowPerformAddRule(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "add-rule")
}

func (self *SSecurityGroup) PerformAddRule(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	secgrouprule := &SSecurityGroupRule{}
	secgrouprule.SecgroupId = self.Id
	secgrouprule.SetModelManager(SecurityGroupRuleManager, secgrouprule)
	if err := data.Unmarshal(secgrouprule); err != nil {
		return nil, err
	}
	if len(secgrouprule.CIDR) > 0 {
		if !regutils.MatchCIDR(secgrouprule.CIDR) && !regutils.MatchIPAddr(secgrouprule.CIDR) {
			return nil, httperrors.NewInputParameterError("invalid ip address: %s", secgrouprule.CIDR)
		}
	} else {
		secgrouprule.CIDR = "0.0.0.0/0"
	}
	rule := secrules.SecurityRule{
		Priority:  int(secgrouprule.Priority),
		Direction: secrules.TSecurityRuleDirection(secgrouprule.Direction),
		Action:    secrules.TSecurityRuleAction(secgrouprule.Action),
		Protocol:  secgrouprule.Protocol,
		Ports:     []int{},
		PortStart: -1,
		PortEnd:   -1,
	}
	if err := rule.ParsePorts(secgrouprule.Ports); err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}
	if err := rule.ValidateRule(); err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}
	if err := SecurityGroupRuleManager.TableSpec().Insert(ctx, secgrouprule); err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}
	self.DoSync(ctx, userCred)
	return nil, nil
}

func (self *SSecurityGroup) AllowPerformClone(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (self *SSecurityGroup) PerformClone(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	name, _ := data.GetString("name")
	if len(name) == 0 {
		return nil, httperrors.NewMissingParameterError("name")
	}
	_, err := SecurityGroupManager.FetchByName(userCred, name)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, httperrors.NewInternalServerError("FetchByName fail %s", err)
		}
	} else {
		return nil, httperrors.NewDuplicateNameError("name", name)
	}

	secgroup := &SSecurityGroup{}
	secgroup.SetModelManager(SecurityGroupManager, secgroup)

	secgroup.Name = name
	secgroup.Description, _ = data.GetString("description")
	secgroup.ProjectId = userCred.GetProjectId()
	secgroup.DomainId = userCred.GetProjectDomainId()

	err = SecurityGroupManager.TableSpec().Insert(ctx, secgroup)
	if err != nil {
		return nil, err
		//db.OpsLog.LogCloneEvent(self, secgroup, userCred, nil)
	}

	secgrouprules := self.getSecurityRules("")
	for _, rule := range secgrouprules {
		secgrouprule := &SSecurityGroupRule{}
		secgrouprule.SetModelManager(SecurityGroupRuleManager, secgrouprule)

		secgrouprule.Priority = rule.Priority
		secgrouprule.Protocol = rule.Protocol
		secgrouprule.Ports = rule.Ports
		secgrouprule.Direction = rule.Direction
		secgrouprule.CIDR = rule.CIDR
		secgrouprule.Action = rule.Action
		secgrouprule.Description = rule.Description
		secgrouprule.SecgroupId = secgroup.Id
		if err := SecurityGroupRuleManager.TableSpec().Insert(ctx, secgrouprule); err != nil {
			return nil, err
		}
	}

	logclient.AddActionLogWithContext(ctx, secgroup, logclient.ACT_CREATE, secgroup.GetShortDesc(ctx), userCred, true)
	return nil, nil
}

func (self *SSecurityGroup) AllowPerformMerge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "merge")
}

func (self *SSecurityGroup) PerformMerge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	secgroupIds := jsonutils.GetQueryStringArray(data, "secgroups")
	if len(secgroupIds) == 0 {
		return nil, httperrors.NewMissingParameterError("secgroups")
	}
	inAllowList := self.GetInAllowList()
	outAllowList := self.GetOutAllowList()
	secgroups := []*SSecurityGroup{}
	for _, secgroupId := range secgroupIds {
		_secgroup, err := SecurityGroupManager.FetchByIdOrName(userCred, secgroupId)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("failed to find secgroup %s error: %v", secgroupId, err)
		}
		secgroup := _secgroup.(*SSecurityGroup)
		secgroup.SetModelManager(SecurityGroupManager, secgroup)
		_inAllowList := secgroup.GetInAllowList()
		if !inAllowList.Equals(_inAllowList) {
			return nil, httperrors.NewUnsupportOperationError("secgroup %s rules not equals %s rules", secgroup.Name, self.Name)
		}
		_outAllowList := secgroup.GetOutAllowList()
		if !outAllowList.Equals(_outAllowList) {
			return nil, httperrors.NewUnsupportOperationError("secgroup %s rules not equals %s rules", secgroup.Name, self.Name)
		}
		secgroups = append(secgroups, secgroup)
	}

	for i := 0; i < len(secgroups); i++ {
		secgroup := secgroups[i]
		if err := self.migrateSecurityGroupCache(secgroup); err != nil {
			return nil, err
		}

		if err := self.migrateGuestSecurityGroup(secgroup); err != nil {
			return nil, err
		}
		secgroup.RealDelete(ctx, userCred)
	}
	self.DoSync(ctx, userCred)
	return nil, nil
}

func (self *SSecurityGroup) GetOutAllowList() secrules.SecurityRuleSet {
	rules := self.GetSecRules("out")
	ruleSet := secrules.SecurityRuleSet(rules)
	rules = append(rules, *secrules.MustParseSecurityRule("out:allow any"))
	return ruleSet.AllowList()
}

func (self *SSecurityGroup) GetInAllowList() secrules.SecurityRuleSet {
	rules := self.GetSecRules("in")
	rules = append(rules, *secrules.MustParseSecurityRule("in:deny any"))
	ruleSet := secrules.SecurityRuleSet(rules)
	return ruleSet.AllowList()
}

func (self *SSecurityGroup) getSecurityGroupRuleSet() secrules.SecurityGroupRuleSet {
	rules := self.GetSecRules("")
	srs := secrules.SecurityGroupRuleSet{}
	for i := 0; i < len(rules); i++ {
		srs.AddRule(rules[i])
	}
	return srs
}

func (self *SSecurityGroup) migrateSecurityGroupCache(secgroup *SSecurityGroup) error {
	caches := []SSecurityGroupCache{}
	q := SecurityGroupCacheManager.Query().Equals("secgroup_id", secgroup.Id)
	err := db.FetchModelObjects(SecurityGroupCacheManager, q, &caches)
	if err != nil {
		return err
	}
	for i := 0; i < len(caches); i++ {
		cache := caches[i]
		_, err := db.Update(&cache, func() error {
			cache.SecgroupId = self.Id
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SSecurityGroup) migrateGuestSecurityGroup(secgroup *SSecurityGroup) error {
	guests := secgroup.GetGuests()
	for i := 0; i < len(guests); i++ {
		guest := guests[i]
		_, err := db.Update(&guest, func() error {
			if guest.SecgrpId == secgroup.Id {
				guest.SecgrpId = self.Id
			}
			if guest.AdminSecgrpId == secgroup.Id {
				guest.AdminSecgrpId = self.Id
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	guestsecgroups, err := GuestsecgroupManager.GetGuestSecgroups(nil, secgroup)
	if err != nil {
		return err
	}

	for i := 0; i < len(guestsecgroups); i++ {
		guestsecgroup := guestsecgroups[i]
		_, err := db.Update(&guestsecgroup, func() error {
			guestsecgroup.SecgroupId = self.Id
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (manager *SSecurityGroupManager) getSecurityGroups() ([]SSecurityGroup, error) {
	secgroups := make([]SSecurityGroup, 0)
	q := manager.Query()
	if err := db.FetchModelObjects(manager, q, &secgroups); err != nil {
		return nil, err
	} else {
		return secgroups, nil
	}
}

func (manager *SSecurityGroupManager) newFromCloudSecgroup(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extSec cloudprovider.ICloudSecurityGroup) (*SSecurityGroup, error) {
	regionDriver, err := provider.GetRegionDriver()
	if err != nil {
		return nil, errors.Wrap(err, "provider.GetRegionDriver")
	}

	rules, err := extSec.GetRules()
	if err != nil {
		return nil, errors.Wrap(err, "extSec.GetRules")
	}

	inRules := []cloudprovider.SecurityRule{}
	outRules := []cloudprovider.SecurityRule{}
	for i := range rules {
		if rules[i].Direction == secrules.DIR_IN {
			inRules = append(inRules, rules[i])
		} else {
			outRules = append(outRules, rules[i])
		}
	}

	maxPriority := regionDriver.GetSecurityGroupRuleMaxPriority()
	minPriority := regionDriver.GetSecurityGroupRuleMinPriority()

	defaultInRule := regionDriver.GetDefaultSecurityGroupInRule()
	defaultOutRule := regionDriver.GetDefaultSecurityGroupOutRule()
	order := regionDriver.GetSecurityGroupRuleOrder()
	onlyAllowRules := regionDriver.IsOnlySupportAllowRules()

	// 查询与provider在同域的安全组，比对寻找一个与云上安全组规则相同的安全组
	secgroups := []SSecurityGroup{}
	q := manager.Query().Equals("domain_id", provider.DomainId)
	err = db.FetchModelObjects(manager, q, &secgroups)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	for i := range secgroups {
		localRules := secrules.SecurityRuleSet(secgroups[i].GetSecRules(""))
		_, inAdds, outAdds, inDels, outDels := cloudprovider.CompareRules(minPriority, maxPriority, order, localRules, rules, defaultInRule, defaultOutRule, onlyAllowRules, false)
		if len(inAdds) == 0 && len(outAdds) == 0 && len(inDels) == 0 && len(outDels) == 0 {
			return &secgroups[i], nil
		}
	}

	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	secgroup := SSecurityGroup{}
	secgroup.SetModelManager(manager, &secgroup)
	newName, err := db.GenerateName(manager, userCred, extSec.GetName())
	if err != nil {
		return nil, err
	}

	secgroup.Name = newName
	secgroup.Description = extSec.GetDescription()
	secgroup.ProjectId = provider.ProjectId
	secgroup.DomainId = provider.DomainId

	if err := manager.TableSpec().Insert(ctx, &secgroup); err != nil {
		return nil, err
	}

	//这里必须先同步下规则,不然下次对比此安全组规则为空
	inRules = cloudprovider.AddDefaultRule(inRules, defaultInRule, "in:deny any", order, minPriority, maxPriority, onlyAllowRules)
	cloudprovider.SortSecurityRule(inRules, order, onlyAllowRules)
	outRules = cloudprovider.AddDefaultRule(outRules, defaultOutRule, "out:allow any", order, minPriority, maxPriority, onlyAllowRules)
	cloudprovider.SortSecurityRule(outRules, order, onlyAllowRules)

	SecurityGroupRuleManager.SyncRules(ctx, userCred, &secgroup, inRules)
	SecurityGroupRuleManager.SyncRules(ctx, userCred, &secgroup, outRules)

	db.OpsLog.LogEvent(&secgroup, db.ACT_CREATE, secgroup.GetShortDesc(ctx), userCred)

	return &secgroup, nil
}

func (manager *SSecurityGroupManager) DelaySync(ctx context.Context, userCred mcclient.TokenCredential, idStr string) {
	if secgrp := manager.FetchSecgroupById(idStr); secgrp == nil {
		log.Errorf("DelaySync secgroup failed")
	} else {
		needSync := false

		func() {
			lockman.LockObject(ctx, secgrp)
			defer lockman.ReleaseObject(ctx, secgrp)

			if secgrp.IsDirty {
				if _, err := db.Update(secgrp, func() error {
					secgrp.IsDirty = false
					return nil
				}); err != nil {
					log.Errorf("Update Security Group error: %s", err.Error())
				}
				needSync = true
			}
		}()

		if needSync {
			for _, guest := range secgrp.GetGuests() {
				guest.StartSyncTask(ctx, userCred, true, "")
			}
		}
	}
}

func (self *SSecurityGroup) DoSync(ctx context.Context, userCred mcclient.TokenCredential) {
	if _, err := db.Update(self, func() error {
		self.IsDirty = true
		return nil
	}); err != nil {
		log.Errorf("Update Security Group error: %s", err.Error())
	}
	time.AfterFunc(10*time.Second, func() {
		SecurityGroupManager.DelaySync(context.Background(), userCred, self.Id)
	})
}

func (manager *SSecurityGroupManager) InitializeData() error {
	_, err := manager.FetchById("default")
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("find default secgroup fail %s", err)
		return err
	}
	if err == sql.ErrNoRows {
		var secGrp *SSecurityGroup
		secGrp = &SSecurityGroup{}
		secGrp.SetModelManager(manager, secGrp)
		secGrp.Id = "default"
		secGrp.Name = "Default"
		secGrp.Status = api.SECGROUP_STATUS_READY
		secGrp.ProjectId = auth.AdminCredential().GetProjectId()
		secGrp.DomainId = auth.AdminCredential().GetProjectDomainId()
		// secGrp.IsEmulated = false
		secGrp.IsPublic = true
		secGrp.PublicScope = string(rbacutils.ScopeSystem)
		err = manager.TableSpec().Insert(context.TODO(), secGrp)
		if err != nil {
			log.Errorf("Insert default secgroup failed!!! %s", err)
			return err
		}

		defRule := SSecurityGroupRule{}
		defRule.SetModelManager(SecurityGroupRuleManager, &defRule)
		defRule.Direction = secrules.DIR_IN
		defRule.Protocol = secrules.PROTO_ANY
		defRule.Priority = 1
		defRule.CIDR = "0.0.0.0/0"
		defRule.Action = string(secrules.SecurityRuleAllow)
		defRule.SecgroupId = "default"
		err = SecurityGroupRuleManager.TableSpec().Insert(context.TODO(), &defRule)
		if err != nil {
			log.Errorf("Insert default secgroup rule fail %s", err)
			return err
		}
	}
	guests := make([]SGuest, 0)
	q := GuestManager.Query()
	q = q.Filter(sqlchemy.OR(sqlchemy.IsEmpty(q.Field("secgrp_id")), sqlchemy.IsNull(q.Field("secgrp_id"))))

	err = db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("fetch guests without secgroup fail %s", err)
		return err
	}
	for i := 0; i < len(guests); i += 1 {
		db.Update(&guests[i], func() error {
			guests[i].SecgrpId = "default"
			return nil
		})
	}

	secgroups := []SSecurityGroup{}
	q = SecurityGroupManager.Query().NotEquals("status", api.SECGROUP_STATUS_READY)
	err = db.FetchModelObjects(manager, q, &secgroups)
	if err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}

	for i := range secgroups {
		db.Update(&secgroups[i], func() error {
			secgroups[i].Status = api.SECGROUP_STATUS_READY
			return nil
		})
	}

	return nil
}

func (self *SSecurityGroup) ValidateDeleteCondition(ctx context.Context) error {
	cnt, err := self.GetGuestsCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetGuestsCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("the security group is in use")
	}
	if self.Id == "default" {
		return httperrors.NewProtectedResourceError("not allow to delete default security group")
	}
	return self.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SSecurityGroup) GetSecurityGroupCaches() []SSecurityGroupCache {
	caches := []SSecurityGroupCache{}
	q := SecurityGroupCacheManager.Query()
	q = q.Filter(sqlchemy.Equals(q.Field("secgroup_id"), self.Id))
	if err := db.FetchModelObjects(SecurityGroupCacheManager, q, &caches); err != nil {
		log.Errorf("get secgroupcache for secgroup %s error: %v", self.Name, err)
	}
	return caches
}

func (self *SSecurityGroup) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteSecurityGroupTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (self *SSecurityGroup) StartDeleteSecurityGroupTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, api.SECGROUP_STATUS_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "SecurityGroupDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SSecurityGroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("do nothing for delete secgroup")
	return nil
}

func (self *SSecurityGroup) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	rules := []SSecurityGroupRule{}
	q := SecurityGroupRuleManager.Query().Equals("secgroup_id", self.Id)
	err := db.FetchModelObjects(SecurityGroupRuleManager, q, &rules)
	if err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}
	for i := 0; i < len(rules); i++ {
		lockman.LockObject(ctx, &rules[i])
		defer lockman.ReleaseObject(ctx, &rules[i])
		err := rules[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "rules[i].Delete")
		}
	}
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

func (sg *SSecurityGroup) GetQuotaKeys() quotas.IQuotaKeys {
	return quotas.OwnerIdProjectQuotaKeys(rbacutils.ScopeProject, sg.GetOwnerId())
}

func (sg *SSecurityGroup) GetUsages() []db.IUsage {
	if sg.PendingDeleted || sg.Deleted {
		return nil
	}
	usage := SProjectQuota{Secgroup: 1}
	keys := sg.GetQuotaKeys()
	usage.SetKeys(keys)
	return []db.IUsage{
		&usage,
	}
}
