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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudpolicyManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
}

var CloudpolicyManager *SCloudpolicyManager

func init() {
	CloudpolicyManager = &SCloudpolicyManager{
		SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(
			SCloudpolicy{},
			"cloudpolicy_tbl",
			"cloudpolicy",
			"cloudpolicies",
		),
	}
	CloudpolicyManager.SetVirtualObject(CloudpolicyManager)
}

type SCloudpolicy struct {
	db.SStatusInfrasResourceBase
	db.SExternalizedResourceBase

	// 权限类型
	//
	// | 权限类型      |  说明                |
	// |---------------|----------------------|
	// | system        | 平台内置权限         |
	// | custom        | 用户自定义权限       |
	PolicyType string `width:"16" charset:"ascii" list:"domain" default:"custom"`

	// 平台
	//
	// | 云平台   | 说明                                        |
	// |----------|---------------------------------------------|
	// | Google   | 支持                                        |
	// | Aliyun   | 支持										|
	// | Huawei   | 支持                                        |
	// | Azure    | 支持                                        |
	// | 腾讯云   | 支持                                        |
	Provider string `width:"64" charset:"ascii" list:"domain" create:"domain_required"`

	// 策略内容
	Document *jsonutils.JSONDict `length:"long" charset:"ascii" list:"domain" update:"domain" create:"domain_required"`

	// 是否锁定, 若锁定后, 此策略不允许被绑定到用户或权限组, 仅管理员可以设置是否锁定
	Locked tristate.TriState `nullable:"false" get:"user" create:"optional" list:"user" default:"false"`

	CloudEnv string `width:"64" charset:"ascii" list:"domain" create:"domain_required"`
}

func (self SCloudpolicy) GetGlobalId() string {
	return self.ExternalId
}

func (manager *SCloudpolicyManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (manager *SCloudpolicyManager) GetIVirtualModelManager() db.IVirtualModelManager {
	return manager.GetVirtualObject().(db.IVirtualModelManager)
}

func (manager *SCloudpolicyManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	return nil, nil
}

// 公有云权限列表
func (manager *SCloudpolicyManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.CloudpolicyListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, err
	}

	if len(query.Provider) > 0 {
		q = q.In("provider", query.Provider)
	}

	if len(query.PolicyType) > 0 {
		q = q.Equals("policy_type", query.PolicyType)
	}

	if len(query.CloudproviderId) > 0 {
		_, err = CloudproviderManager.FetchById(query.CloudproviderId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudprovider", query.CloudproviderId)
			}
			return q, httperrors.NewGeneralError(errors.Wrap(err, "CloudproviderManager.FetchById"))
		}
		sq := ClouduserPolicyManager.Query("cloudpolicy_id").Equals("cloudprovider_id", query.CloudproviderId)
		q = q.In("id", sq.SubQuery())
	}

	if len(query.ClouduserId) > 0 {
		_, err = ClouduserManager.FetchById(query.ClouduserId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("clouduser", query.ClouduserId)
			}
			return q, httperrors.NewGeneralError(errors.Wrap(err, "ClouduserManager.FetchById"))
		}
		sq := ClouduserPolicyManager.Query("cloudpolicy_id").Equals("clouduser_id", query.ClouduserId)
		q = q.In("id", sq.SubQuery())
	}

	if len(query.CloudgroupId) > 0 {
		_, err = CloudgroupManager.FetchById(query.CloudgroupId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudgroup", query.CloudgroupId)
			}
			return q, httperrors.NewGeneralError(errors.Wrap(err, "CloudgroupManager.FetchById"))
		}
		sq := CloudgroupPolicyManager.Query("cloudpolicy_id").Equals("cloudgroup_id", query.CloudgroupId)
		q = q.In("id", sq.SubQuery())
	}

	if query.Locked == nil || !*query.Locked {
		q = q.IsFalse("locked")
	}

	return q, nil
}

func (manager *SCloudpolicyManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowCreate(userCred, manager)
}

// 创建自定义策略
func (manager *SCloudpolicyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.CloudpolicyCreateInput) (api.CloudpolicyCreateInput, error) {
	var err error
	input.StatusInfrasResourceBaseCreateInput, err = manager.SStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

// 更新策略(仅限自定义)
func (self *SCloudpolicy) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudpolicyUpdateInput) (api.CloudpolicyUpdateInput, error) {
	var err error
	input.StatusInfrasResourceBaseUpdateInput, err = self.SStatusInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusInfrasResourceBaseUpdateInput)
	if err != nil {
		return input, err
	}
	if self.PolicyType != api.CLOUD_POLICY_TYPE_CUSTOM {
		return input, httperrors.NewNotSupportedError("only support update custom policy")
	}
	input.OriginDocument = self.Document
	return input, nil
}

