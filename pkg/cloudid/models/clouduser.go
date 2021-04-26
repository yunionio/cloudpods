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
	"time"

	"gopkg.in/fatih/set.v0"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	npk "yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SClouduserManager struct {
	db.SStatusDomainLevelUserResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudaccountResourceBaseManager
}

var ClouduserManager *SClouduserManager

func init() {
	ClouduserManager = &SClouduserManager{
		SStatusDomainLevelUserResourceBaseManager: db.NewStatusDomainLevelUserResourceBaseManager(
			SClouduser{},
			"cloudusers_tbl",
			"clouduser",
			"cloudusers",
		),
	}
	ClouduserManager.SetVirtualObject(ClouduserManager)
}

type SClouduser struct {
	db.SStatusDomainLevelUserResourceBase
	db.SExternalizedResourceBase
	SCloudaccountResourceBase

	Secret string `length:"0" charset:"ascii" nullable:"true" list:"user" create:"domain_optional"`
	// 是否可以控制台登录
	IsConsoleLogin tristate.TriState `nullable:"false" default:"false" list:"user" create:"optional"`
	// 手机号码
	MobilePhone string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"domain_optional"`
	// 邮箱地址
	Email string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"domain_optional"`
}

func (manager *SClouduserManager) EnableGenerateName() bool {
	return false
}

func (manager *SClouduserManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	q := manager.Query()
	domainCnt, err := db.CalculateResourceCount(q, "domain_id")
	if err != nil {
		return nil, errors.Wrap(err, "CalculateResourceCount.domain_id")
	}
	q = manager.Query()
	userCnt, err := db.CalculateResourceCount(q, "owner_id")
	if err != nil {
		return nil, errors.Wrap(err, "CalculateResourceCount.owner_id")
	}
	return append(domainCnt, userCnt...), nil
}

