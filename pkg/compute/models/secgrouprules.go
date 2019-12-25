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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SSecurityGroupRuleManager struct {
	db.SResourceBaseManager
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
	Id          string `width:"128" charset:"ascii" primary:"true" list:"user"`
	Priority    int64  `default:"1" list:"user" update:"user" list:"user"`
	Protocol    string `width:"5" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`
	Ports       string `width:"256" charset:"ascii" list:"user" update:"user" create:"optional"`
	Direction   string `width:"3" charset:"ascii" list:"user" create:"required"`
	CIDR        string `width:"256" charset:"ascii" list:"user" update:"user" create:"required"`
	Action      string `width:"5" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`
	Description string `width:"256" charset:"utf8" list:"user" update:"user" create:"optional"`
	SecgroupID  string `width:"128" charset:"ascii" create:"required"`
}

type SecurityGroupRuleSet []SSecurityGroupRule

func (v SecurityGroupRuleSet) Len() int {
	return len(v)
}

func (v SecurityGroupRuleSet) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v SecurityGroupRuleSet) Less(i, j int) bool {
	if v[i].Priority < v[j].Priority {
		return true
	} else if v[i].Priority == v[j].Priority {
		return strings.Compare(v[i].String(), v[j].String()) <= 0
	}
	return false
}

func (manager *SSecurityGroupRuleManager) FetchParentId(ctx context.Context, data jsonutils.JSONObject) string {
	parentId, _ := data.GetString("secgroup_id")
	return parentId
}

func (manager *SSecurityGroupRuleManager) FilterByParentId(q *sqlchemy.SQuery, parentId string) *sqlchemy.SQuery {
	if len(parentId) > 0 {
		q = q.Equals("secgroup_id", parentId)
	}
	return q
}

func (manager *SSecurityGroupRuleManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	parentId := manager.FetchParentId(ctx, data)
	if len(parentId) > 0 {
		secgroup, err := db.FetchById(SecurityGroupManager, parentId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchById(SecurityGroupManager, %s)", parentId)
		}
		return secgroup.(*SSecurityGroup).GetOwnerId(), nil
	}
	return nil, nil
}

func (manager *SSecurityGroupRuleManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if userCred != nil {
		sq := SecurityGroupManager.Query("id")
		ssq := SecurityGroupManager.FilterByOwner(sq, userCred, scope)
		return q.In("secgroup_id", ssq.SubQuery())
	}
	return q
}

func (manager *SSecurityGroupRuleManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (manager *SSecurityGroupRuleManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SSecurityGroupRule) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	if secgroup := self.GetSecGroup(); secgroup != nil {
		return secgroup.IsOwner(userCred) || db.IsAdminAllowUpdate(userCred, self)
	}
	return false
}

func (self *SSecurityGroupRule) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	if secgroup := self.GetSecGroup(); secgroup != nil {
		return secgroup.IsOwner(userCred) || db.IsAdminAllowDelete(userCred, self)
	}
	return false
}

func (self *SSecurityGroupRule) GetSecGroup() *SSecurityGroup {
	if secgroup, _ := SecurityGroupManager.FetchById(self.SecgroupID); secgroup != nil {
		return secgroup.(*SSecurityGroup)
	}
	return nil
}

func (manager *SSecurityGroupRuleManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.Equals("id", idStr)
}

func (manager *SSecurityGroupRuleManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (sql *sqlchemy.SQuery, err error) {
	if sql, err = manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query); err != nil {
		return nil, err
	}
	if defsecgroup, _ := query.GetString("secgroup"); len(defsecgroup) > 0 {
		if secgroup, _ := SecurityGroupManager.FetchByIdOrName(userCred, defsecgroup); secgroup != nil {
			sql = sql.Equals("secgroup_id", secgroup.GetId())
		} else {
			return nil, httperrors.NewNotFoundError("Security Group %s not found", defsecgroup)
		}
	}
	for _, field := range []string{"direction", "action", "protocol"} {
		if key, _ := query.GetString(field); len(key) > 0 {
			sql = sql.Equals(field, key)
		}
	}
	return sql, err
}

