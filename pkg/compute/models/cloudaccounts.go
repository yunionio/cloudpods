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
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/esxi/vcenter"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	proxyapi "yunion.io/x/onecloud/pkg/apis/cloudcommon/proxy"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/proxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudaccountManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
	SProjectMappingResourceBaseManager
	SSyncableBaseResourceManager
}

var CloudaccountManager *SCloudaccountManager

func init() {
	CloudaccountManager = &SCloudaccountManager{
		SEnabledStatusInfrasResourceBaseManager: db.NewEnabledStatusInfrasResourceBaseManager(
			SCloudaccount{},
			"cloudaccounts_tbl",
			"cloudaccount",
			"cloudaccounts",
		),
	}
	CloudaccountManager.SetVirtualObject(CloudaccountManager)

	proxy.RegisterReferrer(CloudaccountManager)
}

type SCloudaccount struct {
	db.SEnabledStatusInfrasResourceBase

	SSyncableBaseResource

	// 项目Id
	ProjectId string `name:"tenant_id" width:"128" charset:"ascii" list:"user" create:"domain_optional"`

	// 云环境连接地址
	AccessUrl string `width:"64" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`

	// 云账号
	Account string `width:"128" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`

	// 云账号密码
	Secret string `length:"0" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`

	// 云环境唯一标识
	AccountId string `width:"128" charset:"utf8" nullable:"true" list:"domain" create:"domain_optional"`

	// 是否是公有云账号
	// example: true
	IsPublicCloud tristate.TriState `get:"user" create:"optional" list:"user" default:"true"`

	// 是否是本地IDC账号
	// example: false
	IsOnPremise bool `nullable:"false" get:"user" create:"optional" list:"user" default:"false"`

	// 云平台类型
	// example: google
	Provider string `width:"64" charset:"ascii" list:"domain" create:"domain_required"`

	// 账户余额
	// example: 124.2
	Balance float64 `list:"domain" width:"20" precision:"6"`

	// 上次账号探测时间
	ProbeAt time.Time `list:"domain"`

	// 账号健康状态
	// example: normal
	HealthStatus string `width:"16" charset:"ascii" default:"normal" nullable:"false" list:"domain"`

	// 账号探测异常错误次数
	ErrorCount int `list:"domain"`

	// 是否根据云上项目自动在本地创建对应项目
	// example: false
	AutoCreateProject bool `list:"domain" create:"domain_optional"`

	// 云API版本
	Version string `width:"32" charset:"ascii" nullable:"true" list:"domain"`

	// 云系统信息
	Sysinfo jsonutils.JSONObject `get:"domain"`

	// 品牌信息, 一般和provider相同
	// example: DStack
	Brand string `width:"64" charset:"utf8" nullable:"true" list:"domain" create:"optional"`

	// 额外信息
	Options *jsonutils.JSONDict `get:"domain" create:"domain_optional" update:"domain"`

	// for backward compatiblity, keep is_public field, but not usable
	// IsPublic bool `default:"false" nullable:"false"`
	// add share_mode field to indicate the share range of this account
	ShareMode string `width:"32" charset:"ascii" nullable:"true" list:"domain"`

	// 默认值proxyapi.ProxySettingId_DIRECT
	ProxySettingId string `width:"36" charset:"ascii" nullable:"false" list:"domain" create:"optional" update:"domain" default:"DIRECT"`

	// 公有云子账号登录地址
	IamLoginUrl string `width:"512" charset:"ascii" nullable:"false" list:"domain" update:"domain"`

	SAMLAuth            tristate.TriState `get:"user" update:"domain" create:"optional" list:"user" default:"false"`
	vmwareHostWireCache map[string][]SVs2Wire

	SProjectMappingResourceBase

	ReadOnly bool `default:"false" create:"domain_optional" list:"domain" update:"domain"`

	// 设置允许同步的账号及订阅
	SubAccounts *cloudprovider.SubAccounts `nullable:"true" get:"user" create:"optional"`

	// 缺失的权限，云账号操作资源时自动更新
	LakeOfPermissions *api.SAccountPermissions `length:"medium" get:"user" list:"user"`
}

func (self *SCloudaccount) GetCloudproviders() []SCloudprovider {
	return self.getCloudprovidersInternal(tristate.None)
}

func (self *SCloudaccount) IsAvailable() bool {
	if !self.GetEnabled() {
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

func (self *SCloudaccount) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if self.GetEnabled() {
		return httperrors.NewInvalidStatusError("account is enabled")
	}
	if self.Status == api.CLOUD_PROVIDER_CONNECTED && self.getSyncStatus2() != api.CLOUD_PROVIDER_SYNC_STATUS_IDLE {
		return httperrors.NewInvalidStatusError("account is not idle")
	}
	cloudproviders := self.GetCloudproviders()
	for i := 0; i < len(cloudproviders); i++ {
		if err := cloudproviders[i].ValidateDeleteCondition(ctx, nil); err != nil {
			return httperrors.NewInvalidStatusError("provider %s: %v", cloudproviders[i].Name, err)
		}
	}

	return self.SEnabledStatusInfrasResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SCloudaccount) enableAccountOnly(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	return self.SEnabledStatusInfrasResourceBase.PerformEnable(ctx, userCred, query, input)
}

func (self *SCloudaccount) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	if strings.Index(self.Status, "delet") >= 0 {
		return nil, httperrors.NewInvalidStatusError("Cannot enable deleting account")
	}
	_, err := self.enableAccountOnly(ctx, userCred, query, input)
	if err != nil {
		return nil, err
	}
	cloudproviders := self.GetCloudproviders()
	for i := 0; i < len(cloudproviders); i++ {
		if !cloudproviders[i].GetEnabled() {
			_, err := cloudproviders[i].PerformEnable(ctx, userCred, query, input)
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}

func (self *SCloudaccount) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	_, err := self.SEnabledStatusInfrasResourceBase.PerformDisable(ctx, userCred, query, input)
	if err != nil {
		return nil, err
	}
	cloudproviders := self.GetCloudproviders()
	for i := 0; i < len(cloudproviders); i++ {
		if cloudproviders[i].GetEnabled() {
			_, err := cloudproviders[i].PerformDisable(ctx, userCred, query, input)
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}

func (self *SCloudaccount) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.CloudaccountUpdateInput,
) (api.CloudaccountUpdateInput, error) {
	if (input.Options != nil && input.Options.Length() > 0) || len(input.RemoveOptions) > 0 {
		var optionsJson *jsonutils.JSONDict
		if self.Options != nil {
			removes := make([]string, 0)
			if len(input.RemoveOptions) > 0 {
				removes = append(removes, input.RemoveOptions...)
			}
			optionsJson = self.Options.CopyExcludes(removes...)
		} else {
			optionsJson = jsonutils.NewDict()
		}
		if input.Options != nil {
			optionsJson.Update(input.Options)
		}
		input.Options = optionsJson
	}

	factory, err := self.GetProviderFactory()
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrapf(err, "GetProviderFactory"))
	}
	if input.SAMLAuth != nil && *input.SAMLAuth && !factory.IsSupportSAMLAuth() {
		return input, httperrors.NewNotSupportedError("%s not support saml auth", self.Provider)
	}

	defaultRegion, _ := jsonutils.Marshal(self.Options).GetString("default_region")
	if len(input.ProxySettingId) > 0 {
		var proxySetting *proxy.SProxySetting
		proxySetting, input.ProxySettingResourceInput, err = proxy.ValidateProxySettingResourceInput(userCred, input.ProxySettingResourceInput)
		if err != nil {
			return input, errors.Wrap(err, "ValidateProxySettingResourceInput")
		}

		if proxySetting != nil && proxySetting.Id != self.ProxySettingId {
			// updated proxy setting, so do the check
			proxyFunc := proxySetting.HttpTransportProxyFunc()
			secret, _ := self.getPassword()
			_, _, err := cloudprovider.IsValidCloudAccount(cloudprovider.ProviderConfig{
				Vendor:        self.Provider,
				URL:           self.AccessUrl,
				Account:       self.Account,
				Secret:        secret,
				DefaultRegion: defaultRegion,
				ProxyFunc:     proxyFunc,

				AliyunResourceGroupIds: options.Options.AliyunResourceGroups,

				Options: input.Options,
			})
			if err != nil {
				return input, httperrors.NewInputParameterError("invalid proxy setting %s", err)
			}
		}
	}

	input.EnabledStatusInfrasResourceBaseUpdateInput, err = self.SEnabledStatusInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusInfrasResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.ValidateUpdateData")
	}

	return input, nil
}

func (self *SCloudaccount) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusInfrasResourceBase.PostUpdate(ctx, userCred, query, data)
	input := api.CloudaccountUpdateInput{}
	data.Unmarshal(&input)
	if input.Options != nil {
		logclient.AddSimpleActionLog(self, logclient.ACT_UPDATE_BILLING_OPTIONS, input.Options, userCred, true)
	}
	if input.CleanLakeOfPermissions {
		db.Update(self, func() error {
			self.LakeOfPermissions = nil
			return nil
		})
	}
}

func (manager *SCloudaccountManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.CloudaccountCreateInput,
) (api.CloudaccountCreateInput, error) {
	input, err := manager.validateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return input, err
	}

	input.EnabledStatusInfrasResourceBaseCreateInput, err = manager.SEnabledStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ValidateCreateData")
	}

	quota := &SDomainQuota{
		SBaseDomainQuotaKeys: quotas.SBaseDomainQuotaKeys{
			DomainId: ownerId.GetProjectDomainId(),
		},
		Cloudaccount: 1,
	}
	err = quotas.CheckSetPendingQuota(ctx, userCred, quota)
	if err != nil {
		return input, errors.Wrapf(err, "CheckSetPendingQuota")
	}

	return input, nil
}