func (manager *SClouduserManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (manager *SClouduserManager) GetIVirtualModelManager() db.IVirtualModelManager {
	return manager.GetVirtualObject().(db.IVirtualModelManager)
}

func (manager *SClouduserManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	accountId, _ := data.GetString("cloudaccount_id")
	return jsonutils.Marshal(map[string]string{"cloudaccount_id": accountId})
}

func (manager *SClouduserManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	accountId, _ := values.GetString("cloudaccount_id")
	if len(accountId) > 0 {
		q = q.Equals("cloudaccount_id", accountId)
	}
	return q
}

// 公有云用户列表
func (manager *SClouduserManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.ClouduserListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusDomainLevelUserResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusDomainLevelUserResourceListInput)
	if err != nil {
		return nil, err
	}
	time.Sleep(time.Minute)

	q, err = manager.SCloudaccountResourceBaseManager.ListItemFilter(ctx, q, userCred, query.CloudaccountResourceListInput)
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
	q, err = manager.SStatusDomainLevelUserResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusDomainLevelUserResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusDomainLevelUserResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SClouduserManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusDomainLevelUserResourceBaseManager.QueryDistinctExtraField(q, field)
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
	userRows := manager.SStatusDomainLevelUserResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	acRows := manager.SCloudaccountResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	userIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.ClouduserDetails{
			StatusDomainLevelUserResourceDetails: userRows[i],
			CloudaccountResourceDetails:          acRows[i],
		}
		user := objs[i].(*SClouduser)
		userIds[i] = user.Id
	}
	q := ClouduserPolicyManager.Query().In("clouduser_id", userIds)
	ups := []SClouduserPolicy{}
	err := db.FetchModelObjects(ClouduserPolicyManager, q, &ups)
	if err != nil {
		log.Errorf("db.FetchModelObjects error: %v", err)
		return rows
	}
	upMaps := map[string][]SClouduserPolicy{}
	providerIds := []string{}
	policyIds := []string{}
	for _, up := range ups {
		if len(up.CloudproviderId) > 0 {
			providerIds = append(providerIds, up.CloudproviderId)
		}
		_, ok := upMaps[up.ClouduserId]
		if !ok {
			upMaps[up.ClouduserId] = []SClouduserPolicy{}
		}
		upMaps[up.ClouduserId] = append(upMaps[up.ClouduserId], up)
		policyIds = append(policyIds, up.CloudpolicyId)
	}
	providerMaps, err := db.FetchIdNameMap2(CloudproviderManager, providerIds)
	if err != nil {
		log.Errorf("db.FetchIdNameMap2 error: %v", err)
		return rows
	}
	policyMaps, err := db.FetchIdNameMap2(CloudpolicyManager, policyIds)
	if err != nil {
		log.Errorf("db.FetchIdNameMap2 error: %v", err)
		return rows
	}
	q = CloudgroupUserManager.Query().In("clouduser_id", userIds)
	userGroups := []SCloudgroupUser{}
	err = db.FetchModelObjects(CloudgroupUserManager, q, &userGroups)
	if err != nil {
		log.Errorf("db.FetchModelObjects error: %v", err)
		return rows
	}
	groupIds := []string{}
	userGroupMaps := map[string]map[string]bool{}
	for _, ug := range userGroups {
		groupIds = append(groupIds, ug.CloudgroupId)
		_, ok := userGroupMaps[ug.ClouduserId]
		if !ok {
			userGroupMaps[ug.ClouduserId] = map[string]bool{}
		}
		userGroupMaps[ug.ClouduserId][ug.CloudgroupId] = true
	}
	groupMaps, err := db.FetchIdNameMap2(CloudgroupManager, groupIds)
	if err != nil {
		log.Errorf("db.FetchIdNameMap2 error: %v", err)
		return rows
	}
	for i := range rows {
		ups, ok := upMaps[userIds[i]]
		if ok {
			rows[i].Cloudpolicies = []api.ClouduserpolicyDetails{}
			for _, up := range ups {
				item := api.ClouduserpolicyDetails{
					CloudproviderId: up.CloudproviderId,
					Id:              up.CloudpolicyId,
				}
				if len(up.CloudproviderId) > 0 {
					item.Manager, _ = providerMaps[up.CloudproviderId]
				}
				item.Name, _ = policyMaps[up.CloudpolicyId]
				rows[i].Cloudpolicies = append(rows[i].Cloudpolicies, item)
			}
			rows[i].CloudpolicyCount = len(rows[i].Cloudpolicies)
		}
		ugrps, ok := userGroupMaps[userIds[i]]
		if ok {
			rows[i].Cloudgroups = []api.SCloudIdBaseResource{}
			for groupId := range ugrps {
				item := api.SCloudIdBaseResource{
					Id: groupId,
				}
				item.Name, _ = groupMaps[groupId]
				rows[i].Cloudgroups = append(rows[i].Cloudgroups, item)
			}
			rows[i].CloudgroupCount = len(rows[i].Cloudgroups)
		}
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

// 创建公有云用户
func (manager *SClouduserManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ClouduserCreateInput) (api.ClouduserCreateInput, error) {
	var provider *SCloudprovider = nil
	if len(input.CloudproviderId) > 0 {
		var err error
		provider, err = CloudproviderManager.FetchProvider(ctx, input.CloudproviderId)
		if err != nil {
			return input, httperrors.NewGeneralError(errors.Wrap(err, "CloudproviderManager.FetchProvider"))
		}
		input.CloudaccountId = provider.CloudaccountId
	}
	if len(input.CloudaccountId) == 0 {
		return input, httperrors.NewMissingParameterError("cloudaccount_id")
	}
	account, err := CloudaccountManager.FetchAccount(ctx, input.CloudaccountId)
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrap(err, "FetchAccount"))
	}
	input.CloudaccountId = account.Id
	// 只有系统管理员和账号所在的域管理员可以创建子用户
	if !((account.DomainId == userCred.GetProjectDomainId() && db.IsDomainAllowCreate(userCred, manager)) || userCred.HasSystemAdminPrivilege()) {
		return input, httperrors.NewForbiddenError("forbidden to create clouduser for cloudaccount %s", account.Name)
	}
	delegate, err := account.getCloudDelegate(ctx)
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrap(err, "getCloudaccountDelegate"))
	}
	factory, err := account.GetProviderFactory()
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrap(err, "GetProviderFactory"))
	}
	if !factory.IsSupportCloudIdService() {
		return input, httperrors.NewUnsupportOperationError("Not support clouduser for provider %s", delegate.Provider)
	}
	input.StatusDomainLevelUserResourceCreateInput, err = manager.SStatusDomainLevelUserResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusDomainLevelUserResourceCreateInput)
	if err != nil {
		return input, err
	}

	if len(input.Email) > 0 && !regutils.MatchEmail(input.Email) {
		return input, httperrors.NewInputParameterError("invalid email address")
	}

	if len(input.OwnerId) > 0 {
		user, err := db.UserCacheManager.FetchUserById(ctx, input.OwnerId)
		if err != nil {
			return input, errors.Wrap(err, "FetchUserById")
		}
		input.OwnerId = user.Id
		if len(input.Name) == 0 {
			input.Name = user.Name
		}
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
		err = seclib2.ValidatePassword(input.Password)
		if err != nil {
			return input, err
		}
	}

	var iProvider cloudprovider.ICloudProvider = nil
	if factory.IsClouduserpolicyWithSubscription() && provider != nil {
		iProvider, err = provider.GetProvider()
	} else {
		iProvider, err = account.GetProvider()
	}
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrapf(err, "GetProvider"))
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

	iUser, err := iProvider.CreateIClouduser(&conf)
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrap(err, "p.CreateIClouduser"))
	}
	input.ExternalId = iUser.GetGlobalId()
	input.Name = iUser.GetName()

	input.ProjectDomainId = account.DomainId
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
	db.Update(self, func() error {
		self.Source = apis.EXTERNAL_RESOURCE_SOURCE_LOCAL
		return nil
	})
	account, err := self.GetCloudaccount()
	if err != nil {
		return
	}
	factory, err := account.GetProviderFactory()
	if err != nil {
		return
	}
	providerIds := []string{}
	if !factory.IsClouduserpolicyWithSubscription() {
		providerIds = []string{""}
	} else if len(input.CloudproviderId) > 0 {
		providerIds = append(providerIds, input.CloudproviderId)
	} else {
		providers, _ := account.GetCloudproviders()
		for _, provider := range providers {
			providerIds = append(providerIds, provider.Id)
		}
	}
	for _, providerId := range providerIds {
		for _, policyId := range input.CloudpolicyIds {
			self.attachPolicy(policyId, providerId)
		}
	}
	for _, groupId := range input.CloudgroupIds {
		self.joinGroup(groupId)
	}
	if len(self.Email) > 0 && input.Notify {
		msg := struct {
			Account     string
			Name        string
			Password    string
			IamLoginUrl string
			Id          string
		}{
			Id:          self.Id,
			Password:    input.Password,
			IamLoginUrl: account.IamLoginUrl,
		}
		msg.Account, msg.Name = account.GetClouduserAccountName(self.Name)
		notifyclient.NotifyWithContact(ctx, []string{self.Email}, npk.NotifyByEmail, npk.NotifyPriorityNormal, "CLOUD_USER_CREATED", jsonutils.Marshal(msg))
	}
	self.StartClouduserSyncTask(ctx, userCred, "")
}

