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

	"gopkg.in/fatih/set.v0"

	"yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudid/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudgroupManager struct {
	db.SStatusInfrasResourceBaseManager
	SCloudaccountResourceBaseManager
	SCloudproviderResourceBaseManager
}

var CloudgroupManager *SCloudgroupManager

func init() {
	CloudgroupManager = &SCloudgroupManager{
		SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(
			SCloudgroup{},
			"cloudgroups_tbl",
			"cloudgroup",
			"cloudgroups",
		),
	}
	CloudgroupManager.SetVirtualObject(CloudgroupManager)
}

type SCloudgroup struct {
	db.SStatusInfrasResourceBase
	db.SExternalizedResourceBase

	SCloudaccountResourceBase
	SCloudproviderResourceBase
}

// 权限组列表
func (manager *SCloudgroupManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.CloudgroupListInput) (*sqlchemy.SQuery, error) {
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

	if len(query.ClouduserId) > 0 {
		_, err = ClouduserManager.FetchById(query.ClouduserId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("clouduser", query.ClouduserId)
			}
			return q, httperrors.NewGeneralError(errors.Wrap(err, "ClouduserManager.FetchById"))
		}
		sq := CloudgroupUserManager.Query("cloudgroup_id").Equals("clouduser_id", query.ClouduserId)
		q = q.In("id", sq.SubQuery())
	}

	if len(query.CloudpolicyId) > 0 {
		_, err = CloudpolicyManager.FetchById(query.CloudpolicyId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudpolicy", query.CloudpolicyId)
			}
			return q, httperrors.NewGeneralError(errors.Wrap(err, "CloudpolicyManager.FetchById"))
		}
		sq := CloudgroupPolicyManager.Query("cloudgroup_id").Equals("cloudpolicy_id", query.CloudpolicyId)
		q = q.In("id", sq.SubQuery())
	}

	return q, nil
}

func (self *SCloudgroup) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "CloudgroupDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

func (self *SCloudgroup) StartSyncstatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "CloudgroupSyncstatusTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_SYNC_STATUS, "")
	return task.ScheduleRun(nil)
}

func (manager *SCloudgroupManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	accountId, _ := data.GetString("cloudaccount_id")
	managerId, _ := data.GetString("manager_id")
	return jsonutils.Marshal(map[string]string{"cloudaccount_id": accountId, "manager_id": managerId})
}

func (manager *SCloudgroupManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	accountId, _ := values.GetString("cloudaccount_id")
	if len(accountId) > 0 {
		q = q.Equals("cloudaccount_id", accountId)
	}

	providerId, _ := values.GetString("manager_id")
	if len(providerId) > 0 {
		q = q.Equals("manager_id", providerId)
	}

	return q
}

// 更新权限组
func (self *SCloudgroup) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupUpdateInput) (api.CloudgroupUpdateInput, error) {
	return input, nil
}

