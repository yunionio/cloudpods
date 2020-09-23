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
	"yunion.io/x/pkg/util/compare"
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
	"yunion.io/x/onecloud/pkg/compute/options"
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
		inAllowList, outAllowList, err := secgroup.GetAllowList()
		if err != nil {
			return q, httperrors.NewGeneralError(errors.Wrapf(err, "GetAllowList"))
		}
		sq := manager.Query().NotEquals("id", secgroup.Id)
		secgroups := []SSecurityGroup{}
		err = db.FetchModelObjects(manager, sq, &secgroups)
		if err != nil {
			return nil, err
		}
		secgroupIds := []string{}
		for i := 0; i < len(secgroups); i++ {
			_inAllowList, _outAllowList, err := secgroups[i].GetAllowList()
			if err != nil {
				return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetAllowList"))
			}
			if !inAllowList.Equals(_inAllowList) || !outAllowList.Equals(_outAllowList) {
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
		admin := (input.Admin != nil && *input.Admin)
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

	if len(input.DBInstanceId) > 0 {
		_, err = validators.ValidateModel(userCred, DBInstanceManager, &input.DBInstanceId)
		if err != nil {
			return nil, err
		}
		sq := DBInstanceSecgroupManager.Query("secgroup_id").Equals("dbinstance_id", input.DBInstanceId)
		q = q.In("id", sq.SubQuery())
	}

	if len(input.CloudregionId) > 0 || len(input.Providers) > 0 || len(input.Brands) > 0 || len(input.CloudaccountId) > 0 {
		caches := SecurityGroupCacheManager.Query()
		filter := api.SecurityGroupCacheListInput{
			ManagedResourceListInput: input.ManagedResourceListInput,
			RegionalFilterListInput:  input.RegionalFilterListInput,
		}
		caches, err = SecurityGroupCacheManager.ListItemFilter(ctx, caches, userCred, filter)
		if err != nil {
			return nil, errors.Wrapf(err, "SecurityGroupCacheManager.ListItemFilter")
		}

		sq := caches.SubQuery()

		q = q.Join(sq, sqlchemy.Equals(q.Field("id"), sq.Field("secgroup_id")))
	}

	// elastic cache
	q, err = manager.ListItemElasticcacheFilter(ctx, q, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "ListItemElasticcacheFilter")
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

func (manager *SSecurityGroupManager) ListItemElasticcacheFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.SecgroupListInput,
) (*sqlchemy.SQuery, error) {
	cacheId := input.ElasticcacheId
	if len(cacheId) > 0 {
		cache, _, err := ValidateElasticcacheResourceInput(userCred, input.ELasticcacheResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateElasticcacheResourceInput")
		}
		cacheId := cache.GetId()
		filters := []sqlchemy.ICondition{}
		filters = append(filters, sqlchemy.In(q.Field("id"), ElasticcachesecgroupManager.Query("secgroup_id").Equals("elasticcache_id", cacheId).SubQuery()))
		q = q.Filter(sqlchemy.OR(filters...))
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
	switch field {
	case "provider", "brand":
		accountQuery := CloudaccountManager.Query(field, "id").Distinct().SubQuery()
		providers := CloudproviderManager.Query("id", "cloudaccount_id").SubQuery()
		caches := SecurityGroupCacheManager.Query("manager_id", "secgroup_id").SubQuery()
		q.AppendField(accountQuery.Field(field)).Distinct()
		q = q.Join(caches, sqlchemy.Equals(q.Field("id"), caches.Field("secgroup_id")))
		q = q.Join(providers, sqlchemy.Equals(providers.Field("id"), caches.Field("manager_id")))
		q = q.Join(accountQuery, sqlchemy.Equals(accountQuery.Field("id"), providers.Field("cloudaccount_id")))
		return q, nil
	case "region":
		regionQuery := CloudregionManager.Query("name", "id").SubQuery()
		caches := SecurityGroupCacheManager.Query("cloudregion_id", "secgroup_id").SubQuery()
		q.AppendField(regionQuery.Field("name").Label("region")).Distinct()
		q = q.Join(caches, sqlchemy.Equals(q.Field("id"), caches.Field("secgroup_id")))
		q = q.Join(regionQuery, sqlchemy.Equals(caches.Field("cloudregion_id"), regionQuery.Field("id")))
		return q, nil
	case "account":
		accountQuery := CloudaccountManager.Query("name", "id").Distinct().SubQuery()
		providers := CloudproviderManager.Query("id", "cloudaccount_id").SubQuery()
		caches := SecurityGroupCacheManager.Query("manager_id", "secgroup_id").SubQuery()
		q.AppendField(accountQuery.Field("name").Label("account")).Distinct()
		q = q.Join(caches, sqlchemy.Equals(q.Field("id"), caches.Field("secgroup_id")))
		q = q.Join(providers, sqlchemy.Equals(providers.Field("id"), caches.Field("manager_id")))
		q = q.Join(accountQuery, sqlchemy.Equals(accountQuery.Field("id"), providers.Field("cloudaccount_id")))
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

func (self *SSecurityGroup) GetKvmGuests() ([]SGuest, error) {
	guests := []SGuest{}
	q := self.GetGuestsQuery().Equals("hypervisor", api.HYPERVISOR_KVM)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return guests, nil
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

	peerSecgroupIds := []string{}
	for _, rule := range rules {
		if len(rule.PeerSecgroupId) > 0 {
			peerSecgroupIds = append(peerSecgroupIds, rule.PeerSecgroupId)
		}
	}

	peerMaps, err := db.FetchIdNameMap2(SecurityGroupManager, peerSecgroupIds)
	if err != nil {
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
		_rules := []api.SecgroupRuleDetails{}
		_inRules := []api.SecgroupRuleDetails{}
		_outRules := []api.SecgroupRuleDetails{}
		for j := range rules {
			rule := api.SecgroupRuleDetails{}
			jsonutils.Update(&rule, rules[j])
			if len(rules[j].PeerSecgroupId) > 0 {
				rule.PeerSecgroup, _ = peerMaps[rules[j].PeerSecgroupId]
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

	input.Status = api.SECGROUP_STATUS_READY

	for i := range input.Rules {
		err = input.Rules[i].Check()
		if err != nil {
			return input, httperrors.NewInputParameterError("rule %d is invalid: %s", i, err)
		}
	}

	input.SharableVirtualResourceCreateInput, err = manager.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, err
	}

	pendingUsage := SProjectQuota{Secgroup: 1}
	quotaKey := quotas.OwnerIdProjectQuotaKeys(rbacutils.ScopeProject, ownerId)
	pendingUsage.SetKeys(quotaKey)
	err = quotas.CheckSetPendingQuota(ctx, userCred, &pendingUsage)
	if err != nil {
		return input, httperrors.NewOutOfQuotaError("%s", err)
	}

	return input, nil
}

func (self *SSecurityGroup) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	quota := &SProjectQuota{Secgroup: 1}
	quota.SetKeys(self.GetQuotaKeys())
	err := quotas.CancelPendingUsage(ctx, userCred, quota, quota, true)
	if err != nil {
		log.Errorf("Secgroup CancelPendingUsage fail %s", err)
	}

	input := &api.SSecgroupCreateInput{}
	data.Unmarshal(input)

	for _, r := range input.Rules {
		rule := &SSecurityGroupRule{
			Priority:    int64(*r.Priority),
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

func (manager *SSecurityGroupManager) FetchSecgroupById(secId string) (*SSecurityGroup, error) {
	secgrp, err := manager.FetchById(secId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById(%s)", secId)
	}
	return secgrp.(*SSecurityGroup), nil
}

func (self *SSecurityGroup) getSecurityRules() ([]SSecurityGroupRule, error) {
	secgrouprules := SecurityGroupRuleManager.Query().SubQuery()
	sql := secgrouprules.Query().Filter(sqlchemy.Equals(secgrouprules.Field("secgroup_id"), self.Id)).Desc("priority")
	rules := []SSecurityGroupRule{}
	err := db.FetchModelObjects(SecurityGroupRuleManager, sql, &rules)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return rules, nil
}

func (self *SSecurityGroup) GetSecuritRuleSet() (cloudprovider.SecurityRuleSet, error) {
	ruleSet := cloudprovider.SecurityRuleSet{}
	rules, err := self.getSecurityRules()
	if err != nil {
		return ruleSet, errors.Wrapf(err, "getSecurityRules")
	}
	for i := range rules {
		//这里没必要拆分为单个单个的端口,到公有云那边适配
		rule, err := rules[i].toRule()
		if err != nil {
			return nil, errors.Wrapf(err, "toRule")
		}
		ruleSet = append(ruleSet, cloudprovider.SecurityRule{SecurityRule: *rule, ExternalId: rules[i].Id})
	}
	return ruleSet, nil
}

func (self *SSecurityGroup) GetSecRules() ([]secrules.SecurityRule, error) {
	rules := make([]secrules.SecurityRule, 0)
	_rules, err := self.getSecurityRules()
	if err != nil {
		return nil, errors.Wrapf(err, "getSecurityRules()")
	}
	for _, _rule := range _rules {
		//这里没必要拆分为单个单个的端口,到公有云那边适配
		rule, err := _rule.toRule()
		if err != nil {
			return nil, errors.Wrapf(err, "toRule")
		}
		rules = append(rules, *rule)
	}
	return rules, nil
}

func (self *SSecurityGroup) getSecurityRuleString() (string, error) {
	secgrouprules, err := self.getSecurityRules()
	if err != nil {
		return "", errors.Wrapf(err, "getSecurityRules()")
	}
	var rules []string
	for _, rule := range secgrouprules {
		rules = append(rules, rule.String())
	}
	return strings.Join(rules, SECURITY_GROUP_SEPARATOR), nil
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

func (self *SSecurityGroup) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "purge")
}

func (self *SSecurityGroup) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return nil, err
	}
	return nil, self.StartDeleteSecurityGroupTask(ctx, userCred, true, "")
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
		return nil, httperrors.NewInputParameterError("%v", err)
	}
	if err := rule.ValidateRule(); err != nil {
		return nil, httperrors.NewInputParameterError("%v", err)
	}
	if err := SecurityGroupRuleManager.TableSpec().Insert(ctx, secgrouprule); err != nil {
		return nil, httperrors.NewInputParameterError("%v", err)
	}
	self.DoSync(ctx, userCred)
	return nil, nil
}

func (self *SSecurityGroup) AllowPerformClone(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (self *SSecurityGroup) PerformClone(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SecurityGroupCloneInput) (api.SecurityGroupCloneInput, error) {
	if len(input.Name) == 0 {
		return input, httperrors.NewMissingParameterError("name")
	}
	ownerId := &db.SOwnerId{
		DomainId:  userCred.GetProjectDomainId(),
		ProjectId: userCred.GetProjectId(),
	}
	pendingUsage := SProjectQuota{Secgroup: 1}
	quotaKey := quotas.OwnerIdProjectQuotaKeys(rbacutils.ScopeProject, ownerId)
	pendingUsage.SetKeys(quotaKey)
	err := quotas.CheckSetPendingQuota(ctx, userCred, &pendingUsage)
	if err != nil {
		return input, httperrors.NewOutOfQuotaError("%s", err)
	}

	secgroup := &SSecurityGroup{}
	secgroup.SetModelManager(SecurityGroupManager, secgroup)

	secgroup.Name = input.Name
	secgroup.Description = input.Description
	secgroup.Status = api.SECGROUP_STATUS_READY
	secgroup.ProjectId = userCred.GetProjectId()
	secgroup.DomainId = userCred.GetProjectDomainId()

	err = func() error {
		lockman.LockClass(ctx, SecurityGroupManager, "name")
		defer lockman.ReleaseClass(ctx, SecurityGroupManager, "name")

		input.Name, err = db.GenerateName(ctx, SecurityGroupManager, ownerId, input.Name)
		if err != nil {
			return err
		}

		return SecurityGroupManager.TableSpec().Insert(ctx, secgroup)
	}()
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrapf(err, "Insert"))
	}

	secgrouprules, err := self.getSecurityRules()
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrapf(err, "getSecurityRules"))
	}
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
			return input, err
		}
	}

	quota := &SProjectQuota{Secgroup: 1}
	quota.SetKeys(secgroup.GetQuotaKeys())
	err = quotas.CancelPendingUsage(ctx, userCred, quota, quota, true)
	if err != nil {
		log.Errorf("Secgroup CancelPendingUsage fail %s", err)
	}

	logclient.AddActionLogWithContext(ctx, secgroup, logclient.ACT_CREATE, secgroup.GetShortDesc(ctx), userCred, true)
	return input, nil
}