func (self *SClouduser) SyncWithClouduser(ctx context.Context, userCred mcclient.TokenCredential, iUser cloudprovider.IClouduser) error {
	_, err := db.Update(self, func() error {
		self.Name = iUser.GetName()
		self.Status = api.CLOUD_USER_STATUS_AVAILABLE
		switch iUser.IsConsoleLogin() {
		case true:
			self.IsConsoleLogin = tristate.True
		case false:
			self.IsConsoleLogin = tristate.False
		}
		account, err := self.GetCloudaccount()
		if err != nil {
			return errors.Wrap(err, "GetCloudaccount")
		}
		self.DomainId = account.DomainId
		return nil
	})
	return err
}

func (self *SClouduser) detachPolicies(policyId string, providerId string) error {
	policies := []SClouduserPolicy{}
	q := ClouduserPolicyManager.Query().Equals("clouduser_id", self.Id).Equals("cloudpolicy_id", policyId)
	if len(providerId) > 0 {
		q = q.Equals("cloudprovider_id", providerId)
	}
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

func (self *SClouduser) SyncSystemCloudpolicies(ctx context.Context, userCred mcclient.TokenCredential, iPolicies []cloudprovider.ICloudpolicy, cloudproviderId string) compare.SyncResult {
	result := compare.SyncResult{}
	dbPolicies, err := self.GetSystemCloudpolicies(cloudproviderId)
	if err != nil {
		result.Error(errors.Wrap(err, "GetSystemClouduserPolicies"))
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
		err = self.newSystemClouduserPolicy(ctx, userCred, added[i], cloudproviderId)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (self *SClouduser) newSystemClouduserPolicy(ctx context.Context, userCred mcclient.TokenCredential, iPolicy cloudprovider.ICloudpolicy, cloudproviderId string) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	account, err := self.GetCloudaccount()
	if err != nil {
		return errors.Wrap(err, "GetCloudaccount")
	}
	up := &SClouduserPolicy{}
	up.SetModelManager(ClouduserPolicyManager, up)
	up.ClouduserId = self.Id
	up.CloudproviderId = cloudproviderId
	policy, err := db.FetchByExternalIdAndManagerId(CloudpolicyManager, iPolicy.GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("provider", account.Provider)
	})
	if err != nil {
		return errors.Wrapf(err, "db.FetchByExternalId(%s)", iPolicy.GetGlobalId())
	}
	up.CloudpolicyId = policy.GetId()
	return ClouduserPolicyManager.TableSpec().Insert(ctx, up)
}

func (self *SClouduser) SyncCustomCloudpolicies(ctx context.Context, userCred mcclient.TokenCredential, iPolicies []cloudprovider.ICloudpolicy, cloudproviderId string) compare.SyncResult {
	result := compare.SyncResult{}
	dbPolicies, err := self.GetCustomCloudpolicycaches(cloudproviderId)
	if err != nil {
		result.Error(errors.Wrap(err, "GetCustomCloudpolicycaches"))
		return result
	}

	removed := make([]SCloudpolicycache, 0)
	commondb := make([]SCloudpolicycache, 0)
	commonext := make([]cloudprovider.ICloudpolicy, 0)
	added := make([]cloudprovider.ICloudpolicy, 0)

	err = compare.CompareSets(dbPolicies, iPolicies, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrap(err, "compare.CompareSets"))
		return result
	}

	result.UpdateCnt = len(commondb)

	for i := 0; i < len(added); i++ {
		err = self.newCustomClouduserPolicy(ctx, userCred, added[i], cloudproviderId)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (self *SClouduser) newCustomClouduserPolicy(ctx context.Context, userCred mcclient.TokenCredential, iPolicy cloudprovider.ICloudpolicy, cloudproviderId string) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	account, err := self.GetCloudaccount()
	if err != nil {
		return errors.Wrap(err, "GetCloudaccount")
	}
	factory, err := account.GetProviderFactory()
	if err != nil {
		return errors.Wrap(err, "GetProviderFactory")
	}
	up := &SClouduserPolicy{}
	up.SetModelManager(ClouduserPolicyManager, up)
	up.ClouduserId = self.Id
	up.CloudproviderId = cloudproviderId
	cache, err := db.FetchByExternalIdAndManagerId(CloudpolicycacheManager, iPolicy.GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		q = q.Equals("cloudaccount_id", account.Id)
		if factory.IsCloudpolicyWithSubscription() && len(cloudproviderId) > 0 {
			q = q.Equals("cloudprovider_id", cloudproviderId)
		}
		return q
	})
	if err != nil {
		return errors.Wrapf(err, "db.FetchByExternalId(%s)", iPolicy.GetGlobalId())
	}
	policy := cache.(*SCloudpolicycache)
	up.CloudpolicyId = policy.CloudpolicyId
	return ClouduserPolicyManager.TableSpec().Insert(ctx, up)

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
		var cloudgroupId string
		_cache, err := db.FetchByExternalIdAndManagerId(CloudgroupcacheManager, added[i].GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("cloudaccount_id", self.CloudaccountId)
		})
		if err != nil {
			if errors.Cause(err) != sql.ErrNoRows {
				result.AddError(errors.Wrapf(err, "FetchByExternalId(%s)", added[i].GetGlobalId()))
				continue
			}
			account, err := self.GetCloudaccount()
			if err != nil {
				result.AddError(errors.Wrap(err, "GetCloudaccount"))
				continue
			}
			cache, err := account.newCloudgroup(ctx, userCred, added[i])
			if err != nil {
				result.AddError(errors.Wrap(err, "account.newCloudgroup"))
				continue
			}
			cloudgroupId = cache.CloudgroupId
		} else {
			cache := _cache.(*SCloudgroupcache)
			cloudgroupId = cache.CloudgroupId
		}
		err = self.joinGroup(cloudgroupId)
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