func (manager *SCloudaccountManager) validateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.CloudaccountCreateInput,
) (api.CloudaccountCreateInput, error) {
	// check domainId
	err := db.ValidateCreateDomainId(ownerId.GetProjectDomainId())
	if err != nil {
		return input, errors.Wrap(err, "db.ValidateCreateDomainId")
	}

	if !cloudprovider.IsSupported(input.Provider) {
		return input, httperrors.NewInputParameterError("Unsupported provider %s", input.Provider)
	}
	providerDriver, _ := cloudprovider.GetProviderFactory(input.Provider)

	forceAutoCreateProject := providerDriver.IsNeedForceAutoCreateProject()
	if forceAutoCreateProject {
		input.AutoCreateProject = &forceAutoCreateProject
	}

	if len(input.ProjectId) > 0 {
		var proj *db.STenant
		proj, input.ProjectizedResourceInput, err = db.ValidateProjectizedResourceInput(ctx, input.ProjectizedResourceInput)
		if err != nil {
			return input, errors.Wrap(err, "db.ValidateProjectizedResourceInput")
		}
		if proj.DomainId != ownerId.GetProjectDomainId() {
			return input, httperrors.NewInputParameterError("Project %s(%s) not belong to domain %s(%s)", proj.Name, proj.Id, ownerId.GetProjectDomain(), ownerId.GetProjectDomainId())
		}
	} else if input.AutoCreateProject == nil || !*input.AutoCreateProject {
		log.Warningf("auto_create_project is off while no project_id specified")
		createProject := true
		input.AutoCreateProject = &createProject
	}

	if len(input.Zone) > 0 {
		obj, err := ZoneManager.FetchByIdOrName(userCred, input.Zone)
		if err != nil {
			return input, errors.Wrapf(err, "unable to fetch Zone %s", input.Zone)
		}
		input.Zone = obj.GetId()
	}

	if input.Options == nil {
		input.Options = jsonutils.NewDict()
	}
	input.Options.Update(jsonutils.Marshal(input.SCloudaccountCredential.SHCSOEndpoints))
	input.Options.Update(jsonutils.Marshal(input.SCloudaccountCredential.SCtyunExtraOptions))

	if len(input.DefaultRegion) > 0 {
		input.Options.Add(jsonutils.NewString(input.DefaultRegion), "default_region")
	}

	input.SCloudaccount, err = providerDriver.ValidateCreateCloudaccountData(ctx, input.SCloudaccountCredential)
	if err != nil {
		return input, err
	}
	if input.SAMLAuth != nil && *input.SAMLAuth && !providerDriver.IsSupportSAMLAuth() {
		return input, httperrors.NewNotSupportedError("%s not support saml auth", input.Provider)
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

	if !input.SkipDuplicateAccountCheck {
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
	}

	var proxyFunc httputils.TransportProxyFunc
	{
		if input.ProxySettingId == "" {
			input.ProxySettingId = proxyapi.ProxySettingId_DIRECT
		}
		var proxySetting *proxy.SProxySetting
		proxySetting, input.ProxySettingResourceInput, err = proxy.ValidateProxySettingResourceInput(userCred, input.ProxySettingResourceInput)
		if err != nil {
			return input, errors.Wrap(err, "ValidateProxySettingResourceInput")
		}
		proxyFunc = proxySetting.HttpTransportProxyFunc()
	}
	provider, accountId, err := cloudprovider.IsValidCloudAccount(cloudprovider.ProviderConfig{
		Name:          input.Name,
		Vendor:        input.Provider,
		URL:           input.AccessUrl,
		Account:       input.Account,
		Secret:        input.Secret,
		DefaultRegion: input.DefaultRegion,
		ProxyFunc:     proxyFunc,

		AdminProjectId:         auth.GetAdminSession(ctx, options.Options.Region).GetProjectDomainId(),
		AliyunResourceGroupIds: options.Options.AliyunResourceGroups,

		Options: input.Options,
	})
	if err != nil {
		if err == cloudprovider.ErrNoSuchProvder {
			return input, httperrors.NewResourceNotFoundError("no such provider %s", input.Provider)
		}
		return input, httperrors.NewGeneralError(err)
	}
	if input.DryRun && input.ShowSubAccounts {
		input.SubAccounts = &cloudprovider.SubAccounts{}
		input.SubAccounts.Accounts, err = provider.GetSubAccounts()
		if err != nil {
			return input, err
		}
		regions := provider.GetIRegions()
		for _, region := range regions {
			input.SubAccounts.Cloudregions = append(input.SubAccounts.Cloudregions, struct {
				Id     string
				Name   string
				Status string
			}{
				Id:     region.GetGlobalId(),
				Name:   region.GetName(),
				Status: region.GetStatus(),
			})
		}
	}

	// check accountId uniqueness
	if len(accountId) > 0 && !input.SkipDuplicateAccountCheck {
		cnt, err := manager.Query().Equals("account_id", accountId).CountWithError()
		if err != nil {
			return input, httperrors.NewInternalServerError("check account_id duplication error %s", err)
		}
		if cnt > 0 {
			return input, httperrors.NewDuplicateResourceError("the account has been registerd %s", accountId)
		}
		input.AccountId = accountId
	}

	return input, nil
}

func (self *SCloudaccount) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if !data.Contains("enabled") {
		self.SetEnabled(true)
	}
	if len(self.Brand) == 0 {
		self.Brand = self.Provider
	}
	self.DomainId = ownerId.GetProjectDomainId()
	// force private and share_mode=account_domain
	if !data.Contains("public_scope") {
		self.ShareMode = api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN
		self.IsPublic = false
		self.PublicScope = string(rbacutils.ScopeNone)
		// mark the public_scope has been set
		data.(*jsonutils.JSONDict).Set("public_scope", jsonutils.NewString(self.PublicScope))
	}
	if len(self.ShareMode) == 0 {
		if self.IsPublic {
			self.ShareMode = api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM
		} else {
			self.ShareMode = api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN
		}
	}
	return self.SEnabledStatusInfrasResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (self *SCloudaccount) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	quota := &SDomainQuota{
		SBaseDomainQuotaKeys: quotas.SBaseDomainQuotaKeys{
			DomainId: ownerId.GetProjectDomainId(),
		},
		Cloudaccount: 1,
	}
	err := quotas.CancelPendingUsage(ctx, userCred, quota, quota, true)
	if err != nil {
		log.Errorf("CancelPendingUsage fail %s", err)
	}

	self.SEnabledStatusInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.savePassword(self.Secret)

	if self.Enabled.IsTrue() {
		if self.Provider == api.CLOUD_PROVIDER_VMWARE {
			zone, _ := data.GetString("zone")
			self.StartSyncVmwareNetworkTask(ctx, userCred, "", zone)
		} else {
			self.StartSyncCloudProviderInfoTask(ctx, userCred, nil, "")
		}
	}
}

func (ca *SCloudaccount) PerformSyncVMwareNetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudaccountSyncVMwareNetworkInput) (jsonutils.JSONObject, error) {
	return nil, ca.StartSyncVMwareNetworkTask(ctx, userCred, "", input.Zone)
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

func (self *SCloudaccount) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SyncRangeInput) (jsonutils.JSONObject, error) {
	if !self.GetEnabled() {
		return nil, httperrors.NewInvalidStatusError("Account disabled")
	}

	if self.SyncStatus != api.CLOUD_PROVIDER_SYNC_STATUS_IDLE {
		return nil, httperrors.NewInvalidStatusError("Account is not idle")
	}

	syncRange := SSyncRange{SyncRangeInput: input}
	if syncRange.FullSync || len(syncRange.Region) > 0 || len(syncRange.Zone) > 0 || len(syncRange.Host) > 0 || len(syncRange.Resources) > 0 {
		syncRange.DeepSync = true
	}
	if self.CanSync() || syncRange.Force {
		return nil, self.StartSyncCloudProviderInfoTask(ctx, userCred, &syncRange, "")
	}
	return nil, httperrors.NewInvalidStatusError("Unable to synchronize frequently")
}

// 测试账号连通性(更新秘钥信息时)
func (self *SCloudaccount) PerformTestConnectivity(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input cloudprovider.SCloudaccountCredential) (jsonutils.JSONObject, error) {
	providerDriver, err := self.GetProviderFactory()
	if err != nil {
		return nil, httperrors.NewBadRequestError("failed to found provider factory error: %v", err)
	}

	account, err := providerDriver.ValidateUpdateCloudaccountCredential(ctx, input, self.Account)
	if err != nil {
		return nil, err
	}

	defaultRegion, _ := jsonutils.Marshal(self.Options).GetString("default_region")
	_, _, err = cloudprovider.IsValidCloudAccount(cloudprovider.ProviderConfig{
		URL:     self.AccessUrl,
		Vendor:  self.Provider,
		Account: account.Account,
		Secret:  account.Secret,

		DefaultRegion: defaultRegion,

		AliyunResourceGroupIds: options.Options.AliyunResourceGroups,

		ReadOnly: self.ReadOnly,

		ProxyFunc: self.proxyFunc(),
	})
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid cloud account info error: %s", err.Error())
	}

	return nil, nil
}

