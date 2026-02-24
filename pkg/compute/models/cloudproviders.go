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
	"sync"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/proxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudproviderManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	db.SProjectizedResourceBaseManager
	db.SExternalizedResourceBaseManager

	SProjectMappingResourceBaseManager
	SSyncableBaseResourceManager
}

var CloudproviderManager *SCloudproviderManager

func init() {
	CloudproviderManager = &SCloudproviderManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SCloudprovider{},
			"cloudproviders_tbl",
			"cloudprovider",
			"cloudproviders",
		),
	}
	CloudproviderManager.SetVirtualObject(CloudproviderManager)
}

type SCloudprovider struct {
	db.SEnabledStatusStandaloneResourceBase
	db.SProjectizedResourceBase
	db.SExternalizedResourceBase

	SSyncableBaseResource

	// 云端服务健康状态。例如欠费、项目冻结都属于不健康状态。
	//
	// | HealthStatus  | 说明                 |
	// |---------------|----------------------|
	// | normal        | 远端处于健康状态     |
	// | insufficient  | 不足按需资源余额     |
	// | suspended     | 远端处于冻结状态     |
	// | arrears       | 远端处于欠费状态     |
	// | unknown       | 未知状态，查询失败   |
	// | no permission | 没有权限获取账单信息 |
	//
	HealthStatus string `width:"16" charset:"ascii" default:"normal" nullable:"false" list:"domain"`

	// Hostname string `width:"64" charset:"ascii" nullable:"true"` // Column(VARCHAR(64, charset='ascii'), nullable=False)
	// port = Column(Integer, nullable=False)
	// Version string `width:"32" charset:"ascii" nullable:"true" list:"domain"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	// Sysinfo jsonutils.JSONObject `get:"domain"` // Column(JSONEncodedDict, nullable=True)

	AccessUrl string `width:"128" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	// 云账号的用户信息，例如用户名，access key等
	Account string `width:"256" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`
	// 云账号的密码信息，例如密码，access key secret等。该字段在数据库加密存储。Google需要存储秘钥证书,需要此字段比较长
	Secret string `length:"0" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`

	// 归属云账号ID
	CloudaccountId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`

	// ProjectId string `name:"tenant_id" width:"128" charset:"ascii" nullable:"true" list:"domain"`

	// LastSync time.Time `get:"domain" list:"domain"` // = Column(DateTime, nullable=True)

	// 云账号的平台信息
	Provider string `width:"64" charset:"ascii" list:"domain" create:"domain_required"`

	// 云上同步资源是否在本地被更改过配置, local: 更改过, cloud: 未更改过
	// example: local
	ProjectSrc string `width:"10" charset:"ascii" nullable:"false" list:"user" default:"cloud" json:"project_src"`

	SProjectMappingResourceBase
}

type pmCache struct {
	Id                      string
	CloudaccountId          string
	AccountProjectMappingId string
	ManagerProjectMappingId string

	AccountEnableProjectSync bool
	ManagerEnableProjectSync bool

	AccountEnableResourceSync bool
	ManagerEnableResourceSync bool
}

type sProjectMapping struct {
	*SProjectMapping
	EnableProjectSync  bool
	EnableResourceSync bool
}

func (cprvd *sProjectMapping) IsNeedResourceSync() bool {
	return cprvd.EnableResourceSync || !cprvd.EnableProjectSync
}

func (cprvd *sProjectMapping) IsNeedProjectSync() bool {
	return cprvd.EnableProjectSync
}

func (cprvd *pmCache) GetProjectMapping() (*sProjectMapping, error) {
	if len(cprvd.ManagerProjectMappingId) > 0 {
		pm, err := GetRuleMapping(cprvd.ManagerProjectMappingId)
		if err != nil {
			return nil, errors.Wrapf(err, "GetRuleMapping(%s)", cprvd.ManagerProjectMappingId)
		}
		ret := &sProjectMapping{
			SProjectMapping:    pm,
			EnableProjectSync:  cprvd.ManagerEnableProjectSync,
			EnableResourceSync: cprvd.ManagerEnableResourceSync,
		}
		return ret, nil
	}
	if len(cprvd.AccountProjectMappingId) > 0 {
		ret := &sProjectMapping{
			EnableProjectSync:  cprvd.AccountEnableProjectSync,
			EnableResourceSync: cprvd.AccountEnableResourceSync,
		}
		var err error
		ret.SProjectMapping, err = GetRuleMapping(cprvd.AccountProjectMappingId)
		return ret, err
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty project mapping id")
}

var pmCaches map[string]*pmCache = map[string]*pmCache{}

func refreshPmCaches() error {
	q := CloudproviderManager.Query().SubQuery()
	providers := q.Query(
		q.Field("cloudaccount_id"),
		q.Field("id"),
		q.Field("project_mapping_id").Label("manager_project_mapping_id"),
		q.Field("enable_project_sync").Label("manager_enable_project_sync"),
		q.Field("enable_resource_sync").Label("manager_enable_resource_sync"),
	)
	sq := CloudaccountManager.Query().SubQuery()
	mq := providers.LeftJoin(sq, sqlchemy.Equals(q.Field("cloudaccount_id"), sq.Field("id"))).
		AppendField(sq.Field("project_mapping_id").Label("account_project_mapping_id")).
		AppendField(sq.Field("enable_project_sync").Label("account_enable_project_sync")).
		AppendField(sq.Field("enable_resource_sync").Label("account_enable_resource_sync"))
	caches := []pmCache{}
	err := mq.All(&caches)
	if err != nil {
		return errors.Wrapf(err, "q.All")
	}
	for i := range caches {
		pmCaches[caches[i].Id] = &caches[i]
	}
	return nil
}

func (cprvd *SCloudaccount) GetProjectMapping() (*sProjectMapping, error) {
	cache, err := func() (*pmCache, error) {
		for id := range pmCaches {
			if pmCaches[id].CloudaccountId == cprvd.Id {
				return pmCaches[id], nil
			}
		}
		err := refreshPmCaches()
		if err != nil {
			return nil, errors.Wrapf(err, "refreshPmCaches")
		}
		for id := range pmCaches {
			if pmCaches[id].CloudaccountId == cprvd.Id {
				return pmCaches[id], nil
			}
		}
		return nil, cloudprovider.ErrNotFound
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "get project mapping cache")
	}
	return cache.GetProjectMapping()
}

func (cprvd *SCloudprovider) GetProjectMapping() (*sProjectMapping, error) {
	cache, err := func() (*pmCache, error) {
		mp, ok := pmCaches[cprvd.Id]
		if ok {
			return mp, nil
		}
		err := refreshPmCaches()
		if err != nil {
			return nil, errors.Wrapf(err, "refreshPmCaches")
		}
		return pmCaches[cprvd.Id], nil
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "get project mapping cache")
	}
	return cache.GetProjectMapping()
}

func (cprvd *SCloudprovider) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if cprvd.GetEnabled() {
		return httperrors.NewInvalidStatusError("provider is enabled")
	}
	if cprvd.SyncStatus != api.CLOUD_PROVIDER_SYNC_STATUS_IDLE {
		return httperrors.NewInvalidStatusError("provider is not idle")
	}
	return cprvd.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (manager *SCloudproviderManager) GetPublicProviderIdsQuery() *sqlchemy.SSubQuery {
	return manager.GetProviderIdsQuery(tristate.True, tristate.None, nil, nil)
}

func (manager *SCloudproviderManager) GetPrivateProviderIdsQuery() *sqlchemy.SSubQuery {
	return manager.GetProviderIdsQuery(tristate.False, tristate.False, nil, nil)
}

func (manager *SCloudproviderManager) GetOnPremiseProviderIdsQuery() *sqlchemy.SSubQuery {
	return manager.GetProviderIdsQuery(tristate.None, tristate.True, nil, nil)
}

func (manager *SCloudproviderManager) GetPrivateOrOnPremiseProviderIdsQuery() *sqlchemy.SSubQuery {
	return manager.GetProviderIdsQuery(tristate.False, tristate.None, nil, nil)
}

func (manager *SCloudproviderManager) GetProviderIdsQuery(isPublic tristate.TriState, isOnPremise tristate.TriState, providers []string, brands []string) *sqlchemy.SSubQuery {
	return manager.GetProviderFieldQuery("id", isPublic, isOnPremise, providers, brands)
}

func (manager *SCloudproviderManager) GetPublicProviderProvidersQuery() *sqlchemy.SSubQuery {
	return manager.GetProviderProvidersQuery(tristate.True, tristate.None)
}

func (manager *SCloudproviderManager) GetPrivateProviderProvidersQuery() *sqlchemy.SSubQuery {
	return manager.GetProviderProvidersQuery(tristate.False, tristate.False)
}

func (manager *SCloudproviderManager) GetOnPremiseProviderProvidersQuery() *sqlchemy.SSubQuery {
	return manager.GetProviderProvidersQuery(tristate.None, tristate.True)
}

func (manager *SCloudproviderManager) GetProviderProvidersQuery(isPublic tristate.TriState, isOnPremise tristate.TriState) *sqlchemy.SSubQuery {
	return manager.GetProviderFieldQuery("provider", isPublic, isOnPremise, nil, nil)
}

