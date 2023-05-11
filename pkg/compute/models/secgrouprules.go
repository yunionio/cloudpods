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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSecurityGroupRuleManager struct {
	db.SResourceBaseManager
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
	SSecurityGroupResourceBase `create:"required"`

	Id          string `width:"128" charset:"ascii" primary:"true" list:"user"`
	Priority    int64  `default:"1" list:"user" update:"user" list:"user"`
	Protocol    string `width:"5" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`
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
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SSecurityGroupRule) BeforeInsert() {
	if len(self.Id) == 0 {
		self.Id = stringutils.UUID4()
	}
}

func (manager *SSecurityGroupRuleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.SSecgroupRuleCreateInput) (api.SSecgroupRuleCreateInput, error) {
	if input.Priority == nil {
		return input, httperrors.NewMissingParameterError("priority")
	}
	if *input.Priority < 1 || *input.Priority > 100 {
		return input, httperrors.NewOutOfRangeError("Invalid priority %d, must be in range or 1 ~ 100", input.Priority)
	}

	_secgroup, err := validators.ValidateModel(userCred, SecurityGroupManager, &input.SecgroupId)
	if err != nil {
		return input, err
	}

	secgroup := _secgroup.(*SSecurityGroup)

	if !secgroup.IsOwner(userCred) && !userCred.HasSystemAdminPrivilege() {
		return input, httperrors.NewForbiddenError("not enough privilege")
	}

	err = input.Check()
	if err != nil {
		return input, err
	}

	input.ResourceBaseCreateInput, err = manager.SResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.ResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SSecurityGroupRule) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SSecgroupRuleUpdateInput) (api.SSecgroupRuleUpdateInput, error) {
	priority := int(self.Priority)
	if input.Priority == nil {
		input.Priority = &priority
	}
	if len(input.Direction) == 0 {
		input.Direction = self.Direction
	}
	if len(input.Action) == 0 {
		input.Action = self.Action
	}
	if len(input.Protocol) == 0 {
		input.Protocol = self.Protocol
	}
	if len(input.Ports) == 0 && input.Protocol != string(secrules.PROTO_ANY) && input.Protocol != string(secrules.PROTO_ICMP) {
		input.Ports = self.Ports
	}

	if *input.Priority < 1 || *input.Priority > 100 {
		return input, httperrors.NewOutOfRangeError("Invalid priority %d, must be in range or 1 ~ 100", input.Priority)
	}

	if len(input.CIDR) == 0 {
		input.CIDR = self.CIDR
	}

	err := input.Check()
	if err != nil {
		return input, err
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
	if secgroup := self.GetSecGroup(); secgroup != nil {
		logclient.AddSimpleActionLog(secgroup, logclient.ACT_ALLOCATE, data, userCred, true)
		secgroup.DoSync(ctx, userCred)
	}
}

func (self *SSecurityGroupRule) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	self.SResourceBase.PreDelete(ctx, userCred)

	if secgroup := self.GetSecGroup(); secgroup != nil {
		logclient.AddSimpleActionLog(secgroup, logclient.ACT_DELETE, jsonutils.Marshal(self), userCred, true)
		secgroup.DoSync(ctx, userCred)
	}
}

func (self *SSecurityGroupRule) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SResourceBase.PostUpdate(ctx, userCred, query, data)

	log.Debugf("POST Update %s", data)
	if secgroup := self.GetSecGroup(); secgroup != nil {
		logclient.AddSimpleActionLog(secgroup, logclient.ACT_UPDATE, data, userCred, true)
		secgroup.DoSync(ctx, userCred)
	}
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
	secgrp := self.GetSecGroup()
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