// 获取权限组详情
func (manager *SCloudgroupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudgroupDetails {
	rows := make([]api.CloudgroupDetails, len(objs))
	statusRows := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	acRows := manager.SCloudaccountResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	mRows := manager.SCloudproviderResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	groupIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.CloudgroupDetails{
			StatusInfrasResourceBaseDetails: statusRows[i],
			CloudaccountResourceDetails:     acRows[i],
			CloudproviderResourceDetails:    mRows[i],
			Cloudpolicies:                   []api.SCloudIdBaseResource{},
		}
		group := objs[i].(*SCloudgroup)
		groupIds[i] = group.Id
	}

	userSQ := ClouduserManager.Query().SubQuery()
	ugQ := CloudgroupUserManager.Query().SubQuery()
	q := userSQ.Query(
		userSQ.Field("id"),
		userSQ.Field("name"),
		ugQ.Field("cloudgroup_id"),
	).
		Join(ugQ, sqlchemy.Equals(ugQ.Field("clouduser_id"), userSQ.Field("id"))).
		Filter(sqlchemy.In(ugQ.Field("cloudgroup_id"), groupIds))

	userInfo := []struct {
		Id           string
		Name         string
		CloudgroupId string
	}{}
	err := q.All(&userInfo)
	if err != nil {
		log.Errorf("query group user info error: %v", err)
		return rows
	}

	users := map[string][]api.SCloudIdBaseResource{}
	for _, user := range userInfo {
		_, ok := users[user.CloudgroupId]
		if !ok {
			users[user.CloudgroupId] = []api.SCloudIdBaseResource{}
		}
		users[user.CloudgroupId] = append(users[user.CloudgroupId], api.SCloudIdBaseResource{
			Id:   user.Id,
			Name: user.Name,
		})
	}

	policySQ := CloudpolicyManager.Query().SubQuery()
	pgQ := CloudgroupPolicyManager.Query().SubQuery()
	q = policySQ.Query(
		policySQ.Field("id"),
		policySQ.Field("name"),
		pgQ.Field("cloudgroup_id"),
	).
		Join(pgQ, sqlchemy.Equals(pgQ.Field("cloudpolicy_id"), policySQ.Field("id"))).
		Filter(sqlchemy.In(pgQ.Field("cloudgroup_id"), groupIds))

	policyInfo := []struct {
		Id           string
		Name         string
		CloudgroupId string
	}{}
	err = q.All(&policyInfo)
	if err != nil {
		log.Errorf("query group policy info error: %v", err)
		return rows
	}

	policies := map[string][]api.SCloudIdBaseResource{}
	for _, policy := range policyInfo {
		_, ok := policies[policy.CloudgroupId]
		if !ok {
			policies[policy.CloudgroupId] = []api.SCloudIdBaseResource{}
		}
		policies[policy.CloudgroupId] = append(policies[policy.CloudgroupId], api.SCloudIdBaseResource{
			Id:   policy.Id,
			Name: policy.Name,
		})
	}

	for i := range rows {
		rows[i].Cloudusers, _ = users[groupIds[i]]
		rows[i].ClouduserCount = len(rows[i].Cloudusers)
		rows[i].Cloudpolicies, _ = policies[groupIds[i]]
		rows[i].CloudpolicyCount = len(rows[i].Cloudpolicies)
	}

	return rows
}

func (manager *SCloudgroupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "manager":
		managerQuery := CloudproviderManager.Query("name", "id").SubQuery()
		q.AppendField(managerQuery.Field("name", field)).Distinct()
		q = q.Join(managerQuery, sqlchemy.Equals(q.Field("manager_id"), managerQuery.Field("id")))
		return q, nil
	case "account":
		accountQuery := CloudaccountManager.Query("name", "id").SubQuery()
		providers := CloudproviderManager.Query("id", "cloudaccount_id").SubQuery()
		q.AppendField(accountQuery.Field("name", field)).Distinct()
		q = q.Join(providers, sqlchemy.Equals(q.Field("manager_id"), providers.Field("id")))
		q = q.Join(accountQuery, sqlchemy.Equals(providers.Field("cloudaccount_id"), accountQuery.Field("id")))
		return q, nil
	case "provider", "brand":
		accountQuery := CloudaccountManager.Query(field, "id").Distinct().SubQuery()
		providers := CloudproviderManager.Query("id", "cloudaccount_id").SubQuery()
		q.AppendField(accountQuery.Field(field)).Distinct()
		q = q.Join(providers, sqlchemy.Equals(q.Field("manager_id"), providers.Field("id")))
		q = q.Join(accountQuery, sqlchemy.Equals(providers.Field("cloudaccount_id"), accountQuery.Field("id")))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

// 创建权限组
func (manager *SCloudgroupManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.CloudgroupCreateInput,
) (*api.CloudgroupCreateInput, error) {
	if len(input.ManagerId) == 0 {
		return nil, httperrors.NewMissingParameterError("manager_id")
	}
	providerObj, err := validators.ValidateModel(ctx, userCred, CloudproviderManager, &input.ManagerId)
	if err != nil {
		return nil, err
	}
	provider := providerObj.(*SCloudprovider)
	input.CloudaccountId = provider.CloudaccountId
	driver, err := provider.GetDriver()
	if err != nil {
		return nil, err
	}
	input, err = driver.ValidateCreateCloudgroup(ctx, userCred, provider, input)
	if err != nil {
		return nil, err
	}
	input.Status = apis.STATUS_CREATING
	return input, nil
}

func (self *SCloudgroup) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	input := api.CloudgroupCreateInput{}
	data.Unmarshal(&input)
	for _, policyId := range input.CloudpolicyIds {
		self.attachPolicy(policyId)
	}
	self.StartCreateTask(ctx, userCred, "")
}