func (manager *SCloudproviderManager) GetProviderFieldQuery(field string, isPublic tristate.TriState, isOnPremise tristate.TriState, providers []string, brands []string) *sqlchemy.SSubQuery {
	q := manager.Query(field).Distinct()
	account := CloudaccountManager.Query().SubQuery()
	q = q.Join(account, sqlchemy.Equals(
		account.Field("id"), q.Field("cloudaccount_id")),
	)
	if isPublic.IsTrue() {
		q = q.Filter(sqlchemy.IsTrue(account.Field("is_public_cloud")))
	} else if isPublic.IsFalse() {
		q = q.Filter(sqlchemy.IsFalse(account.Field("is_public_cloud")))
	}
	if isOnPremise.IsTrue() {
		q = q.Filter(sqlchemy.IsTrue(account.Field("is_on_premise")))
	} else if isOnPremise.IsFalse() {
		q = q.Filter(sqlchemy.IsFalse(account.Field("is_on_premise")))
	}
	if len(providers) > 0 || len(brands) > 0 {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(account.Field("provider"), providers),
			sqlchemy.In(account.Field("brand"), brands),
		))
	}
	return q.SubQuery()
}

func CloudProviderFilter(q *sqlchemy.SQuery, managerIdField sqlchemy.IQueryField, providers []string, brands []string, cloudEnv string) *sqlchemy.SQuery {
	if len(cloudEnv) == 0 && len(providers) == 0 && len(brands) == 0 {
		return q
	}
	isPublic := tristate.None
	isOnPremise := tristate.None
	includeOneCloud := false
	switch cloudEnv {
	case api.CLOUD_ENV_PUBLIC_CLOUD:
		isPublic = tristate.True
	case api.CLOUD_ENV_PRIVATE_CLOUD:
		isPublic = tristate.False
		isOnPremise = tristate.False
	case api.CLOUD_ENV_ON_PREMISE:
		isOnPremise = tristate.True
		includeOneCloud = true
	default:
		includeOneCloud = true
	}
	if includeOneCloud && len(providers) > 0 && !utils.IsInStringArray(api.CLOUD_PROVIDER_ONECLOUD, providers) {
		includeOneCloud = false
	}
	if includeOneCloud && len(brands) > 0 && !utils.IsInStringArray(api.CLOUD_PROVIDER_ONECLOUD, brands) {
		includeOneCloud = false
	}
	subq := CloudproviderManager.GetProviderIdsQuery(isPublic, isOnPremise, providers, brands)
	if includeOneCloud {
		return q.Filter(sqlchemy.OR(
			sqlchemy.In(managerIdField, subq),
			sqlchemy.IsNullOrEmpty(managerIdField),
		))
	} else {
		return q.Filter(sqlchemy.In(managerIdField, subq))
	}
}

func (cprvd *SCloudprovider) CleanSchedCache() {
	hosts := []SHost{}
	q := HostManager.Query().Equals("manager_id", cprvd.Id)
	if err := db.FetchModelObjects(HostManager, q, &hosts); err != nil {
		log.Errorf("failed to get hosts for cloudprovider %s error: %v", cprvd.Name, err)
		return
	}
	for _, host := range hosts {
		host.ClearSchedDescCache()
	}
}

func (cprvd *SCloudprovider) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudproviderUpdateInput) (api.CloudproviderUpdateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceBaseUpdateInput, err = cprvd.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusStandaloneResourceBase.ValidateUpdateData")
	}
	return input, nil
}

// +onecloud:swagger-gen-ignore
func (cprvd *SCloudproviderManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.CloudproviderCreateInput) (api.CloudproviderCreateInput, error) {
	return input, httperrors.NewUnsupportOperationError("Directly creating cloudprovider is not supported, create cloudaccount instead")
}

func (cprvd *SCloudprovider) getAccessUrl() string {
	if len(cprvd.AccessUrl) > 0 {
		return cprvd.AccessUrl
	}
	account, _ := cprvd.GetCloudaccount()
	if account != nil {
		return account.AccessUrl
	}
	return ""
}

func (cprvd *SCloudprovider) getPassword() (string, error) {
	if len(cprvd.Secret) == 0 {
		account, err := cprvd.GetCloudaccount()
		if err != nil {
			return "", errors.Wrapf(err, "GetCloudaccount")
		}
		return account.getPassword()
	}
	return utils.DescryptAESBase64(cprvd.Id, cprvd.Secret)
}

func getTenant(ctx context.Context, projectId string, name string, domainId string) (*db.STenant, error) {
	if len(projectId) > 0 {
		tenant, err := db.TenantCacheManager.FetchTenantById(ctx, projectId)
		if err != nil {
			return nil, errors.Wrap(err, "TenantCacheManager.FetchTenantById")
		}
		return tenant, nil
	}
	if len(name) == 0 {
		return nil, errors.Error("cannot syncProject for empty name")
	}
	return db.TenantCacheManager.FetchTenantByNameInDomain(ctx, name, domainId)
}

func createTenant(ctx context.Context, name, domainId, desc string) (string, string, error) {
	s := auth.GetAdminSession(ctx, options.Options.Region)
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(name), "generate_name")

	params.Add(jsonutils.NewString(domainId), "domain_id")
	params.Add(jsonutils.NewString(desc), "description")

	resp, err := identity.Projects.Create(s, params)
	if err != nil {
		return "", "", errors.Wrap(err, "Projects.Create")
	}
	projectId, err := resp.GetString("id")
	if err != nil {
		return "", "", errors.Wrap(err, "resp.GetString")
	}
	_, err = db.TenantCacheManager.FetchTenantById(ctx, projectId)
	if err != nil {
		log.Errorf("fetch tenant %s error: %v", name, err)
	}
	return domainId, projectId, nil
}

func (cprvd *SCloudprovider) syncProject(ctx context.Context, userCred mcclient.TokenCredential) error {
	account, err := cprvd.GetCloudaccount()
	if err != nil {
		return errors.Wrapf(err, "GetCloudaccount")
	}

	desc := fmt.Sprintf("auto create from cloud provider %s (%s)", cprvd.Name, cprvd.Id)
	domainId, projectId, err := account.getOrCreateTenant(ctx, cprvd.Name, "", cprvd.ProjectId, desc)
	if err != nil {
		return errors.Wrap(err, "getOrCreateTenant")
	}

	return cprvd.saveProject(userCred, domainId, projectId, true)
}

func (cprvd *SCloudprovider) saveProject(userCred mcclient.TokenCredential, domainId, projectId string, auto bool) error {
	if projectId != cprvd.ProjectId {
		diff, err := db.Update(cprvd, func() error {
			cprvd.DomainId = domainId
			cprvd.ProjectId = projectId
			// 自动改变项目时不改变配置，仅在performChangeOnwer（手动更改项目）时改变
			if !auto {
				cprvd.ProjectSrc = string(apis.OWNER_SOURCE_LOCAL)
			}
			return nil
		})
		if err != nil {
			log.Errorf("update projectId fail: %s", err)
			return err
		}
		db.OpsLog.LogEvent(cprvd, db.ACT_UPDATE, diff, userCred)
	}
	return nil
}

type SSyncRange struct {
	api.SyncRangeInput
}

func (sr *SSyncRange) GetRegionIds() ([]string, error) {
	regionIds := []string{}
	if len(sr.Host) == 0 && len(sr.Zone) == 0 && len(sr.Region) == 0 {
		return regionIds, nil
	}
	hostQ := HostManager.Query().SubQuery()
	hosts := hostQ.Query().Filter(sqlchemy.OR(
		sqlchemy.In(hostQ.Field("id"), sr.Host),
		sqlchemy.In(hostQ.Field("name"), sr.Host),
	)).SubQuery()
	zoneQ := ZoneManager.Query().SubQuery()
	zones := zoneQ.Query().Filter(sqlchemy.OR(
		sqlchemy.In(zoneQ.Field("id"), sr.Zone),
		sqlchemy.In(zoneQ.Field("name"), sr.Zone),
		sqlchemy.In(zoneQ.Field("id"), hosts.Query(hosts.Field("zone_id")).SubQuery()),
	)).SubQuery()
	regionQ := CloudregionManager.Query().SubQuery()
	q := regionQ.Query(regionQ.Field("id")).Filter(sqlchemy.OR(
		sqlchemy.In(regionQ.Field("id"), sr.Region),
		sqlchemy.In(regionQ.Field("name"), sr.Region),
		sqlchemy.In(regionQ.Field("id"), zones.Query(zones.Field("cloudregion_id")).SubQuery()),
	))
	rows, err := q.Rows()
	if err != nil {
		return nil, errors.Wrap(err, "q.Rows")
	}
	defer rows.Close()
	for rows.Next() {
		var regionId string
		err = rows.Scan(&regionId)
		if err != nil {
			return nil, errors.Wrap(err, "rows.Scan")
		}
		regionIds = append(regionIds, regionId)
	}
	return regionIds, nil
}

func (sr *SSyncRange) NeedSyncResource(res string) bool {
	if sr.FullSync {
		return true
	}

	if len(sr.Resources) == 0 {
		return true
	}
	return utils.IsInStringArray(res, sr.Resources)
}

func (sr *SSyncRange) NeedSyncInfo() bool {
	if sr.FullSync {
		return true
	}
	if len(sr.Region) > 0 || len(sr.Zone) > 0 || len(sr.Host) > 0 || len(sr.Resources) > 0 {
		return true
	}
	return false
}

func (sr *SSyncRange) normalizeRegionIds(ctx context.Context) error {
	for i := 0; i < len(sr.Region); i += 1 {
		obj, err := CloudregionManager.FetchByIdOrName(ctx, nil, sr.Region[i])
		if err != nil {
			if err == sql.ErrNoRows {
				return httperrors.NewResourceNotFoundError("Region %s not found", sr.Region[i])
			} else {
				return err
			}
		}
		sr.Region[i] = obj.GetId()
	}
	return nil
}

