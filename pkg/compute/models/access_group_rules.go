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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SAccessGroupRuleManager struct {
	db.SResourceBaseManager
	SAccessGroupResourceBaseManager
}

var AccessGroupRuleManager *SAccessGroupRuleManager

func init() {
	AccessGroupRuleManager = &SAccessGroupRuleManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SAccessGroupRule{},
			"access_group_rules_tbl",
			"access_group_rule",
			"access_group_rules",
		),
	}
	AccessGroupRuleManager.SetVirtualObject(AccessGroupRuleManager)
}

type SAccessGroupRule struct {
	db.SResourceBase
	SAccessGroupResourceBase

	Id             string `width:"128" charset:"ascii" primary:"true" list:"user"`
	Priority       int    `default:"1" list:"user" update:"user" list:"user"`
	Source         string `width:"16" charset:"ascii" list:"user" update:"user" create:"required"`
	RWAccessType   string `width:"16" charset:"ascii" list:"user" update:"user" create:"required"`
	UserAccessType string `width:"16" charset:"ascii" list:"user" update:"user" create:"required"`
	Description    string `width:"256" charset:"utf8" list:"user" update:"user" create:"optional"`
}

func (self *SAccessGroupRule) BeforeInsert() {
	if len(self.Id) == 0 {
		self.Id = stringutils.UUID4()
	}
}

func (self *SAccessGroupRule) GetId() string {
	return self.Id
}

func (manager *SAccessGroupRuleManager) CreateByInsertOrUpdate() bool {
	return false
}

func (manager *SAccessGroupRuleManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (manager *SAccessGroupRuleManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	groupId, _ := data.GetString("access_group_id")
	return jsonutils.Marshal(map[string]string{"access_group_id": groupId})
}

func (manager *SAccessGroupRuleManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	groupId, _ := values.GetString("access_group_id")
	if len(groupId) > 0 {
		q = q.Equals("access_group_id", groupId)
	}
	return q
}

func (manager *SAccessGroupRuleManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	groupId, _ := data.GetString("access_group_id")
	if len(groupId) > 0 {
		accessGroup, err := db.FetchById(AccessGroupManager, groupId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchById(%s)", groupId)
		}
		return accessGroup.(*SAccessGroup).GetOwnerId(), nil
	}
	return db.FetchDomainInfo(ctx, data)
}

func (manager *SAccessGroupRuleManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	sq := AccessGroupManager.Query("id")
	sq = db.SharableManagerFilterByOwner(AccessGroupManager, sq, userCred, scope)
	return q.In("access_group_id", sq.SubQuery())
}

func (manager *SAccessGroupRuleManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.Equals("id", idStr)
}

// 权限组规则列表
func (manager *SAccessGroupRuleManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AccessGroupRuleListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SAccessGroupResourceBaseManager.ListItemFilter(ctx, q, userCred, query.AccessGroupFilterListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SAccessGroupResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SAccessGroupRuleManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.AccessGroupRuleDetails {
	rows := make([]api.AccessGroupRuleDetails, len(objs))
	bRows := manager.SResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	gRows := manager.SAccessGroupResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.AccessGroupRuleDetails{
			ResourceBaseDetails:     bRows[i],
			AccessGroupResourceInfo: gRows[i],
		}
	}

	return rows
}

func (manager *SAccessGroupRuleManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AccessGroupRuleListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SAccessGroupResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.AccessGroupFilterListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SAccessGroupResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SAccessGroupRuleManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SAccessGroupResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SAccessGroupRule) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

// 创建权限组规则
func (manager *SAccessGroupRuleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.AccessGroupRuleCreateInput) (api.AccessGroupRuleCreateInput, error) {
	if len(input.AccessGroupId) == 0 {
		return input, httperrors.NewMissingParameterError("access_group_id")
	}
	if input.AccessGroupId == api.DEFAULT_ACCESS_GROUP {
		return input, httperrors.NewNotSupportedError("can not add rule for default access group")
	}
	_ag, err := validators.ValidateModel(userCred, AccessGroupManager, &input.AccessGroupId)
	if err != nil {
		return input, err
	}
	ag := _ag.(*SAccessGroup)
	if !ag.IsOwner(userCred) && !userCred.HasSystemAdminPrivilege() {
		return input, httperrors.NewForbiddenError("not enough privilege")
	}
	if ag.Status != api.ACCESS_GROUP_STATUS_AVAILABLE {
		return input, httperrors.NewInvalidStatusError("access group %s status is not available", ag.Name)
	}
	if len(input.Source) == 0 {
		return input, httperrors.NewMissingParameterError("source")
	}
	if !regutils.MatchCIDR(input.Source) && !regutils.MatchIP4Addr(input.Source) {
		return input, httperrors.NewInputParameterError("invalid source %s", input.Source)
	}
	if input.Priority < 1 || input.Priority > 100 {
		return input, httperrors.NewOutOfRangeError("Invalid priority %d, must be in range or 1 ~ 100", input.Priority)
	}
	if len(input.RWAccessType) == 0 {
		return input, httperrors.NewMissingParameterError("rw_access_type")
	}
	if isIn, _ := utils.InArray(cloudprovider.TRWAccessType(input.RWAccessType), []cloudprovider.TRWAccessType{
		cloudprovider.RWAccessTypeR,
		cloudprovider.RWAccessTypeRW,
	}); !isIn {
		return input, httperrors.NewInputParameterError("invalid rw_access_type %s", input.RWAccessType)
	}
	if len(input.UserAccessType) == 0 {
		return input, httperrors.NewMissingParameterError("user_access_type")
	}
	if isIn, _ := utils.InArray(cloudprovider.TUserAccessType(input.UserAccessType), []cloudprovider.TUserAccessType{
		cloudprovider.UserAccessTypeAllSquash,
		cloudprovider.UserAccessTypeRootSquash,
		cloudprovider.UserAccessTypeNoRootSquash,
	}); !isIn {
		return input, httperrors.NewInputParameterError("invalid user_access_type %s", input.UserAccessType)
	}

	input.ResourceBaseCreateInput, err = manager.SResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.ResourceBaseCreateInput)
	if err != nil {
		return input, err
	}

	return input, nil
}

func (self *SAccessGroupRule) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	log.Debugf("POST Create %s", data)

	group, err := self.GetAccessGroup()
	if err == nil {
		logclient.AddSimpleActionLog(group, logclient.ACT_ALLOCATE, data, userCred, true)
		group.DoSync(ctx, userCred)
	}
}

