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
	"strings"

	"gopkg.in/fatih/set.v0"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SClouduserManager struct {
	db.SStatusUserResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudaccountResourceBaseManager
	SCloudproviderResourceBaseManager
}

var ClouduserManager *SClouduserManager

func init() {
	ClouduserManager = &SClouduserManager{
		SStatusUserResourceBaseManager: db.NewStatusUserResourceBaseManager(
			SClouduser{},
			"cloudusers_tbl",
			"clouduser",
			"cloudusers",
		),
	}
	ClouduserManager.NameRequireAscii = false
	ClouduserManager.SetVirtualObject(ClouduserManager)
}

type SClouduser struct {
	db.SStatusUserResourceBase
	db.SExternalizedResourceBase
	SCloudproviderResourceBase
	SCloudaccountResourceBase

	Secret string `length:"0" charset:"ascii" nullable:"true" list:"user" create:"domain_optional"`
	// 是否可以控制台登录
	IsConsoleLogin tristate.TriState `nullable:"false" default:"false" list:"user" create:"optional"`
	// 手机号码
	MobilePhone string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"domain_optional"`
	// 邮箱地址
	Email string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"domain_optional"`
}

func (manager *SClouduserManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (manager *SClouduserManager) GetIVirtualModelManager() db.IVirtualModelManager {
	return manager.GetVirtualObject().(db.IVirtualModelManager)
}

// 公有云用户列表
func (manager *SClouduserManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.ClouduserListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusUserResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusUserResourceListInput)
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

	if len(query.CloudpolicyId) > 0 {
		_, err = CloudpolicyManager.FetchById(query.CloudpolicyId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudpolicy", query.CloudpolicyId)
			}
			return q, httperrors.NewGeneralError(errors.Wrap(err, "CloudpolicyManager.FetchById"))
		}
		sq := ClouduserPolicyManager.Query("clouduser_id").Equals("cloudpolicy_id", query.CloudpolicyId)
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
		sq := CloudgroupUserManager.Query("clouduser_id").Equals("cloudgroup_id", query.CloudgroupId)
		q = q.In("id", sq.SubQuery())
	}

	return q, nil
}

// +onecloud:swagger-gen-ignore
func (self *SClouduser) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserUpdateInput) (api.ClouduserUpdateInput, error) {
	return input, nil
}

func (manager *SClouduserManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ClouduserListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusUserResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusUserResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusUserResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SClouduserManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusUserResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

// 获取公有云用户详情
func (self *SClouduser) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.ClouduserDetails, error) {
	return api.ClouduserDetails{}, nil
}

func (manager *SClouduserManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ClouduserDetails {
	rows := make([]api.ClouduserDetails, len(objs))
	userRows := manager.SStatusUserResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	acRows := manager.SCloudaccountResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	apRows := manager.SCloudproviderResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.ClouduserDetails{
			StatusUserResourceDetails:    userRows[i],
			CloudaccountResourceDetails:  acRows[i],
			CloudproviderResourceDetails: apRows[i],
		}
		user := objs[i].(*SClouduser)
		policies, _ := user.GetCloudpolicies()
		for _, policy := range policies {
			rows[i].Cloudpolicies = append(rows[i].Cloudpolicies, api.SCloudIdBaseResource{Id: policy.Id, Name: policy.Name})
		}
		rows[i].CloudpolicyCount = len(policies)
		groups, _ := user.GetCloudgroups()
		for _, group := range groups {
			rows[i].Cloudgroups = append(rows[i].Cloudpolicies, api.SCloudIdBaseResource{Id: group.Id, Name: group.Name})
		}
		rows[i].CloudgroupCount = len(groups)
	}

	return rows
}

func (self *SClouduser) GetIClouduser() (cloudprovider.IClouduser, error) {
	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudaccount")
	}
	provider, err := account.GetProvider()
	if err != nil {
		return nil, errors.Wrap(err, "GetProvider")
	}
	return provider.GetIClouduserByName(self.Name)
}

func (manager *SClouduserManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowCreate(userCred, manager)
}

func (self *SClouduser) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsDomainAllowUpdate(userCred, self)
}

func (self *SClouduser) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowDelete(userCred, self)
}