func (sr *SSyncRange) normalizeZoneIds(ctx context.Context) error {
	for i := 0; i < len(sr.Zone); i += 1 {
		obj, err := ZoneManager.FetchByIdOrName(ctx, nil, sr.Zone[i])
		if err != nil {
			if err == sql.ErrNoRows {
				return httperrors.NewResourceNotFoundError("Zone %s not found", sr.Zone[i])
			} else {
				return err
			}
		}
		zone := obj.(*SZone)
		region, _ := zone.GetRegion()
		if region == nil {
			continue
		}
		sr.Zone[i] = zone.GetId()
		if !utils.IsInStringArray(region.Id, sr.Region) {
			sr.Region = append(sr.Region, region.Id)
		}
	}
	return nil
}

func (sr *SSyncRange) normalizeHostIds(ctx context.Context) error {
	for i := 0; i < len(sr.Host); i += 1 {
		obj, err := HostManager.FetchByIdOrName(ctx, nil, sr.Host[i])
		if err != nil {
			if err == sql.ErrNoRows {
				return httperrors.NewResourceNotFoundError("Host %s not found", sr.Host[i])
			} else {
				return err
			}
		}
		host := obj.(*SHost)
		zone, _ := host.GetZone()
		if zone == nil {
			continue
		}
		region, _ := zone.GetRegion()
		if region == nil {
			continue
		}
		sr.Host[i] = host.GetId()
		if !utils.IsInStringArray(zone.Id, sr.Zone) {
			sr.Zone = append(sr.Zone, zone.Id)
		}
		if !utils.IsInStringArray(region.Id, sr.Region) {
			sr.Region = append(sr.Region, region.Id)
		}
	}
	return nil
}

func (sr *SSyncRange) Normalize(ctx context.Context) error {
	if sr.Region != nil && len(sr.Region) > 0 {
		err := sr.normalizeRegionIds(ctx)
		if err != nil {
			return err
		}
	} else {
		sr.Region = make([]string, 0)
	}
	if sr.Zone != nil && len(sr.Zone) > 0 {
		err := sr.normalizeZoneIds(ctx)
		if err != nil {
			return err
		}
	} else {
		sr.Zone = make([]string, 0)
	}
	if sr.Host != nil && len(sr.Host) > 0 {
		err := sr.normalizeHostIds(ctx)
		if err != nil {
			return err
		}
	} else {
		sr.Host = make([]string, 0)
	}
	return nil
}

func (cprvd *SCloudprovider) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SyncRangeInput) (jsonutils.JSONObject, error) {
	if !cprvd.GetEnabled() {
		return nil, httperrors.NewInvalidStatusError("Cloudprovider disabled")
	}
	account, err := cprvd.GetCloudaccount()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudaccount")
	}
	if !account.GetEnabled() {
		return nil, httperrors.NewInvalidStatusError("Cloudaccount disabled")
	}
	syncRange := SSyncRange{input}
	if syncRange.FullSync || len(syncRange.Region) > 0 || len(syncRange.Zone) > 0 || len(syncRange.Host) > 0 || len(syncRange.Resources) > 0 {
		syncRange.DeepSync = true
	}
	syncRange.SkipSyncResources = []string{}
	if account.SkipSyncResources != nil {
		for _, res := range *account.SkipSyncResources {
			syncRange.SkipSyncResources = append(syncRange.SkipSyncResources, res)
		}
	}
	if cprvd.CanSync() || syncRange.Force {
		return nil, cprvd.StartSyncCloudProviderInfoTask(ctx, userCred, &syncRange, "")
	}
	return nil, httperrors.NewInvalidStatusError("Unable to synchronize frequently")
}

func (cprvd *SCloudprovider) StartSyncCloudProviderInfoTask(ctx context.Context, userCred mcclient.TokenCredential, syncRange *SSyncRange, parentTaskId string) error {
	params := jsonutils.NewDict()
	if syncRange != nil {
		params.Add(jsonutils.Marshal(syncRange), "sync_range")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "CloudProviderSyncInfoTask", cprvd, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	if cloudaccount, _ := cprvd.GetCloudaccount(); cloudaccount != nil {
		cloudaccount.MarkSyncing(userCred)
	}
	cprvd.markStartSync(userCred, syncRange)
	db.OpsLog.LogEvent(cprvd, db.ACT_SYNC_HOST_START, "", userCred)
	return task.ScheduleRun(nil)
}

func (cprvd *SCloudprovider) PerformChangeProject(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeProjectOwnerInput) (jsonutils.JSONObject, error) {
	project := input.ProjectId
	domain := input.ProjectDomainId
	if len(domain) == 0 {
		domain = cprvd.DomainId
	}

	tenant, err := db.TenantCacheManager.FetchTenantByIdOrNameInDomain(ctx, project, domain)
	if err != nil {
		return nil, httperrors.NewNotFoundError("project %s not found", project)
	}

	if cprvd.ProjectId == tenant.Id {
		return nil, nil
	}

	account, err := cprvd.GetCloudaccount()
	if err != nil {
		return nil, err
	}
	if cprvd.DomainId != tenant.DomainId {
		if !db.IsAdminAllowPerform(ctx, userCred, cprvd, "change-project") {
			return nil, httperrors.NewForbiddenError("not allow to change project across domain")
		}
		if account.ShareMode == api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN && account.DomainId != tenant.DomainId {
			return nil, httperrors.NewInvalidStatusError("cannot change to a different domain from a private cloud account")
		}
		// if account's public_scope=domain and share_mode=provider_domain, only allow to share to specific domains
		if account.PublicScope == string(rbacscope.ScopeDomain) {
			sharedDomains := account.GetSharedDomains()
			if !utils.IsInStringArray(tenant.DomainId, sharedDomains) && account.DomainId != tenant.DomainId {
				return nil, errors.Wrap(httperrors.ErrForbidden, "cannot set to domain outside of the shared domains")
			}
		}
		// otherwise, allow change project across domain
	}

	notes := struct {
		OldProjectId string
		OldDomainId  string
		NewProjectId string
		NewProject   string
		NewDomainId  string
		NewDomain    string
	}{
		OldProjectId: cprvd.ProjectId,
		OldDomainId:  cprvd.DomainId,
		NewProjectId: tenant.Id,
		NewProject:   tenant.Name,
		NewDomainId:  tenant.DomainId,
		NewDomain:    tenant.Domain,
	}

	err = cprvd.saveProject(userCred, tenant.DomainId, tenant.Id, false)
	if err != nil {
		log.Errorf("Update cloudprovider error: %v", err)
		return nil, httperrors.NewGeneralError(err)
	}

	logclient.AddSimpleActionLog(cprvd, logclient.ACT_CHANGE_OWNER, notes, userCred, true)

	return nil, cprvd.StartSyncCloudProviderInfoTask(ctx, userCred, &SSyncRange{SyncRangeInput: api.SyncRangeInput{
		FullSync: true, DeepSync: true,
	}}, "")
}

func (cprvd *SCloudprovider) markStartingSync(userCred mcclient.TokenCredential, syncRange *SSyncRange) error {
	_, err := db.Update(cprvd, func() error {
		cprvd.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_QUEUING
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.Update")
	}
	cprs := cprvd.GetCloudproviderRegions()
	for i := range cprs {
		if cprs[i].Enabled {
			err := cprs[i].markStartingSync(userCred, syncRange)
			if err != nil {
				return errors.Wrap(err, "cprs[i].markStartingSync")
			}
		}
	}
	return nil
}

func (cprvd *SCloudprovider) markStartSync(userCred mcclient.TokenCredential, syncRange *SSyncRange) error {
	_, err := db.Update(cprvd, func() error {
		cprvd.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_QUEUED
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	cprs := cprvd.GetCloudproviderRegions()
	for i := range cprs {
		if cprs[i].Enabled {
			err := cprs[i].markStartingSync(userCred, syncRange)
			if err != nil {
				return errors.Wrap(err, "cprs[i].markStartingSync")
			}
		}
	}
	return nil
}

func (cprvd *SCloudprovider) markSyncing(userCred mcclient.TokenCredential) error {
	_, err := db.Update(cprvd, func() error {
		cprvd.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_SYNCING
		cprvd.LastSync = timeutils.UtcNow()
		cprvd.LastSyncEndAt = time.Time{}
		return nil
	})
	if err != nil {
		log.Errorf("Failed to markSyncing error: %v", err)
		return err
	}
	return nil
}

func (cprvd *SCloudprovider) markEndSyncWithLock(ctx context.Context, userCred mcclient.TokenCredential, deepSync bool) error {
	err := func() error {
		lockman.LockObject(ctx, cprvd)
		defer lockman.ReleaseObject(ctx, cprvd)

		if cprvd.SyncStatus == api.CLOUD_PROVIDER_SYNC_STATUS_IDLE {
			return nil
		}

		if cprvd.GetSyncStatus2() != api.CLOUD_PROVIDER_SYNC_STATUS_IDLE {
			return nil
		}

		err := cprvd.markEndSync(userCred)
		if err != nil {
			return err
		}
		return nil
	}()

	if err != nil {
		return err
	}

	account, err := cprvd.GetCloudaccount()
	if err != nil {
		return errors.Wrapf(err, "GetCloudaccount")
	}
	return account.MarkEndSyncWithLock(ctx, userCred, deepSync)
}

func (cprvd *SCloudprovider) markEndSync(userCred mcclient.TokenCredential) error {
	_, err := db.Update(cprvd, func() error {
		cprvd.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
		cprvd.LastSyncEndAt = timeutils.UtcNow()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "markEndSync")
	}
	return nil
}

func (cprvd *SCloudprovider) cancelStartingSync(userCred mcclient.TokenCredential) error {
	if cprvd.SyncStatus == api.CLOUD_PROVIDER_SYNC_STATUS_QUEUING {
		cprs := cprvd.GetCloudproviderRegions()
		for i := range cprs {
			err := cprs[i].cancelStartingSync(userCred)
			if err != nil {
				return errors.Wrap(err, "cprs[i].cancelStartingSync")
			}
		}
		_, err := db.Update(cprvd, func() error {
			cprvd.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "db.Update")
		}
	}
	return nil
}

func (cprvd *SCloudprovider) GetProviderFactory() (cloudprovider.ICloudProviderFactory, error) {
	return cloudprovider.GetProviderFactory(cprvd.Provider)
}

func (cprvd *SCloudprovider) GetProvider(ctx context.Context) (cloudprovider.ICloudProvider, error) {
	if !cprvd.GetEnabled() {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "Cloud provider is not enabled")
	}

	accessUrl := cprvd.getAccessUrl()
	passwd, err := cprvd.getPassword()
	if err != nil {
		return nil, err
	}

	account, err := cprvd.GetCloudaccount()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudaccount")
	}
	return cloudprovider.GetProvider(cloudprovider.ProviderConfig{
		Id:        cprvd.Id,
		Name:      cprvd.Name,
		Vendor:    cprvd.Provider,
		URL:       accessUrl,
		Account:   cprvd.Account,
		Secret:    passwd,
		ProxyFunc: account.proxyFunc(),

		AliyunResourceGroupIds: options.Options.AliyunResourceGroups,

		ReadOnly: account.ReadOnly,

		RegionId: account.regionId(),
		Options:  account.Options,

		UpdatePermission: account.UpdatePermission(ctx),
	})
}

func (cprvd *SCloudprovider) savePassword(secret string) error {
	sec, err := utils.EncryptAESBase64(cprvd.Id, secret)
	if err != nil {
		return err
	}

	_, err = db.Update(cprvd, func() error {
		cprvd.Secret = sec
		return nil
	})
	return err
}

func (cprvd *SCloudprovider) GetCloudaccount() (*SCloudaccount, error) {
	obj, err := CloudaccountManager.FetchById(cprvd.CloudaccountId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById(%s)", cprvd.CloudaccountId)
	}
	return obj.(*SCloudaccount), nil
}

func (manager *SCloudproviderManager) FetchCloudproviderById(providerId string) *SCloudprovider {
	providerObj, err := manager.FetchById(providerId)
	if err != nil {
		return nil
	}
	return providerObj.(*SCloudprovider)
}

func IsProviderAccountEnabled(providerId string) bool {
	if len(providerId) == 0 {
		return true
	}
	return CloudproviderManager.IsProviderAccountEnabled(providerId)
}

func (manager *SCloudproviderManager) IsProviderAccountEnabled(providerId string) bool {
	providerObj := manager.FetchCloudproviderById(providerId)
	if providerObj == nil {
		return false
	}
	if !providerObj.GetEnabled() {
		return false
	}
	account, _ := providerObj.GetCloudaccount()
	if account == nil {
		return false
	}
	return account.GetEnabled()
}

func (manager *SCloudproviderManager) FetchCloudproviderByIdOrName(ctx context.Context, providerId string) *SCloudprovider {
	providerObj, err := manager.FetchByIdOrName(ctx, nil, providerId)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("%s", err)
		}
		return nil
	}
	return providerObj.(*SCloudprovider)
}