func (self *SCloudgroup) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "CloudgroupCreateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

// 删除权限组
func (self *SCloudgroup) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	params := jsonutils.NewDict()
	return self.StartCloudgroupDeleteTask(ctx, userCred, params, "")
}

func (self *SCloudgroup) StartCloudgroupDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "CloudgroupDeleteTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

func (self *SCloudgroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SCloudgroup) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.purge(ctx)
}

func (self *SCloudgroup) GetSamlusers() ([]SSamluser, error) {
	q := SamluserManager.Query().Equals("cloudgroup_id", self.Id)
	users := []SSamluser{}
	err := db.FetchModelObjects(SamluserManager, q, &users)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return users, nil
}

func (self *SCloudgroup) GetCloudpolicyQuery() *sqlchemy.SQuery {
	sq := CloudgroupPolicyManager.Query("cloudpolicy_id").Equals("cloudgroup_id", self.Id).SubQuery()
	return CloudpolicyManager.Query().In("id", sq)
}

func (self *SCloudgroup) GetCloudpolicyCount() (int, error) {
	return self.GetCloudpolicyQuery().CountWithError()
}

func (self *SCloudgroup) GetCloudpolicies() ([]SCloudpolicy, error) {
	policies := []SCloudpolicy{}
	q := self.GetCloudpolicyQuery()
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SCloudgroup) GetCloudpolicy(policyId string) (*SCloudpolicy, error) {
	policies := []SCloudpolicy{}
	q := self.GetCloudpolicyQuery().Equals("id", policyId)
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	if len(policies) > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	}
	if len(policies) == 0 {
		return nil, sql.ErrNoRows
	}
	return &policies[0], nil
}

func (self *SCloudgroup) detachPolicy(policyId string) error {
	policies := []SCloudgroupPolicy{}
	q := CloudgroupPolicyManager.Query().Equals("cloudgroup_id", self.Id).Equals("cloudpolicy_id", policyId)
	err := db.FetchModelObjects(CloudgroupPolicyManager, q, &policies)
	if err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}
	for i := range policies {
		err = policies[i].Delete(context.Background(), nil)
		if err != nil {
			return errors.Wrap(err, "Delete")
		}
	}
	return nil
}

func (self *SCloudgroup) GetClouduserQuery() *sqlchemy.SQuery {
	sq := CloudgroupUserManager.Query("clouduser_id").Equals("cloudgroup_id", self.Id).SubQuery()
	return ClouduserManager.Query().In("id", sq)
}

func (self *SCloudgroup) GetClouduserCount() (int, error) {
	return self.GetClouduserQuery().CountWithError()
}

func (self *SCloudgroup) GetClouduser(userId string) (*SClouduser, error) {
	users := []SClouduser{}
	q := self.GetClouduserQuery().Equals("id", userId)
	err := db.FetchModelObjects(ClouduserManager, q, &users)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	if len(users) > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	}
	if len(users) == 0 {
		return nil, sql.ErrNoRows
	}
	return &users[0], nil
}

func (self *SCloudgroup) GetCloudusers() ([]SClouduser, error) {
	users := []SClouduser{}
	q := self.GetClouduserQuery()
	err := db.FetchModelObjects(ClouduserManager, q, &users)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return users, nil
}

