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
	"net"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSecurityGroupRuleManager struct {
	db.SResourceBaseManager
	db.SStatusResourceBaseManager
	db.SExternalizedResourceBaseManager
	SSecurityGroupResourceBaseManager
}

var SecurityGroupRuleManager *SSecurityGroupRuleManager

func init() {
	SecurityGroupRuleManager = &SSecurityGroupRuleManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SSecurityGroupRule{},
			"secgrouprules_tbl",
			"secgrouprule",
			"secgrouprules",
		),
	}
	SecurityGroupRuleManager.SetVirtualObject(SecurityGroupRuleManager)
}

type SSecurityGroupRule struct {
	db.SResourceBase
	db.SStatusResourceBase `default:"available"`
	db.SExternalizedResourceBase
	SSecurityGroupResourceBase `create:"required"`

	Id          string `width:"128" charset:"ascii" primary:"true" list:"user"`
	Priority    int    `list:"user" update:"user" list:"user"`
	Protocol    string `width:"32" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`
	Ports       string `width:"256" charset:"ascii" list:"user" update:"user" create:"optional"`
	Direction   string `width:"3" charset:"ascii" list:"user" create:"required"`
	CIDR        string `width:"256" charset:"ascii" list:"user" update:"user" create:"optional"`
	Action      string `width:"5" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`
	Description string `width:"256" charset:"utf8" list:"user" update:"user" create:"optional"`
}

func (self *SSecurityGroupRule) GetId() string {
	return self.Id
}

func (manager *SSecurityGroupRuleManager) CreateByInsertOrUpdate() bool {
	return false
}

func (manager *SSecurityGroupRuleManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	secgroupId, _ := data.GetString("secgroup_id")
	return jsonutils.Marshal(map[string]string{"secgroup_id": secgroupId})
}

func (manager *SSecurityGroupRuleManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	secgroupId, _ := values.GetString("secgroup_id")
	if len(secgroupId) > 0 {
		q = q.Equals("secgroup_id", secgroupId)
	}
	return q
}

func (manager *SSecurityGroupRuleManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	secgroupId, _ := data.GetString("secgroup_id")
	if len(secgroupId) > 0 {
		secgroup, err := db.FetchById(SecurityGroupManager, secgroupId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchById(SecurityGroupManager, %s)", secgroupId)
		}
		return secgroup.(*SSecurityGroup).GetOwnerId(), nil
	}
	return db.FetchProjectInfo(ctx, data)
}

func (manager *SSecurityGroupRuleManager) FilterByOwner(q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	sq := SecurityGroupManager.Query("id")
	sq = db.SharableManagerFilterByOwner(SecurityGroupManager, sq, userCred, ownerId, scope)
	return q.In("secgroup_id", sq.SubQuery())
}

func (manager *SSecurityGroupRuleManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.Equals("id", idStr)
}

// 安全组规则列表
func (manager *SSecurityGroupRuleManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SecurityGroupRuleListInput,
) (*sqlchemy.SQuery, error) {
	sql, err := manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	sql, err = manager.SSecurityGroupResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SecgroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.ListItemFilter")
	}
	if len(query.Projects) > 0 {
		sq := SecurityGroupManager.Query("id")
		tenants := db.TenantCacheManager.GetTenantQuery().SubQuery()
		subq := tenants.Query(tenants.Field("id")).Filter(sqlchemy.OR(
			sqlchemy.In(tenants.Field("id"), query.Projects),
			sqlchemy.In(tenants.Field("name"), query.Projects),
		)).SubQuery()
		sq = sq.In("tenant_id", subq)
		sql = sql.In("secgroup_id", sq.SubQuery())
	}
	if len(query.Direction) > 0 {
		sql = sql.Equals("direction", query.Direction)
	}
	if len(query.Action) > 0 {
		sql = sql.Equals("action", query.Action)
	}
	if len(query.Protocol) > 0 {
		sql = sql.Equals("protocol", query.Protocol)
	}
	if len(query.Ports) > 0 {
		sql = sql.Equals("ports", query.Ports)
	}
	if len(query.Ip) > 0 {
		sql = sql.Like("cidr", "%"+query.Ip+"%")
	}

	return sql, nil
}

func (manager *SSecurityGroupRuleManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.SecgroupRuleDetails {
	rows := make([]api.SecgroupRuleDetails, len(objs))
	bRows := manager.SResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	secRows := manager.SSecurityGroupResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	secIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.SecgroupRuleDetails{
			ResourceBaseDetails:       bRows[i],
			SecurityGroupResourceInfo: secRows[i],
		}
		rule := objs[i].(*SSecurityGroupRule)
		secIds[i] = rule.SecgroupId
	}

	secgroups := make(map[string]SSecurityGroup)
	err := db.FetchStandaloneObjectsByIds(SecurityGroupManager, secIds, &secgroups)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail: %v", err)
		return rows
	}

	virObjs := make([]interface{}, len(objs))
	for i := range rows {
		if secgroup, ok := secgroups[secIds[i]]; ok {
			virObjs[i] = &secgroup
			rows[i].ProjectId = secgroup.ProjectId
		}
	}

	projRows := SecurityGroupManager.SProjectizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, virObjs, fields, isList)
	for i := range rows {
		rows[i].ProjectizedResourceInfo = projRows[i]
	}

	return rows
}