func (self *SCloudpolicy) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStatusInfrasResourceBase.PostUpdate(ctx, userCred, query, data)

	if data.Contains("document") {
		document, err := data.Get("origin_document")
		if err != nil {
			log.Errorf("failed get origin document")
			return
		}
		if !document.Equals(self.Document) {
			err = self.StartCloudpolicyUpdateTask(ctx, userCred, "")
			if err != nil {
				log.Errorf("StartCloudpolicyUpdateTask error: %v", err)
			}
		}
	}
}

func (self *SCloudpolicy) StartCloudpolicyUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "CloudpolicyUpdateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.CLOUD_POLICY_STATUS_SYNCING, "update")
	task.ScheduleRun(nil)
	return nil
}

// 删除自定义权限
func (self *SCloudpolicy) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	err := self.SStatusInfrasResourceBase.CustomizeDelete(ctx, userCred, query, data)
	if err != nil {
		return err
	}
	if self.PolicyType == api.CLOUD_POLICY_TYPE_SYSTEM {
		return httperrors.NewForbiddenError("not allow delete system policy")
	}
	return self.StartCloudpolicyDeleteTask(ctx, userCred, "")
}

func (self *SCloudpolicy) StartCloudpolicyDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "CloudpolicyDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.CLOUD_POLICY_STATUS_DELETING, "delete")
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudpolicy) GetCloudusers() ([]SClouduser, error) {
	sq := ClouduserPolicyManager.Query("clouduser_id").Equals("cloudpolicy_id", self.Id)
	q := ClouduserManager.Query().In("id", sq.SubQuery())
	users := []SClouduser{}
	err := db.FetchModelObjects(ClouduserManager, q, &users)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return users, nil
}

func (self *SCloudpolicy) GetCloudgroups() ([]SCloudgroup, error) {
	sq := CloudgroupPolicyManager.Query("cloudgroup_id").Equals("cloudpolicy_id", self.Id)
	q := CloudgroupManager.Query().In("id", sq.SubQuery())
	groups := []SCloudgroup{}
	err := db.FetchModelObjects(CloudgroupManager, q, &groups)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return groups, nil
}

func (self *SCloudpolicy) ValidateDeleteCondition(ctx context.Context) error {
	users, err := self.GetCloudusers()
	if err != nil {
		return httperrors.NewGeneralError(errors.Wrapf(err, "GetCloudusers"))
	}
	if len(users) > 0 {
		return httperrors.NewNotEmptyError("policy %s has %d users used", self.Name, len(users))
	}
	groups, err := self.GetCloudgroups()
	if err != nil {
		return httperrors.NewGeneralError(errors.Wrapf(err, "GetCloudgroups"))
	}
	if len(groups) > 0 {
		return httperrors.NewNotEmptyError("policy %s has %d groups used", self.Name, len(groups))
	}
	return self.SStatusInfrasResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SCloudpolicy) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}

	return self.RealDelete(ctx, userCred)
}

func (self *SCloudpolicy) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SCloudpolicy) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SCloudpolicy) ValidateUse() error {
	if self.Status != api.CLOUD_POLICY_STATUS_AVAILABLE {
		return httperrors.NewInvalidStatusError("policy %s status is %s", self.Name, self.Status)
	}
	if self.Locked.IsTrue() {
		return httperrors.NewForbiddenError("policy %s is locked", self.Name)
	}
	return nil
}

func (manager *SCloudpolicyManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudpolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SCloudpolicyManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (self *SCloudpolicy) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "syncstatus")
}

// 恢复权限组状态
func (self *SCloudpolicy) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupSyncstatusInput) (jsonutils.JSONObject, error) {
	self.SetStatus(userCred, api.CLOUD_POLICY_STATUS_AVAILABLE, "syncstatus")
	return nil, nil
}

func (self *SCloudpolicy) AllowPerformLock(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "lock")
}

// 锁定权限(禁止使用此权限)
func (self *SCloudpolicy) PerformLock(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudpolicyAssignGroupInput) (jsonutils.JSONObject, error) {
	_, err := db.Update(self, func() error {
		self.Locked = tristate.True
		return nil
	})
	return nil, err
}

func (self *SCloudpolicy) AllowPerformUnlock(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "unlock")
}

