package models

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type SCloudaccountManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
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
}

type SCloudaccount struct {
	db.SEnabledStatusStandaloneResourceBase

	SSyncableBaseResource

	AccessUrl string `width:"64" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`

	Account string `width:"128" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(64, charset='ascii'), nullable=False)
	Secret  string `width:"256" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(256, charset='ascii'), nullable=False)

	BalanceKey string `width:"256" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`

	IsPublicCloud *bool `nullable:"false" get:"user" create:"optional" list:"user" default:"true"`
	IsOnPremise   bool  `nullable:"false" get:"user" create:"optional" list:"user" default:"false"`

	Provider string `width:"64" charset:"ascii" list:"admin" create:"admin_required"`

	EnableAutoSync      bool `default:"false" create:"admin_optional" list:"admin"`
	SyncIntervalSeconds int  `create:"admin_optional" list:"admin" update:"admin"`

	Balance float64   `list:"admin"`
	ProbeAt time.Time `list:"admin"`

	ErrorCount int `list:"admin"`

	AutoCreateProject bool `list:"admin" create:"admin_optional"`

	Version string               `width:"32" charset:"ascii" nullable:"true" list:"admin"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	Sysinfo jsonutils.JSONObject `get:"admin"`                                             // Column(JSONEncodedDict, nullable=True)
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
	if self.Enabled {
		return httperrors.NewInvalidStatusError("account is enabled")
	}
	cloudproviders := self.GetCloudproviders()
	for i := 0; i < len(cloudproviders); i++ {
		if err := cloudproviders[i].ValidateDeleteCondition(ctx); err != nil {
			return err
		}
	}

	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SCloudaccount) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	_, err := self.SEnabledStatusStandaloneResourceBase.PerformEnable(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	cloudproviders := self.GetCloudproviders()
	for i := 0; i < len(cloudproviders); i++ {
		_, err := cloudproviders[i].PerformEnable(ctx, userCred, query, data)
		if err != nil {
			return nil, err
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
		_, err := cloudproviders[i].PerformDisable(ctx, userCred, query, data)
		if err != nil {
			return nil, err
		}
	}
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
	return self.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (manager *SCloudaccountManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	// check provider
	// name, _ := data.GetString("name")
	provider, _ := data.GetString("provider")
	if !cloudprovider.IsSupported(provider) {
		return nil, httperrors.NewInputParameterError("Unsupported provider %s", provider)
	}
	providerDriver, _ := cloudprovider.GetProviderFactory(provider)
	if err := providerDriver.ValidateCreateCloudaccountData(ctx, userCred, data); err != nil {
		return nil, err
	}
	data.Set("is_public_cloud", jsonutils.NewBool(providerDriver.IsPublicCloud()))
	data.Set("is_on_premise", jsonutils.NewBool(providerDriver.IsOnPremise()))
	// check duplication
	// url, account, provider must be unique
	account, _ := data.GetString("account")
	secret, _ := data.GetString("secret")
	url, _ := data.GetString("access_url")

	q := manager.Query().Equals("provider", provider)
	if len(account) > 0 {
		q = q.Equals("account", account)
	}
	if len(url) > 0 {
		q = q.Equals("access_url", url)
	}

	if q.Count() > 0 {
		return nil, httperrors.NewConflictError("The account has been registered")
	}

	err := cloudprovider.IsValidCloudAccount(url, account, secret, provider)
	if err != nil {
		if err == cloudprovider.ErrNoSuchProvder {
			return nil, httperrors.NewResourceNotFoundError("no such provider %s", provider)
		}
		//log.Debugf("ValidateCreateData %s", err.Error())
		return nil, httperrors.NewInputParameterError("invalid cloud account info error: %s", err.Error())
	}

	syncIntervalSecs, _ := data.Int("sync_interval_seconds")
	if syncIntervalSecs == 0 {
		syncIntervalSecs = int64(options.Options.DefaultSyncIntervalSeconds)
	} else if syncIntervalSecs < int64(options.Options.MinimalSyncIntervalSeconds) {
		syncIntervalSecs = int64(options.Options.MinimalSyncIntervalSeconds)
	}
	data.Set("sync_interval_seconds", jsonutils.NewInt(syncIntervalSecs))

	return manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (self *SCloudaccount) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	cloudproviders := self.GetCloudproviders()
	for i := 0; i < len(cloudproviders); i++ {
		cloudproviders[i].Delete(ctx, userCred)
	}
}

func (self *SCloudaccount) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	self.Enabled = true
	self.EnableAutoSync = false
	return self.SEnabledStatusStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerProjId, query, data)
}

func (self *SCloudaccount) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	self.savePassword(self.Secret)

	self.StartSyncCloudProviderInfoTask(ctx, userCred, nil, "")
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

	providerDriver, _ := self.GetProviderFactory()
	account, err := providerDriver.ValidateUpdateCloudaccountCredential(ctx, userCred, data, self.Account)
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
		if q.Count() > 0 {
			return nil, httperrors.NewConflictError("account %s conflict", account.Account)
		}
	}

	originSecret, _ := self.getPassword()

	if err := cloudprovider.IsValidCloudAccount(self.AccessUrl, account.Account, account.Secret, self.Provider); err != nil {
		return nil, httperrors.NewInputParameterError("invalid cloud account info error: %s", err.Error())
	}

	if (account.Account != self.Account) || (account.Secret != originSecret) {
		if account.Account != self.Account {
			for _, cloudprovider := range self.GetCloudproviders() {
				if cloudprovider.Account == self.Account {
					_, err = db.Update(&cloudprovider, func() error {
						cloudprovider.Account = account.Account
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
		self.SetStatus(userCred, CLOUD_PROVIDER_INIT, "Change credential")
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
		self.SyncStatus = CLOUD_PROVIDER_SYNC_STATUS_QUEUED
		return nil
	})
	if err != nil {
		log.Errorf("Fail tp update last_sync %s", err)
		return err
	}
	return nil
}

func (self *SCloudaccount) MarkSyncing(userCred mcclient.TokenCredential) error {
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

func (self *SCloudaccount) MarkEndSync(userCred mcclient.TokenCredential) error {
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

func (self *SCloudaccount) GetProviderFactory() (cloudprovider.ICloudProviderFactory, error) {
	return cloudprovider.GetProviderFactory(self.Provider)
}

func (self *SCloudaccount) GetProvider() (cloudprovider.ICloudProvider, error) {
	if !self.Enabled {
		return nil, fmt.Errorf("Cloud provider is not enabled")
	}
	secret, err := self.getPassword()
	if err != nil {
		return nil, fmt.Errorf("Invalid password %s", err)
	}
	return cloudprovider.GetProvider(self.Id, self.Name, self.AccessUrl, self.Account, secret, self.Provider)
}

func (self *SCloudaccount) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	provider, err := self.GetProvider()
	if err != nil {
		return nil, err
	}
	return provider.GetSubAccounts()
}

func (self *SCloudaccount) importSubAccount(ctx context.Context, userCred mcclient.TokenCredential, subAccount cloudprovider.SSubAccount) (*SCloudprovider, bool, error) {
	isNew := false
	q := CloudproviderManager.Query().Equals("cloudaccount_id", self.Id).Equals("account", subAccount.Account)
	providerCount := q.Count()
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
		provider.markProviderConnected(ctx, userCred)
		return provider, isNew, nil
	}
	// not found, create a new cloudprovider
	isNew = true

	newCloudprovider, err := func() (*SCloudprovider, error) {
		lockman.LockClass(ctx, CloudproviderManager, "")
		defer lockman.ReleaseClass(ctx, CloudproviderManager, "")

		newCloudprovider := SCloudprovider{}
		newCloudprovider.Account = subAccount.Account
		newCloudprovider.Secret = self.Secret
		newCloudprovider.CloudaccountId = self.Id
		newCloudprovider.Provider = self.Provider
		newCloudprovider.AccessUrl = self.AccessUrl
		newCloudprovider.Enabled = true
		newCloudprovider.Status = CLOUD_PROVIDER_CONNECTED
		// newCloudprovider.HealthStatus = subAccount.HealthStatus
		newCloudprovider.Name = db.GenerateName(CloudproviderManager, "", subAccount.Name)
		if !self.AutoCreateProject {
			newCloudprovider.ProjectId = auth.AdminCredential().GetProjectId()
		}

		newCloudprovider.SetModelManager(CloudproviderManager)

		err := CloudproviderManager.TableSpec().Insert(&newCloudprovider)
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

	if self.AutoCreateProject {
		err = newCloudprovider.syncProject(ctx, userCred)
		if err != nil {
			log.Errorf("syncproject fail %s", err)
			return nil, isNew, err
		}
	}

	return newCloudprovider, isNew, nil
}

func (self *SCloudaccount) AllowPerformImport(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "import")
}

func (self *SCloudaccount) PerformImport(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// autoCreateProject := jsonutils.QueryBoolean(data, "auto_create_project", false)
	// autoSync := jsonutils.QueryBoolean(data, "auto_sync", false)
	// err := self.startImportSubAccountTask(ctx, userCred, autoCreateProject, autoSync, "")
	// noop
	return nil, nil
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

func (self *SCloudaccount) getProviderCount() int {
	q := CloudproviderManager.Query().Equals("cloudaccount_id", self.Id)
	return q.Count()
}

func (self *SCloudaccount) getHostCount() int {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	q := HostManager.Query().In("manager_id", subq)
	return q.Count()
}

func (self *SCloudaccount) getVpcCount() int {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	q := VpcManager.Query().In("manager_id", subq)
	return q.Count()
}

func (self *SCloudaccount) getStorageCount() int {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	q := StorageManager.Query().In("manager_id", subq)
	return q.Count()
}

func (self *SCloudaccount) getStoragecacheCount() int {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	q := StoragecacheManager.Query().In("manager_id", subq)
	return q.Count()
}

func (self *SCloudaccount) getEipCount() int {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	q := ElasticipManager.Query().In("manager_id", subq)
	return q.Count()
}

func (self *SCloudaccount) getRoutetableCount() int {
	subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	q := RouteTableManager.Query().In("manager_id", subq)
	return q.Count()
}

func (self *SCloudaccount) getGuestCount() int {
	subsubq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	subq := HostManager.Query("id").In("manager_id", subsubq).SubQuery()
	q := GuestManager.Query().In("host_id", subq)
	return q.Count()
}

func (self *SCloudaccount) getDiskCount() int {
	subsubq := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id).SubQuery()
	subq := StorageManager.Query("id").In("manager_id", subsubq).SubQuery()
	q := DiskManager.Query().In("storage_id", subq)
	return q.Count()
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
		ret = append(ret, projId)
	}
	return ret
}

func (self *SCloudaccount) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	extra.Add(jsonutils.NewInt(int64(self.getProviderCount())), "provider_count")
	extra.Add(jsonutils.NewInt(int64(self.getHostCount())), "host_count")
	extra.Add(jsonutils.NewInt(int64(self.getGuestCount())), "guest_count")
	extra.Add(jsonutils.NewInt(int64(self.getDiskCount())), "disk_count")
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

	if cloudprovider.Provider == CLOUD_PROVIDER_AZURE {
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
	account.SetModelManager(CloudaccountManager)
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
			account.SetStatus(auth.AdminCredential(), CLOUD_PROVIDER_DISCONNECTED, "invalid secret")
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
		queryDict := query.(*jsonutils.JSONDict)
		queryDict.Remove("account")
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

	if jsonutils.QueryBoolean(query, "public_cloud", false) {
		q = q.IsTrue("is_public_cloud")
	}

	if jsonutils.QueryBoolean(query, "private_cloud", false) {
		q = q.IsFalse("is_public_cloud")
	}

	if jsonutils.QueryBoolean(query, "is_on_premise", false) {
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

	if self.Status != CLOUD_PROVIDER_CONNECTED {
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
	db.OpsLog.LogEvent(self, db.ACT_UPDATE, sqlchemy.UpdateDiffString(diff), userCred)

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
	db.OpsLog.LogEvent(self, db.ACT_UPDATE, sqlchemy.UpdateDiffString(diff), userCred)

	return nil
}

func (account *SCloudaccount) markAccountDiscconected(ctx context.Context, userCred mcclient.TokenCredential) error {
	_, err := db.UpdateWithLock(ctx, account, func() error {
		account.ErrorCount = account.ErrorCount + 1
		return nil
	})
	if err != nil {
		return err
	}
	return account.SetStatus(userCred, CLOUD_PROVIDER_DISCONNECTED, "")
}

func (account *SCloudaccount) markAllProvidersDicconnected(ctx context.Context, userCred mcclient.TokenCredential) error {
	providers := account.GetCloudproviders()
	for i := 0; i < len(providers); i += 1 {
		err := providers[i].SetStatus(userCred, CLOUD_PROVIDER_DISCONNECTED, "cloud account disconnected")
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
	return account.SetStatus(userCred, CLOUD_PROVIDER_CONNECTED, "")
}

func (account *SCloudaccount) shouldProbeStatus() bool {
	// connected state
	if account.Status != CLOUD_PROVIDER_DISCONNECTED {
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

func (manager *SCloudaccountManager) AutoSyncCloudaccountTask(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	log.Debugf("AutoSyncCloudaccountTask")

	if isStart {
		// mark all the records to be init
		CloudproviderRegionManager.initAllRecords()
	}

	q := manager.Query()
	/*
		q = q.Filter(
			sqlchemy.OR(
				sqlchemy.NotEquals(q.Field("status"), CLOUD_PROVIDER_DISCONNECTED),
				sqlchemy.LT(q.Field("error_count"), options.Options.MaxCloudAccountErrorCount),
			),
		)
	*/
	// q = q.Equals("sync_status", CLOUD_PROVIDER_SYNC_STATUS_IDLE)

	accounts := make([]SCloudaccount, 0)
	err := db.FetchModelObjects(manager, q, &accounts)
	if err != nil {
		log.Errorf("fail to fetch cloudaccount list to check status")
		return
	}

	for i := range accounts {
		if accounts[i].shouldProbeStatus() && accounts[i].needSync() && accounts[i].CanSync() {
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
	manager, err := account.GetProvider()
	if err != nil {
		log.Errorf("account.GetProvider failed: %s", err)
		return nil, err
	}
	balance, err := manager.GetBalance()
	if err != nil {
		log.Errorf("manager.GetBalance fail %s", err)
		return nil, err
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
		account.IsPublicCloud = &isPublic
		account.IsOnPremise = factory.IsOnPremise()
		account.Balance = balance
		account.ProbeAt = timeutils.UtcNow()
		account.Version = version
		account.Sysinfo = sysInfo
		return nil
	})
	if err != nil {
		log.Errorf("fail to update db %s", err)
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
			oldProviders[i].markProviderDisconnected(ctx, userCred)
		}
	}
	return existProviders
}

func (account *SCloudaccount) syncAccountStatus(ctx context.Context, userCred mcclient.TokenCredential) error {
	account.MarkSyncing(userCred)
	subaccounts, err := account.probeAccountStatus(ctx, userCred)
	if err != nil {
		account.markAccountDiscconected(ctx, userCred)
		account.markAllProvidersDicconnected(ctx, userCred)
		return err
	}
	account.markAccountConnected(ctx, userCred)
	providers := account.importAllSubaccounts(ctx, userCred, subaccounts)
	for i := range providers {
		_, err := providers[i].prepareCloudproviderRegions(ctx, userCred)
		if err != nil {
			log.Errorf("syncCloudproviderRegion fail %s", err)
			return err
		}
	}
	return nil
}

func (account *SCloudaccount) SubmitSyncAccountTask(ctx context.Context, userCred mcclient.TokenCredential, waitChan chan error, autoSync bool) {
	RunSyncCloudAccountTask(func() {
		log.Debugf("syncAccountStatus %s %s", account.Id, account.Name)
		err := account.syncAccountStatus(ctx, userCred)
		if waitChan != nil {
			waitChan <- err
		} else {
			if err == nil && autoSync && account.EnableAutoSync {
				syncRange := SSyncRange{FullSync: true}
				providers := account.GetEnabledCloudproviders()
				for i := range providers {
					providers[i].syncCloudproviderRegions(userCred, &syncRange, nil)
				}
			}
			account.MarkEndSync(userCred)
		}
	})
}

func (account *SCloudaccount) SyncCallSyncAccountTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	waitChan := make(chan error)
	account.SubmitSyncAccountTask(ctx, userCred, waitChan, false)
	err := <-waitChan
	return err
}
