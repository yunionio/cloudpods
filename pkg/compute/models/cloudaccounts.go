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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCloudaccountManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	SInfrastructureManager
}

var CloudaccountManager *SCloudaccountManager

func init() {
	CloudaccountManager = &SCloudaccountManager{SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(SCloudaccount{}, "cloudaccounts_tbl", "cloudaccount", "cloudaccounts")}
}

type SCloudaccount struct {
	db.SEnabledStatusStandaloneResourceBase
	SInfrastructure

	AccessUrl string `width:"64" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`

	Account string `width:"128" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(64, charset='ascii'), nullable=False)
	Secret  string `width:"256" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(256, charset='ascii'), nullable=False)

	BalanceKey string    `width:"256" charset:"ascii" nullable:"true" list:"admin" create:"admin_optional"`
	LastSync   time.Time `get:"admin" list:"admin"` // = Column(DateTime, nullable=True)

	Version string `width:"32" charset:"ascii" nullable:"true" list:"admin"` // Column(VARCHAR(32, charset='ascii'), nullable=True)

	Sysinfo jsonutils.JSONObject `get:"admin"` // Column(JSONEncodedDict, nullable=True)

	Provider string `width:"64" charset:"ascii" list:"admin" create:"admin_required"`
}

func (self *SCloudaccount) getCloudproviders() []SCloudprovider {
	cloudproviders := []SCloudprovider{}
	q := CloudproviderManager.Query().Equals("cloudaccount_id", self.Id)
	if err := db.FetchModelObjects(CloudproviderManager, q, &cloudproviders); err != nil {
		log.Errorf("getCloudproviders error: %v", err)
	}
	return cloudproviders
}

