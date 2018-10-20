package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
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
)

const (
	CLOUD_PROVIDER_INIT         = "init"
	CLOUD_PROVIDER_CONNECTED    = "connected"
	CLOUD_PROVIDER_DISCONNECTED = "disconnected"
	CLOUD_PROVIDER_START_SYNC   = "start_sync"
	CLOUD_PROVIDER_SYNCING      = "syncing"

	CLOUD_PROVIDER_VMWARE = "VMware"
	CLOUD_PROVIDER_ALIYUN = "Aliyun"
	CLOUD_PROVIDER_AZURE  = "Azure"
)

type SCloudproviderManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	SInfrastructureManager
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
	SInfrastructure

	AccessUrl string `width:"64" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`
	// Hostname string `width:"64" charset:"ascii" nullable:"true"` // Column(VARCHAR(64, charset='ascii'), nullable=False)
	// port = Column(Integer, nullable=False)
	Account string `width:"128" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(64, charset='ascii'), nullable=False)
	Secret  string `width:"256" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(256, charset='ascii'), nullable=False)

	CloudaccountId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" key_index:"true"`

	ProjectId string `name:"tenant_id" width:"128" charset:"ascii" nullable:"true" list:"admin"`

	LastSync time.Time `get:"admin" list:"admin"` // = Column(DateTime, nullable=True)

	Version string `width:"32" charset:"ascii" nullable:"true" list:"admin"` // Column(VARCHAR(32, charset='ascii'), nullable=True)

	Sysinfo jsonutils.JSONObject `get:"admin"` // Column(JSONEncodedDict, nullable=True)

	Provider string `width:"64" charset:"ascii" list:"admin" create:"admin_required"`
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

func (self *SCloudprovider) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return self.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (self *SCloudproviderManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewUnsupportOperationError("Not support create cloudprovider, please considir create cloudaccount")
}

func (self *SCloudprovider) getPassword() (string, error) {
	return utils.DescryptAESBase64(self.Id, self.Secret)
}

func (self *SCloudprovider) CanSync() bool {
	if self.Status == CLOUD_PROVIDER_SYNCING {
		if self.LastSync.IsZero() || time.Now().Sub(self.LastSync) > 900*time.Second {
			return true
		} else {
			return false
		}
	} else {
		return true
	}
}

func (self *SCloudprovider) syncProject(ctx context.Context) error {
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
		s := auth.GetAdminSession(options.Options.Region, "")
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(self.Name), "name")
		params.Add(jsonutils.NewString(fmt.Sprintf("auto create from cloud provider %s", self.Name, self.Id)), "description")

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

	_, err = self.GetModelManager().TableSpec().Update(self, func() error {
		self.ProjectId = projectId
		return nil
	})

	if err != nil {
		log.Errorf("update projectId fail: %s", err)
		return err
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
		sr.Zone[i] = obj.GetId()
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
		sr.Host[i] = obj.GetId()
	}
	return nil
}

func (sr *SSyncRange) Normalize() error {
	if sr.Region != nil && len(sr.Region) > 0 {
		err := sr.normalizeRegionIds()
		if err != nil {
			return err
		}
	}
	if sr.Zone != nil && len(sr.Zone) > 0 {
		err := sr.normalizeZoneIds()
		if err != nil {
			return err
		}
	}
	if sr.Host != nil && len(sr.Host) > 0 {
		err := sr.normalizeHostIds()
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SCloudprovider) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SCloudprovider) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.Enabled {
		return nil, httperrors.NewInvalidStatusError("Cloudprovider disabled")
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
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudprovider) AllowPerformChangeProject(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
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

	_, err = self.GetModelManager().TableSpec().Update(self, func() error {
		self.ProjectId = tenant.Id
		return nil
	})

	if err != nil {
		log.Errorf("Update cloudprovider error: %v", err)
		return nil, httperrors.NewGeneralError(err)
	}

	return nil, self.StartSyncCloudProviderInfoTask(ctx, userCred, &SSyncRange{FullSync: true, ProjectSync: true}, "")
}

func (self *SCloudprovider) MarkStartSync(userCred mcclient.TokenCredential) {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.LastSync = timeutils.UtcNow()
		return nil
	})
	if err != nil {
		log.Errorf("Fail tp update last_sync %s", err)
		return
	}
	self.SetStatus(userCred, CLOUD_PROVIDER_START_SYNC, "")
}

func (self *SCloudprovider) GetDriver() (cloudprovider.ICloudProvider, error) {
	if !self.Enabled {
		return nil, fmt.Errorf("Cloud provider is not enabled")
	}

	account, err := self.getAccount()
	if err != nil {
		return nil, err
	}
	return cloudprovider.GetProvider(self.Id, self.Name, account.AccessUrl, account.Account, account.Secret, self.Provider)
}

type SAccount struct {
	AccessUrl string
	Account   string
	Secret    string
}

func (self *SCloudprovider) GetCloudaccount() *SCloudaccount {
	return CloudaccountManager.FetchCloudaccountById(self.CloudaccountId)
}

func (self *SCloudprovider) getAccount() (SAccount, error) {
	account := SAccount{}

	cloudaccount := self.GetCloudaccount()
	if cloudaccount == nil {
		return account, fmt.Errorf("fail to find cloudaccount???")
	}

	passwd, err := cloudaccount.getPassword()
	if err != nil {
		return account, err
	}

	account.Account = cloudaccount.Account
	account.AccessUrl = cloudaccount.AccessUrl
	account.Secret = passwd

	if len(self.Account) > 0 && self.Account != account.Account {
		account.Account = fmt.Sprintf("%s/%s", account.Account, self.Account)
	}

	return account, nil
}

func (self *SCloudprovider) SaveSysInfo(info jsonutils.JSONObject) {
	self.GetModelManager().TableSpec().Update(self, func() error {
		self.Sysinfo = info
		return nil
	})
}

func (manager *SCloudproviderManager) FetchCloudproviderById(providerId string) *SCloudprovider {
	providerObj, err := manager.FetchById(providerId)
	if err != nil {
		log.Errorf("%s", err)
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
	return extra
}

func (self *SCloudprovider) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(ctx, extra)
}

func (self *SCloudprovider) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	return self.getMoreDetails(ctx, extra)
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
				_, err = VCenterManager.TableSpec().Update(&vc, func() error {
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
		_, err := CloudproviderManager.TableSpec().Update(&providers[i], func() error {
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
	cp.Sysinfo = vc.Sysinfo
	cp.Provider = CLOUD_PROVIDER_VMWARE

	return manager.TableSpec().Insert(&cp)
}

func (self *SCloudprovider) GetBalance() (float64, error) {
	driver, err := self.GetDriver()
	if err != nil {
		return 0.0, err
	}
	return driver.GetBalance()
}

func (self *SCloudprovider) AllowGetDetailsBalance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SCloudprovider) GetDetailsBalance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	balance, err := self.GetBalance()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewFloat(balance), "balance")
	return ret, nil
}
