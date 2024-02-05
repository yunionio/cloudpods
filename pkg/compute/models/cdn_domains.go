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
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCDNDomainManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	db.SEnabledResourceBaseManager
	SManagedResourceBaseManager
	SDeletePreventableResourceBaseManager
}

var CDNDomainManager *SCDNDomainManager

func init() {
	CDNDomainManager = &SCDNDomainManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SCDNDomain{},
			"cdn_domains_tbl",
			"cdn_domain",
			"cdn_domains",
		),
	}
	CDNDomainManager.SetVirtualObject(CDNDomainManager)
}

type SCDNDomain struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase
	db.SExternalizedResourceBase

	SDeletePreventableResourceBase
	SManagedResourceBase

	Cname string `list:"user" width:"256"`

	// 源站信息
	Origins *cloudprovider.SCdnOrigins `list:"user" create:"domain_required"`
	// 服务类别
	ServiceType string `list:"user" width:"32" create:"domain_required"`
	// 加速区域
	Area string `list:"user" width:"32" update:"domain" create:"domain_required"`
	// 是否忽略参数
	CacheKeys *cloudprovider.SCDNCacheKeys `list:"user" create:"domain_optional"`
	// 是否分片回源
	RangeOriginPull *cloudprovider.SCDNRangeOriginPull `list:"user" create:"domain_optional"`
	// 缓存配置
	Cache *cloudprovider.SCDNCache `list:"user" create:"domain_optional"`
	// https配置
	HTTPS *cloudprovider.SCDNHttps `list:"user" create:"domain_optional"`
	// 强制跳转
	ForceRedirect *cloudprovider.SCDNForceRedirect `list:"user" create:"domain_optional"`
	// 防盗链配置
	Referer *cloudprovider.SCDNReferer `list:"user" create:"domain_optional"`
	// 浏览器缓存配置
	MaxAge *cloudprovider.SCDNMaxAge `list:"user" create:"domain_optional"`
}

func (manager *SCDNDomainManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudproviderManager},
	}
}

func (manager *SCDNDomainManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CDNDomainDetails {
	rows := make([]api.CDNDomainDetails, len(objs))
	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.CDNDomainDetails{
			VirtualResourceDetails: virtRows[i],
			ManagedResourceInfo:    managerRows[i],
		}
	}
	return rows
}

func (self *SCloudprovider) GetCDNDomains() ([]SCDNDomain, error) {
	q := CDNDomainManager.Query().Equals("manager_id", self.Id)
	domains := []SCDNDomain{}
	err := db.FetchModelObjects(CDNDomainManager, q, &domains)
	if err != nil {
		return nil, err
	}
	return domains, nil
}

func (self *SCloudprovider) SyncCDNDomains(ctx context.Context, userCred mcclient.TokenCredential, exts []cloudprovider.ICloudCDNDomain, xor bool) compare.SyncResult {
	lockman.LockRawObject(ctx, CDNDomainManager.Keyword(), self.Id)
	defer lockman.ReleaseRawObject(ctx, CDNDomainManager.Keyword(), self.Id)

	result := compare.SyncResult{}

	dbDomains, err := self.GetCDNDomains()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SCDNDomain, 0)
	commondb := make([]SCDNDomain, 0)
	commonext := make([]cloudprovider.ICloudCDNDomain, 0)
	added := make([]cloudprovider.ICloudCDNDomain, 0)

	err = compare.CompareSets(dbDomains, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudCDNDomain(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}
	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].SyncWithCloudCDNDomain(ctx, userCred, commonext[i])
			if err != nil {
				result.UpdateError(err)
				continue
			}
			result.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		_, err := self.newFromCloudCDNDomain(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

// 启用资源
func (self *SCDNDomain) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(self, ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

// 禁用资源
func (self *SCDNDomain) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(self, ctx, userCred, false)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (self *SCDNDomain) syncRemoveCloudCDNDomain(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	self.DeletePreventionOff(self, userCred)

	err := self.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "ValidateDeleteCondition")
	}
	err = self.RealDelete(ctx, userCred)
	if err != nil {
		return err
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    self,
		Action: notifyclient.ActionSyncDelete,
	})
	return nil
}