func (self *SAccessGroupRule) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.AccessGroupRuleUpdateInput) (api.AccessGroupRuleUpdateInput, error) {
	if input.Priority != nil && *input.Priority < 1 || *input.Priority > 100 {
		return input, httperrors.NewOutOfRangeError("Invalid priority %d, must be in range or 1 ~ 100", input.Priority)
	}
	if len(input.Source) > 0 && !regutils.MatchCIDR(input.Source) {
		return input, httperrors.NewInputParameterError("invalid source %s", input.Source)
	}
	if isIn, _ := utils.InArray(cloudprovider.TRWAccessType(input.RWAccessType), []cloudprovider.TRWAccessType{
		cloudprovider.RWAccessTypeR,
		cloudprovider.RWAccessTypeRW,
	}); !isIn && len(input.RWAccessType) > 0 {
		return input, httperrors.NewInputParameterError("invalid rw_access_type %s", input.RWAccessType)
	}
	if isIn, _ := utils.InArray(cloudprovider.TUserAccessType(input.UserAccessType), []cloudprovider.TUserAccessType{
		cloudprovider.UserAccessTypeAllSquash,
		cloudprovider.UserAccessTypeRootSquash,
		cloudprovider.UserAccessTypeNoRootSquash,
	}); !isIn && len(input.UserAccessType) > 0 {
		return input, httperrors.NewInputParameterError("invalid user_access_type %s", input.UserAccessType)
	}
	return input, nil
}

func (self *SAccessGroupRule) GetOwnerId() mcclient.IIdentityProvider {
	group, err := self.GetAccessGroup()
	if err != nil {
		return nil
	}
	return group.GetOwnerId()
}

func (manager *SAccessGroupRuleManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SAccessGroupResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrapf(err, "SAccessGroupResourceBaseManager.ListItemExportKeys")
	}

	return q, nil
}

func (self *SAccessGroupRule) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	self.SResourceBase.PreDelete(ctx, userCred)

	group, err := self.GetAccessGroup()
	if err == nil {
		logclient.AddSimpleActionLog(group, logclient.ACT_DELETE, jsonutils.Marshal(self), userCred, true)
		group.DoSync(ctx, userCred)
	}
}

func (self *SAccessGroupRule) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if self.AccessGroupId == api.DEFAULT_ACCESS_GROUP {
		return httperrors.NewProtectedResourceError("not allow to delete default access group rule")
	}
	return self.SResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (manager *SAccessGroupRuleManager) InitializeData() error {
	q := manager.Query().Equals("access_group_id", api.DEFAULT_ACCESS_GROUP)
	rules := []SAccessGroupRule{}
	err := db.FetchModelObjects(manager, q, &rules)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	if len(rules) == 0 {
		rule := &SAccessGroupRule{}
		rule.SetModelManager(manager, rule)
		rule.AccessGroupId = api.DEFAULT_ACCESS_GROUP
		rule.Source = "0.0.0.0/0"
		rule.RWAccessType = string(cloudprovider.RWAccessTypeRW)
		rule.UserAccessType = string(cloudprovider.UserAccessTypeNoRootSquash)
		err = manager.TableSpec().Insert(context.TODO(), rule)
		return errors.Wrapf(err, "Insert")
	}
	return nil
}
