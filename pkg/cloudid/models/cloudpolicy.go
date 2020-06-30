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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudpolicyManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
}

var CloudpolicyManager *SCloudpolicyManager

func init() {
	CloudpolicyManager = &SCloudpolicyManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SCloudpolicy{},
			"cloudpolicy_tbl",
			"cloudpolicy",
			"cloudpolicies",
		),
	}
	CloudpolicyManager.SetVirtualObject(CloudpolicyManager)
}

type SCloudpolicy struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	// 权限类型
	//
	// | 权限类型      |  说明                |
	// |---------------|----------------------|
	// | system        | 平台内置权限         |
	// | custom        | 目前不支持           |
	PolicyType string `width:"16" charset:"ascii" list:"domain" default:"custom"`

	// 平台
	Provider string `width:"64" charset:"ascii" list:"domain"`
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
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}

	if len(query.Provider) > 0 {
		q = q.In("provider", query.Provider)
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

	return q, nil
}

func (manager *SCloudpolicyManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

// +onecloud:swagger-gen-ignore
func (manager *SCloudpolicyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.CloudpolicyCreateInput) (api.CloudpolicyCreateInput, error) {
	return input, httperrors.NewNotImplementedError("Not Implement")
}

// +onecloud:swagger-gen-ignore
func (self *SCloudpolicy) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudpolicyUpdateInput) (api.CloudpolicyUpdateInput, error) {
	return input, nil
}

// +onecloud:swagger-gen-ignore
func (self *SCloudpolicy) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.SStatusStandaloneResourceBase.CustomizeDelete(ctx, userCred, query, data)
}

func (manager *SCloudpolicyManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudpolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SCloudpolicyManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (self *SCloudpolicy) AllowPerformAssignGroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "assign-group")
}

// 将权限加入权限组
func (self *SCloudpolicy) PerformAssignGroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudpolicyAssignGroupInput) (jsonutils.JSONObject, error) {
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
func (self *SCloudpolicy) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.CloudpolicyDetails, error) {
	return api.CloudpolicyDetails{}, nil
}

func (manager *SCloudpolicyManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudpolicyDetails {
	rows := make([]api.CloudpolicyDetails, len(objs))
	statusRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.CloudpolicyDetails{
			StatusStandaloneResourceDetails: statusRows[i],
		}
	}
	return rows
}

func (self *SCloudpolicy) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (self *SCloudpolicy) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (manager *SCloudpolicyManager) newFromCloudpolicy(ctx context.Context, userCred mcclient.TokenCredential, iPolicy cloudprovider.ICloudpolicy, provider string) (*SCloudpolicy, error) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	policy := &SCloudpolicy{}
	policy.SetModelManager(manager, policy)
	policy.Name = iPolicy.GetName()
	policy.Status = api.CLOUD_POLICY_STATUS_AVAILABLE
	policy.PolicyType = api.CLOUD_POLICY_TYPE_SYSTEM
	policy.Provider = provider
	policy.ExternalId = iPolicy.GetGlobalId()
	policy.Description = iPolicy.GetDescription()
	return policy, manager.TableSpec().Insert(ctx, policy)
}

func (self *SCloudpolicy) SyncWithCloudpolicy(ctx context.Context, userCred mcclient.TokenCredential, iPolicy cloudprovider.ICloudpolicy) error {
	_, err := db.Update(self, func() error {
		self.Name = iPolicy.GetName()
		self.Description = iPolicy.GetDescription()
		self.Status = api.CLOUD_POLICY_STATUS_AVAILABLE
		return nil
	})
	return err
}