func (cm *SCloudproviderManager) query(manager db.IModelManager, field string, providerIds []string, filter func(*sqlchemy.SQuery) *sqlchemy.SQuery) *sqlchemy.SSubQuery {
	q := manager.Query()

	if filter != nil {
		q = filter(q)
	}

	sq := q.SubQuery()

	key := "manager_id"
	if manager.Keyword() == CloudproviderRegionManager.Keyword() {
		key = "cloudprovider_id"
	}

	return sq.Query(
		sq.Field(key),
		sqlchemy.COUNT(field),
	).In(key, providerIds).GroupBy(sq.Field(key)).SubQuery()
}

type SCloudproviderUsageCount struct {
	Id string
	api.SCloudproviderUsage
}

func (cm *SCloudproviderManager) TotalResourceCount(providerIds []string) (map[string]api.SCloudproviderUsage, error) {
	ret := map[string]api.SCloudproviderUsage{}

	guestSQ := cm.query(GuestManager, "guest_cnt", providerIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		hosts := HostManager.Query().SubQuery()
		sq := q.SubQuery()
		return sq.Query(
			sq.Field("id").Label("guest_id"),
			sq.Field("host_id").Label("host_id"),
			hosts.Field("manager_id").Label("manager_id"),
		).LeftJoin(hosts, sqlchemy.Equals(sq.Field("host_id"), hosts.Field("id")))
	})

	hostSQ := cm.query(HostManager, "host_cnt", providerIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.IsFalse("is_emulated")
	})

	vpcSQ := cm.query(VpcManager, "vpc_cnt", providerIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.IsFalse("is_emulated")
	})

	storageSQ := cm.query(StorageManager, "storage_cnt", providerIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.IsFalse("is_emulated")
	})

	storagecacheSQ := cm.query(StoragecacheManager, "storage_cache_cnt", providerIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.IsFalse("is_emulated")
	})

	redisSQ := cm.query(ElasticcacheManager, "elasticcache_cnt", providerIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		vpcs := VpcManager.Query().SubQuery()
		sq := q.SubQuery()
		return sq.Query(
			sq.Field("id").Label("redis_id"),
			sq.Field("vpc_id").Label("vpc_id"),
			vpcs.Field("manager_id").Label("manager_id"),
		).LeftJoin(vpcs, sqlchemy.Equals(sq.Field("vpc_id"), vpcs.Field("id")))
	})

	eipSQ := cm.query(ElasticipManager, "eip_cnt", providerIds, nil)
	snapshotSQ := cm.query(SnapshotManager, "snapshot_cnt", providerIds, nil)
	lbSQ := cm.query(LoadbalancerManager, "loadbalancer_cnt", providerIds, nil)
	rdsSQ := cm.query(DBInstanceManager, "dbinstance_cnt", providerIds, nil)
	projectSQ := cm.query(ExternalProjectManager, "project_cnt", providerIds, nil)
	sregionSQ := cm.query(CloudproviderRegionManager, "sync_region_cnt", providerIds, nil)

	providers := cm.Query().SubQuery()
	providerQ := providers.Query(
		sqlchemy.SUM("guest_count", guestSQ.Field("guest_cnt")),
		sqlchemy.SUM("host_count", hostSQ.Field("host_cnt")),
		sqlchemy.SUM("vpc_count", vpcSQ.Field("vpc_cnt")),
		sqlchemy.SUM("storage_count", storageSQ.Field("storage_cnt")),
		sqlchemy.SUM("storage_cache_count", storagecacheSQ.Field("storage_cache_cnt")),
		sqlchemy.SUM("eip_count", eipSQ.Field("eip_cnt")),
		sqlchemy.SUM("snapshot_count", snapshotSQ.Field("snapshot_cnt")),
		sqlchemy.SUM("loadbalancer_count", lbSQ.Field("loadbalancer_cnt")),
		sqlchemy.SUM("dbinstance_count", rdsSQ.Field("dbinstance_cnt")),
		sqlchemy.SUM("elasticcache_count", redisSQ.Field("elasticcache_cnt")),
		sqlchemy.SUM("project_count", projectSQ.Field("project_cnt")),
		sqlchemy.SUM("sync_region_count", sregionSQ.Field("sync_region_cnt")),
	)

	providerQ.AppendField(providerQ.Field("id"))

	providerQ = providerQ.LeftJoin(guestSQ, sqlchemy.Equals(providerQ.Field("id"), guestSQ.Field("manager_id")))
	providerQ = providerQ.LeftJoin(hostSQ, sqlchemy.Equals(providerQ.Field("id"), hostSQ.Field("manager_id")))
	providerQ = providerQ.LeftJoin(vpcSQ, sqlchemy.Equals(providerQ.Field("id"), vpcSQ.Field("manager_id")))
	providerQ = providerQ.LeftJoin(storageSQ, sqlchemy.Equals(providerQ.Field("id"), storageSQ.Field("manager_id")))
	providerQ = providerQ.LeftJoin(storagecacheSQ, sqlchemy.Equals(providerQ.Field("id"), storagecacheSQ.Field("manager_id")))
	providerQ = providerQ.LeftJoin(eipSQ, sqlchemy.Equals(providerQ.Field("id"), eipSQ.Field("manager_id")))
	providerQ = providerQ.LeftJoin(snapshotSQ, sqlchemy.Equals(providerQ.Field("id"), snapshotSQ.Field("manager_id")))
	providerQ = providerQ.LeftJoin(lbSQ, sqlchemy.Equals(providerQ.Field("id"), lbSQ.Field("manager_id")))
	providerQ = providerQ.LeftJoin(rdsSQ, sqlchemy.Equals(providerQ.Field("id"), rdsSQ.Field("manager_id")))
	providerQ = providerQ.LeftJoin(redisSQ, sqlchemy.Equals(providerQ.Field("id"), redisSQ.Field("manager_id")))
	providerQ = providerQ.LeftJoin(projectSQ, sqlchemy.Equals(providerQ.Field("id"), projectSQ.Field("manager_id")))
	providerQ = providerQ.LeftJoin(sregionSQ, sqlchemy.Equals(providerQ.Field("id"), sregionSQ.Field("cloudprovider_id")))

	providerQ = providerQ.Filter(sqlchemy.In(providerQ.Field("id"), providerIds)).GroupBy(providerQ.Field("id"))

	counts := []SCloudproviderUsageCount{}
	err := providerQ.All(&counts)
	if err != nil {
		return nil, errors.Wrapf(err, "providerQ.All")
	}
	for i := range counts {
		ret[counts[i].Id] = counts[i].SCloudproviderUsage
	}

	return ret, nil
}