func (manager *SClouduserManager) FetchParentId(ctx context.Context, data jsonutils.JSONObject) string {
	input := api.ClouduserCreateInput{}
	data.Unmarshal(&input)
	if len(input.CloudproviderId) > 0 {
		return fmt.Sprintf("cloudprovider_id/%s", input.CloudproviderId)
	}
	if len(input.CloudaccountId) > 0 {
		return fmt.Sprintf("cloudaccount_id/%s", input.CloudaccountId)
	}
	return ""
}

func (manager *SClouduserManager) FilterByParentId(q *sqlchemy.SQuery, parentId string) *sqlchemy.SQuery {
	if len(parentId) > 0 {
		if info := strings.Split(parentId, "/"); len(info) == 2 {
			q = q.Equals(info[0], info[1])
		}
	}
	return q
}

// 创建公有云用户
func (manager *SClouduserManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ClouduserCreateInput) (api.ClouduserCreateInput, error) {
	var cp *SCloudprovider = nil
	var ca *SCloudaccount = nil
	if len(input.CloudproviderId) > 0 {
		provider, err := CloudproviderManager.FetchProvider(ctx, input.CloudproviderId)
		if err != nil {
			return input, httperrors.NewGeneralError(errors.Wrap(err, "CloudproviderManager.FetchProvider"))
		}
		cp = provider
		factory, err := cloudprovider.GetProviderFactory(provider.Provider)
		if err != nil {
			return input, httperrors.NewGeneralError(errors.Wrap(err, "GetProviderFactory"))
		}
		if !factory.IsClouduserBelongCloudprovider() {
			return input, httperrors.NewInputParameterError("clouduser not belong cloudprovider for %s", provider.Provider)
		}
		input.CloudproviderId = provider.Id
		input.CloudaccountId = provider.CloudaccountId
	}
	if len(input.CloudaccountId) == 0 {
		return input, httperrors.NewMissingParameterError("cloudaccount_id")
	}
	account, err := CloudaccountManager.FetchAccount(ctx, input.CloudaccountId)
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrap(err, "FetchAccount"))
	}
	if !account.IsSupportCloudId.Bool() {
		return input, httperrors.NewUnsupportOperationError("account %s not support create clouduser", account.Name)
	}
	ca = account
	delegate, err := account.getCloudDelegate(ctx)
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrap(err, "getCloudaccountDelegate"))
	}
	factory, err := account.GetProviderFactory()
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrap(err, "GetProviderFactory"))
	}
	if !factory.IsSupportClouduser() {
		return input, httperrors.NewUnsupportOperationError("Not support clouduser for provider %s", delegate.Provider)
	}
	if factory.IsClouduserBelongCloudprovider() && len(input.CloudproviderId) == 0 {
		return input, httperrors.NewMissingParameterError("cloudprovider_id")
	}
	input.CloudaccountId = delegate.Id
	input.StatusUserResourceCreateInput, err = manager.SStatusUserResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusUserResourceCreateInput)
	if err != nil {
		return input, err
	}
	user, err := db.UserCacheManager.FetchUserById(ctx, input.OwnerId)
	if err != nil {
		return input, errors.Wrap(err, "FetchUserById")
	}
	if len(input.Name) == 0 {
		input.Name = user.Name
	}

	policyExternalIds := []string{}
	for _, policyId := range input.CloudpolicyIds {
		_policy, err := CloudpolicyManager.FetchById(policyId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, httperrors.NewResourceNotFoundError2("cloudpolicy", policyId)
			}
			return input, httperrors.NewGeneralError(err)
		}
		policy := _policy.(*SCloudpolicy)
		if policy.Provider != account.Provider {
			return input, httperrors.NewConflictError("%s policy can not apply for %s user", policy.Provider, account.Provider)
		}
		if policy.PolicyType != api.CLOUD_POLICY_TYPE_SYSTEM {
			return input, httperrors.NewInputParameterError("current not support custom cloudpolicy")
		}
		policyExternalIds = append(policyExternalIds, policy.ExternalId)
	}

	for _, groupId := range input.CloudgroupIds {
		_group, err := CloudgroupManager.FetchById(groupId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, httperrors.NewResourceNotFoundError2("cloudgroup", groupId)
			}
			return input, httperrors.NewGeneralError(err)
		}
		group := _group.(*SCloudgroup)
		if group.Provider != account.Provider {
			return input, httperrors.NewConflictError("%s group can not apply for %s user", group.Provider, account.Provider)
		}
		policies, err := group.GetCloudpolicies()
		if err != nil {
			return input, httperrors.NewGeneralError(errors.Wrap(err, "group.GetCloudpolicies"))
		}
		if !factory.IsSupportCreateCloudgroup() {
			for _, policy := range policies {
				if policy.PolicyType == api.CLOUD_POLICY_TYPE_SYSTEM {
					policyExternalIds = append(policyExternalIds, policy.ExternalId)
				}
			}
		}
	}

	if len(input.CloudpolicyIds) > 0 && !factory.IsSupportClouduserPolicy() {
		return input, httperrors.NewInputParameterError("%s not support user policy", account.Provider)
	}

	if len(policyExternalIds) == 0 && factory.IsClouduserNeedInitPolicy() {
		return input, httperrors.NewMissingParameterError("cloudpolicy_ids or cloudgroup_ids")
	}

	var p cloudprovider.ICloudProvider = nil

	if cp != nil {
		p, err = cp.GetProvider()
	} else if ca != nil {
		p, err = ca.GetProvider()
	}

	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrap(err, "cp.GetProvider"))
	}

	isConsoleLogin := true
	if input.IsConsoleLogin != nil && !*input.IsConsoleLogin {
		isConsoleLogin = false
	}
	input.IsConsoleLogin = &isConsoleLogin

	if isConsoleLogin && len(input.Password) == 0 && factory.IsSupportResetClouduserPassword() {
		input.Password = seclib2.RandomPassword2(12)
	}

	if len(input.Password) > 0 && !factory.IsSupportResetClouduserPassword() {
		return input, httperrors.NewInputParameterError("%s not support password", account.Provider)
	}

	if len(input.Password) > 0 {
		if !seclib2.MeetComplxity(input.Password) {
			return input, httperrors.NewWeakPasswordError()
		}
	}

	conf := cloudprovider.SClouduserCreateConfig{
		Name:              input.Name,
		Desc:              input.Description,
		Password:          input.Password,
		IsConsoleLogin:    isConsoleLogin,
		Email:             input.Email,
		MobilePhone:       input.MobilePhone,
		ExternalPolicyIds: policyExternalIds,
	}

	iUser, err := p.CreateIClouduser(&conf)
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrap(err, "p.CreateIClouduser"))
	}
	input.ExternalId = iUser.GetGlobalId()

	input.OwnerId = user.Id
	return input, nil
}

