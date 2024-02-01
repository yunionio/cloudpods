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
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSecurityGroupManager struct {
	db.SSharableVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	SVpcResourceBaseManager
	SGlobalVpcResourceBaseManager
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
	SecurityGroupManager.SetVirtualObject(SecurityGroupManager)
}

const (
	SECURITY_GROUP_SEPARATOR = ";"
)

type SSecurityGroup struct {
	db.SSharableVirtualResourceBase
	db.SExternalizedResourceBase
	IsDirty bool `nullable:"false" default:"false"`

	SManagedResourceBase

	SCloudregionResourceBase `width:"36" charset:"ascii" nullable:"false" list:"domain" create:"domain_required" default:"default"`

	SGlobalVpcResourceBase `width:"36" charset:"ascii" list:"user" create:"domain_optional" json:"globalvpc_id"`

	SVpcResourceBase `wdith:"36" charset:"ascii" nullable:"true" list:"domain" create:"domain_optional" update:""`
}

func (self *SSecurityGroup) GetCloudproviderId() string {
	return self.ManagerId
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

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, input.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	if len(input.VpcId) > 0 {
		vpcObj, err := validators.ValidateModel(userCred, VpcManager, &input.VpcId)
		if err != nil {
			return nil, err
		}
		vpc := vpcObj.(*SVpc)
		region, err := vpc.GetRegion()
		if err != nil {
			return nil, err
		}
		filter, err := region.GetDriver().GetSecurityGroupFilter(vpc)
		if err != nil {
			return nil, err
		}
		q = filter(q)
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
		// admin := (input.Admin != nil && *input.Admin)
		allowScope, _ := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
		if allowScope == rbacscope.ScopeSystem || allowScope == rbacscope.ScopeDomain {
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

	if len(input.ElasticcacheId) > 0 {
		_, err = validators.ValidateModel(userCred, ElasticcacheManager, &input.ElasticcacheId)
		if err != nil {
			return nil, err
		}
		sq := ElasticcachesecgroupManager.Query("secgroup_id").Equals("elasticcache_id", input.ElasticcacheId).Distinct()
		q = q.In("id", sq.SubQuery())
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

	if input.CloudEnv == "onpremise" {
		q = q.IsNullOrEmpty("manager_id")
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

	q, err = manager.SVpcResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SGlobalVpcResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SSecurityGroup) GetChangeOwnerCandidateDomainIds() []string {
	candidates := [][]string{}
	vpc, _ := self.GetVpc()
	if vpc != nil {
		candidates = append(candidates, vpc.GetChangeOwnerCandidateDomainIds())
	}
	return db.ISharableMergeChangeOwnerCandidateDomainIds(self, candidates...)
}

func (self *SSecurityGroup) GetGuestsQuery() *sqlchemy.SQuery {
	guests := GuestManager.Query().SubQuery()
	return guests.Query().Filter(
		sqlchemy.OR(
			sqlchemy.Equals(guests.Field("secgrp_id"), self.Id),
			sqlchemy.Equals(guests.Field("admin_secgrp_id"), self.Id),
			sqlchemy.In(guests.Field("id"), GuestsecgroupManager.Query("guest_id").Equals("secgroup_id", self.Id).SubQuery()),
		),
	).Filter(sqlchemy.NotIn(guests.Field("hypervisor"), []string{api.HYPERVISOR_POD, api.HYPERVISOR_BAREMETAL, api.HYPERVISOR_ESXI}))
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

func (self *SSecurityGroup) getDesc() *api.SecgroupJsonDesc {
	return &api.SecgroupJsonDesc{
		Id:   self.Id,
		Name: self.Name,
	}
}

func (self *SSecurityGroup) ClearRuleDirty() error {
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"update %s set is_dirty = false where secgroup_id = ?",
			SecurityGroupRuleManager.TableSpec().Name(),
		), self.Id,
	)
	return err
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
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	vpcRows := manager.SVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	globalVpcRows := manager.SGlobalVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	secgroupIds := make([]string, len(objs))
	secgroups := make([]*SSecurityGroup, len(objs))
	for i := range rows {
		rows[i] = api.SecgroupDetails{
			SharableVirtualResourceDetails: virtRows[i],
			VpcResourceInfo:                vpcRows[i],
			GlobalVpcResourceInfo:          globalVpcRows[i],
		}
		rows[i].ManagedResourceInfo = managerRows[i]
		rows[i].CloudregionResourceInfo = regionRows[i]
		secgroup := objs[i].(*SSecurityGroup)
		secgroupIds[i] = secgroup.Id
		secgroups[i] = secgroup
	}

	guests := []SGuest{}
	q := GuestManager.Query().IsFalse("pending_deleted")
	q = q.Filter(sqlchemy.OR(
		sqlchemy.In(q.Field("secgrp_id"), secgroupIds),
		sqlchemy.In(q.Field("admin_secgrp_id"), secgroupIds),
	))

	ownerId, queryScope, err, _ := db.FetchCheckQueryOwnerScope(ctx, userCred, query, GuestManager, policy.PolicyActionList, true)
	if err != nil {
		log.Errorf("FetchCheckQueryOwnerScope error: %v", err)
		return rows
	}

	q = GuestManager.FilterByOwner(q, GuestManager, userCred, ownerId, queryScope)
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

	sq := GuestManager.Query("id").IsFalse("pending_deleted")
	sq = GuestManager.FilterByOwner(sq, GuestManager, userCred, ownerId, queryScope)

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

	totalCnt, err := manager.TotalCnt(secgroupIds)
	if err != nil {
		return rows
	}
	for i := range rows {
		rows[i].GuestCnt, _ = normalGuestMaps[secgroupIds[i]]
		rows[i].AdminGuestCnt, _ = adminGuestMaps[secgroupIds[i]]
		rows[i].SystemGuestCnt, _ = systemGuestMaps[secgroupIds[i]]
		if cnt, ok := totalCnt[secgroupIds[i]]; ok {
			rows[i].TotalCnt = cnt.TotalCnt
		}
	}
	return rows
}

func (manager *SSecurityGroupManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.SSecgroupCreateInput,
) (*api.SSecgroupCreateInput, error) {
	if len(input.VpcId) == 0 {
		input.VpcId = api.DEFAULT_VPC_ID
	}

	vpcObj, err := validators.ValidateModel(userCred, VpcManager, &input.VpcId)
	if err != nil {
		return nil, err
	}
	vpc := vpcObj.(*SVpc)
	input.CloudproviderId = vpc.ManagerId
	input.CloudregionId = vpc.CloudregionId
	input.GlobalvpcId = vpc.GlobalvpcId

	region, err := vpc.GetRegion()
	if err != nil {
		return nil, err
	}

	driver := region.GetDriver()

	input, err = driver.ValidateCreateSecurityGroupInput(ctx, userCred, input)
	if err != nil {
		return nil, err
	}

	input.SharableVirtualResourceCreateInput, err = manager.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, err
	}

	pendingUsage := SProjectQuota{Secgroup: 1}
	quotaKey := quotas.OwnerIdProjectQuotaKeys(rbacscope.ScopeProject, ownerId)
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

	self.StartSecurityGroupCreateTask(ctx, userCred, input.Rules, "")
}

func (self *SSecurityGroup) StartSecurityGroupCreateTask(ctx context.Context, userCred mcclient.TokenCredential, rules []api.SSecgroupRuleCreateInput, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Set("rules", jsonutils.Marshal(rules))
	self.SetStatus(userCred, apis.STATUS_CREATING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "SecurityGroupCreateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (manager *SSecurityGroupManager) FetchSecgroupById(secId string) (*SSecurityGroup, error) {
	secgrp, err := manager.FetchById(secId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById(%s)", secId)
	}
	return secgrp.(*SSecurityGroup), nil
}

func (self *SSecurityGroup) GetSecurityRules() ([]SSecurityGroupRule, error) {
	q := SecurityGroupRuleManager.Query().Equals("secgroup_id", self.Id).Desc("priority")
	rules := []SSecurityGroupRule{}
	err := db.FetchModelObjects(SecurityGroupRuleManager, q, &rules)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return rules, nil
}

func (self *SSecurityGroup) getSecurityRuleString() (string, error) {
	secgrouprules, err := self.GetSecurityRules()
	if err != nil {
		return "", errors.Wrapf(err, "getSecurityRules()")
	}
	var rules []string
	for _, rule := range secgrouprules {
		rules = append(rules, rule.String())
	}
	return strings.Join(rules, SECURITY_GROUP_SEPARATOR), nil
}

func totalSecurityGroupCount(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider) (int, error) {
	q := SecurityGroupManager.Query()

	switch scope {
	case rbacscope.ScopeSystem:
		// do nothing
	case rbacscope.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	case rbacscope.ScopeProject:
		q = q.Equals("tenant_id", ownerId.GetProjectId())
	}

	return q.CountWithError()
}

func (self *SSecurityGroup) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.SecurityGroupSyncstatusInput) (jsonutils.JSONObject, error) {
	return nil, self.StartSecurityGroupSyncTask(ctx, userCred, "")
}

func (self *SSecurityGroup) StartSecurityGroupSyncTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	self.SetStatus(userCred, apis.STATUS_SYNC_STATUS, "")
	task, err := taskman.TaskManager.NewTask(ctx, "SecurityGroupSyncTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SSecurityGroup) StartSecurityGroupRuleCreateTask(ctx context.Context, userCred mcclient.TokenCredential, ruleId, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Set("rule_id", jsonutils.NewString(ruleId))
	task, err := taskman.TaskManager.NewTask(ctx, "SecurityGroupRuleCreateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SSecurityGroup) StartSecurityGroupRuleDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, ruleId, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Set("rule_id", jsonutils.NewString(ruleId))
	task, err := taskman.TaskManager.NewTask(ctx, "SecurityGroupRuleDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
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

func (self *SSecurityGroup) PerformClone(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SecurityGroupCloneInput) (api.SecurityGroupCloneInput, error) {
	if len(input.Name) == 0 {
		return input, httperrors.NewMissingParameterError("name")
	}
	ownerId := &db.SOwnerId{
		DomainId:  userCred.GetProjectDomainId(),
		ProjectId: userCred.GetProjectId(),
	}
	pendingUsage := SProjectQuota{Secgroup: 1}
	quotaKey := quotas.OwnerIdProjectQuotaKeys(rbacscope.ScopeProject, ownerId)
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

	secgrouprules, err := self.GetSecurityRules()
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

/*func (self *SSecurityGroup) GetAllowList() (secrules.SecurityRuleSet, secrules.SecurityRuleSet, error) {
	in := secrules.SecurityRuleSet{}
	out := secrules.SecurityRuleSet{}
	rules, err := self.GetSecurityRules()
	if err != nil {
		return in, out, errors.Wrapf(err, "GetSecRules")
	}
	for i := range rules {
		r, _ := rules[i].toRule()
		if rules[i].Direction == secrules.DIR_IN {
			in = append(in, *r)
		} else {
			out = append(out, *r)
		}
	}
	in = append(in, *secrules.MustParseSecurityRule("in:deny any"))
	out = append(out, *secrules.MustParseSecurityRule("out:allow any"))
	return in.AllowList(), out.AllowList(), nil
}*/

func (self *SSecurityGroup) clearRules() error {
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"delete from %s where secgroup_id = ?",
			SecurityGroupRuleManager.TableSpec().Name(),
		), self.Id,
	)
	return err
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
	_, err := manager.FetchById(api.SECGROUP_DEFAULT_ID)
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrapf(err, `manager.FetchById("default")`)
	}
	if errors.Cause(err) == sql.ErrNoRows {
		log.Debugf("Init default secgroup")
		secGrp := &SSecurityGroup{}
		secGrp.SetModelManager(manager, secGrp)
		secGrp.Id = api.SECGROUP_DEFAULT_ID
		secGrp.Name = "Default"
		secGrp.Status = api.SECGROUP_STATUS_READY
		secGrp.ProjectId = auth.AdminCredential().GetProjectId()
		secGrp.DomainId = auth.AdminCredential().GetProjectDomainId()
		// secGrp.IsEmulated = false
		secGrp.IsPublic = true
		secGrp.Deleted = false
		secGrp.PublicScope = string(rbacscope.ScopeSystem)
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
		defRule.SecgroupId = api.SECGROUP_DEFAULT_ID
		err = SecurityGroupRuleManager.TableSpec().Insert(context.TODO(), &defRule)
		if err != nil {
			return errors.Wrapf(err, "Insert default secgroup rule")
		}
	}
	guests := make([]SGuest, 0)
	q := GuestManager.Query().Equals("hypervisor", api.HYPERVISOR_KVM).IsNullOrEmpty("secgrp_id")
	err = db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("fetch guests without secgroup fail %s", err)
		return err
	}
	for i := 0; i < len(guests); i += 1 {
		db.Update(&guests[i], func() error {
			guests[i].SecgrpId = api.SECGROUP_DEFAULT_ID
			return nil
		})
	}
	if options.Options.CleanUselessKvmSecurityGroup {
		q := SecurityGroupManager.Query()
		q = q.Filter(
			sqlchemy.AND(
				sqlchemy.Equals(q.Field("cloudregion_id"), api.DEFAULT_REGION_ID),
				sqlchemy.NotEquals(q.Field("id"), api.SECGROUP_DEFAULT_ID),
				sqlchemy.NotIn(q.Field("id"), GuestManager.Query("secgrp_id").IsNotNull("secgrp_id").SubQuery()),
				sqlchemy.NotIn(q.Field("id"), GuestManager.Query("admin_secgrp_id").IsNotNull("admin_secgrp_id").SubQuery()),
				sqlchemy.NotIn(q.Field("id"), GuestsecgroupManager.Query("secgroup_id").IsNotNull("secgroup_id").SubQuery()),
				sqlchemy.NotIn(q.Field("id"), GuestsecgroupManager.Query("secgroup_id").IsNotNull("secgroup_id").SubQuery()),
				sqlchemy.NotIn(q.Field("id"), DBInstanceSecgroupManager.Query("secgroup_id").IsNotNull("secgroup_id").SubQuery()),
				sqlchemy.NotIn(q.Field("id"), ElasticcachesecgroupManager.Query("secgroup_id").IsNotNull("secgroup_id").SubQuery()),
			),
		)
		secgroups := []SSecurityGroup{}
		err := db.FetchModelObjects(SecurityGroupManager, q, &secgroups)
		if err != nil {
			return err
		}
		ctx := context.Background()
		ctx = context.WithValue(ctx, "clean", "kvm security groups")
		userCred := auth.GetAdminSession(ctx, options.Options.Region).GetToken()
		for i := range secgroups {
			err = secgroups[i].RealDelete(ctx, userCred)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *SSecurityGroup) GetRegionDriver() (IRegionDriver, error) {
	if len(self.ManagerId) > 0 {
		manager, err := self.GetCloudprovider()
		if err != nil {
			return nil, errors.Wrapf(err, "GetCloudprovider")
		}
		return GetRegionDriver(manager.Provider), nil
	}
	return GetRegionDriver(api.CLOUD_PROVIDER_ONECLOUD), nil
}

func (self *SSecurityGroup) GetRegion() (*SCloudregion, error) {
	regionObj, err := CloudregionManager.FetchById(self.CloudregionId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById")
	}
	return regionObj.(*SCloudregion), nil
}

func (self *SSecurityGroup) GetGlobalVpc() (*SGlobalVpc, error) {
	vpc, err := GlobalVpcManager.FetchById(self.GlobalvpcId)
	if err != nil {
		return nil, err
	}
	return vpc.(*SGlobalVpc), nil
}

func (self *SSecurityGroup) GetISecurityGroup(ctx context.Context) (cloudprovider.ICloudSecurityGroup, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	// google
	if len(self.GlobalvpcId) > 0 {
		vpc, err := self.GetGlobalVpc()
		if err != nil {
			return nil, errors.Wrapf(err, "GetGlobalVpc")
		}
		iVpc, err := vpc.GetICloudGlobalVpc(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetICloudGlobalVpc")
		}
		securityGroups, err := iVpc.GetISecurityGroups()
		if err != nil {
			return nil, errors.Wrapf(err, "GetISecurityGroups")
		}
		for i := range securityGroups {
			if securityGroups[i].GetGlobalId() == self.ExternalId {
				return securityGroups[i], nil
			}
		}
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, self.ExternalId)
	}
	iRegion, err := self.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIRegion")
	}
	return iRegion.GetISecurityGroupById(self.ExternalId)
}

func (self *SSecurityGroup) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	region, err := self.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	provider, err := self.GetProvider(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetProvider")
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (self *SSecurityGroup) GetCloudprovider() (*SCloudprovider, error) {
	providerObj, err := CloudproviderManager.FetchById(self.ManagerId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById")
	}
	return providerObj.(*SCloudprovider), nil
}

func (self *SSecurityGroup) GetProvider(ctx context.Context) (cloudprovider.ICloudProvider, error) {
	manager, err := self.GetCloudprovider()
	if err != nil {
		return nil, errors.Wrapf(err, "GetProvider")
	}
	return manager.GetProvider(ctx)
}

func (sm *SSecurityGroupManager) query(manager db.IModelManager, field, label string, secIds []string) *sqlchemy.SSubQuery {
	sq := manager.Query().SubQuery()

	return sq.Query(
		sq.Field(field),
		sqlchemy.COUNT(label),
	).In(field, secIds).GroupBy(sq.Field(field)).SubQuery()
}

type sSecuriyGroupCnts struct {
	Id        string
	Guest1Cnt int
	api.SSecurityGroupRef
}

func (sm *SSecurityGroupManager) TotalCnt(secIds []string) (map[string]api.SSecurityGroupRef, error) {
	g1SQ := sm.query(GuestsecgroupManager, "secgroup_id", "guest1", secIds)
	g2SQ := sm.query(GuestManager, "secgrp_id", "guest2", secIds)
	g3SQ := sm.query(GuestManager, "admin_secgrp_id", "guest3", secIds)

	rdsSQ := sm.query(DBInstanceSecgroupManager, "secgroup_id", "rds", secIds)
	redisSQ := sm.query(ElasticcachesecgroupManager, "secgroup_id", "redis", secIds)

	secs := sm.Query().SubQuery()
	secQ := secs.Query(
		sqlchemy.SUM("guest_cnt", g1SQ.Field("guest1")),
		sqlchemy.SUM("guest1_cnt", g2SQ.Field("guest2")),
		sqlchemy.SUM("admin_guest_cnt", g3SQ.Field("guest3")),
		sqlchemy.SUM("rds_cnt", rdsSQ.Field("rds")),
		sqlchemy.SUM("redis_cnt", redisSQ.Field("redis")),
	)

	secQ.AppendField(secQ.Field("id"))

	secQ = secQ.LeftJoin(g1SQ, sqlchemy.Equals(secQ.Field("id"), g1SQ.Field("secgroup_id")))
	secQ = secQ.LeftJoin(g2SQ, sqlchemy.Equals(secQ.Field("id"), g2SQ.Field("secgrp_id")))
	secQ = secQ.LeftJoin(g3SQ, sqlchemy.Equals(secQ.Field("id"), g3SQ.Field("admin_secgrp_id")))
	secQ = secQ.LeftJoin(rdsSQ, sqlchemy.Equals(secQ.Field("id"), rdsSQ.Field("secgroup_id")))
	secQ = secQ.LeftJoin(redisSQ, sqlchemy.Equals(secQ.Field("id"), redisSQ.Field("secgroup_id")))

	secQ = secQ.Filter(sqlchemy.In(secQ.Field("id"), secIds)).GroupBy(secQ.Field("id"))

	cnts := []sSecuriyGroupCnts{}
	err := secQ.All(&cnts)
	if err != nil {
		return nil, errors.Wrapf(err, "secQ.All")
	}
	result := map[string]api.SSecurityGroupRef{}
	for i := range cnts {
		cnts[i].GuestCnt += cnts[i].Guest1Cnt
		cnts[i].Sum()
		result[cnts[i].Id] = cnts[i].SSecurityGroupRef
	}
	return result, nil
}

func (self *SSecurityGroup) ValidateDeleteCondition(ctx context.Context, info api.SecgroupDetails) error {
	if self.Id == options.Options.DefaultSecurityGroupId {
		return httperrors.NewProtectedResourceError("not allow to delete default security group")
	}
	if self.Id == options.Options.DefaultAdminSecurityGroupId {
		return httperrors.NewProtectedResourceError("not allow to delete default admin security group")
	}
	if info.TotalCnt > 0 {
		return httperrors.NewNotEmptyError("the security group %s is in use cnt: %d", self.Id, info.TotalCnt)
	}
	return self.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SSecurityGroup) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteSecurityGroupTask(ctx, userCred, "")
}

func (self *SSecurityGroup) StartDeleteSecurityGroupTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	self.SetStatus(userCred, apis.STATUS_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "SecurityGroupDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SSecurityGroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("do nothing for delete secgroup")
	return nil
}

func (self *SSecurityGroup) OnMetadataUpdated(ctx context.Context, userCred mcclient.TokenCredential) {
	if len(self.ExternalId) == 0 || options.Options.KeepTagLocalization {
		return
	}
	if account := self.GetCloudaccount(); account != nil && account.ReadOnly {
		return
	}
	err := self.StartRemoteUpdateTask(ctx, userCred, true, "")
	if err != nil {
		log.Errorf("StartRemoteUpdateTask fail: %s", err)
	}
}

func (self *SSecurityGroup) StartRemoteUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, replaceTags bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Set("replace_tags", jsonutils.NewBool(replaceTags))
	task, err := taskman.TaskManager.NewTask(ctx, "SecurityGroupRemoteUpdateTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "RemoteUpdateTask")
	}
	self.SetStatus(userCred, apis.STATUS_UPDATE_TAGS, "StartRemoteUpdateTask")
	return task.ScheduleRun(nil)
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
		err := rules[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "rules[i].Delete")
		}
	}
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

