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
	"time"

	"gopkg.in/fatih/set.v0"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudgroupManager struct {
	db.SStatusInfrasResourceBaseManager
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

	Provider string `width:"64" charset:"ascii" list:"domain" create:"required"`
}

func (manager *SCloudgroupManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

// 权限组列表
func (manager *SCloudgroupManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.CloudgroupListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, err
	}

	time.Sleep(time.Minute)

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

	if query.Usable != nil && *query.Usable {
		sq := CloudaccountManager.Query("provider").SubQuery()
		q = q.In("provider", sq)
	}

	return q, nil
}

func (manager *SCloudgroupManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	provider, _ := data.GetString("provider")
	return jsonutils.Marshal(map[string]string{"provider": provider})
}

func (manager *SCloudgroupManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	provider, _ := values.GetString("provider")
	if len(provider) > 0 {
		q = q.Equals("provider", provider)
	}
	return q
}

// 更新权限组
func (self *SCloudgroup) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupUpdateInput) (api.CloudgroupUpdateInput, error) {
	return input, nil
}

// 获取权限组详情
func (self *SCloudgroup) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.CloudgroupDetails, error) {
	return api.CloudgroupDetails{}, nil
}

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
	for i := range rows {
		rows[i] = api.CloudgroupDetails{
			StatusInfrasResourceBaseDetails: statusRows[i],
			Cloudpolicies:                   []api.SCloudIdBaseResource{},
		}
		group := objs[i].(*SCloudgroup)
		rows[i].CloudgroupcacheCount, _ = group.GetCloudgroupcacheCount()
		policies, _ := group.GetCloudpolicies()
		for _, policy := range policies {
			rows[i].Cloudpolicies = append(rows[i].Cloudpolicies, api.SCloudIdBaseResource{Id: policy.Id, Name: policy.Name})
		}
		rows[i].CloudpolicyCount = len(policies)
		users, _ := group.GetCloudusers()
		for _, user := range users {
			rows[i].Cloudusers = append(rows[i].Cloudusers, api.SCloudIdBaseResource{Id: user.Id, Name: user.Name})
		}
		rows[i].ClouduserCount = len(users)
	}
	return rows
}

// 创建权限组
func (manager *SCloudgroupManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.CloudgroupCreateInput) (api.CloudgroupCreateInput, error) {
	if len(input.Provider) == 0 {
		return input, httperrors.NewMissingParameterError("provider")
	}
	factory, err := cloudprovider.GetProviderFactory(input.Provider)
	if err != nil {
		return input, errors.Wrap(err, "cloudprovider.GetProviderFactory")
	}
	if !factory.IsSupportCloudIdService() {
		return input, httperrors.NewUnsupportOperationError("Unsupport cloudgroup for %s", input.Provider)
	}
	for _, cloudpolicyId := range input.CloudpolicyIds {
		_policy, err := CloudpolicyManager.FetchById(cloudpolicyId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, httperrors.NewResourceNotFoundError2("cloudpolicy", cloudpolicyId)
			}
			return input, httperrors.NewGeneralError(err)
		}
		policy := _policy.(*SCloudpolicy)
		if policy.Provider != input.Provider {
			return input, httperrors.NewConflictError("cloudpolicy %s(%s) and cloudgroup not with same provider", policy.Name, policy.Id)
		}
	}
	input.Status = api.CLOUD_GROUP_STATUS_AVAILABLE
	return input, nil
}

func (self *SCloudgroup) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	input := api.CloudgroupCreateInput{}
	data.Unmarshal(&input)
	for _, policyId := range input.CloudpolicyIds {
		self.attachPolicy(policyId)
	}
}