func (self *SSecurityGroup) AllowPerformMerge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "merge")
}

func (self *SSecurityGroup) PerformMerge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SecgroupMergeInput) (jsonutils.JSONObject, error) {
	if len(input.SecgroupIds) == 0 {
		return nil, httperrors.NewMissingParameterError("secgroup_ids")
	}
	inAllowList, outAllowList, err := self.GetAllowList()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetAllowList"))
	}
	secgroups := []*SSecurityGroup{}
	for _, secgroupId := range input.SecgroupIds {
		_secgroup, err := SecurityGroupManager.FetchByIdOrName(userCred, secgroupId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("secgroup", secgroupId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		secgroup := _secgroup.(*SSecurityGroup)
		secgroup.SetModelManager(SecurityGroupManager, secgroup)
		_inAllowList, _outAllowList, err := secgroup.GetAllowList()
		if err != nil {
			return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetAllowList"))
		}
		if !inAllowList.Equals(_inAllowList) {
			return nil, httperrors.NewUnsupportOperationError("secgroup %s rules not equals %s rules", secgroup.Name, self.Name)
		}
		if !outAllowList.Equals(_outAllowList) {
			return nil, httperrors.NewUnsupportOperationError("secgroup %s rules not equals %s rules", secgroup.Name, self.Name)
		}
		secgroups = append(secgroups, secgroup)
	}

	for i := 0; i < len(secgroups); i++ {
		secgroup := secgroups[i]
		err := self.mergeSecurityGroupCache(secgroup)
		if err != nil {
			return nil, httperrors.NewGeneralError(errors.Wrapf(err, "mergeSecurityGroupCache"))
		}

		err = self.mergeGuestSecurityGroup(ctx, userCred, secgroup)
		if err != nil {
			return nil, httperrors.NewGeneralError(errors.Wrapf(err, "mergeGuestSecurityGroup"))
		}
		secgroup.RealDelete(ctx, userCred)
	}
	self.DoSync(ctx, userCred)
	return nil, nil
}

func (self *SSecurityGroup) GetAllowList() (secrules.SecurityRuleSet, secrules.SecurityRuleSet, error) {
	in, out := secrules.SecurityRuleSet{*secrules.MustParseSecurityRule("in:deny any")}, secrules.SecurityRuleSet{*secrules.MustParseSecurityRule("out:allow any")}
	rules, err := self.GetSecRules()
	if err != nil {
		return in, out, errors.Wrapf(err, "GetSecRules")
	}
	for i := range rules {
		if rules[i].Direction == secrules.DIR_IN {
			in = append(in, rules[i])
		} else {
			in = append(in, rules[i])
		}
	}
	return in.AllowList(), out.AllowList(), nil
}

func (self *SSecurityGroup) mergeSecurityGroupCache(secgroup *SSecurityGroup) error {
	caches, err := secgroup.GetSecurityGroupCaches()
	if err != nil {
		return errors.Wrapf(err, "GetSecurityGroupCaches")
	}
	for i := 0; i < len(caches); i++ {
		cache := caches[i]
		_, err := db.Update(&cache, func() error {
			cache.SecgroupId = self.Id
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "db.Update")
		}
	}
	return nil
}

func (self *SSecurityGroup) mergeGuestSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, fade *SSecurityGroup) error {
	guests := fade.GetGuests()
	for i := 0; i < len(guests); i++ {
		secgroups, err := guests[i].GetSecgroups()
		if err != nil {
			return errors.Wrapf(err, "GetSecgroups for guest %s(%s)", guests[i].Name, guests[i].Id)
		}
		secgroupIds := []string{}
		for i := range secgroups {
			if secgroups[i].Id == fade.Id {
				continue
			}
			if utils.IsInStringArray(secgroups[i].Id, secgroupIds) {
				continue
			}
			secgroupIds = append(secgroupIds, secgroups[i].Id)
		}
		if !utils.IsInStringArray(self.Id, secgroupIds) {
			secgroupIds = append(secgroupIds, self.Id)
		}
		err = guests[i].saveSecgroups(ctx, userCred, secgroupIds)
		if err != nil {
			return errors.Wrap(err, "saveSecgroups")
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

func (self *SSecurityGroup) removeRules(ruleIds []string, result *compare.SyncResult) {
	if len(ruleIds) == 0 {
		return
	}
	rules := []SSecurityGroupRule{}
	q := SecurityGroupRuleManager.Query().In("id", ruleIds)
	err := db.FetchModelObjects(SecurityGroupRuleManager, q, &rules)
	if err != nil {
		result.DeleteError(errors.Wrapf(err, "db.FetchModelObjects"))
		return
	}
	for i := range rules {
		err = rules[i].Delete(context.TODO(), nil)
		if err != nil {
			result.DeleteError(errors.Wrapf(err, "delte rule %s", rules[i].Id))
			continue
		}
		result.Delete()
	}
}

func (self *SSecurityGroup) SyncSecurityGroupRules(ctx context.Context, userCred mcclient.TokenCredential, src cloudprovider.SecRuleInfo) ([]SSecurityGroupRule, compare.SyncResult) {
	result := compare.SyncResult{}
	localRules, err := self.GetSecuritRuleSet()
	if err != nil {
		result.Error(errors.Wrapf(err, "GetSecuritRuleSet"))
		return nil, result
	}

	dest := cloudprovider.NewSecRuleInfo(GetRegionDriver(api.CLOUD_PROVIDER_ONECLOUD))
	dest.Rules = localRules

	_, inAdds, outAdds, inDels, outDels := cloudprovider.CompareRules(src, dest, false)
	if len(inAdds)+len(inDels)+len(outAdds)+len(outDels) == 0 {
		return nil, result
	}

	ruleIds := []string{}
	for _, dels := range [][]cloudprovider.SecurityRule{inDels, outDels} {
		for i := range dels {
			if len(dels[i].ExternalId) > 0 {
				ruleIds = append(ruleIds, dels[i].ExternalId)
			}
		}
	}

	self.removeRules(ruleIds, &result)

	rules := []SSecurityGroupRule{}
	for _, adds := range [][]cloudprovider.SecurityRule{inAdds, outAdds} {
		for i := range adds {
			rule, isNeedFix, err := self.newFromCloudSecurityGroupRule(ctx, userCred, adds[i])
			if err != nil {
				result.AddError(errors.Wrapf(err, "newFromCloudSecurityGroupRule"))
				continue
			}
			if isNeedFix && rule != nil {
				rules = append(rules, *rule)
			}
			result.Add()
		}
	}

	log.Infof("Sync Rules for Secgroup %s(%s) result: %s", self.Name, self.Id, result.Result())
	return rules, result
}

func (manager *SSecurityGroupManager) newFromCloudSecgroup(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extSec cloudprovider.ICloudSecurityGroup) (*SSecurityGroup, []SSecurityGroupRule, error) {
	dest := cloudprovider.NewSecRuleInfo(GetRegionDriver(provider.Provider))
	var err error
	dest.Rules, err = extSec.GetRules()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "extSec.GetRules")
	}
	src := cloudprovider.NewSecRuleInfo(GetRegionDriver(api.CLOUD_PROVIDER_ONECLOUD))

	if options.Options.EnableAutoMergeSecurityGroup {
		// 查询与provider在同域的安全组，比对寻找一个与云上安全组规则相同的安全组
		secgroups := []SSecurityGroup{}
		q := manager.Query().Equals("domain_id", provider.DomainId)
		err = db.FetchModelObjects(manager, q, &secgroups)
		if err != nil {
			return nil, nil, errors.Wrap(err, "db.FetchModelObjects")
		}
		for i := range secgroups {
			src.Rules, err = secgroups[i].GetSecuritRuleSet()
			if err != nil {
				log.Warningf("GetSecuritRuleSet %s(%s) error: %v", secgroups[i].Name, secgroups[i].Id, err)
				continue
			}
			_, inAdds, outAdds, inDels, outDels := cloudprovider.CompareRules(src, dest, false)
			if len(inAdds) == 0 && len(outAdds) == 0 && len(inDels) == 0 && len(outDels) == 0 {
				return &secgroups[i], nil, nil
			}
		}
	}

	secgroup := SSecurityGroup{}
	secgroup.SetModelManager(manager, &secgroup)

	secgroup.Status = api.SECGROUP_STATUS_READY
	secgroup.Description = extSec.GetDescription()
	secgroup.ProjectId = provider.ProjectId
	secgroup.DomainId = provider.DomainId

	err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		secgroup.Name, err = db.GenerateName(ctx, manager, userCred, extSec.GetName())
		if err != nil {
			return errors.Wrapf(err, "db.GenerateName")
		}

		return manager.TableSpec().Insert(ctx, &secgroup)
	}()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Insert")
	}

	rules, _ := secgroup.SyncSecurityGroupRules(ctx, userCred, dest)

	db.OpsLog.LogEvent(&secgroup, db.ACT_CREATE, secgroup.GetShortDesc(ctx), userCred)
	return &secgroup, rules, nil
}

