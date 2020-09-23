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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/choices"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// SElasticcache.Account
type SElasticcacheAccountManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SElasticcacheResourceBaseManager
}

var ElasticcacheAccountManager *SElasticcacheAccountManager

func init() {
	ElasticcacheAccountManager = &SElasticcacheAccountManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SElasticcacheAccount{},
			"elasticcacheaccounts_tbl",
			"elasticcacheaccount",
			"elasticcacheaccounts",
		),
	}
	ElasticcacheAccountManager.SetVirtualObject(ElasticcacheAccountManager)
}

type SElasticcacheAccount struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	SElasticcacheResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`

	// ElasticcacheId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"` // elastic cache instance id

	AccountType      string `width:"16" charset:"ascii" nullable:"false" list:"user" update:"user" create:"optional"` // 账号类型 normal |admin
	AccountPrivilege string `width:"16" charset:"ascii" nullable:"false" list:"user" update:"user" create:"optional"` // 账号权限 read | write | repl（复制, 复制权限支持读写，且开放SYNC/PSYNC命令）
	Password         string `width:"256" charset:"ascii" nullable:"false" list:"user" create:"optional"`              // 账号密码
}

func (manager *SElasticcacheAccountManager) SyncElasticcacheAccounts(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, cloudElasticcacheAccounts []cloudprovider.ICloudElasticcacheAccount) compare.SyncResult {
	lockman.LockRawObject(ctx, "elastic-cache-accounts", elasticcache.Id)
	defer lockman.ReleaseRawObject(ctx, "elastic-cache-accounts", elasticcache.Id)

	syncResult := compare.SyncResult{}

	dbAccounts, err := elasticcache.GetElasticcacheAccounts()
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := make([]SElasticcacheAccount, 0)
	commondb := make([]SElasticcacheAccount, 0)
	commonext := make([]cloudprovider.ICloudElasticcacheAccount, 0)
	added := make([]cloudprovider.ICloudElasticcacheAccount, 0)
	if err := compare.CompareSets(dbAccounts, cloudElasticcacheAccounts, &removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudElasticcacheAccount(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudElasticcacheAccount(ctx, userCred, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}

		syncResult.Update()
	}

	for i := 0; i < len(added); i++ {
		_, err := manager.newFromCloudElasticcacheAccount(ctx, userCred, elasticcache, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}

		syncResult.Add()
	}
	return syncResult
}

func (self *SElasticcacheAccount) syncRemoveCloudElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudElasticcacheAccount.Remove")
	}
	return self.Delete(ctx, userCred)
}

func (self *SElasticcacheAccount) SyncWithCloudElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential, extAccount cloudprovider.ICloudElasticcacheAccount) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extAccount.GetStatus()
		self.AccountType = extAccount.GetAccountType()
		self.AccountPrivilege = extAccount.GetAccountPrivilege()

		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "SyncWithCloudElasticcacheAccount.UpdateWithLock")
	}

	return nil
}

func (self *SElasticcacheAccount) GetRegion() *SCloudregion {
	iec, err := db.FetchById(ElasticcacheManager, self.ElasticcacheId)
	if err != nil {
		return nil
	}

	return iec.(*SElasticcache).GetRegion()
}

func (self *SElasticcacheAccount) GetOwnerId() mcclient.IIdentityProvider {
	return ElasticcacheManager.GetOwnerIdByElasticcacheId(self.ElasticcacheId)
}

func (manager *SElasticcacheAccountManager) newFromCloudElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, extAccount cloudprovider.ICloudElasticcacheAccount) (*SElasticcacheAccount, error) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	account := SElasticcacheAccount{}
	account.SetModelManager(manager, &account)

	account.ElasticcacheId = elasticcache.GetId()
	account.Name = extAccount.GetName()
	account.ExternalId = extAccount.GetGlobalId()
	account.Status = extAccount.GetStatus()
	account.AccountType = extAccount.GetAccountType()
	account.AccountPrivilege = extAccount.GetAccountPrivilege()

	err := manager.TableSpec().Insert(ctx, &account)
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudElasticcacheAccount.Insert")
	}

	return &account, nil
}