func (self *SCloudaccount) PerformUpdateCredential(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.GetEnabled() {
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

	account, err := providerDriver.ValidateUpdateCloudaccountCredential(ctx, input, self.Account)
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

	hcsoEndpoints := cloudprovider.SHCSOEndpoints{}
	if self.Provider == api.CLOUD_PROVIDER_HCSO && input.SHCSOEndpoints != nil {
		if self.Options == nil {
			self.Options = jsonutils.NewDict()
		}

		newOptions := jsonutils.Marshal(input.SHCSOEndpoints)
		_, err = db.UpdateWithLock(ctx, self, func() error {
			self.Options.Update(newOptions)
			return nil
		})
		if err != nil {
			return nil, err
		}

		err = self.Options.Unmarshal(&hcsoEndpoints)
		if err != nil {
			return nil, err
		}
	}

	defaultRegion, _ := jsonutils.Marshal(self.Options).GetString("default_region")
	_, accountId, err := cloudprovider.IsValidCloudAccount(cloudprovider.ProviderConfig{
		Name:          self.Name,
		Vendor:        self.Provider,
		URL:           self.AccessUrl,
		Account:       account.Account,
		Secret:        account.Secret,
		Options:       self.Options,
		DefaultRegion: defaultRegion,

		AliyunResourceGroupIds: options.Options.AliyunResourceGroups,

		ReadOnly: self.ReadOnly,

		ProxyFunc: self.proxyFunc(),
	})
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid cloud account info error: %s", err.Error())
	}

	isEqual := providerDriver.GetAccountIdEqualizer()
	// for backward compatibility
	if !isEqual(self.AccountId, accountId) {
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
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_UPDATE_CREDENTIAL, account.Account, userCred, true)

		self.SetStatus(userCred, api.CLOUD_PROVIDER_INIT, "Change credential")
		self.StartSyncCloudProviderInfoTask(ctx, userCred, nil, "")
	}

	return nil, nil
}

func (self *SCloudaccount) StartSyncVmwareNetworkTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, zone string) error {
	params := jsonutils.NewDict()
	if len(zone) != 0 {
		params.Set("zone", jsonutils.NewString(zone))
	}
	task, err := taskman.TaskManager.NewTask(ctx, "CloudAccountSyncVMwareNetworkTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "unable to create task CloudAccountSyncVMwareNetworkTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudaccount) StartSyncCloudProviderInfoTask(ctx context.Context, userCred mcclient.TokenCredential, syncRange *SSyncRange, parentTaskId string) error {
	params := jsonutils.NewDict()
	if syncRange != nil {
		params.Add(jsonutils.Marshal(syncRange), "sync_range")
	}

	task, err := taskman.TaskManager.NewTask(ctx, "CloudAccountSyncInfoTask", self, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.markStartSync(userCred, syncRange)
	db.OpsLog.LogEvent(self, db.ACT_SYNC_HOST_START, "", userCred)
	return task.ScheduleRun(nil)
}

func (self *SCloudaccount) markStartSync(userCred mcclient.TokenCredential, syncRange *SSyncRange) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_QUEUED
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update")
	}
	providers := self.GetCloudproviders()
	for i := range providers {
		if providers[i].GetEnabled() {
			err := providers[i].markStartingSync(userCred, syncRange)
			if err != nil {
				return errors.Wrap(err, "providers.markStartSync")
			}
		}
	}
	return nil
}

func (self *SCloudaccount) MarkSyncing(userCred mcclient.TokenCredential, probe bool) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_SYNCING
		if !probe {
			self.LastSync = timeutils.UtcNow()
			self.LastSyncEndAt = time.Time{}
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	return nil
}

func (self *SCloudaccount) MarkEndSyncWithLock(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

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

	return self.markEndSync(userCred, false)
}

func (self *SCloudaccount) markEndSync(userCred mcclient.TokenCredential, probe bool) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
		if !probe {
			self.LastSyncEndAt = timeutils.UtcNow()
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	return nil
}

func (self *SCloudaccount) GetProviderFactory() (cloudprovider.ICloudProviderFactory, error) {
	return cloudprovider.GetProviderFactory(self.Provider)
}

func (self *SCloudaccount) GetProvider(ctx context.Context) (cloudprovider.ICloudProvider, error) {
	if !self.GetEnabled() {
		return nil, fmt.Errorf("Cloud provider is not enabled")
	}
	return self.getProviderInternal(ctx)
}

func (self *SCloudaccount) proxySetting() *proxy.SProxySetting {
	m, err := proxy.ProxySettingManager.FetchById(self.ProxySettingId)
	if err != nil {
		log.Errorf("cloudaccount %s(%s): get proxysetting %s: %v",
			self.Name, self.Id, self.ProxySettingId, err)
		return nil
	}
	ps := m.(*proxy.SProxySetting)
	return ps
}

func (self *SCloudaccount) proxyFunc() httputils.TransportProxyFunc {
	ps := self.proxySetting()
	if ps != nil {
		return ps.HttpTransportProxyFunc()
	}
	return nil
}

func (self *SCloudaccount) UpdatePermission(ctx context.Context) func(string, string) {
	return func(service, permission string) {
		key := "update permission"

		lockman.LockRawObject(ctx, self.Id, key)
		defer lockman.ReleaseRawObject(ctx, self.Id, key)

		db.Update(self, func() error {
			data := api.SAccountPermissions{}
			if self.LakeOfPermissions != nil {
				data = *self.LakeOfPermissions
			}
			_, ok := data[service]
			if !ok {
				data[service] = api.SAccountPermission{}
			}
			permissions := data[service].Permissions
			if !utils.IsInStringArray(permission, permissions) {
				permissions = append(permissions, permission)
				data[service] = api.SAccountPermission{
					Permissions: permissions,
				}
			}
			self.LakeOfPermissions = &data
			return nil
		})
	}
}

func (self *SCloudaccount) getProviderInternal(ctx context.Context) (cloudprovider.ICloudProvider, error) {
	secret, err := self.getPassword()
	if err != nil {
		return nil, fmt.Errorf("Invalid password %s", err)
	}

	defaultRegion, _ := jsonutils.Marshal(self.Options).GetString("default_region")
	return cloudprovider.GetProvider(cloudprovider.ProviderConfig{
		Id:      self.Id,
		Name:    self.Name,
		Vendor:  self.Provider,
		URL:     self.AccessUrl,
		Account: self.Account,
		Secret:  secret,

		Options:       self.Options,
		DefaultRegion: defaultRegion,
		ProxyFunc:     self.proxyFunc(),

		ReadOnly:               self.ReadOnly,
		AliyunResourceGroupIds: options.Options.AliyunResourceGroups,

		UpdatePermission: self.UpdatePermission(ctx),
	})
}

func (self *SCloudaccount) GetSubAccounts(ctx context.Context) ([]cloudprovider.SSubAccount, error) {
	provider, err := self.getProviderInternal(ctx)
	if err != nil {
		return nil, err
	}
	return provider.GetSubAccounts()
}