func (manager *SSecurityGroupRuleManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SecurityGroupRuleListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SSecurityGroupResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SecgroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SSecurityGroupRuleManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SSecurityGroupResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SSecurityGroupRule) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SSecurityGroupRule) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SSecurityGroupRule) BeforeInsert() {
	if len(self.Id) == 0 {
		self.Id = stringutils.UUID4()
	}
}

func (manager *SSecurityGroupRuleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.SSecgroupRuleCreateInput) (*api.SSecgroupRuleCreateInput, error) {
	_secgroup, err := validators.ValidateModel(userCred, SecurityGroupManager, &input.SecgroupId)
	if err != nil {
		return input, err
	}

	input.Status = apis.STATUS_CREATING

	secgroup := _secgroup.(*SSecurityGroup)

	driver, err := secgroup.GetRegionDriver()
	if err != nil {
		return nil, err
	}

	opts := &api.SSecgroupCreateInput{}
	opts.Rules = []api.SSecgroupRuleCreateInput{*input}
	_, err = driver.ValidateCreateSecurityGroupInput(ctx, userCred, opts)
	if err != nil {
		return nil, err
	}

	if !secgroup.IsOwner(userCred) && !userCred.HasSystemAdminPrivilege() {
		return input, httperrors.NewForbiddenError("not enough privilege")
	}

	input.ResourceBaseCreateInput, err = manager.SResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.ResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SSecurityGroupRule) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.SSecgroupRuleUpdateInput) (*api.SSecgroupRuleUpdateInput, error) {
	secgrp, err := self.GetSecGroup()
	if err != nil {
		return nil, err
	}

	if input.CIDR == nil {
		input.CIDR = &self.CIDR
	}

	driver, err := secgrp.GetRegionDriver()
	if err != nil {
		return nil, err
	}

	input, err = driver.ValidateUpdateSecurityGroupRuleInput(ctx, userCred, input)
	if err != nil {
		return nil, err
	}

	input.ResourceBaseUpdateInput, err = self.SResourceBase.ValidateUpdateData(ctx, userCred, query, input.ResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SResourceBase.ValidateUpdateData")
	}

	return input, nil
}

func (self *SSecurityGroupRule) String() string {
	rule, err := self.toRule()
	if err != nil {
		return ""
	}
	return rule.String()
}

func (self *SSecurityGroupRule) toRule() (*secrules.SecurityRule, error) {
	rule := secrules.SecurityRule{
		Priority:    int(self.Priority),
		Direction:   secrules.TSecurityRuleDirection(self.Direction),
		Action:      secrules.TSecurityRuleAction(self.Action),
		Protocol:    self.Protocol,
		Description: self.Description,
	}
	if regutils.MatchCIDR(self.CIDR) {
		_, rule.IPNet, _ = net.ParseCIDR(self.CIDR)
	} else if regutils.MatchIPAddr(self.CIDR) {
		rule.IPNet = &net.IPNet{
			IP:   net.ParseIP(self.CIDR),
			Mask: net.CIDRMask(32, 32),
		}
	} else {
		rule.IPNet = &net.IPNet{
			IP:   net.IPv4zero,
			Mask: net.CIDRMask(0, 32),
		}
	}

	err := rule.ParsePorts(self.Ports)
	if err != nil {
		return nil, err
	}

	return &rule, rule.ValidateRule()
}

func (self *SSecurityGroupRule) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	log.Debugf("POST Create %s", data)
	if secgroup, _ := self.GetSecGroup(); secgroup != nil {
		logclient.AddSimpleActionLog(secgroup, logclient.ACT_ALLOCATE, data, userCred, true)
		if len(secgroup.ManagerId) == 0 {
			secgroup.DoSync(ctx, userCred)
			return
		}
		secgroup.StartSecurityGroupRuleCreateTask(ctx, userCred, self.Id, "")
	}
}

func (self *SSecurityGroupRule) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	self.SResourceBase.PreDelete(ctx, userCred)

	if secgroup, _ := self.GetSecGroup(); secgroup != nil {
		logclient.AddSimpleActionLog(secgroup, logclient.ACT_DELETE, jsonutils.Marshal(self), userCred, true)
		if len(secgroup.ManagerId) == 0 {
			self.RealDelete(ctx, userCred)
			secgroup.DoSync(ctx, userCred)
			return
		}
		self.SetStatus(userCred, apis.STATUS_DELETING, "")
		secgroup.StartSecurityGroupRuleDeleteTask(ctx, userCred, self.Id, "")
	}
}