func (manager *SCloudgroupManager) newCloudgroup(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, iGroup cloudprovider.ICloudgroup, provider string) (*SCloudgroup, error) {
	group := &SCloudgroup{}
	group.Description = iGroup.GetDescription()
	group.Provider = provider
	group.DomainId = ownerId.GetProjectDomainId()
	group.Status = api.CLOUD_GROUP_STATUS_AVAILABLE
	group.SetModelManager(manager, group)
	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		var err error
		group.Name, err = db.GenerateName(ctx, manager, ownerId, iGroup.GetName())
		if err != nil {
			return errors.Wrapf(err, "db.GenerateName")
		}

		return manager.TableSpec().Insert(ctx, group)
	}()
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}
	return group, nil
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
	self.SetStatus(userCred, api.CLOUD_GROUP_STATUS_DELETING, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudgroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SCloudgroup) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := self.removePolicies()
	if err != nil {
		return errors.Wrap(err, "removePolicies")
	}
	err = self.removeUsers()
	if err != nil {
		return errors.Wrap(err, "remoteUsers")
	}
	err = self.removeSamlusers()
	if err != nil {
		return errors.Wrapf(err, "removeSamlusers")
	}
	return self.SStatusInfrasResourceBase.Delete(ctx, userCred)
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

func (self *SCloudgroup) removeSamlusers() error {
	users, err := self.GetSamlusers()
	if err != nil {
		return errors.Wrap(err, "GetSamlusers")
	}
	for i := range users {
		err = users[i].Delete(context.TODO(), nil)
		if err != nil {
			return errors.Wrapf(err, "rm saml user(%s)", users[i].Id)
		}
	}
	return nil
}

func (self *SCloudgroup) removeUsers() error {
	users, err := self.GetCloudusers()
	if err != nil {
		return errors.Wrap(err, "GetCloudusers")
	}
	for i := range users {
		err = self.removeUser(users[i].Id)
		if err != nil {
			return errors.Wrapf(err, "removeUser(%s)", users[i].Id)
		}
	}
	return nil
}

func (self *SCloudgroup) removePolicies() error {
	policies, err := self.GetCloudpolicies()
	if err != nil {
		return errors.Wrap(err, "GetCloudpolicies")
	}
	for i := range policies {
		err = self.detachPolicy(policies[i].Id)
		if err != nil {
			return errors.Wrapf(err, "detachPolicy(%s)", policies[i].Id)
		}
	}
	return nil
}

func (self *SCloudgroup) GetCloudpolicyQuery() *sqlchemy.SQuery {
	sq := CloudgroupPolicyManager.Query("cloudpolicy_id").Equals("cloudgroup_id", self.Id).SubQuery()
	return CloudpolicyManager.Query().In("id", sq)
}

func (self *SCloudgroup) GetCloudpolicyCount() (int, error) {
	return self.GetCloudpolicyQuery().CountWithError()
}

func (self *SCloudgroup) GetCloudgroupcacheQuery() *sqlchemy.SQuery {
	return CloudgroupcacheManager.Query().Equals("cloudgroup_id", self.Id)
}

func (self *SCloudgroup) GetCloudgroupcacheCount() (int, error) {
	return self.GetCloudgroupcacheQuery().CountWithError()
}

func (self *SCloudgroup) GetCloudgroupcaches() ([]SCloudgroupcache, error) {
	caches := []SCloudgroupcache{}
	q := self.GetCloudgroupcacheQuery()
	err := db.FetchModelObjects(CloudgroupcacheManager, q, &caches)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return caches, nil
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

func (self *SCloudgroup) GetSystemCloudpolicies() ([]SCloudpolicy, error) {
	policies := []SCloudpolicy{}
	q := self.GetCloudpolicyQuery().Equals("policy_type", api.CLOUD_POLICY_TYPE_SYSTEM)
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SCloudgroup) GetCustomCloudpolicies() ([]SCloudpolicy, error) {
	policies := []SCloudpolicy{}
	q := self.GetCloudpolicyQuery().Equals("policy_type", api.CLOUD_POLICY_TYPE_CUSTOM)
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

func (self *SCloudgroup) GetProviderFactory() (cloudprovider.ICloudProviderFactory, error) {
	return cloudprovider.GetProviderFactory(self.Provider)
}

func (self *SCloudgroup) AllowPerformAddUser(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "add-user")
}

// 向权限组加入用户
// 权限组状态必须为: available
func (self *SCloudgroup) PerformAddUser(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupAddUserInput) (jsonutils.JSONObject, error) {
	if len(input.ClouduserId) == 0 {
		return nil, httperrors.NewMissingParameterError("clouduser_id")
	}
	_user, err := ClouduserManager.FetchById(input.ClouduserId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2("clouduser", input.ClouduserId)
		}
		return nil, httperrors.NewGeneralError(err)
	}
	user := _user.(*SClouduser)
	account, err := user.GetCloudaccount()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if account.Provider != self.Provider {
		return nil, httperrors.NewDuplicateResourceError("group and user not with same provider")
	}

	_, err = self.GetClouduser(input.ClouduserId)
	if err == nil || errors.Cause(err) == sqlchemy.ErrDuplicateEntry {
		return nil, httperrors.NewDuplicateResourceError("user %s has aleady in this group", input.ClouduserId)
	}

	err = self.addUser(input.ClouduserId)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "addUser"))
	}

	logclient.AddSimpleActionLog(self, logclient.ACT_ADD_USER, user, userCred, true)
	logclient.AddSimpleActionLog(user, logclient.ACT_ADD_USER, self, userCred, true)
	return nil, self.StartCloudgroupSyncUsersTask(ctx, userCred, "")
}