func (self *SCloudaccount) getDefaultExternalProject(id string) (*SExternalProject, error) {
	q := ExternalProjectManager.Query().Equals("cloudaccount_id", self.Id).Equals("external_id", id)
	projects := []SExternalProject{}
	err := db.FetchModelObjects(ExternalProjectManager, q, &projects)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	if len(projects) > 1 {
		return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, id)
	}
	if len(projects) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
	}
	return &projects[0], nil
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
		provider := &SCloudprovider{}
		provider.SetModelManager(CloudproviderManager, provider)
		err = q.First(provider)
		if err != nil {
			return nil, isNew, errors.Wrapf(err, "q.First")
		}
		if len(provider.ProjectId) == 0 {
			db.Update(provider, func() error {
				if len(subAccount.DefaultProjectId) > 0 {
					proj, err := self.getDefaultExternalProject(subAccount.DefaultProjectId)
					if err != nil {
						logclient.AddSimpleActionLog(provider, logclient.ACT_UPDATE, errors.Wrapf(err, "getDefaultExternalProject(%s)", subAccount.DefaultProjectId), userCred, false)
					} else {
						provider.DomainId = proj.DomainId
						provider.ProjectId = proj.ProjectId
						return nil
					}
				}
				// find default project of domain
				ownerId := self.GetOwnerId()
				t, err := db.TenantCacheManager.FindFirstProjectOfDomain(ctx, ownerId.GetProjectDomainId())
				if err != nil {
					logclient.AddSimpleActionLog(provider, logclient.ACT_UPDATE, errors.Wrapf(err, "FindFirstProjectOfDomain(%s)", ownerId.GetProjectDomainId()), userCred, false)
					return errors.Wrapf(err, "FindFirstProjectOfDomain(%s)", ownerId.GetProjectDomainId())
				}
				provider.DomainId = t.DomainId
				provider.ProjectId = t.Id
				return nil
			})
		}
		provider.markProviderConnected(ctx, userCred, subAccount.HealthStatus)
		provider.updateName(ctx, userCred, subAccount.Name, subAccount.Desc)
		return provider, isNew, nil
	}
	// not found, create a new cloudprovider
	isNew = true

	newCloudprovider, err := func() (*SCloudprovider, error) {
		newCloudprovider := SCloudprovider{}
		newCloudprovider.Account = subAccount.Account
		newCloudprovider.Secret = self.Secret
		newCloudprovider.CloudaccountId = self.Id
		newCloudprovider.Provider = self.Provider
		newCloudprovider.AccessUrl = self.AccessUrl
		newCloudprovider.HealthStatus = subAccount.HealthStatus
		newCloudprovider.Description = subAccount.Desc
		newCloudprovider.DomainId = self.DomainId
		newCloudprovider.ProjectId = self.ProjectId
		if !options.Options.CloudaccountHealthStatusCheck {
			newCloudprovider.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
		}
		if newCloudprovider.HealthStatus == api.CLOUD_PROVIDER_HEALTH_NORMAL {
			newCloudprovider.SetEnabled(true)
			newCloudprovider.Status = api.CLOUD_PROVIDER_CONNECTED
		} else {
			newCloudprovider.SetEnabled(false)
			newCloudprovider.Status = api.CLOUD_PROVIDER_DISCONNECTED
		}
		if len(newCloudprovider.ProjectId) == 0 {
			ownerId := self.GetOwnerId()
			if ownerId.GetProjectDomainId() == userCred.GetProjectDomainId() {
				ownerId = userCred
			}
			newCloudprovider.DomainId = ownerId.GetProjectDomainId()
			newCloudprovider.ProjectId = ownerId.GetProjectId()
		}

		newCloudprovider.SetModelManager(CloudproviderManager, &newCloudprovider)

		err = func() error {
			lockman.LockRawObject(ctx, CloudproviderManager.Keyword(), "name")
			defer lockman.ReleaseRawObject(ctx, CloudproviderManager.Keyword(), "name")

			newCloudprovider.Name, err = db.GenerateName(ctx, CloudproviderManager, nil, subAccount.Name)
			if err != nil {
				return err
			}
			return CloudproviderManager.TableSpec().Insert(ctx, &newCloudprovider)
		}()
		if err != nil {
			return nil, err
		}
		return &newCloudprovider, nil
	}()
	if err != nil {
		return nil, isNew, errors.Wrapf(err, "insert new cloudprovider")
	}

	db.OpsLog.LogEvent(newCloudprovider, db.ACT_CREATE, newCloudprovider.GetShortDesc(ctx), userCred)

	passwd, err := self.getPassword()
	if err != nil {
		return nil, isNew, err
	}

	newCloudprovider.savePassword(passwd)

	if len(subAccount.DefaultProjectId) == 0 && self.AutoCreateProject && len(self.ProjectId) == 0 {
		err = newCloudprovider.syncProject(ctx, userCred)
		if err != nil {
			return nil, isNew, errors.Wrapf(err, "syncProject")
		}
	}

	return newCloudprovider, isNew, nil
}

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
	q := VpcManager.Query().In("manager_id", subq).IsFalse("is_emulated")
	return q.CountWithError()
}