func (self *SClouduser) SavePassword(password string) error {
	sec, err := utils.EncryptAESBase64(self.Id, password)
	if err != nil {
		return err
	}
	_, err = db.Update(self, func() error {
		self.Secret = sec
		return nil
	})
	return err
}

func (self *SClouduser) GetPassword() (string, error) {
	return utils.DescryptAESBase64(self.Id, self.Secret)
}

func (self *SClouduser) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	input := api.ClouduserCreateInput{}
	data.Unmarshal(&input)
	if len(input.Password) > 0 {
		self.SavePassword(input.Password)
	}
	for _, policyId := range input.CloudpolicyIds {
		self.attachPolicy(policyId)
	}
	for _, groupId := range input.CloudgroupIds {
		self.joinGroup(groupId)
	}
	self.StartClouduserSyncTask(ctx, userCred, "")
}

func (self *SClouduser) SyncWithClouduser(ctx context.Context, userCred mcclient.TokenCredential, iUser cloudprovider.IClouduser, cloudproviderId string) error {
	_, err := db.Update(self, func() error {
		self.Name = iUser.GetName()
		self.Status = api.CLOUD_USER_STATUS_AVAILABLE
		switch iUser.IsConsoleLogin() {
		case true:
			self.IsConsoleLogin = tristate.True
		case false:
			self.IsConsoleLogin = tristate.False
		}
		self.CloudproviderId = cloudproviderId
		return nil
	})
	return err
}

