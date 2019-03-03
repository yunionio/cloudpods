package models

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

const (
	CLOUD_PROVIDER_INIT         = "init"
	CLOUD_PROVIDER_CONNECTED    = "connected"
	CLOUD_PROVIDER_DISCONNECTED = "disconnected"

	CLOUD_PROVIDER_SYNC_STATUS_QUEUED  = "queued"
	CLOUD_PROVIDER_SYNC_STATUS_SYNCING = "syncing"
	CLOUD_PROVIDER_SYNC_STATUS_IDLE    = "idle"

	CLOUD_PROVIDER_KVM       = "KVM"
	CLOUD_PROVIDER_VMWARE    = "VMware"
	CLOUD_PROVIDER_ALIYUN    = "Aliyun"
	CLOUD_PROVIDER_QCLOUD    = "Qcloud"
	CLOUD_PROVIDER_AZURE     = "Azure"
	CLOUD_PROVIDER_AWS       = "Aws"
	CLOUD_PROVIDER_HUAWEI    = "Huawei"
	CLOUD_PROVIDER_OPENSTACK = "OpenStack"

	CLOUD_PROVIDER_HEALTH_NORMAL    = "normal"    // 远端处于健康状态
	CLOUD_PROVIDER_HEALTH_SUSPENDED = "suspended" // 远端处于冻结状态
	CLOUD_PROVIDER_HEALTH_ARREARS   = "arrears"   // 远端处于欠费状态
)

var (
	CLOUD_PROVIDER_VALID_STATUS = []string{CLOUD_PROVIDER_CONNECTED}

	CLOUD_PROVIDERS = []string{
		CLOUD_PROVIDER_KVM,
		CLOUD_PROVIDER_VMWARE,
		CLOUD_PROVIDER_ALIYUN,
		CLOUD_PROVIDER_QCLOUD,
		CLOUD_PROVIDER_AZURE,
		CLOUD_PROVIDER_AWS,
		CLOUD_PROVIDER_HUAWEI,
		CLOUD_PROVIDER_OPENSTACK,
	}
)

type SCloudproviderManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
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
}

