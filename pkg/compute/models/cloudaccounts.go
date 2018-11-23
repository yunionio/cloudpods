package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type SCloudaccountManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	SInfrastructureManager
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
	SInfrastructure

	AccessUrl string `width:"64" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`

	Account string `width:"128" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(64, charset='ascii'), nullable=False)
	Secret  string `width:"256" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(256, charset='ascii'), nullable=False)

	BalanceKey string    `width:"256" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`
	LastSync   time.Time `get:"admin" list:"admin"` // = Column(DateTime, nullable=True)

	Version string `width:"32" charset:"ascii" nullable:"true" list:"admin"` // Column(VARCHAR(32, charset='ascii'), nullable=True)

	Sysinfo jsonutils.JSONObject `get:"admin"` // Column(JSONEncodedDict, nullable=True)

	Provider string `width:"64" charset:"ascii" list:"admin" create:"admin_required"`
}

func (self *SCloudaccount) GetCloudproviders() []SCloudprovider {
	cloudproviders := []SCloudprovider{}
	q := CloudproviderManager.Query().Equals("cloudaccount_id", self.Id)
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
	return self.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (manager *SCloudaccountManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	// check provider
	// name, _ := data.GetString("name")
	provider, _ := data.GetString("provider")
	if !cloudprovider.IsSupported(provider) {
		return nil, httperrors.NewInputParameterError("Unsupported provider %s", provider)
	}
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
		return nil, httperrors.NewInputParameterError("invalid cloud account info")
	}

	return manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (self *SCloudaccount) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	cloudproviders := self.GetCloudproviders()
	for i := 0; i < len(cloudproviders); i++ {
		cloudproviders[i].Delete(ctx, userCred)
	}
}

func (self *SCloudaccount) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	self.savePassword(self.Secret)

	autoCreateProject := jsonutils.QueryBoolean(data, "auto_create_project", false)
	autoSync := jsonutils.QueryBoolean(data, "auto_sync", false)
	self.startImportSubAccountTask(ctx, userCred, autoCreateProject, autoSync, "")
}

func (self *SCloudaccount) savePassword(secret string) error {
	sec, err := utils.EncryptAESBase64(self.Id, secret)
	if err != nil {
		return err
	}

	_, err = self.GetModelManager().TableSpec().Update(self, func() error {
		self.Secret = sec
		return nil
	})
	return err
}

func (self *SCloudaccount) getPassword() (string, error) {
	return utils.DescryptAESBase64(self.Id, self.Secret)
}

func (self *SCloudaccount) CanSync() bool {
	if self.Status == CLOUD_PROVIDER_SYNCING || self.Status == CLOUD_PROVIDER_START_SYNC {
		if self.LastSync.IsZero() || time.Now().Sub(self.LastSync) > 900*time.Second {
			return true
		} else {
			return false
		}
	} else {
		return true
	}
}

func (self *SCloudaccount) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SCloudaccount) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.Enabled {
		return nil, httperrors.NewInvalidStatusError("Account disabled")
	}
	syncRange := SSyncRange{}
	err := data.Unmarshal(&syncRange)
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid input %s", err)
	}
	if self.CanSync() || syncRange.Force {
		err = self.StartSyncCloudProviderInfoTask(ctx, userCred, nil, &syncRange, "")
	}
	return nil, err
}