func (manager *SClouduserManager) newFromClouduser(ctx context.Context, userCred mcclient.TokenCredential, iUser cloudprovider.IClouduser, accountId, providerId string) (*SClouduser, error) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	user := &SClouduser{}
	user.SetModelManager(manager, user)
	user.Name = iUser.GetName()
	user.ExternalId = iUser.GetGlobalId()
	user.Status = api.CLOUD_USER_STATUS_AVAILABLE
	user.CloudaccountId = accountId
	user.CloudproviderId = providerId
	switch iUser.IsConsoleLogin() {
	case true:
		user.IsConsoleLogin = tristate.True
	case false:
		user.IsConsoleLogin = tristate.False
	}
	err := manager.TableSpec().Insert(ctx, user)
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}
	return user, nil
}

func (self *SClouduser) detachPolicy(policyId string) error {
	policies := []SClouduserPolicy{}
	q := ClouduserPolicyManager.Query().Equals("clouduser_id", self.Id).Equals("cloudpolicy_id", policyId)
	err := db.FetchModelObjects(ClouduserPolicyManager, q, &policies)
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

func (self *SClouduser) SyncCloudpolicies(ctx context.Context, userCred mcclient.TokenCredential, iPolicies []cloudprovider.ICloudpolicy) compare.SyncResult {
	result := compare.SyncResult{}
	dbPolicies, err := self.GetCloudpolicies()
	if err != nil {
		result.Error(errors.Wrap(err, "GetClouduserPolicies"))
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

	result.UpdateCnt = len(commondb)

	for i := 0; i < len(added); i++ {
		err = ClouduserPolicyManager.newFromClouduserPolicy(ctx, userCred, added[i], self)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (self *SClouduser) GetCloudgroupcaches() ([]SCloudgroupcache, error) {
	q := CloudgroupcacheManager.Query().Equals("cloudaccount_id", self.CloudaccountId)
	sq := CloudgroupUserManager.Query("cloudgroup_id").Equals("clouduser_id", self.Id).SubQuery()
	q = q.In("cloudgroup_id", sq)
	caches := []SCloudgroupcache{}
	err := db.FetchModelObjects(CloudgroupcacheManager, q, &caches)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return caches, nil
}

func (self *SClouduser) SyncCloudgroups(ctx context.Context, userCred mcclient.TokenCredential, iGroups []cloudprovider.ICloudgroup) compare.SyncResult {
	result := compare.SyncResult{}
	dbGroups, err := self.GetCloudgroupcaches()
	if err != nil {
		result.Error(errors.Wrap(err, "GetCloudgroupcaches"))
		return result
	}

	removed := make([]SCloudgroupcache, 0)
	commondb := make([]SCloudgroupcache, 0)
	commonext := make([]cloudprovider.ICloudgroup, 0)
	added := make([]cloudprovider.ICloudgroup, 0)

	err = compare.CompareSets(dbGroups, iGroups, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrap(err, "compare.CompareSets"))
		return result
	}

	result.UpdateCnt = len(commondb)

	for i := 0; i < len(added); i++ {
		_cache, err := db.FetchByExternalId(CloudgroupcacheManager, added[i].GetGlobalId())
		if err != nil {
			result.AddError(errors.Wrapf(err, "FetchByExternalId(%s)", added[i].GetGlobalId()))
			continue
		}
		cache := _cache.(*SCloudgroupcache)
		err = self.joinGroup(cache.CloudgroupId)
		if err != nil {
			result.AddError(errors.Wrap(err, "joinGroup"))
			continue
		}
		result.Add()
	}

	return result
}

// 删除公有云用户
func (self *SClouduser) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	params := jsonutils.NewDict()
	return self.StartClouduserDeleteTask(ctx, userCred, params, "")
}

func (self *SClouduser) StartClouduserDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClouduserDeleteTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.CLOUD_USER_STATUS_DELETING, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SClouduser) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SClouduser) GetClouduserPolicies() ([]SClouduserPolicy, error) {
	policies := []SClouduserPolicy{}
	q := ClouduserPolicyManager.Query().Equals("clouduser_id", self.Id)
	err := db.FetchModelObjects(ClouduserPolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SClouduser) GetCloudpolicyQuery() *sqlchemy.SQuery {
	sq := ClouduserPolicyManager.Query("cloudpolicy_id").Equals("clouduser_id", self.Id).SubQuery()
	return CloudpolicyManager.Query().In("id", sq)
}

func (self *SClouduser) GetCloudpolicyCount() (int, error) {
	return self.GetCloudpolicyQuery().CountWithError()
}

func (self *SClouduser) GetCloudpolicies() ([]SCloudpolicy, error) {
	policies := []SCloudpolicy{}
	q := self.GetCloudpolicyQuery()
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SClouduser) joinGroup(groupId string) error {
	gu := &SCloudgroupUser{}
	gu.SetModelManager(CloudgroupUserManager, gu)
	gu.ClouduserId = self.Id
	gu.CloudgroupId = groupId
	return CloudgroupUserManager.TableSpec().Insert(context.Background(), gu)
}

func (self *SClouduser) leaveGroup(groupId string) error {
	ugs := []SCloudgroupUser{}
	q := CloudgroupUserManager.Query().Equals("clouduser_id", self.Id).Equals("cloudgroup_id", groupId)
	err := db.FetchModelObjects(CloudgroupUserManager, q, &ugs)
	if err != nil {
		return errors.Wrap(err, "FetchModelObjects")
	}
	for i := range ugs {
		err = ugs[i].Delete(context.Background(), nil)
		if err != nil {
			return errors.Wrapf(err, "ug %d Delete", ugs[i].RowId)
		}
	}
	return nil
}

func (self *SClouduser) leaveGroups() error {
	q := CloudgroupUserManager.Query().Equals("clouduser_id", self.Id)
	groups := []SCloudgroupUser{}
	err := db.FetchModelObjects(CloudgroupUserManager, q, &groups)
	if err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}
	for i := range groups {
		err = groups[i].Delete(context.Background(), nil)
		if err != nil {
			return errors.Wrap(err, "group user delete")
		}
	}
	return nil
}