func (self *SElasticcacheAccount) GetUniqValues() jsonutils.JSONObject {
	return jsonutils.Marshal(map[string]string{"elasticcache_id": self.ElasticcacheId})
}

func (manager *SElasticcacheAccountManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	cacheId := jsonutils.GetAnyString(data, []string{"elasticcache_id", "elasticcache"})
	return jsonutils.Marshal(map[string]string{"elasticcache_id": cacheId})
}

func (manager *SElasticcacheAccountManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeProject
}

func (manager *SElasticcacheAccountManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return elasticcacheSubResourceFetchOwnerId(ctx, data)
}

func (manager *SElasticcacheAccountManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	return elasticcacheSubResourceFetchOwner(q, userCred, scope)
}

func (manager *SElasticcacheAccountManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	cacheId, _ := values.GetString("elasticcache_id")
	if len(cacheId) > 0 {
		q = q.Equals("elasticcache_id", cacheId)
	}
	return q
}

func (manager *SElasticcacheAccountManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (manager *SElasticcacheAccountManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var region *SCloudregion
	if id, _ := data.GetString("elasticcache"); len(id) > 0 {
		ec, err := db.FetchByIdOrName(ElasticcacheManager, userCred, id)
		if err != nil {
			return nil, fmt.Errorf("getting elastic cache instance failed")
		}
		region = ec.(*SElasticcache).GetRegion()
	} else {
		return nil, httperrors.NewMissingParameterError("elasticcache_id")
	}

	input := apis.StatusStandaloneResourceCreateInput{}
	var err error
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal StandaloneResourceCreateInput fail %s", err)
	}
	input, err = manager.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))

	passwd, _ := data.GetString("password")
	if reset, _ := data.Bool("reset_password"); reset && len(passwd) == 0 {
		passwd = seclib2.RandomPassword2(12)
		data.Set("password", jsonutils.NewString(passwd))
	}

	return region.GetDriver().ValidateCreateElasticcacheAccountData(ctx, userCred, ownerId, data)
}

func (self *SElasticcacheAccount) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if len(self.Password) > 0 {
		self.SavePassword(self.Password)
	}

	self.SetStatus(userCred, api.ELASTIC_CACHE_ACCOUNT_STATUS_CREATING, "")
	if err := self.StartElasticcacheAccountCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("Failed to create elastic account cache error: %v", err)
	}
}

func (self *SElasticcacheAccount) StartElasticcacheAccountCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ElasticcacheAccountCreateTask", self, userCred, jsonutils.NewDict(), parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticcacheAccount) GetCreateAliyunElasticcacheAccountParams() (cloudprovider.SCloudElasticCacheAccountInput, error) {
	ret := cloudprovider.SCloudElasticCacheAccountInput{}
	ret.AccountName = self.Name
	ret.Description = self.Description
	passwd, err := self.GetDecodedPassword()
	if err != nil {
		return ret, err
	}

	ret.AccountPassword = passwd

	switch self.AccountPrivilege {
	case "read":
		ret.AccountPrivilege = "RoleReadOnly"
	case "write":
		ret.AccountPrivilege = "RoleReadWrite"
	case "repl":
		ret.AccountPrivilege = "RoleRepl"
	}

	return ret, nil
}

func (self *SElasticcacheAccount) GetCreateQcloudElasticcacheAccountParams() (cloudprovider.SCloudElasticCacheAccountInput, error) {
	ret := cloudprovider.SCloudElasticCacheAccountInput{}
	ret.AccountName = self.Name
	ret.Description = self.Description
	passwd, err := self.GetDecodedPassword()
	if err != nil {
		return ret, err
	}

	ret.AccountPassword = passwd

	switch self.AccountPrivilege {
	case "read":
		ret.AccountPrivilege = "r"
	case "write":
		ret.AccountPrivilege = "rw"
	default:
		return ret, fmt.Errorf("ElasticcacheAccount.GetUpdateQcloudElasticcacheAccountParams invalid account_privilege %s", self.AccountPrivilege)
	}

	return ret, nil
}