func (self *SCloudgroup) StartCloudgroupSyncUsersTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "CloudgroupSyncUsersTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.CLOUD_GROUP_STATUS_SYNC_USERS, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudgroup) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "syncstatus")
}

func (self *SCloudgroup) StartCloudgroupSyncstatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "CloudgroupSyncstatusTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.CLOUD_GROUP_STATUS_SYNC_STATUS, "")
	task.ScheduleRun(nil)
	return nil
}

// 恢复权限组状态
func (self *SCloudgroup) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupSyncstatusInput) (jsonutils.JSONObject, error) {
	return nil, self.StartCloudgroupSyncstatusTask(ctx, userCred, "")
}

func (self *SCloudgroup) AllowPerformRemoveUser(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "remove-user")
}

// 从权限组移除用户
// 权限组状态必须为: available
func (self *SCloudgroup) PerformRemoveUser(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupRemoveUserInput) (jsonutils.JSONObject, error) {
	if self.Status != api.CLOUD_GROUP_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not remove user in status %s", self.Status)
	}
	if len(input.ClouduserId) == 0 {
		return nil, httperrors.NewMissingParameterError("clouduser_id")
	}
	_user, err := ClouduserManager.FetchById(input.ClouduserId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2("clouduser", input.ClouduserId)
		}
		return nil, httperrors.NewGeneralError(err)
	}
	user := _user.(*SClouduser)

	_, err = self.GetClouduser(input.ClouduserId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}
		return nil, httperrors.NewGeneralError(err)
	}
	err = self.removeUser(input.ClouduserId)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "RemoveUser"))
	}

	logclient.AddSimpleActionLog(self, logclient.ACT_REMOVE_USER, user, userCred, true)
	logclient.AddSimpleActionLog(user, logclient.ACT_REMOVE_USER, self, userCred, true)
	return nil, self.StartCloudgroupSyncUsersTask(ctx, userCred, "")
}

func (self *SCloudgroup) addUser(userId string) error {
	gu := &SCloudgroupUser{}
	gu.SetModelManager(CloudgroupUserManager, gu)
	gu.ClouduserId = userId
	gu.CloudgroupId = self.Id
	return CloudgroupUserManager.TableSpec().Insert(context.Background(), gu)
}

func (self *SCloudgroup) AllowPerformSetUsers(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "set-users")
}

