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
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	proxyapi "yunion.io/x/onecloud/pkg/apis/cloudcommon/proxy"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/proxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/util/logclient"
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
	AccessUrl string `width:"128" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`

	// 云账号
	Account string `width:"256" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`

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

	// 账户余额货币类型
	// enmu: CNY, USD
	Currency string `width:"5" charset:"utf8" nullable:"false" list:"user" create:"domain_optional" update:"domain"`

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

	// 是否根据云订阅自动在本地创建对应项目
	// example: false
	AutoCreateProjectForProvider bool `list:"domain" create:"domain_optional"`

	// 云API版本
	Version string `width:"64" charset:"ascii" nullable:"true" list:"domain"`

	// 云系统信息
	Sysinfo jsonutils.JSONObject `get:"domain"`

	// 品牌信息, 一般和provider相同
	Brand string `width:"64" charset:"utf8" nullable:"true" list:"domain" create:"optional"`

	// 额外信息
	Options *jsonutils.JSONDict `get:"domain" list:"domain" create:"domain_optional" update:"domain"`

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

	// 跳过部分资源同步
	SkipSyncResources *api.SkipSyncResources `length:"medium" get:"user" update:"domain" list:"user"`

	EnableAutoSyncResource tristate.TriState `get:"user" update:"domain" create:"optional" list:"user" default:"true"`

	// 云平台默认区域id
	RegionId string `width:"64" charset:"utf8" list:"user" create:"domain_optional"`
}

func (acnt *SCloudaccount) IsNotSkipSyncResource(res lockman.ILockedClass) bool {
	if acnt.SkipSyncResources != nil && utils.IsInStringArray(res.Keyword(), *acnt.SkipSyncResources) {
		return false
	}
	return true
}

func (acnt *SCloudaccount) GetCloudproviders() []SCloudprovider {
	return acnt.getCloudprovidersInternal(tristate.None)
}

func (acnt *SCloudaccount) IsAvailable() bool {
	if !acnt.GetEnabled() {
		return false
	}

	if !utils.IsInStringArray(acnt.HealthStatus, api.CLOUD_PROVIDER_VALID_HEALTH_STATUS) {
		return false
	}

	return true
}

func (acnt *SCloudaccount) GetEnabledCloudproviders() []SCloudprovider {
	return acnt.getCloudprovidersInternal(tristate.True)
}

func (acnt *SCloudaccount) getCloudprovidersInternal(enabled tristate.TriState) []SCloudprovider {
	cloudproviders := []SCloudprovider{}
	q := CloudproviderManager.Query().Equals("cloudaccount_id", acnt.Id)
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

func (acnt *SCloudaccount) ValidateDeleteCondition(ctx context.Context, info *api.CloudaccountDetail) error {
	if acnt.GetEnabled() {
		return httperrors.NewInvalidStatusError("account is enabled")
	}
	if gotypes.IsNil(info) {
		cnt, err := CloudaccountManager.TotalResourceCount([]string{acnt.Id})
		if err != nil {
			return errors.Wrapf(err, "TotalResourceCount")
		}
		info = &api.CloudaccountDetail{}
		info.SAccountUsage, _ = cnt[acnt.Id]
	}
	if acnt.Status == api.CLOUD_PROVIDER_CONNECTED && info.SyncCount > 0 {
		return httperrors.NewInvalidStatusError("account is not idle")
	}
	if info.EnabledProviderCount > 0 {
		return httperrors.NewInvalidStatusError("account has enabled provider")
	}
	return acnt.SEnabledStatusInfrasResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (acnt *SCloudaccount) enableAccountOnly(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	return acnt.SEnabledStatusInfrasResourceBase.PerformEnable(ctx, userCred, query, input)
}

func (acnt *SCloudaccount) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	if strings.Contains(acnt.Status, "delet") {
		return nil, httperrors.NewInvalidStatusError("Cannot enable deleting account")
	}
	_, err := acnt.enableAccountOnly(ctx, userCred, query, input)
	if err != nil {
		return nil, err
	}
	cloudproviders := acnt.GetCloudproviders()
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

func (acnt *SCloudaccount) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	_, err := acnt.SEnabledStatusInfrasResourceBase.PerformDisable(ctx, userCred, query, input)
	if err != nil {
		return nil, err
	}
	cloudproviders := acnt.GetCloudproviders()
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

func (acnt *SCloudaccount) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.CloudaccountUpdateInput,
) (api.CloudaccountUpdateInput, error) {
	if (input.Options != nil && input.Options.Length() > 0) || len(input.RemoveOptions) > 0 {
		var optionsJson *jsonutils.JSONDict
		if acnt.Options != nil {
			removes := make([]string, 0)
			if len(input.RemoveOptions) > 0 {
				removes = append(removes, input.RemoveOptions...)
			}
			optionsJson = acnt.Options.CopyExcludes(removes...)
		} else {
			optionsJson = jsonutils.NewDict()
		}
		if input.Options != nil {
			if input.Options.Contains("password") {
				key, _ := acnt.getPassword()
				passwd, _ := input.Options.GetString("password")
				passwd, _ = utils.EncryptAESBase64(key, passwd)
				input.Options.Set("password", jsonutils.NewString(passwd))
			}
			optionsJson.Update(input.Options)
		}
		input.Options = optionsJson
	}

	skipSyncResources := &api.SkipSyncResources{}
	if acnt.SkipSyncResources != nil {
		for _, res := range *acnt.SkipSyncResources {
			skipSyncResources.Add(res)
		}
	}
	if input.SkipSyncResources != nil {
		skipSyncResources = input.SkipSyncResources
	}
	for _, res := range input.AddSkipSyncResources {
		skipSyncResources.Add(res)
	}
	for _, res := range input.RemoveSkipSyncResources {
		skipSyncResources.Remove(res)
	}
	input.SkipSyncResources = skipSyncResources
	if len(*skipSyncResources) == 0 {
		db.Update(acnt, func() error {
			acnt.SkipSyncResources = nil
			return nil
		})
	}

	factory, err := acnt.GetProviderFactory()
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrapf(err, "GetProviderFactory"))
	}
	if input.SAMLAuth != nil && *input.SAMLAuth && !factory.IsSupportSAMLAuth() {
		return input, httperrors.NewNotSupportedError("%s not support saml auth", acnt.Provider)
	}

	if len(input.ProxySettingId) > 0 {
		var proxySetting *proxy.SProxySetting
		proxySetting, input.ProxySettingResourceInput, err = proxy.ValidateProxySettingResourceInput(ctx, userCred, input.ProxySettingResourceInput)
		if err != nil {
			return input, errors.Wrap(err, "ValidateProxySettingResourceInput")
		}

		if proxySetting != nil && proxySetting.Id != acnt.ProxySettingId {
			// updated proxy setting, so do the check
			proxyFunc := proxySetting.HttpTransportProxyFunc()
			secret, _ := acnt.getPassword()
			_, _, err := cloudprovider.IsValidCloudAccount(cloudprovider.ProviderConfig{
				Vendor:    acnt.Provider,
				URL:       acnt.AccessUrl,
				Account:   acnt.Account,
				Secret:    secret,
				RegionId:  acnt.regionId(),
				ProxyFunc: proxyFunc,

				AliyunResourceGroupIds: options.Options.AliyunResourceGroups,

				Options: input.Options,
			})
			if err != nil {
				return input, httperrors.NewInputParameterError("invalid proxy setting %s", err)
			}
		}
	}

	input.EnabledStatusInfrasResourceBaseUpdateInput, err = acnt.SEnabledStatusInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusInfrasResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.ValidateUpdateData")
	}

	return input, nil
}