func (self *SElasticcacheAccount) GetUpdateAliyunElasticcacheAccountParams(data jsonutils.JSONDict) (cloudprovider.SCloudElasticCacheAccountUpdateInput, error) {
	ret := cloudprovider.SCloudElasticCacheAccountUpdateInput{}

	if desc, _ := data.GetString("description"); len(desc) > 0 {
		ret.Description = &desc
	}

	if password, _ := data.GetString("password"); len(password) > 0 {
		ret.Password = &password
	}

	if ok := data.Contains("no_password_access"); ok {
		passwordAccess, _ := data.Bool("no_password_access")
		ret.NoPasswordAccess = &passwordAccess
	}

	if privilege, _ := data.GetString("account_privilege"); len(privilege) > 0 {
		var p string
		switch privilege {
		case "read":
			p = "RoleReadOnly"
		case "write":
			p = "RoleReadWrite"
		case "repl":
			p = "RoleRepl"
		default:
			return ret, fmt.Errorf("ElasticcacheAccount.GetUpdateAliyunElasticcacheAccountParams invalid account_privilege %s", privilege)
		}

		ret.AccountPrivilege = &p
	}

	return ret, nil
}

func (self *SElasticcacheAccount) GetUpdateHuaweiElasticcacheAccountParams(data jsonutils.JSONDict) (cloudprovider.SCloudElasticCacheAccountUpdateInput, error) {
	ret := cloudprovider.SCloudElasticCacheAccountUpdateInput{}

	if desc, _ := data.GetString("description"); len(desc) > 0 {
		ret.Description = &desc
	}

	if password, _ := data.GetString("password"); len(password) > 0 {
		ret.Password = &password
	}

	if oldpasswd, _ := data.GetString("old_password"); len(oldpasswd) > 0 {
		ret.OldPassword = &oldpasswd
	} else {
		oldpasswd, err := self.GetDecodedPassword()
		if err != nil {
			// can not update password, if old password is emtpy
			return ret, errors.Wrap(err, "ElasticcacheAccount.GetUpdateHuaweiElasticcacheAccountParams.OldPassword")
		}

		ret.OldPassword = &oldpasswd
	}

	if ok := data.Contains("no_password_access"); ok {
		passwordAccess, _ := data.Bool("no_password_access")
		ret.NoPasswordAccess = &passwordAccess
	}

	return ret, nil
}

func (self *SElasticcacheAccount) GetUpdateQcloudElasticcacheAccountParams(data jsonutils.JSONDict) (cloudprovider.SCloudElasticCacheAccountUpdateInput, error) {
	ret := cloudprovider.SCloudElasticCacheAccountUpdateInput{}

	if desc, _ := data.GetString("description"); len(desc) > 0 {
		ret.Description = &desc
	}

	if password, _ := data.GetString("password"); len(password) > 0 {
		ret.Password = &password
	}

	if ok := data.Contains("no_password_access"); ok {
		passwordAccess, _ := data.Bool("no_password_access")
		if self.AccountType == api.ELASTIC_CACHE_ACCOUNT_TYPE_ADMIN {
			ret.NoPasswordAccess = &passwordAccess
		} else {
			if passwordAccess == false {
				return ret, fmt.Errorf("ElasticcacheAccount.GetUpdateQcloudElasticcacheAccountParams normal account not support no auth access")
			}
		}
	}

	if privilege, _ := data.GetString("account_privilege"); len(privilege) > 0 {
		var p string
		switch privilege {
		case "read":
			p = "r"
		case "write":
			p = "rw"
		default:
			return ret, fmt.Errorf("ElasticcacheAccount.GetUpdateQcloudElasticcacheAccountParams invalid account_privilege %s", privilege)
		}

		ret.AccountPrivilege = &p
	}

	return ret, nil
}

func (self *SElasticcacheAccount) SavePassword(passwd string) error {
	passwd, err := utils.EncryptAESBase64(self.Id, passwd)
	if err != nil {
		return err
	}

	_, err = db.Update(self, func() error {
		self.Password = passwd
		return nil
	})

	return err
}

func (self *SElasticcacheAccount) GetDecodedPassword() (string, error) {
	return utils.DescryptAESBase64(self.Id, self.Password)
}