// 设置权限组用户(全量覆盖)
// 权限组状态必须为: available
func (self *SCloudgroup) PerformSetUsers(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupSetUsersInput) (jsonutils.JSONObject, error) {
	if self.Status != api.CLOUD_GROUP_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not set users in status %s", self.Status)
	}

	users, err := self.GetCloudusers()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	userMaps := map[string]*SClouduser{}
	local := set.New(set.ThreadSafe)
	for i := range users {
		local.Add(users[i].Id)
		userMaps[users[i].Id] = &users[i]
	}

	newU := set.New(set.ThreadSafe)
	for _, userId := range input.ClouduserIds {
		_user, err := ClouduserManager.FetchById(userId)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, httperrors.NewResourceNotFoundError2("clouduser", userId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		user := _user.(*SClouduser)
		account, err := user.GetCloudaccount()
		if err != nil {
			return nil, errors.Wrap(err, "user.GetCloudaccount")
		}
		if account.Provider != self.Provider {
			return nil, httperrors.NewConflictError("user %s(%s) and group not with same provider", user.Name, user.Id)
		}
		newU.Add(userId)
		userMaps[user.Id] = user
	}

	for _, del := range set.Difference(local, newU).List() {
		id := del.(string)
		user, ok := userMaps[id]
		if ok {
			err = self.removeUser(id)
			if err != nil {
				return nil, httperrors.NewGeneralError(errors.Wrap(err, "removeUser"))
			}
			logclient.AddSimpleActionLog(self, logclient.ACT_REMOVE_USER, user, userCred, true)
			logclient.AddSimpleActionLog(user, logclient.ACT_REMOVE_USER, self, userCred, true)
		}
	}

	for _, add := range set.Difference(newU, local).List() {
		id := add.(string)
		user, ok := userMaps[id]
		if ok {
			err = self.addUser(id)
			if err != nil {
				return nil, httperrors.NewGeneralError(errors.Wrap(err, "addUser"))
			}
			logclient.AddSimpleActionLog(self, logclient.ACT_ADD_USER, user, userCred, true)
			logclient.AddSimpleActionLog(user, logclient.ACT_ADD_USER, self, userCred, true)
		}
	}

	return nil, self.StartCloudgroupSyncUsersTask(ctx, userCred, "")
}

func (self *SCloudgroup) AllowPerformSetPolicies(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "set-policies")
}

// 设置权限组添权限(全量覆盖)
// 权限组状态必须为: available
func (self *SCloudgroup) PerformSetPolicies(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupSetPoliciesInput) (jsonutils.JSONObject, error) {
	if self.Status != api.CLOUD_GROUP_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not set policies in status %s", self.Status)
	}

	policies, err := self.GetCloudpolicies()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	policyMaps := map[string]*SCloudpolicy{}
	local := set.New(set.ThreadSafe)
	for i := range policies {
		local.Add(policies[i].Id)
		policyMaps[policies[i].Id] = &policies[i]
	}

	newP := set.New(set.ThreadSafe)
	for _, policyId := range input.CloudpolicyIds {
		_policy, err := CloudpolicyManager.FetchById(policyId)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, httperrors.NewResourceNotFoundError2("cloudpolicy", policyId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		policy := _policy.(*SCloudpolicy)
		err = policy.ValidateUse()
		if err != nil {
			return nil, err
		}
		if policy.Provider != self.Provider {
			return nil, httperrors.NewConflictError("policy %s(%s) and group not with same provider", policy.Name, policy.Id)
		}
		newP.Add(policyId)
		policyMaps[policyId] = policy
	}

	for _, del := range set.Difference(local, newP).List() {
		id := del.(string)
		policy, ok := policyMaps[id]
		if ok {
			err = self.detachPolicy(id)
			if err != nil {
				return nil, httperrors.NewGeneralError(errors.Wrap(err, "detachPolicy"))
			}
			logclient.AddSimpleActionLog(self, logclient.ACT_DETACH_POLICY, policy, userCred, true)
		}
	}

	for _, add := range set.Difference(newP, local).List() {
		id := add.(string)
		policy, ok := policyMaps[id]
		if ok {
			err = self.attachPolicy(id)
			if err != nil {
				return nil, httperrors.NewGeneralError(errors.Wrap(err, "attachPolicy"))
			}
			logclient.AddSimpleActionLog(self, logclient.ACT_ATTACH_POLICY, policy, userCred, true)
		}
	}

	return nil, self.StartCloudgroupSyncPoliciesTask(ctx, userCred, "")
}

func (self *SCloudgroup) StartCloudgroupSyncPoliciesTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "CloudgroupSyncPoliciesTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.CLOUD_GROUP_STATUS_SYNC_POLICIES, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudgroup) AllowPerformAttachPolicy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "attach-policy")
}