func (self *SCloudaccount) GetStorageCount() (int, error) {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	q := StorageManager.Query().In("manager_id", subq).IsFalse("is_emulated")
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
	vpcs := VpcManager.Query("id", "manager_id").SubQuery()
	q := RouteTableManager.Query()
	q = q.Join(vpcs, sqlchemy.Equals(q.Field("vpc_id"), vpcs.Field("id")))
	q = q.Filter(sqlchemy.In(vpcs.Field("manager_id"), subq))
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

func (self *SCloudaccount) GetEnvironment() string {
	return self.AccessUrl
}

func (self *SCloudaccount) getMoreDetails(ctx context.Context, out api.CloudaccountDetail) api.CloudaccountDetail {
	out.EipCount, _ = self.GetEipCount()
	out.VpcCount, _ = self.GetVpcCount()
	out.DiskCount, _ = self.GetDiskCount()
	out.HostCount, _ = self.GetHostCount()
	out.GuestCount, _ = self.GetGuestCount()
	out.StorageCount, _ = self.GetStorageCount()
	out.ProviderCount, _ = self.GetProviderCount()
	out.RoutetableCount, _ = self.GetRoutetableCount()
	out.StoragecacheCount, _ = self.GetStoragecacheCount()

	out.Projects = []api.ProviderProject{}
	for _, projectId := range self.getProjectIds() {
		if proj, _ := db.TenantCacheManager.FetchTenantById(ctx, projectId); proj != nil {
			project := api.ProviderProject{
				Tenant:   proj.Name,
				TenantId: proj.Id,
			}
			out.Projects = append(out.Projects, project)
		}
	}
	out.SyncStatus2 = self.getSyncStatus2()
	out.CloudEnv = self.GetCloudEnv()
	if len(self.ProjectId) > 0 {
		if proj, _ := db.TenantCacheManager.FetchTenantById(context.Background(), self.ProjectId); proj != nil {
			out.Tenant = proj.Name
		}
	}
	return out
}

func (manager *SCloudaccountManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudaccountDetail {
	rows := make([]api.CloudaccountDetail, len(objs))
	stdRows := manager.SEnabledStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	pmRows := manager.SProjectMappingResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	proxySettings := make(map[string]proxy.SProxySetting)
	{
		proxySettingIds := make([]string, len(objs))
		for i := range objs {
			proxySettingId := objs[i].(*SCloudaccount).ProxySettingId
			if !utils.IsInStringArray(proxySettingId, proxySettingIds) {
				proxySettingIds = append(proxySettingIds, proxySettingId)
			}
		}
		if err := db.FetchStandaloneObjectsByIds(
			proxy.ProxySettingManager,
			proxySettingIds,
			&proxySettings,
		); err != nil {
			log.Errorf("FetchStandaloneObjectsByIds (%s) fail %s",
				proxy.ProxySettingManager.KeywordPlural(), err)
			return rows
		}
	}
	for i := range rows {
		account := objs[i].(*SCloudaccount)
		detail := api.CloudaccountDetail{
			EnabledStatusInfrasResourceBaseDetails: stdRows[i],
			ProjectMappingResourceInfo:             pmRows[i],
		}
		if proxySetting, ok := proxySettings[account.ProxySettingId]; ok {
			detail.ProxySetting.Id = proxySetting.Id
			detail.ProxySetting.Name = proxySetting.Name
			detail.ProxySetting.HTTPProxy = proxySetting.HTTPProxy
			detail.ProxySetting.HTTPSProxy = proxySetting.HTTPSProxy
			detail.ProxySetting.NoProxy = proxySetting.NoProxy
		}
		rows[i] = account.getMoreDetails(ctx, detail)
	}

	return rows
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

		err := CloudaccountManager.TableSpec().Insert(context.Background(), &account)
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

func (manager *SCloudaccountManager) initializePublicScope() error {
	accounts := []SCloudaccount{}
	q := manager.Query().IsFalse("is_public").Equals("public_scope", "system")
	err := db.FetchModelObjects(manager, q, &accounts)
	if err != nil {
		log.Errorf("fetch all clound account fail %v", err)
		return err
	}
	for i := 0; i < len(accounts); i++ {
		account := &accounts[i]
		_, err = db.Update(account, func() error {
			switch account.ShareMode {
			case api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN:
				account.PublicScope = string(rbacutils.ScopeNone)
				account.IsPublic = false
			case api.CLOUD_ACCOUNT_SHARE_MODE_PROVIDER_DOMAIN:
				account.PublicScope = string(rbacutils.ScopeSystem)
				account.IsPublic = true
			case api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM:
				account.PublicScope = string(rbacutils.ScopeSystem)
				account.IsPublic = true
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (manager *SCloudaccountManager) initializeVMWareAccountId() error {
	// init accountid
	q := manager.Query().Equals("provider", api.CLOUD_PROVIDER_VMWARE)
	cloudaccounts := make([]SCloudaccount, 0)
	err := db.FetchModelObjects(manager, q, &cloudaccounts)
	if err != nil {
		return errors.Wrap(err, "fetch vmware cloudaccount fail")
	}
	for i := range cloudaccounts {
		account := cloudaccounts[i]
		if len(account.AccountId) != 0 && account.Account != account.AccountId {
			continue
		}
		url, err := url.Parse(account.AccessUrl)
		if err != nil {
			return errors.Wrapf(err, "parse vmware account's accessurl %s", account.AccessUrl)
		}
		hostPort := url.Host
		if i := strings.IndexByte(hostPort, ':'); i < 0 {
			hostPort = fmt.Sprintf("%s:%d", hostPort, 443)
		}
		_, err = db.Update(&account, func() error {
			account.AccountId = fmt.Sprintf("%s@%s", account.Account, hostPort)
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "db.Update for account")
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
		return errors.Wrap(err, "initializeBrand")
	}
	err = manager.initializeShareMode()
	if err != nil {
		return errors.Wrap(err, "initializeShareMode")
	}
	err = manager.initializeVMWareAccountId()
	if err != nil {
		return errors.Wrap(err, "initializeVMWareAccountId")
	}
	err = manager.initializePublicScope()
	if err != nil {
		return errors.Wrap(err, "initializePublicScope")
	}

	return nil
}

func (self *SCloudaccount) GetBalance() (float64, error) {
	return self.Balance, nil
}

func (self *SCloudaccount) GetDetailsBalance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	balance, err := self.GetBalance()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewFloat64(balance), "balance")
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

func (self *SCloudaccount) GetVCenterAccessInfo(privateId string) (vcenter.SVCenterAccessInfo, error) {
	info := vcenter.SVCenterAccessInfo{}

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

// +onecloud:swagger-gen-ignore
func (account *SCloudaccount) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeDomainOwnerInput) (jsonutils.JSONObject, error) {
	return nil, errors.Wrap(httperrors.ErrForbidden, "can't change domain owner of cloudaccount, use PerformChangeProject instead")
}

func (self *SCloudaccount) PerformChangeProject(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeProjectOwnerInput) (jsonutils.JSONObject, error) {
	if self.IsShared() {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "cannot change owner when shared!")
	}

	project := input.ProjectId

	tenant, err := db.TenantCacheManager.FetchTenantByIdOrName(ctx, project)
	if err != nil {
		return nil, httperrors.NewNotFoundError("project %s not found", project)
	}

	if tenant.Id == self.ProjectId {
		return nil, nil
	}

	providers := self.GetCloudproviders()
	if len(self.ProjectId) > 0 {
		if len(providers) > 0 {
			for i := range providers {
				if providers[i].ProjectId != self.ProjectId {
					return nil, errors.Wrap(httperrors.ErrConflict, "cloudproviders' project is different from cloudaccount's")
				}
			}
		}
	}

	if tenant.DomainId != self.DomainId {
		// do change domainId
		input2 := apis.PerformChangeDomainOwnerInput{}
		input2.ProjectDomainId = tenant.DomainId
		_, err := self.SEnabledStatusInfrasResourceBase.PerformChangeOwner(ctx, userCred, query, input2)
		if err != nil {
			return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.PerformChangeOwner")
		}
	}

	// save project_id change
	diff, err := db.Update(self, func() error {
		self.ProjectId = tenant.Id
		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "db.Update ProjectId")
	}

	db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)

	if len(providers) > 0 {
		for i := range providers {
			_, err := providers[i].PerformChangeProject(ctx, userCred, query, input)
			if err != nil {
				return nil, errors.Wrapf(err, "providers[i].PerformChangeProject %s(%s)", providers[i].Name, providers[i].Id)
			}
		}
	}

	return nil, nil
}

// 云账号列表
func (manager *SCloudaccountManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudaccountListInput,
) (*sqlchemy.SQuery, error) {
	accountArr := query.CloudaccountId
	if len(accountArr) > 0 {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(q.Field("id"), accountArr),
			sqlchemy.In(q.Field("name"), accountArr),
		))
	}

	q, err := manager.SEnabledStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager")
	}
	q, err = manager.SSyncableBaseResourceManager.ListItemFilter(ctx, q, userCred, query.SyncableBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSyncableBaseResourceManager.ListItemFilter")
	}

	if len(query.ProxySetting) > 0 {
		proxy, err := proxy.ProxySettingManager.FetchByIdOrName(nil, query.ProxySetting)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("proxy_setting", query.ProxySetting)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Equals("proxy_setting_id", proxy.GetId())
	}

	managerStr := query.CloudproviderId
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

	cloudEnvStr := query.CloudEnv
	if cloudEnvStr == api.CLOUD_ENV_PUBLIC_CLOUD {
		q = q.IsTrue("is_public_cloud").IsFalse("is_on_premise")
	}

	if cloudEnvStr == api.CLOUD_ENV_PRIVATE_CLOUD {
		q = q.IsFalse("is_public_cloud").IsFalse("is_on_premise")
	}

	if cloudEnvStr == api.CLOUD_ENV_ON_PREMISE {
		q = q.IsTrue("is_on_premise").IsFalse("is_public_cloud")
	}

	capabilities := query.Capability
	if len(capabilities) > 0 {
		cloudproviders := CloudproviderManager.Query().SubQuery()
		cloudprovidercapabilities := CloudproviderCapabilityManager.Query().SubQuery()
		subq := cloudproviders.Query(cloudproviders.Field("cloudaccount_id"))
		subq = subq.Join(cloudprovidercapabilities, sqlchemy.Equals(cloudprovidercapabilities.Field("cloudprovider_id"), cloudproviders.Field("id")))
		subq = subq.Filter(sqlchemy.In(cloudprovidercapabilities.Field("capability"), capabilities))
		q = q.In("id", subq.SubQuery())
	}

	if len(query.HealthStatus) > 0 {
		q = q.In("health_status", query.HealthStatus)
	}
	if len(query.ShareMode) > 0 {
		q = q.In("share_mode", query.ShareMode)
	}
	if len(query.Providers) > 0 {
		q = q.In("provider", query.Providers)
	}
	if len(query.Brands) > 0 {
		q = q.In("brand", query.Brands)
	}

	return q, nil
}

func (manager *SCloudaccountManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "account":
		q = q.AppendField(q.Field("name").Label("account")).Distinct()
		return q, nil
	}
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SCloudaccountManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudaccountListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
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
	if account.ErrorCount != 0 {
		_, err := db.UpdateWithLock(ctx, account, func() error {
			account.ErrorCount = 0
			return nil
		})
		if err != nil {
			return err
		}
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
	if account.ProbeAt.IsZero() {
		return true
	}
	// last sync is long time ago
	if time.Now().Sub(account.ProbeAt) > time.Duration(options.Options.DisconnectedCloudAccountRetryProbeIntervalHours)*time.Hour {
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

func (self *SCloudaccount) CanSync() bool {
	if self.SyncStatus == api.CLOUD_PROVIDER_SYNC_STATUS_QUEUED || self.SyncStatus == api.CLOUD_PROVIDER_SYNC_STATUS_SYNCING || self.getSyncStatus2() == api.CLOUD_PROVIDER_SYNC_STATUS_SYNCING {
		if self.LastSync.IsZero() || time.Now().Sub(self.LastSync) > time.Minute*30 {
			return true
		}
		return false
	}
	return true
}

func (manager *SCloudaccountManager) AutoSyncCloudaccountStatusTask(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
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
		if accounts[i].GetEnabled() && accounts[i].shouldProbeStatus() && accounts[i].CanSync() {
			id, name, account := accounts[i].Id, accounts[i].Name, &accounts[i]
			cloudaccountProbeMutex.Lock()
			if _, ok := cloudaccountProbe[id]; ok {
				cloudaccountProbeMutex.Unlock()
				continue
			}
			cloudaccountProbe[id] = struct{}{}
			cloudaccountProbeMutex.Unlock()
			RunSyncCloudAccountTask(ctx, func() {
				defer func() {
					cloudaccountProbeMutex.Lock()
					defer cloudaccountProbeMutex.Unlock()
					delete(cloudaccountProbe, id)
				}()
				log.Debugf("syncAccountStatus %s %s", id, name)
				err := account.syncAccountStatus(ctx, userCred)
				if err != nil {
					log.Errorf("unable to syncAccountStatus for cloudaccount %s: %s", account.Id, err.Error())
				}
				account.markEndSync(userCred, true)
			})
		}
	}
}

func (account *SCloudaccount) probeAccountStatus(ctx context.Context, userCred mcclient.TokenCredential) ([]cloudprovider.SSubAccount, error) {
	manager, err := account.getProviderInternal(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "account.getProviderInternal")
	}
	balance, status, err := manager.GetBalance()
	if err != nil {
		switch err {
		case cloudprovider.ErrNotSupported:
			status = api.CLOUD_PROVIDER_HEALTH_NORMAL
		case cloudprovider.ErrNoBalancePermission:
			status = api.CLOUD_PROVIDER_HEALTH_NO_PERMISSION
		default:
			status = api.CLOUD_PROVIDER_HEALTH_UNKNOWN
			if account.Status != status {
				logclient.AddSimpleActionLog(account, logclient.ACT_PROBE, errors.Wrapf(err, "GetBalance"), userCred, false)
			}
		}
	}
	version := manager.GetVersion()
	sysInfo, err := manager.GetSysInfo()
	if err != nil {
		return nil, errors.Wrap(err, "manager.GetSysInfo")
	}
	iamLoginUrl := manager.GetIamLoginUrl()
	factory := manager.GetFactory()
	diff, err := db.Update(account, func() error {
		isPublic := factory.IsPublicCloud()
		account.IsPublicCloud = tristate.NewFromBool(isPublic)
		account.IsOnPremise = factory.IsOnPremise()
		account.Balance = balance
		if !options.Options.CloudaccountHealthStatusCheck {
			status = api.CLOUD_PROVIDER_HEALTH_NORMAL
		}
		// if len(account.AccountId) == 0 || account.AccountId != manager.GetAccountId() {
		account.AccountId = manager.GetAccountId()
		// }
		account.HealthStatus = status
		account.ProbeAt = timeutils.UtcNow()
		account.Version = version
		account.Sysinfo = sysInfo
		account.IamLoginUrl = iamLoginUrl
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

func (self *SCloudaccount) setSubAccountStatus() error {
	if self.SubAccounts == nil || (len(self.SubAccounts.Accounts) == 0 && len(self.SubAccounts.Cloudregions) == 0) {
		return nil
	}
	accounts := []string{}
	accountNames := []string{}
	regionIds := []string{}
	for _, account := range self.SubAccounts.Accounts {
		if len(account.Account) > 0 {
			accounts = append(accounts, account.Account)
		} else if len(account.Name) > 0 {
			accountNames = append(accountNames, account.Name)
		}
	}
	for _, region := range self.SubAccounts.Cloudregions {
		if len(region.Id) > 0 && !strings.HasSuffix(region.Id, "/") {
			regionIds = append(regionIds, region.Id)
		}
	}

	providers := self.GetCloudproviders()
	enabledIds := []string{}
	if len(accounts) > 0 || len(accountNames) > 0 {
		for i := range providers {
			if !utils.IsInStringArray(providers[i].Name, accountNames) && !utils.IsInStringArray(providers[i].Account, accounts) {
				_, err := db.Update(&providers[i], func() error {
					providers[i].Enabled = tristate.False
					return nil
				})
				if err != nil {
					log.Errorf("db.Update %v", err)
				}
			} else {
				enabledIds = append(enabledIds, providers[i].Id)
			}
		}
	}

	if len(regionIds) > 0 {
		q := CloudproviderRegionManager.Query()
		providerQ := CloudproviderManager.Query().SubQuery()
		q = q.Join(providerQ, sqlchemy.Equals(providerQ.Field("id"), q.Field("cloudprovider_id")))
		if len(enabledIds) > 0 {
			q = q.Filter(
				sqlchemy.In(providerQ.Field("id"), enabledIds),
			)
		}
		accountQ := CloudaccountManager.Query().Equals("id", self.Id).SubQuery()
		q = q.Join(accountQ, sqlchemy.Equals(providerQ.Field("cloudaccount_id"), accountQ.Field("id")))
		regionQ := CloudregionManager.Query().SubQuery()
		q = q.Join(regionQ, sqlchemy.Equals(regionQ.Field("id"), q.Field("cloudregion_id"))).Filter(
			sqlchemy.NotIn(regionQ.Field("external_id"), regionIds),
		)

		cpcrs := []SCloudproviderregion{}
		err := db.FetchModelObjects(CloudproviderRegionManager, q, &cpcrs)
		if err != nil {
			return errors.Wrapf(err, "db.FetchModelObjects")
		}

		for i := range cpcrs {
			db.Update(&cpcrs[i], func() error {
				cpcrs[i].Enabled = false
				return nil
			})
		}
	}

	_, err := db.Update(self, func() error {
		self.SubAccounts = nil
		return nil
	})

	return err
}

func (account *SCloudaccount) syncAccountStatus(ctx context.Context, userCred mcclient.TokenCredential) error {
	account.MarkSyncing(userCred, true)
	subaccounts, err := account.probeAccountStatus(ctx, userCred)
	if err != nil {
		account.markAllProvidersDicconnected(ctx, userCred)
		account.markAccountDiscconected(ctx, userCred)
		return errors.Wrap(err, "account.probeAccountStatus")
	}
	account.markAccountConnected(ctx, userCred)
	providers := account.importAllSubaccounts(ctx, userCred, subaccounts)
	for i := range providers {
		if providers[i].GetEnabled() {
			_, err := providers[i].prepareCloudproviderRegions(ctx, userCred)
			if err != nil {
				return errors.Wrapf(err, "prepareCloudproviderRegions for provider %s", providers[i].Name)
			}
		}
	}
	return account.setSubAccountStatus()
}

var (
	cloudaccountPendingSyncs      = map[string]struct{}{}
	cloudaccountPendingSyncsMutex = &sync.Mutex{}

	cloudaccountProbe      = map[string]struct{}{}
	cloudaccountProbeMutex = &sync.Mutex{}
)

func (account *SCloudaccount) SubmitSyncAccountTask(ctx context.Context, userCred mcclient.TokenCredential, waitChan chan error) {
	cloudaccountPendingSyncsMutex.Lock()
	defer cloudaccountPendingSyncsMutex.Unlock()
	if _, ok := cloudaccountPendingSyncs[account.Id]; ok {
		if waitChan != nil {
			go func() {
				waitChan <- errors.Wrap(httperrors.ErrConflict, "an active cloudaccount sync task is running, early return with conflict error")
			}()
		}
		return
	}
	cloudaccountPendingSyncs[account.Id] = struct{}{}

	RunSyncCloudAccountTask(ctx, func() {
		defer func() {
			cloudaccountPendingSyncsMutex.Lock()
			defer cloudaccountPendingSyncsMutex.Unlock()
			delete(cloudaccountPendingSyncs, account.Id)
		}()
		log.Debugf("syncAccountStatus %s %s", account.Id, account.Name)
		err := account.syncAccountStatus(ctx, userCred)
		if waitChan != nil {
			if err != nil {
				err = errors.Wrap(err, "account.syncAccountStatus")
			}
			waitChan <- err
		}
	})
}

func (account *SCloudaccount) SyncCallSyncAccountTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	waitChan := make(chan error)
	account.SubmitSyncAccountTask(ctx, userCred, waitChan)
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
	projects, err := self.GetExternalProjects()
	if err != nil {
		return errors.Wrap(err, "GetExternalProjects")
	}
	for i := range projects {
		err = projects[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "project %s Delete", projects[i].Id)
		}
	}
	caches, err := self.GetDnsZoneCaches()
	if err != nil {
		return errors.Wrapf(err, "GetDnsZoneCaches")
	}
	for i := range caches {
		err = caches[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "dns zone cache %s delete", caches[i].Id)
		}
	}
	return self.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
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

func (account *SCloudaccount) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudaccountPerformPublicInput) (jsonutils.JSONObject, error) {
	if !account.CanSync() {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "cannot public in sync")
	}

	switch input.ShareMode {
	case api.CLOUD_ACCOUNT_SHARE_MODE_PROVIDER_DOMAIN:
		if len(input.SharedDomainIds) == 0 {
			input.Scope = string(rbacutils.ScopeSystem)
		} else {
			input.Scope = string(rbacutils.ScopeDomain)
			providers := account.GetCloudproviders()
			for i := range providers {
				if !utils.IsInStringArray(providers[i].DomainId, input.SharedDomainIds) && providers[i].DomainId != account.DomainId {
					log.Warningf("provider's domainId %s is outside of list of shared domains", providers[i].DomainId)
					input.SharedDomainIds = append(input.SharedDomainIds, providers[i].DomainId)
				}
			}
		}
	case api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM:
		if len(input.SharedDomainIds) == 0 {
			input.Scope = string(rbacutils.ScopeSystem)
		} else {
			input.Scope = string(rbacutils.ScopeDomain)
		}
	default:
		return nil, errors.Wrap(httperrors.ErrInputParameter, "share_mode cannot be account_domain")
	}

	_, err := account.SInfrasResourceBase.PerformPublic(ctx, userCred, query, input.PerformPublicDomainInput)
	if err != nil {
		return nil, errors.Wrap(err, "SInfrasResourceBase.PerformPublic")
	}
	// scope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), account.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "public")
	// if scope != rbacutils.ScopeSystem {
	// 	return nil, httperrors.NewForbiddenError("not enough privilege")
	// }

	err = account.setShareMode(userCred, input.ShareMode)
	if err != nil {
		return nil, errors.Wrap(err, "account.setShareMode")
	}

	syncRange := &SSyncRange{
		SyncRangeInput: api.SyncRangeInput{
			FullSync: true,
		},
	}
	account.StartSyncCloudProviderInfoTask(ctx, userCred, syncRange, "")
	return nil, nil
}

func (account *SCloudaccount) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) (jsonutils.JSONObject, error) {
	if !account.CanSync() {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "cannot private in sync")
	}

	providers := account.GetCloudproviders()
	for i := range providers {
		if providers[i].DomainId != account.DomainId {
			return nil, httperrors.NewConflictError("provider is shared outside of domain")
		}
	}
	_, err := account.SInfrasResourceBase.PerformPrivate(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SInfrasResourceBase.PerformPrivate")
	}
	// scope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), account.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "private")
	// if scope != rbacutils.ScopeSystem {
	// 	return nil, httperrors.NewForbiddenError("not enough privilege")
	// }

	err = account.setShareMode(userCred, api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN)
	if err != nil {
		return nil, errors.Wrap(err, "account.setShareMode")
	}

	syncRange := &SSyncRange{
		SyncRangeInput: api.SyncRangeInput{
			FullSync: true,
		},
	}
	account.StartSyncCloudProviderInfoTask(ctx, userCred, syncRange, "")

	return nil, nil
}

