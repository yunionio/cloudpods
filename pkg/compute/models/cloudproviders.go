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
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
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

type SCloudproviderManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	db.SProjectizedResourceBaseManager
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

	SSyncableBaseResource

	HealthStatus string `width:"16" charset:"ascii" default:"normal" nullable:"false" list:"domain"` // 云端服务健康状态。例如欠费、项目冻结都属于不健康状态。
	// Hostname string `width:"64" charset:"ascii" nullable:"true"` // Column(VARCHAR(64, charset='ascii'), nullable=False)
	// port = Column(Integer, nullable=False)
	// Version string `width:"32" charset:"ascii" nullable:"true" list:"domain"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	// Sysinfo jsonutils.JSONObject `get:"domain"` // Column(JSONEncodedDict, nullable=True)

	AccessUrl string `width:"64" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	Account   string `width:"128" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"` // Column(VARCHAR(64, charset='ascii'), nullable=False)
	Secret    string `length:"0" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`  // Google需要秘钥认证,需要此字段比较长

	CloudaccountId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`

	// ProjectId string `name:"tenant_id" width:"128" charset:"ascii" nullable:"true" list:"domain"`

	// LastSync time.Time `get:"domain" list:"domain"` // = Column(DateTime, nullable=True)

	Provider string `width:"64" charset:"ascii" list:"domain" create:"domain_required"`
}

func (self *SCloudproviderManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SCloudproviderManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SCloudprovider) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SCloudprovider) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SCloudprovider) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SCloudprovider) ValidateDeleteCondition(ctx context.Context) error {
	// allow delete cloudprovider if it is disabled
	// account := self.GetCloudaccount()
	// if account != nil && account.EnableAutoSync {
	// 	return httperrors.NewInvalidStatusError("auto syncing is enabled on account")
	// }
	if self.Enabled {
		return httperrors.NewInvalidStatusError("provider is enabled")
	}
	if self.SyncStatus != api.CLOUD_PROVIDER_SYNC_STATUS_IDLE {
		return httperrors.NewInvalidStatusError("provider is not idle")
	}
	// usage := self.getUsage()
	// if !usage.isEmpty() {
	// 	return httperrors.NewNotEmptyError("Not an empty cloud provider")
	// }
	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
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
	if len(providers) > 0 {
		q = q.Filter(sqlchemy.In(account.Field("provider"), providers))
	}
	if len(brands) > 0 {
		q = q.Filter(sqlchemy.In(account.Field("brand"), brands))
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

func (self *SCloudprovider) CleanSchedCache() {
	hosts := []SHost{}
	q := HostManager.Query().Equals("manager_id", self.Id)
	if err := db.FetchModelObjects(HostManager, q, &hosts); err != nil {
		log.Errorf("failed to get hosts for cloudprovider %s error: %v", self.Name, err)
		return
	}
	for _, host := range hosts {
		host.ClearSchedDescCache()
	}
}

func (self *SCloudprovider) GetGuestCount() (int, error) {
	sq := HostManager.Query("id").Equals("manager_id", self.Id)
	return GuestManager.Query().In("host_id", sq).CountWithError()
}

func (self *SCloudprovider) GetHostCount() (int, error) {
	return HostManager.Query().Equals("manager_id", self.Id).CountWithError()
}

func (self *SCloudprovider) getVpcCount() (int, error) {
	return VpcManager.Query().Equals("manager_id", self.Id).CountWithError()
}

func (self *SCloudprovider) getStorageCount() (int, error) {
	return StorageManager.Query().Equals("manager_id", self.Id).CountWithError()
}

func (self *SCloudprovider) getStoragecacheCount() (int, error) {
	return StoragecacheManager.Query().Equals("manager_id", self.Id).CountWithError()
}

func (self *SCloudprovider) getEipCount() (int, error) {
	return ElasticipManager.Query().Equals("manager_id", self.Id).CountWithError()
}

func (self *SCloudprovider) getSnapshotCount() (int, error) {
	return SnapshotManager.Query().Equals("manager_id", self.Id).CountWithError()
}

func (self *SCloudprovider) getLoadbalancerCount() (int, error) {
	return LoadbalancerManager.Query().Equals("manager_id", self.Id).CountWithError()
}

func (self *SCloudprovider) getExternalProjectCount() (int, error) {
	return ExternalProjectManager.Query().Equals("manager_id", self.Id).CountWithError()
}

func (self *SCloudprovider) getSyncRegionCount() (int, error) {
	return CloudproviderRegionManager.Query().Equals("cloudprovider_id", self.Id).CountWithError()
}

func (self *SCloudprovider) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return self.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (self *SCloudproviderManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewUnsupportOperationError("Directly creating cloudprovider is not supported, create cloudaccount instead")
}

func (self *SCloudprovider) getAccessUrl() string {
	if len(self.AccessUrl) > 0 {
		return self.AccessUrl
	}
	account := self.GetCloudaccount()
	return account.AccessUrl
}

func (self *SCloudprovider) getPassword() (string, error) {
	if len(self.Secret) == 0 {
		account := self.GetCloudaccount()
		return account.getPassword()
	}
	return utils.DescryptAESBase64(self.Id, self.Secret)
}

func (self *SCloudprovider) syncProject(ctx context.Context, userCred mcclient.TokenCredential) error {
	if len(self.ProjectId) > 0 {
		_, err := db.TenantCacheManager.FetchTenantById(ctx, self.ProjectId)
		if err != nil && err != sql.ErrNoRows {
			log.Errorf("fetch existing tenant by id fail %s", err)
			return errors.Wrap(err, "db.TenantCacheManager.FetchTenantById")
		} else if err == nil {
			return nil // find the project, skip sync
		}
	}

	if len(self.Name) == 0 {
		log.Errorf("syncProject: provider name is empty???")
		return errors.Error("cannot syncProject for empty name")
	}

	tenant, err := db.TenantCacheManager.FetchTenantByIdOrName(ctx, self.Name)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("fetchTenantByIdorName error %s: %s", self.Name, err)
		return errors.Wrap(err, "db.TenantCacheManager.FetchTenantByIdOrName")
	}

	account := self.GetCloudaccount()
	if account == nil {
		return errors.Error("no valid cloudaccount???")
	}

	var projectId, domainId string
	if err == sql.ErrNoRows || tenant.DomainId != account.DomainId { // create one
		s := auth.GetAdminSession(ctx, options.Options.Region, "")
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(self.Name), "generate_name")

		domainId = account.DomainId
		params.Add(jsonutils.NewString(domainId), "domain_id")
		params.Add(jsonutils.NewString(fmt.Sprintf("auto create from cloud provider %s (%s)", self.Name, self.Id)), "description")

		project, err := modules.Projects.Create(s, params)

		if err != nil {
			log.Errorf("create project fail %s", err)
			return err
		}
		projectId, err = project.GetString("id")
		if err != nil {
			return err
		}
	} else {
		domainId = tenant.DomainId
		projectId = tenant.Id
	}

	return self.saveProject(userCred, domainId, projectId)
}

func (self *SCloudprovider) saveProject(userCred mcclient.TokenCredential, domainId, projectId string) error {
	if projectId != self.ProjectId {
		diff, err := db.Update(self, func() error {
			self.DomainId = domainId
			self.ProjectId = projectId
			return nil
		})
		if err != nil {
			log.Errorf("update projectId fail: %s", err)
			return err
		}
		db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
	}
	return nil
}

type SSyncRange struct {
	Force    bool
	FullSync bool
	DeepSync bool
	// ProjectSync bool

	Region []string
	Zone   []string
	Host   []string
}

func (sr *SSyncRange) NeedSyncInfo() bool {
	if sr.FullSync {
		return true
	}
	if sr.Region != nil && len(sr.Region) > 0 {
		return true
	}
	if sr.Zone != nil && len(sr.Zone) > 0 {
		return true
	}
	if sr.Host != nil && len(sr.Host) > 0 {
		return true
	}
	return false
}

func (sr *SSyncRange) normalizeRegionIds() error {
	for i := 0; i < len(sr.Region); i += 1 {
		obj, err := CloudregionManager.FetchByIdOrName(nil, sr.Region[i])
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

func (sr *SSyncRange) normalizeZoneIds() error {
	for i := 0; i < len(sr.Zone); i += 1 {
		obj, err := ZoneManager.FetchByIdOrName(nil, sr.Zone[i])
		if err != nil {
			if err == sql.ErrNoRows {
				return httperrors.NewResourceNotFoundError("Zone %s not found", sr.Zone[i])
			} else {
				return err
			}
		}
		zone := obj.(*SZone)
		region := zone.GetRegion()
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

func (sr *SSyncRange) normalizeHostIds() error {
	for i := 0; i < len(sr.Host); i += 1 {
		obj, err := HostManager.FetchByIdOrName(nil, sr.Host[i])
		if err != nil {
			if err == sql.ErrNoRows {
				return httperrors.NewResourceNotFoundError("Host %s not found", sr.Host[i])
			} else {
				return err
			}
		}
		host := obj.(*SHost)
		zone := host.GetZone()
		if zone == nil {
			continue
		}
		region := zone.GetRegion()
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

func (sr *SSyncRange) Normalize() error {
	if sr.Region != nil && len(sr.Region) > 0 {
		err := sr.normalizeRegionIds()
		if err != nil {
			return err
		}
	} else {
		sr.Region = make([]string, 0)
	}
	if sr.Zone != nil && len(sr.Zone) > 0 {
		err := sr.normalizeZoneIds()
		if err != nil {
			return err
		}
	} else {
		sr.Zone = make([]string, 0)
	}
	if sr.Host != nil && len(sr.Host) > 0 {
		err := sr.normalizeHostIds()
		if err != nil {
			return err
		}
	} else {
		sr.Host = make([]string, 0)
	}
	return nil
}

func (self *SCloudprovider) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "sync")
}

func (self *SCloudprovider) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.Enabled {
		return nil, httperrors.NewInvalidStatusError("Cloudprovider disabled")
	}
	account := self.GetCloudaccount()
	if !account.Enabled {
		return nil, httperrors.NewInvalidStatusError("Cloudaccount disabled")
	}
	if account.EnableAutoSync {
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

func (self *SCloudprovider) StartSyncCloudProviderInfoTask(ctx context.Context, userCred mcclient.TokenCredential, syncRange *SSyncRange, parentTaskId string) error {
	params := jsonutils.NewDict()
	if syncRange != nil {
		params.Add(jsonutils.Marshal(syncRange), "sync_range")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "CloudProviderSyncInfoTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("startSyncCloudProviderInfoTask newTask error %s", err)
		return err
	}
	if cloudaccount := self.GetCloudaccount(); cloudaccount != nil {
		cloudaccount.markAutoSync(userCred)
	}
	self.markStartSync(userCred)
	db.OpsLog.LogEvent(self, db.ACT_SYNC_HOST_START, "", userCred)
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudprovider) AllowPerformChangeProject(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "change-project")
}

func (self *SCloudprovider) PerformChangeProject(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	project, err := data.GetString("project")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("project")
	}

	tenant, err := db.TenantCacheManager.FetchTenantByIdOrName(ctx, project)
	if err != nil {
		return nil, httperrors.NewNotFoundError("project %s not found", project)
	}

	if self.ProjectId == tenant.Id {
		return nil, nil
	}

	account := self.GetCloudaccount()
	if self.DomainId != tenant.DomainId {
		if !db.IsAdminAllowPerform(userCred, self, "change-project") {
			return nil, httperrors.NewForbiddenError("not allow to change project across domain")
		}
		if account.ShareMode == api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN {
			return nil, httperrors.NewInvalidStatusError("not a public cloud account")
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
		OldProjectId: self.ProjectId,
		OldDomainId:  self.DomainId,
		NewProjectId: tenant.Id,
		NewProject:   tenant.Name,
		NewDomainId:  tenant.DomainId,
		NewDomain:    tenant.Domain,
	}

	err = self.saveProject(userCred, tenant.DomainId, tenant.Id)
	if err != nil {
		log.Errorf("Update cloudprovider error: %v", err)
		return nil, httperrors.NewGeneralError(err)
	}

	logclient.AddSimpleActionLog(self, logclient.ACT_CHANGE_OWNER, notes, userCred, true)

	if account.EnableAutoSync { // no need to sync rightnow, will do it in auto sync
		return nil, nil
	}

	return nil, self.StartSyncCloudProviderInfoTask(ctx, userCred, &SSyncRange{FullSync: true, DeepSync: true}, "")
}

func (self *SCloudprovider) markStartingSync(userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_QUEUING
		return nil
	})
	if err != nil {
		log.Errorf("Failed to markStartSync error: %v", err)
		return errors.Wrap(err, "Update")
	}
	cprs := self.GetCloudproviderRegions()
	for i := range cprs {
		if cprs[i].Enabled {
			err := cprs[i].markStartingSync(userCred)
			if err != nil {
				return errors.Wrap(err, "cprs[i].markStartingSync")
			}
		}
	}
	return nil
}

func (self *SCloudprovider) markStartSync(userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_QUEUED
		return nil
	})
	if err != nil {
		log.Errorf("Failed to markStartSync error: %v", err)
		return err
	}
	cprs := self.GetCloudproviderRegions()
	for i := range cprs {
		if cprs[i].Enabled {
			err := cprs[i].markStartingSync(userCred)
			if err != nil {
				return errors.Wrap(err, "cprs[i].markStartingSync")
			}
		}
	}
	return nil
}

func (self *SCloudprovider) markSyncing(userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_SYNCING
		self.LastSync = timeutils.UtcNow()
		self.LastSyncEndAt = time.Time{}
		return nil
	})
	if err != nil {
		log.Errorf("Failed to markSyncing error: %v", err)
		return err
	}
	return nil
}

func (self *SCloudprovider) markEndSyncWithLock(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := func() error {
		lockman.LockObject(ctx, self)
		defer lockman.ReleaseObject(ctx, self)

		if self.SyncStatus == api.CLOUD_PROVIDER_SYNC_STATUS_IDLE {
			return nil
		}

		if self.getSyncStatus2() != api.CLOUD_PROVIDER_SYNC_STATUS_IDLE {
			return nil
		}

		err := self.markEndSync(userCred)
		if err != nil {
			return err
		}
		return nil
	}()

	if err != nil {
		return err
	}

	account := self.GetCloudaccount()
	return account.MarkEndSyncWithLock(ctx, userCred)
}

func (self *SCloudprovider) markEndSync(userCred mcclient.TokenCredential) error {
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

func (self *SCloudprovider) cancelStartingSync(userCred mcclient.TokenCredential) error {
	if self.SyncStatus == api.CLOUD_PROVIDER_SYNC_STATUS_QUEUING {
		cprs := self.GetCloudproviderRegions()
		for i := range cprs {
			err := cprs[i].cancelStartingSync(userCred)
			if err != nil {
				return errors.Wrap(err, "cprs[i].cancelStartingSync")
			}
		}
		_, err := db.Update(self, func() error {
			self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "db.Update")
		}
	}
	return nil
}

func (self *SCloudprovider) GetProviderFactory() (cloudprovider.ICloudProviderFactory, error) {
	return cloudprovider.GetProviderFactory(self.Provider)
}

func (self *SCloudprovider) GetProvider() (cloudprovider.ICloudProvider, error) {
	if !self.Enabled {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "Cloud provider is not enabled")
	}

	accessUrl := self.getAccessUrl()
	passwd, err := self.getPassword()
	if err != nil {
		return nil, err
	}
	return cloudprovider.GetProvider(self.Id, self.Name, accessUrl, self.Account, passwd, self.Provider)
}

func (self *SCloudprovider) savePassword(secret string) error {
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

func (self *SCloudprovider) GetCloudaccount() *SCloudaccount {
	return CloudaccountManager.FetchCloudaccountById(self.CloudaccountId)
}

func (manager *SCloudproviderManager) FetchCloudproviderById(providerId string) *SCloudprovider {
	providerObj, err := manager.FetchById(providerId)
	if err != nil {
		log.Errorf("fetch cloud provider %s: %s", providerId, err)
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
	if !providerObj.Enabled {
		return false
	}
	account := providerObj.GetCloudaccount()
	if account == nil {
		return false
	}
	return account.Enabled
}

func (manager *SCloudproviderManager) FetchCloudproviderByIdOrName(providerId string) *SCloudprovider {
	providerObj, err := manager.FetchByIdOrName(nil, providerId)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("%s", err)
		}
		return nil
	}
	return providerObj.(*SCloudprovider)
}

type SCloudproviderUsage struct {
	GuestCount        int
	HostCount         int
	VpcCount          int
	StorageCount      int
	StorageCacheCount int
	EipCount          int
	SnapshotCount     int
	LoadbalancerCount int
	ProjectCount      int
	SyncRegionCount   int
}

func (usage *SCloudproviderUsage) isEmpty() bool {
	if usage.HostCount > 0 {
		return false
	}
	if usage.VpcCount > 0 {
		return false
	}
	if usage.StorageCount > 0 {
		return false
	}
	if usage.StorageCacheCount > 0 {
		return false
	}
	if usage.EipCount > 0 {
		return false
	}
	if usage.SnapshotCount > 0 {
		return false
	}
	if usage.LoadbalancerCount > 0 {
		return false
	}
	/*if usage.ProjectCount > 0 {
		return false
	}
	if usage.SyncRegionCount > 0 {
		return false
	}*/
	return true
}

func (self *SCloudprovider) getUsage() *SCloudproviderUsage {
	usage := SCloudproviderUsage{}

	usage.GuestCount, _ = self.GetGuestCount()
	usage.HostCount, _ = self.GetHostCount()
	usage.VpcCount, _ = self.getVpcCount()
	usage.StorageCount, _ = self.getStorageCount()
	usage.StorageCacheCount, _ = self.getStoragecacheCount()
	usage.EipCount, _ = self.getEipCount()
	usage.SnapshotCount, _ = self.getSnapshotCount()
	usage.LoadbalancerCount, _ = self.getLoadbalancerCount()
	usage.ProjectCount, _ = self.getExternalProjectCount()
	usage.SyncRegionCount, _ = self.getSyncRegionCount()

	return &usage
}

func (self *SCloudprovider) getProject(ctx context.Context) *db.STenant {
	proj, _ := db.TenantCacheManager.FetchTenantById(ctx, self.ProjectId)
	return proj
}

func (self *SCloudprovider) getMoreDetails(ctx context.Context, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	extra.Update(jsonutils.Marshal(self.getUsage()))
	// project := self.getProject(ctx)
	// if project != nil {
	// 	extra.Add(jsonutils.NewString(project.Name), "tenant")
	// }
	account := self.GetCloudaccount()
	if account != nil {
		extra.Add(jsonutils.NewString(account.GetName()), "cloudaccount")
	}
	extra.Set("sync_status2", jsonutils.NewString(self.getSyncStatus2()))
	return extra
}

func (self *SCloudprovider) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(ctx, extra)
}

func (self *SCloudprovider) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SEnabledStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(ctx, extra), nil
}

func (manager *SCloudproviderManager) InitializeData() error {
	// move vmware info from vcenter to cloudprovider
	vcenters := make([]SVCenter, 0)
	q := VCenterManager.Query()
	err := db.FetchModelObjects(VCenterManager, q, &vcenters)
	if err != nil {
		return err
	}
	for _, vc := range vcenters {
		_, err := CloudproviderManager.FetchById(vc.Id)
		if err != nil {
			if err == sql.ErrNoRows {
				err = manager.migrateVCenterInfo(&vc)
				if err != nil {
					log.Errorf("migrateVcenterInfo fail %s", err)
					return err
				}
				_, err = db.Update(&vc, func() error {
					return vc.MarkDelete()
				})
				if err != nil {
					log.Errorf("delete vcenter record fail %s", err)
					return err
				}
			} else {
				log.Errorf("fetch cloudprovider fail %s", err)
				return err
			}
		} else {
			log.Debugf("vcenter info has been migrate into cloudprovider")
		}
	}

	// fill empty projectId with system project ID
	providers := make([]SCloudprovider, 0)
	q = CloudproviderManager.Query()
	q = q.Filter(sqlchemy.OR(sqlchemy.IsEmpty(q.Field("tenant_id")), sqlchemy.IsNull(q.Field("tenant_id"))))
	err = db.FetchModelObjects(CloudproviderManager, q, &providers)
	if err != nil {
		log.Errorf("query cloudproviders with empty tenant_id fail %s", err)
		return err
	}
	for i := 0; i < len(providers); i += 1 {
		_, err := db.Update(&providers[i], func() error {
			providers[i].DomainId = auth.AdminCredential().GetProjectDomainId()
			providers[i].ProjectId = auth.AdminCredential().GetProjectId()
			return nil
		})
		if err != nil {
			log.Errorf("update cloudprovider project fail %s", err)
			return err
		}
	}

	return nil
}

func (manager *SCloudproviderManager) migrateVCenterInfo(vc *SVCenter) error {
	cp := SCloudprovider{}
	cp.SetModelManager(manager, &cp)

	newName, err := db.GenerateName(manager, nil, vc.Name)
	if err != nil {
		return err
	}
	cp.Id = vc.Id
	cp.Name = newName
	cp.Status = vc.Status
	cp.AccessUrl = fmt.Sprintf("https://%s:%d", vc.Hostname, vc.Port)
	cp.Account = vc.Account
	cp.Secret = vc.Password
	cp.LastSync = vc.LastSync
	cp.Provider = api.CLOUD_PROVIDER_VMWARE

	return manager.TableSpec().Insert(&cp)
}

func (manager *SCloudproviderManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	accountStr, _ := query.GetString("account")
	if len(accountStr) > 0 {
		queryDict := query.(*jsonutils.JSONDict)
		queryDict.Remove("account")
		accountObj, err := CloudaccountManager.FetchByIdOrName(userCred, accountStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(manager.Keyword(), accountStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		q = q.Equals("cloudaccount_id", accountObj.GetId())
	}

	if jsonutils.QueryBoolean(query, "usable", false) {
		providers := CloudproviderManager.Query().SubQuery()
		networks := NetworkManager.Query().SubQuery()
		wires := WireManager.Query().SubQuery()
		vpcs := VpcManager.Query().SubQuery()

		sq := providers.Query(sqlchemy.DISTINCT("id", providers.Field("id")))
		sq = sq.Join(vpcs, sqlchemy.Equals(vpcs.Field("manager_id"), providers.Field("id")))
		sq = sq.Join(wires, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
		sq = sq.Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
		sq = sq.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))
		sq = sq.Filter(sqlchemy.IsTrue(providers.Field("enabled")))
		sq = sq.Filter(sqlchemy.In(providers.Field("status"), api.CLOUD_PROVIDER_VALID_STATUS))
		sq = sq.Filter(sqlchemy.In(providers.Field("health_status"), api.CLOUD_PROVIDER_VALID_HEALTH_STATUS))
		sq = sq.Filter(sqlchemy.Equals(vpcs.Field("status"), api.VPC_STATUS_AVAILABLE))

		sq2 := providers.Query(sqlchemy.DISTINCT("id", providers.Field("id")))
		sq2 = sq2.Join(vpcs, sqlchemy.Equals(vpcs.Field("manager_id"), providers.Field("id")))
		sq2 = sq2.Join(wires, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
		sq2 = sq2.Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
		sq2 = sq2.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))
		sq2 = sq2.Filter(sqlchemy.IsNullOrEmpty(vpcs.Field("manager_id")))
		sq2 = sq2.Filter(sqlchemy.Equals(vpcs.Field("status"), api.VPC_STATUS_AVAILABLE))

		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(q.Field("id"), sq.SubQuery()),
			sqlchemy.In(q.Field("id"), sq2.SubQuery()),
		))
	}

	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	managerStr, _ := query.GetString("manager")
	if len(managerStr) > 0 {
		providerObj, err := manager.FetchByIdOrName(userCred, managerStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudproviderManager.Keyword(), managerStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		q = q.Equals("id", providerObj.GetId())
	}

	cloudEnvStr, _ := query.GetString("cloud_env")
	if cloudEnvStr == api.CLOUD_ENV_PUBLIC_CLOUD || jsonutils.QueryBoolean(query, "public_cloud", false) {
		cloudaccounts := CloudaccountManager.Query().SubQuery()
		q = q.Join(cloudaccounts, sqlchemy.Equals(cloudaccounts.Field("id"), q.Field("cloudaccount_id")))
		q = q.Filter(sqlchemy.IsTrue(cloudaccounts.Field("is_public_cloud")))
		q = q.Filter(sqlchemy.IsFalse(cloudaccounts.Field("is_on_premise")))
	}

	if cloudEnvStr == api.CLOUD_ENV_PRIVATE_CLOUD || jsonutils.QueryBoolean(query, "private_cloud", false) {
		cloudaccounts := CloudaccountManager.Query().SubQuery()
		q = q.Join(cloudaccounts, sqlchemy.Equals(cloudaccounts.Field("id"), q.Field("cloudaccount_id")))
		q = q.Filter(sqlchemy.IsFalse(cloudaccounts.Field("is_public_cloud")))
		q = q.Filter(sqlchemy.IsFalse(cloudaccounts.Field("is_on_premise")))
	}

	if cloudEnvStr == api.CLOUD_ENV_ON_PREMISE || jsonutils.QueryBoolean(query, "is_on_premise", false) {
		cloudaccounts := CloudaccountManager.Query().SubQuery()
		q = q.Join(cloudaccounts, sqlchemy.Equals(cloudaccounts.Field("id"), q.Field("cloudaccount_id")))
		q = q.Filter(sqlchemy.IsFalse(cloudaccounts.Field("is_public_cloud")))
		q = q.Filter(sqlchemy.IsTrue(cloudaccounts.Field("is_on_premise")))
	}

	if query.Contains("has_object_storage") {
		hasObjectStorage, _ := query.Bool("has_object_storage")
		cloudaccounts := CloudaccountManager.Query().SubQuery()
		q = q.Join(cloudaccounts, sqlchemy.Equals(cloudaccounts.Field("id"), q.Field("cloudaccount_id")))
		if hasObjectStorage {
			q = q.Filter(sqlchemy.IsTrue(cloudaccounts.Field("has_object_storage")))
		} else {
			q = q.Filter(sqlchemy.IsFalse(cloudaccounts.Field("has_object_storage")))
		}
	}

	return q, nil
}

func (provider *SCloudprovider) markProviderDisconnected(ctx context.Context, userCred mcclient.TokenCredential, reason string) error {
	_, err := db.UpdateWithLock(ctx, provider, func() error {
		provider.HealthStatus = api.CLOUD_PROVIDER_HEALTH_UNKNOWN
		return nil
	})
	if err != nil {
		return err
	}
	provider.SetStatus(userCred, api.CLOUD_PROVIDER_DISCONNECTED, reason)
	return provider.ClearSchedDescCache()
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
	provider.SetStatus(userCred, api.CLOUD_PROVIDER_CONNECTED, "")
	return provider.ClearSchedDescCache()
}

func (provider *SCloudprovider) prepareCloudproviderRegions(ctx context.Context, userCred mcclient.TokenCredential) ([]SCloudproviderregion, error) {
	driver, err := provider.GetProvider()
	if err != nil {
		return nil, err
	}
	if driver.GetFactory().IsOnPremise() {
		cpr := CloudproviderRegionManager.FetchByIdsOrCreate(provider.Id, api.DEFAULT_REGION_ID)
		return []SCloudproviderregion{*cpr}, nil
	}
	iregions := driver.GetIRegions()
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

func (provider *SCloudprovider) syncCloudproviderRegions(ctx context.Context, userCred mcclient.TokenCredential, syncRange SSyncRange, wg *sync.WaitGroup, autoSync bool) {
	provider.markSyncing(userCred)
	cprs := provider.GetCloudproviderRegions()
	syncCnt := 0
	for i := range cprs {
		if cprs[i].Enabled && cprs[i].CanSync() && (!autoSync || cprs[i].needAutoSync()) {
			syncCnt += 1
			var waitChan chan bool = nil
			if wg != nil {
				wg.Add(1)
				waitChan = make(chan bool)
			}
			cprs[i].submitSyncTask(userCred, syncRange, waitChan)
			if wg != nil {
				<-waitChan
				wg.Done()
			}
		}
	}
	if syncCnt == 0 {
		provider.markEndSyncWithLock(ctx, userCred)
	}
}

func (provider *SCloudprovider) SyncCallSyncCloudproviderRegions(ctx context.Context, userCred mcclient.TokenCredential, syncRange SSyncRange) {
	var wg sync.WaitGroup
	provider.syncCloudproviderRegions(ctx, userCred, syncRange, &wg, false)
	wg.Wait()
}

func (self *SCloudprovider) IsAvailable() bool {
	if !self.Enabled {
		return false
	}
	if !utils.IsInStringArray(self.Status, api.CLOUD_PROVIDER_VALID_STATUS) {
		return false
	}
	if !utils.IsInStringArray(self.HealthStatus, api.CLOUD_PROVIDER_VALID_HEALTH_STATUS) {
		return false
	}
	return true
}

func (self *SCloudprovider) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	// override
	log.Infof("cloud provider delete do nothing")
	return nil
}

func (self *SCloudprovider) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	var err error

	for _, manager := range []IPurgeableManager{
		BucketManager,
		HostManager,
		SnapshotManager,
		SnapshotPolicyManager,
		StorageManager,
		StoragecacheManager,
		SecurityGroupCacheManager,
		LoadbalancerManager,
		LoadbalancerBackendGroupManager,
		CachedLoadbalancerAclManager,
		CachedLoadbalancerCertificateManager,
		NatGatewayManager,
		DBInstanceManager,
		DBInstanceBackupManager,
		ElasticcacheManager,
		VpcManager,
		ElasticipManager,
		NetworkInterfaceManager,
		CloudproviderRegionManager,
		ExternalProjectManager,
		CloudregionManager,
	} {
		err = manager.purgeAll(ctx, userCred, self.Id)
		if err != nil {
			log.Errorf("%s purgeall failed %s", manager.Keyword(), err)
			return err
		}
		log.Debugf("%s purgeall success!", manager.Keyword())
	}

	return self.SEnabledStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SCloudprovider) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartCloudproviderDeleteTask(ctx, userCred, "")
}

func (self *SCloudprovider) StartCloudproviderDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "CloudProviderDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("%s", err)
		return err
	}
	self.SetStatus(userCred, api.CLOUD_PROVIDER_START_DELETE, "StartCloudproviderDeleteTask")
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudprovider) ClearSchedDescCache() error {
	hosts := make([]SHost, 0)
	q := HostManager.Query().Equals("manager_id", self.Id)
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

func (self *SCloudprovider) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	_, err := self.SEnabledStatusStandaloneResourceBase.PerformEnable(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	account := self.GetCloudaccount()
	if account != nil {
		allEnabled := true
		providers := account.GetCloudproviders()
		for i := range providers {
			if !providers[i].Enabled {
				allEnabled = false
			}
		}
		if allEnabled && !account.Enabled {
			return account.PerformEnable(ctx, userCred, nil, nil)
		}
	}
	return nil, nil
}

func (self *SCloudprovider) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	_, err := self.SEnabledStatusStandaloneResourceBase.PerformDisable(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	account := self.GetCloudaccount()
	if account != nil {
		allDisable := true
		providers := account.GetCloudproviders()
		for i := range providers {
			if providers[i].Enabled {
				allDisable = false
			}
		}
		if allDisable && account.Enabled {
			return account.PerformDisable(ctx, userCred, nil, nil)
		}
	}
	return nil, nil
}

func (manager *SCloudproviderManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []db.IModel, fields stringutils2.SSortedStrings) []*jsonutils.JSONDict {
	rows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields)
	projectIds := stringutils2.SSortedStrings{}
	for i := range objs {
		idStr := objs[i].GetOwnerId().GetProjectId()
		projectIds = stringutils2.Append(projectIds, idStr)
	}
	if len(fields) == 0 || fields.Contains("tenant") || fields.Contains("domain") {
		projects := db.FetchProjects(projectIds, false)
		if projects != nil {
			for i := range rows {
				idStr := objs[i].GetOwnerId().GetProjectId()
				if proj, ok := projects[idStr]; ok {
					if len(fields) == 0 || fields.Contains("domain") {
						rows[i].Add(jsonutils.NewString(proj.Domain), "domain")
					}
					if len(fields) == 0 || fields.Contains("tenant") {
						rows[i].Add(jsonutils.NewString(proj.Name), "tenant")
					}
				}

			}
		}
	}
	return rows
}

func (manager *SCloudproviderManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacutils.ScopeProject, rbacutils.ScopeDomain:
			if len(owner.GetProjectDomainId()) > 0 {
				cloudaccounts := CloudaccountManager.Query().SubQuery()
				q = q.Join(cloudaccounts, sqlchemy.Equals(
					q.Field("cloudaccount_id"),
					cloudaccounts.Field("id"),
				))
				q = q.Filter(sqlchemy.OR(
					sqlchemy.AND(
						sqlchemy.Equals(q.Field("domain_id"), owner.GetProjectDomainId()),
						sqlchemy.Equals(cloudaccounts.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_PROVIDER_DOMAIN),
					),
					sqlchemy.Equals(cloudaccounts.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM),
					sqlchemy.AND(
						sqlchemy.Equals(cloudaccounts.Field("domain_id"), owner.GetProjectDomainId()),
						sqlchemy.Equals(cloudaccounts.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN),
					),
				))
			}
		}
	}
	return q
}

func (self *SCloudprovider) getSyncStatus2() string {
	q := CloudproviderRegionManager.Query()
	q = q.Equals("cloudprovider_id", self.Id)
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

func (provider *SCloudprovider) AllowGetDetailsClirc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, provider, "client-rc")
}

func (provider *SCloudprovider) GetDetailsClirc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	accessUrl := provider.getAccessUrl()
	passwd, err := provider.getPassword()
	if err != nil {
		return nil, err
	}
	rc, err := cloudprovider.GetClientRC(accessUrl, provider.Account, passwd, provider.Provider)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(rc), nil
}

func (manager *SCloudproviderManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (provider *SCloudprovider) AllowGetDetailsStorageClasses(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) bool {
	return db.IsAdminAllowGetSpec(userCred, provider, "storage-classes")
}

func (provider *SCloudprovider) GetDetailsStorageClasses(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	driver, err := provider.GetProvider()
	if err != nil {
		return nil, httperrors.NewInternalServerError("fail to get provider driver %s", err)
	}
	extId := ""
	regionStr := jsonutils.GetAnyString(query, []string{"cloudregion", "cloudregion_id"})
	if len(regionStr) > 0 {
		regionObj, err := CloudregionManager.FetchByIdOrName(userCred, regionStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudregionManager.Keyword(), regionStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		extId = regionObj.(*SCloudregion).GetExternalId()
	}

	sc := driver.GetStorageClasses(extId)
	if sc == nil {
		return nil, httperrors.NewInternalServerError("storage classes not supported")
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewStringArray(sc), "storage_classes")
	return ret, nil
}