func (cprvd *SCloudprovider) getProject(ctx context.Context) *db.STenant {
	proj, _ := db.TenantCacheManager.FetchTenantById(ctx, cprvd.ProjectId)
	return proj
}

func (manager *SCloudproviderManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudproviderDetails {
	rows := make([]api.CloudproviderDetails, len(objs))

	stdRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	projRows := manager.SProjectizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	pmRows := manager.SProjectMappingResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	accountIds := make([]string, len(objs))
	providerIds := make([]string, len(objs))
	for i := range rows {
		provider := objs[i].(*SCloudprovider)
		accountIds[i] = provider.CloudaccountId
		providerIds[i] = provider.Id
		rows[i] = api.CloudproviderDetails{
			EnabledStatusStandaloneResourceDetails: stdRows[i],
			ProjectizedResourceInfo:                projRows[i],
			ProjectMappingResourceInfo:             pmRows[i],
			LastSyncCost:                           provider.GetLastSyncCost(),
		}
	}

	q := CloudproviderRegionManager.Query()
	q = q.In("cloudprovider_id", providerIds)
	q = q.NotEquals("sync_status", api.CLOUD_PROVIDER_SYNC_STATUS_IDLE)
	cprs := []SCloudproviderregion{}
	err := q.All(&cprs)
	if err != nil {
		return rows
	}
	cprsMap := map[string]int{}
	for i := range cprs {
		_, ok := cprsMap[cprs[i].CloudproviderId]
		if !ok {
			cprsMap[cprs[i].CloudproviderId] = 0
		}
		cprsMap[cprs[i].CloudproviderId] += 1
	}

	accounts := make(map[string]SCloudaccount)
	err = db.FetchStandaloneObjectsByIds(CloudaccountManager, accountIds, &accounts)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds (%s) fail %s",
			CloudaccountManager.KeywordPlural(), err)
		return rows
	}

	proxySettingIds := make([]string, len(accounts))
	for i := range accounts {
		proxySettingId := accounts[i].ProxySettingId
		if !utils.IsInStringArray(proxySettingId, proxySettingIds) {
			proxySettingIds = append(proxySettingIds, proxySettingId)
		}
	}
	proxySettings := make(map[string]proxy.SProxySetting)
	err = db.FetchStandaloneObjectsByIds(proxy.ProxySettingManager, proxySettingIds, &proxySettings)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds (%s) fail %s",
			proxy.ProxySettingManager.KeywordPlural(), err)
		return rows
	}
	usages, err := manager.TotalResourceCount(providerIds)
	if err != nil {
		return rows
	}

	capabilities, err := CloudproviderCapabilityManager.getProvidersCapabilities(providerIds)
	if err != nil {
		return rows
	}

	for i := range rows {
		if account, ok := accounts[accountIds[i]]; ok {
			rows[i].Cloudaccount = account.Name
			rows[i].ReadOnly = account.ReadOnly
			rows[i].Brand = account.Brand

			ps := &rows[i].ProxySetting
			if proxySetting, ok := proxySettings[account.ProxySettingId]; ok {
				ps.Id = proxySetting.Id
				ps.Name = proxySetting.Name
				ps.HTTPProxy = proxySetting.HTTPProxy
				ps.HTTPSProxy = proxySetting.HTTPSProxy
				ps.NoProxy = proxySetting.NoProxy
			}
		}
		rows[i].SyncStatus2 = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
		if _, ok := cprsMap[providerIds[i]]; ok {
			rows[i].SyncStatus2 = api.CLOUD_PROVIDER_SYNC_STATUS_SYNCING
		}
		if usage, ok := usages[providerIds[i]]; ok {
			rows[i].SCloudproviderUsage = usage
		}
		if capas, ok := capabilities[providerIds[i]]; ok {
			rows[i].Capabilities = capas
		}
	}

	return rows
}

func (manager *SCloudproviderManager) initializeDefaultTenantId() error {
	// init accountid
	q := manager.Query().IsNullOrEmpty("tenant_id")
	providers := make([]SCloudprovider, 0)
	err := db.FetchModelObjects(manager, q, &providers)
	if err != nil {
		return errors.Wrap(err, "fetch empty defaullt tenant_id fail")
	}
	for i := range providers {
		provider := providers[i]
		domainId := provider.DomainId
		if len(domainId) == 0 {
			account, err := provider.GetCloudaccount()
			if err != nil {
				log.Errorf("GetCloudaccount fail %s", err)
				continue
			}
			domainId = account.DomainId
		}
		// auto fix accounts without default project
		defaultTenant, err := db.TenantCacheManager.FindFirstProjectOfDomain(context.Background(), domainId)
		if err != nil {
			return errors.Wrapf(err, "FindFirstProjectOfDomain(%s)", provider.DomainId)
		}
		_, err = db.Update(&provider, func() error {
			provider.ProjectId = defaultTenant.Id
			provider.DomainId = defaultTenant.DomainId
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "db.Update for account")
		}
	}
	return nil
}

func (manager *SCloudproviderManager) InitializeData() error {
	err := manager.initializeDefaultTenantId()
	if err != nil {
		log.Errorf("initializeDefaultTenantId %s", err)
	}

	return nil
}

