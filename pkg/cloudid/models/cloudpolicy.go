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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudpolicyManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudaccountResourceBaseManager
	SCloudproviderResourceBaseManager
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
	SCloudaccountResourceBase
	SCloudproviderResourceBase

	// 权限类型
	//
	// | 权限类型      |  说明                |
	// |---------------|----------------------|
	// | system        | 平台内置权限         |
	// | custom        | 用户自定义权限       |
	PolicyType string `width:"16" charset:"ascii" list:"domain" create:"optional" default:"custom"`

	// 策略内容
	Document *jsonutils.JSONDict `length:"long" charset:"utf8" list:"domain" update:"domain" create:"domain_required"`
}

func (self SCloudpolicy) GetGlobalId() string {
	return self.ExternalId
}

func (manager *SCloudpolicyManager) GetIVirtualModelManager() db.IVirtualModelManager {
	return manager.GetVirtualObject().(db.IVirtualModelManager)
}

func (manager *SCloudpolicyManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	return nil, nil
}

func (self *SCloudpolicy) GetCloudprovider() (*SCloudprovider, error) {
	provider, err := CloudproviderManager.FetchById(self.ManagerId)
	if err != nil {
		return nil, errors.Wrapf(err, "CloudproviderManager.FetchById(%s)", self.ManagerId)
	}
	return provider.(*SCloudprovider), nil
}

func (self *SCloudpolicy) GetProvider() (cloudprovider.ICloudProvider, error) {
	if len(self.ManagerId) > 0 {
		provider, err := self.GetCloudprovider()
		if err != nil {
			return nil, errors.Wrap(err, "GetCloudprovider")
		}
		return provider.GetProvider()
	}
	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudaccount")
	}
	return account.GetProvider()
}

// 公有云权限列表
func (manager *SCloudpolicyManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.CloudpolicyListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, err
	}

	q, err = manager.SCloudaccountResourceBaseManager.ListItemFilter(ctx, q, userCred, query.CloudaccountResourceListInput)
	if err != nil {
		return nil, err
	}

	q, err = manager.SCloudproviderResourceBaseManager.ListItemFilter(ctx, q, userCred, query.CloudproviderResourceListInput)
	if err != nil {
		return nil, err
	}

	if len(query.PolicyType) > 0 {
		q = q.Equals("policy_type", query.PolicyType)
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

// +onecloud:swagger-gen-ignore
func (manager *SCloudpolicyManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.CloudpolicyCreateInput,
) (*api.CloudpolicyCreateInput, error) {
	return nil, cloudprovider.ErrNotImplemented
}

// +onecloud:swagger-gen-ignore
func (self *SCloudpolicy) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.CloudpolicyUpdateInput,
) (*api.CloudpolicyUpdateInput, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SCloudpolicy) ValidateDeleteCondition(ctx context.Context, info *api.CloudpolicyDetails) error {
	if self.PolicyType == api.CLOUD_POLICY_TYPE_SYSTEM {
		return httperrors.NewNotSupportedError("can not delete system policy")
	}
	if gotypes.IsNil(info) {
		info := &api.CloudpolicyDetails{}
		usage, err := CloudpolicyManager.TotalResourceCount([]string{self.Id})
		if err != nil {
			return err
		}
		info.PolicyUsage, _ = usage[self.Id]
	}
	if info.CloudgroupCount > 0 || info.ClouduserCount > 0 {
		return httperrors.NewNotEmptyError("attach %d groups, %d users", info.CloudgroupCount, info.ClouduserCount)
	}
	return self.SStatusInfrasResourceBase.ValidateDeleteCondition(ctx, nil)
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

type SPolicyUsageCount struct {
	Id string
	api.PolicyUsage
}

func (m *SCloudpolicyManager) query(manager db.IModelManager, field string, policyIds []string, filter func(*sqlchemy.SQuery) *sqlchemy.SQuery) *sqlchemy.SSubQuery {
	q := manager.Query()

	if filter != nil {
		q = filter(q)
	}

	sq := q.SubQuery()

	return sq.Query(
		sq.Field("cloudpolicy_id"),
		sqlchemy.COUNT(field),
	).In("cloudpolicy_id", policyIds).GroupBy(sq.Field("cloudpolicy_id")).SubQuery()
}

func (manager *SCloudpolicyManager) TotalResourceCount(policyIds []string) (map[string]api.PolicyUsage, error) {
	// group
	groupSQ := manager.query(CloudgroupPolicyManager, "group_cnt", policyIds, nil)

	userSQ := manager.query(ClouduserPolicyManager, "user_cnt", policyIds, nil)

	policy := manager.Query().SubQuery()
	policyQ := policy.Query(
		sqlchemy.SUM("cloudgroup_count", groupSQ.Field("group_cnt")),
		sqlchemy.SUM("clouduser_count", userSQ.Field("user_cnt")),
	)

	policyQ.AppendField(policyQ.Field("id"))

	policyQ = policyQ.LeftJoin(groupSQ, sqlchemy.Equals(policyQ.Field("id"), groupSQ.Field("cloudpolicy_id")))
	policyQ = policyQ.LeftJoin(userSQ, sqlchemy.Equals(policyQ.Field("id"), userSQ.Field("cloudpolicy_id")))

	policyQ = policyQ.Filter(sqlchemy.In(policyQ.Field("id"), policyIds)).GroupBy(policyQ.Field("id"))

	policyCount := []SPolicyUsageCount{}
	err := policyQ.All(&policyCount)
	if err != nil {
		return nil, errors.Wrapf(err, "policyQ.All")
	}

	result := map[string]api.PolicyUsage{}
	for i := range policyCount {
		result[policyCount[i].Id] = policyCount[i].PolicyUsage
	}

	return result, nil
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
	acRows := manager.SCloudaccountResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	mRows := manager.SCloudproviderResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	policyIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.CloudpolicyDetails{
			StatusInfrasResourceBaseDetails: infsRows[i],
			CloudaccountResourceDetails:     acRows[i],
			CloudproviderResourceDetails:    mRows[i],
		}
		policy := objs[i].(*SCloudpolicy)
		policyIds[i] = policy.Id
	}

	usage, err := manager.TotalResourceCount(policyIds)
	if err != nil {
		log.Errorf("TotalResourceCount error: %v", err)
		return rows
	}

	for i := range rows {
		rows[i].PolicyUsage, _ = usage[policyIds[i]]
	}

	return rows
}