func (self *SCloudgroup) removeUser(userId string) error {
	users := []SCloudgroupUser{}
	q := CloudgroupUserManager.Query().Equals("cloudgroup_id", self.Id).Equals("clouduser_id", userId)
	err := db.FetchModelObjects(CloudgroupUserManager, q, &users)
	if err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}
	for i := range users {
		err = users[i].Delete(context.Background(), nil)
		if err != nil {
			return errors.Wrap(err, "Delete")
		}
	}
	return nil
}

// 向权限组加入用户
// 权限组状态必须为: available
func (self *SCloudgroup) PerformAddUser(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupAddUserInput) (jsonutils.JSONObject, error) {
	if self.Status != apis.STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not remove user in status %s", self.Status)
	}
	userObj, err := validators.ValidateModel(ctx, userCred, ClouduserManager, &input.ClouduserId)
	if err != nil {
		return nil, err
	}
	user := userObj.(*SClouduser)
	if user.ManagerId != self.ManagerId || user.CloudaccountId != self.CloudaccountId {
		return nil, httperrors.NewConflictError("Users and user groups do not belong to the same account")
	}
	_, err = self.GetClouduser(input.ClouduserId)
	if err == nil || errors.Cause(err) == sqlchemy.ErrDuplicateEntry {
		return nil, httperrors.NewDuplicateResourceError("user %s has aleady in this group", input.ClouduserId)
	}

	add := []api.GroupUser{
		{
			Name:       user.Name,
			ExternalId: user.ExternalId,
		},
	}

	return nil, self.StartSetUsersTask(ctx, userCred, add, nil, "")
}

func (self *SCloudgroup) StartSetUsersTask(ctx context.Context, userCred mcclient.TokenCredential, add, del []api.GroupUser, parentTaskId string) error {
	params := jsonutils.NewDict()
	if len(add) > 0 {
		params.Set("add", jsonutils.Marshal(add))
	}
	if len(del) > 0 {
		params.Set("del", jsonutils.Marshal(del))
	}
	task, err := taskman.TaskManager.NewTask(ctx, "CloudgroupSetUsersTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_SYNC_STATUS, "")
	return task.ScheduleRun(nil)
}

func (self *SCloudgroup) StartCloudgroupSyncstatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "CloudgroupSyncstatusTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_SYNC_STATUS, "")
	return task.ScheduleRun(nil)
}

// 恢复权限组状态
func (self *SCloudgroup) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupSyncstatusInput) (jsonutils.JSONObject, error) {
	return nil, self.StartCloudgroupSyncstatusTask(ctx, userCred, "")
}

// 从权限组移除用户
// 权限组状态必须为: available
func (self *SCloudgroup) PerformRemoveUser(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupRemoveUserInput) (jsonutils.JSONObject, error) {
	if self.Status != apis.STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not remove user in status %s", self.Status)
	}

	userObj, err := validators.ValidateModel(ctx, userCred, ClouduserManager, &input.ClouduserId)
	if err != nil {
		return nil, err
	}
	user := userObj.(*SClouduser)

	_, err = self.GetClouduser(input.ClouduserId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}
		return nil, httperrors.NewGeneralError(err)
	}

	del := []api.GroupUser{
		{
			Name:       user.Name,
			ExternalId: user.ExternalId,
		},
	}

	return nil, self.StartSetUsersTask(ctx, userCred, nil, del, "")
}

func (self *SCloudgroup) addUser(userId string) error {
	gu := &SCloudgroupUser{}
	gu.SetModelManager(CloudgroupUserManager, gu)
	gu.ClouduserId = userId
	gu.CloudgroupId = self.Id
	return CloudgroupUserManager.TableSpec().Insert(context.Background(), gu)
}