// 判断资源是否可以删除
func (self *SCDNDomain) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if self.DisableDelete.IsTrue() {
		return httperrors.NewInvalidStatusError("CDN is locked, cannot delete")
	}
	return self.SVirtualResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SCDNDomain) GetICloudCDNDomain(ctx context.Context) (cloudprovider.ICloudCDNDomain, error) {
	manager := self.GetCloudprovider()
	if manager == nil {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "GetCloudprovider")
	}
	provider, err := manager.GetProvider(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetProvider")
	}
	return provider.GetICloudCDNDomainByName(self.Name)
}

func (self *SCDNDomain) SyncWithCloudCDNDomain(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudCDNDomain) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		if options.Options.EnableSyncName {
			self.Name = ext.GetName()
		}
		self.Status = ext.GetStatus()
		self.Area = ext.GetArea()
		self.ServiceType = ext.GetServiceType()
		self.Cname = ext.GetCname()
		self.Origins = ext.GetOrigins()
		cacheKeys, err := ext.GetCacheKeys()
		if err == nil {
			self.CacheKeys = cacheKeys
		}
		if rangeOrigin, err := ext.GetRangeOriginPull(); err == nil {
			self.RangeOriginPull = rangeOrigin
		}
		if cache, err := ext.GetCache(); err == nil {
			self.Cache = cache
		}
		if https, err := ext.GetHTTPS(); err == nil {
			self.HTTPS = https
		}
		if fr, err := ext.GetForceRedirect(); err == nil {
			self.ForceRedirect = fr
		}
		if referer, err := ext.GetReferer(); err == nil {
			self.Referer = referer
		}
		if maxAge, err := ext.GetMaxAge(); err == nil {
			self.MaxAge = maxAge
		}
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	if len(diff) > 0 {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    self,
			Action: notifyclient.ActionSyncUpdate,
		})
	}

	if account := self.GetCloudaccount(); account != nil {
		syncVirtualResourceMetadata(ctx, userCred, self, ext, account.ReadOnly)
	}

	if provider := self.GetCloudprovider(); provider != nil {
		SyncCloudProject(ctx, userCred, self, provider.GetOwnerId(), ext, provider)
	}

	return nil
}

func (self *SCloudprovider) newFromCloudCDNDomain(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudCDNDomain) (*SCDNDomain, error) {
	domain := SCDNDomain{}
	domain.SetModelManager(CDNDomainManager, &domain)

	domain.ExternalId = ext.GetGlobalId()
	domain.ManagerId = self.Id
	domain.Name = ext.GetName()
	domain.Status = ext.GetStatus()
	domain.Area = ext.GetArea()
	domain.ServiceType = ext.GetServiceType()
	domain.Cname = ext.GetCname()
	domain.Origins = ext.GetOrigins()
	domain.CacheKeys, _ = ext.GetCacheKeys()
	domain.RangeOriginPull, _ = ext.GetRangeOriginPull()
	domain.Cache, _ = ext.GetCache()
	domain.HTTPS, _ = ext.GetHTTPS()
	domain.ForceRedirect, _ = ext.GetForceRedirect()
	domain.Referer, _ = ext.GetReferer()
	domain.MaxAge, _ = ext.GetMaxAge()

	err := CDNDomainManager.TableSpec().Insert(ctx, &domain)
	if err != nil {
		return nil, err
	}

	syncVirtualResourceMetadata(ctx, userCred, &domain, ext, false)
	SyncCloudProject(ctx, userCred, &domain, self.GetOwnerId(), ext, self)

	db.OpsLog.LogEvent(&domain, db.ACT_CREATE, domain.GetShortDesc(ctx), userCred)
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    &domain,
		Action: notifyclient.ActionSyncCreate,
	})

	return &domain, nil
}