func (acnt *SCloudaccount) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	acnt.SEnabledStatusInfrasResourceBase.PostUpdate(ctx, userCred, query, data)
	input := api.CloudaccountUpdateInput{}
	data.Unmarshal(&input)
	if input.Options != nil {
		logclient.AddSimpleActionLog(acnt, logclient.ACT_UPDATE_BILLING_OPTIONS, input.Options, userCred, true)
	}
	if input.CleanLakeOfPermissions {
		db.Update(acnt, func() error {
			acnt.LakeOfPermissions = nil
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

	input.SProjectMappingResourceInput, err = manager.SProjectMappingResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SProjectMappingResourceInput)
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
		projInput := apis.ProjectizedResourceCreateInput{
			DomainizedResourceInput:  input.DomainizedResourceInput,
			ProjectizedResourceInput: input.ProjectizedResourceInput,
		}
		proj, input.ProjectizedResourceInput, err = db.ValidateProjectizedResourceInput(ctx, projInput)
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
		obj, err := ZoneManager.FetchByIdOrName(ctx, userCred, input.Zone)
		if err != nil {
			return input, errors.Wrapf(err, "unable to fetch Zone %s", input.Zone)
		}
		input.Zone = obj.GetId()
	}

	if input.Options == nil {
		input.Options = jsonutils.NewDict()
	}
	input.Options.Update(jsonutils.Marshal(input.SCloudaccountCredential.SHCSOEndpoints))

	input.SCloudaccount, err = providerDriver.ValidateCreateCloudaccountData(ctx, input.SCloudaccountCredential)
	if err != nil {
		return input, err
	}

	if input.Options.Contains("password") {
		passwd, _ := input.Options.GetString("password")
		passwd, _ = utils.EncryptAESBase64(input.Secret, passwd)
		if len(passwd) > 0 {
			input.Options.Set("password", jsonutils.NewString(passwd))
		}
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
	if providerDriver.IsReadOnly() {
		input.ReadOnly = true
	}

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
		proxySetting, input.ProxySettingResourceInput, err = proxy.ValidateProxySettingResourceInput(ctx, userCred, input.ProxySettingResourceInput)
		if err != nil {
			return input, errors.Wrap(err, "ValidateProxySettingResourceInput")
		}
		proxyFunc = proxySetting.HttpTransportProxyFunc()
	}
	provider, accountId, err := cloudprovider.IsValidCloudAccount(cloudprovider.ProviderConfig{
		Name:      input.Name,
		Vendor:    input.Provider,
		URL:       input.AccessUrl,
		Account:   input.Account,
		Secret:    input.Secret,
		RegionId:  input.RegionId,
		ProxyFunc: proxyFunc,

		AdminProjectId:         auth.GetAdminSession(ctx, options.Options.Region).GetProjectId(),
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
		regions, err := provider.GetIRegions()
		if err != nil {
			return input, err
		}
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

func (acnt *SCloudaccount) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if !data.Contains("enabled") {
		acnt.SetEnabled(true)
	}
	if len(acnt.Brand) == 0 {
		acnt.Brand = acnt.Provider
	}
	acnt.DomainId = ownerId.GetProjectDomainId()
	// force private and share_mode=account_domain
	if !data.Contains("public_scope") {
		acnt.ShareMode = api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN
		acnt.IsPublic = false
		acnt.PublicScope = string(rbacscope.ScopeNone)
		// mark the public_scope has been set
		data.(*jsonutils.JSONDict).Set("public_scope", jsonutils.NewString(acnt.PublicScope))
	}
	if len(acnt.ShareMode) == 0 {
		if acnt.IsPublic {
			acnt.ShareMode = api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM
		} else {
			acnt.ShareMode = api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN
		}
	}
	return acnt.SEnabledStatusInfrasResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (acnt *SCloudaccount) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
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

	acnt.SEnabledStatusInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	acnt.savePassword(acnt.Secret)

	if acnt.Enabled.IsTrue() && jsonutils.QueryBoolean(data, "start_sync", true) {
		acnt.StartSyncCloudAccountInfoTask(ctx, userCred, nil, "", data)
	} else {
		acnt.SubmitSyncAccountTask(ctx, userCred, nil)
	}

	if acnt.Brand == api.CLOUD_PROVIDER_VMWARE {
		_, err := image.Images.PerformClassAction(auth.GetAdminSession(ctx, options.Options.Region), "vmware-account-added", nil)
		if err != nil {
			log.Errorf("failed inform glance vmware account added: %s", err)
		}
	}
}

func (acnt *SCloudaccount) savePassword(secret string) error {
	sec, err := utils.EncryptAESBase64(acnt.Id, secret)
	if err != nil {
		return err
	}

	_, err = db.Update(acnt, func() error {
		acnt.Secret = sec
		return nil
	})
	return err
}

func (acnt *SCloudaccount) getPassword() (string, error) {
	return utils.DescryptAESBase64(acnt.Id, acnt.Secret)
}

func (acnt *SCloudaccount) GetOptionPassword() (string, error) {
	passwd, err := acnt.getPassword()
	if err != nil {
		return "", err
	}
	passwdStr, _ := acnt.Options.GetString("password")
	if len(passwdStr) == 0 {
		return "", fmt.Errorf("missing password")
	}
	return utils.DescryptAESBase64(passwd, passwdStr)
}

func (acnt *SCloudaccount) regionId() string {
	if len(acnt.RegionId) > 0 {
		return acnt.RegionId
	}
	if gotypes.IsNil(acnt.Options) {
		return ""
	}
	regionId, _ := acnt.Options.GetString("default_region")
	return regionId
}

func (acnt *SCloudaccount) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SyncRangeInput) (jsonutils.JSONObject, error) {
	if !acnt.GetEnabled() {
		return nil, httperrors.NewInvalidStatusError("Account disabled")
	}

	syncRange := SSyncRange{SyncRangeInput: input}
	if syncRange.FullSync || len(syncRange.Region) > 0 || len(syncRange.Zone) > 0 || len(syncRange.Host) > 0 || len(syncRange.Resources) > 0 {
		syncRange.DeepSync = true
	}
	syncRange.SkipSyncResources = []string{}
	if acnt.SkipSyncResources != nil {
		for _, res := range *acnt.SkipSyncResources {
			syncRange.SkipSyncResources = append(syncRange.SkipSyncResources, res)
		}
	}
	if acnt.CanSync() || syncRange.Force {
		return nil, acnt.StartSyncCloudAccountInfoTask(ctx, userCred, &syncRange, "", nil)
	}
	return nil, httperrors.NewInvalidStatusError("Unable to synchronize frequently")
}

// 测试账号连通性(更新秘钥信息时)
func (acnt *SCloudaccount) PerformTestConnectivity(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input cloudprovider.SCloudaccountCredential) (jsonutils.JSONObject, error) {
	providerDriver, err := acnt.GetProviderFactory()
	if err != nil {
		return nil, httperrors.NewBadRequestError("failed to found provider factory error: %v", err)
	}

	account, err := providerDriver.ValidateUpdateCloudaccountCredential(ctx, input, acnt.Account)
	if err != nil {
		return nil, err
	}

	_, _, err = cloudprovider.IsValidCloudAccount(cloudprovider.ProviderConfig{
		URL:     acnt.AccessUrl,
		Vendor:  acnt.Provider,
		Account: account.Account,
		Secret:  account.Secret,

		RegionId: acnt.regionId(),

		AliyunResourceGroupIds: options.Options.AliyunResourceGroups,

		ReadOnly: acnt.ReadOnly,

		ProxyFunc: acnt.proxyFunc(),
	})
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid cloud account info error: %s", err.Error())
	}

	return nil, nil
}