func (self *SClouduser) removePolicies() error {
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

func (self *SClouduser) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := self.removePolicies()
	if err != nil {
		return errors.Wrap(err, "removePolicies")
	}
	err = self.leaveGroups()
	if err != nil {
		return errors.Wrap(err, "leaveGroups")
	}
	return self.SStatusUserResourceBase.Delete(ctx, userCred)
}

func (self *SClouduser) GetCloudpolicy(policyId string) (*SCloudpolicy, error) {
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

func (self *SClouduser) AllowPerformSetPolicies(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "set-policies")
}

// 设置用户权限列表(全量覆盖)
// 用户状态必须为: available
func (self *SClouduser) PerformSetPolicies(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserSetPoliciesInput) (jsonutils.JSONObject, error) {
	if self.Status != api.CLOUD_USER_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not set policies in status %s", self.Status)
	}

	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	factory, err := account.GetProviderFactory()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if !factory.IsSupportClouduserPolicy() {
		return nil, httperrors.NewUnsupportOperationError("Unsupport operation for %s user", account.Provider)
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
		if policy.Provider != account.Provider {
			return nil, httperrors.NewConflictError("policy %s(%s) and group not with same provider", policy.Name, policy.Id)
		}
		newP.Add(policyId)
	}

	for _, del := range set.Difference(local, newP).List() {
		id := del.(string)
		policy, ok := policyMaps[id]
		if ok {
			err = self.detachPolicy(policy.Id)
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
				return nil, httperrors.NewGeneralError(errors.Wrap(err, "AttachPolicy"))
			}
			logclient.AddSimpleActionLog(self, logclient.ACT_ATTACH_POLICY, policy, userCred, true)
		}
	}

	return nil, self.StartClouduserSyncPoliciesTask(ctx, userCred, "")
}

func (self *SClouduser) AllowPerformSetGroups(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "set-groups")
}

