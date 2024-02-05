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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SAccessGroupRuleManager struct {
	db.SStandaloneAnonResourceBaseManager
	SAccessGroupResourceBaseManager
}

var AccessGroupRuleManager *SAccessGroupRuleManager

func init() {
	AccessGroupRuleManager = &SAccessGroupRuleManager{
		SStandaloneAnonResourceBaseManager: db.NewStandaloneAnonResourceBaseManager(
			SAccessGroupRule{},
			"access_group_rules_tbl",
			"access_group_rule",
			"access_group_rules",
		),
	}
	AccessGroupRuleManager.SetVirtualObject(AccessGroupRuleManager)
}

type SAccessGroupRule struct {
	db.SStandaloneAnonResourceBase
	db.SStatusResourceBase `default:"available"`
	SAccessGroupResourceBase
	// 云上Id, 对应云上资源自身Id
	ExternalId string `width:"256" charset:"utf8" index:"true" list:"user" create:"domain_optional" update:"admin" json:"external_id"`

	Priority       int    `default:"1" list:"user" update:"user" list:"user"`
	Source         string `width:"16" charset:"ascii" list:"user" update:"user" create:"required"`
	RWAccessType   string `width:"16" charset:"ascii" list:"user" update:"user" create:"required"`
	UserAccessType string `width:"16" charset:"ascii" list:"user" update:"user" create:"required"`
}

func (manager *SAccessGroupRuleManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
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

func (manager *SAccessGroupRuleManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	sq := AccessGroupManager.Query("id")
	sq = db.SharableManagerFilterByOwner(ctx, AccessGroupManager, sq, userCred, owner, scope)
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
	return nil
}

func (self *SAccessGroupRule) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SAccessGroupRule) SyncWithAccessGroupRule(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.IAccessGroupRule) error {
	_, err := db.Update(self, func() error {
		self.ExternalId = ext.GetGlobalId()
		self.Source = ext.GetSource()
		self.RWAccessType = string(ext.GetRWAccessType())
		self.UserAccessType = string(ext.GetUserAccessType())
		self.Priority = ext.GetPriority()
		self.Status = apis.STATUS_AVAILABLE
		return nil
	})
	return err
}

// 创建权限组规则
func (manager *SAccessGroupRuleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.AccessGroupRuleCreateInput) (api.AccessGroupRuleCreateInput, error) {
	if len(input.AccessGroupId) == 0 {
		return input, httperrors.NewMissingParameterError("access_group_id")
	}
	_ag, err := validators.ValidateModel(ctx, userCred, AccessGroupManager, &input.AccessGroupId)
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
	self.StartCreateTask(ctx, userCred)
}

func (self *SAccessGroupRule) SetName(name string) {
}

func (self *SAccessGroupRule) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "AccessGroupRuleCreateTask", self, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SAccessGroupRule) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SAccessGroupRule) SetStatus(ctx context.Context, userCred mcclient.TokenCredential, status string, reason string) error {
	_, err := db.Update(self, func() error {
		self.Status = status
		return nil
	})
	return err
}

func (self *SAccessGroupRule) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	var err = func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "AccessGroupRuleDeleteTask", self, userCred, nil, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		self.SetStatus(ctx, userCred, apis.STATUS_DELETE_FAILED, err.Error())
		return nil
	}
	self.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
	return nil
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

func (self *SAccessGroupRule) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	return self.SResourceBase.ValidateDeleteCondition(ctx, nil)
}