func (acnt *SCloudaccount) PerformUpdateCredential(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input cloudprovider.SCloudaccountCredential,
) (jsonutils.JSONObject, error) {
	if !acnt.GetEnabled() {
		return nil, httperrors.NewInvalidStatusError("Account disabled")
	}

	providerDriver, err := acnt.GetProviderFactory()
	if err != nil {
		return nil, httperrors.NewBadRequestError("failed to found provider factory error: %v", err)
	}

	account, err := providerDriver.ValidateUpdateCloudaccountCredential(ctx, input, acnt.Account)
	if err != nil {
		return nil, err
	}

	accountAccessUrl := acnt.AccessUrl
	if len(account.AccessUrl) > 0 {
		accountAccessUrl = account.AccessUrl
	}

	changed := false
	if acnt.Account != account.Account && (len(account.Secret) > 0 || len(account.Account) > 0) {
		// check duplication
		q := acnt.GetModelManager().Query()
		q = q.Equals("account", account.Account)
		q = q.Equals("access_url", accountAccessUrl)
		q = q.NotEquals("id", acnt.Id)
		cnt, err := q.CountWithError()
		if err != nil {
			return nil, httperrors.NewInternalServerError("check uniqueness fail %s", err)
		}
		if cnt > 0 {
			return nil, httperrors.NewConflictError("account %s conflict", account.Account)
		}
	}

	originSecret, _ := acnt.getPassword()

	hcsoEndpoints := cloudprovider.SHCSOEndpoints{}
	if acnt.Provider == api.CLOUD_PROVIDER_HCSO && input.SHCSOEndpoints != nil {
		if acnt.Options == nil {
			acnt.Options = jsonutils.NewDict()
		}

		newOptions := jsonutils.Marshal(input.SHCSOEndpoints)
		_, err = db.UpdateWithLock(ctx, acnt, func() error {
			acnt.Options.Update(newOptions)
			return nil
		})
		if err != nil {
			return nil, err
		}

		err = acnt.Options.Unmarshal(&hcsoEndpoints)
		if err != nil {
			return nil, err
		}
	}

	_, accountId, err := cloudprovider.IsValidCloudAccount(cloudprovider.ProviderConfig{
		Name:     acnt.Name,
		Vendor:   acnt.Provider,
		URL:      accountAccessUrl,
		Account:  account.Account,
		Secret:   account.Secret,
		Options:  acnt.Options,
		RegionId: acnt.regionId(),

		AliyunResourceGroupIds: options.Options.AliyunResourceGroups,

		ReadOnly: acnt.ReadOnly,

		ProxyFunc: acnt.proxyFunc(),
	})
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid cloud account info error: %s", err.Error())
	}

	isEqual := providerDriver.GetAccountIdEqualizer()
	// for backward compatibility
	if !isEqual(acnt.AccountId, accountId) {
		return nil, httperrors.NewConflictError("inconsistent account_id, previous '%s' and now '%s'", acnt.AccountId, accountId)
	}

	if (account.Account != acnt.Account) || (account.Secret != originSecret) {
		if account.Account != acnt.Account {
			for _, cloudprovider := range acnt.GetCloudproviders() {
				if strings.Contains(cloudprovider.Account, acnt.Account) {
					_, err = db.Update(&cloudprovider, func() error {
						cloudprovider.Account = strings.ReplaceAll(cloudprovider.Account, acnt.Account, account.Account)
						return nil
					})
					if err != nil {
						return nil, errors.Wrap(err, "save account for cloud provider")
					}
				}
			}
		}
		_, err = db.Update(acnt, func() error {
			acnt.Account = account.Account
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "save account")
		}

		err = acnt.savePassword(account.Secret)
		if err != nil {
			return nil, errors.Wrap(err, "save password")
		}

		for _, provider := range acnt.GetCloudproviders() {
			provider.savePassword(account.Secret)
		}
		changed = true
	}

	if len(account.AccessUrl) > 0 && account.AccessUrl != acnt.AccessUrl {
		// save accessUrl

		for _, cloudprovider := range acnt.GetCloudproviders() {
			_, err = db.Update(&cloudprovider, func() error {
				cloudprovider.AccessUrl = strings.ReplaceAll(cloudprovider.AccessUrl, acnt.AccessUrl, account.AccessUrl)
				return nil
			})
			if err != nil {
				return nil, errors.Wrap(err, "save access_url for cloud provider")
			}
		}

		_, err = db.Update(acnt, func() error {
			acnt.AccessUrl = account.AccessUrl
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "save access_url")
		}

		changed = true
	}

	if changed {
		db.OpsLog.LogEvent(acnt, db.ACT_UPDATE, account, userCred)
		logclient.AddActionLogWithContext(ctx, acnt, logclient.ACT_UPDATE_CREDENTIAL, account, userCred, true)

		acnt.SetStatus(ctx, userCred, api.CLOUD_PROVIDER_INIT, "Change credential")
		acnt.StartSyncCloudAccountInfoTask(ctx, userCred, nil, "", nil)
	}

	return nil, nil
}