func (manager *SCDNDomainManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.CDNDomainCreateInput,
) (api.CDNDomainCreateInput, error) {
	if len(input.CloudproviderId) == 0 {
		return input, httperrors.NewMissingParameterError("cloudprovider_id")
	}
	_provider, err := validators.ValidateModel(ctx, userCred, CloudproviderManager, &input.CloudproviderId)
	if err != nil {
		return input, err
	}
	input.ManagerId = input.CloudproviderId
	provider := _provider.(*SCloudprovider)
	pp, err := provider.GetProvider(ctx)
	if err != nil {
		return input, errors.Wrapf(err, "GetProvider")
	}
	if !cloudprovider.IsSupportCDN(pp) {
		return input, httperrors.NewNotSupportedError("%s not support cdn", provider.Provider)
	}
	input, err = GetRegionDriver(provider.Provider).ValidateCreateCdnData(ctx, userCred, input)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SCDNDomain) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.StartCdnCreateTask(ctx, userCred, "")
}

func (self *SCDNDomain) StartCdnCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "CDNDomainCreateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_CREATING, "")
	return task.ScheduleRun(nil)
}

func (self *SCDNDomain) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SCDNDomain) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	var err = func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "CDNDomainDeleteTask", self, userCred, nil, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		self.SetStatus(ctx, userCred, api.CDN_DOMAIN_STATUS_DELETE_FAILED, err.Error())
		return nil
	}
	return nil
}

func (self *SCDNDomain) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	self.SetStatus(ctx, userCred, api.CDN_DOMAIN_STATUS_DELETING, "")
	return nil
}

func (self *SCDNDomain) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

// 列出CDN域名
func (manager *SCDNDomainManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CDNDomainListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SCDNDomainManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	default:
		var err error
		q, err = manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}

		q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}
	}
	return q, httperrors.ErrNotFound
}

func (manager *SCDNDomainManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CDNDomainListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SCDNDomainManager) totalCount(
	ownerId mcclient.IIdentityProvider,
	scope rbacscope.TRbacScope,
	rangeObjs []db.IStandaloneModel,
	providers []string,
	brands []string,
	cloudEnv string,
) int {
	q := CDNDomainManager.Query()

	if scope != rbacscope.ScopeSystem && ownerId != nil {
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	}
	q = CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
	q = RangeObjectsFilter(q, rangeObjs, q.Field("cloudregion_id"), nil, q.Field("manager_id"), nil, nil)

	cnt, _ := q.CountWithError()

	return cnt
}

func (manager *SCDNDomainManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

// 同步域名状态
func (self *SCDNDomain) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.NatGatewaySyncstatusInput) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(self, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("CDN domain has %d task active, can't sync status", count)
	}

	return nil, self.StartSyncstatus(ctx, userCred, "")
}

func (self *SCDNDomain) StartSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return StartResourceSyncStatusTask(ctx, userCred, self, "CDNDomainSyncstatusTask", parentTaskId)
}

func (self *SCDNDomain) PerformRemoteUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.MongoDBRemoteUpdateInput) (jsonutils.JSONObject, error) {
	err := self.StartRemoteUpdateTask(ctx, userCred, (input.ReplaceTags != nil && *input.ReplaceTags), "")
	if err != nil {
		return nil, errors.Wrap(err, "StartRemoteUpdateTask")
	}
	return nil, nil
}

func (self *SCDNDomain) StartRemoteUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, replaceTags bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewBool(replaceTags), "replace_tags")
	task, err := taskman.TaskManager.NewTask(ctx, "CDNDomainRemoteUpdateTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_UPDATE_TAGS, "StartRemoteUpdateTask")
	return task.ScheduleRun(nil)
}

func (self *SCDNDomain) OnMetadataUpdated(ctx context.Context, userCred mcclient.TokenCredential) {
	if len(self.ExternalId) == 0 || options.Options.KeepTagLocalization {
		return
	}
	err := self.StartRemoteUpdateTask(ctx, userCred, true, "")
	if err != nil {
		log.Errorf("StartRemoteUpdateTask fail: %s", err)
	}
}
