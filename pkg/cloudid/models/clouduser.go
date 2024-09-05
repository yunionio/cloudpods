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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
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
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SClouduserManager struct {
	db.SStatusDomainLevelUserResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudaccountResourceBaseManager
	SCloudproviderResourceBaseManager
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
	SCloudproviderResourceBase

	Secret string `length:"0" charset:"ascii" nullable:"true" list:"user" create:"domain_optional"`
	// 是否可以控制台登录
	IsConsoleLogin tristate.TriState `default:"false" list:"user" create:"optional"`
	// 手机号码
	MobilePhone string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"domain_optional"`
	// 邮箱地址
	Email string `width:"36" charset:"ascii" list:"user" create:"domain_optional"`
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

// 获取公有云用户详情
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
	mRows := manager.SCloudproviderResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	userIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.ClouduserDetails{
			StatusDomainLevelUserResourceDetails: userRows[i],
			CloudaccountResourceDetails:          acRows[i],
			CloudproviderResourceDetails:         mRows[i],
		}
		user := objs[i].(*SClouduser)
		userIds[i] = user.Id
	}

	groupSQ := CloudgroupManager.Query().SubQuery()
	ugQ := CloudgroupUserManager.Query().SubQuery()
	q := groupSQ.Query(
		groupSQ.Field("id"),
		groupSQ.Field("name"),
		ugQ.Field("clouduser_id"),
	).
		Join(ugQ, sqlchemy.Equals(ugQ.Field("cloudgroup_id"), groupSQ.Field("id"))).
		Filter(sqlchemy.In(ugQ.Field("clouduser_id"), userIds))

	baseInfo := []struct {
		Id          string
		Name        string
		ClouduserId string
	}{}
	err := q.All(&baseInfo)
	if err != nil {
		log.Errorf("query user group info error: %v", err)
		return rows
	}

	groups := map[string][]api.SCloudIdBaseResource{}
	for _, group := range baseInfo {
		_, ok := groups[group.ClouduserId]
		if !ok {
			groups[group.ClouduserId] = []api.SCloudIdBaseResource{}
		}
		groups[group.ClouduserId] = append(groups[group.ClouduserId], api.SCloudIdBaseResource{
			Id:   group.Id,
			Name: group.Name,
		})
	}

	policySQ := CloudpolicyManager.Query().SubQuery()
	upQ := ClouduserPolicyManager.Query().SubQuery()
	q = policySQ.Query(
		policySQ.Field("id"),
		policySQ.Field("name"),
		upQ.Field("clouduser_id"),
	).
		Join(upQ, sqlchemy.Equals(upQ.Field("cloudpolicy_id"), policySQ.Field("id"))).
		Filter(sqlchemy.In(upQ.Field("clouduser_id"), userIds))

	baseInfo = []struct {
		Id          string
		Name        string
		ClouduserId string
	}{}

	err = q.All(&baseInfo)
	if err != nil {
		log.Errorf("query user policy info error: %v", err)
		return rows
	}

	policies := map[string][]api.SCloudIdBaseResource{}
	for _, policy := range baseInfo {
		_, ok := policies[policy.ClouduserId]
		if !ok {
			policies[policy.ClouduserId] = []api.SCloudIdBaseResource{}
		}
		policies[policy.ClouduserId] = append(policies[policy.ClouduserId], api.SCloudIdBaseResource{
			Id:   policy.Id,
			Name: policy.Name,
		})
	}

	for i := range rows {
		rows[i].Cloudgroups, _ = groups[userIds[i]]
		rows[i].CloudgroupCount = len(rows[i].Cloudgroups)
		rows[i].Cloudpolicies, _ = policies[userIds[i]]
		rows[i].CloudpolicyCount = len(rows[i].Cloudpolicies)
	}

	return rows
}

func (self *SClouduser) GetProvider() (cloudprovider.ICloudProvider, error) {
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

func (self *SClouduser) GetIClouduser() (cloudprovider.IClouduser, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external_id")
	}
	provider, err := self.GetProvider()
	if err != nil {
		return nil, errors.Wrap(err, "GetProvider")
	}
	return provider.GetIClouduserByName(self.Name)
}

// 创建公有云用户
func (manager *SClouduserManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.ClouduserCreateInput,
) (*api.ClouduserCreateInput, error) {
	if len(input.ManagerId) == 0 {
		return nil, httperrors.NewMissingParameterError("manager_id")
	}
	providerObj, err := validators.ValidateModel(ctx, userCred, CloudproviderManager, &input.ManagerId)
	if err != nil {
		return nil, err
	}
	provider := providerObj.(*SCloudprovider)
	input.CloudaccountId = provider.CloudaccountId
	input.Status = apis.STATUS_CREATING

	account, err := provider.GetCloudaccount()
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrap(err, "GetCloudaccount"))
	}
	input.ProjectDomainId = account.DomainId
	// 只有系统管理员和账号所在的域管理员可以创建子用户
	if !((account.DomainId == userCred.GetProjectDomainId() && db.IsDomainAllowCreate(userCred, manager).Result.IsAllow()) || userCred.HasSystemAdminPrivilege()) {
		return input, httperrors.NewForbiddenError("forbidden to create clouduser for cloudaccount %s", account.Name)
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

	driver, err := provider.GetDriver()
	if err != nil {
		return nil, err
	}

	input, err = driver.ValidateCreateClouduser(ctx, userCred, provider, input)
	if err != nil {
		return nil, err
	}

	isConsoleLogin := true
	if input.IsConsoleLogin != nil && !*input.IsConsoleLogin {
		isConsoleLogin = false
	}
	input.IsConsoleLogin = &isConsoleLogin

	if isConsoleLogin && len(input.Password) == 0 {
		input.Password = seclib2.RandomPassword2(12)
	}

	if len(input.Password) > 0 {
		err = seclib2.ValidatePassword(input.Password)
		if err != nil {
			return input, err
		}
	}

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
	self.StartClouduserCreateTask(ctx, userCred, input.Notify, "")
}

func (self *SClouduser) GetCloudprovider() (*SCloudprovider, error) {
	provider, err := CloudproviderManager.FetchById(self.ManagerId)
	if err != nil {
		return nil, err
	}
	return provider.(*SCloudprovider), nil
}

func (self *SClouduser) StartClouduserCreateTask(ctx context.Context, userCred mcclient.TokenCredential, notify bool, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Set("notify", jsonutils.NewBool(notify))
	task, err := taskman.TaskManager.NewTask(ctx, "ClouduserCreateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_CREATING, "")
	return task.ScheduleRun(nil)
}

func (self *SClouduser) SyncWithClouduser(ctx context.Context, userCred mcclient.TokenCredential, iUser cloudprovider.IClouduser) error {
	_, err := db.Update(self, func() error {
		self.Name = iUser.GetName()
		self.Status = apis.STATUS_AVAILABLE
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

func (self *SClouduser) SyncCloudpolicies(ctx context.Context, userCred mcclient.TokenCredential, iUser cloudprovider.IClouduser) {
	iPolicies, err := iUser.GetICloudpolicies()
	if err == nil {
		result := self.SyncPolicies(ctx, userCred, iPolicies)
		log.Infof("SyncCloudpolicies for user %s(%s) result: %s", self.Name, self.Id, result.Result())
	}
}

func (self *SClouduser) SyncCloudgroups(ctx context.Context, userCred mcclient.TokenCredential, iUser cloudprovider.IClouduser) {
	iGroups, err := iUser.GetICloudgroups()
	if err == nil {
		result := self.SyncGroups(ctx, userCred, iGroups)
		log.Infof("SyncCloudgroups for user %s(%s) result: %s", self.Name, self.Id, result.Result())
	}
}

func (self *SClouduser) SyncPolicies(ctx context.Context, userCred mcclient.TokenCredential, iPolicies []cloudprovider.ICloudpolicy) compare.SyncResult {
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
			result.AddError(errors.Wrapf(err, "add %s", added[i].GetName()))
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

func (self *SClouduser) SyncGroups(ctx context.Context, userCred mcclient.TokenCredential, iGroups []cloudprovider.ICloudgroup) compare.SyncResult {
	result := compare.SyncResult{}
	dbGroups, err := self.GetCloudgroups()
	if err != nil {
		result.Error(errors.Wrap(err, "GetCloudgroupcaches"))
		return result
	}

	removed := make([]SCloudgroup, 0)
	commondb := make([]SCloudgroup, 0)
	commonext := make([]cloudprovider.ICloudgroup, 0)
	added := make([]cloudprovider.ICloudgroup, 0)

	err = compare.CompareSets(dbGroups, iGroups, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrap(err, "compare.CompareSets"))
		return result
	}

	result.UpdateCnt = len(commondb)

	for i := 0; i < len(added); i++ {
		group, err := db.FetchByExternalIdAndManagerId(CloudgroupManager, added[i].GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			if len(self.ManagerId) > 0 {
				return q.Equals("manager_id", self.ManagerId)
			}
			return q.Equals("cloudaccount_id", self.CloudaccountId)
		})
		if err != nil {
			result.AddError(errors.Wrapf(err, "FetchByExternalId(%s)", added[i].GetGlobalId()))
			continue
		}
		err = self.joinGroup(group.GetId())
		if err != nil {
			result.AddError(errors.Wrap(err, "joinGroup"))
			continue
		}
		result.Add()
	}

	for i := 0; i < len(removed); i++ {
		err = self.leaveGroup(removed[i].Id)
		if err != nil {
			result.DeleteError(errors.Wrap(err, "leaveGroup"))
			continue
		}
		result.Delete()
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
	self.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
	return task.ScheduleRun(nil)
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
	sq := ClouduserPolicyManager.Query("cloudpolicy_id").Equals("clouduser_id", self.Id)
	return CloudpolicyManager.Query().In("id", sq.SubQuery())
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

func (self *SClouduser) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.purge(ctx)
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

// 设置用户权限列表(全量覆盖)
// 用户状态必须为: available
func (self *SClouduser) PerformSetPolicies(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserSetPoliciesInput) (jsonutils.JSONObject, error) {
	if self.Status != apis.STATUS_AVAILABLE {
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

	for i := range input.CloudpolicyIds {
		policyObj, err := validators.ValidateModel(ctx, userCred, CloudpolicyManager, &input.CloudpolicyIds[i])
		if err != nil {
			return nil, err
		}
		policy := policyObj.(*SCloudpolicy)
		if policy.ManagerId != self.ManagerId || policy.CloudaccountId != policy.CloudaccountId {
			return nil, httperrors.NewConflictError("policy %s(%s) and user not with same provider", policy.Name, policy.Id)
		}
		newP.Add(policy.Id)
	}

	add, del := []api.SPolicy{}, []api.SPolicy{}
	for _, id := range set.Difference(local, newP).List() {
		policy := policyMaps[id.(string)]
		del = append(del, api.SPolicy{
			Name:       policy.Name,
			ExternalId: policy.ExternalId,
			PolicyType: policy.PolicyType,
		})
	}

	for _, id := range set.Difference(newP, local).List() {
		policy := policyMaps[id.(string)]
		add = append(add, api.SPolicy{
			Name:       policy.Name,
			ExternalId: policy.ExternalId,
			PolicyType: policy.PolicyType,
		})
	}
	return nil, self.StartSetPoliciesTask(ctx, userCred, add, del, "")
}

func (self *SClouduser) StartSetPoliciesTask(ctx context.Context, userCred mcclient.TokenCredential, add, del []api.SPolicy, parentTaskId string) error {
	params := jsonutils.NewDict()
	if len(add) > 0 {
		params.Set("add", jsonutils.Marshal(add))
	}
	if len(del) > 0 {
		params.Set("del", jsonutils.Marshal(del))
	}
	task, err := taskman.TaskManager.NewTask(ctx, "ClouduserSetPoliciesTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_SYNC_STATUS, "")
	return task.ScheduleRun(nil)
}

// 设置用户权限组列表(全量覆盖)
// 用户状态必须为: available
func (self *SClouduser) PerformSetGroups(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserSetGroupsInput) (jsonutils.JSONObject, error) {
	if self.Status != apis.STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not set groups in status %s", self.Status)
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
	for i := range input.CloudgroupIds {
		groupObj, err := validators.ValidateModel(ctx, userCred, CloudgroupManager, &input.CloudgroupIds[i])
		if err != nil {
			return nil, err
		}
		group := groupObj.(*SCloudgroup)
		if group.ManagerId != self.ManagerId || group.CloudaccountId != self.CloudaccountId {
			return nil, httperrors.NewConflictError("group and user do not belong to the same account")
		}
		newG.Add(group.Id)
		groupMaps[group.Id] = group
	}

	add, del := []api.SGroup{}, []api.SGroup{}
	for _, id := range set.Difference(local, newG).List() {
		group := groupMaps[id.(string)]
		del = append(del, api.SGroup{
			Id:   group.Id,
			Name: group.Name,
		})
	}

	for _, id := range set.Difference(newG, local).List() {
		group := groupMaps[id.(string)]
		add = append(add, api.SGroup{
			Id:   group.Id,
			Name: group.Name,
		})
	}

	return nil, self.StartSetGroupsTask(ctx, userCred, add, del, "")
}

func (self *SClouduser) StartSetGroupsTask(ctx context.Context, userCred mcclient.TokenCredential, add, del []api.SGroup, parentTaskId string) error {
	params := jsonutils.NewDict()
	if len(add) > 0 {
		params.Set("add", jsonutils.Marshal(add))
	}
	if len(del) > 0 {
		params.Set("del", jsonutils.Marshal(del))
	}
	task, err := taskman.TaskManager.NewTask(ctx, "ClouduserSetGroupsTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_SYNC_STATUS, "")
	return task.ScheduleRun(nil)
}

// 将用户加入权限组
// 用户状态必须为: available
func (self *SClouduser) PerformJoinGroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserJoinGroupInput) (jsonutils.JSONObject, error) {
	if self.Status != apis.STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not join group in status %s", self.Status)
	}
	groupObj, err := validators.ValidateModel(ctx, userCred, CloudgroupManager, &input.CloudgroupId)
	if err != nil {
		return nil, err
	}
	group := groupObj.(*SCloudgroup)

	if group.ManagerId != self.ManagerId || group.CloudaccountId != self.CloudaccountId {
		return nil, httperrors.NewConflictError("group and user do not belong to the same account")
	}

	add := []api.SGroup{
		{
			Id:   group.Id,
			Name: group.Name,
		},
	}

	return nil, self.StartSetGroupsTask(ctx, userCred, add, nil, "")
}

// 将用户从权限组中移除
// 用户状态必须为: available
func (self *SClouduser) PerformLeaveGroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserLeaveGroupInput) (jsonutils.JSONObject, error) {
	if self.Status != apis.STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not leave group in status %s", self.Status)
	}
	groupObj, err := validators.ValidateModel(ctx, userCred, CloudgroupManager, &input.CloudgroupId)
	if err != nil {
		return nil, err
	}
	group := groupObj.(*SCloudgroup)

	_, err = self.GetCloudgroup(group.Id)
	if err != nil && errors.Cause(err) == sql.ErrNoRows {
		return nil, nil
	}

	del := []api.SGroup{
		{
			Id:   group.Id,
			Name: group.Name,
		},
	}

	return nil, self.StartSetGroupsTask(ctx, userCred, nil, del, "")
}

// 绑定用户权限
// 用户状态必须为: available
func (self *SClouduser) PerformAttachPolicy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserAttachPolicyInput) (jsonutils.JSONObject, error) {
	if self.Status != apis.STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not detach policy in status %s", self.Status)
	}
	policyObj, err := validators.ValidateModel(ctx, userCred, CloudpolicyManager, &input.CloudpolicyId)
	if err != nil {
		return nil, err
	}
	policy := policyObj.(*SCloudpolicy)
	if policy.ManagerId != self.ManagerId || policy.CloudaccountId != self.CloudaccountId {
		return nil, httperrors.NewConflictError("policy and user do not belong to the same account")
	}

	_, err = self.GetCloudpolicy(input.CloudpolicyId)
	if err == nil || errors.Cause(err) == sqlchemy.ErrDuplicateEntry {
		return nil, httperrors.NewDuplicateResourceError("policy %s has aleady attach this user", input.CloudpolicyId)
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

func (self *SClouduser) attachPolicy(policyId string) error {
	up := &SClouduserPolicy{}
	up.SetModelManager(ClouduserPolicyManager, up)
	up.CloudpolicyId = policyId
	up.ClouduserId = self.Id
	return ClouduserPolicyManager.TableSpec().Insert(context.Background(), up)
}

// 解绑用户权限
// 用户状态必须为: available
func (self *SClouduser) PerformDetachPolicy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserDetachPolicyInput) (jsonutils.JSONObject, error) {
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

// 同步用户状态
func (self *SClouduser) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserSyncstatusInput) (jsonutils.JSONObject, error) {
	return nil, self.StartClouduserSyncstatusTask(ctx, userCred, "")
}

func (self *SClouduser) StartClouduserSyncstatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClouduserSyncstatusTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_SYNC_STATUS, "")
	return task.ScheduleRun(nil)
}

// 重置用户密码
// 用户状态必须为: available
func (self *SClouduser) PerformResetPassword(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserResetPasswordInput) (jsonutils.JSONObject, error) {
	if self.Status != apis.STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Can not reset password in status %s", self.Status)
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
	self.SetStatus(ctx, userCred, api.CLOUD_USER_STATUS_RESET_PASSWORD, "")
	return task.ScheduleRun(nil)
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

	logclient.AddSimpleActionLog(self, logclient.ACT_CHANGE_OWNER, map[string]interface{}{"old": oldUserId, "newUserId": newUserId}, userCred, true)

	return nil, self.StartClouduserResetPasswordTask(ctx, userCred, "", "")
}

func (self *SClouduser) GetCloudgroupQuery() *sqlchemy.SQuery {
	sq := CloudgroupUserManager.Query("cloudgroup_id").Equals("clouduser_id", self.Id)
	return CloudgroupManager.Query().In("id", sq.SubQuery())
}

func (self *SClouduser) GetCloudgroupCount() (int, error) {
	return self.GetCloudgroupQuery().CountWithError()
}

func (self *SClouduser) GetCloudgroup(id string) (*SCloudgroup, error) {
	q := self.GetCloudgroupQuery().Equals("id", id)
	groups := []SCloudgroup{}
	err := db.FetchModelObjects(CloudgroupManager, q, &groups)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	if len(groups) > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	}
	if len(groups) == 0 {
		return nil, sql.ErrNoRows
	}
	return &groups[0], nil
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

func (self *SClouduser) PerformCreateAccessKey(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserCreateAccessKeyInput) (jsonutils.JSONObject, error) {
	user, err := self.GetIClouduser()
	if err != nil {
		return nil, errors.Wrapf(err, "GetIClouduser error")
	}
	ak, err := user.CreateAccessKey(input.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDetailsAccessKeys error")
	}
	return jsonutils.Marshal(ak), nil
}

func (self *SClouduser) GetDetailsAccessKeys(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	user, err := self.GetIClouduser()
	if err != nil {
		return nil, errors.Wrapf(err, "GetIClouduser")
	}
	aks, err := user.GetAccessKeys()
	if err != nil {
		return nil, errors.Wrapf(err, "GetAccessKeys")
	}
	ret := struct {
		Data  []cloudprovider.SAccessKey
		Total int
		Limit int
	}{
		Data:  aks,
		Total: len(aks),
		Limit: 20,
	}
	return jsonutils.Marshal(ret), nil
}

func (self *SClouduser) PerformDeleteAccessKey(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ClouduserDeleteAccessKeyInput) (jsonutils.JSONObject, error) {
	user, err := self.GetIClouduser()
	if err != nil {
		return nil, errors.Wrapf(err, "GetIClouduser")
	}
	err = user.DeleteAccessKey(input.AccessKey)
	if err != nil {
		return nil, errors.Wrapf(err, "DeleteAccessKey")
	}
	return nil, nil
}

func (self *SCloudaccount) SyncCloudusers(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	iUsers []cloudprovider.IClouduser,
	managerId string,
) ([]SClouduser, []cloudprovider.IClouduser, compare.SyncResult) {
	lockman.LockRawObject(ctx, ClouduserManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, managerId))
	defer lockman.ReleaseRawObject(ctx, ClouduserManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, managerId))

	result := compare.SyncResult{}
	dbUsers, err := self.GetCloudusers(managerId)
	if err != nil {
		result.Error(errors.Wrap(err, "GetCloudusers"))
		return nil, nil, result
	}

	localUsers := []SClouduser{}
	remoteUsers := []cloudprovider.IClouduser{}

	removed := make([]SClouduser, 0)
	commondb := make([]SClouduser, 0)
	commonext := make([]cloudprovider.IClouduser, 0)
	added := make([]cloudprovider.IClouduser, 0)

	err = compare.CompareSets(dbUsers, iUsers, &removed, &commondb, &commonext, &added)
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
		err = commondb[i].SyncWithClouduser(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		localUsers = append(localUsers, commondb[i])
		remoteUsers = append(remoteUsers, commonext[i])
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		user, err := self.newClouduser(ctx, userCred, added[i], managerId)
		if err != nil {
			result.AddError(err)
			continue
		}
		localUsers = append(localUsers, *user)
		remoteUsers = append(remoteUsers, added[i])
		result.Add()
	}

	return localUsers, remoteUsers, result
}

func (self *SCloudaccount) GetCloudusers(managerId string) ([]SClouduser, error) {
	users := []SClouduser{}
	q := ClouduserManager.Query().Equals("cloudaccount_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	err := db.FetchModelObjects(ClouduserManager, q, &users)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return users, nil
}

func (self *SCloudaccount) newClouduser(ctx context.Context, userCred mcclient.TokenCredential, iUser cloudprovider.IClouduser, managerId string) (*SClouduser, error) {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	user := &SClouduser{}
	user.SetModelManager(ClouduserManager, user)
	user.Name = iUser.GetName()
	user.ExternalId = iUser.GetGlobalId()
	user.Status = apis.STATUS_AVAILABLE
	user.CloudaccountId = self.Id
	user.ManagerId = managerId
	user.DomainId = self.DomainId
	switch iUser.IsConsoleLogin() {
	case true:
		user.IsConsoleLogin = tristate.True
	case false:
		user.IsConsoleLogin = tristate.False
	}
	err := ClouduserManager.TableSpec().Insert(ctx, user)
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}
	return user, nil
}