// 设置用户权限组列表(全量覆盖)
// 用户状态必须为: available
func (self *SClouduser) PerformSetGroups(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserSetGroupsInput) (jsonutils.JSONObject, error) {
	if self.Status != api.CLOUD_USER_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not set groups in status %s", self.Status)
	}

	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetCloudaccount"))
	}

	groups, err := self.GetCloudgroups()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetCloudgroups"))
	}

	local := set.New(set.ThreadSafe)
	groupMaps := map[string]*SCloudgroup{}
	for i := range groups {
		local.Add(groups[i].Id)
		groupMaps[groups[i].Id] = &groups[i]
	}

	newG := set.New(set.ThreadSafe)
	for _, groupId := range input.CloudgroupIds {
		_group, err := CloudgroupManager.FetchById(groupId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudgroup", groupId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		group := _group.(*SCloudgroup)
		if group.Provider != account.Provider {
			return nil, httperrors.NewConflictError("group %s(%s) and user not with same provider", group.Name, group.Id)
		}
		newG.Add(groupId)
		groupMaps[group.Id] = group
	}
	for _, del := range set.Difference(local, newG).List() {
		id := del.(string)
		group, ok := groupMaps[id]
		if ok {
			err = self.leaveGroup(id)
			if err != nil {
				return nil, httperrors.NewGeneralError(errors.Wrapf(err, "leaveGroup(%s)", group.Name))
			}
			logclient.AddSimpleActionLog(self, logclient.ACT_REMOVE_USER, group, userCred, true)
			logclient.AddSimpleActionLog(group, logclient.ACT_REMOVE_USER, self, userCred, true)
		}
	}

	for _, add := range set.Difference(newG, local).List() {
		id := add.(string)
		group, ok := groupMaps[id]
		if ok {
			err = self.joinGroup(id)
			if err != nil {
				return nil, httperrors.NewGeneralError(errors.Wrapf(err, "%s.AddUser", group.Name))
			}
			logclient.AddSimpleActionLog(self, logclient.ACT_ADD_USER, group, userCred, true)
			logclient.AddSimpleActionLog(group, logclient.ACT_ADD_USER, self, userCred, true)
		}
	}

	return nil, self.StartClouduserSyncGroupsTask(ctx, userCred, "")
}

func (self *SClouduser) AllowPerformJoinGroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "join-group")
}

// 将用户加入权限组
// 用户状态必须为: available
func (self *SClouduser) PerformJoinGroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserJoinGroupInput) (jsonutils.JSONObject, error) {
	if self.Status != api.CLOUD_USER_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not join group in status %s", self.Status)
	}
	if len(input.CloudgroupId) == 0 {
		return nil, httperrors.NewMissingParameterError("cloudgroup_id")
	}
	group, err := CloudgroupManager.FetchById(input.CloudgroupId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2("cloudgroup", input.CloudgroupId)
		}
		return nil, httperrors.NewGeneralError(err)
	}

	err = self.joinGroup(group.GetId())
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	logclient.AddSimpleActionLog(group, logclient.ACT_ADD_USER, self, userCred, true)
	logclient.AddSimpleActionLog(self, logclient.ACT_ADD_USER, group, userCred, true)
	return nil, self.StartClouduserSyncGroupsTask(ctx, userCred, "")
}

func (self *SClouduser) AllowPerformLeaveGroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "leave-group")
}

// 将用户从权限组中移除
// 用户状态必须为: available
func (self *SClouduser) PerformLeaveGroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserLeaveGroupInput) (jsonutils.JSONObject, error) {
	if self.Status != api.CLOUD_USER_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not leave group in status %s", self.Status)
	}
	if len(input.CloudgroupId) == 0 {
		return nil, httperrors.NewMissingParameterError("cloudgroup_id")
	}
	group, err := CloudgroupManager.FetchById(input.CloudgroupId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2("cloudgroup", input.CloudgroupId)
		}
		return nil, httperrors.NewGeneralError(err)
	}

	err = self.leaveGroup(group.GetId())
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	logclient.AddSimpleActionLog(group, logclient.ACT_REMOVE_USER, self, userCred, true)
	logclient.AddSimpleActionLog(self, logclient.ACT_REMOVE_USER, group, userCred, true)
	return nil, self.StartClouduserSyncGroupsTask(ctx, userCred, "")
}

func (self *SClouduser) AllowPerformAttachPolicy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "attach-policy")
}