// 解锁权限(允许使用此权限)
func (self *SCloudpolicy) PerformUnlock(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudpolicyAssignGroupInput) (jsonutils.JSONObject, error) {
	_, err := db.Update(self, func() error {
		self.Locked = tristate.False
		return nil
	})
	return nil, err
}

func (self *SCloudpolicy) AllowPerformAssignGroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "assign-group")
}

// 将权限加入权限组
func (self *SCloudpolicy) PerformAssignGroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudpolicyAssignGroupInput) (jsonutils.JSONObject, error) {
	if self.Locked.IsTrue() {
		return nil, httperrors.NewForbiddenError("policy %s is locked", self.Name)
	}
	if len(input.CloudgroupId) == 0 {
		return nil, httperrors.NewMissingParameterError("cloudgroup_id")
	}
	_group, err := CloudgroupManager.FetchById(input.CloudgroupId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2("cloudgroup", input.CloudgroupId)
		}
		return nil, httperrors.NewGeneralError(err)
	}
	group := _group.(*SCloudgroup)
	if self.Provider != group.Provider {
		return nil, httperrors.NewConflictError("policy and group not with same provider")
	}
	err = group.attachPolicy(self.Id)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	return nil, group.StartCloudgroupSyncPoliciesTask(ctx, userCred, "")
}

func (self *SCloudpolicy) AllowPerformRevokeGroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "revoke-group")
}

// 将权限从权限组中移除
func (self *SCloudpolicy) PerformRevokeGroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudpolicyRevokeGroupInput) (jsonutils.JSONObject, error) {
	if len(input.CloudgroupId) == 0 {
		return nil, httperrors.NewMissingParameterError("cloudgroup_id")
	}
	_group, err := CloudgroupManager.FetchById(input.CloudgroupId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2("cloudgroup", input.CloudgroupId)
		}
		return nil, httperrors.NewGeneralError(err)
	}
	group := _group.(*SCloudgroup)
	if self.Provider != group.Provider {
		return nil, httperrors.NewConflictError("policy and group not with same provider")
	}
	err = group.detachPolicy(self.Id)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	return nil, group.StartCloudgroupSyncPoliciesTask(ctx, userCred, "")
}

func (self *SCloudpolicy) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

// 获取公有云权限详情
func (manager *SCloudpolicyManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudpolicyDetails {
	rows := make([]api.CloudpolicyDetails, len(objs))
	infsRows := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	policyIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.CloudpolicyDetails{
			StatusInfrasResourceBaseDetails: infsRows[i],
		}
		policy := objs[i].(*SCloudpolicy)
		policyIds[i] = policy.Id
	}
	return rows
}

func (self *SCloudpolicy) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowDelete(userCred, self)
}

func (self *SCloudpolicy) SyncWithCloudpolicy(ctx context.Context, userCred mcclient.TokenCredential, iPolicy SCloudpolicy) error {
	_, err := db.Update(self, func() error {
		self.Name = iPolicy.GetName()
		self.Description = iPolicy.Description
		self.Status = api.CLOUD_POLICY_STATUS_AVAILABLE
		if self.PolicyType == api.CLOUD_POLICY_TYPE_SYSTEM {
			self.IsPublic = true
		}
		self.CloudEnv = iPolicy.CloudEnv
		self.Document = iPolicy.Document
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.Update")
	}
	return nil
}

func (self *SCloudpolicy) newCloudpolicycache(ctx context.Context, userCred mcclient.TokenCredential, iPolicy cloudprovider.ICloudpolicy, cloudaccontId, cloudproviderId string) error {
	cache := &SCloudpolicycache{}
	cache.SetModelManager(CloudpolicycacheManager, cache)
	cache.ExternalId = iPolicy.GetGlobalId()
	cache.CloudpolicyId = self.Id
	cache.Status = api.CLOUD_POLICY_STATUS_AVAILABLE
	cache.CloudaccountId = cloudaccontId
	cache.CloudproviderId = cloudproviderId
	cache.Name = iPolicy.GetName()
	return CloudpolicycacheManager.TableSpec().Insert(ctx, cache)
}

func (self *SCloudpolicy) GetCloudpolicycaches() ([]SCloudpolicycache, error) {
	caches := []SCloudpolicycache{}
	q := CloudpolicycacheManager.Query().Equals("cloudpolicy_id", self.Id)
	err := db.FetchModelObjects(CloudpolicycacheManager, q, &caches)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return caches, nil
}
