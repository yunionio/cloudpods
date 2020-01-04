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
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudaccountManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	db.SDomainizedResourceBaseManager
}

var CloudaccountManager *SCloudaccountManager

func init() {
	CloudaccountManager = &SCloudaccountManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SCloudaccount{},
			"cloudaccounts_tbl",
			"cloudaccount",
			"cloudaccounts",
		),
	}
	CloudaccountManager.SetVirtualObject(CloudaccountManager)
}

type SCloudaccount struct {
	db.SEnabledStatusStandaloneResourceBase
	db.SDomainizedResourceBase
	SSyncableBaseResource

	LastAutoSync time.Time `list:"domain"`

	ProjectId string `name:"tenant_id" width:"128" charset:"ascii" list:"user" create:"domain_optional"`

	AccessUrl string `width:"64" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`

	Account string `width:"128" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"` // Column(VARCHAR(64, charset='ascii'), nullable=False)
	Secret  string `length:"0" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`  // Google需要秘钥认证,需要此字段比较长

	// BalanceKey string `width:"256" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`

	AccountId string `width:"128" charset:"utf8" nullable:"true" list:"domain" create:"domain_optional"`

	IsPublicCloud tristate.TriState `nullable:"false" get:"user" create:"optional" list:"user" default:"true"`
	IsOnPremise   bool              `nullable:"false" get:"user" create:"optional" list:"user" default:"false"`

	HasObjectStorage bool `nullable:"false" get:"user" create:"optional" list:"user" default:"false"`

	Provider string `width:"64" charset:"ascii" list:"domain" create:"domain_required"`

	EnableAutoSync      bool `default:"false" create:"domain_optional" list:"domain"`
	SyncIntervalSeconds int  `create:"domain_optional" list:"domain" update:"domain"`

	Balance      float64   `list:"domain"`
	ProbeAt      time.Time `list:"domain"`
	HealthStatus string    `width:"16" charset:"ascii" default:"normal" nullable:"false" list:"domain"`

	ErrorCount int `list:"domain"`

	AutoCreateProject bool `list:"domain" create:"domain_optional"`

	Version string               `width:"32" charset:"ascii" nullable:"true" list:"domain"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	Sysinfo jsonutils.JSONObject `get:"domain"`                                             // Column(JSONEncodedDict, nullable=True)

	Brand string `width:"64" charset:"utf8" nullable:"true" list:"domain" create:"optional"`

	Options *jsonutils.JSONDict `get:"domain" create:"domain_optional" update:"domain"`

	// for backward compatiblity, keep is_public field, but not usable
	IsPublic bool `default:"false" nullable:"false"`
	// add share_mode field to indicate the share range of this account
	ShareMode string `width:"32" charset:"ascii" nullable:"true" list:"domain"`
}

func (self *SCloudaccountManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SCloudaccountManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SCloudaccount) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SCloudaccount) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SCloudaccount) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SCloudaccount) GetCloudproviders() []SCloudprovider {
	return self.getCloudprovidersInternal(tristate.None)
}

func (self *SCloudaccount) IsAvailable() bool {
	if !self.Enabled {
		return false
	}

	if !utils.IsInStringArray(self.HealthStatus, api.CLOUD_PROVIDER_VALID_HEALTH_STATUS) {
		return false
	}

	return true
}

func (self *SCloudaccount) GetEnabledCloudproviders() []SCloudprovider {
	return self.getCloudprovidersInternal(tristate.True)
}

func (self *SCloudaccount) getCloudprovidersInternal(enabled tristate.TriState) []SCloudprovider {
	cloudproviders := []SCloudprovider{}
	q := CloudproviderManager.Query().Equals("cloudaccount_id", self.Id)
	if enabled.IsTrue() {
		q = q.IsTrue("enabled")
	} else if enabled.IsFalse() {
		q = q.IsFalse("enabled")
	}
	err := db.FetchModelObjects(CloudproviderManager, q, &cloudproviders)
	if err != nil {
		log.Errorf("getCloudproviders error: %v", err)
		return nil
	}
	return cloudproviders
}

func (self *SCloudaccount) ValidateDeleteCondition(ctx context.Context) error {
	// allow delete cloudaccount if it is disabled
	// if self.EnableAutoSync {
	//	return httperrors.NewInvalidStatusError("automatic syncing is enabled")
	// }
	if self.Enabled {
		return httperrors.NewInvalidStatusError("account is enabled")
	}
	if self.Status == api.CLOUD_PROVIDER_CONNECTED && self.getSyncStatus2() != api.CLOUD_PROVIDER_SYNC_STATUS_IDLE {
		return httperrors.NewInvalidStatusError("account is not idle")
	}
	cloudproviders := self.GetCloudproviders()
	for i := 0; i < len(cloudproviders); i++ {
		if err := cloudproviders[i].ValidateDeleteCondition(ctx); err != nil {
			return httperrors.NewInvalidStatusError("provider %s: %v", cloudproviders[i].Name, err)
		}
	}

	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SCloudaccount) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if strings.Index(self.Status, "delet") >= 0 {
		return nil, httperrors.NewInvalidStatusError("Cannot enable deleting account")
	}
	_, err := self.SEnabledStatusStandaloneResourceBase.PerformEnable(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	cloudproviders := self.GetCloudproviders()
	for i := 0; i < len(cloudproviders); i++ {
		if !cloudproviders[i].Enabled {
			_, err := cloudproviders[i].PerformEnable(ctx, userCred, query, data)
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}

func (self *SCloudaccount) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	_, err := self.SEnabledStatusStandaloneResourceBase.PerformDisable(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	cloudproviders := self.GetCloudproviders()
	for i := 0; i < len(cloudproviders); i++ {
		if cloudproviders[i].Enabled {
			_, err := cloudproviders[i].PerformDisable(ctx, userCred, query, data)
			if err != nil {
				return nil, err
			}
		}
	}
	/*if self.EnableAutoSync {
		err := self.disableAutoSync(ctx, userCred)
		if err != nil {
			return nil, err
		}
	}*/
	return nil, nil
}

func (self *SCloudaccount) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("sync_interval_seconds") {
		syncIntervalSecs, _ := data.Int("sync_interval_seconds")
		if syncIntervalSecs == 0 {
			syncIntervalSecs = int64(options.Options.DefaultSyncIntervalSeconds)
		} else if syncIntervalSecs < int64(options.Options.MinimalSyncIntervalSeconds) {
			syncIntervalSecs = int64(options.Options.MinimalSyncIntervalSeconds)
		}
		data.Set("sync_interval_seconds", jsonutils.NewInt(syncIntervalSecs))
	}
	if data.Contains("options") || data.Contains("remove_options") {
		var optionsJson *jsonutils.JSONDict
		if self.Options != nil {
			toRemoveKeys, _ := data.GetArray("remove_options")
			removes := make([]string, 0)
			if len(toRemoveKeys) > 0 {
				for i := range toRemoveKeys {
					key, _ := toRemoveKeys[i].GetString()
					removes = append(removes, key)
				}
			}
			optionsJson = self.Options.CopyExcludes(removes...)
		} else {
			optionsJson = jsonutils.NewDict()
		}
		toUpdate, _ := data.Get("options")
		if toUpdate != nil {
			optionsJson.Update(toUpdate)
		}
		data.Set("options", optionsJson)
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (manager *SCloudaccountManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.CloudaccountCreateInput) (api.CloudaccountCreateInput, error) {
	// check domainId
	err := db.ValidateCreateDomainId(ownerId.GetProjectDomainId())
	if err != nil {
		return input, err
	}

	data := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	tenantV := validators.NewModelIdOrNameValidator("tenant", "tenant", ownerId)
	tenantV.Optional(true)
	err = tenantV.Validate(data)
	if err != nil {
		return input, err
	}

	err = data.Unmarshal(&input)
	if err != nil {
		return input, httperrors.NewInputParameterError("failed to unmarshal input params error: %v", err)
	}

	if !cloudprovider.IsSupported(input.Provider) {
		return input, httperrors.NewInputParameterError("Unsupported provider %s", input.Provider)
	}
	providerDriver, _ := cloudprovider.GetProviderFactory(input.Provider)
	input.SCloudaccount, err = providerDriver.ValidateCreateCloudaccountData(ctx, userCred, input.SCloudaccountCredential)
	if err != nil {
		return input, err
	}
	if len(input.Brand) > 0 && input.Brand != providerDriver.GetName() {
		brands := providerDriver.GetSupportedBrands()
		if !utils.IsInStringArray(providerDriver.GetName(), brands) {
			brands = append(brands, providerDriver.GetName())
		}
		if !utils.IsInStringArray(input.Brand, brands) {
			return input, httperrors.NewUnsupportOperationError("Not support brand %s, only support %s", input.Brand, brands)
		}
	}
	input.IsPublicCloud = providerDriver.IsPublicCloud()
	input.IsOnPremise = providerDriver.IsOnPremise()

	q := manager.Query().Equals("provider", input.Provider)
	if len(input.Account) > 0 {
		q = q.Equals("account", input.Account)
	}
	if len(input.AccessUrl) > 0 {
		q = q.Equals("access_url", input.AccessUrl)
	}

	cnt, err := q.CountWithError()
	if err != nil {
		return input, httperrors.NewInternalServerError("check uniqness fail %s", err)
	}
	if cnt > 0 {
		return input, httperrors.NewConflictError("The account has been registered")
	}

	accountId, err := cloudprovider.IsValidCloudAccount(input.AccessUrl, input.Account, input.Secret, input.Provider)
	if err != nil {
		if err == cloudprovider.ErrNoSuchProvder {
			return input, httperrors.NewResourceNotFoundError("no such provider %s", input.Provider)
		}
		//log.Debugf("ValidateCreateData %s", err.Error())
		return input, httperrors.NewInputParameterError("invalid cloud account info error: %s", err.Error())
	}

	// check accountId uniqueness
	if len(accountId) > 0 {
		cnt, err := manager.Query().Equals("account_id", accountId).CountWithError()
		if err != nil {
			return input, httperrors.NewInternalServerError("check account_id duplication error %s", err)
		}
		if cnt > 0 {
			return input, httperrors.NewDuplicateResourceError("the account has been registerd %s", accountId)
		}
		input.AccountId = accountId
	}

	if input.SyncIntervalSeconds == 0 {
		input.SyncIntervalSeconds = options.Options.DefaultSyncIntervalSeconds
	} else if input.SyncIntervalSeconds < options.Options.MinimalSyncIntervalSeconds {
		input.SyncIntervalSeconds = options.Options.MinimalSyncIntervalSeconds
	}

	if !input.AutoCreateProject {
		if userCred.GetProjectDomainId() != ownerId.GetProjectDomainId() {
			s := auth.GetAdminSession(ctx, consts.GetRegion(), "v1")
			params := jsonutils.Marshal(map[string]string{"domain_id": ownerId.GetProjectDomainId()})
			tenants, err := modules.Projects.List(s, params)
			if err != nil {
				return input, err
			}
			if tenants.Total == 0 {
				return input, httperrors.NewInputParameterError("There is no projects under the domain %s", ownerId.GetProjectDomainId())
			}
		}
	}

	input.EnabledStatusStandaloneResourceCreateInput, err = manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusStandaloneResourceCreateInput)
	if err != nil {
		return input, err
	}

	return input, nil
}

func (self *SCloudaccount) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	self.Enabled = true
	if len(self.Brand) == 0 {
		self.Brand = self.Provider
	}
	self.DomainId = ownerId.GetProjectDomainId()
	// self.EnableAutoSync = false
	self.ShareMode = api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN
	return self.SEnabledStatusStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (self *SCloudaccount) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.savePassword(self.Secret)

	// if !self.EnableAutoSync {
	self.StartSyncCloudProviderInfoTask(ctx, userCred, nil, "")
	// }
}

func (self *SCloudaccount) savePassword(secret string) error {
	sec, err := utils.EncryptAESBase64(self.Id, secret)
	if err != nil {
		return err
	}

	_, err = db.Update(self, func() error {
		self.Secret = sec
		return nil
	})
	return err
}

func (self *SCloudaccount) getPassword() (string, error) {
	return utils.DescryptAESBase64(self.Id, self.Secret)
}

func (self *SCloudaccount) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "sync")
}

func (self *SCloudaccount) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.Enabled {
		return nil, httperrors.NewInvalidStatusError("Account disabled")
	}
	if self.EnableAutoSync {
		return nil, httperrors.NewInvalidStatusError("Account auto sync enabled")
	}
	syncRange := SSyncRange{}
	err := data.Unmarshal(&syncRange)
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid input %s", err)
	}
	if syncRange.FullSync || len(syncRange.Region) > 0 || len(syncRange.Zone) > 0 || len(syncRange.Host) > 0 {
		syncRange.DeepSync = true
	}
	if self.CanSync() || syncRange.Force {
		err = self.StartSyncCloudProviderInfoTask(ctx, userCred, &syncRange, "")
	}
	return nil, err
}

func (self *SCloudaccount) AllowPerformUpdateCredential(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "update-credential")
}

func (self *SCloudaccount) PerformUpdateCredential(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.Enabled {
		return nil, httperrors.NewInvalidStatusError("Account disabled")
	}

	providerDriver, err := self.GetProviderFactory()
	if err != nil {
		return nil, httperrors.NewBadRequestError("failed to found provider factory error: %v", err)
	}

	input := cloudprovider.SCloudaccountCredential{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInputParameterError("failed to unmarshal input params: %v", err)
	}

	account, err := providerDriver.ValidateUpdateCloudaccountCredential(ctx, userCred, input, self.Account)
	if err != nil {
		return nil, err
	}

	changed := false
	if len(account.Secret) > 0 || len(account.Account) > 0 {
		// check duplication
		q := self.GetModelManager().Query()
		q = q.Equals("account", account.Account)
		q = q.Equals("access_url", self.AccessUrl)
		q = q.NotEquals("id", self.Id)
		cnt, err := q.CountWithError()
		if err != nil {
			return nil, httperrors.NewInternalServerError("check uniqueness fail %s", err)
		}
		if cnt > 0 {
			return nil, httperrors.NewConflictError("account %s conflict", account.Account)
		}
	}

	originSecret, _ := self.getPassword()

	accountId, err := cloudprovider.IsValidCloudAccount(self.AccessUrl, account.Account, account.Secret, self.Provider)
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid cloud account info error: %s", err.Error())
	}
	// for backward compatibility
	if len(self.AccountId) > 0 && accountId != self.AccountId {
		return nil, httperrors.NewConflictError("inconsistent account_id, previous '%s' and now '%s'", self.AccountId, accountId)
	}

	if (account.Account != self.Account) || (account.Secret != originSecret) {
		if account.Account != self.Account {
			for _, cloudprovider := range self.GetCloudproviders() {
				if strings.Contains(cloudprovider.Account, self.Account) {
					_, err = db.Update(&cloudprovider, func() error {
						cloudprovider.Account = strings.ReplaceAll(cloudprovider.Account, self.Account, account.Account)
						return nil
					})
					if err != nil {
						return nil, err
					}
				}
			}
		}
		_, err = db.Update(self, func() error {
			self.Account = account.Account
			return nil
		})
		if err != nil {
			return nil, err
		}

		err = self.savePassword(account.Secret)
		if err != nil {
			return nil, err
		}

		for _, provider := range self.GetCloudproviders() {
			provider.savePassword(account.Secret)
		}
		changed = true
	}

	if changed {
		db.OpsLog.LogEvent(self, db.ACT_UPDATE, account.Account, userCred)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_UPDATE, account.Account, userCred, true)

		self.SetStatus(userCred, api.CLOUD_PROVIDER_INIT, "Change credential")
		self.StartSyncCloudProviderInfoTask(ctx, userCred, nil, "")
	}

	return nil, nil
}

func (self *SCloudaccount) StartSyncCloudProviderInfoTask(ctx context.Context, userCred mcclient.TokenCredential, syncRange *SSyncRange, parentTaskId string) error {
	params := jsonutils.NewDict()
	if syncRange != nil {
		params.Add(jsonutils.Marshal(syncRange), "sync_range")
	}

	task, err := taskman.TaskManager.NewTask(ctx, "CloudAccountSyncInfoTask", self, userCred, params, "", "", nil)
	if err != nil {
		log.Errorf("CloudAccountSyncInfoTask newTask error %s", err)
		return err
	}
	self.markStartSync(userCred)
	db.OpsLog.LogEvent(self, db.ACT_SYNC_HOST_START, "", userCred)
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudaccount) markStartSync(userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_QUEUED
		return nil
	})
	if err != nil {
		log.Errorf("Failed to markStartSync error: %v", err)
		return errors.Wrap(err, "Update")
	}
	providers := self.GetCloudproviders()
	for i := range providers {
		if providers[i].Enabled {
			err := providers[i].markStartingSync(userCred)
			if err != nil {
				return errors.Wrap(err, "providers.markStartSync")
			}
		}
	}
	return nil
}

func (self *SCloudaccount) MarkSyncing(userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_SYNCING
		self.LastSync = timeutils.UtcNow()
		self.LastSyncEndAt = time.Time{}
		return nil
	})
	if err != nil {
		log.Errorf("Failed to MarkSyncing error: %v", err)
		return err
	}
	return nil
}

func (self *SCloudaccount) MarkEndSyncWithLock(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	if self.SyncStatus == api.CLOUD_PROVIDER_SYNC_STATUS_IDLE {
		return nil
	}

	providers := self.GetCloudproviders()
	for i := range providers {
		err := providers[i].cancelStartingSync(userCred)
		if err != nil {
			return errors.Wrap(err, "providers.cancelStartingSync")
		}
	}

	if self.getSyncStatus2() != api.CLOUD_PROVIDER_SYNC_STATUS_IDLE {
		return errors.Error("some cloud providers not idle")
	}

	return self.markEndSync(userCred)
}

func (self *SCloudaccount) markEndSync(userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
		self.LastSyncEndAt = timeutils.UtcNow()
		return nil
	})
	if err != nil {
		log.Errorf("Failed to markEndSync error: %v", err)
		return err
	}
	return nil
}

func (self *SCloudaccount) GetProviderFactory() (cloudprovider.ICloudProviderFactory, error) {
	return cloudprovider.GetProviderFactory(self.Provider)
}

func (self *SCloudaccount) GetProvider() (cloudprovider.ICloudProvider, error) {
	if !self.Enabled {
		return nil, fmt.Errorf("Cloud provider is not enabled")
	}
	return self.getProviderInternal()
}

func (self *SCloudaccount) getProviderInternal() (cloudprovider.ICloudProvider, error) {
	secret, err := self.getPassword()
	if err != nil {
		return nil, fmt.Errorf("Invalid password %s", err)
	}
	return cloudprovider.GetProvider(self.Id, self.Name, self.AccessUrl, self.Account, secret, self.Provider)
}

func (self *SCloudaccount) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	provider, err := self.getProviderInternal()
	if err != nil {
		return nil, err
	}
	return provider.GetSubAccounts()
}

func (self *SCloudaccount) importSubAccount(ctx context.Context, userCred mcclient.TokenCredential, subAccount cloudprovider.SSubAccount) (*SCloudprovider, bool, error) {
	isNew := false
	q := CloudproviderManager.Query().Equals("cloudaccount_id", self.Id).Equals("account", subAccount.Account)
	providerCount, err := q.CountWithError()
	if err != nil {
		return nil, false, err
	}
	if providerCount > 1 {
		log.Errorf("cloudaccount %s has duplicate subaccount with name %s", self.Name, subAccount.Account)
		return nil, isNew, cloudprovider.ErrDuplicateId
	}
	if providerCount == 1 {
		providerObj, err := db.NewModelObject(CloudproviderManager)
		if err != nil {
			return nil, isNew, err
		}
		provider := providerObj.(*SCloudprovider)
		err = q.First(provider)
		if err != nil {
			return nil, isNew, err
		}
		provider.markProviderConnected(ctx, userCred, self.HealthStatus)
		return provider, isNew, nil
	}
	// not found, create a new cloudprovider
	isNew = true

	newCloudprovider, err := func() (*SCloudprovider, error) {
		lockman.LockClass(ctx, CloudproviderManager, "")
		defer lockman.ReleaseClass(ctx, CloudproviderManager, "")

		newName, err := db.GenerateName(CloudproviderManager, nil, subAccount.Name)
		if err != nil {
			return nil, err
		}
		newCloudprovider := SCloudprovider{}
		newCloudprovider.Account = subAccount.Account
		newCloudprovider.Secret = self.Secret
		newCloudprovider.CloudaccountId = self.Id
		newCloudprovider.Provider = self.Provider
		newCloudprovider.AccessUrl = self.AccessUrl
		newCloudprovider.Enabled = true
		newCloudprovider.Status = api.CLOUD_PROVIDER_CONNECTED
		if !options.Options.CloudaccountHealthStatusCheck {
			self.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
		}
		newCloudprovider.HealthStatus = self.HealthStatus
		newCloudprovider.Name = newName
		if !self.AutoCreateProject || len(self.ProjectId) > 0 {
			ownerId := self.GetOwnerId()
			if len(self.ProjectId) > 0 {
				t, err := db.TenantCacheManager.FetchTenantById(ctx, self.ProjectId)
				if err != nil {
					log.Errorf("cannot find tenant %s for domain %s", self.ProjectId, ownerId.GetProjectDomainId())
					return nil, err
				}
				ownerId = &db.SOwnerId{
					DomainId:  t.DomainId,
					ProjectId: t.Id,
				}
			} else if ownerId == nil || ownerId.GetProjectDomainId() == userCred.GetProjectDomainId() {
				ownerId = userCred
			} else {
				// find default project of domain
				t, err := db.TenantCacheManager.FindFirstProjectOfDomain(ownerId.GetProjectDomainId())
				if err != nil {
					log.Errorf("cannot find a valid porject for domain %s", ownerId.GetProjectDomainId())
					return nil, err
				}
				ownerId = &db.SOwnerId{
					DomainId:  t.DomainId,
					ProjectId: t.Id,
				}
			}
			newCloudprovider.DomainId = ownerId.GetProjectDomainId()
			newCloudprovider.ProjectId = ownerId.GetProjectId()
		}

		newCloudprovider.SetModelManager(CloudproviderManager, &newCloudprovider)

		err = CloudproviderManager.TableSpec().Insert(&newCloudprovider)
		if err != nil {
			return nil, err
		} else {
			return &newCloudprovider, nil
		}
	}()
	if err != nil {
		log.Errorf("insert new cloudprovider fail %s", err)
		return nil, isNew, err
	}

	db.OpsLog.LogEvent(newCloudprovider, db.ACT_CREATE, newCloudprovider.GetShortDesc(ctx), userCred)

	passwd, err := self.getPassword()
	if err != nil {
		return nil, isNew, err
	}

	newCloudprovider.savePassword(passwd)

	if self.AutoCreateProject && len(self.ProjectId) == 0 {
		err = newCloudprovider.syncProject(ctx, userCred)
		if err != nil {
			log.Errorf("syncproject fail %s", err)
			return nil, isNew, err
		}
	}

	return newCloudprovider, isNew, nil
}

//func (self *SCloudaccount) AllowPerformImport(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
//	return db.IsAdminAllowPerform(userCred, self, "import")
//}

//func (self *SCloudaccount) PerformImport(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
// autoCreateProject := jsonutils.QueryBoolean(data, "auto_create_project", false)
// autoSync := jsonutils.QueryBoolean(data, "auto_sync", false)
// err := self.startImportSubAccountTask(ctx, userCred, autoCreateProject, autoSync, "")
// noop
// return nil, nil
//}

func (manager *SCloudaccountManager) FetchCloudaccountById(accountId string) *SCloudaccount {
	providerObj, err := manager.FetchById(accountId)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return providerObj.(*SCloudaccount)
}

func (manager *SCloudaccountManager) FetchCloudaccountByIdOrName(accountId string) *SCloudaccount {
	providerObj, err := manager.FetchByIdOrName(nil, accountId)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("%s", err)
		}
		return nil
	}
	return providerObj.(*SCloudaccount)
}

func (self *SCloudaccount) GetProviderCount() (int, error) {
	q := CloudproviderManager.Query().Equals("cloudaccount_id", self.Id)
	return q.CountWithError()
}

func (self *SCloudaccount) GetHostCount() (int, error) {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	q := HostManager.Query().In("manager_id", subq).IsFalse("is_emulated")
	return q.CountWithError()
}

func (self *SCloudaccount) GetVpcCount() (int, error) {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	q := VpcManager.Query().In("manager_id", subq)
	return q.CountWithError()
}

func (self *SCloudaccount) GetStorageCount() (int, error) {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	q := StorageManager.Query().In("manager_id", subq)
	return q.CountWithError()
}

func (self *SCloudaccount) GetStoragecacheCount() (int, error) {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	q := StoragecacheManager.Query().In("manager_id", subq)
	return q.CountWithError()
}

func (self *SCloudaccount) GetEipCount() (int, error) {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	q := ElasticipManager.Query().In("manager_id", subq)
	return q.CountWithError()
}

func (self *SCloudaccount) GetRoutetableCount() (int, error) {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	q := RouteTableManager.Query().In("manager_id", subq)
	return q.CountWithError()
}

func (self *SCloudaccount) GetGuestCount() (int, error) {
	subsubq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	subq := HostManager.Query("id").In("manager_id", subsubq).SubQuery()
	q := GuestManager.Query().In("host_id", subq)
	return q.CountWithError()
}

func (self *SCloudaccount) GetDiskCount() (int, error) {
	subsubq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	subq := StorageManager.Query("id").In("manager_id", subsubq).SubQuery()
	q := DiskManager.Query().In("storage_id", subq)
	return q.CountWithError()
}

func (self *SCloudaccount) getProjectIds() []string {
	q := CloudproviderManager.Query("tenant_id").Equals("cloudaccount_id", self.Id).Distinct()
	rows, err := q.Rows()
	if err != nil {
		return nil
	}
	defer rows.Close()
	ret := make([]string, 0)
	for rows.Next() {
		var projId string
		err := rows.Scan(&projId)
		if err != nil {
			return nil
		}
		if len(projId) > 0 && utils.IsInStringArray(projId, ret) {
			ret = append(ret, projId)
		}
	}
	return ret
}

func (self *SCloudaccount) GetCloudEnv() string {
	if self.IsOnPremise {
		return api.CLOUD_ENV_ON_PREMISE
	} else if self.IsPublicCloud.IsTrue() {
		return api.CLOUD_ENV_PUBLIC_CLOUD
	} else {
		return api.CLOUD_ENV_PRIVATE_CLOUD
	}
}

func (self *SCloudaccount) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	extra = db.FetchModelExtraCountProperties(self, extra)
	// cnt, _ := self.getProviderCount()
	// int64(cnt)), "provider_count")
	// extra.Add(jsonutils.NewInt(int64(self.getHostCount())), "host_count")
	// extra.Add(jsonutils.NewInt(int64(self.getGuestCount())), "guest_count")
	// extra.Add(jsonutils.NewInt(int64(self.getDiskCount())), "disk_count")
	// extra.Add(jsonutils.NewString(self.getVersion()), "version")
	projects := jsonutils.NewArray()
	for _, projectId := range self.getProjectIds() {
		if proj, _ := db.TenantCacheManager.FetchTenantById(context.Background(), projectId); proj != nil {
			projJson := jsonutils.NewDict()
			projJson.Add(jsonutils.NewString(proj.Name), "tenant")
			projJson.Add(jsonutils.NewString(proj.Id), "tenant_id")
			projects.Add(projJson)
		}
	}
	extra.Add(projects, "projects")
	extra.Set("sync_interval_seconds", jsonutils.NewInt(int64(self.getSyncIntervalSeconds())))
	extra.Set("sync_status2", jsonutils.NewString(self.getSyncStatus2()))
	extra.Set("cloud_env", jsonutils.NewString(self.GetCloudEnv()))
	if len(self.ProjectId) > 0 {
		if proj, _ := db.TenantCacheManager.FetchTenantById(context.Background(), self.ProjectId); proj != nil {
			extra.Add(jsonutils.NewString(proj.Name), "tenant")
		}
	}
	return extra
}

func (self *SCloudaccount) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (self *SCloudaccount) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SEnabledStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(extra), nil
}

func migrateCloudprovider(cloudprovider *SCloudprovider) error {
	mainAccount, providerName := cloudprovider.Account, cloudprovider.Name

	if cloudprovider.Provider == api.CLOUD_PROVIDER_AZURE {
		accountInfo := strings.Split(cloudprovider.Account, "/")
		if len(accountInfo) == 2 {
			mainAccount = accountInfo[0]
			if len(cloudprovider.Description) > 0 {
				providerName = cloudprovider.Description
			}
		} else {
			msg := fmt.Sprintf("error azure provider account format %s", cloudprovider.Account)
			log.Errorf(msg)
			return fmt.Errorf(msg)
		}
	}

	account := SCloudaccount{}
	account.SetModelManager(CloudaccountManager, &account)
	q := CloudaccountManager.Query().Equals("access_url", cloudprovider.AccessUrl).
		Equals("account", mainAccount).
		Equals("provider", cloudprovider.Provider)
	err := q.First(&account)

	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if err == sql.ErrNoRows {
		account.AccessUrl = cloudprovider.AccessUrl
		account.Account = mainAccount
		account.Secret = cloudprovider.Secret
		account.LastSync = cloudprovider.LastSync
		// account.Sysinfo = cloudprovider.Sysinfo
		account.Provider = cloudprovider.Provider
		account.Name = providerName
		account.Status = cloudprovider.Status

		err := CloudaccountManager.TableSpec().Insert(&account)
		if err != nil {
			log.Errorf("Insert Account error: %v", err)
			return err
		}

		secret, err := cloudprovider.getPassword()
		if err != nil {
			account.markAccountDiscconected(context.Background(), auth.AdminCredential())
			log.Errorf("Get password from provider %s error %v", cloudprovider.Name, err)
		} else {
			err = account.savePassword(secret)
			if err != nil {
				log.Errorf("Set password for account %s error %v", account.Name, err)
				return err
			}
		}
	}

	_, err = db.Update(cloudprovider, func() error {
		cloudprovider.CloudaccountId = account.Id
		return nil
	})
	if err != nil {
		log.Errorf("Update provider %s error: %v", cloudprovider.Name, err)
		return err
	}

	return nil
}

func (manager *SCloudaccountManager) initializeBrand() error {
	accounts := []SCloudaccount{}
	q := manager.Query().IsNullOrEmpty("brand")
	err := db.FetchModelObjects(manager, q, &accounts)
	if err != nil {
		log.Errorf("fetch all clound account fail %v", err)
		return err
	}
	for i := 0; i < len(accounts); i++ {
		account := &accounts[i]
		_, err = db.Update(account, func() error {
			account.Brand = account.Provider
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (manager *SCloudaccountManager) initializeShareMode() error {
	accounts := []SCloudaccount{}
	q := manager.Query().IsNullOrEmpty("share_mode")
	err := db.FetchModelObjects(manager, q, &accounts)
	if err != nil {
		log.Errorf("fetch all clound account fail %v", err)
		return err
	}
	for i := 0; i < len(accounts); i++ {
		account := &accounts[i]
		_, err = db.Update(account, func() error {
			if account.IsPublic {
				account.ShareMode = api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM
			} else {
				account.ShareMode = api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (manager *SCloudaccountManager) InitializeData() error {
	cloudproviders := []SCloudprovider{}
	q := CloudproviderManager.Query()
	q = q.IsNullOrEmpty("cloudaccount_id")
	err := db.FetchModelObjects(CloudproviderManager, q, &cloudproviders)
	if err != nil {
		log.Errorf("fetch all clound provider fail %s", err)
		return err
	}
	for i := 0; i < len(cloudproviders); i++ {
		err = migrateCloudprovider(&cloudproviders[i])
		if err != nil {
			return err
		}
	}
	err = manager.initializeBrand()
	if err != nil {
		return err
	}
	err = manager.initializeShareMode()
	if err != nil {
		return err
	}
	return nil
}

func (self *SCloudaccount) GetBalance() (float64, error) {
	/*driver, err := self.GetProvider()
	if err != nil {
		return 0.0, err
	}
	return driver.GetBalance()*/
	return self.Balance, nil
}

func (self *SCloudaccount) AllowGetDetailsBalance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, self, "balance")
}

func (self *SCloudaccount) GetDetailsBalance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	balance, err := self.GetBalance()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewFloat(balance), "balance")
	return ret, nil
}

func (self *SCloudaccount) getHostPort() (string, int, error) {
	urlComponent, err := url.Parse(self.AccessUrl)
	if err != nil {
		return "", 0, err
	}
	host := urlComponent.Hostname()
	portStr := urlComponent.Port()
	port := 0
	if len(portStr) > 0 {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return "", 0, err
		}
	}
	if port == 0 {
		if urlComponent.Scheme == "http" {
			port = 80
		} else if urlComponent.Scheme == "https" {
			port = 443
		}
	}
	return host, port, nil
}

type SVCenterAccessInfo struct {
	VcenterId string
	Host      string
	Port      int
	Account   string
	Password  string
	PrivateId string
}

func (self *SCloudaccount) GetVCenterAccessInfo(privateId string) (SVCenterAccessInfo, error) {
	info := SVCenterAccessInfo{}

	host, port, err := self.getHostPort()
	if err != nil {
		return info, err
	}

	info.VcenterId = self.Id
	info.Host = host
	info.Port = port
	info.Account = self.Account
	info.Password = self.Secret
	info.PrivateId = privateId

	return info, nil
}

func (self *SCloudaccount) AllowPerformChangeProject(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "change-project")
}

func (self *SCloudaccount) PerformChangeProject(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	providers := self.GetCloudproviders()
	if len(providers) > 1 {
		return nil, httperrors.NewInvalidStatusError("multiple subaccounts")
	}
	if len(providers) == 0 {
		return nil, httperrors.NewInvalidStatusError("no subaccount")
	}
	return providers[0].PerformChangeProject(ctx, userCred, query, data)
}

func (manager *SCloudaccountManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	accountStr, _ := query.GetString("account")
	if len(accountStr) > 0 {
		query.(*jsonutils.JSONDict).Remove("account")
		accountObj, err := manager.FetchByIdOrName(userCred, accountStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(manager.Keyword(), accountStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		q = q.Equals("id", accountObj.GetId())
	}

	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	managerStr, _ := query.GetString("manager")
	if len(managerStr) > 0 {
		providerObj, err := CloudproviderManager.FetchByIdOrName(userCred, managerStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudproviderManager.Keyword(), managerStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		provider := providerObj.(*SCloudprovider)
		q = q.Equals("id", provider.CloudaccountId)
	}

	cloudEnvStr, _ := query.GetString("cloud_env")
	if cloudEnvStr == api.CLOUD_ENV_PUBLIC_CLOUD || jsonutils.QueryBoolean(query, "public_cloud", false) {
		q = q.IsTrue("is_public_cloud").IsFalse("is_on_premise")
	}

	if cloudEnvStr == api.CLOUD_ENV_PRIVATE_CLOUD || jsonutils.QueryBoolean(query, "private_cloud", false) {
		q = q.IsFalse("is_public_cloud").IsFalse("is_on_premise")
	}

	if cloudEnvStr == api.CLOUD_ENV_ON_PREMISE || jsonutils.QueryBoolean(query, "is_on_premise", false) {
		q = q.IsTrue("is_on_premise").IsFalse("is_public_cloud")
	}

	return q, nil
}

func (self *SCloudaccount) AllowPerformEnableAutoSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "enable-auto-sync")
}

func (self *SCloudaccount) PerformEnableAutoSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.EnableAutoSync {
		return nil, nil
	}

	if self.Status != api.CLOUD_PROVIDER_CONNECTED {
		return nil, httperrors.NewInvalidStatusError("cannot enable auto sync in status %s", self.Status)
	}

	syncIntervalSecs := int64(0)
	syncIntervalSecs, _ = data.Int("sync_interval_seconds")

	self.enableAutoSync(ctx, userCred, int(syncIntervalSecs))

	return nil, nil
}

func (self *SCloudaccount) enableAutoSync(ctx context.Context, userCred mcclient.TokenCredential, syncIntervalSecs int) error {
	diff, err := db.Update(self, func() error {
		if syncIntervalSecs > 0 {
			self.SyncIntervalSeconds = syncIntervalSecs
		}
		self.EnableAutoSync = true
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)

	return nil
}

func (self *SCloudaccount) AllowPerformDisableAutoSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "disable-auto-sync")
}

func (self *SCloudaccount) PerformDisableAutoSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.EnableAutoSync {
		return nil, nil
	}

	self.disableAutoSync(ctx, userCred)

	return nil, nil
}

func (self *SCloudaccount) disableAutoSync(ctx context.Context, userCred mcclient.TokenCredential) error {
	diff, err := db.Update(self, func() error {
		self.EnableAutoSync = false
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)

	return nil
}

func (account *SCloudaccount) markAccountDiscconected(ctx context.Context, userCred mcclient.TokenCredential) error {
	_, err := db.UpdateWithLock(ctx, account, func() error {
		account.ErrorCount = account.ErrorCount + 1
		account.HealthStatus = api.CLOUD_PROVIDER_HEALTH_UNKNOWN
		return nil
	})
	if err != nil {
		return err
	}
	return account.SetStatus(userCred, api.CLOUD_PROVIDER_DISCONNECTED, "")
}

func (account *SCloudaccount) markAllProvidersDicconnected(ctx context.Context, userCred mcclient.TokenCredential) error {
	providers := account.GetCloudproviders()
	for i := 0; i < len(providers); i += 1 {
		err := providers[i].markProviderDisconnected(ctx, userCred, "cloud account disconnected")
		if err != nil {
			return err
		}
	}
	return nil
}

func (account *SCloudaccount) markAccountConnected(ctx context.Context, userCred mcclient.TokenCredential) error {
	_, err := db.UpdateWithLock(ctx, account, func() error {
		account.ErrorCount = 0
		return nil
	})
	if err != nil {
		return err
	}
	return account.SetStatus(userCred, api.CLOUD_PROVIDER_CONNECTED, "")
}

func (account *SCloudaccount) shouldProbeStatus() bool {
	// connected state
	if account.Status != api.CLOUD_PROVIDER_DISCONNECTED {
		return true
	}
	// disconencted, but errorCount < threshold
	if account.ErrorCount < options.Options.MaxCloudAccountErrorCount {
		return true
	}
	// never synced
	if account.LastSyncEndAt.IsZero() {
		return true
	}
	// last sync is long time ago
	if time.Now().Sub(account.LastSyncEndAt) > time.Duration(options.Options.DisconnectedCloudAccountRetryProbeIntervalHours)*time.Hour {
		return true
	}
	return false
}

func (account *SCloudaccount) needSync() bool {
	if account.LastSyncEndAt.IsZero() {
		return true
	}
	if time.Now().Sub(account.LastSyncEndAt) > time.Duration(account.getSyncIntervalSeconds())*time.Second {
		return true
	}
	return false
}

func (manager *SCloudaccountManager) fetchRecordsByQuery(q *sqlchemy.SQuery) []SCloudaccount {
	recs := make([]SCloudaccount, 0)
	err := db.FetchModelObjects(manager, q, &recs)
	if err != nil {
		return nil
	}
	return recs
}

func (manager *SCloudaccountManager) initAllRecords() {
	recs := manager.fetchRecordsByQuery(manager.Query())
	for i := range recs {
		db.Update(&recs[i], func() error {
			recs[i].SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
			return nil
		})
	}
}

func (manager *SCloudaccountManager) AutoSyncCloudaccountTask(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if isStart && !options.Options.IsSlaveNode {
		// mark all the records to be idle
		CloudproviderRegionManager.initAllRecords()
		CloudproviderManager.initAllRecords()
		CloudaccountManager.initAllRecords()
	}

	q := manager.Query()

	accounts := make([]SCloudaccount, 0)
	err := db.FetchModelObjects(manager, q, &accounts)
	if err != nil {
		log.Errorf("Failed to fetch cloudaccount list to check status")
		return
	}

	for i := range accounts {
		if accounts[i].Enabled && accounts[i].shouldProbeStatus() && accounts[i].needSync() && accounts[i].CanSync() && rand.Float32() < 0.6 {
			accounts[i].SubmitSyncAccountTask(ctx, userCred, nil, true)
		}
	}
}

func (account *SCloudaccount) getSyncIntervalSeconds() int {
	if account.SyncIntervalSeconds > options.Options.MinimalSyncIntervalSeconds {
		return account.SyncIntervalSeconds
	}
	return options.Options.MinimalSyncIntervalSeconds
}

func (account *SCloudaccount) probeAccountStatus(ctx context.Context, userCred mcclient.TokenCredential) ([]cloudprovider.SSubAccount, error) {
	manager, err := account.getProviderInternal()
	if err != nil {
		log.Errorf("account.GetProvider failed: %s", err)
		return nil, err
	}
	balance, status, err := manager.GetBalance()
	if err != nil {
		switch err {
		case cloudprovider.ErrNotSupported:
			status = api.CLOUD_PROVIDER_HEALTH_NORMAL
		case cloudprovider.ErrNoBalancePermission:
			status = api.CLOUD_PROVIDER_HEALTH_NO_PERMISSION
		default:
			log.Errorf("manager.GetBalance %s fail %s", account.Name, err)
			status = api.CLOUD_PROVIDER_HEALTH_UNKNOWN
		}
	}
	version := manager.GetVersion()
	sysInfo, err := manager.GetSysInfo()
	if err != nil {
		log.Errorf("manager.GetSysInfo fail %s", err)
		return nil, err
	}
	factory := manager.GetFactory()
	diff, err := db.Update(account, func() error {
		isPublic := factory.IsPublicCloud()
		account.IsPublicCloud = tristate.NewFromBool(isPublic)
		account.IsOnPremise = factory.IsOnPremise()
		account.HasObjectStorage = factory.IsSupportObjectStorage()
		account.Balance = balance
		if !options.Options.CloudaccountHealthStatusCheck {
			status = api.CLOUD_PROVIDER_HEALTH_NORMAL
		}
		account.HealthStatus = status
		account.ProbeAt = timeutils.UtcNow()
		account.Version = version
		account.Sysinfo = sysInfo
		return nil
	})
	if err != nil {
		log.Errorf("Failed to update db %s", err)
	} else {
		db.OpsLog.LogSyncUpdate(account, diff, userCred)
	}

	return manager.GetSubAccounts()
}

func (account *SCloudaccount) importAllSubaccounts(ctx context.Context, userCred mcclient.TokenCredential, subAccounts []cloudprovider.SSubAccount) []SCloudprovider {
	oldProviders := account.GetCloudproviders()
	existProviders := make([]SCloudprovider, 0)
	existProviderKeys := make(map[string]int)
	for i := 0; i < len(subAccounts); i += 1 {
		provider, _, err := account.importSubAccount(ctx, userCred, subAccounts[i])
		if err != nil {
			log.Errorf("importSubAccount fail %s", err)
		} else {
			existProviders = append(existProviders, *provider)
			existProviderKeys[provider.Id] = 1
		}
	}
	for i := range oldProviders {
		if _, exist := existProviderKeys[oldProviders[i].Id]; !exist {
			oldProviders[i].markProviderDisconnected(ctx, userCred, "invalid subaccount")
		}
	}
	return existProviders
}

func (account *SCloudaccount) syncAccountStatus(ctx context.Context, userCred mcclient.TokenCredential) error {
	account.MarkSyncing(userCred)
	subaccounts, err := account.probeAccountStatus(ctx, userCred)
	if err != nil {
		account.markAllProvidersDicconnected(ctx, userCred)
		account.markAccountDiscconected(ctx, userCred)
		return err
	}
	account.markAccountConnected(ctx, userCred)
	providers := account.importAllSubaccounts(ctx, userCred, subaccounts)
	for i := range providers {
		if providers[i].Enabled {
			_, err := providers[i].prepareCloudproviderRegions(ctx, userCred)
			if err != nil {
				log.Errorf("syncCloudproviderRegion fail %s", err)
				return err
			}
		}
	}
	return nil
}

func (account *SCloudaccount) markAutoSync(userCred mcclient.TokenCredential) error {
	_, err := db.Update(account, func() error {
		account.LastAutoSync = timeutils.UtcNow()
		return nil
	})
	if err != nil {
		log.Errorf("Failed to markAutoSync error: %v", err)
		return err
	}
	return nil
}

func (account *SCloudaccount) SubmitSyncAccountTask(ctx context.Context, userCred mcclient.TokenCredential, waitChan chan error, autoSync bool) {
	RunSyncCloudAccountTask(func() {
		log.Debugf("syncAccountStatus %s %s", account.Id, account.Name)
		err := account.syncAccountStatus(ctx, userCred)
		if waitChan != nil {
			if err != nil {
				account.markEndSync(userCred)
			}
			waitChan <- err
		} else {
			syncCnt := 0
			if err == nil && autoSync && account.Enabled && account.EnableAutoSync {
				syncRange := SSyncRange{FullSync: true}
				account.markAutoSync(userCred)
				providers := account.GetEnabledCloudproviders()
				for i := range providers {
					providers[i].syncCloudproviderRegions(ctx, userCred, syncRange, nil, autoSync)
					syncCnt += 1
				}
			}
			if syncCnt == 0 {
				account.markEndSync(userCred)
			}
		}
	})
}

func (account *SCloudaccount) SyncCallSyncAccountTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	waitChan := make(chan error)
	account.SubmitSyncAccountTask(ctx, userCred, waitChan, false)
	err := <-waitChan
	return err
}

func (self *SCloudaccount) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	// override
	log.Infof("cloud account delete do nothing")
	return nil
}

func (self *SCloudaccount) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	self.SetStatus(userCred, api.CLOUD_PROVIDER_DELETED, "real delete")
	return self.SEnabledStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SCloudaccount) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartCloudaccountDeleteTask(ctx, userCred, "")
}

func (self *SCloudaccount) StartCloudaccountDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "CloudAccountDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("%s", err)
		return err
	}
	self.SetStatus(userCred, api.CLOUD_PROVIDER_START_DELETE, "StartCloudaccountDeleteTask")
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudaccount) getSyncStatus2() string {
	cprs := CloudproviderRegionManager.Query().SubQuery()
	providers := CloudproviderManager.Query().SubQuery()

	q := cprs.Query()
	q = q.Join(providers, sqlchemy.Equals(cprs.Field("cloudprovider_id"), providers.Field("id")))
	q = q.Filter(sqlchemy.Equals(providers.Field("cloudaccount_id"), self.Id))
	q = q.Filter(sqlchemy.NotEquals(cprs.Field("sync_status"), api.CLOUD_PROVIDER_SYNC_STATUS_IDLE))

	cnt, err := q.CountWithError()
	if err != nil {
		return api.CLOUD_PROVIDER_SYNC_STATUS_ERROR
	}
	if cnt > 0 {
		return api.CLOUD_PROVIDER_SYNC_STATUS_SYNCING
	} else {
		return api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
	}
}

func (manager *SCloudaccountManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []db.IModel, fields stringutils2.SSortedStrings) []*jsonutils.JSONDict {
	rows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields)
	if len(fields) == 0 || fields.Contains("domain") {
		domainIds := stringutils2.SSortedStrings{}
		for i := range objs {
			idStr := objs[i].GetOwnerId().GetProjectDomainId()
			domainIds = stringutils2.Append(domainIds, idStr)
		}
		domains := db.FetchProjects(domainIds, true)
		if domains != nil {
			for i := range rows {
				idStr := objs[i].GetOwnerId().GetProjectDomainId()
				if domain, ok := domains[idStr]; ok {
					rows[i].Add(jsonutils.NewString(domain.Name), "domain")
				}
			}
		}
	}
	return rows
}

func (account *SCloudaccount) setShareMode(userCred mcclient.TokenCredential, mode string) error {
	if account.ShareMode == mode {
		return nil
	}
	diff, err := db.Update(account, func() error {
		account.ShareMode = mode
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.Update")
	}
	db.OpsLog.LogEvent(account, db.ACT_UPDATE, diff, userCred)
	return nil
}

func (account *SCloudaccount) AllowPerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAllowPerform(rbacutils.ScopeSystem, userCred, account, "public")
}

func (account *SCloudaccount) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	scope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), account.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "public")
	if scope != rbacutils.ScopeSystem {
		return nil, httperrors.NewForbiddenError("not enough privilege")
	}

	err := account.setShareMode(userCred, api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	return nil, nil
}

func (account *SCloudaccount) AllowPerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAllowPerform(rbacutils.ScopeSystem, userCred, account, "private")
}

func (account *SCloudaccount) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	providers := account.GetCloudproviders()
	for i := range providers {
		if providers[i].DomainId != account.DomainId {
			return nil, httperrors.NewConflictError("provider is shared outside of domain")
		}
	}
	scope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), account.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "private")
	if scope != rbacutils.ScopeSystem {
		return nil, httperrors.NewForbiddenError("not enough privilege")
	}

	err := account.setShareMode(userCred, api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	return nil, nil
}

func (account *SCloudaccount) AllowPerformShareMode(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAllowPerform(rbacutils.ScopeSystem, userCred, account, "share-mode")
}

func (account *SCloudaccount) PerformShareMode(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := api.CloudaccountShareModeInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInputParameterError("fail to unmarshal input: %s", err)
	}
	err = input.Validate()
	if err != nil {
		return nil, err
	}
	if account.ShareMode == input.ShareMode {
		return nil, nil
	}

	if input.ShareMode == api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN {
		providers := account.GetCloudproviders()
		for i := range providers {
			if providers[i].DomainId != account.DomainId {
				return nil, httperrors.NewConflictError("provider is shared outside of domain")
			}
		}
	}

	scope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), account.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "share-mode")
	if scope != rbacutils.ScopeSystem {
		return nil, httperrors.NewForbiddenError("not enough privilege")
	}

	err = account.setShareMode(userCred, input.ShareMode)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	return nil, nil
}

func (manager *SCloudaccountManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacutils.ScopeProject, rbacutils.ScopeDomain:
			if len(owner.GetProjectDomainId()) > 0 {
				cloudproviders := CloudproviderManager.Query().SubQuery()
				q = q.LeftJoin(cloudproviders, sqlchemy.Equals(
					q.Field("id"),
					cloudproviders.Field("cloudaccount_id"),
				))
				q = q.Distinct()
				q = q.Filter(sqlchemy.OR(
					sqlchemy.AND(
						sqlchemy.Equals(q.Field("domain_id"), owner.GetProjectDomainId()),
						sqlchemy.Equals(q.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN),
					),
					sqlchemy.Equals(q.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM),
					sqlchemy.AND(
						sqlchemy.Equals(q.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_PROVIDER_DOMAIN),
						sqlchemy.Equals(cloudproviders.Field("domain_id"), owner.GetProjectDomainId()),
					),
				))
			}
		}
	}
	return q
}