// Deprecated
func (account *SCloudaccount) PerformShareMode(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudaccountShareModeInput) (jsonutils.JSONObject, error) {

	err := input.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "CloudaccountShareModeInput.Validate")
	}
	if account.ShareMode == input.ShareMode {
		return nil, nil
	}

	if input.ShareMode == api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN {
		return account.PerformPrivate(ctx, userCred, query, apis.PerformPrivateInput{})
	} else {
		input2 := api.CloudaccountPerformPublicInput{
			ShareMode: input.ShareMode,
			PerformPublicDomainInput: apis.PerformPublicDomainInput{
				Scope: string(rbacutils.ScopeSystem),
			},
		}
		return account.PerformPublic(ctx, userCred, query, input2)
	}

	/*scope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), account.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "share-mode")
	if scope != rbacutils.ScopeSystem {
		return nil, httperrors.NewForbiddenError("not enough privilege")
	}

	err = account.setShareMode(userCred, input.ShareMode)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	return nil, nil*/
}

func (manager *SCloudaccountManager) filterByDomainId(q *sqlchemy.SQuery, domainId string) *sqlchemy.SQuery {
	subq := db.SharedResourceManager.Query("resource_id")
	subq = subq.Equals("resource_type", manager.Keyword())
	subq = subq.Equals("target_project_id", domainId)
	subq = subq.Equals("target_type", db.SharedTargetDomain)

	cloudproviders := CloudproviderManager.Query().SubQuery()
	q = q.LeftJoin(cloudproviders, sqlchemy.Equals(
		q.Field("id"),
		cloudproviders.Field("cloudaccount_id"),
	))

	q = q.Distinct()

	q = q.Filter(sqlchemy.OR(
		// share_mode=account_domain/private
		sqlchemy.Equals(q.Field("domain_id"), domainId),
		// share_mode=provider_domain/public_scope=domain
		// share_mode=provider_domain/public_scope=system
		sqlchemy.AND(
			sqlchemy.Equals(q.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_PROVIDER_DOMAIN),
			sqlchemy.Equals(cloudproviders.Field("domain_id"), domainId),
		),
		// share_mode=system/public_scope=domain
		sqlchemy.AND(
			// sqlchemy.Equals(q.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM),
			sqlchemy.In(q.Field("id"), subq.SubQuery()),
			sqlchemy.IsTrue(q.Field("is_public")),
			sqlchemy.Equals(q.Field("public_scope"), rbacutils.ScopeDomain),
		),
		// share_mode=system/public_scope=system
		sqlchemy.AND(
			// sqlchemy.Equals(q.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM),
			sqlchemy.IsTrue(q.Field("is_public")),
			sqlchemy.Equals(q.Field("public_scope"), rbacutils.ScopeSystem),
		),
	))

	return q
}