// 设置权限组用户(全量覆盖)
// 权限组状态必须为: available
func (self *SCloudgroup) PerformSetUsers(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupSetUsersInput) (jsonutils.JSONObject, error) {
	if self.Status != apis.STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not set users in status %s", self.Status)
	}

	users, err := self.GetCloudusers()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	userMap := map[string]*SClouduser{}
	local := set.New(set.ThreadSafe)
	for i := range users {
		local.Add(users[i].Id)
		userMap[users[i].Id] = &users[i]
	}

	newU := set.New(set.ThreadSafe)
	for i := range input.ClouduserIds {
		_user, err := validators.ValidateModel(ctx, userCred, ClouduserManager, &input.ClouduserIds[i])
		if err != nil {
			return nil, err
		}
		user := _user.(*SClouduser)
		if user.ManagerId != self.ManagerId || user.CloudaccountId != self.CloudaccountId {
			return nil, httperrors.NewConflictError("Users and user groups do not belong to the same account")
		}
		newU.Add(user.Id)
		userMap[user.Id] = user
	}

	del, add := []api.GroupUser{}, []api.GroupUser{}
	for _, id := range set.Difference(local, newU).List() {
		user := userMap[id.(string)]
		del = append(del, api.GroupUser{
			Name:       user.Name,
			ExternalId: user.ExternalId,
		})
	}
	for _, id := range set.Difference(newU, local).List() {
		user := userMap[id.(string)]
		add = append(add, api.GroupUser{
			Name:       user.Name,
			ExternalId: user.ExternalId,
		})
	}

	return nil, self.StartSetUsersTask(ctx, userCred, add, del, "")
}

// 设置权限组添权限(全量覆盖)
// 权限组状态必须为: available
func (self *SCloudgroup) PerformSetPolicies(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupSetPoliciesInput) (jsonutils.JSONObject, error) {
	if self.Status != apis.STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not set policies in status %s", self.Status)
	}

	policies, err := self.GetCloudpolicies()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	policyMap := map[string]*SCloudpolicy{}
	local := set.New(set.ThreadSafe)
	for i := range policies {
		local.Add(policies[i].Id)
		policyMap[policies[i].Id] = &policies[i]
	}

	newP := set.New(set.ThreadSafe)
	for i := range input.CloudpolicyIds {
		policObj, err := validators.ValidateModel(ctx, userCred, CloudpolicyManager, &input.CloudpolicyIds[i])
		if err != nil {
			return nil, err
		}
		policy := policObj.(*SCloudpolicy)
		if (policy.ManagerId != self.ManagerId && len(self.ManagerId) > 0) || policy.CloudaccountId != self.CloudaccountId {
			return nil, httperrors.NewConflictError("Policies and groups do not belong to the same account")
		}
		newP.Add(policy.Id)
		policyMap[policy.Id] = policy
	}

	del, add := []api.SPolicy{}, []api.SPolicy{}
	for _, id := range set.Difference(local, newP).List() {
		policy := policyMap[id.(string)]
		del = append(del, api.SPolicy{
			Name:       policy.Name,
			ExternalId: policy.ExternalId,
			PolicyType: policy.PolicyType,
		})
	}
	for _, id := range set.Difference(newP, local).List() {
		policy := policyMap[id.(string)]
		add = append(add, api.SPolicy{
			Name:       policy.Name,
			ExternalId: policy.ExternalId,
			PolicyType: policy.PolicyType,
		})
	}

	return nil, self.StartSetPoliciesTask(ctx, userCred, add, del, "")
}

func (self *SCloudgroup) StartSetPoliciesTask(ctx context.Context, userCred mcclient.TokenCredential, add, del []api.SPolicy, parentTaskId string) error {
	params := jsonutils.NewDict()
	if len(add) > 0 {
		params.Set("add", jsonutils.Marshal(add))
	}
	if len(del) > 0 {
		params.Set("del", jsonutils.Marshal(del))
	}
	task, err := taskman.TaskManager.NewTask(ctx, "CloudgroupSetPoliciesTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_SYNC_STATUS, "")
	return task.ScheduleRun(nil)
}

// 向权限组添加权限
// 权限组状态必须为: available
func (self *SCloudgroup) PerformAttachPolicy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupAttachPolicyInput) (jsonutils.JSONObject, error) {
	if self.Status != apis.STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not attach policy in status %s", self.Status)
	}
	policyObj, err := validators.ValidateModel(ctx, userCred, CloudpolicyManager, &input.CloudpolicyId)
	if err != nil {
		return nil, err
	}
	policy := policyObj.(*SCloudpolicy)
	if policy.ManagerId != self.ManagerId || policy.CloudaccountId != self.CloudaccountId {
		return nil, httperrors.NewConflictError("policy and groups do not belong to the same account")
	}
	_, err = self.GetCloudpolicy(input.CloudpolicyId)
	if err == nil || errors.Cause(err) == sqlchemy.ErrDuplicateEntry {
		return nil, httperrors.NewDuplicateResourceError("policy %s has aleady in this group", input.CloudpolicyId)
	}

	add := []api.SPolicy{
		{
			Name:       policy.Name,
			ExternalId: policy.ExternalId,
			PolicyType: policy.PolicyType,
		},
	}

	return nil, self.StartSetPoliciesTask(ctx, userCred, add, nil, "")
}