func (self *SCloudaccount) AllowPerformUpdateCredential(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SCloudaccount) PerformUpdateCredential(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.Enabled {
		return nil, httperrors.NewInvalidStatusError("Account disabled")
	}

	var err error
	changed := false
	secret, _ := data.GetString("secret")
	account, _ := data.GetString("account")
	if len(account) > 0 && self.Provider == CLOUD_PROVIDER_AZURE {
		return nil, httperrors.NewInputParameterError("not allow update azure tenant info")
	}
	accessUrl, _ := data.GetString("access_url")
	if len(secret) > 0 || len(account) > 0 || len(accessUrl) > 0 {
		// check duplication
		q := self.GetModelManager().Query()
		q = q.Equals("access_url", accessUrl)
		q = q.Equals("account", account)
		q = q.NotEquals("id", self.Id)
		if q.Count() > 0 {
			return nil, httperrors.NewConflictError("Access url and account conflict")
		}
	}

	if (len(account) > 0 && account != self.Account) || (len(accessUrl) > 0 && accessUrl != self.AccessUrl) {
		if len(account) > 0 && account != self.Account {
			for _, cloudprovider := range self.GetCloudproviders() {
				if cloudprovider.Account == self.Account {
					_, err = cloudprovider.GetModelManager().TableSpec().Update(&cloudprovider, func() error {
						cloudprovider.Account = account
						return nil
					})
					if err != nil {
						return nil, err
					}
				}
			}
		}
		_, err = self.GetModelManager().TableSpec().Update(self, func() error {
			if len(account) > 0 {
				self.Account = account
			}
			if len(accessUrl) > 0 {
				self.AccessUrl = accessUrl
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		changed = true
	}

	if len(secret) > 0 {
		err = self.savePassword(secret)
		if err != nil {
			return nil, err
		}

		for _, provider := range self.GetCloudproviders() {
			provider.savePassword(secret)
		}

		changed = true
	}

	if changed {
		self.SetStatus(userCred, CLOUD_PROVIDER_INIT, "Change credential")
		self.StartSyncCloudProviderInfoTask(ctx, userCred, nil, nil, "")
	}
	return nil, nil
}

func (self *SCloudaccount) StartSyncCloudProviderInfoTask(ctx context.Context, userCred mcclient.TokenCredential, cloudProviders []SCloudprovider, syncRange *SSyncRange, parentTaskId string) error {
	params := jsonutils.NewDict()
	if syncRange != nil {
		params.Add(jsonutils.Marshal(syncRange), "sync_range")
	}

	if cloudProviders == nil {
		cloudProviders = self.GetCloudproviders()
	}

	taskItems := make([]db.IStandaloneModel, 0)
	for i := 0; i < len(cloudProviders); i++ {
		if cloudProviders[i].Enabled {
			taskItems = append(taskItems, &cloudProviders[i])
		}
	}

	task, err := taskman.TaskManager.NewParallelTask(ctx, "CloudAccountSyncInfoTask", taskItems, userCred, params, "", "", nil)
	if err != nil {
		log.Errorf("CloudAccountSyncInfoTask newTask error %s", err)
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SCloudaccount) MarkStartSync(userCred mcclient.TokenCredential) {
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

func (self *SCloudaccount) GetDriver() (cloudprovider.ICloudProvider, error) {
	if !self.Enabled {
		return nil, fmt.Errorf("Cloud provider is not enabled")
	}

	secret, err := self.getPassword()
	if err != nil {
		return nil, fmt.Errorf("Invalid password %s", err)
	}
	// log.Debugf("XXXXX secret: %s", secret)

	return cloudprovider.GetProvider(self.Id, self.Name, self.AccessUrl, self.Account, secret, self.Provider)
}

func (self *SCloudaccount) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	secret, err := self.getPassword()
	if err != nil {
		return nil, err
	}
	return getSubAccounts(self.Name, self.AccessUrl, self.Account, secret, self.Provider)
}

func (self *SCloudaccount) ImportSubAccount(ctx context.Context, userCred mcclient.TokenCredential, subAccount cloudprovider.SSubAccount, autoCreateProject bool) (*SCloudprovider, bool, error) {
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

		_, err = CloudproviderManager.TableSpec().Update(provider, func() error {
			provider.Name = subAccount.Name
			provider.Enabled = true
			return nil
		})

		if err != nil {
			log.Errorf("Update cloudprovider error: %v", err)
			return nil, isNew, err
		}

		return provider, isNew, nil
	}
	// not found, create a new cloudprovider
	isNew = true

	newCloudprovider := SCloudprovider{}
	newCloudprovider.Account = subAccount.Account
	newCloudprovider.CloudaccountId = self.Id
	newCloudprovider.Provider = self.Provider
	newCloudprovider.AccessUrl = self.AccessUrl
	newCloudprovider.Enabled = true
	newCloudprovider.Status = CLOUD_PROVIDER_CONNECTED
	newCloudprovider.Name = subAccount.Name
	if !autoCreateProject {
		newCloudprovider.ProjectId = auth.AdminCredential().GetProjectId()
	}

	newCloudprovider.SetModelManager(CloudproviderManager)

	err := CloudproviderManager.TableSpec().Insert(&newCloudprovider)
	if err != nil {
		log.Errorf("insert new cloudprovider fail %s", err)
		return nil, isNew, err
	}

	passwd, err := self.getPassword()
	if err != nil {
		return nil, isNew, err
	}

	newCloudprovider.savePassword(passwd)

	if autoCreateProject {
		err = newCloudprovider.syncProject(ctx)
		if err != nil {
			log.Errorf("syncproject fail %s", err)
			return nil, isNew, err
		}
	}

	return &newCloudprovider, isNew, nil
}

func (self *SCloudaccount) AllowPerformImport(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SCloudaccount) PerformImport(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	autoCreateProject := jsonutils.QueryBoolean(data, "auto_create_project", false)
	autoSync := jsonutils.QueryBoolean(data, "auto_sync", false)

	err := self.startImportSubAccountTask(ctx, userCred, autoCreateProject, autoSync, "")

	return nil, err
}

func (self *SCloudaccount) startImportSubAccountTask(ctx context.Context, userCred mcclient.TokenCredential, autoCreateProject bool, autoSync bool, parentTaskId string) error {
	params := jsonutils.NewDict()
	if autoCreateProject {
		params.Add(jsonutils.JSONTrue, "auto_create_project")
	}
	if autoSync {
		params.Add(jsonutils.JSONTrue, "auto_sync")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "CloudAccountImportTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("create task fail %s", err)
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func getSubAccounts(name, accessUrl, account, secret, provider string) ([]cloudprovider.SSubAccount, error) {
	iprovider, err := cloudprovider.GetProvider("", name, accessUrl, account, secret, provider)
	if err != nil {
		return nil, err
	}
	return iprovider.GetSubAccounts()
}

func (self *SCloudaccount) SaveSysInfo(info jsonutils.JSONObject) {
	self.GetModelManager().TableSpec().Update(self, func() error {
		self.Sysinfo = info
		return nil
	})
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

func (self *SCloudaccount) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	extra.Add(jsonutils.Marshal(self.GetCloudproviders()), "accounts")
	return extra
}

func (self *SCloudaccount) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (self *SCloudaccount) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	return self.getMoreDetails(extra)
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
		account.LastSync = cloudprovider.LastSync
		account.Sysinfo = cloudprovider.Sysinfo
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

	_, err = CloudproviderManager.TableSpec().Update(cloudprovider, func() error {
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
	q = q.Filter(sqlchemy.OR(sqlchemy.IsEmpty(q.Field("cloudaccount_id")), sqlchemy.IsNull(q.Field("cloudaccount_id"))))
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
	driver, err := self.GetDriver()
	if err != nil {
		return 0.0, err
	}
	return driver.GetBalance()
}

func (self *SCloudaccount) AllowGetDetailsBalance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
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