func (manager *SCloudaccountManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacutils.ScopeProject, rbacutils.ScopeDomain:
			if len(owner.GetProjectDomainId()) > 0 {
				q = manager.filterByDomainId(q, owner.GetProjectDomainId())
				/*cloudproviders := CloudproviderManager.Query().SubQuery()
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
				))*/
			}
		}
	}
	return q
}

func (manager *SCloudaccountManager) getBrandsOfProvider(provider string) ([]string, error) {
	q := manager.Query().Equals("provider", provider)
	cloudaccounts := make([]SCloudaccount, 0)
	err := db.FetchModelObjects(manager, q, &cloudaccounts)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	ret := make([]string, 0)
	for i := range cloudaccounts {
		if cloudaccounts[i].IsAvailable() && !utils.IsInStringArray(cloudaccounts[i].Brand, ret) {
			ret = append(ret, cloudaccounts[i].Brand)
		}
	}
	return ret, nil
}

func guessBrandForHypervisor(hypervisor string) string {
	if hypervisor == "" {
		return api.HYPERVISOR_KVM
	}
	driver := GetDriver(hypervisor)
	if driver == nil {
		log.Errorf("guestBrandFromHypervisor: fail to find driver for hypervisor %s", hypervisor)
		return ""
	}
	provider := driver.GetProvider()
	if len(provider) == 0 {
		log.Errorf("guestBrandFromHypervisor: fail to find provider for hypervisor %s", hypervisor)
		return ""
	}
	brands, err := CloudaccountManager.getBrandsOfProvider(provider)
	if err != nil {
		log.Errorf("guestBrandFromHypervisor: fail to find brands for hypervisor %s", hypervisor)
		return ""
	}
	if len(brands) != 1 {
		log.Errorf("guestBrandFromHypervisor: find mistached number of brands for hypervisor %s %s", hypervisor, brands)
		return ""
	}
	return brands[0]
}

func (account *SCloudaccount) PerformSyncSkus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudaccountSyncSkusInput) (jsonutils.JSONObject, error) {
	if !account.GetEnabled() {
		return nil, httperrors.NewInvalidStatusError("Account disabled")
	}

	if len(input.Resource) == 0 {
		return nil, httperrors.NewMissingParameterError("resource")
	}

	params := jsonutils.NewDict()

	if !utils.IsInStringArray(input.Resource, []string{
		ServerSkuManager.Keyword(),
		ElasticcacheSkuManager.Keyword(),
		DBInstanceSkuManager.Keyword(),
		NatSkuManager.Keyword(),
		NasSkuManager.Keyword(),
	}) {
		return nil, httperrors.NewInputParameterError("invalid resource %s", input.Resource)
	}
	params.Add(jsonutils.NewString(input.Resource), "resource")

	if len(input.CloudregionId) > 0 {
		_, err := validators.ValidateModel(userCred, CloudregionManager, &input.CloudregionId)
		if err != nil {
			return nil, err
		}
		params.Add(jsonutils.NewString(input.CloudregionId), "cloudregion_id")
	}
	if len(input.CloudproviderId) > 0 {
		_, err := validators.ValidateModel(userCred, CloudproviderManager, &input.CloudproviderId)
		if err != nil {
			return nil, err
		}
		params.Add(jsonutils.NewString(input.CloudproviderId), "cloudprovider_id")
	}

	if account.CanSync() || input.Force {
		task, err := taskman.TaskManager.NewTask(ctx, "CloudAccountSyncSkusTask", account, userCred, params, "", "", nil)
		if err != nil {
			return nil, errors.Wrapf(err, "CloudAccountSyncSkusTask")
		}
		task.ScheduleRun(nil)
	}

	return nil, nil
}

func (self *SCloudaccount) GetExternalProjects() ([]SExternalProject, error) {
	projects := []SExternalProject{}
	q := ExternalProjectManager.Query().Equals("cloudaccount_id", self.Id)
	err := db.FetchModelObjects(ExternalProjectManager, q, &projects)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return projects, nil
}

func (self *SCloudaccount) GetExternalProjectsByProjectIdOrName(projectId, name string) ([]SExternalProject, error) {
	projects := []SExternalProject{}
	q := ExternalProjectManager.Query().Equals("cloudaccount_id", self.Id)
	q = q.Filter(
		sqlchemy.OR(
			sqlchemy.Equals(q.Field("name"), name),
			sqlchemy.Equals(q.Field("tenant_id"), projectId),
		),
	)
	err := db.FetchModelObjects(ExternalProjectManager, q, &projects)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return projects, nil
}

func (manager *SCloudaccountManager) queryCloudAccountByCapability(region *SCloudregion, zone *SZone, domainId string, enabled tristate.TriState, capability string) *sqlchemy.SQuery {
	providers := CloudproviderManager.Query().SubQuery()
	q := manager.Query()
	q = q.Join(providers, sqlchemy.Equals(q.Field("id"), providers.Field("cloudaccount_id")))
	cloudproviderCapabilities := CloudproviderCapabilityManager.Query().SubQuery()
	q = q.Join(cloudproviderCapabilities, sqlchemy.Equals(providers.Field("id"), cloudproviderCapabilities.Field("cloudprovider_id")))
	q = q.Filter(sqlchemy.Equals(cloudproviderCapabilities.Field("capability"), capability))
	if zone != nil {
		region, _ = zone.GetRegion()
	}
	if region != nil {
		providerregions := CloudproviderRegionManager.Query().SubQuery()
		q = q.Join(providerregions, sqlchemy.Equals(providers.Field("id"), providerregions.Field("cloudprovider_id")))
		q = q.Filter(sqlchemy.Equals(providerregions.Field("cloudregion_id"), region.Id))
	}
	if len(domainId) > 0 {
		q = manager.filterByDomainId(q, domainId)
	}
	return q
}

type sBrandCapability struct {
	Brand      string
	Enabled    bool
	Capability string
}

func (manager *SCloudaccountManager) getBrandsOfCapability(region *SCloudregion, zone *SZone, domainId string) ([]sBrandCapability, error) {
	accounts := manager.Query("id", "enabled", "brand")
	if len(domainId) > 0 {
		accounts = manager.filterByDomainId(accounts, domainId)
	}

	accountSQ := accounts.SubQuery()
	providers := CloudproviderManager.Query().SubQuery()
	q := CloudproviderCapabilityManager.Query("capability")

	q.AppendField(accountSQ.Field("enabled"))
	q.AppendField(accountSQ.Field("brand"))

	q = q.Join(providers, sqlchemy.Equals(q.Field("cloudprovider_id"), providers.Field("id")))
	q = q.Join(accountSQ, sqlchemy.Equals(providers.Field("cloudaccount_id"), accountSQ.Field("id")))

	if zone != nil {
		var err error
		region, err = zone.GetRegion()
		if err != nil {
			return nil, errors.Wrapf(err, "GetRegion")
		}
	}
	if region != nil {
		providerregions := CloudproviderRegionManager.Query().SubQuery()
		q = q.Join(providerregions, sqlchemy.Equals(q.Field("cloudprovider_id"), providerregions.Field("cloudprovider_id"))).Filter(
			sqlchemy.Equals(providerregions.Field("cloudregion_id"), region.Id),
		)
	}

	q = q.Distinct()

	result := []sBrandCapability{}
	err := q.All(&result)
	if err != nil {
		return nil, errors.Wrapf(err, "q.All")
	}
	return result, nil
}

func (account *SCloudaccount) getAccountShareInfo() apis.SAccountShareInfo {
	return apis.SAccountShareInfo{
		ShareMode:     account.ShareMode,
		IsPublic:      account.IsPublic,
		PublicScope:   rbacutils.String2Scope(account.PublicScope),
		SharedDomains: account.GetSharedDomains(),
	}
}

func (manager *SCloudaccountManager) totalCount(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) int {
	q := manager.Query()
	switch scope {
	case rbacutils.ScopeProject, rbacutils.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	}
	cnt, _ := q.CountWithError()
	return cnt
}

func (account *SCloudaccount) GetUsages() []db.IUsage {
	if account.Deleted {
		return nil
	}
	usage := SDomainQuota{Cloudaccount: 1}
	usage.SetKeys(quotas.SBaseDomainQuotaKeys{DomainId: account.DomainId})
	return []db.IUsage{
		&usage,
	}
}