// 云订阅列表
func (manager *SCloudproviderManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudproviderListInput,
) (*sqlchemy.SQuery, error) {
	accountArr := query.CloudaccountId
	if len(accountArr) > 0 {
		cpq := CloudaccountManager.Query().SubQuery()
		subcpq := cpq.Query(cpq.Field("id")).Filter(sqlchemy.OR(
			sqlchemy.In(cpq.Field("id"), stringutils2.RemoveUtf8Strings(accountArr)),
			sqlchemy.In(cpq.Field("name"), accountArr),
		)).SubQuery()
		q = q.In("cloudaccount_id", subcpq)
	}

	var zone *SZone
	var region *SCloudregion

	if len(query.ZoneId) > 0 {
		zoneObj, err := validators.ValidateModel(ctx, userCred, ZoneManager, &query.ZoneId)
		if err != nil {
			return nil, err
		}
		zone = zoneObj.(*SZone)
		region, err = zone.GetRegion()
		if err != nil {
			return nil, err
		}
		vpcs := VpcManager.Query("manager_id").Equals("cloudregion_id", region.Id).Distinct()
		if !utils.IsInStringArray(region.Provider, api.REGIONAL_NETWORK_PROVIDERS) {
			wires := WireManager.Query().Equals("zone_id", query.ZoneId).SubQuery()
			vpcs = vpcs.Join(wires, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
		}
		wireManager := WireManager.Query("manager_id").Equals("zone_id", query.ZoneId).Distinct().SubQuery()
		q = q.Filter(
			sqlchemy.OR(
				sqlchemy.In(q.Field("id"), vpcs.SubQuery()),
				sqlchemy.In(q.Field("id"), wireManager), //vmware
			),
		)
	} else if len(query.CloudregionId) > 0 {
		regionObj, err := CloudregionManager.FetchByIdOrName(ctx, userCred, query.CloudregionId)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudregion", query.CloudregionId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		region = regionObj.(*SCloudregion)
		pr := CloudproviderRegionManager.Query().SubQuery()
		sq := pr.Query(pr.Field("cloudprovider_id")).Equals("cloudregion_id", region.Id).Distinct()
		q = q.In("id", sq)
	}

	if query.Usable != nil && *query.Usable {
		providers := usableCloudProviders().SubQuery()
		networks := NetworkManager.Query().SubQuery()
		wires := WireManager.Query().SubQuery()
		vpcs := VpcManager.Query().SubQuery()
		providerRegions := CloudproviderRegionManager.Query().SubQuery()

		sq := providers.Query(sqlchemy.DISTINCT("id", providers.Field("id")))
		sq = sq.Join(providerRegions, sqlchemy.Equals(providers.Field("id"), providerRegions.Field("cloudprovider_id")))
		sq = sq.Join(vpcs, sqlchemy.Equals(providerRegions.Field("cloudregion_id"), vpcs.Field("cloudregion_id")))
		sq = sq.Join(wires, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
		sq = sq.Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
		sq = sq.Filter(sqlchemy.Equals(vpcs.Field("status"), api.VPC_STATUS_AVAILABLE))
		sq = sq.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))
		sq = sq.Filter(sqlchemy.OR(
			sqlchemy.IsNullOrEmpty(vpcs.Field("manager_id")),
			sqlchemy.Equals(vpcs.Field("manager_id"), providers.Field("id")),
		))
		if zone != nil {
			zoneFilter := sqlchemy.OR(sqlchemy.Equals(wires.Field("zone_id"), zone.GetId()), sqlchemy.IsNullOrEmpty(wires.Field("zone_id")))
			sq = sq.Filter(zoneFilter)
		} else if region != nil {
			sq = sq.Filter(sqlchemy.Equals(vpcs.Field("cloudregion_id"), region.GetId()))
		}

		q = q.Filter(sqlchemy.In(q.Field("id"), sq.SubQuery()))
	}

	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SProjectizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ProjectizedResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SProjectizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SSyncableBaseResourceManager.ListItemFilter(ctx, q, userCred, query.SyncableBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSyncableBaseResourceManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	managerStrs := query.CloudproviderId
	conditions := []sqlchemy.ICondition{}
	for _, managerStr := range managerStrs {
		if len(managerStr) == 0 {
			continue
		}
		providerObj, err := manager.FetchByIdOrName(ctx, userCred, managerStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudproviderManager.Keyword(), managerStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		conditions = append(conditions, sqlchemy.Equals(q.Field("id"), providerObj.GetId()))
	}
	if len(conditions) > 0 {
		q = q.Filter(sqlchemy.OR(conditions...))
	}

	cloudEnvStr := query.CloudEnv
	if cloudEnvStr == api.CLOUD_ENV_PUBLIC_CLOUD {
		cloudaccounts := CloudaccountManager.Query().SubQuery()
		q = q.Join(cloudaccounts, sqlchemy.Equals(cloudaccounts.Field("id"), q.Field("cloudaccount_id")))
		q = q.Filter(sqlchemy.IsTrue(cloudaccounts.Field("is_public_cloud")))
		q = q.Filter(sqlchemy.IsFalse(cloudaccounts.Field("is_on_premise")))
	}

	if cloudEnvStr == api.CLOUD_ENV_PRIVATE_CLOUD {
		cloudaccounts := CloudaccountManager.Query().SubQuery()
		q = q.Join(cloudaccounts, sqlchemy.Equals(cloudaccounts.Field("id"), q.Field("cloudaccount_id")))
		q = q.Filter(sqlchemy.IsFalse(cloudaccounts.Field("is_public_cloud")))
		q = q.Filter(sqlchemy.IsFalse(cloudaccounts.Field("is_on_premise")))
	}

	if cloudEnvStr == api.CLOUD_ENV_ON_PREMISE {
		cloudaccounts := CloudaccountManager.Query().SubQuery()
		q = q.Join(cloudaccounts, sqlchemy.Equals(cloudaccounts.Field("id"), q.Field("cloudaccount_id")))
		q = q.Filter(sqlchemy.IsFalse(cloudaccounts.Field("is_public_cloud")))
		q = q.Filter(sqlchemy.IsTrue(cloudaccounts.Field("is_on_premise")))
	}

	capabilities := query.Capability
	if len(capabilities) > 0 {
		subq := CloudproviderCapabilityManager.Query("cloudprovider_id").In("capability", capabilities).Distinct().SubQuery()
		q = q.In("id", subq)
	}

	if len(query.HealthStatus) > 0 {
		q = q.In("health_status", query.HealthStatus)
	}
	if len(query.Providers) > 0 {
		subq := CloudaccountManager.Query("id").In("provider", query.Providers).SubQuery()
		q = q.In("cloudaccount_id", subq)
	}
	if len(query.Brands) > 0 {
		subq := CloudaccountManager.Query("id").In("brand", query.Brands).SubQuery()
		q = q.In("cloudaccount_id", subq)
	}

	if len(query.HostSchedtagId) > 0 {
		schedTagObj, err := SchedtagManager.FetchByIdOrName(ctx, userCred, query.HostSchedtagId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", SchedtagManager.Keyword(), query.HostSchedtagId)
			} else {
				return nil, errors.Wrap(err, "SchedtagManager.FetchByIdOrName")
			}
		}
		subq := HostManager.Query("manager_id")
		hostschedtags := HostschedtagManager.Query().Equals("schedtag_id", schedTagObj.GetId()).SubQuery()
		subq = subq.Join(hostschedtags, sqlchemy.Equals(hostschedtags.Field("host_id"), subq.Field("id")))
		log.Debugf("%s", subq.String())
		q = q.In("id", subq.SubQuery())
	}

	if query.ReadOnly != nil {
		sq := CloudaccountManager.Query("id").Equals("read_only", *query.ReadOnly).SubQuery()
		q = q.In("cloudaccount_id", sq)
	}

	return q, nil
}

func (manager *SCloudproviderManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudproviderListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SCloudproviderManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	if field == "manager" {
		q = q.AppendField(q.Field("name").Label("manager")).Distinct()
		return q, nil
	}

	if field == "account" {
		accounts := CloudaccountManager.Query("name", "id").SubQuery()
		q.AppendField(accounts.Field("name", field)).Distinct()
		q = q.Join(accounts, sqlchemy.Equals(q.Field("cloudaccount_id"), accounts.Field("id")))
		return q, nil
	}

	q, err = manager.SProjectizedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (provider *SCloudprovider) markProviderDisconnected(ctx context.Context, userCred mcclient.TokenCredential, reason string) error {
	_, err := db.UpdateWithLock(ctx, provider, func() error {
		provider.HealthStatus = api.CLOUD_PROVIDER_HEALTH_UNKNOWN
		return nil
	})
	if err != nil {
		return err
	}
	if provider.Status != api.CLOUD_PROVIDER_DISCONNECTED {
		provider.SetStatus(ctx, userCred, api.CLOUD_PROVIDER_DISCONNECTED, reason)
		return provider.ClearSchedDescCache()
	}
	return nil
}

func (cprvd *SCloudprovider) updateName(ctx context.Context, userCred mcclient.TokenCredential, name, desc string) error {
	if cprvd.Name != name || cprvd.Description != desc {
		diff, err := db.Update(cprvd, func() error {
			cprvd.Name = name
			if len(cprvd.Description) == 0 {
				cprvd.Description = desc
			}
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "db.Update")
		}
		db.OpsLog.LogEvent(cprvd, db.ACT_UPDATE, diff, userCred)
	}
	return nil
}

func (provider *SCloudprovider) markProviderConnected(ctx context.Context, userCred mcclient.TokenCredential, healthStatus string) error {
	if healthStatus != provider.HealthStatus {
		diff, err := db.Update(provider, func() error {
			provider.HealthStatus = healthStatus
			return nil
		})
		if err != nil {
			return err
		}
		db.OpsLog.LogEvent(provider, db.ACT_UPDATE, diff, userCred)
	}
	if provider.Status != api.CLOUD_PROVIDER_CONNECTED {
		provider.SetStatus(ctx, userCred, api.CLOUD_PROVIDER_CONNECTED, "")
		return provider.ClearSchedDescCache()
	}
	return nil
}

func (provider *SCloudprovider) prepareCloudproviderRegions(ctx context.Context, userCred mcclient.TokenCredential) ([]SCloudproviderregion, error) {
	driver, err := provider.GetProvider(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "provider.GetProvider")
	}
	err = CloudproviderCapabilityManager.setCapabilities(ctx, userCred, provider.Id, driver.GetCapabilities())
	if err != nil {
		return nil, errors.Wrap(err, "CloudproviderCapabilityManager.setCapabilities")
	}
	if driver.GetFactory().IsOnPremise() {
		cpr, err := CloudproviderRegionManager.FetchByIdsOrCreate(provider.Id, api.DEFAULT_REGION_ID)
		if err != nil {
			return nil, errors.Wrapf(err, "FetchByIdsOrCreate")
		}
		cpr.setCapabilities(ctx, userCred, driver.GetCapabilities())
		return []SCloudproviderregion{*cpr}, nil
	}
	iregions, err := driver.GetIRegions()
	if err != nil {
		return nil, errors.Wrapf(err, "GetIRegions")
	}
	externalIdPrefix := driver.GetCloudRegionExternalIdPrefix()
	_, _, cprs, result := CloudregionManager.SyncRegions(ctx, userCred, provider, externalIdPrefix, iregions)
	if result.IsError() {
		log.Errorf("syncRegion fail %s", result.Result())
	}
	return cprs, nil
}

func (provider *SCloudprovider) GetCloudproviderRegions() []SCloudproviderregion {
	q := CloudproviderRegionManager.Query()
	q = q.Equals("cloudprovider_id", provider.Id)
	// q = q.IsTrue("enabled")
	// q = q.Equals("sync_status", api.CLOUD_PROVIDER_SYNC_STATUS_IDLE)

	return CloudproviderRegionManager.fetchRecordsByQuery(q)
}

func (provider *SCloudprovider) GetRegions() ([]SCloudregion, error) {
	q := CloudregionManager.Query()
	crcp := CloudproviderRegionManager.Query().SubQuery()
	q = q.Join(crcp, sqlchemy.Equals(q.Field("id"), crcp.Field("cloudregion_id"))).Filter(sqlchemy.Equals(crcp.Field("cloudprovider_id"), provider.Id))
	ret := []SCloudregion{}
	return ret, db.FetchModelObjects(CloudregionManager, q, &ret)
}

func (provider *SCloudprovider) GetUsableRegions() ([]SCloudregion, error) {
	q := CloudregionManager.Query()
	crcp := CloudproviderRegionManager.Query().SubQuery()
	q = q.Join(crcp, sqlchemy.Equals(q.Field("id"), crcp.Field("cloudregion_id"))).Filter(
		sqlchemy.AND(
			sqlchemy.Equals(crcp.Field("cloudprovider_id"), provider.Id),
			sqlchemy.IsTrue(crcp.Field("enabled")),
		),
	)
	ret := []SCloudregion{}
	err := db.FetchModelObjects(CloudregionManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (provider *SCloudprovider) resetAutoSync() {
	cprs := provider.GetCloudproviderRegions()
	for i := range cprs {
		cprs[i].resetAutoSync()
	}
}

func (provider *SCloudprovider) syncCloudproviderRegions(ctx context.Context, userCred mcclient.TokenCredential, syncRange SSyncRange, wg *sync.WaitGroup) {
	provider.markSyncing(userCred)
	cprs := provider.GetCloudproviderRegions()
	regionIds, _ := syncRange.GetRegionIds()
	syncCnt := 0
	for i := range cprs {
		if cprs[i].Enabled && cprs[i].CanSync() && (len(regionIds) == 0 || utils.IsInStringArray(cprs[i].CloudregionId, regionIds)) {
			syncCnt += 1
			if wg != nil {
				wg.Add(1)
			}
			cprs[i].submitSyncTask(ctx, userCred, syncRange)
			if wg != nil {
				wg.Done()
			}
		}
	}
	if syncCnt == 0 {
		err := provider.markEndSyncWithLock(ctx, userCred, false)
		if err != nil {
			log.Errorf("markEndSyncWithLock for %s error: %v", provider.Name, err)
		}
	}
}

func (provider *SCloudprovider) SyncCallSyncCloudproviderRegions(ctx context.Context, userCred mcclient.TokenCredential, syncRange SSyncRange) {
	var wg sync.WaitGroup
	provider.syncCloudproviderRegions(ctx, userCred, syncRange, &wg)
	wg.Wait()
}

func (cprvd *SCloudprovider) IsAvailable() bool {
	if !cprvd.GetEnabled() {
		return false
	}
	if !utils.IsInStringArray(cprvd.Status, api.CLOUD_PROVIDER_VALID_STATUS) {
		return false
	}
	if !utils.IsInStringArray(cprvd.HealthStatus, api.CLOUD_PROVIDER_VALID_HEALTH_STATUS) {
		return false
	}
	return true
}

func (cprvd *SCloudprovider) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	// override
	log.Infof("cloud provider delete do nothing")
	return nil
}

func (cprvd *SCloudprovider) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	regions, err := cprvd.GetRegions()
	if err != nil {
		return errors.Wrapf(err, "GetRegions")
	}

	for i := range regions {
		err = regions[i].purgeAll(ctx, cprvd.Id)
		if err != nil {
			return err
		}
	}
	return cprvd.purge(ctx, userCred)
}

func (cprvd *SCloudprovider) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return cprvd.StartCloudproviderDeleteTask(ctx, userCred, "")
}