// 从权限组移除权限
// 权限组状态必须为: available
func (self *SCloudgroup) PerformDetachPolicy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupDetachPolicyInput) (jsonutils.JSONObject, error) {
	if self.Status != apis.STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not detach policy in status %s", self.Status)
	}

	policObj, err := validators.ValidateModel(ctx, userCred, CloudpolicyManager, &input.CloudpolicyId)
	if err != nil {
		return nil, err
	}
	policy := policObj.(*SCloudpolicy)

	_, err = self.GetCloudpolicy(input.CloudpolicyId)
	if err != nil && errors.Cause(err) == sql.ErrNoRows {
		return nil, nil
	}

	del := []api.SPolicy{
		{
			Name:       policy.Name,
			ExternalId: policy.ExternalId,
			PolicyType: policy.PolicyType,
		},
	}

	return nil, self.StartSetPoliciesTask(ctx, userCred, nil, del, "")
}

func (self *SCloudgroup) attachPolicy(policyId string) error {
	gp := &SCloudgroupPolicy{}
	gp.SetModelManager(CloudgroupPolicyManager, gp)
	gp.CloudpolicyId = policyId
	gp.CloudgroupId = self.Id
	return CloudgroupPolicyManager.TableSpec().Insert(context.Background(), gp)
}

func (self *SCloudaccount) SyncCloudgroups(ctx context.Context, userCred mcclient.TokenCredential, iGroups []cloudprovider.ICloudgroup, managerId string) ([]SCloudgroup, []cloudprovider.ICloudgroup, compare.SyncResult) {
	lockman.LockRawObject(ctx, CloudgroupManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, managerId))
	defer lockman.ReleaseRawObject(ctx, CloudgroupManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, managerId))

	result := compare.SyncResult{}
	dbGroups, err := self.GetCloudgroups(managerId)
	if err != nil {
		result.Error(errors.Wrap(err, "GetCloudgroups"))
		return nil, nil, result
	}

	localGroups := []SCloudgroup{}
	remoteGroups := []cloudprovider.ICloudgroup{}

	removed := make([]SCloudgroup, 0)
	commondb := make([]SCloudgroup, 0)
	commonext := make([]cloudprovider.ICloudgroup, 0)
	added := make([]cloudprovider.ICloudgroup, 0)

	err = compare.CompareSets(dbGroups, iGroups, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrap(err, "compare.CompareSets"))
		return nil, nil, result
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].RealDelete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudgroup(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		localGroups = append(localGroups, commondb[i])
		remoteGroups = append(remoteGroups, commonext[i])
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		group, err := self.newCloudgroup(ctx, userCred, added[i], managerId)
		if err != nil {
			result.AddError(err)
			continue
		}
		localGroups = append(localGroups, *group)
		remoteGroups = append(remoteGroups, added[i])
		result.Add()
	}

	return localGroups, remoteGroups, result
}

func (group *SCloudgroup) GetCloudprovider() (*SCloudprovider, error) {
	provider, err := CloudproviderManager.FetchById(group.ManagerId)
	if err != nil {
		return nil, err
	}
	return provider.(*SCloudprovider), nil
}