func (self *SClouduser) GetCloudpolicyQuery(providerId string) *sqlchemy.SQuery {
	sq := ClouduserPolicyManager.Query("cloudpolicy_id").Equals("clouduser_id", self.Id)
	if len(providerId) > 0 {
		sq = sq.Equals("cloudprovider_id", providerId)
	}
	return CloudpolicyManager.Query().In("id", sq.SubQuery())
}

func (self *SClouduser) GetCloudpolicyCount() (int, error) {
	return self.GetCloudpolicyQuery("").CountWithError()
}

func (self *SClouduser) GetCloudpolicies(providerId string) ([]SCloudpolicy, error) {
	policies := []SCloudpolicy{}
	q := self.GetCloudpolicyQuery(providerId)
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SClouduser) GetSystemCloudpolicies(cloudproviderId string) ([]SCloudpolicy, error) {
	policies := []SCloudpolicy{}
	q := self.GetCloudpolicyQuery(cloudproviderId).Equals("policy_type", api.CLOUD_POLICY_TYPE_SYSTEM)
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SClouduser) GetCustomCloudpolicies(cloudproviderId string) ([]SCloudpolicy, error) {
	policies := []SCloudpolicy{}
	q := self.GetCloudpolicyQuery(cloudproviderId).Equals("policy_type", api.CLOUD_POLICY_TYPE_CUSTOM)
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SClouduser) GetCustomCloudpolicycaches(cloudproviderId string) ([]SCloudpolicycache, error) {
	caches := []SCloudpolicycache{}
	q := CloudpolicycacheManager.Query()
	sq := self.GetCloudpolicyQuery(cloudproviderId).Equals("policy_type", api.CLOUD_POLICY_TYPE_CUSTOM).SubQuery()
	q = q.In("cloudpolicy_id", sq.Query(sq.Field("id")).SubQuery())
	err := db.FetchModelObjects(CloudpolicycacheManager, q, &caches)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return caches, nil
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
	policies, err := self.GetCloudpolicies("")
	if err != nil {
		return errors.Wrap(err, "GetCloudpolicies")
	}
	for i := range policies {
		err = self.detachPolicies(policies[i].Id, "")
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
	return self.SStatusDomainLevelResourceBase.Delete(ctx, userCred)
}

func (self *SClouduser) GetCloudpolicy(policyId string, providerId string) ([]SCloudpolicy, error) {
	policies := []SCloudpolicy{}
	q := self.GetCloudpolicyQuery(providerId).Equals("id", policyId)
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return policies, nil
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

	providerIds := []string{}

	if !factory.IsClouduserpolicyWithSubscription() {
		input.CloudproviderId = ""
		providerIds = []string{""}
	} else if len(input.CloudproviderId) > 0 {
		_provider, err := CloudproviderManager.FetchById(input.CloudproviderId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudprovider", input.CloudproviderId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		provider := _provider.(*SCloudprovider)
		if self.CloudaccountId != provider.CloudaccountId {
			return nil, httperrors.NewConflictError("provider %s and user %s not with same account", provider.Name, self.Name)
		}
	} else {
		providers, err := account.GetCloudproviders()
		if err != nil {
			return nil, httperrors.NewGeneralError(errors.Wrapf(err, "account.GetCloudproviders"))
		}
		for _, provider := range providers {
			providerIds = append(providerIds, provider.Id)
		}
	}

	policies, err := self.GetCloudpolicies(input.CloudproviderId)
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
		if policy.Provider != account.Provider {
			return nil, httperrors.NewConflictError("policy %s(%s) and group not with same provider", policy.Name, policy.Id)
		}
		newP.Add(policyId)
	}

	for _, del := range set.Difference(local, newP).List() {
		id := del.(string)
		policy, ok := policyMaps[id]
		if ok {
			err = self.detachPolicies(policy.Id, input.CloudproviderId)
			if err != nil {
				return nil, httperrors.NewGeneralError(errors.Wrap(err, "detachPolicy"))
			}
			notes := map[string]interface{}{"cloudprovider_id": input.CloudproviderId, "policy": policy}
			logclient.AddSimpleActionLog(self, logclient.ACT_DETACH_POLICY, notes, userCred, true)
		}
	}

	for _, add := range set.Difference(newP, local).List() {
		id := add.(string)
		policy, ok := policyMaps[id]
		if ok {
			for _, providerId := range providerIds {
				err = self.attachPolicy(id, providerId)
				if err != nil {
					return nil, httperrors.NewGeneralError(errors.Wrap(err, "AttachPolicy"))
				}
				notes := map[string]interface{}{"cloudprovider_id": providerId, "policy": policy}
				logclient.AddSimpleActionLog(self, logclient.ACT_ATTACH_POLICY, notes, userCred, true)
			}
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
	err = policy.ValidateUse()
	if err != nil {
		return nil, err
	}
	if policy.Provider != account.Provider {
		return nil, httperrors.NewDuplicateResourceError("user and policy not with same provider")
	}

	if !factory.IsClouduserpolicyWithSubscription() {
		input.CloudproviderId = ""
	} else if len(input.CloudproviderId) > 0 {
		_provider, err := CloudproviderManager.FetchById(input.CloudproviderId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudprovider", input.CloudproviderId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		provider := _provider.(*SCloudprovider)
		if provider.CloudaccountId != self.CloudaccountId {
			return nil, httperrors.NewConflictError("provider %s and user %s not with same account", input.CloudproviderId, self.Name)
		}
	}

	policies, _ := self.GetCloudpolicy(input.CloudpolicyId, input.CloudproviderId)
	if len(policies) > 0 {
		return nil, httperrors.NewDuplicateResourceError("policy %s has aleady binding this user", input.CloudpolicyId)
	}

	err = self.attachPolicy(input.CloudpolicyId, input.CloudpolicyId)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "AttachPolicy"))
	}

	logclient.AddSimpleActionLog(self, logclient.ACT_ATTACH_POLICY, policy, userCred, true)
	return nil, self.StartClouduserSyncPoliciesTask(ctx, userCred, "")
}

func (self *SClouduser) attachPolicy(policyId string, providerId string) error {
	up := &SClouduserPolicy{}
	up.SetModelManager(ClouduserPolicyManager, up)
	up.CloudpolicyId = policyId
	up.ClouduserId = self.Id
	up.CloudproviderId = providerId
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

	if !factory.IsClouduserpolicyWithSubscription() {
		input.CloudproviderId = ""
	} else if len(input.CloudproviderId) > 0 {
		_, err := CloudproviderManager.FetchById(input.CloudproviderId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudprovider", input.CloudproviderId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
	}

	policies, err := self.GetCloudpolicy(input.CloudpolicyId, input.CloudproviderId)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if len(policies) == 0 {
		return nil, nil
	}

	err = self.detachPolicies(input.CloudpolicyId, input.CloudproviderId)
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
	oldUserId := self.OwnerId
	newUserId := ""
	if len(input.UserId) > 0 {
		user, err := db.UserCacheManager.FetchUserById(ctx, input.UserId)
		if err != nil {
			return nil, httperrors.NewGeneralError(errors.Wrapf(err, "Not found user %s", input.UserId))
		}
		newUserId = user.Id
	}

	_, err := db.Update(self, func() error {
		self.OwnerId = newUserId
		return nil
	})
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "db.Update"))
	}

	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetCloudaccount"))
	}

	factory, err := account.GetProviderFactory()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetProviderFactory"))
	}

	logclient.AddSimpleActionLog(self, logclient.ACT_CHANGE_OWNER, map[string]interface{}{"old": oldUserId, "newUserId": newUserId}, userCred, true)
	if len(newUserId) == 0 || !factory.IsSupportResetClouduserPassword() {
		return nil, nil
	}

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