func (manager *SSecurityGroupManager) DelaySync(ctx context.Context, userCred mcclient.TokenCredential, idStr string) error {
	secgrp, err := manager.FetchSecgroupById(idStr)
	if err != nil {
		return errors.Wrapf(err, "FetchSecgroupById(%s)", idStr)
	}
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
		guests, err := secgrp.GetKvmGuests()
		if err != nil {
			return errors.Wrapf(err, "GetKvmGuests")
		}
		for _, guest := range guests {
			guest.StartSyncTask(ctx, userCred, true, "")
		}
	}
	return secgrp.StartSecurityGroupSyncRulesTask(ctx, userCred, "")
}

func (self *SSecurityGroup) StartSecurityGroupSyncRulesTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "SecurityGroupSyncRulesTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(userCred, api.SECGROUP_STATUS_SYNC_RULES, "")
	task.ScheduleRun(nil)
	return nil
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
		return errors.Wrapf(err, `manager.FetchById("default")`)
	}
	if errors.Cause(err) == sql.ErrNoRows {
		log.Debugf("Init default secgroup")
		secGrp := &SSecurityGroup{}
		secGrp.SetModelManager(manager, secGrp)
		secGrp.Id = "default"
		secGrp.Name = "Default"
		secGrp.Status = api.SECGROUP_STATUS_READY
		secGrp.ProjectId = auth.AdminCredential().GetProjectId()
		secGrp.DomainId = auth.AdminCredential().GetProjectDomainId()
		// secGrp.IsEmulated = false
		secGrp.IsPublic = true
		secGrp.Deleted = false
		secGrp.PublicScope = string(rbacutils.ScopeSystem)
		err = manager.TableSpec().InsertOrUpdate(context.TODO(), secGrp)
		if err != nil {
			return errors.Wrapf(err, "Insert default secgroup")
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
			return errors.Wrapf(err, "Insert default secgroup rule")
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

func (self *SSecurityGroup) GetSecurityGroupReferences() ([]SSecurityGroup, error) {
	sq := SecurityGroupRuleManager.Query("secgroup_id").Equals("peer_secgroup_id", self.Id).Distinct().SubQuery()
	q := SecurityGroupManager.Query().In("id", sq)
	groups := []SSecurityGroup{}
	err := db.FetchModelObjects(SecurityGroupManager, q, &groups)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return groups, nil
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
	references, err := self.GetSecurityGroupReferences()
	if err != nil {
		return httperrors.NewGeneralError(err)
	}
	if len(references) > 0 {
		return httperrors.NewNotEmptyError("the other security group is in use")
	}
	return self.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SSecurityGroup) GetSecurityGroupCaches() ([]SSecurityGroupCache, error) {
	caches := []SSecurityGroupCache{}
	q := SecurityGroupCacheManager.Query()
	q = q.Filter(sqlchemy.Equals(q.Field("secgroup_id"), self.Id))
	err := db.FetchModelObjects(SecurityGroupCacheManager, q, &caches)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return caches, nil
}

func (self *SSecurityGroup) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteSecurityGroupTask(ctx, userCred, false, "")
}

func (self *SSecurityGroup) StartDeleteSecurityGroupTask(ctx context.Context, userCred mcclient.TokenCredential, isPurge bool, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewBool(isPurge), "purge")
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

func (self *SSecurityGroup) AllowGetDetailsReferences(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsProjectAllowGetSpec(userCred, self, "references")
}

// 获取引用信息
func (self *SSecurityGroup) GetDetailsReferences(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) ([]cloudprovider.SecurityGroupReference, error) {
	groups, err := self.GetSecurityGroupReferences()
	if err != nil {
		return nil, errors.Wrapf(err, "GetSecurityGroupReferences")
	}
	ret := []cloudprovider.SecurityGroupReference{}
	for i := range groups {
		ret = append(ret, cloudprovider.SecurityGroupReference{
			Id:   groups[i].Id,
			Name: groups[i].Name,
		})
	}
	return ret, nil
}

func (self *SSecurityGroup) AllowPerformImportRules(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "import-rules")
}

func (self *SSecurityGroup) PerformImportRules(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SecgroupImportRulesInput) (jsonutils.JSONObject, error) {
	for i := range input.Rules {
		if input.Rules[i].Priority == nil {
			return nil, httperrors.NewMissingParameterError("priority")
		}
		err := input.Rules[i].Check()
		if err != nil {
			return nil, httperrors.NewInputParameterError("rule %d is invalid: %s", i+1, err)
		}
	}
	for _, r := range input.Rules {
		rule := &SSecurityGroupRule{
			Priority:    int64(*r.Priority),
			Protocol:    r.Protocol,
			Ports:       r.Ports,
			Direction:   r.Direction,
			CIDR:        r.CIDR,
			Action:      r.Action,
			Description: r.Description,
		}
		rule.SecgroupId = self.Id

		err := SecurityGroupRuleManager.TableSpec().Insert(ctx, rule)
		if err != nil {
			return nil, httperrors.NewGeneralError(errors.Wrapf(err, "Insert rule"))
		}
	}
	return nil, nil
}