func (sg *SSecurityGroup) GetQuotaKeys() quotas.IQuotaKeys {
	return quotas.OwnerIdProjectQuotaKeys(rbacscope.ScopeProject, sg.GetOwnerId())
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
			Priority:    int(*r.Priority),
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

func (manager *SSecurityGroupManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SGlobalVpcResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SGlobalVpcResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SGlobalVpcResourceBaseManager.ListItemExportKeys")
		}
	}

	if keys.ContainsAny(manager.SVpcResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SVpcResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (self *SCloudregion) GetSecgroups(vpcId string) ([]SSecurityGroup, error) {
	q := SecurityGroupManager.Query().Equals("cloudregion_id", self.Id)
	if len(vpcId) > 0 {
		q = q.Equals("vpc_id", vpcId)
	}
	ret := []SSecurityGroup{}
	return ret, db.FetchModelObjects(SecurityGroupManager, q, &ret)
}

func (self *SCloudregion) SyncSecgroups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, vpc *SVpc, exts []cloudprovider.ICloudSecurityGroup, xor bool) compare.SyncResult {
	vpcId := ""
	if !gotypes.IsNil(vpc) {
		vpcId = vpc.Id
	}
	key := fmt.Sprintf("%s-%s", self.Id, vpcId)

	lockman.LockRawObject(ctx, SecurityGroupManager.Keyword(), key)
	defer lockman.ReleaseRawObject(ctx, SecurityGroupManager.Keyword(), key)

	result := compare.SyncResult{}

	dbSecs, err := self.GetSecgroups(vpcId)
	if err != nil {
		result.Error(err)
		return result
	}

	syncOwnerId := provider.GetOwnerId()

	removed := make([]SSecurityGroup, 0)
	commondb := make([]SSecurityGroup, 0)
	commonext := make([]cloudprovider.ICloudSecurityGroup, 0)
	added := make([]cloudprovider.ICloudSecurityGroup, 0)

	err = compare.CompareSets(dbSecs, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].RealDelete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(commondb); i += 1 {
		if !xor {
			err = commondb[i].SyncWithCloudSecurityGroup(ctx, userCred, commonext[i], syncOwnerId, true)
			if err != nil {
				result.UpdateError(err)
				continue
			}
		}
		result.Update()
	}

	for i := 0; i < len(added); i += 1 {
		err := self.newFromCloudSecurityGroup(ctx, userCred, provider, vpc, added[i], syncOwnerId)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SSecurityGroup) SyncWithCloudSecurityGroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ext cloudprovider.ICloudSecurityGroup,
	syncOwnerId mcclient.IIdentityProvider,
	syncRule bool,
) error {
	_, err := db.Update(self, func() error {
		self.Name = ext.GetName()
		if len(self.Description) == 0 {
			self.Description = ext.GetDescription()
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}

	if account := self.GetCloudaccount(); account != nil {
		syncVirtualResourceMetadata(ctx, userCred, self, ext, account.ReadOnly)
	}

	if provider, _ := self.GetCloudprovider(); provider != nil {
		SyncCloudProject(ctx, userCred, self, syncOwnerId, ext, provider)
	}

	if !syncRule {
		return nil
	}

	rules, err := ext.GetRules()
	if err != nil {
		return errors.Wrapf(err, "GetRules")
	}
	result := self.SyncRules(ctx, userCred, rules)
	if result.IsError() {
		logclient.AddSimpleActionLog(self, logclient.ACT_CLOUD_SYNC, result, userCred, false)
	}
	return nil
}

func (self *SSecurityGroup) GetSecurityGroups(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	filter func(q *sqlchemy.SQuery) *sqlchemy.SQuery,
) ([]SSecurityGroup, error) {
	query := SecurityGroupManager.Query().Equals("status", api.SECGROUP_STATUS_READY).IsNotEmpty("external_id")
	query = filter(query)
	query = query.Filter(
		sqlchemy.OR(
			sqlchemy.AND(
				sqlchemy.Equals(query.Field("public_scope"), "system"),
				sqlchemy.Equals(query.Field("is_public"), true),
			),
			sqlchemy.AND(
				sqlchemy.Equals(query.Field("tenant_id"), ownerId.GetProjectId()),
				sqlchemy.Equals(query.Field("domain_id"), ownerId.GetDomainId()),
			),
		),
	)
	ret := []SSecurityGroup{}
	return ret, db.FetchModelObjects(SecurityGroupManager, query, &ret)
}

func (self *SCloudregion) newFromCloudSecurityGroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	vpc *SVpc,
	ext cloudprovider.ICloudSecurityGroup,
	syncOwnerId mcclient.IIdentityProvider,
) error {
	ret := &SSecurityGroup{}
	ret.SetModelManager(SecurityGroupManager, ret)
	ret.Name = ext.GetName()
	ret.CloudregionId = self.Id
	if vpc != nil {
		ret.VpcId = vpc.Id
	}
	ret.Description = ext.GetDescription()
	ret.ExternalId = ext.GetGlobalId()
	ret.ManagerId = provider.Id
	ret.Status = api.SECGROUP_STATUS_READY
	err := SecurityGroupManager.TableSpec().Insert(ctx, ret)
	if err != nil {
		return errors.Wrapf(err, "Insert")
	}

	syncVirtualResourceMetadata(ctx, userCred, ret, ext, false)
	SyncCloudProject(ctx, userCred, ret, syncOwnerId, ext, provider)

	rules, err := ext.GetRules()
	if err != nil {
		return errors.Wrapf(err, "GetRules")
	}
	result := ret.SyncRules(ctx, userCred, rules)
	if result.IsError() {
		logclient.AddSimpleActionLog(ret, logclient.ACT_CLOUD_SYNC, result, userCred, false)
	}
	return nil
}