func (self *SSecurityGroupRule) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SResourceBase.PostUpdate(ctx, userCred, query, data)

	if self.Protocol == secrules.PROTO_ICMP || self.Protocol == secrules.PROTO_ANY {
		db.Update(self, func() error {
			self.Ports = ""
			return nil
		})
	}

	log.Debugf("POST Update %s", data)
	if secgroup, _ := self.GetSecGroup(); secgroup != nil {
		logclient.AddSimpleActionLog(secgroup, logclient.ACT_UPDATE, data, userCred, true)
		if len(secgroup.ManagerId) == 0 {
			secgroup.DoSync(ctx, userCred)
			return
		}
		self.SetStatus(userCred, apis.STATUS_SYNC_STATUS, "")
		secgroup.StartSecurityGroupRuleUpdateTask(ctx, userCred, self.Id, "")
	}
}

func (self *SSecurityGroup) StartSecurityGroupRuleUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, ruleId, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Set("rule_id", jsonutils.NewString(ruleId))
	task, err := taskman.TaskManager.NewTask(ctx, "SecurityGroupRuleUpdateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (manager *SSecurityGroupRuleManager) getRulesBySecurityGroup(secgroup *SSecurityGroup) ([]SSecurityGroupRule, error) {
	rules := make([]SSecurityGroupRule, 0)
	q := manager.Query().Equals("secgroup_id", secgroup.Id)
	if err := db.FetchModelObjects(manager, q, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (self *SSecurityGroupRule) GetOwnerId() mcclient.IIdentityProvider {
	secgrp, _ := self.GetSecGroup()
	if secgrp != nil {
		return secgrp.GetOwnerId()
	}
	return nil
}

func (manager *SSecurityGroupRuleManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (manager *SSecurityGroupRuleManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SSecurityGroupResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (self *SSecurityGroup) SyncRules(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	exts []cloudprovider.ISecurityGroupRule,
) compare.SyncResult {
	lockman.LockRawObject(ctx, SecurityGroupManager.Keyword(), self.Id)
	defer lockman.ReleaseRawObject(ctx, SecurityGroupManager.Keyword(), self.Id)

	result := compare.SyncResult{}

	dbRules, err := self.GetSecurityRules()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SSecurityGroupRule, 0)
	commondb := make([]SSecurityGroupRule, 0)
	commonext := make([]cloudprovider.ISecurityGroupRule, 0)
	added := make([]cloudprovider.ISecurityGroupRule, 0)

	err = compare.CompareSets(dbRules, exts, &removed, &commondb, &commonext, &added)
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
		err = commondb[i].syncWithCloudRule(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	for i := 0; i < len(added); i += 1 {
		err := self.newFromCloudRule(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (rule *SSecurityGroupRule) syncWithCloudRule(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ISecurityGroupRule) error {
	_, err := db.Update(rule, func() error {
		rule.Action = string(ext.GetAction())
		rule.Direction = string(ext.GetDirection())
		rule.Protocol = string(ext.GetProtocol())
		rule.Description = string(ext.GetDescription())
		rule.CIDR = strings.Join(ext.GetCIDRs(), ",")
		rule.Priority = ext.GetPriority()
		rule.Ports = ext.GetPorts()
		rule.Status = apis.STATUS_AVAILABLE
		return nil
	})
	return err
}

func (self *SSecurityGroup) newFromCloudRule(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ISecurityGroupRule) error {
	rule := &SSecurityGroupRule{}
	rule.SetModelManager(SecurityGroupRuleManager, rule)
	rule.SecgroupId = self.Id
	rule.Action = string(ext.GetAction())
	rule.Direction = string(ext.GetDirection())
	rule.Protocol = string(ext.GetProtocol())
	rule.Description = string(ext.GetDescription())
	rule.CIDR = strings.Join(ext.GetCIDRs(), ",")
	rule.Priority = ext.GetPriority()
	rule.Ports = ext.GetPorts()
	rule.ExternalId = ext.GetGlobalId()
	rule.Status = apis.STATUS_AVAILABLE
	return SecurityGroupRuleManager.TableSpec().Insert(ctx, rule)
}

func (self *SSecurityGroupRule) SetStatus(userCred mcclient.TokenCredential, status, reason string) error {
	if self.Status == status {
		return nil
	}
	_, err := db.Update(self, func() error {
		self.Status = status
		return nil
	})
	return err
}

func (manager *SSecurityGroupRuleManager) FetchRuleById(id string) (*SSecurityGroupRule, error) {
	rule, err := db.FetchById(manager, id)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById(%s)", id)
	}
	return rule.(*SSecurityGroupRule), nil
}