func (cprvd *SCloudprovider) StartCloudproviderDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "CloudProviderDeleteTask", cprvd, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	cprvd.SetStatus(ctx, userCred, api.CLOUD_PROVIDER_START_DELETE, "StartCloudproviderDeleteTask")
	return task.ScheduleRun(nil)
}

func (cprvd *SCloudprovider) GetRegionDriver() (IRegionDriver, error) {
	driver := GetRegionDriver(cprvd.Provider)
	if driver == nil {
		return nil, fmt.Errorf("failed to found region driver for %s", cprvd.Provider)
	}
	return driver, nil
}

func (cprvd *SCloudprovider) ClearSchedDescCache() error {
	hosts := make([]SHost, 0)
	q := HostManager.Query().Equals("manager_id", cprvd.Id)
	err := db.FetchModelObjects(HostManager, q, &hosts)
	if err != nil {
		return err
	}
	for i := range hosts {
		err := hosts[i].ClearSchedDescCache()
		if err != nil {
			log.Errorf("host CleanHostSchedCache error: %v", err)
			return err
		}
	}
	return nil
}

func (cprvd *SCloudprovider) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	if strings.Index(cprvd.Status, "delet") >= 0 {
		return nil, httperrors.NewInvalidStatusError("Cannot enable deleting account")
	}
	_, err := cprvd.SEnabledStatusStandaloneResourceBase.PerformEnable(ctx, userCred, query, input)
	if err != nil {
		return nil, err
	}
	account, err := cprvd.GetCloudaccount()
	if err != nil {
		return nil, err
	}
	if !account.GetEnabled() {
		return account.enableAccountOnly(ctx, userCred, nil, input)
	}
	return nil, nil
}

func (cprvd *SCloudprovider) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	_, err := cprvd.SEnabledStatusStandaloneResourceBase.PerformDisable(ctx, userCred, query, input)
	if err != nil {
		return nil, err
	}
	account, err := cprvd.GetCloudaccount()
	if err != nil {
		return nil, err
	}
	allDisable := true
	providers := account.GetCloudproviders()
	for i := range providers {
		if providers[i].GetEnabled() {
			allDisable = false
			break
		}
	}
	if allDisable && account.GetEnabled() {
		return account.PerformDisable(ctx, userCred, nil, input)
	}
	return nil, nil
}

func (manager *SCloudproviderManager) filterByDomainId(q *sqlchemy.SQuery, domainId string) *sqlchemy.SQuery {
	subq := db.SharedResourceManager.Query("resource_id")
	subq = subq.Equals("resource_type", CloudaccountManager.Keyword())
	subq = subq.Equals("target_project_id", domainId)
	subq = subq.Equals("target_type", db.SharedTargetDomain)

	cloudaccounts := CloudaccountManager.Query().SubQuery()
	q = q.Join(cloudaccounts, sqlchemy.Equals(
		q.Field("cloudaccount_id"),
		cloudaccounts.Field("id"),
	))
	q = q.Filter(sqlchemy.OR(
		sqlchemy.AND(
			sqlchemy.Equals(q.Field("domain_id"), domainId),
			sqlchemy.Equals(cloudaccounts.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_PROVIDER_DOMAIN),
		),
		sqlchemy.AND(
			sqlchemy.Equals(cloudaccounts.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM),
			sqlchemy.OR(
				sqlchemy.AND(
					sqlchemy.Equals(cloudaccounts.Field("public_scope"), rbacscope.ScopeNone),
					sqlchemy.Equals(cloudaccounts.Field("domain_id"), domainId),
				),
				sqlchemy.AND(
					sqlchemy.Equals(cloudaccounts.Field("public_scope"), rbacscope.ScopeDomain),
					sqlchemy.OR(
						sqlchemy.Equals(cloudaccounts.Field("domain_id"), domainId),
						sqlchemy.In(cloudaccounts.Field("id"), subq.SubQuery()),
					),
				),
				sqlchemy.Equals(cloudaccounts.Field("public_scope"), rbacscope.ScopeSystem),
			),
		),
		sqlchemy.AND(
			sqlchemy.Equals(cloudaccounts.Field("domain_id"), domainId),
			sqlchemy.Equals(cloudaccounts.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN),
		),
	))
	return q
}

func (manager *SCloudproviderManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacscope.ScopeProject, rbacscope.ScopeDomain:
			if len(owner.GetProjectDomainId()) > 0 {
				q = manager.filterByDomainId(q, owner.GetProjectDomainId())
			}
		}
	}
	return q
}

func (cprvd *SCloudprovider) GetSyncStatus2() string {
	q := CloudproviderRegionManager.Query()
	q = q.Equals("cloudprovider_id", cprvd.Id)
	q = q.NotEquals("sync_status", api.CLOUD_PROVIDER_SYNC_STATUS_IDLE)

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

func (manager *SCloudproviderManager) fetchRecordsByQuery(q *sqlchemy.SQuery) []SCloudprovider {
	recs := make([]SCloudprovider, 0)
	err := db.FetchModelObjects(manager, q, &recs)
	if err != nil {
		return nil
	}
	return recs
}

func (manager *SCloudproviderManager) initAllRecords() {
	recs := manager.fetchRecordsByQuery(manager.Query())
	for i := range recs {
		db.Update(&recs[i], func() error {
			recs[i].SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
			return nil
		})
	}
}

func (provider *SCloudprovider) GetDetailsClirc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	accessUrl := provider.getAccessUrl()
	passwd, err := provider.getPassword()
	if err != nil {
		return nil, err
	}

	account, err := provider.GetCloudaccount()
	if err != nil {
		return nil, err
	}
	info := cloudprovider.SProviderInfo{
		Name:    provider.Name,
		Url:     accessUrl,
		Account: provider.Account,
		Secret:  passwd,
		Options: account.Options,
	}
	regions, err := provider.GetRegions()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegions")
	}
	if len(regions) > 0 {
		info.Region = fetchExternalId(regions[0].ExternalId)
	}
	rc, err := cloudprovider.GetClientRC(provider.Provider, info)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(rc), nil
}