func (acnt *SCloudaccount) StartSyncCloudAccountInfoTask(ctx context.Context, userCred mcclient.TokenCredential, syncRange *SSyncRange, parentTaskId string, data jsonutils.JSONObject) error {
	params := jsonutils.NewDict()
	if data != nil {
		params.Update(data)
	}
	if gotypes.IsNil(syncRange) {
		syncRange = &SSyncRange{}
		syncRange.FullSync = true
		syncRange.DeepSync = true
	}
	syncRange.SkipSyncResources = []string{}
	if acnt.SkipSyncResources != nil {
		for _, res := range *acnt.SkipSyncResources {
			syncRange.SkipSyncResources = append(syncRange.SkipSyncResources, res)
		}
	}
	params.Add(jsonutils.Marshal(syncRange), "sync_range")

	task, err := taskman.TaskManager.NewTask(ctx, "CloudAccountSyncInfoTask", acnt, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	acnt.markStartSync(userCred, syncRange)
	db.OpsLog.LogEvent(acnt, db.ACT_SYNC_HOST_START, "", userCred)
	return task.ScheduleRun(nil)
}

func (acnt *SCloudaccount) markStartSync(userCred mcclient.TokenCredential, syncRange *SSyncRange) error {
	_, err := db.Update(acnt, func() error {
		acnt.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_QUEUED
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update")
	}
	providers := acnt.GetCloudproviders()
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

func (acnt *SCloudaccount) MarkSyncing(userCred mcclient.TokenCredential) error {
	_, err := db.Update(acnt, func() error {
		if acnt.SyncStatus != api.CLOUD_PROVIDER_SYNC_STATUS_SYNCING {
			acnt.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_SYNCING
			acnt.LastSync = timeutils.UtcNow()
			acnt.LastSyncEndAt = time.Time{}
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	return nil
}

func (acnt *SCloudaccount) MarkEndSyncWithLock(ctx context.Context, userCred mcclient.TokenCredential, deepSync bool) error {
	lockman.LockObject(ctx, acnt)
	defer lockman.ReleaseObject(ctx, acnt)

	providers := acnt.GetCloudproviders()
	for i := range providers {
		err := providers[i].cancelStartingSync(userCred)
		if err != nil {
			return errors.Wrap(err, "providers.cancelStartingSync")
		}
	}

	if acnt.getSyncStatus2() != api.CLOUD_PROVIDER_SYNC_STATUS_IDLE {
		return errors.Error("some cloud providers not idle")
	}

	return acnt.MarkEndSync(userCred, deepSync)
}

func (acnt *SCloudaccount) MarkEndSync(userCred mcclient.TokenCredential, deepSync bool) error {
	_, err := db.Update(acnt, func() error {
		acnt.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
		if deepSync {
			acnt.LastSyncEndAt = timeutils.UtcNow()
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	return nil
}

func (acnt *SCloudaccount) GetProviderFactory() (cloudprovider.ICloudProviderFactory, error) {
	return cloudprovider.GetProviderFactory(acnt.Provider)
}

func (acnt *SCloudaccount) GetProvider(ctx context.Context) (cloudprovider.ICloudProvider, error) {
	if !acnt.GetEnabled() {
		return nil, fmt.Errorf("Cloud provider is not enabled")
	}
	return acnt.getProviderInternal(ctx)
}

func (acnt *SCloudaccount) proxySetting() *proxy.SProxySetting {
	m, err := proxy.ProxySettingManager.FetchById(acnt.ProxySettingId)
	if err != nil {
		log.Errorf("cloudaccount %s(%s): get proxysetting %s: %v",
			acnt.Name, acnt.Id, acnt.ProxySettingId, err)
		return nil
	}
	ps := m.(*proxy.SProxySetting)
	return ps
}

func (acnt *SCloudaccount) proxyFunc() httputils.TransportProxyFunc {
	ps := acnt.proxySetting()
	if ps != nil {
		return ps.HttpTransportProxyFunc()
	}
	return nil
}

func (acnt *SCloudaccount) UpdatePermission(ctx context.Context) func(string, string) {
	return func(service, permission string) {
		key := "update permission"

		lockman.LockRawObject(ctx, acnt.Id, key)
		defer lockman.ReleaseRawObject(ctx, acnt.Id, key)

		db.Update(acnt, func() error {
			data := api.SAccountPermissions{}
			if acnt.LakeOfPermissions != nil {
				data = *acnt.LakeOfPermissions
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
			acnt.LakeOfPermissions = &data
			return nil
		})
	}
}

func (acnt *SCloudaccount) getProviderInternal(ctx context.Context) (cloudprovider.ICloudProvider, error) {
	secret, err := acnt.getPassword()
	if err != nil {
		return nil, fmt.Errorf("Invalid password %s", err)
	}

	return cloudprovider.GetProvider(cloudprovider.ProviderConfig{
		Id:      acnt.Id,
		Name:    acnt.Name,
		Vendor:  acnt.Provider,
		URL:     acnt.AccessUrl,
		Account: acnt.Account,
		Secret:  secret,

		Options:   acnt.Options,
		RegionId:  acnt.regionId(),
		ProxyFunc: acnt.proxyFunc(),

		ReadOnly:               acnt.ReadOnly,
		AliyunResourceGroupIds: options.Options.AliyunResourceGroups,

		UpdatePermission: acnt.UpdatePermission(ctx),
	})
}

/*func (acnt *SCloudaccount) GetSubAccounts(ctx context.Context) ([]cloudprovider.SSubAccount, error) {
	provider, err := acnt.getProviderInternal(ctx)
	if err != nil {
		return nil, err
	}
	return provider.GetSubAccounts()
}*/

func (acnt *SCloudaccount) getDefaultExternalProject(id string) (*SExternalProject, error) {
	q := ExternalProjectManager.Query().Equals("cloudaccount_id", acnt.Id).Equals("external_id", id)
	projects := []SExternalProject{}
	err := db.FetchModelObjects(ExternalProjectManager, q, &projects)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	if len(projects) > 1 {
		return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, "%s", id)
	}
	if len(projects) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s", id)
	}
	return &projects[0], nil
}

func (acnt *SCloudaccount) removeSubAccounts(ctx context.Context, userCred mcclient.TokenCredential, subAccounts []cloudprovider.SSubAccount) error {
	accounts := []string{}
	for i := range subAccounts {
		accounts = append(accounts, subAccounts[i].Account)
	}
	q := CloudproviderManager.Query().Equals("cloudaccount_id", acnt.Id).NotIn("account", accounts)
	providers := []SCloudprovider{}
	err := db.FetchModelObjects(CloudproviderManager, q, &providers)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range providers {
		// 禁用云订阅，并设置为未连接状态，避免权限异常删除云订阅
		providers[i].PerformDisable(ctx, userCred, jsonutils.NewDict(), apis.PerformDisableInput{})
		providers[i].SetStatus(ctx, userCred, api.CLOUD_PROVIDER_DISCONNECTED, "sync lost")
	}
	return nil
}

func (acnt *SCloudaccount) importSubAccount(ctx context.Context, userCred mcclient.TokenCredential, subAccount cloudprovider.SSubAccount) (*SCloudprovider, bool, error) {
	// log.Debugf("XXXX importSubAccount %s", jsonutils.Marshal(subAccount))
	isNew := false
	q := CloudproviderManager.Query().Equals("cloudaccount_id", acnt.Id).Equals("account", subAccount.Account)
	providerCount, err := q.CountWithError()
	if err != nil {
		return nil, isNew, err
	}
	if providerCount > 1 {
		log.Errorf("cloudaccount %s has duplicate subaccount with name %s", acnt.Name, subAccount.Account)
		return nil, isNew, cloudprovider.ErrDuplicateId
	}
	if providerCount == 1 {
		provider := &SCloudprovider{}
		provider.SetModelManager(CloudproviderManager, provider)
		err = q.First(provider)
		if err != nil {
			return nil, isNew, errors.Wrapf(err, "q.First")
		}
		err = func() error {
			// 根据云订阅归属且云订阅之前没有手动指定过项目
			if acnt.AutoCreateProjectForProvider && provider.ProjectSrc != string(apis.OWNER_SOURCE_LOCAL) {
				lockman.LockRawObject(ctx, CloudproviderManager.Keyword(), "name")
				defer lockman.ReleaseRawObject(ctx, CloudproviderManager.Keyword(), "name")
				// 根据云订阅名称获取或创建项目
				domainId, projectId, err := acnt.getOrCreateTenant(ctx, provider.Name, provider.DomainId, "", subAccount.Desc)
				if err != nil {
					return errors.Wrapf(err, "getOrCreateTenant err,provider_name :%s", provider.Name)
				}
				// 覆盖云订阅项目
				db.Update(provider, func() error {
					provider.ProjectId = projectId
					provider.DomainId = domainId
					return nil
				})
			}
			return nil
		}()
		if err != nil {
			return nil, isNew, errors.Wrapf(err, "sync autro create project for provider")
		}
		// 没有项目归属时以默认最初项目做归属
		if len(provider.ProjectId) == 0 {
			_, err := db.Update(provider, func() error {
				if len(subAccount.DefaultProjectId) > 0 {
					proj, err := acnt.getDefaultExternalProject(subAccount.DefaultProjectId)
					if err != nil {
						logclient.AddSimpleActionLog(provider, logclient.ACT_UPDATE, errors.Wrapf(err, "getDefaultExternalProject(%s)", subAccount.DefaultProjectId), userCred, false)
					} else {
						provider.DomainId = proj.DomainId
						provider.ProjectId = proj.ProjectId
						return nil
					}
				}
				// find default project of domain
				ownerId := acnt.GetOwnerId()
				t, err := db.TenantCacheManager.FindFirstProjectOfDomain(ctx, ownerId.GetProjectDomainId())
				if err != nil {
					logclient.AddSimpleActionLog(provider, logclient.ACT_UPDATE, errors.Wrapf(err, "FindFirstProjectOfDomain(%s)", ownerId.GetProjectDomainId()), userCred, false)
					return errors.Wrapf(err, "FindFirstProjectOfDomain(%s)", ownerId.GetProjectDomainId())
				}
				provider.DomainId = t.DomainId
				provider.ProjectId = t.Id
				return nil
			})
			if err != nil {
				return nil, isNew, errors.Wrap(err, "Update project and domain")
			}
		}
		provider.markProviderConnected(ctx, userCred, subAccount.HealthStatus)
		provider.updateName(ctx, userCred, subAccount.Name, subAccount.Desc)
		if len(provider.ExternalId) == 0 || provider.ExternalId != subAccount.Id {
			_, err := db.Update(provider, func() error {
				provider.ExternalId = subAccount.Id
				return nil
			})
			if err != nil {
				return nil, isNew, errors.Wrap(err, "Update ExternalId")
			}
		}
		return provider, isNew, nil
	}
	// not found, create a new cloudprovider
	isNew = true

	newCloudprovider, err := func() (*SCloudprovider, error) {
		newCloudprovider := SCloudprovider{}
		newCloudprovider.ProjectSrc = string(apis.OWNER_SOURCE_CLOUD)
		newCloudprovider.Account = subAccount.Account
		newCloudprovider.ExternalId = subAccount.Id
		newCloudprovider.Secret = acnt.Secret
		newCloudprovider.CloudaccountId = acnt.Id
		newCloudprovider.Provider = acnt.Provider
		newCloudprovider.AccessUrl = acnt.AccessUrl
		newCloudprovider.HealthStatus = subAccount.HealthStatus
		newCloudprovider.Description = subAccount.Desc
		newCloudprovider.DomainId = acnt.DomainId
		newCloudprovider.ProjectId = acnt.ProjectId
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
			ownerId := acnt.GetOwnerId()
			if ownerId.GetProjectDomainId() == userCred.GetProjectDomainId() {
				ownerId = userCred
			}
			newCloudprovider.DomainId = ownerId.GetProjectDomainId()
			newCloudprovider.ProjectId = ownerId.GetProjectId()

		}

		newCloudprovider.SetModelManager(CloudproviderManager, &newCloudprovider)
		err = func() error {
			if acnt.AutoCreateProjectForProvider {
				lockman.LockRawObject(ctx, CloudproviderManager.Keyword(), "name")
				defer lockman.ReleaseRawObject(ctx, CloudproviderManager.Keyword(), "name")
				newCloudprovider.Name, err = db.GenerateName(ctx, CloudproviderManager, nil, subAccount.Name)
				if err != nil {
					return err
				}
				domainId, projectId, err := acnt.getOrCreateTenant(ctx, newCloudprovider.Name, newCloudprovider.DomainId, "", subAccount.Desc)
				if err != nil {
					return errors.Wrapf(err, "getOrCreateTenant err,provider_name :%s", newCloudprovider.Name)
				}
				newCloudprovider.ProjectId = projectId
				newCloudprovider.DomainId = domainId
			} else {
				lockman.LockRawObject(ctx, CloudproviderManager.Keyword(), "name")
				defer lockman.ReleaseRawObject(ctx, CloudproviderManager.Keyword(), "name")
				newCloudprovider.Name, err = db.GenerateName(ctx, CloudproviderManager, nil, subAccount.Name)
				if err != nil {
					return err
				}
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

	passwd, err := acnt.getPassword()
	if err != nil {
		return nil, isNew, err
	}

	newCloudprovider.savePassword(passwd)

	if len(subAccount.DefaultProjectId) == 0 && acnt.AutoCreateProject && len(acnt.ProjectId) == 0 {
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

func (manager *SCloudaccountManager) FetchCloudaccountByIdOrName(ctx context.Context, accountId string) *SCloudaccount {
	providerObj, err := manager.FetchByIdOrName(ctx, nil, accountId)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("%s", err)
		}
		return nil
	}
	return providerObj.(*SCloudaccount)
}

func (acnt *SCloudaccount) GetProviderCount() (int, error) {
	q := CloudproviderManager.Query().Equals("cloudaccount_id", acnt.Id)
	return q.CountWithError()
}

func (acnt *SCloudaccount) GetHostCount() (int, error) {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", acnt.Id).SubQuery()
	q := HostManager.Query().In("manager_id", subq).IsFalse("is_emulated")
	return q.CountWithError()
}

func (acnt *SCloudaccount) GetVpcCount() (int, error) {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", acnt.Id).SubQuery()
	q := VpcManager.Query().In("manager_id", subq).IsFalse("is_emulated")
	return q.CountWithError()
}

func (acnt *SCloudaccount) GetStorageCount() (int, error) {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", acnt.Id).SubQuery()
	q := StorageManager.Query().In("manager_id", subq).IsFalse("is_emulated")
	return q.CountWithError()
}

func (acnt *SCloudaccount) GetStoragecacheCount() (int, error) {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", acnt.Id).SubQuery()
	q := StoragecacheManager.Query().In("manager_id", subq)
	return q.CountWithError()
}

func (acnt *SCloudaccount) GetEipCount() (int, error) {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", acnt.Id).SubQuery()
	q := ElasticipManager.Query().In("manager_id", subq)
	return q.CountWithError()
}

func (acnt *SCloudaccount) GetRoutetableCount() (int, error) {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", acnt.Id).SubQuery()
	vpcs := VpcManager.Query("id", "manager_id").SubQuery()
	q := RouteTableManager.Query()
	q = q.Join(vpcs, sqlchemy.Equals(q.Field("vpc_id"), vpcs.Field("id")))
	q = q.Filter(sqlchemy.In(vpcs.Field("manager_id"), subq))
	return q.CountWithError()
}

func (acnt *SCloudaccount) GetGuestCount() (int, error) {
	subsubq := CloudproviderManager.Query("id").Equals("cloudaccount_id", acnt.Id).SubQuery()
	subq := HostManager.Query("id").In("manager_id", subsubq).SubQuery()
	q := GuestManager.Query().In("host_id", subq)
	return q.CountWithError()
}

func (acnt *SCloudaccount) GetDiskCount() (int, error) {
	subsubq := CloudproviderManager.Query("id").Equals("cloudaccount_id", acnt.Id).SubQuery()
	subq := StorageManager.Query("id").In("manager_id", subsubq).SubQuery()
	q := DiskManager.Query().In("storage_id", subq)
	return q.CountWithError()
}

func (acnt *SCloudaccount) GetCloudEnv() string {
	if acnt.IsOnPremise {
		return api.CLOUD_ENV_ON_PREMISE
	} else if acnt.IsPublicCloud.IsTrue() {
		return api.CLOUD_ENV_PUBLIC_CLOUD
	} else {
		return api.CLOUD_ENV_PRIVATE_CLOUD
	}
}

func (acnt *SCloudaccount) GetEnvironment() string {
	return acnt.AccessUrl
}

type SAccountUsageCount struct {
	Id string
	api.SAccountUsage
}

func (nm *SCloudaccountManager) query(manager db.IModelManager, field string, accountIds []string, filter func(*sqlchemy.SQuery) *sqlchemy.SQuery) *sqlchemy.SSubQuery {
	q := manager.Query()

	if filter != nil {
		q = filter(q)
	}

	sq := q.SubQuery()

	return sq.Query(
		sq.Field("cloudaccount_id"),
		sqlchemy.COUNT(field),
	).In("cloudaccount_id", accountIds).GroupBy(sq.Field("cloudaccount_id")).SubQuery()
}

func (manager *SCloudaccountManager) TotalResourceCount(accountIds []string) (map[string]api.SAccountUsage, error) {
	// eip
	eipSQ := manager.query(ElasticipManager, "eip_cnt", accountIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		providers := CloudproviderManager.Query().SubQuery()
		sq := q.SubQuery()
		return sq.Query(
			sq.Field("id"),
			providers.Field("cloudaccount_id").Label("cloudaccount_id"),
		).LeftJoin(providers, sqlchemy.Equals(sq.Field("manager_id"), providers.Field("id")))
	})

	// vpc
	vpcSQ := manager.query(VpcManager, "vpc_cnt", accountIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		providers := CloudproviderManager.Query().SubQuery()
		sq := q.SubQuery()
		return sq.Query(
			sq.Field("id"),
			providers.Field("cloudaccount_id").Label("cloudaccount_id"),
		).LeftJoin(providers, sqlchemy.Equals(sq.Field("manager_id"), providers.Field("id")))
	})

	// disk
	diskSQ := manager.query(DiskManager, "disk_cnt", accountIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		storages := StorageManager.Query().SubQuery()
		providers := CloudproviderManager.Query().SubQuery()
		sq := q.SubQuery()
		return sq.Query(
			sq.Field("id"),
			providers.Field("cloudaccount_id").Label("cloudaccount_id"),
		).LeftJoin(storages, sqlchemy.Equals(sq.Field("storage_id"), storages.Field("id"))).
			LeftJoin(providers, sqlchemy.Equals(storages.Field("manager_id"), providers.Field("id")))
	})

	// host
	hostSQ := manager.query(HostManager, "host_cnt", accountIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		providers := CloudproviderManager.Query().SubQuery()
		sq := q.SubQuery()
		return sq.Query(
			sq.Field("id"),
			providers.Field("cloudaccount_id").Label("cloudaccount_id"),
		).LeftJoin(providers, sqlchemy.Equals(sq.Field("manager_id"), providers.Field("id")))
	})

	// guest
	guestSQ := manager.query(GuestManager, "guest_cnt", accountIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		hosts := HostManager.Query().SubQuery()
		providers := CloudproviderManager.Query().SubQuery()
		sq := q.SubQuery()
		return sq.Query(
			sq.Field("id"),
			providers.Field("cloudaccount_id").Label("cloudaccount_id"),
		).LeftJoin(hosts, sqlchemy.Equals(sq.Field("host_id"), hosts.Field("id"))).
			LeftJoin(providers, sqlchemy.Equals(hosts.Field("manager_id"), providers.Field("id")))
	})

	// storage
	storageSQ := manager.query(StorageManager, "storage_cnt", accountIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		providers := CloudproviderManager.Query().SubQuery()
		sq := q.SubQuery()
		return sq.Query(
			sq.Field("id"),
			providers.Field("cloudaccount_id").Label("cloudaccount_id"),
		).LeftJoin(providers, sqlchemy.Equals(sq.Field("manager_id"), providers.Field("id")))
	})

	// provider
	providerSQ := manager.query(CloudproviderManager, "provider_cnt", accountIds, nil)

	// enabled provider
	enabledProviderSQ := manager.query(CloudproviderManager, "enabled_provider_cnt", accountIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.IsTrue("enabled")
	})

	// route
	routeSQ := manager.query(RouteTableManager, "routetable_cnt", accountIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		vpcs := VpcManager.Query().SubQuery()
		providers := CloudproviderManager.Query().SubQuery()
		sq := q.SubQuery()
		return sq.Query(
			sq.Field("id"),
			providers.Field("cloudaccount_id").Label("cloudaccount_id"),
		).LeftJoin(vpcs, sqlchemy.Equals(sq.Field("vpc_id"), vpcs.Field("id"))).
			LeftJoin(providers, sqlchemy.Equals(vpcs.Field("manager_id"), providers.Field("id")))
	})

	// scache
	scacheSQ := manager.query(StoragecacheManager, "storagecache_cnt", accountIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		providers := CloudproviderManager.Query().SubQuery()
		sq := q.SubQuery()
		return sq.Query(
			sq.Field("id"),
			providers.Field("cloudaccount_id").Label("cloudaccount_id"),
		).LeftJoin(providers, sqlchemy.Equals(sq.Field("manager_id"), providers.Field("id")))
	})

	// sync count
	syncSQ := manager.query(CloudproviderRegionManager, "sync_cnt", accountIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		providers := CloudproviderManager.Query().SubQuery()
		sq := q.NotEquals("sync_status", api.CLOUD_PROVIDER_SYNC_STATUS_IDLE).SubQuery()
		return sq.Query(
			sq.Field("row_id").Label("id"),
			providers.Field("cloudaccount_id").Label("cloudaccount_id"),
		).LeftJoin(providers, sqlchemy.Equals(sq.Field("cloudprovider_id"), providers.Field("id")))
	})

	accounts := manager.Query().SubQuery()
	accountQ := accounts.Query(
		sqlchemy.SUM("eip_count", eipSQ.Field("eip_cnt")),
		sqlchemy.SUM("vpc_count", vpcSQ.Field("vpc_cnt")),
		sqlchemy.SUM("disk_count", diskSQ.Field("disk_cnt")),
		sqlchemy.SUM("host_count", hostSQ.Field("host_cnt")),
		sqlchemy.SUM("guest_count", guestSQ.Field("guest_cnt")),
		sqlchemy.SUM("storage_count", storageSQ.Field("storage_cnt")),
		sqlchemy.SUM("provider_count", providerSQ.Field("provider_cnt")),
		sqlchemy.SUM("enabled_provider_count", enabledProviderSQ.Field("enabled_provider_cnt")),
		sqlchemy.SUM("routetable_count", routeSQ.Field("routetable_cnt")),
		sqlchemy.SUM("storagecache_count", scacheSQ.Field("storagecache_cnt")),
		sqlchemy.SUM("sync_count", syncSQ.Field("sync_cnt")),
	)

	accountQ.AppendField(accountQ.Field("id"))

	accountQ = accountQ.LeftJoin(eipSQ, sqlchemy.Equals(accountQ.Field("id"), eipSQ.Field("cloudaccount_id")))
	accountQ = accountQ.LeftJoin(vpcSQ, sqlchemy.Equals(accountQ.Field("id"), vpcSQ.Field("cloudaccount_id")))
	accountQ = accountQ.LeftJoin(diskSQ, sqlchemy.Equals(accountQ.Field("id"), diskSQ.Field("cloudaccount_id")))
	accountQ = accountQ.LeftJoin(hostSQ, sqlchemy.Equals(accountQ.Field("id"), hostSQ.Field("cloudaccount_id")))
	accountQ = accountQ.LeftJoin(guestSQ, sqlchemy.Equals(accountQ.Field("id"), guestSQ.Field("cloudaccount_id")))
	accountQ = accountQ.LeftJoin(storageSQ, sqlchemy.Equals(accountQ.Field("id"), storageSQ.Field("cloudaccount_id")))
	accountQ = accountQ.LeftJoin(providerSQ, sqlchemy.Equals(accountQ.Field("id"), providerSQ.Field("cloudaccount_id")))
	accountQ = accountQ.LeftJoin(enabledProviderSQ, sqlchemy.Equals(accountQ.Field("id"), enabledProviderSQ.Field("cloudaccount_id")))
	accountQ = accountQ.LeftJoin(routeSQ, sqlchemy.Equals(accountQ.Field("id"), routeSQ.Field("cloudaccount_id")))
	accountQ = accountQ.LeftJoin(scacheSQ, sqlchemy.Equals(accountQ.Field("id"), scacheSQ.Field("cloudaccount_id")))
	accountQ = accountQ.LeftJoin(syncSQ, sqlchemy.Equals(accountQ.Field("id"), syncSQ.Field("cloudaccount_id")))

	accountQ = accountQ.Filter(sqlchemy.In(accountQ.Field("id"), accountIds)).GroupBy(accountQ.Field("id"))

	accountCount := []SAccountUsageCount{}
	err := accountQ.All(&accountCount)
	if err != nil {
		return nil, errors.Wrapf(err, "accountQ.All")
	}

	result := map[string]api.SAccountUsage{}
	for i := range accountCount {
		result[accountCount[i].Id] = accountCount[i].SAccountUsage
	}

	return result, nil
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

	accounts := make([]*SCloudaccount, len(objs))
	accountIds := make([]string, len(objs))
	for i := range objs {
		account := objs[i].(*SCloudaccount)
		accountIds[i] = account.Id
		accounts[i] = account
	}

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

	virObjs := make([]interface{}, len(objs))
	for i := range rows {
		obj := &db.SProjectizedResourceBase{ProjectId: accounts[i].ProjectId}
		obj.DomainId = accounts[i].DomainId
		virObjs[i] = obj
	}
	projManager := &db.SProjectizedResourceBaseManager{}
	projRows := projManager.FetchCustomizeColumns(ctx, userCred, query, virObjs, stringutils2.SSortedStrings{}, isList)

	usage, err := manager.TotalResourceCount(accountIds)
	if err != nil {
		log.Errorf("TotalResourceCount error: %v", err)
		return rows
	}

	for i := range rows {
		account := objs[i].(*SCloudaccount)
		rows[i] = api.CloudaccountDetail{
			EnabledStatusInfrasResourceBaseDetails: stdRows[i],
			ProjectMappingResourceInfo:             pmRows[i],
			LastSyncCost:                           account.GetLastSyncCost(),
		}
		if proxySetting, ok := proxySettings[account.ProxySettingId]; ok {
			rows[i].ProxySetting.Id = proxySetting.Id
			rows[i].ProxySetting.Name = proxySetting.Name
			rows[i].ProxySetting.HTTPProxy = proxySetting.HTTPProxy
			rows[i].ProxySetting.HTTPSProxy = proxySetting.HTTPSProxy
			rows[i].ProxySetting.NoProxy = proxySetting.NoProxy
		}
		rows[i].CloudEnv = account.GetCloudEnv()
		rows[i].ProjectizedResourceInfo = projRows[i]
		rows[i].SAccountUsage, _ = usage[accountIds[i]]
		rows[i].SyncStatus2 = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
		if rows[i].SyncCount > 0 {
			rows[i].SyncStatus2 = api.CLOUD_PROVIDER_SYNC_STATUS_SYNCING
		}
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
			log.Errorf("%s", msg)
			return fmt.Errorf("%s", msg)
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
			account.markAccountDisconected(context.Background(), auth.AdminCredential(), errors.Wrapf(err, "getPassword for provider %s", cloudprovider.Name).Error())
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
				account.PublicScope = string(rbacscope.ScopeNone)
				account.IsPublic = false
			case api.CLOUD_ACCOUNT_SHARE_MODE_PROVIDER_DOMAIN:
				account.PublicScope = string(rbacscope.ScopeSystem)
				account.IsPublic = true
			case api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM:
				account.PublicScope = string(rbacscope.ScopeSystem)
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

func (manager *SCloudaccountManager) initializeDefaultTenantId() error {
	// init accountid
	q := manager.Query().IsNullOrEmpty("tenant_id")
	cloudaccounts := make([]SCloudaccount, 0)
	err := db.FetchModelObjects(manager, q, &cloudaccounts)
	if err != nil {
		return errors.Wrap(err, "fetch empty defaullt tenant_id fail")
	}
	for i := range cloudaccounts {
		account := cloudaccounts[i]
		// auto fix accounts without default project
		defaultTenant, err := db.TenantCacheManager.FindFirstProjectOfDomain(context.Background(), account.DomainId)
		if err != nil {
			return errors.Wrapf(err, "FindFirstProjectOfDomain(%s)", account.DomainId)
		}
		_, err = db.Update(&account, func() error {
			account.ProjectId = defaultTenant.Id
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
	err = manager.initializeDefaultTenantId()
	if err != nil {
		return errors.Wrap(err, "initializeDefaultTenantId")
	}

	return nil
}

func (acnt *SCloudaccount) GetBalance() (float64, error) {
	return acnt.Balance, nil
}

func (acnt *SCloudaccount) GetDetailsBalance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	balance, err := acnt.GetBalance()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewFloat64(balance), "balance")
	return ret, nil
}

func (acnt *SCloudaccount) getHostPort() (string, int, error) {
	urlComponent, err := url.Parse(acnt.AccessUrl)
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

func (acnt *SCloudaccount) GetVCenterAccessInfo(privateId string) (vcenter.SVCenterAccessInfo, error) {
	info := vcenter.SVCenterAccessInfo{}

	host, port, err := acnt.getHostPort()
	if err != nil {
		return info, err
	}

	info.VcenterId = acnt.Id
	info.Host = host
	info.Port = port
	info.Account = acnt.Account
	info.Password = acnt.Secret
	info.PrivateId = privateId

	return info, nil
}

// +onecloud:swagger-gen-ignore
func (account *SCloudaccount) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeDomainOwnerInput) (jsonutils.JSONObject, error) {
	return nil, errors.Wrap(httperrors.ErrForbidden, "can't change domain owner of cloudaccount, use PerformChangeProject instead")
}

func (account *SCloudaccount) PerformChangeProject(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeProjectOwnerInput) (jsonutils.JSONObject, error) {
	// 未开启三级权限(默认共享), 允许更改项目
	if consts.GetNonDefaultDomainProjects() && account.IsShared() {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "cannot change owner when shared!")
	}

	project := input.ProjectId
	domain := input.ProjectDomainId
	if len(domain) == 0 {
		domain = account.DomainId
	}

	tenant, err := db.TenantCacheManager.FetchTenantByIdOrNameInDomain(ctx, project, domain)
	if err != nil {
		return nil, httperrors.NewNotFoundError("project %s not found", project)
	}

	if tenant.Id == account.ProjectId {
		return nil, nil
	}

	providers := account.GetCloudproviders()
	if len(account.ProjectId) > 0 {
		if len(providers) > 0 {
			for i := range providers {
				if providers[i].ProjectId != account.ProjectId {
					return nil, errors.Wrap(httperrors.ErrConflict, "cloudproviders' project is different from cloudaccount's")
				}
			}
		}
	}

	if tenant.DomainId != account.DomainId {
		// do change domainId
		input2 := apis.PerformChangeDomainOwnerInput{}
		input2.ProjectDomainId = tenant.DomainId
		_, err := account.SEnabledStatusInfrasResourceBase.PerformChangeOwner(ctx, userCred, query, input2)
		if err != nil {
			return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.PerformChangeOwner")
		}
	}

	// save project_id change
	diff, err := db.Update(account, func() error {
		account.ProjectId = tenant.Id
		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "db.Update ProjectId")
	}

	if len(diff) > 0 {
		syncRange := &SSyncRange{
			SyncRangeInput: api.SyncRangeInput{
				Force:     true,
				Resources: []string{"project"},
			},
		}
		account.StartSyncCloudAccountInfoTask(ctx, userCred, syncRange, "", nil)
	}

	db.OpsLog.LogEvent(account, db.ACT_UPDATE, diff, userCred)

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
		proxy, err := proxy.ProxySettingManager.FetchByIdOrName(ctx, nil, query.ProxySetting)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("proxy_setting", query.ProxySetting)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Equals("proxy_setting_id", proxy.GetId())
	}

	managerStrs := query.CloudproviderId
	conditions := []sqlchemy.ICondition{}
	for _, managerStr := range managerStrs {
		if len(managerStr) == 0 {
			continue
		}
		providerObj, err := CloudproviderManager.FetchByIdOrName(ctx, userCred, managerStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudproviderManager.Keyword(), managerStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		provider := providerObj.(*SCloudprovider)
		conditions = append(conditions, sqlchemy.Equals(q.Field("id"), provider.CloudaccountId))
	}
	if len(conditions) > 0 {
		q = q.Filter(sqlchemy.OR(conditions...))
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

	if query.ReadOnly != nil {
		q = q.Equals("read_only", *query.ReadOnly)
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
	if db.NeedOrderQuery([]string{query.OrderByHostCount}) {
		hostQ := HostManager.Query()
		hostQ = hostQ.AppendField(hostQ.Field("manager_id"), sqlchemy.COUNT("host_count"))
		hostQ = hostQ.GroupBy("manager_id")
		hostSQ := hostQ.SubQuery()

		providerQ := CloudproviderManager.Query()
		providerQ = providerQ.Join(hostSQ, sqlchemy.Equals(providerQ.Field("id"), hostSQ.Field("manager_id")))
		providerQ = providerQ.AppendField(providerQ.Field("cloudaccount_id"), sqlchemy.SUM("host_count", hostSQ.Field("host_count")))
		providerQ = providerQ.GroupBy("cloudaccount_id")
		providerSQ := providerQ.SubQuery()

		q = q.Join(providerSQ, sqlchemy.Equals(providerSQ.Field("cloudaccount_id"), q.Field("id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(providerSQ.Field("host_count"))
		q = db.OrderByFields(q, []string{query.OrderByHostCount}, []sqlchemy.IQueryField{q.Field("host_count")})
	}
	if db.NeedOrderQuery([]string{query.OrderByGuestCount}) {
		guestQ := GuestManager.Query()
		guestQ = guestQ.AppendField(guestQ.Field("host_id"), sqlchemy.COUNT("guest_count"))
		guestQ = guestQ.GroupBy("host_id")
		guestSQ := guestQ.SubQuery()
		hostQ := HostManager.Query()
		hostQ = hostQ.Join(guestSQ, sqlchemy.Equals(hostQ.Field("id"), guestSQ.Field("host_id")))
		hostQ = hostQ.AppendField(hostQ.Field("manager_id"), sqlchemy.SUM("guest_count", guestQ.Field("guest_count")))
		hostQ = hostQ.GroupBy("manager_id")
		hostSQ := hostQ.SubQuery()
		providerQ := CloudproviderManager.Query()
		providerQ = providerQ.Join(hostSQ, sqlchemy.Equals(providerQ.Field("id"), hostSQ.Field("manager_id")))
		providerQ = providerQ.AppendField(providerQ.Field("cloudaccount_id"), sqlchemy.SUM("guest_count", hostSQ.Field("guest_count")))
		providerQ = providerQ.GroupBy("cloudaccount_id")
		providerSQ := providerQ.SubQuery()
		q = q.Join(providerSQ, sqlchemy.Equals(providerSQ.Field("cloudaccount_id"), q.Field("id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(providerSQ.Field("guest_count"))
		q = db.OrderByFields(q, []string{query.OrderByGuestCount}, []sqlchemy.IQueryField{q.Field("guest_count")})
	}
	return q, nil
}

func (account *SCloudaccount) markAccountDisconected(ctx context.Context, userCred mcclient.TokenCredential, reason string) error {
	_, err := db.UpdateWithLock(ctx, account, func() error {
		account.ErrorCount = account.ErrorCount + 1
		account.HealthStatus = api.CLOUD_PROVIDER_HEALTH_UNKNOWN
		return nil
	})
	if err != nil {
		return err
	}
	if account.Status == api.CLOUD_PROVIDER_CONNECTED {
		account.EventNotify(ctx, userCred, notify.ActionSyncAccountStatus)
	}
	return account.SetStatus(ctx, userCred, api.CLOUD_PROVIDER_DISCONNECTED, reason)
}

func (account *SCloudaccount) markAllProvidersDisconnected(ctx context.Context, userCred mcclient.TokenCredential) error {
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
	return account.SetStatus(ctx, userCred, api.CLOUD_PROVIDER_CONNECTED, "")
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

func (acnt *SCloudaccount) CanSync() bool {
	if acnt.SyncStatus == api.CLOUD_PROVIDER_SYNC_STATUS_QUEUED || acnt.SyncStatus == api.CLOUD_PROVIDER_SYNC_STATUS_SYNCING || acnt.getSyncStatus2() == api.CLOUD_PROVIDER_SYNC_STATUS_SYNCING {
		if acnt.LastSync.IsZero() || time.Now().Sub(acnt.LastSync) > time.Minute*30 {
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
			cloudaccountPendingSyncsMutex.Lock()
			if _, ok := cloudaccountPendingSyncs[id]; ok {
				cloudaccountPendingSyncsMutex.Unlock()
				continue
			}
			cloudaccountPendingSyncs[id] = struct{}{}
			cloudaccountPendingSyncsMutex.Unlock()
			RunSyncCloudAccountTask(ctx, func() {
				defer func() {
					cloudaccountPendingSyncsMutex.Lock()
					defer cloudaccountPendingSyncsMutex.Unlock()
					delete(cloudaccountPendingSyncs, id)
				}()
				log.Debugf("syncAccountStatus %s %s", id, name)
				idctx := context.WithValue(ctx, "id", id)
				lockman.LockObject(idctx, account)
				defer lockman.ReleaseObject(idctx, account)
				err := account.syncAccountStatus(idctx, userCred)
				if err != nil {
					log.Errorf("unable to syncAccountStatus for cloudaccount %s: %s", account.Id, err.Error())
				}
			})
		}
	}
}

func (account *SCloudaccount) probeAccountStatus(ctx context.Context, userCred mcclient.TokenCredential) ([]cloudprovider.SSubAccount, error) {
	manager, err := account.getProviderInternal(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "account.getProviderInternal")
	}
	balance, err := manager.GetBalance()
	if err != nil {
		switch err {
		case cloudprovider.ErrNotSupported:
			balance.Status = api.CLOUD_PROVIDER_HEALTH_NORMAL
		case cloudprovider.ErrNoBalancePermission:
			balance.Status = api.CLOUD_PROVIDER_HEALTH_NO_PERMISSION
		default:
			balance.Status = api.CLOUD_PROVIDER_HEALTH_UNKNOWN
			if account.Status != balance.Status {
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
		account.Balance = balance.Amount
		if !options.Options.CloudaccountHealthStatusCheck {
			balance.Status = api.CLOUD_PROVIDER_HEALTH_NORMAL
		}
		if len(account.Currency) == 0 && len(balance.Currency) > 0 {
			account.Currency = balance.Currency
		}
		account.AccountId = manager.GetAccountId()
		account.HealthStatus = balance.Status
		account.ProbeAt = timeutils.UtcNow()
		account.Version = version
		account.Sysinfo = sysInfo
		account.IamLoginUrl = iamLoginUrl

		return nil
	})
	if err != nil {
		log.Errorf("Failed to update db %s for account %s", err, account.Name)
	} else {
		db.OpsLog.LogSyncUpdate(account, diff, userCred)
	}

	return manager.GetSubAccounts()
}

func (account *SCloudaccount) importAllSubaccounts(ctx context.Context, userCred mcclient.TokenCredential, subAccounts []cloudprovider.SSubAccount) []SCloudprovider {
	for i := 0; i < len(subAccounts); i += 1 {
		_, _, err := account.importSubAccount(ctx, userCred, subAccounts[i])
		if err != nil {
			log.Errorf("importSubAccount fail %s", err)
		}
	}
	err := account.removeSubAccounts(ctx, userCred, subAccounts)
	if err != nil {
		log.Errorf("removeSubAccounts error: %v", err)
	}
	return account.GetCloudproviders()
}

func (acnt *SCloudaccount) setSubAccountStatus() error {
	if acnt.SubAccounts == nil || (len(acnt.SubAccounts.Accounts) == 0 && len(acnt.SubAccounts.Cloudregions) == 0) {
		return nil
	}
	accounts := []string{}
	accountNames := []string{}
	regionIds := []string{}
	for _, account := range acnt.SubAccounts.Accounts {
		if len(account.Account) > 0 {
			accounts = append(accounts, account.Account)
		} else if len(account.Name) > 0 {
			accountNames = append(accountNames, account.Name)
		}
	}
	for _, region := range acnt.SubAccounts.Cloudregions {
		if len(region.Id) > 0 && !strings.HasSuffix(region.Id, "/") {
			regionIds = append(regionIds, region.Id)
		}
	}

	providers := acnt.GetCloudproviders()
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
		accountQ := CloudaccountManager.Query().Equals("id", acnt.Id).SubQuery()
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

	_, err := db.Update(acnt, func() error {
		acnt.SubAccounts = nil
		return nil
	})

	return err
}

func (account *SCloudaccount) syncAccountStatus(ctx context.Context, userCred mcclient.TokenCredential) error {
	subaccounts, err := account.probeAccountStatus(ctx, userCred)
	if err != nil {
		account.markAllProvidersDisconnected(ctx, userCred)
		account.markAccountDisconected(ctx, userCred, errors.Wrapf(err, "probeAccountStatus").Error())
		return errors.Wrap(err, "account.probeAccountStatus")
	}
	account.markAccountConnected(ctx, userCred)
	providers := account.importAllSubaccounts(ctx, userCred, subaccounts)
	for i := range providers {
		if providers[i].GetEnabled() {
			_, err := providers[i].prepareCloudproviderRegions(ctx, userCred)
			if err != nil {
				providers[i].SetStatus(ctx, userCred, api.CLOUD_PROVIDER_DISCONNECTED, errors.Wrapf(err, "prepareCloudproviderRegions").Error())
			}
		}
	}
	return account.setSubAccountStatus()
}

var (
	cloudaccountPendingSyncs      = map[string]struct{}{}
	cloudaccountPendingSyncsMutex = &sync.Mutex{}
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

func (acnt *SCloudaccount) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	// override
	log.Infof("cloud account delete do nothing")
	return nil
}

func (acnt *SCloudaccount) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	acnt.SetStatus(ctx, userCred, api.CLOUD_PROVIDER_DELETED, "real delete")
	return acnt.purge(ctx, userCred)
}

func (acnt *SCloudaccount) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return acnt.StartCloudaccountDeleteTask(ctx, userCred, "")
}

func (acnt *SCloudaccount) StartCloudaccountDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "CloudAccountDeleteTask", acnt, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("%s", err)
		return err
	}
	acnt.SetStatus(ctx, userCred, api.CLOUD_PROVIDER_START_DELETE, "StartCloudaccountDeleteTask")
	task.ScheduleRun(nil)
	return nil
}

func (acnt *SCloudaccount) getSyncStatus2() string {
	cprs := CloudproviderRegionManager.Query().SubQuery()
	providers := CloudproviderManager.Query().Equals("cloudaccount_id", acnt.Id).SubQuery()

	q := cprs.Query().NotEquals("sync_status", api.CLOUD_PROVIDER_SYNC_STATUS_IDLE)
	q = q.Join(providers, sqlchemy.Equals(cprs.Field("cloudprovider_id"), providers.Field("id")))

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
			input.Scope = string(rbacscope.ScopeSystem)
		} else {
			input.Scope = string(rbacscope.ScopeDomain)
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
			input.Scope = string(rbacscope.ScopeSystem)
		} else {
			input.Scope = string(rbacscope.ScopeDomain)
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
	account.StartSyncCloudAccountInfoTask(ctx, userCred, syncRange, "", nil)
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
	account.StartSyncCloudAccountInfoTask(ctx, userCred, syncRange, "", nil)

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
				Scope: string(rbacscope.ScopeSystem),
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
			sqlchemy.Equals(q.Field("public_scope"), rbacscope.ScopeDomain),
		),
		// share_mode=system/public_scope=system
		sqlchemy.AND(
			// sqlchemy.Equals(q.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM),
			sqlchemy.IsTrue(q.Field("is_public")),
			sqlchemy.Equals(q.Field("public_scope"), rbacscope.ScopeSystem),
		),
	))

	return q
}

func (manager *SCloudaccountManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacscope.ScopeProject, rbacscope.ScopeDomain:
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
		_, err := validators.ValidateModel(ctx, userCred, CloudregionManager, &input.CloudregionId)
		if err != nil {
			return nil, err
		}
		params.Add(jsonutils.NewString(input.CloudregionId), "cloudregion_id")
	}
	if len(input.CloudproviderId) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, CloudproviderManager, &input.CloudproviderId)
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

func (manager *SCloudaccountManager) getBrandsOfCapability(region *SCloudregion, domainId string) ([]sBrandCapability, error) {
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
		PublicScope:   rbacscope.String2Scope(account.PublicScope),
		SharedDomains: account.GetSharedDomains(),
	}
}

func (manager *SCloudaccountManager) totalCount(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider) int {
	q := manager.Query()
	switch scope {
	case rbacscope.ScopeProject, rbacscope.ScopeDomain:
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

// 获取Azure Enrollment Accounts
func (acnt *SCloudaccount) GetDetailsEnrollmentAccounts(ctx context.Context, userCred mcclient.TokenCredential, query api.EnrollmentAccountQuery) ([]cloudprovider.SEnrollmentAccount, error) {
	if acnt.Provider != api.CLOUD_PROVIDER_AZURE {
		return nil, httperrors.NewNotSupportedError("%s not support", acnt.Provider)
	}
	provider, err := acnt.GetProvider(ctx)
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
func (acnt *SCloudaccount) PerformCreateSubscription(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SubscriptonCreateInput) (jsonutils.JSONObject, error) {
	if acnt.Provider != api.CLOUD_PROVIDER_AZURE {
		return nil, httperrors.NewNotSupportedError("%s not support create subscription", acnt.Provider)
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

	provider, err := acnt.GetProvider(ctx)
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
	return nil, acnt.StartSyncCloudAccountInfoTask(ctx, userCred, &syncRange, "", nil)
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
func (account *SCloudaccount) PerformProjectMapping(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudaccountProjectMappingInput) (jsonutils.JSONObject, error) {
	if len(input.ProjectMappingId) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, ProjectMappingManager, &input.ProjectMappingId)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateModel")
		}
		/*if len(account.ProjectMappingId) > 0 && account.ProjectMappingId != input.ProjectMappingId {
			return nil, httperrors.NewInputParameterError("account %s has aleady bind project mapping %s", account.Name, account.ProjectMappingId)
		}*/
		if (input.EnableProjectSync == nil || !*input.EnableProjectSync) && (input.EnableResourceSync == nil || !*input.EnableResourceSync) {
			return nil, errors.Wrap(httperrors.ErrInputParameter, "either enable_project_sync or enable_resource_sync must be set")
		}
	}

	if len(input.ProjectId) == 0 && !input.AutoCreateProjectForProvider {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "empty project_id")
	}
	if input.AutoCreateProject || input.AutoCreateProjectForProvider {
		t, err := db.TenantCacheManager.FetchTenantByIdOrNameInDomain(ctx, input.ProjectId, account.DomainId)
		if err != nil {
			return nil, errors.Wrap(err, "FetchTenantByIdOrNameInDomain")
		}
		input.ProjectId = t.Id
	}

	if len(input.ProjectId) > 0 {
		changeOwnerInput := apis.PerformChangeProjectOwnerInput{}
		changeOwnerInput.ProjectId = input.ProjectId
		_, err := account.PerformChangeProject(ctx, userCred, query, changeOwnerInput)
		if err != nil {
			return nil, errors.Wrapf(err, "PerformChangeProject")
		}
	}

	_, err := db.Update(account, func() error {
		account.AutoCreateProject = input.AutoCreateProject
		account.AutoCreateProjectForProvider = input.AutoCreateProjectForProvider
		account.ProjectMappingId = input.ProjectMappingId
		if input.EnableProjectSync != nil {
			account.EnableProjectSync = tristate.NewFromBool(*input.EnableProjectSync)
		}
		if input.EnableResourceSync != nil {
			account.EnableResourceSync = tristate.NewFromBool(*input.EnableResourceSync)
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "update")
	}
	return nil, refreshPmCaches()
}

// 同步云账号消息通知
func (account *SCloudaccount) EventNotify(ctx context.Context, userCred mcclient.TokenCredential, action notify.SAction) {
	var resourceType string
	resourceType = notify.TOPIC_RESOURCE_ACCOUNT_STATUS

	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:          account,
		ResourceType: resourceType,
		Action:       action,
		AdvanceDays:  0,
	})
}