func (self *SCloudgroup) GetProvider() (cloudprovider.ICloudProvider, error) {
	if len(self.ManagerId) > 0 {
		provider, err := self.GetCloudprovider()
		if err != nil {
			return nil, err
		}
		return provider.GetProvider()
	}
	if len(self.CloudaccountId) > 0 {
		account, err := self.GetCloudaccount()
		if err != nil {
			if err != nil {
				return nil, err
			}
		}
		return account.GetProvider()
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty account info")
}

func (group *SCloudgroup) GetICloudgroup() (cloudprovider.ICloudgroup, error) {
	if len(group.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	provider, err := group.GetProvider()
	if err != nil {
		return nil, err
	}
	groups, err := provider.GetICloudgroups()
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].GetGlobalId() == group.ExternalId {
			return groups[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, group.ExternalId)
}

func (group *SCloudgroup) SyncWithCloudgroup(ctx context.Context, userCred mcclient.TokenCredential, iGroup cloudprovider.ICloudgroup) error {
	_, err := db.Update(group, func() error {
		group.Name = iGroup.GetName()
		group.Status = apis.STATUS_AVAILABLE
		return nil
	})
	return err
}

func (self *SCloudaccount) newCloudgroup(ctx context.Context, userCred mcclient.TokenCredential, iGroup cloudprovider.ICloudgroup, managerId string) (*SCloudgroup, error) {
	group := &SCloudgroup{}
	group.SetModelManager(CloudgroupManager, group)
	group.Name = iGroup.GetName()
	group.ExternalId = iGroup.GetGlobalId()
	group.Status = apis.STATUS_AVAILABLE
	group.CloudaccountId = self.Id
	group.ManagerId = managerId
	group.DomainId = self.DomainId
	err := CloudgroupManager.TableSpec().Insert(ctx, group)
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}
	return group, nil
}

func (self *SCloudgroup) SyncCloudusers(ctx context.Context, userCred mcclient.TokenCredential, iGroup cloudprovider.ICloudgroup) {
	iUsers, err := iGroup.GetICloudusers()
	if err == nil {
		result := self.SyncUsers(ctx, userCred, iUsers)
		log.Debugf("sync cloudusers for group %s result: %s", self.Name, result.Result())
	}
}

func (self *SCloudgroup) SyncUsers(ctx context.Context, userCred mcclient.TokenCredential, iUsers []cloudprovider.IClouduser) compare.SyncResult {
	lockman.LockRawObject(ctx, ClouduserManager.Keyword(), self.Id)
	defer lockman.ReleaseRawObject(ctx, ClouduserManager.Keyword(), self.Id)

	result := compare.SyncResult{}
	dbUsers, err := self.GetCloudusers()
	if err != nil {
		result.Error(errors.Wrap(err, "GetCloudusers"))
		return result
	}

	removed := make([]SClouduser, 0)
	commondb := make([]SClouduser, 0)
	commonext := make([]cloudprovider.IClouduser, 0)
	added := make([]cloudprovider.IClouduser, 0)

	err = compare.CompareSets(dbUsers, iUsers, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrap(err, "compare.CompareSets"))
		return result
	}

	for i := 0; i < len(removed); i++ {
		self.removeUser(removed[i].Id)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	result.UpdateCnt = len(commondb)

	for i := 0; i < len(added); i++ {
		user, err := db.FetchByExternalIdAndManagerId(ClouduserManager, added[i].GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			if len(self.ManagerId) > 0 {
				return q.Equals("manager_id", self.ManagerId)
			}
			return q.Equals("cloudaccount_id", self.CloudaccountId)
		})
		if err != nil {
			result.AddError(errors.Wrapf(err, "Fetch %s", added[i].GetGlobalId()))
			continue
		}
		err = self.addUser(user.GetId())
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SCloudgroup) SyncCloudpolicies(ctx context.Context, userCred mcclient.TokenCredential, iGroup cloudprovider.ICloudgroup) {
	iPolicies, err := iGroup.GetICloudpolicies()
	if err == nil {
		result := self.SyncPolicies(ctx, userCred, iPolicies)
		log.Infof("SyncCloudpolicies for group %s(%s) result: %s", self.Name, self.Id, result.Result())
	}
}

func (self *SCloudgroup) SyncPolicies(ctx context.Context, userCred mcclient.TokenCredential, iPolicies []cloudprovider.ICloudpolicy) compare.SyncResult {
	result := compare.SyncResult{}
	dbPolicies, err := self.GetCloudpolicies()
	if err != nil {
		result.Error(errors.Wrapf(err, "GetCloudpolicies"))
		return result
	}

	removed := make([]SCloudpolicy, 0)
	commondb := make([]SCloudpolicy, 0)
	commonext := make([]cloudprovider.ICloudpolicy, 0)
	added := make([]cloudprovider.ICloudpolicy, 0)

	err = compare.CompareSets(dbPolicies, iPolicies, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrap(err, "compare.CompareSets"))
		return result
	}

	for i := 0; i < len(removed); i++ {
		err := self.detachPolicy(removed[i].Id)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	result.UpdateCnt = len(commondb)

	for i := 0; i < len(added); i++ {
		policy, err := db.FetchByExternalIdAndManagerId(CloudpolicyManager, added[i].GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			if len(self.ManagerId) > 0 {
				return q.Equals("manager_id", self.ManagerId)
			}
			return q.Equals("cloudaccount_id", self.CloudaccountId)
		})
		if err != nil {
			result.AddError(errors.Wrapf(err, "add %s(%s)", added[i].GetName(), added[i].GetGlobalId()))
			continue
		}
		err = self.attachPolicy(policy.GetId())
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (self *SCloudgroup) GetSamlProvider() (*SSAMLProvider, error) {
	q := SAMLProviderManager.Query().Equals("status", apis.STATUS_AVAILABLE).
		Equals("entity_id", options.Options.ApiServer).
		Equals("cloudaccount_id", self.CloudaccountId).
		IsNotEmpty("external_id")
	if len(self.ManagerId) > 0 {
		q = q.Equals("manager_id", self.ManagerId)
	}
	ret := &SSAMLProvider{}
	ret.SetModelManager(SAMLProviderManager, ret)
	err := q.First(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SCloudgroup) GetCloudroles() ([]SCloudrole, error) {
	sq := SamluserManager.Query("cloudrole_id").Equals("cloudgroup_id", self.Id).SubQuery()
	q := CloudroleManager.Query().Equals("cloudgroup_id", self.Id).IsNotEmpty("external_id").In("id", sq)
	ret := []SCloudrole{}
	err := db.FetchModelObjects(CloudroleManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SCloudgroup) GetDetailsSaml(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*api.GetCloudaccountSamlOutput, error) {
	output := &api.GetCloudaccountSamlOutput{}

	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, err
	}
	if account.SAMLAuth.IsFalse() {
		return output, httperrors.NewNotSupportedError("account %s not enable saml auth", account.Name)
	}

	provider, err := account.GetProvider()
	if err != nil {
		return output, errors.Wrap(err, "GetProviderFactory")
	}

	samlProvider, err := self.GetSamlProvider()
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotReadyError("no available saml provider")
		}
		return nil, errors.Wrapf(err, "GetSamlProvider")
	}

	output.EntityId = provider.GetSamlEntityId()
	if len(output.EntityId) == 0 {
		return output, errors.Wrap(httperrors.ErrNotSupported, "SAML login not supported")
	}

	id := self.CloudaccountId
	if len(self.ManagerId) > 0 {
		id = self.ManagerId
	}

	output.RedirectLoginUrl = httputils.JoinPath(options.Options.ApiServer, cloudid.SAML_IDP_PREFIX, "redirect/login", id)
	output.RedirectLogoutUrl = httputils.JoinPath(options.Options.ApiServer, cloudid.SAML_IDP_PREFIX, "redirect/logout", id)
	output.MetadataUrl = httputils.JoinPath(options.Options.ApiServer, cloudid.SAML_IDP_PREFIX, "metadata", id)
	output.InitLoginUrl = samlProvider.AuthUrl
	return output, nil
}