func GetAvailableExternalProject(local *db.STenant, projects []SExternalProject) *SExternalProject {
	var ret *SExternalProject = nil
	for i := 0; i < len(projects); i++ {
		if projects[i].Status == api.EXTERNAL_PROJECT_STATUS_AVAILABLE {
			if projects[i].ProjectId == local.Id {
				return &projects[i]
			}
			if projects[i].Name == local.Name {
				ret = &projects[i]
			}
		}
	}
	return ret
}

// 若本地项目映射了多个云上项目，则在云上随机找一个项目
// 若本地项目没有映射云上任何项目，则在云上新建一个同名项目
// 若本地项目a映射云上项目b，但b项目不可用,则看云上是否有a项目，有则直接使用,若没有则在云上创建a-1, a-2类似项目
func (self *SCloudaccount) SyncProject(ctx context.Context, userCred mcclient.TokenCredential, projectId string) (string, error) {
	lockman.LockRawObject(ctx, "projects", self.Id)
	defer lockman.ReleaseRawObject(ctx, "projects", self.Id)

	provider, err := self.GetProvider(ctx)
	if err != nil {
		return "", errors.Wrap(err, "GetProvider")
	}

	if !cloudprovider.IsSupportProject(provider) {
		return "", nil
	}

	project, err := db.TenantCacheManager.FetchTenantById(ctx, projectId)
	if err != nil {
		return "", errors.Wrapf(err, "FetchTenantById(%s)", projectId)
	}

	projects, err := self.GetExternalProjectsByProjectIdOrName(projectId, project.Name)
	if err != nil {
		return "", errors.Wrapf(err, "GetExternalProjectsByProjectIdOrName(%s,%s)", projectId, project.Name)
	}

	extProj := GetAvailableExternalProject(project, projects)
	if extProj != nil {
		return extProj.ExternalId, nil
	}

	retry := 1
	if len(projects) > 0 {
		retry = 10
	}

	var iProject cloudprovider.ICloudProject = nil
	projectName := project.Name
	for i := 0; i < retry; i++ {
		iProject, err = provider.CreateIProject(projectName)
		if err == nil {
			break
		}
		projectName = fmt.Sprintf("%s-%d", project.Name, i)
	}
	if err != nil {
		if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
			logclient.AddSimpleActionLog(self, logclient.ACT_CREATE, err, userCred, false)
		}
		return "", errors.Wrapf(err, "CreateIProject(%s)", projectName)
	}

	extProj, err = ExternalProjectManager.newFromCloudProject(ctx, userCred, self, project, iProject)
	if err != nil {
		return "", errors.Wrap(err, "newFromCloudProject")
	}

	return extProj.ExternalId, nil
}

// 获取Azure Enrollment Accounts
func (self *SCloudaccount) GetDetailsEnrollmentAccounts(ctx context.Context, userCred mcclient.TokenCredential, query api.EnrollmentAccountQuery) ([]cloudprovider.SEnrollmentAccount, error) {
	if self.Provider != api.CLOUD_PROVIDER_AZURE {
		return nil, httperrors.NewNotSupportedError("%s not support", self.Provider)
	}
	provider, err := self.GetProvider(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "GetProvider")
	}

	result, err := provider.GetEnrollmentAccounts()
	if err != nil {
		return nil, errors.Wrap(err, "GetEnrollmentAccounts")
	}

	return result, nil
}

// 创建Azure订阅
func (self *SCloudaccount) PerformCreateSubscription(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SubscriptonCreateInput) (jsonutils.JSONObject, error) {
	if self.Provider != api.CLOUD_PROVIDER_AZURE {
		return nil, httperrors.NewNotSupportedError("%s not support create subscription", self.Provider)
	}
	if len(input.Name) == 0 {
		return nil, httperrors.NewMissingParameterError("name")
	}
	if len(input.EnrollmentAccountId) == 0 {
		return nil, httperrors.NewMissingParameterError("enrollment_account_id")
	}
	if len(input.OfferType) == 0 {
		return nil, httperrors.NewMissingParameterError("offer_type")
	}

	provider, err := self.GetProvider(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "GetProvider")
	}

	conf := cloudprovider.SubscriptionCreateInput{
		Name:                input.Name,
		EnrollmentAccountId: input.EnrollmentAccountId,
		OfferType:           input.OfferType,
	}

	err = provider.CreateSubscription(conf)
	if err != nil {
		return nil, errors.Wrap(err, "CreateSubscription")
	}

	syncRange := SSyncRange{}
	return nil, self.StartSyncCloudProviderInfoTask(ctx, userCred, &syncRange, "")
}

func (self *SCloudaccount) GetDnsZoneCaches() ([]SDnsZoneCache, error) {
	caches := []SDnsZoneCache{}
	q := DnsZoneCacheManager.Query().Equals("cloudaccount_id", self.Id)
	err := db.FetchModelObjects(DnsZoneCacheManager, q, &caches)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return caches, nil
}

func (self *SCloudaccount) SyncDnsZones(ctx context.Context, userCred mcclient.TokenCredential, dnsZones []cloudprovider.ICloudDnsZone) ([]SDnsZone, []cloudprovider.ICloudDnsZone, compare.SyncResult) {
	lockman.LockRawObject(ctx, self.Keyword(), fmt.Sprintf("%s-dnszone", self.Id))
	defer lockman.ReleaseRawObject(ctx, self.Keyword(), fmt.Sprintf("%s-dnszone", self.Id))

	result := compare.SyncResult{}

	localZones := []SDnsZone{}
	remoteZones := []cloudprovider.ICloudDnsZone{}

	dbZones, err := self.GetDnsZoneCaches()
	if err != nil {
		result.Error(errors.Wrapf(err, "GetDnsZoneCaches"))
		return nil, nil, result
	}

	removed := make([]SDnsZoneCache, 0)
	commondb := make([]SDnsZoneCache, 0)
	commonext := make([]cloudprovider.ICloudDnsZone, 0)
	added := make([]cloudprovider.ICloudDnsZone, 0)

	err = compare.CompareSets(dbZones, dnsZones, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return nil, nil, result
	}

	for i := 0; i < len(removed); i += 1 {
		if len(removed[i].ExternalId) > 0 {
			err = removed[i].syncRemove(ctx, userCred)
			if err != nil {
				result.DeleteError(err)
				continue
			}
			result.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudDnsZone(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(errors.Wrapf(err, "SyncWithCloudDnsZone"))
			continue
		}
		zone, err := commondb[i].GetDnsZone()
		if err != nil {
			result.UpdateError(errors.Wrapf(err, "GetDnsZone"))
			continue
		}
		localZones = append(localZones, *zone)
		remoteZones = append(remoteZones, commonext[i])

		result.Update()
	}

	for i := 0; i < len(added); i += 1 {
		dnsZone, isNew, err := DnsZoneManager.newFromCloudDnsZone(ctx, userCred, added[i], self)
		if err != nil {
			result.AddError(err)
			continue
		}
		if !isNew {
			_, err = dnsZone.newCache(ctx, userCred, self.Id, added[i])
			if err != nil {
				result.AddError(errors.Wrapf(err, "newCache"))
			}
		}
		result.Add()
		localZones = append(localZones, *dnsZone)
		remoteZones = append(remoteZones, added[i])
	}

	return localZones, remoteZones, result
}

type SVs2Wire struct {
	WireId      string
	VsId        string
	Distributed bool
	Mac         string
	SyncTimes   int
}

var METADATA_EXT_HOST2WIRE_KEY = "ext:vmware:host2wire"

func (cd *SCloudaccount) SetHost2Wire(ctx context.Context, userCred mcclient.TokenCredential, hw map[string][]SVs2Wire) error {
	err := cd.SetMetadata(ctx, METADATA_EXT_HOST2WIRE_KEY, hw, userCred)
	if err != nil {
		return err
	}
	cd.vmwareHostWireCache = hw
	return nil
}

func (cd *SCloudaccount) GetHost2Wire(ctx context.Context, userCred mcclient.TokenCredential) (map[string][]SVs2Wire, error) {
	if cd.vmwareHostWireCache != nil {
		return cd.vmwareHostWireCache, nil
	}
	hwJson := cd.GetMetadataJson(ctx, METADATA_EXT_HOST2WIRE_KEY, userCred)
	if hwJson == nil {
		return nil, fmt.Errorf("The cloud account synchronization network may have failed, please check the operation log first and solve the synchronization network problem")
	}
	ret := make(map[string][]SVs2Wire)
	err := hwJson.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "unable unmarshal json %s to map[string][]SVs2Wire", hwJson.String())
	}
	cd.vmwareHostWireCache = ret
	return cd.vmwareHostWireCache, nil
}

// 绑定同步策略
func (self *SCloudaccount) PerformProjectMapping(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudaccountProjectMappingInput) (jsonutils.JSONObject, error) {
	if len(input.ProjectMappingId) > 0 {
		_, err := validators.ValidateModel(userCred, ProjectMappingManager, &input.ProjectMappingId)
		if err != nil {
			return nil, err
		}
		if len(self.ProjectMappingId) > 0 && self.ProjectMappingId != input.ProjectMappingId {
			return nil, httperrors.NewInputParameterError("account %s has aleady bind project mapping %s", self.Name, self.ProjectMappingId)
		}
	}
	// no changes
	if self.ProjectMappingId == input.ProjectMappingId {
		return nil, nil
	}
	_, err := db.Update(self, func() error {
		self.ProjectMappingId = input.ProjectMappingId
		return nil
	})
	if err != nil {
		return nil, err
	}
	return nil, refreshPmCaches()
}