func (self *SSecurityGroupRule) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SSecurityGroupRule) BeforeInsert() {
	if len(self.Id) == 0 {
		self.Id = stringutils.UUID4()
	}
}

func (manager *SSecurityGroupRuleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	priorityV := validators.NewRangeValidator("priority", 1, 100)
	priorityV.Optional(true)
	err := priorityV.Validate(data)
	if err != nil {
		return nil, err
	}

	secgroupV := validators.NewModelIdOrNameValidator("secgroup", "secgroup", ownerId)
	err = secgroupV.Validate(data)
	if err != nil {
		return nil, err
	}

	secgroup := secgroupV.Model.(*SSecurityGroup)

	if !secgroup.IsOwner(userCred) && !userCred.HasSystemAdminPrivilege() {
		return nil, httperrors.NewForbiddenError("not enough privilege")
	}

	input := &api.SSecgroupRuleCreateInput{}
	err = data.Unmarshal(input)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Failed to unmarshal input: %v", err)
	}

	err = input.Check()
	if err != nil {
		return nil, err
	}

	data, err = manager.SResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
	if err != nil {
		return nil, err
	}
	return input.JSON(input), nil
}

func (self *SSecurityGroupRule) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	priorityV := validators.NewRangeValidator("priority", 1, 100)
	priorityV.Optional(true)
	err := priorityV.Validate(data)
	if err != nil {
		return nil, err
	}

	input := &api.SSecgroupRuleCreateInput{
		Direction: self.Direction,
		Action:    self.Action,
		CIDR:      self.CIDR,
		Protocol:  self.Protocol,
		Ports:     self.Ports,
	}

	err = jsonutils.Update(input, data)
	if err != nil {
		return nil, err
	}

	err = input.Check()
	if err != nil {
		return nil, err
	}

	return self.SResourceBase.ValidateUpdateData(ctx, userCred, query, data)
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
		secgroup.DoSync(ctx, userCred)
	}
}

func (self *SSecurityGroupRule) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	self.SResourceBase.PreDelete(ctx, userCred)

	if secgroup := self.GetSecGroup(); secgroup != nil {
		secgroup.DoSync(ctx, userCred)
	}
}

func (self *SSecurityGroupRule) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SResourceBase.PostUpdate(ctx, userCred, query, data)

	log.Debugf("POST Update %s", data)
	if secgroup := self.GetSecGroup(); secgroup != nil {
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

func (manager *SSecurityGroupRuleManager) SyncRules(ctx context.Context, userCred mcclient.TokenCredential, secgroup *SSecurityGroup, rules []secrules.SecurityRule) compare.SyncResult {
	syncResult := compare.SyncResult{}
	for i := 0; i < len(rules); i++ {
		_, err := manager.newFromCloudSecurityGroup(ctx, userCred, rules[i], secgroup)
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		syncResult.Add()
	}
	return syncResult
}

func (manager *SSecurityGroupRuleManager) newFromCloudSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, rule secrules.SecurityRule, secgroup *SSecurityGroup) (*SSecurityGroupRule, error) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	protocol := rule.Protocol
	if len(protocol) == 0 {
		protocol = secrules.PROTO_ANY
	}

	cidr := "0.0.0.0/0"
	if rule.IPNet != nil && rule.IPNet.String() != "<nil>" {
		cidr = rule.IPNet.String()
	}

	secrule := &SSecurityGroupRule{
		Priority:    int64(rule.Priority),
		Protocol:    protocol,
		Ports:       rule.GetPortsString(),
		Direction:   string(rule.Direction),
		CIDR:        cidr,
		Action:      string(rule.Action),
		Description: rule.Description,
		SecgroupID:  secgroup.Id,
	}

	err := manager.TableSpec().Insert(secrule)
	if err != nil {
		return nil, err
	}
	return secrule, nil
}

func (self *SSecurityGroupRule) GetOwnerId() mcclient.IIdentityProvider {
	secgrp := self.GetSecGroup()
	if secgrp != nil {
		return secgrp.GetOwnerId()
	}
	return nil
}

func (manager *SSecurityGroupRuleManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeProject
}