func (self *SClouduser) GetCloudgroupPolicies() ([]SCloudpolicy, error) {
	q := self.GetCloudgorupPoliciesQuery("")
	policies := []SCloudpolicy{}
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SClouduser) GetSystemCloudgroupPolicies() ([]SCloudpolicy, error) {
	q := self.GetCloudgorupPoliciesQuery(api.CLOUD_POLICY_TYPE_SYSTEM)
	policies := []SCloudpolicy{}
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SClouduser) GetCustomCloudgroupPolicies() ([]SCloudpolicy, error) {
	q := self.GetCloudgorupPoliciesQuery(api.CLOUD_POLICY_TYPE_CUSTOM)
	policies := []SCloudpolicy{}
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SClouduser) GetCloudgorupPoliciesQuery(policyType string) *sqlchemy.SQuery {
	q := CloudpolicyManager.Query()
	gu := CloudgroupUserManager.Query("cloudgroup_id").Equals("clouduser_id", self.Id)
	gp := CloudgroupPolicyManager.Query("cloudpolicy_id").In("cloudgroup_id", gu.SubQuery())
	q = q.In("id", gp.SubQuery())
	if len(policyType) > 0 {
		q = q.Equals("policy_type", policyType)
	}
	return q
}

// 将本地的权限推送到云上(覆盖云上设置)
func (self *SClouduser) SyncSystemCloudpoliciesForCloud(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	account, err := self.GetCloudaccount()
	if err != nil {
		return errors.Wrap(err, "GetCloudaccount")
	}

	return account.SyncSystemCloudpoliciesForCloud(ctx, userCred, self)
}

func (self *SClouduser) SyncCustomCloudpoliciesForCloud(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	account, err := self.GetCloudaccount()
	if err != nil {
		return errors.Wrap(err, "GetCloudaccount")
	}

	return account.SyncCustomCloudpoliciesForCloud(ctx, userCred, self)
}