type SCloudprovider struct {
	db.SEnabledStatusStandaloneResourceBase

	SSyncableBaseResource

	// HealthStatus string `width:"64" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_optional"` // 云端服务健康状态。例如欠费、项目冻结都属于不健康状态。
	// Hostname string `width:"64" charset:"ascii" nullable:"true"` // Column(VARCHAR(64, charset='ascii'), nullable=False)
	// port = Column(Integer, nullable=False)
	// Version string `width:"32" charset:"ascii" nullable:"true" list:"admin"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	// Sysinfo jsonutils.JSONObject `get:"admin"` // Column(JSONEncodedDict, nullable=True)

	AccessUrl string `width:"64" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`
	Account   string `width:"128" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(64, charset='ascii'), nullable=False)
	Secret    string `width:"256" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(256, charset='ascii'), nullable=False)

	CloudaccountId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`

	ProjectId string `name:"tenant_id" width:"128" charset:"ascii" nullable:"true" list:"admin"`

	// LastSync time.Time `get:"admin" list:"admin"` // = Column(DateTime, nullable=True)

	Provider string `width:"64" charset:"ascii" list:"admin" create:"admin_required"`
}

func (manager *SCloudproviderManager) GetOwnerId(userCred mcclient.IIdentityProvider) string {
	return userCred.GetProjectId()
}

func (self *SCloudprovider) GetOwnerProjectId() string {
	return self.ProjectId
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
	if self.Enabled {
		return httperrors.NewInvalidStatusError("provider is enabled")
	}
	usage := self.getUsage()
	if !usage.isEmpty() {
		return httperrors.NewNotEmptyError("Not an empty cloud provider")
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (manager *SCloudproviderManager) GetPublicProviderIdsQuery() *sqlchemy.SSubQuery {
	return manager.GetProviderIdsQuery(tristate.True, tristate.None)
}

func (manager *SCloudproviderManager) GetPrivateProviderIdsQuery() *sqlchemy.SSubQuery {
	return manager.GetProviderIdsQuery(tristate.False, tristate.None)
}

func (manager *SCloudproviderManager) GetOnPremiseProviderIdsQuery() *sqlchemy.SSubQuery {
	return manager.GetProviderIdsQuery(tristate.False, tristate.True)
}

func (manager *SCloudproviderManager) GetProviderIdsQuery(isPublic tristate.TriState, isOnPremise tristate.TriState) *sqlchemy.SSubQuery {
	q := manager.Query("id")
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
		q = q.Filter(sqlchemy.IsTrue(account.Field("is_on_premise")))
	}
	return q.SubQuery()
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

func (self *SCloudprovider) GetGuestCount() int {
	sq := HostManager.Query("id").Equals("manager_id", self.Id)
	return GuestManager.Query().In("host_id", sq).Count()
}

func (self *SCloudprovider) GetHostCount() int {
	return HostManager.Query().Equals("manager_id", self.Id).Count()
}

func (self *SCloudprovider) getVpcCount() int {
	return VpcManager.Query().Equals("manager_id", self.Id).Count()
}

func (self *SCloudprovider) getStorageCount() int {
	return StorageManager.Query().Equals("manager_id", self.Id).Count()
}

func (self *SCloudprovider) getStoragecacheCount() int {
	return StoragecacheManager.Query().Equals("manager_id", self.Id).Count()
}

func (self *SCloudprovider) getEipCount() int {
	return ElasticipManager.Query().Equals("manager_id", self.Id).Count()
}

func (self *SCloudprovider) getSnapshotCount() int {
	return SnapshotManager.Query().Equals("manager_id", self.Id).Count()
}

func (self *SCloudprovider) getLoadbalancerCount() int {
	return LoadbalancerManager.Query().Equals("manager_id", self.Id).Count()
}

func (self *SCloudprovider) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return self.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (self *SCloudproviderManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
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
		} else if err == nil {
			return nil // find the project, skip sync
		}
	}

	if len(self.Name) == 0 {
		log.Errorf("syncProject: provider name is empty???")
		return fmt.Errorf("cannot syncProject for empty name")
	}

	tenant, err := db.TenantCacheManager.FetchTenantByIdOrName(ctx, self.Name)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("fetchTenantByIdorName error %s: %s", self.Name, err)
		return err
	}

	var projectId string
	if err == sql.ErrNoRows { // create one
		s := auth.GetAdminSession(ctx, options.Options.Region, "")
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(self.Name), "name")
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
		projectId = tenant.Id
	}

	return self.saveProject(userCred, projectId)
}

func (self *SCloudprovider) saveProject(userCred mcclient.TokenCredential, projectId string) error {
	if projectId != self.ProjectId {
		diff, err := db.Update(self, func() error {
			self.ProjectId = projectId
			return nil
		})
		if err != nil {
			log.Errorf("update projectId fail: %s", err)
			return err
		}
		db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
		logclient.AddSimpleActionLog(self, db.ACT_UPDATE, diff, userCred, true)
	}
	return nil
}

type SSyncRange struct {
	Force       bool
	FullSync    bool
	ProjectSync bool
	Region      []string
	Zone        []string
	Host        []string
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
	if account.EnableAutoSync {
		return nil, httperrors.NewInvalidStatusError("Account auto sync enabled")
	}
	syncRange := SSyncRange{}
	err := data.Unmarshal(&syncRange)
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid input %s", err)
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
		return nil, httperrors.NewInputParameterError("Missing project parameter")
	}

	tenant, err := db.TenantCacheManager.FetchTenantByIdOrName(ctx, project)
	if err != nil {
		return nil, httperrors.NewNotFoundError("project %s not found", project)
	}

	if self.ProjectId == tenant.Id {
		return nil, nil
	}

	err = self.saveProject(userCred, tenant.Id)
	if err != nil {
		log.Errorf("Update cloudprovider error: %v", err)
		return nil, httperrors.NewGeneralError(err)
	}

	return nil, self.StartSyncCloudProviderInfoTask(ctx, userCred, &SSyncRange{FullSync: true, ProjectSync: true}, "")
}

func (self *SCloudprovider) markStartSync(userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = CLOUD_PROVIDER_SYNC_STATUS_QUEUED
		return nil
	})
	if err != nil {
		log.Errorf("Fail tp update last_sync %s", err)
		return err
	}
	return nil
}

func (self *SCloudprovider) MarkSyncing(userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = CLOUD_PROVIDER_SYNC_STATUS_SYNCING
		self.LastSync = timeutils.UtcNow()
		self.LastSyncEndAt = time.Time{}
		return nil
	})
	if err != nil {
		log.Errorf("Fail tp update last_sync %s", err)
		return err
	}
	return nil
}

func (self *SCloudprovider) MarkEndSync(userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = CLOUD_PROVIDER_SYNC_STATUS_IDLE
		self.LastSyncEndAt = timeutils.UtcNow()
		return nil
	})
	if err != nil {
		log.Errorf("Fail tp update last_sync %s", err)
		return err
	}
	return nil
}

func (self *SCloudprovider) GetProviderFactory() (cloudprovider.ICloudProviderFactory, error) {
	return cloudprovider.GetProviderFactory(self.Provider)
}

func (self *SCloudprovider) GetProvider() (cloudprovider.ICloudProvider, error) {
	if !self.Enabled {
		return nil, fmt.Errorf("Cloud provider is not enabled")
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
	return true
}

func (self *SCloudprovider) getUsage() *SCloudproviderUsage {
	usage := SCloudproviderUsage{}

	usage.GuestCount = self.GetGuestCount()
	usage.HostCount = self.GetHostCount()
	usage.VpcCount = self.getVpcCount()
	usage.StorageCount = self.getStorageCount()
	usage.StorageCacheCount = self.getStoragecacheCount()
	usage.EipCount = self.getEipCount()
	usage.SnapshotCount = self.getSnapshotCount()
	usage.LoadbalancerCount = self.getLoadbalancerCount()

	return &usage
}

func (self *SCloudprovider) getProject(ctx context.Context) *db.STenant {
	proj, _ := db.TenantCacheManager.FetchTenantById(ctx, self.ProjectId)
	return proj
}

func (self *SCloudprovider) getMoreDetails(ctx context.Context, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	extra.Update(jsonutils.Marshal(self.getUsage()))
	project := self.getProject(ctx)
	if project != nil {
		extra.Add(jsonutils.NewString(project.Name), "tenant")
	}
	account := self.GetCloudaccount()
	if account != nil {
		extra.Add(jsonutils.NewString(account.GetName()), "cloudaccount")
	}
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
	cp.SetModelManager(manager)

	cp.Id = vc.Id
	cp.Name = db.GenerateName(manager, "", vc.Name)
	cp.Status = vc.Status
	cp.AccessUrl = fmt.Sprintf("https://%s:%d", vc.Hostname, vc.Port)
	cp.Account = vc.Account
	cp.Secret = vc.Password
	cp.LastSync = vc.LastSync
	cp.Provider = CLOUD_PROVIDER_VMWARE

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

	if jsonutils.QueryBoolean(query, "public_cloud", false) {
		cloudaccounts := CloudaccountManager.Query().SubQuery()
		q = q.Join(cloudaccounts, sqlchemy.Equals(cloudaccounts.Field("id"), q.Field("cloudaccount_id")))
		q = q.Filter(sqlchemy.IsTrue(cloudaccounts.Field("is_public_cloud")))
	}

	if jsonutils.QueryBoolean(query, "private_cloud", false) {
		cloudaccounts := CloudaccountManager.Query().SubQuery()
		q = q.Join(cloudaccounts, sqlchemy.Equals(cloudaccounts.Field("id"), q.Field("cloudaccount_id")))
		q = q.Filter(sqlchemy.IsFalse(cloudaccounts.Field("is_public_cloud")))
	}

	if jsonutils.QueryBoolean(query, "is_on_premise", false) {
		cloudaccounts := CloudaccountManager.Query().SubQuery()
		q = q.Join(cloudaccounts, sqlchemy.Equals(cloudaccounts.Field("id"), q.Field("cloudaccount_id")))
		q = q.Filter(sqlchemy.IsFalse(cloudaccounts.Field("is_public_cloud")))
		q = q.Filter(sqlchemy.IsTrue(cloudaccounts.Field("is_on_premise")))
	}

	return q, nil
}

func (provider *SCloudprovider) markProviderDisconnected(ctx context.Context, userCred mcclient.TokenCredential) error {
	_, err := db.UpdateWithLock(ctx, provider, func() error {
		provider.Enabled = false
		return nil
	})
	if err != nil {
		return err
	}
	return provider.SetStatus(userCred, CLOUD_PROVIDER_DISCONNECTED, "not a subaccount")
}

func (provider *SCloudprovider) markProviderConnected(ctx context.Context, userCred mcclient.TokenCredential) error {
	return provider.SetStatus(userCred, CLOUD_PROVIDER_CONNECTED, "")
}

func (provider *SCloudprovider) prepareCloudproviderRegions(ctx context.Context, userCred mcclient.TokenCredential) ([]SCloudproviderregion, error) {
	driver, err := provider.GetProvider()
	if err != nil {
		return nil, err
	}
	if driver.GetFactory().IsOnPremise() {
		cpr := CloudproviderRegionManager.FetchByIdsOrCreate(provider.Id, DEFAULT_REGION_ID)
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

func (provider *SCloudprovider) GetEnabledCloudproviderRegions() []SCloudproviderregion {
	q := CloudproviderRegionManager.Query()
	q = q.IsTrue("enabled")
	q = q.Equals("cloudprovider_id", provider.Id)
	q = q.Equals("sync_status", CLOUD_PROVIDER_SYNC_STATUS_IDLE)

	return CloudproviderRegionManager.fetchRecordsByQuery(q)
}

func (provider *SCloudprovider) syncCloudproviderRegions(userCred mcclient.TokenCredential, syncRange *SSyncRange, wg *sync.WaitGroup) {
	cprs := provider.GetEnabledCloudproviderRegions()
	for i := range cprs {
		if cprs[i].LastSyncEndAt.IsZero() || time.Now().Sub(cprs[i].LastSyncEndAt) > time.Duration(cprs[i].getSyncIntervalSeconds(nil))*time.Second {
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
}

func (provider *SCloudprovider) SyncCallSyncCloudproviderRegions(userCred mcclient.TokenCredential, syncRange *SSyncRange) {
	var wg sync.WaitGroup
	provider.syncCloudproviderRegions(userCred, syncRange, &wg)
	wg.Wait()
}

func (self *SCloudprovider) IsAvailable() bool {
	if !self.Enabled {
		return false
	}
	if !utils.IsInStringArray(self.Status, CLOUD_PROVIDER_VALID_STATUS) {
		return false
	}
	return true
}