// 向权限组添加权限
// 权限组状态必须为: available
func (self *SCloudgroup) PerformAttachPolicy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupAttachPolicyInput) (jsonutils.JSONObject, error) {
	if self.Status != api.CLOUD_GROUP_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not attach policy in status %s", self.Status)
	}

	if len(input.CloudpolicyId) == 0 {
		return nil, httperrors.NewMissingParameterError("cloudpolicy_id")
	}

	_policy, err := CloudpolicyManager.FetchById(input.CloudpolicyId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2("cloudpolicy", input.CloudpolicyId)
		}
		return nil, httperrors.NewGeneralError(err)
	}
	policy := _policy.(*SCloudpolicy)
	err = policy.ValidateUse()
	if err != nil {
		return nil, err
	}
	if policy.Provider != self.Provider {
		return nil, httperrors.NewDuplicateResourceError("group and policy not with same provider")
	}

	_, err = self.GetCloudpolicy(input.CloudpolicyId)
	if err == nil || errors.Cause(err) == sqlchemy.ErrDuplicateEntry {
		return nil, httperrors.NewDuplicateResourceError("policy %s has aleady in this group", input.CloudpolicyId)
	}

	err = self.attachPolicy(input.CloudpolicyId)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "attachPolicy"))
	}

	logclient.AddSimpleActionLog(self, logclient.ACT_ATTACH_POLICY, policy, userCred, true)
	return nil, self.StartCloudgroupSyncPoliciesTask(ctx, userCred, "")
}

func (self *SCloudgroup) AllowPerformDetachPolicy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "detach-policy")
}

// 从权限组移除权限
// 权限组状态必须为: available
func (self *SCloudgroup) PerformDetachPolicy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudgroupDetachPolicyInput) (jsonutils.JSONObject, error) {
	if self.Status != api.CLOUD_GROUP_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not detach policy in status %s", self.Status)
	}
	if len(input.CloudpolicyId) == 0 {
		return nil, httperrors.NewMissingParameterError("cloudpolicy_id")
	}
	_policy, err := CloudpolicyManager.FetchById(input.CloudpolicyId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2("cloudpolicy", input.CloudpolicyId)
		}
		return nil, httperrors.NewGeneralError(err)
	}
	policy := _policy.(*SCloudpolicy)

	_, err = self.GetCloudpolicy(input.CloudpolicyId)
	if err != nil && errors.Cause(err) == sql.ErrNoRows {
		return nil, nil
	}

	err = self.detachPolicy(input.CloudpolicyId)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "detachPolicy"))
	}

	logclient.AddSimpleActionLog(self, logclient.ACT_DETACH_POLICY, policy, userCred, true)
	return nil, self.StartCloudgroupSyncPoliciesTask(ctx, userCred, "")
}

func (self *SCloudgroup) attachPolicy(policyId string) error {
	gp := &SCloudgroupPolicy{}
	gp.SetModelManager(CloudgroupPolicyManager, gp)
	gp.CloudpolicyId = policyId
	gp.CloudgroupId = self.Id
	return CloudgroupPolicyManager.TableSpec().Insert(context.Background(), gp)
}

func (self *SCloudgroup) IsEqual(system, custom []cloudprovider.ICloudpolicy) (bool, error) {
	dbSystem, err := self.GetSystemCloudpolicies()
	if err != nil {
		return false, errors.Wrap(err, "GetCloudpolicies")
	}

	removed := make([]SCloudpolicy, 0)
	commondb := make([]SCloudpolicy, 0)
	commonext := make([]cloudprovider.ICloudpolicy, 0)
	added := make([]cloudprovider.ICloudpolicy, 0)
	err = compare.CompareSets(dbSystem, system, &removed, &commondb, &commonext, &added)
	if err != nil {
		return false, errors.Wrap(err, "CompareSets")
	}
	if len(removed)+len(added) != 0 {
		return false, nil
	}
	dbCustom, err := self.GetCustomCloudpolicies()
	if err != nil {
		return false, errors.Wrapf(err, "GetCustomCloudpolicies")
	}
	if len(custom) != len(dbCustom) {
		return false, nil
	}
	local := set.New(set.ThreadSafe)
	for _, l := range dbCustom {
		local.Add(l.Document)
	}
	remote := set.New(set.ThreadSafe)
	for _, r := range custom {
		doc, _ := r.GetDocument()
		remote.Add(doc)
	}
	if local.IsEqual(remote) {
		return true, nil
	}
	return false, nil
}