func (self *SElasticcacheAccount) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	self.SetStatus(userCred, api.ELASTIC_CACHE_ACCOUNT_STATUS_DELETING, "")
	return self.StartDeleteElasticcacheAccountTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (self *SElasticcacheAccount) StartDeleteElasticcacheAccountTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ElasticcacheAccountDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticcacheAccount) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SElasticcacheAccount) AllowPerformResetPassword(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "reset_password")
}

func (self *SElasticcacheAccount) ValidatorResetPasswordData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if reset, _ := data.Bool("reset_password"); reset {
		if _, err := data.GetString("password"); err != nil {
			randomPasswd := seclib2.RandomPassword2(12)
			data.(*jsonutils.JSONDict).Set("password", jsonutils.NewString(randomPasswd))
		}
	}

	passwd, _ := data.GetString("password")
	if len(passwd) > 0 {
		err := seclib2.ValidatePassword(passwd)
		if err != nil {
			return nil, err
		}
	}

	privilegeV := validators.NewStringChoicesValidator("account_privilege", choices.NewChoices(api.ELASTIC_CACHE_ACCOUNT_PRIVILEGE_READ, api.ELASTIC_CACHE_ACCOUNT_PRIVILEGE_WRITE, api.ELASTIC_CACHE_ACCOUNT_PRIVILEGE_REPL)).Optional(true)
	if err := privilegeV.Validate(data.(*jsonutils.JSONDict)); err != nil {
		return nil, err
	}
	return data, nil
}

func (self *SElasticcacheAccount) PerformResetPassword(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	self.SetStatus(userCred, api.ELASTIC_CACHE_STATUS_CHANGING, "")
	data, err := self.ValidatorResetPasswordData(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}

	return nil, self.StartResetPasswordTask(ctx, userCred, data.(*jsonutils.JSONDict), "")
}

func (self *SElasticcacheAccount) StartResetPasswordTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ElasticcacheAccountResetPasswordTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}

	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticcacheAccount) AllowGetDetailsLoginInfo(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, self, "login-info")
}

func (self *SElasticcacheAccount) GetDetailsLoginInfo(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(self.Name), "username")
	ret.Add(jsonutils.NewString(self.Password), "password")
	return ret, nil
}

func (self *SElasticcacheAccount) ValidatePurgeCondition(ctx context.Context) error {
	return nil
}

// 弹性缓存账号列表
func (manager *SElasticcacheAccountManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.ElasticcacheAccountListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SElasticcacheResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ElasticcacheFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SElasticcacheResourceBaseManager.ListItemFilter")
	}

	if len(input.AccountType) > 0 {
		q = q.In("account_type", input.AccountType)
	}
	if len(input.AccountPrivilege) > 0 {
		q = q.In("account_privilege", input.AccountPrivilege)
	}

	return q, nil
}

func (manager *SElasticcacheAccountManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.ElasticcacheAccountListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SElasticcacheResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ElasticcacheFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SElasticcacheResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SElasticcacheAccountManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SElasticcacheResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (self *SElasticcacheAccount) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.ElasticcacheAccountDetails, error) {
	return api.ElasticcacheAccountDetails{}, nil
}

func (manager *SElasticcacheAccountManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ElasticcacheAccountDetails {
	rows := make([]api.ElasticcacheAccountDetails, len(objs))

	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	cacheRows := manager.SElasticcacheResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	cacheIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.ElasticcacheAccountDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			ElasticcacheResourceInfo:        cacheRows[i],
		}
		account := objs[i].(*SElasticcacheAccount)
		cacheIds[i] = account.ElasticcacheId
	}

	caches := make(map[string]SElasticcache)
	err := db.FetchStandaloneObjectsByIds(ElasticcacheManager, cacheIds, &caches)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail: %v", err)
		return rows
	}

	virObjs := make([]interface{}, len(objs))
	for i := range rows {
		if cache, ok := caches[cacheIds[i]]; ok {
			virObjs[i] = &cache
			rows[i].ProjectId = cache.ProjectId
		}
	}

	projRows := ElasticcacheManager.SProjectizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, virObjs, fields, isList)
	for i := range rows {
		rows[i].ProjectizedResourceInfo = projRows[i]
	}

	return rows
}

func (manager *SElasticcacheAccountManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SElasticcacheResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SElasticcacheResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SElasticcacheResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}