// 绑定用户权限
// 用户状态必须为: available
func (self *SClouduser) PerformAttachPolicy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserAttachPolicyInput) (jsonutils.JSONObject, error) {
	if self.Status != api.CLOUD_USER_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not detach policy in status %s", self.Status)
	}
	if len(input.CloudpolicyId) == 0 {
		return nil, httperrors.NewMissingParameterError("cloudpolicy_id")
	}

	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetCloudaccount"))
	}

	factory, err := account.GetProviderFactory()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetProviderFactory"))
	}

	if !factory.IsSupportClouduserPolicy() {
		return nil, httperrors.NewNotSupportedError("%s not support attach policy for user", account.Provider)
	}

	_policy, err := CloudpolicyManager.FetchById(input.CloudpolicyId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2("cloudpolicy", input.CloudpolicyId)
		}
		return nil, httperrors.NewGeneralError(err)
	}

	policy := _policy.(*SCloudpolicy)
	if policy.Provider != account.Provider {
		return nil, httperrors.NewDuplicateResourceError("user and policy not with same provider")
	}
	_, err = self.GetCloudpolicy(input.CloudpolicyId)
	if err == nil || errors.Cause(err) == sqlchemy.ErrDuplicateEntry {
		return nil, httperrors.NewDuplicateResourceError("policy %s has aleady binding this user", input.CloudpolicyId)
	}

	err = self.attachPolicy(input.CloudpolicyId)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "AttachPolicy"))
	}

	logclient.AddSimpleActionLog(self, logclient.ACT_ATTACH_POLICY, policy, userCred, true)
	return nil, self.StartClouduserSyncPoliciesTask(ctx, userCred, "")
}

func (self *SClouduser) attachPolicy(policyId string) error {
	up := &SClouduserPolicy{}
	up.SetModelManager(ClouduserPolicyManager, up)
	up.CloudpolicyId = policyId
	up.ClouduserId = self.Id
	return ClouduserPolicyManager.TableSpec().Insert(context.Background(), up)
}

func (self *SClouduser) AllowPerformDetachPolicy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "detach-policy")
}

// 解绑用户权限
// 用户状态必须为: available
func (self *SClouduser) PerformDetachPolicy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserDetachPolicyInput) (jsonutils.JSONObject, error) {
	if self.Status != api.CLOUD_USER_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not detach policy in status %s", self.Status)
	}
	if len(input.CloudpolicyId) == 0 {
		return nil, httperrors.NewMissingParameterError("cloudpolicy_id")
	}
	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetCloudaccount"))
	}

	factory, err := account.GetProviderFactory()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetProviderFactory"))
	}

	if !factory.IsSupportClouduserPolicy() {
		return nil, httperrors.NewUnsupportOperationError("Unsupport operation for %s user", account.Provider)
	}

	policyCount, _ := self.GetCloudpolicyCount()
	if minPolicyCount := factory.GetClouduserMinPolicyCount(); minPolicyCount > 0 && minPolicyCount >= policyCount {
		return nil, httperrors.NewUnsupportOperationError("%s that at least %d cloudpolicy be retained", account.Provider, minPolicyCount)
	}

	policy, err := CloudpolicyManager.FetchById(input.CloudpolicyId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2("cloudpolicy", input.CloudpolicyId)
		}
		return nil, httperrors.NewGeneralError(err)
	}

	_, err = self.GetCloudpolicy(input.CloudpolicyId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}
		return nil, httperrors.NewGeneralError(err)
	}

	err = self.detachPolicy(input.CloudpolicyId)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "detachPolicy"))
	}

	logclient.AddSimpleActionLog(self, logclient.ACT_DETACH_POLICY, policy, userCred, true)
	return nil, self.StartClouduserSyncPoliciesTask(ctx, userCred, "")
}

func (self *SClouduser) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "syncstatus")
}

// 同步用户状态
func (self *SClouduser) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserSyncstatusInput) (jsonutils.JSONObject, error) {
	if len(self.ExternalId) == 0 {
		return nil, httperrors.NewGeneralError(fmt.Errorf("not managed resource"))
	}
	return nil, self.StartClouduserSyncstatusTask(ctx, userCred, "")
}

func (self *SClouduser) StartClouduserSyncstatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClouduserSyncstatusTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.CLOUD_USER_STATUS_SYNC_STATUS, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SClouduser) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "sync")
}

// 同步用户权限和权限组到云上
// 用户状态必须为: available
func (self *SClouduser) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserSyncInput) (jsonutils.JSONObject, error) {
	if self.Status != api.CLOUD_USER_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not sync in status %s", self.Status)
	}
	return nil, self.StartClouduserSyncTask(ctx, userCred, "")
}

