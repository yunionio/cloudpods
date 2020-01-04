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
)

// SElasticcache.Account
type SElasticcacheAccountManager struct {
	db.SStatusStandaloneResourceBaseManager
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

	ElasticcacheId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"` // elastic cache instance id

	AccountType      string `width:"16" charset:"ascii" nullable:"false" list:"user" update:"user" create:"optional"` // 账号类型 normal |admin
	AccountPrivilege string `width:"16" charset:"ascii" nullable:"false" list:"user" update:"user" create:"optional"` // 账号权限 read | write | repl（复制, 复制权限支持读写，且开放SYNC/PSYNC命令）
	Password         string `width:"256" charset:"ascii" nullable:"false" list:"user" create:"optional"`              // 账号密码
}

func (manager *SElasticcacheAccountManager) SyncElasticcacheAccounts(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, cloudElasticcacheAccounts []cloudprovider.ICloudElasticcacheAccount) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, elasticcache.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, elasticcache.GetOwnerId()))

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

	err := manager.TableSpec().Insert(&account)
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudElasticcacheAccount.Insert")
	}

	return &account, nil
}

func (manager *SElasticcacheAccountManager) FetchParentId(ctx context.Context, data jsonutils.JSONObject) string {
	return jsonutils.GetAnyString(data, []string{"elasticcache_id", "elasticcache"})
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

func (manager *SElasticcacheAccountManager) FilterByParentId(q *sqlchemy.SQuery, parentId string) *sqlchemy.SQuery {
	if len(parentId) > 0 {
		q = q.Equals("elasticcache_id", parentId)
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

func (self *SElasticcacheAccount) GetIRegion() (cloudprovider.ICloudRegion, error) {
	_ec, err := db.FetchById(ElasticcacheManager, self.ElasticcacheId)
	if err != nil {
		return nil, err
	}

	ec := _ec.(*SElasticcache)
	provider, err := ec.GetDriver()
	if err != nil {
		return nil, fmt.Errorf("No cloudprovider for elastic cache %s: %s", ec.Name, err)
	}
	region := self.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("failed to find region for elastic cache %s", self.Name)
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (self *SElasticcacheAccount) AllowPerformResetPassword(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "reset_password")
}

func (self *SElasticcacheAccount) ValidatorResetPasswordData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	passwd, err := data.GetString("password")
	if err == nil && !seclib2.MeetComplxity(passwd) {
		return nil, httperrors.NewWeakPasswordError()
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
	password, err := self.GetDecodedPassword()
	if err != nil {
		return nil, err
	}

	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(self.Name), "username")
	ret.Add(jsonutils.NewString(password), "password")
	return ret, nil
}

func (self *SElasticcacheAccount) ValidatePurgeCondition(ctx context.Context) error {
	return nil
}