func (manager *SCloudproviderManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (provider *SCloudprovider) GetDetailsStorageClasses(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input api.CloudproviderGetStorageClassInput,
) (api.CloudproviderGetStorageClassOutput, error) {
	output := api.CloudproviderGetStorageClassOutput{}
	driver, err := provider.GetProvider(ctx)
	if err != nil {
		return output, httperrors.NewInternalServerError("fail to get provider driver %s", err)
	}
	if len(input.CloudregionId) > 0 {
		_, input.CloudregionResourceInput, err = ValidateCloudregionResourceInput(ctx, userCred, input.CloudregionResourceInput)
		if err != nil {
			return output, errors.Wrap(err, "ValidateCloudregionResourceInput")
		}
	}

	sc := driver.GetStorageClasses(input.CloudregionId)
	if sc == nil {
		return output, httperrors.NewInternalServerError("storage classes not supported")
	}
	output.StorageClasses = sc
	return output, nil
}

func (provider *SCloudprovider) GetDetailsCannedAcls(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input api.CloudproviderGetCannedAclInput,
) (api.CloudproviderGetCannedAclOutput, error) {
	output := api.CloudproviderGetCannedAclOutput{}
	driver, err := provider.GetProvider(ctx)
	if err != nil {
		return output, httperrors.NewInternalServerError("fail to get provider driver %s", err)
	}
	if len(input.CloudregionId) > 0 {
		_, input.CloudregionResourceInput, err = ValidateCloudregionResourceInput(ctx, userCred, input.CloudregionResourceInput)
		if err != nil {
			return output, errors.Wrap(err, "ValidateCloudregionResourceInput")
		}
	}

	output.BucketCannedAcls = driver.GetBucketCannedAcls(input.CloudregionId)
	output.ObjectCannedAcls = driver.GetObjectCannedAcls(input.CloudregionId)
	return output, nil
}

func (provider *SCloudprovider) getAccountShareInfo() apis.SAccountShareInfo {
	account, _ := provider.GetCloudaccount()
	if account != nil {
		return account.getAccountShareInfo()
	}
	return apis.SAccountShareInfo{}
}

func (provider *SCloudprovider) IsSharable(reqUsrId mcclient.IIdentityProvider) bool {
	account, _ := provider.GetCloudaccount()
	if account != nil {
		if account.ShareMode == api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM {
			return account.IsSharable(reqUsrId)
		}
	}
	return false
}

func (provider *SCloudprovider) GetDetailsChangeOwnerCandidateDomains(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (apis.ChangeOwnerCandidateDomainsOutput, error) {
	return db.IOwnerResourceBaseModelGetChangeOwnerCandidateDomains(provider)
}

func (provider *SCloudprovider) GetChangeOwnerCandidateDomainIds() []string {
	account, _ := provider.GetCloudaccount()
	if account == nil {
		return []string{}
	}
	if account.ShareMode == api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN {
		return []string{account.DomainId}
	}
	// if account's public_scope=domain and share_mode=provider_domain, only allow to share to specific domains
	if account.PublicScope == string(rbacscope.ScopeDomain) {
		sharedDomains := account.GetSharedDomains()
		return append(sharedDomains, account.DomainId)
	}
	return []string{}
}

func (cprvd *SCloudprovider) GetExternalProjectsByProjectIdOrName(projectId, name string) ([]SExternalProject, error) {
	projects := []SExternalProject{}
	q := ExternalProjectManager.Query().Equals("manager_id", cprvd.Id)
	q = q.Filter(
		sqlchemy.OR(
			sqlchemy.Equals(q.Field("name"), name),
			sqlchemy.Equals(q.Field("tenant_id"), projectId),
		),
	).Desc("priority")
	err := db.FetchModelObjects(ExternalProjectManager, q, &projects)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return projects, nil
}

// 若本地项目映射了多个云上项目，则在根据优先级找优先级最大的云上项目
// 若本地项目没有映射云上任何项目，则在云上新建一个同名项目
// 若本地项目a映射云上项目b，但b项目不可用,则看云上是否有a项目，有则直接使用,若没有则在云上创建a-1, a-2类似项目
func (cprvd *SCloudprovider) SyncProject(ctx context.Context, userCred mcclient.TokenCredential, id string) (string, error) {
	lockman.LockRawObject(ctx, ExternalProjectManager.Keyword(), cprvd.Id)
	defer lockman.ReleaseRawObject(ctx, ExternalProjectManager.Keyword(), cprvd.Id)

	provider, err := cprvd.GetProvider(ctx)
	if err != nil {
		return "", errors.Wrap(err, "GetProvider")
	}

	project, err := db.TenantCacheManager.FetchTenantById(ctx, id)
	if err != nil {
		return "", errors.Wrapf(err, "FetchTenantById(%s)", id)
	}

	projects, err := cprvd.GetExternalProjectsByProjectIdOrName(id, project.Name)
	if err != nil {
		return "", errors.Wrapf(err, "GetExternalProjectsByProjectIdOrName(%s,%s)", id, project.Name)
	}

	extProj := GetAvailableExternalProject(project, projects)
	if extProj != nil {
		idx := strings.Index(extProj.ExternalId, "/")
		if idx > -1 {
			return extProj.ExternalId[idx+1:], nil
		}
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
			logclient.AddSimpleActionLog(cprvd, logclient.ACT_CREATE, err, userCred, false)
		}
		return "", errors.Wrapf(err, "CreateIProject(%s)", projectName)
	}

	extProj, err = cprvd.newFromCloudProject(ctx, userCred, project, iProject)
	if err != nil {
		return "", errors.Wrap(err, "newFromCloudProject")
	}

	db.Update(extProj, func() error {
		extProj.ManagerId = cprvd.Id
		return nil
	})

	idx := strings.Index(extProj.ExternalId, "/")
	if idx > -1 {
		return extProj.ExternalId[idx+1:], nil
	}

	return extProj.ExternalId, nil
}

func (cprvd *SCloudprovider) GetSchedtags() []SSchedtag {
	return GetSchedtags(CloudproviderschedtagManager, cprvd.Id)
}

func (cprvd *SCloudprovider) GetDynamicConditionInput() *jsonutils.JSONDict {
	return jsonutils.Marshal(cprvd).(*jsonutils.JSONDict)
}

func (cprvd *SCloudprovider) PerformSetSchedtag(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return PerformSetResourceSchedtag(cprvd, ctx, userCred, query, data)
}

func (cprvd *SCloudprovider) GetSchedtagJointManager() ISchedtagJointManager {
	return CloudproviderschedtagManager
}

func (cprvd *SCloudprovider) GetInterVpcNetworks() ([]SInterVpcNetwork, error) {
	networks := []SInterVpcNetwork{}
	q := InterVpcNetworkManager.Query().Equals("manager_id", cprvd.Id)
	err := db.FetchModelObjects(InterVpcNetworkManager, q, &networks)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return networks, nil

}

func (cprvd *SCloudprovider) SyncInterVpcNetwork(ctx context.Context, userCred mcclient.TokenCredential, interVpcNetworks []cloudprovider.ICloudInterVpcNetwork, xor bool) ([]SInterVpcNetwork, []cloudprovider.ICloudInterVpcNetwork, compare.SyncResult) {
	lockman.LockRawObject(ctx, cprvd.Keyword(), fmt.Sprintf("%s-interVpcNetwork", cprvd.Id))
	defer lockman.ReleaseRawObject(ctx, cprvd.Keyword(), fmt.Sprintf("%s-interVpcNetwork", cprvd.Id))

	result := compare.SyncResult{}

	localNetworks := []SInterVpcNetwork{}
	remoteNetworks := []cloudprovider.ICloudInterVpcNetwork{}

	dbNetworks, err := cprvd.GetInterVpcNetworks()
	if err != nil {
		result.Error(errors.Wrapf(err, "GetInterVpcNetworks"))
		return nil, nil, result
	}

	removed := make([]SInterVpcNetwork, 0)
	commondb := make([]SInterVpcNetwork, 0)
	commonext := make([]cloudprovider.ICloudInterVpcNetwork, 0)
	added := make([]cloudprovider.ICloudInterVpcNetwork, 0)

	err = compare.CompareSets(dbNetworks, interVpcNetworks, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return nil, nil, result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemove(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(commondb); i += 1 {
		if !xor {
			err = commondb[i].SyncWithCloudInterVpcNetwork(ctx, userCred, commonext[i])
			if err != nil {
				result.UpdateError(errors.Wrapf(err, "SyncWithCloudInterVpcNetwork"))
				continue
			}
		}
		localNetworks = append(localNetworks, commondb[i])
		remoteNetworks = append(remoteNetworks, commonext[i])

		result.Update()
	}

	for i := 0; i < len(added); i += 1 {
		network, err := InterVpcNetworkManager.newFromCloudInterVpcNetwork(ctx, userCred, added[i], cprvd)
		if err != nil {
			result.AddError(err)
			continue
		}

		localNetworks = append(localNetworks, *network)
		remoteNetworks = append(remoteNetworks, added[i])

		result.Add()
	}

	return localNetworks, remoteNetworks, result
}

func (manager *SCloudproviderManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SProjectizedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrapf(err, "SProjectizedResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SProjectMappingResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrapf(err, "SProjectMappingResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

// 绑定同步策略
func (cprvd *SCloudprovider) PerformProjectMapping(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudaccountProjectMappingInput) (jsonutils.JSONObject, error) {
	if len(input.ProjectMappingId) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, ProjectMappingManager, &input.ProjectMappingId)
		if err != nil {
			return nil, err
		}
		if len(cprvd.ProjectMappingId) > 0 && cprvd.ProjectMappingId != input.ProjectMappingId {
			return nil, httperrors.NewInputParameterError("cloudprovider %s has aleady bind project mapping %s", cprvd.Name, cprvd.ProjectMappingId)
		}
	}
	_, err := db.Update(cprvd, func() error {
		cprvd.ProjectMappingId = input.ProjectMappingId
		if input.EnableProjectSync != nil {
			cprvd.EnableProjectSync = tristate.NewFromBool(*input.EnableProjectSync)
		}
		if input.EnableResourceSync != nil {
			cprvd.EnableResourceSync = tristate.NewFromBool(*input.EnableResourceSync)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return nil, refreshPmCaches()
}

func (cprvd *SCloudprovider) PerformSetSyncing(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudproviderSync) (jsonutils.JSONObject, error) {
	regionIds := []string{}
	for i := range input.CloudregionIds {
		_, err := validators.ValidateModel(ctx, userCred, CloudregionManager, &input.CloudregionIds[i])
		if err != nil {
			return nil, err
		}
		regionIds = append(regionIds, input.CloudregionIds[i])
	}
	if len(regionIds) == 0 {
		return nil, nil
	}
	q := CloudproviderRegionManager.Query().Equals("cloudprovider_id", cprvd.Id).In("cloudregion_id", regionIds)
	cpcds := []SCloudproviderregion{}
	err := db.FetchModelObjects(CloudproviderRegionManager, q, &cpcds)
	if err != nil {
		return nil, err
	}
	for i := range cpcds {
		_, err := db.Update(&cpcds[i], func() error {
			cpcds[i].Enabled = input.Enabled
			return nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "db.Update")
		}
	}
	return nil, nil
}

func (cprvd *SCloudprovider) SyncError(result compare.SyncResult, iNotes interface{}, userCred mcclient.TokenCredential) {
	if result.IsGenerateError() {
		account := &SCloudaccount{}
		account.Id = cprvd.CloudaccountId
		account.Name = cprvd.Account
		if len(account.Name) == 0 {
			account.Name = cprvd.Name
		}
		account.SetModelManager(CloudaccountManager, account)
		logclient.AddSimpleActionLog(account, logclient.ACT_CLOUD_SYNC, iNotes, userCred, false)
	}
}

func (cprvd *SCloudaccount) SyncError(result compare.SyncResult, iNotes interface{}, userCred mcclient.TokenCredential) {
	if result.IsError() {
		logclient.AddSimpleActionLog(cprvd, logclient.ACT_CLOUD_SYNC, iNotes, userCred, false)
	}
}