func (self *SCloudpolicy) SyncWithCloudpolicy(ctx context.Context, userCred mcclient.TokenCredential, iPolicy cloudprovider.ICloudpolicy) error {
	_, err := db.Update(self, func() error {
		self.Name = iPolicy.GetName()
		if self.PolicyType == api.CLOUD_POLICY_TYPE_CUSTOM || len(self.Description) == 0 {
			self.Description = iPolicy.GetDescription()
		}
		self.Status = apis.STATUS_AVAILABLE
		self.IsPublic = true
		if self.PolicyType == api.CLOUD_POLICY_TYPE_CUSTOM || gotypes.IsNil(self.Document) {
			doc, err := iPolicy.GetDocument()
			if err != nil {
				return errors.Wrapf(err, "GetDocument")
			}
			self.Document = doc
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.Update")
	}
	return nil
}

func (self *SCloudaccount) newCloudpolicy(ctx context.Context, userCred mcclient.TokenCredential, iPolicy cloudprovider.ICloudpolicy, managerId string) (*SCloudpolicy, error) {
	policy := &SCloudpolicy{}
	policy.SetModelManager(CloudpolicyManager, policy)
	doc, err := iPolicy.GetDocument()
	if err != nil {
		return nil, err
	}
	policy.Document = doc
	policy.Name = iPolicy.GetName()
	policy.Status = apis.STATUS_AVAILABLE
	policy.PolicyType = string(iPolicy.GetPolicyType())
	policy.IsPublic = true
	policy.ExternalId = iPolicy.GetGlobalId()
	policy.Description = iPolicy.GetDescription()
	policy.CloudaccountId = self.Id
	policy.ManagerId = managerId
	return policy, CloudpolicyManager.TableSpec().Insert(ctx, policy)
}

func (self *SCloudaccount) SyncPolicies(ctx context.Context, userCred mcclient.TokenCredential, iPolicies []cloudprovider.ICloudpolicy, managerId string) compare.SyncResult {
	lockman.LockRawObject(ctx, CloudproviderManager.Keyword(), managerId)
	defer lockman.ReleaseRawObject(ctx, CloudproviderManager.Keyword(), managerId)

	result := compare.SyncResult{}

	removed := make([]SCloudpolicy, 0)
	commondb := make([]SCloudpolicy, 0)
	commonext := make([]cloudprovider.ICloudpolicy, 0)
	added := make([]cloudprovider.ICloudpolicy, 0)

	dbPolicies, err := self.GetCloudpolicies(managerId)
	if err != nil {
		result.Error(errors.Wrapf(err, "GetCloudpolicies"))
		return result
	}

	err = compare.CompareSets(dbPolicies, iPolicies, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrapf(err, "compare.CompareSets"))
		return result
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].Delete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudpolicy(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		_, err := self.newCloudpolicy(ctx, userCred, added[i], managerId)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}