func (self *SClouduser) StartClouduserSyncTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClouduserSyncTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.CLOUD_USER_STATUS_SYNC, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SClouduser) StartClouduserSyncPoliciesTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClouduserSyncPoliciesTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.CLOUD_USER_STATUS_SYNC_POLICIES, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SClouduser) StartClouduserSyncGroupsTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClouduserSyncGroupsTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.CLOUD_USER_STATUS_SYNC_GROUPS, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SClouduser) AllowPerformResetPassword(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "reset-password")
}

// 重置用户密码
// 用户状态必须为: available
func (self *SClouduser) PerformResetPassword(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserResetPasswordInput) (jsonutils.JSONObject, error) {
	if self.Status != api.CLOUD_USER_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not reset password in status %s", self.Status)
	}

	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetCloudaccount"))
	}
	factory, err := account.GetProviderFactory()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetProviderFactory"))
	}

	if !factory.IsSupportResetClouduserPassword() {
		return nil, httperrors.NewUnsupportOperationError("Not support reset %s user password", account.Provider)
	}

	return nil, self.StartClouduserResetPasswordTask(ctx, userCred, input.Password, "")
}

func (self *SClouduser) StartClouduserResetPasswordTask(ctx context.Context, userCred mcclient.TokenCredential, password string, parentTaskId string) error {
	if len(password) == 0 {
		password = seclib2.RandomPassword2(12)
	}
	params := jsonutils.Marshal(map[string]string{"password": password}).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "ClouduserResetPasswordTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.CLOUD_USER_STATUS_RESET_PASSWORD, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SClouduser) AllowPerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserChangeOwnerInput) bool {
	return db.IsDomainAllowPerform(userCred, self, "change-owner")
}

// 变更子账号所属本地用户
func (self *SClouduser) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserChangeOwnerInput) (jsonutils.JSONObject, error) {
	user, err := db.UserCacheManager.FetchById(input.UserId)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "Not found user %s", input.UserId))
	}
	old := self.OwnerId
	_, err = db.Update(self, func() error {
		self.OwnerId = user.GetId()
		return nil
	})

	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	logclient.AddSimpleActionLog(self, logclient.ACT_CHANGE_OWNER, map[string]interface{}{"old": old, "new": user}, userCred, true)
	return nil, self.StartClouduserResetPasswordTask(ctx, userCred, "", "")
}

func (self *SClouduser) GetCloudgroupQuery() *sqlchemy.SQuery {
	sq := CloudgroupUserManager.Query("cloudgroup_id").Equals("clouduser_id", self.Id)
	return CloudgroupManager.Query().In("id", sq.SubQuery())
}

func (self *SClouduser) GetCloudgroupCount() (int, error) {
	return self.GetCloudgroupQuery().CountWithError()
}

func (self *SClouduser) GetCloudgroups() ([]SCloudgroup, error) {
	q := self.GetCloudgroupQuery()
	groups := []SCloudgroup{}
	err := db.FetchModelObjects(CloudgroupManager, q, &groups)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return groups, nil
}

// 将本地的权限推送到云上(覆盖云上设置)
func (self *SClouduser) SyncCloudpoliciesForCloud(ctx context.Context) (result compare.SyncResult, err error) {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	iUser, err := self.GetIClouduser()
	if err != nil {
		return result, errors.Wrap(err, "GetIClouduser")
	}
	iPolicies, err := iUser.GetISystemCloudpolicies()
	if err != nil {
		return result, errors.Wrap(err, "GetISystemCloudpolicies")
	}

	dbPolicies, err := self.GetCloudpolicies()
	if err != nil {
		return result, errors.Wrap(err, "GetCloudpolicies")
	}

	added := make([]SCloudpolicy, 0)
	commondb := make([]SCloudpolicy, 0)
	commonext := make([]cloudprovider.ICloudpolicy, 0)
	removed := make([]cloudprovider.ICloudpolicy, 0)

	err = compare.CompareSets(dbPolicies, iPolicies, &added, &commondb, &commonext, &removed)
	if err != nil {
		return result, errors.Wrap(err, "compare.CompareSets")
	}

	for i := 0; i < len(removed); i++ {
		err = iUser.DetachSystemPolicy(removed[i].GetGlobalId())
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	result.UpdateCnt = len(commondb)

	for i := 0; i < len(added); i++ {
		err = iUser.AttachSystemPolicy(added[i].ExternalId)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result, nil
}