func (self *SCloudaccount) ValidateDeleteCondition(ctx context.Context) error {
	if self.Enabled {
		return httperrors.NewInvalidStatusError("account is enabled")
	}
	if len(self.getCloudproviders()) > 0 {
		return httperrors.NewNotEmptyError("Not an empty cloud account")
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SCloudaccount) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return self.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (self *SCloudaccountManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	// check provider
	name, _ := data.GetString("name")
	provider, _ := data.GetString("provider")
	if !cloudprovider.IsSupported(provider) {
		return nil, httperrors.NewInputParameterError("Unsupported provider %s", provider)
	}
	// check duplication
	// url, account, provider must be unique
	account, _ := data.GetString("account")
	secret, _ := data.GetString("secret")
	url, _ := data.GetString("access_url")
	q := self.Query().Equals("provider", provider)
	if len(account) > 0 {
		q = q.Equals("account", account)
	}
	if len(url) > 0 {
		q = q.Equals("access_url", url)
	}
	if q.Count() > 0 {
		return nil, httperrors.NewConflictError("The account has been registered")
	}

	if subAccount, err := getSubAccounts(name, url, account, secret, provider); err != nil {
		return nil, err
	} else if accounts, err := subAccount.GetArray("data"); err != nil {
		return nil, err
	} else {
		data.Add(jsonutils.NewArray(accounts...), "accounts")
	}
	return self.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (self *SCloudaccount) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	self.savePassword(self.Secret)

	if subAccounts, err := data.GetArray("accounts"); err == nil && len(subAccounts) > 0 {
		for _, subAccount := range subAccounts {
			name, _ := subAccount.GetString("name")
			account, _ := subAccount.GetString("account")
			if len(name) > 0 && len(account) > 0 {
				if q := CloudproviderManager.Query().Equals("cloudaccount_id", self.Id).Equals("account", account); q.Count() > 0 {
					log.Errorf("sub account conflict")
				} else {
					newCloudprovider := SCloudprovider{
						Account:        account,
						CloudaccountId: self.Id,
						Provider:       self.Provider,
					}
					if err := CloudproviderManager.TableSpec().Insert(&newCloudprovider); err != nil {
						log.Errorf("Create cloudprovider error: %v", err)
					} else if _, err := CloudproviderManager.TableSpec().Update(&newCloudprovider, func() error {
						newCloudprovider.Name = name
						return nil
					}); err != nil {
						log.Errorf("Update cloudprovider error: %v", err)
					}
				}
			}
		}
	}

	if self.Enabled {
		self.startSyncCloudProviderInfoTask(ctx, userCred, nil, "")
	}
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
		err = self.startSyncCloudProviderInfoTask(ctx, userCred, nil, "")
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
	if len(secret) > 0 {
		err = self.savePassword(secret)
		if err != nil {
			return nil, err
		}
		changed = true
	}
	if (len(account) > 0 && account != self.Account) || (len(accessUrl) > 0 && accessUrl != self.AccessUrl) {
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
	if changed {
		self.SetStatus(userCred, CLOUD_PROVIDER_INIT, "Change credential")
		self.startSyncCloudProviderInfoTask(ctx, userCred, nil, "")
	}
	return nil, nil
}

func (self *SCloudaccount) startSyncCloudProviderInfoTask(ctx context.Context, userCred mcclient.TokenCredential, syncRange *SSyncRange, parentTaskId string) error {
	for _, cloudprovider := range self.getCloudproviders() {
		params := jsonutils.NewDict()
		if syncRange != nil {
			params.Add(jsonutils.Marshal(syncRange), "sync_range")
		}
		if cloudprovider.Enabled {
			cloudprovider.startSyncCloudProviderInfoTask(ctx, userCred, nil, "")
		}
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

func (self *SCloudaccount) AllowPerformGetSubAccounts(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SCloudaccount) AllowPerformImport(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SCloudaccount) PerformImport(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if secret, err := self.getPassword(); err != nil {
		return nil, err
	} else if subAccounts, err := getSubAccounts(self.Name, self.AccessUrl, self.Account, secret, self.Provider); err != nil {
		return nil, err
	} else if accounts, err := subAccounts.GetArray("data"); err != nil {
		return nil, err
	} else {
		enabled, _ := data.Bool("enabled")
		for _, _account := range accounts {
			name, _ := _account.GetString("name")
			account, _ := _account.GetString("account")
			if len(name) > 0 && len(account) > 0 {
				if q := CloudproviderManager.Query().Equals("cloudaccount_id", self.Id).Equals("account", account); q.Count() > 0 {
					log.Debugf("account %s has imported", account)
				} else {
					newCloudprovider := SCloudprovider{
						Account:        account,
						CloudaccountId: self.Id,
						Provider:       self.Provider,
					}
					if err := CloudproviderManager.TableSpec().Insert(&newCloudprovider); err != nil {
						log.Errorf("Create cloudprovider error: %v", err)
						return nil, err
					} else if _, err := CloudproviderManager.TableSpec().Update(&newCloudprovider, func() error {
						newCloudprovider.Name = name
						newCloudprovider.Enabled = true
						return nil
					}); err != nil {
						log.Errorf("Update cloudprovider error: %v", err)
						return nil, err
					}
					if enabled {
						newCloudprovider.startSyncCloudProviderInfoTask(ctx, userCred, &SSyncRange{FullSync: true}, "")
					}
				}
			}
		}
	}
	return jsonutils.NewDict(), nil
}

// func (self *SCloudaccount) PerformGetSubAccounts(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
// 	if !self.Enabled {
// 		return nil, httperrors.NewInvalidStatusError("Account disabled")
// 	}
// 	if provider, err := self.GetDriver(); err != nil {
// 		return nil, err
// 	} else if _subAccounts, err := provider.GetSubAccounts(); err != nil {
// 		return nil, err
// 	} else {
// 		result := jsonutils.NewDict()
// 		data := jsonutils.NewArray()
// 		accounts := []string{}
// 		for _, account := range self.getCloudproviders() {
// 			accounts = append(accounts, account.Account)
// 			_account := jsonutils.NewDict()
// 			_account.Add(jsonutils.NewString(account.Account), "account")
// 			_account.Add(jsonutils.NewString(account.Name), "name")
// 			_account.Add(jsonutils.JSONTrue, "exist")
// 			data.Add(_account)
// 		}
// 		if _subAccounts != nil {
// 			if subAccounts, err := _subAccounts.GetArray("data"); err != nil {
// 				return nil, err
// 			} else {
// 				for _, subAccount := range subAccounts {
// 					if account, err := subAccount.GetString("account"); err != nil {
// 						log.Errorf("Get subAccount error %v", err)
// 					} else if !utils.IsInStringArray(account, accounts) {
// 						_account := subAccount.(*jsonutils.JSONDict)
// 						_account.Add(jsonutils.JSONFalse, "exist")
// 						data.Add(_account)
// 					}
// 				}
// 			}
// 		}
// 		result.Add(data, "data")
// 		result.Add(jsonutils.NewInt(int64(data.Length())), "total")
// 		return result, nil
// 	}
// }

// func (manager *SCloudaccountManager) AllowPerformGetSubAccounts(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
// 	return userCred.IsSystemAdmin()
// }

// func (manager *SCloudaccountManager) PerformGetSubAccounts(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
// 	name, _ := data.GetString("name")
// 	accessUrl, _ := data.GetString("access_url")
// 	account, _ := data.GetString("account")
// 	secret, _ := data.GetString("secret")
// 	_provider, _ := data.GetString("provider")
// 	if provider, err := cloudprovider.GetProvider("", name, accessUrl, account, secret, _provider); err != nil {
// 		return nil, err
// 	} else {
// 		return provider.GetSubAccounts()
// 	}
// }

func getSubAccounts(name, accessUrl, account, secret, provider string) (jsonutils.JSONObject, error) {
	if provider, err := cloudprovider.GetProvider("", name, accessUrl, account, secret, provider); err != nil {
		return nil, err
	} else {
		return provider.GetSubAccounts()
	}
}

func (self *SCloudaccount) SaveSysInfo(info jsonutils.JSONObject) {
	self.GetModelManager().TableSpec().Update(self, func() error {
		self.Sysinfo = info
		return nil
	})
}

func (manager *SCloudaccountManager) FetchCloudproviderById(providerId string) *SCloudprovider {
	providerObj, err := manager.FetchById(providerId)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return providerObj.(*SCloudprovider)
}

func (manager *SCloudaccountManager) FetchCloudproviderByIdOrName(providerId string) *SCloudprovider {
	providerObj, err := manager.FetchByIdOrName("", providerId)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("%s", err)
		}
		return nil
	}
	return providerObj.(*SCloudprovider)
}

func (self *SCloudaccount) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
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

func (manager *SCloudaccountManager) InitializeData() error {
	cloudproviders := []SCloudprovider{}
	q := CloudproviderManager.Query()
	q = q.Filter(sqlchemy.OR(sqlchemy.IsEmpty(q.Field("cloudaccount_id")), sqlchemy.IsNull(q.Field("cloudaccount_id"))))
	if err := db.FetchModelObjects(CloudproviderManager, q, &cloudproviders); err != nil {
		return err
	}
	newAccounts := map[string]string{}
	for _, cloudprovider := range cloudproviders {
		Account, providerAccount, providerName := cloudprovider.Account, cloudprovider.Account, cloudprovider.Name
		if cloudprovider.Provider == CLOUD_PROVIDER_AZURE {
			if accountInfo := strings.Split(cloudprovider.Account, "/"); len(accountInfo) == 2 {
				if _, ok := newAccounts[accountInfo[0]]; ok {
					continue
				}
				Account, providerAccount = accountInfo[0], accountInfo[1]
				if len(cloudprovider.Description) > 0 {
					providerName = cloudprovider.Description
				}
			} else {
				log.Errorf("Error provider format %s", cloudprovider.Account)
				continue
			}
		}
		account := SCloudaccount{
			AccessUrl: cloudprovider.AccessUrl,
			Account:   Account,
			LastSync:  cloudprovider.LastSync,
			Sysinfo:   cloudprovider.Sysinfo,
			Provider:  cloudprovider.Provider,
		}
		account.SetModelManager(CloudaccountManager)
		if err := CloudaccountManager.TableSpec().Insert(&account); err != nil {
			log.Errorf("Insert Account error: %v", err)
		} else {
			newAccounts[Account] = account.Id
			if _, err := CloudaccountManager.TableSpec().Update(&account, func() error {
				account.Name = Account
				account.Status = cloudprovider.Status
				return nil
			}); err != nil {
				log.Errorf("Update Account %s error: %v", account.Id, err)
			}
			if secret, err := cloudprovider.getPassword(); err != nil {
				log.Errorf("Get password from provider %s error %v", cloudprovider.Name, err)
			} else if err := account.savePassword(secret); err != nil {
				log.Errorf("Set password for account %s error %v", account.Name, err)
			} else if _, err := CloudproviderManager.TableSpec().Update(&cloudprovider, func() error {
				cloudprovider.CloudaccountId = account.Id
				cloudprovider.Account = providerAccount
				cloudprovider.Name = providerName
				return nil
			}); err != nil {
				log.Errorf("Update provider %s error: %v", cloudprovider.Name, err)
			}
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
