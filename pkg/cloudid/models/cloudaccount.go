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
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/http/httpproxy"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	proxyapi "yunion.io/x/onecloud/pkg/apis/cloudcommon/proxy"
	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/options"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

// +onecloud:swagger-gen-ignore
type SCloudaccountManager struct {
	db.SDomainLevelResourceBaseManager
}

var CloudaccountManager *SCloudaccountManager
var isCloudacountSynced bool
var providersForSystemPolicySynced []string

func init() {
	CloudaccountManager = &SCloudaccountManager{
		SDomainLevelResourceBaseManager: db.NewDomainLevelResourceBaseManager(
			SCloudaccount{},
			"cloudaccounts_tbl",
			"cloudaccount",
			"cloudaccounts",
		),
	}
	CloudaccountManager.SetVirtualObject(CloudaccountManager)
	isCloudacountSynced = false
	providersForSystemPolicySynced = []string{}
}

type SCloudaccount struct {
	db.SStandaloneResourceBase
	db.SDomainizedResourceBase

	Provider    string `width:"64" charset:"ascii" list:"domain"`
	Brand       string `width:"64" charset:"utf8" nullable:"true" list:"domain"`
	IamLoginUrl string `width:"512" charset:"ascii"`
}

func (manager *SCloudaccountManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	return []db.SScopeResourceCount{}, nil
}

func (manager *SCloudaccountManager) GetICloudaccounts() ([]SCloudaccount, error) {
	s := auth.GetAdminSession(context.Background(), options.Options.Region, "")

	data := []jsonutils.JSONObject{}
	offset := int64(0)
	params := jsonutils.NewDict()
	params.Set("scope", jsonutils.NewString("system"))
	params.Set("limit", jsonutils.NewInt(1024))
	for {
		params.Set("offset", jsonutils.NewInt(offset))
		result, err := modules.Cloudaccounts.List(s, params)
		if err != nil {
			return nil, errors.Wrap(err, "modules.Cloudaccounts.List")
		}
		data = append(data, result.Data...)
		if len(data) >= result.Total {
			break
		}
		offset += 1024
	}

	accounts := []SCloudaccount{}
	err := jsonutils.Update(&accounts, data)
	if err != nil {
		return nil, errors.Wrap(err, "jsonutils.Update")
	}
	return accounts, nil
}

func (manager *SCloudaccountManager) GetCloudaccounts() ([]SCloudaccount, error) {
	accounts := []SCloudaccount{}
	q := manager.Query()
	err := db.FetchModelObjects(manager, q, &accounts)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (manager *SCloudaccountManager) syncCloudaccounts(ctx context.Context, userCred mcclient.TokenCredential) (localAccounts []SCloudaccount, result compare.SyncResult) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	accounts, err := manager.GetICloudaccounts()
	if err != nil {
		result.Error(errors.Wrap(err, "GetRegionCloudaccounts"))
		return
	}

	dbAccounts, err := manager.GetCloudaccounts()
	if err != nil {
		result.Error(errors.Wrap(err, "GetLocalCloudaccounts"))
		return
	}

	removed := make([]SCloudaccount, 0)
	commondb := make([]SCloudaccount, 0)
	commonext := make([]SCloudaccount, 0)
	added := make([]SCloudaccount, 0)

	err = compare.CompareSets(dbAccounts, accounts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrap(err, "compare.CompareSets"))
		return
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemoveCloudaccount(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(commondb); i++ {
		err = commondb[i].syncWithICloudaccount(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		localAccounts = append(localAccounts, commondb[i])
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		account, err := manager.newFromICloudaccount(ctx, userCred, &added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		localAccounts = append(localAccounts, *account)
		result.Add()
	}
	return
}

func (self *SCloudaccount) removeCloudproviders(ctx context.Context, userCred mcclient.TokenCredential) error {
	providers, err := self.GetCloudproviders()
	if err != nil {
		return errors.Wrap(err, "GetCloudproviders")
	}
	for i := range providers {
		err = providers[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "provider.Delete")
		}
	}
	return nil
}

func (self *SCloudaccount) GetCloudproviderId() string {
	return ""
}

func (self *SCloudaccount) removeCloudgroupcaches(ctx context.Context, userCred mcclient.TokenCredential) error {
	caches, err := self.GetCloudgroupcaches()
	if err != nil {
		return errors.Wrap(err, "GetCloudgroupcaches")
	}
	for i := range caches {
		err = caches[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "caches[i].RealDelete")
		}
	}
	return nil
}

func (self *SCloudaccount) syncRemoveCloudaccount(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := self.syncRemoveClouduser(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "syncRemoveClouduser")
	}

	err = self.removeCloudproviders(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "removeCloudproviders")
	}

	err = self.removeCloudgroupcaches(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "removeCloudgroupcaches")
	}

	return self.Delete(ctx, userCred)
}

func (self *SCloudaccount) syncRemoveClouduser(ctx context.Context, userCred mcclient.TokenCredential) error {
	users, err := self.getCloudusers()
	if err != nil {
		return errors.Wrap(err, "getCloudusers")
	}
	for i := range users {
		err = users[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "RealDelete user %s(%s)", users[i].Name, users[i].Id)
		}
	}
	return nil
}

func (manager *SCloudaccountManager) newFromICloudaccount(ctx context.Context, userCred mcclient.TokenCredential, account *SCloudaccount) (*SCloudaccount, error) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	account.SetModelManager(manager, account)
	err := manager.TableSpec().Insert(ctx, account)
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}

	return account, nil
}

func (self *SCloudaccount) syncWithICloudaccount(ctx context.Context, userCred mcclient.TokenCredential, account SCloudaccount) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.Name = account.Name
		self.DomainId = account.DomainId
		self.Brand = account.Brand
		self.IamLoginUrl = account.IamLoginUrl
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.UpdateWithLock")
	}
	return nil
}

func (manager *SCloudaccountManager) SyncCloudaccounts(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	localAccounts, result := manager.syncCloudaccounts(ctx, userCred)
	log.Infof("SyncCloudaccounts: %s", result.Result())
	for i, account := range localAccounts {
		lockman.LockObject(ctx, &localAccounts[i])
		defer lockman.ReleaseObject(ctx, &localAccounts[i])

		result = account.syncCloudprovider(ctx, userCred)
		log.Debugf("sync cloudprovider for cloudaccount %s(%s) result: %s", account.Name, account.Id, result.Result())
	}
	isCloudacountSynced = true
}

func waitForSync() {
	now := time.Now()
	for !isCloudacountSynced {
		log.Infof("cloudaccount not sync, wait for 10 seconds")
		time.Sleep(time.Second * 10)
		if time.Now().Sub(now) > time.Minute*3 {
			break
		}
	}
}

func (self SCloudaccount) GetGlobalId() string {
	return self.Id
}

func (self SCloudaccount) GetExternalId() string {
	return self.Id
}

func (manager *SCloudaccountManager) FetchAccount(ctx context.Context, id string) (*SCloudaccount, error) {
	account, err := manager.FetchById(id)
	if err != nil {
		if err == sql.ErrNoRows {
			session := auth.GetAdminSession(context.Background(), options.Options.Region, "")
			result, err := modules.Cloudaccounts.Get(session, id, nil)
			if err != nil {
				return nil, errors.Wrap(err, "Cloudaccounts.Get")
			}
			_account := &SCloudaccount{}
			_account.SetModelManager(manager, _account)
			err = result.Unmarshal(_account)
			if err != nil {
				return nil, errors.Wrap(err, "result.Unmarshal")
			}

			lockman.LockRawObject(ctx, manager.KeywordPlural(), id)
			defer lockman.ReleaseRawObject(ctx, manager.KeywordPlural(), id)
			return _account, manager.TableSpec().InsertOrUpdate(ctx, _account)
		}
		return nil, errors.Wrap(err, "manager.FetchById")
	}
	return account.(*SCloudaccount), nil
}

type SCloudDelegate struct {
	Id         string
	Name       string
	Enabled    bool
	Status     string
	SyncStatus string

	AccessUrl string
	Account   string
	Secret    string

	Provider string
	Brand    string

	ProxySetting proxyapi.SProxySetting
}

func (self *SCloudaccount) getCloudDelegate(ctx context.Context) (*SCloudDelegate, error) {
	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	result, err := modules.Cloudaccounts.Get(s, self.Id, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Cloudaccounts.Get")
	}
	account := &SCloudDelegate{}
	err = result.Unmarshal(account)
	if err != nil {
		return nil, errors.Wrap(err, "result.Unmarshal")
	}
	return account, nil
}

func (self *SCloudaccount) GetProvider() (cloudprovider.ICloudProvider, error) {
	delegate, err := self.getCloudDelegate(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "getCloudDelegate")
	}
	return delegate.GetProvider()
}

func (self *SCloudaccount) GetCloudDelegaes(ctx context.Context) ([]SCloudDelegate, error) {
	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	params := map[string]string{"cloudaccount": self.Id}
	result, err := modules.Cloudproviders.List(s, jsonutils.Marshal(params))
	if err != nil {
		return nil, errors.Wrap(err, "Cloudproviders.List")
	}
	providers := []SCloudDelegate{}
	err = jsonutils.Update(&providers, result.Data)
	if err != nil {
		return nil, errors.Wrap(err, "jsonutils.Update")
	}
	return providers, nil
}

func (account *SCloudDelegate) getPassword() (string, error) {
	return utils.DescryptAESBase64(account.Id, account.Secret)
}

func (account *SCloudDelegate) getAccessUrl() string {
	return account.AccessUrl
}

func (self *SCloudaccount) GetProviderFactory() (cloudprovider.ICloudProviderFactory, error) {
	return cloudprovider.GetProviderFactory(self.Provider)
}

func (account *SCloudDelegate) GetProvider() (cloudprovider.ICloudProvider, error) {
	if !account.Enabled {
		log.Warningf("Cloud account %s is disabled", account.Name)
	}

	accessUrl := account.getAccessUrl()
	passwd, err := account.getPassword()
	if err != nil {
		return nil, err
	}
	var proxyFunc httputils.TransportProxyFunc
	{
		cfg := &httpproxy.Config{
			HTTPProxy:  account.ProxySetting.HTTPProxy,
			HTTPSProxy: account.ProxySetting.HTTPSProxy,
			NoProxy:    account.ProxySetting.NoProxy,
		}
		cfgProxyFunc := cfg.ProxyFunc()
		proxyFunc = func(req *http.Request) (*url.URL, error) {
			return cfgProxyFunc(req.URL)
		}
	}
	return cloudprovider.GetProvider(cloudprovider.ProviderConfig{
		Id:        account.Id,
		Name:      account.Name,
		Vendor:    account.Provider,
		URL:       accessUrl,
		Account:   account.Account,
		Secret:    passwd,
		ProxyFunc: proxyFunc,
	})
}

func (self *SCloudaccount) StartSyncCloudusersTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "SyncCloudusersTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudaccount) getCloudusers() ([]SClouduser, error) {
	users := []SClouduser{}
	q := ClouduserManager.Query().Equals("cloudaccount_id", self.Id)
	err := db.FetchModelObjects(ClouduserManager, q, &users)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return users, nil
}

func (self *SCloudaccount) GetCloudusers() ([]SClouduser, error) {
	users := []SClouduser{}
	q := ClouduserManager.Query().Equals("status", api.CLOUD_USER_STATUS_AVAILABLE).Equals("cloudaccount_id", self.Id)
	err := db.FetchModelObjects(ClouduserManager, q, &users)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return users, nil
}

func (self *SCloudaccount) SyncCloudusers(ctx context.Context, userCred mcclient.TokenCredential, iUsers []cloudprovider.IClouduser) ([]SClouduser, []cloudprovider.IClouduser, compare.SyncResult) {
	result := compare.SyncResult{}
	dbUsers, err := self.GetCloudusers()
	if err != nil {
		result.Error(errors.Wrap(err, "GetCloudusersByProviderId"))
		return nil, nil, result
	}

	localUsers := []SClouduser{}
	remoteUsers := []cloudprovider.IClouduser{}

	removed := make([]SClouduser, 0)
	commondb := make([]SClouduser, 0)
	commonext := make([]cloudprovider.IClouduser, 0)
	added := make([]cloudprovider.IClouduser, 0)

	err = compare.CompareSets(dbUsers, iUsers, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrap(err, "compare.CompareSets"))
		return nil, nil, result
	}

	for i := 0; i < len(removed); i++ {
		if len(removed[i].ExternalId) > 0 {
			err = removed[i].RealDelete(ctx, userCred)
			if err != nil {
				result.DeleteError(err)
				continue
			}
			result.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithClouduser(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		localUsers = append(localUsers, commondb[i])
		remoteUsers = append(remoteUsers, commonext[i])
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		user, err := self.newClouduser(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		localUsers = append(localUsers, *user)
		remoteUsers = append(remoteUsers, added[i])
		result.Add()
	}

	return localUsers, remoteUsers, result
}

func (self *SCloudaccount) newClouduser(ctx context.Context, userCred mcclient.TokenCredential, iUser cloudprovider.IClouduser) (*SClouduser, error) {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	user := &SClouduser{}
	user.SetModelManager(ClouduserManager, user)
	user.Name = iUser.GetName()
	user.ExternalId = iUser.GetGlobalId()
	user.Status = api.CLOUD_USER_STATUS_AVAILABLE
	user.CloudaccountId = self.Id
	user.DomainId = self.DomainId
	switch iUser.IsConsoleLogin() {
	case true:
		user.IsConsoleLogin = tristate.True
	case false:
		user.IsConsoleLogin = tristate.False
	}
	err := ClouduserManager.TableSpec().Insert(ctx, user)
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}
	return user, nil
}

func (self *SCloudaccount) GetCloudpolicies() ([]SCloudpolicy, error) {
	q := CloudpolicyManager.Query().Equals("provider", self.Provider)
	policies := []SCloudpolicy{}
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SCloudaccount) GetSystemCloudpolicies() ([]SCloudpolicy, error) {
	q := CloudpolicyManager.Query().Equals("provider", self.Provider).Equals("policy_type", api.CLOUD_POLICY_TYPE_SYSTEM)
	policies := []SCloudpolicy{}
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SCloudaccount) GetCustomCloudpolicies() ([]SCloudpolicy, error) {
	q := CloudpolicyManager.Query().Equals("provider", self.Provider).Equals("policy_type", api.CLOUD_POLICY_TYPE_CUSTOM).Equals("domain_id", self.DomainId)
	policies := []SCloudpolicy{}
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SCloudaccount) GetCloudpolicycaches(policyIds []string, cloudproviderId string) ([]SCloudpolicycache, error) {
	q := CloudpolicycacheManager.Query().Equals("cloudaccount_id", self.Id)
	if len(policyIds) > 0 {
		q = q.In("cloudpolicy_id", policyIds)
	}
	if len(cloudproviderId) > 0 {
		q = q.Equals("cloudprovider_id", cloudproviderId)
	}
	caches := []SCloudpolicycache{}
	err := db.FetchModelObjects(CloudpolicycacheManager, q, &caches)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return caches, nil
}

func (self *SCloudaccount) SyncCustomCloudpoliciesFromCloud(ctx context.Context, userCred mcclient.TokenCredential) error {
	factory, err := self.GetProviderFactory()
	if err != nil {
		return errors.Wrapf(err, "GetProviderFactory")
	}
	if factory.IsCloudpolicyWithSubscription() {
		providers, err := self.GetCloudproviders()
		if err != nil {
			return errors.Wrapf(err, "GetCloudproviders")
		}
		for i := range providers {
			err = providers[i].SyncCustomCloudpoliciesFromCloud(ctx, userCred)
			if err != nil {
				return errors.Wrapf(err, "SyncCustomCloudpoliciesFromCloud for cloudprovider %s(%s)", providers[i].Provider, providers[i].Name)
			}
		}
		return nil
	}

	provider, err := self.GetProvider()
	if err != nil {
		return errors.Wrapf(err, "GetProvider")
	}

	policies, err := provider.GetICustomCloudpolicies()
	if err != nil {
		return errors.Wrapf(err, "GetICustomCloudpolicies for account %s(%s)", self.Name, self.Provider)
	}
	result := self.SyncCustomCloudpoliciesToLocal(ctx, userCred, policies, "")
	log.Infof("Sync %s custom policies for account %s result: %s", self.Provider, self.Name, result.Result())
	return nil
}

func (self *SCloudaccount) SyncCustomCloudpoliciesToLocal(ctx context.Context, userCred mcclient.TokenCredential, iPolicies []cloudprovider.ICloudpolicy, cloudproviderId string) compare.SyncResult {
	result := compare.SyncResult{}

	dbCaches, err := self.GetCloudpolicycaches(nil, cloudproviderId)
	if err != nil {
		result.Error(errors.Wrap(err, "GetCloudpolicycaches"))
		return result
	}
	removed := make([]SCloudpolicycache, 0)
	commondb := make([]SCloudpolicycache, 0)
	commonext := make([]cloudprovider.ICloudpolicy, 0)
	added := make([]cloudprovider.ICloudpolicy, 0)
	err = compare.CompareSets(dbCaches, iPolicies, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrap(err, "compare.CompareSets"))
		return result
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].Delete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(added); i++ {
		policy, err := self.newCustomPolicy(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		err = policy.newCloudpolicycache(ctx, userCred, added[i], self.Id, cloudproviderId)
		if err != nil {
			result.AddError(errors.Wrap(err, "newCloudpolicycache"))
			continue
		}
		result.Add()
	}

	return result
}

func (self *SCloudaccount) newCustomPolicy(ctx context.Context, userCred mcclient.TokenCredential, iPolicy cloudprovider.ICloudpolicy) (*SCloudpolicy, error) {
	policies, err := self.GetCustomCloudpolicies()
	if err != nil {
		return nil, errors.Wrap(err, "GetCustomCloudpolicies")
	}
	document, err := iPolicy.GetDocument()
	if err != nil {
		return nil, errors.Wrap(err, "GetDocument")
	}
	for i := range policies {
		if policies[i].Document != nil && policies[i].Document.Equals(document) {
			return &policies[i], nil
		}
	}
	policy, err := self.newCloudpolicy(ctx, userCred, iPolicy, api.CLOUD_POLICY_TYPE_CUSTOM)
	if err != nil {
		return nil, errors.Wrap(err, "newFromCloudpolicy")
	}
	return policy, nil
}

func (self *SCloudaccount) GetCloudaccountByProvider(provider string) ([]SCloudaccount, error) {
	accounts := []SCloudaccount{}
	q := CloudaccountManager.Query().Equals("provider", provider)
	err := db.FetchModelObjects(CloudaccountManager, q, &accounts)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return accounts, nil
}

func (self *SCloudaccount) SyncSystemCloudpoliciesFromCloud(ctx context.Context, userCred mcclient.TokenCredential) error {
	accounts, err := self.GetCloudaccountByProvider(self.Provider)
	if err != nil {
		return errors.Wrapf(err, "GetCloudaccountByProvider")
	}
	iPolicies := []cloudprovider.ICloudpolicy{}
	policyMaps := map[string]bool{}
	for i := range accounts {
		provider, err := accounts[i].GetProvider()
		if err != nil {
			return errors.Wrapf(err, "GetProvider for account %s(%s)", accounts[i].Name, accounts[i].Provider)
		}
		policies, err := provider.GetISystemCloudpolicies()
		if err != nil {
			return errors.Wrapf(err, "GetISystemCloudpolicies for account %s(%s)", accounts[i].Name, accounts[i].Provider)
		}
		for i := range policies {
			_, ok := policyMaps[policies[i].GetGlobalId()]
			if !ok {
				policyMaps[policies[i].GetGlobalId()] = true
				iPolicies = append(iPolicies, policies[i])
			}
		}
	}
	return self.syncSystemCloudpoliciesFromCloud(ctx, userCred, iPolicies)
}

func (self *SCloudaccount) syncSystemCloudpoliciesFromCloud(ctx context.Context, userCred mcclient.TokenCredential, iPolicies []cloudprovider.ICloudpolicy) error {
	result := compare.SyncResult{}
	dbPolicies, err := self.GetSystemCloudpolicies()
	if err != nil {
		return errors.Wrapf(err, "GetSystemCloudpolicies")
	}

	removed := make([]SCloudpolicy, 0)
	commondb := make([]SCloudpolicy, 0)
	commonext := make([]cloudprovider.ICloudpolicy, 0)
	added := make([]cloudprovider.ICloudpolicy, 0)

	err = compare.CompareSets(dbPolicies, iPolicies, &removed, &commondb, &commonext, &added)
	if err != nil {
		return errors.Wrapf(err, "compare.CompareSets")
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemove(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudpolicy(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		_, err := self.newCloudpolicy(ctx, userCred, added[i], api.CLOUD_POLICY_TYPE_SYSTEM)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	if !utils.IsInStringArray(self.Provider, providersForSystemPolicySynced) {
		providersForSystemPolicySynced = append(providersForSystemPolicySynced, self.Provider)
	}

	log.Infof("Sync %s system policies result: %s", self.Provider, result.Result())
	return nil
}

func (self *SCloudaccount) newCloudpolicy(ctx context.Context, userCred mcclient.TokenCredential, iPolicy cloudprovider.ICloudpolicy, policyType string) (*SCloudpolicy, error) {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	policy := &SCloudpolicy{}
	policy.SetModelManager(CloudpolicyManager, policy)
	var err error
	policy.Document, err = iPolicy.GetDocument()
	if err != nil {
		return nil, errors.Wrapf(err, "GetDocument")
	}
	policy.Name = iPolicy.GetName()
	policy.Status = api.CLOUD_POLICY_STATUS_AVAILABLE
	policy.PolicyType = policyType
	if policy.PolicyType == api.CLOUD_POLICY_TYPE_SYSTEM {
		policy.IsPublic = true
	} else {
		policy.DomainId = self.DomainId
	}
	policy.Provider = self.Provider
	policy.ExternalId = iPolicy.GetGlobalId()
	policy.Description = iPolicy.GetDescription()
	return policy, CloudpolicyManager.TableSpec().Insert(ctx, policy)
}

func (self *SCloudaccount) GetCloudproviders() ([]SCloudprovider, error) {
	q := CloudproviderManager.Query().Equals("cloudaccount_id", self.Id)
	providers := []SCloudprovider{}
	err := db.FetchModelObjects(CloudproviderManager, q, &providers)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return providers, nil
}

func (self *SCloudaccount) GetICloudprovider() ([]SCloudprovider, error) {
	s := auth.GetAdminSession(context.Background(), options.Options.Region, "")
	data := []jsonutils.JSONObject{}
	offset := int64(0)
	params := map[string]interface{}{
		"cloudaccount": self.Id,
		"scope":        "system",
		"limit":        1024,
	}
	for {
		params["offset"] = offset
		result, err := modules.Cloudproviders.List(s, jsonutils.Marshal(params))
		if err != nil {
			return nil, errors.Wrap(err, "Cloudproviders.List")
		}
		data = append(data, result.Data...)
		if len(data) >= result.Total {
			break
		}
		offset += 1024
	}
	providers := []SCloudprovider{}
	err := jsonutils.Update(&providers, data)
	if err != nil {
		return nil, errors.Wrap(err, "jsonutils.Update")
	}
	return providers, nil
}

func (self *SCloudaccount) syncCloudprovider(ctx context.Context, userCred mcclient.TokenCredential) compare.SyncResult {
	result := compare.SyncResult{}

	providers, err := self.GetICloudprovider()
	if err != nil {
		result.Error(errors.Wrap(err, "GetRegionCloudprovider"))
		return result
	}

	dbProviders, err := self.GetCloudproviders()
	if err != nil {
		result.Error(errors.Wrap(err, "GetCloudproviders"))
		return result
	}

	removed := make([]SCloudprovider, 0)
	commondb := make([]SCloudprovider, 0)
	commonext := make([]SCloudprovider, 0)
	added := make([]SCloudprovider, 0)

	err = compare.CompareSets(dbProviders, providers, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrap(err, "compare.CompareSets"))
		return result
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].Delete(ctx, userCred)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(commondb); i++ {
		err = commondb[i].syncWithRegionProvider(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		err = CloudproviderManager.newFromRegionProvider(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (self *SCloudaccount) GetCloudgroups() ([]SCloudgroup, error) {
	groups := []SCloudgroup{}
	q := CloudgroupManager.Query().Equals("provider", self.Provider).Equals("domain_id", self.DomainId)
	err := db.FetchModelObjects(CloudgroupManager, q, &groups)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return groups, nil
}

func (self *SCloudaccount) GetCloudgroupcaches() ([]SCloudgroupcache, error) {
	caches := []SCloudgroupcache{}
	q := CloudgroupcacheManager.Query().Equals("cloudaccount_id", self.Id)
	err := db.FetchModelObjects(CloudgroupcacheManager, q, &caches)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return caches, nil
}

func (manager *SCloudaccountManager) GetSupportCreateCloudgroupAccounts() ([]SCloudaccount, error) {
	accounts := []SCloudaccount{}
	q := manager.Query().In("provider", cloudprovider.GetSupportCloudgroupProviders())
	err := db.FetchModelObjects(manager, q, &accounts)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return accounts, nil
}

func (manager *SCloudaccountManager) GetSupportCloudIdAccounts() ([]SCloudaccount, error) {
	accounts := []SCloudaccount{}
	q := manager.Query().In("provider", cloudprovider.GetSupportCloudIdProvider())
	err := db.FetchModelObjects(manager, q, &accounts)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return accounts, nil
}

func (manager *SCloudaccountManager) SyncCloudidSystemPolicies(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	accounts, err := manager.GetSupportCloudIdAccounts()
	if err != nil {
		log.Errorf("GetSupportCloudIdAccounts error: %v", err)
		return
	}
	providers := []string{}
	for i := range accounts {
		if isStart && utils.IsInStringArray(accounts[i].Provider, providersForSystemPolicySynced) {
			continue
		}
		if utils.IsInStringArray(accounts[i].Provider, providers) {
			continue
		}
		err = accounts[i].StartSystemCloudpolicySyncTask(ctx, userCred, "")
		if err != nil {
			log.Errorf("StartSystemCloudpolicySyncTask for account %s(%s) error: %v", accounts[i].Name, accounts[i].Provider, err)
			continue
		}
		providers = append(providers, accounts[i].Provider)
	}
}

func (self *SCloudaccount) StartSystemCloudpolicySyncTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "SystemCloudpolicySyncTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (manager *SCloudaccountManager) SyncCloudidResources(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	waitForSync()
	accounts, err := manager.GetSupportCloudIdAccounts()
	if err != nil {
		log.Errorf("GetSupportCloudIdAccounts error: %v", err)
		return
	}
	for i := range accounts {
		if !utils.IsInStringArray(accounts[i].Provider, providersForSystemPolicySynced) {
			log.Warningf("%s system policies not syncing, sync for next loop", accounts[i].Provider)
			continue
		}
		err = accounts[i].StartSyncCloudIdResourcesTask(ctx, userCred, "")
		if err != nil {
			log.Errorf("StartSyncCloudIdResourcesTask for account %s(%s) error: %v", accounts[i].Name, accounts[i].Provider, err)
		}
	}
}

func (self *SCloudaccount) StartSyncCloudIdResourcesTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "SyncCloudIdResourcesTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudaccount) SyncCloudgroupcaches(ctx context.Context, userCred mcclient.TokenCredential, iGroups []cloudprovider.ICloudgroup) error {
	result := compare.SyncResult{}

	dbCaches, err := self.GetCloudgroupcaches()
	if err != nil {
		return errors.Wrapf(err, "GetCloudgroupcaches")
	}

	removed := make([]SCloudgroupcache, 0)
	commondb := make([]SCloudgroupcache, 0)
	commonext := make([]cloudprovider.ICloudgroup, 0)
	added := make([]cloudprovider.ICloudgroup, 0)

	err = compare.CompareSets(dbCaches, iGroups, &removed, &commondb, &commonext, &added)
	if err != nil {
		return errors.Wrapf(err, "compare.CompareSets")
	}

	for i := 0; i < len(removed); i++ {
		if len(removed[i].ExternalId) > 0 { // 只删除云上已经删除过的组
			err = removed[i].RealDelete(ctx, userCred)
			if err != nil {
				result.DeleteError(err)
				continue
			}
			result.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err = commondb[i].syncWithCloudgroupcache(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		_, err := self.newCloudgroup(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	log.Infof("Sync groups for %s(%s) result: %s", self.Name, self.Provider, result.Result())
	return nil
}

func (self *SCloudaccount) newCloudgroup(ctx context.Context, userCred mcclient.TokenCredential, iGroup cloudprovider.ICloudgroup) (*SCloudgroupcache, error) {
	group, err := self.GetOrCreateCloudgroup(ctx, userCred, iGroup)
	if err != nil {
		return nil, errors.Wrap(err, "GetOrCreateCloudgroup")
	}
	cache, err := CloudgroupcacheManager.newFromCloudgroup(ctx, userCred, iGroup, group, self.Id)
	if err != nil {
		return nil, errors.Wrap(err, "newFromCloudgroup")
	}
	return cache, nil
}

func (self *SCloudaccount) GetSystemPolicyByExternalId(id string) (*SCloudpolicy, error) {
	policies := []SCloudpolicy{}
	q := CloudpolicyManager.Query().Equals("external_id", id).Equals("provider", self.Provider)
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	if len(policies) == 0 {
		return nil, errors.Wrapf(sql.ErrNoRows, "search by external id %s", id)
	}
	if len(policies) > 1 {
		return nil, errors.Wrapf(sqlchemy.ErrDuplicateEntry, "%d policy find by external id %s", len(policies), id)
	}
	return &policies[0], nil
}

func (self *SCloudaccount) GetCustomPolicyByExternalId(id string) (*SCloudpolicy, error) {
	policies := []SCloudpolicy{}
	sq := CloudpolicycacheManager.Query("cloudpolicy_id").Equals("cloudaccount_id", self.Id).Equals("external_id", id)
	q := CloudpolicyManager.Query().In("id", sq.SubQuery())
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	if len(policies) == 0 {
		return nil, errors.Wrapf(sql.ErrNoRows, "search by external id %s", id)
	}
	if len(policies) > 1 {
		return nil, errors.Wrapf(sqlchemy.ErrDuplicateEntry, "%d policy find by external id %s", len(policies), id)
	}
	return &policies[0], nil
}

func (self *SCloudaccount) GetOrCreateCloudgroup(ctx context.Context, userCred mcclient.TokenCredential, iGroup cloudprovider.ICloudgroup) (*SCloudgroup, error) {
	groups, err := self.GetCloudgroups()
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudgroups")
	}
	system, err := iGroup.GetISystemCloudpolicies()
	if err != nil {
		return nil, errors.Wrap(err, "GetICloudpolicies")
	}
	custom, err := iGroup.GetICustomCloudpolicies()
	if err != nil {
		return nil, errors.Wrap(err, "GetICustomCloudpolicies")
	}

	for i := range groups {
		isEqual, err := groups[i].IsEqual(system, custom)
		if err != nil {
			return nil, errors.Wrap(err, "IsEqual")
		}
		if isEqual {
			return &groups[i], nil
		}
	}
	policyIds := []string{}
	for i := range system {
		policy, err := self.GetSystemPolicyByExternalId(system[i].GetGlobalId())
		if err != nil {
			return nil, errors.Wrapf(err, "GetSystemPolicyByExternalId")
		}
		if !utils.IsInStringArray(policy.Id, policyIds) {
			policyIds = append(policyIds, policy.Id)
		}
	}
	for i := range custom {
		policy, err := self.GetCustomPolicyByExternalId(custom[i].GetGlobalId())
		if err != nil {
			return nil, errors.Wrap(err, "GetCustomPolicyByExternalId")
		}
		if !utils.IsInStringArray(policy.Id, policyIds) {
			policyIds = append(policyIds, policy.Id)
		}
	}
	group, err := CloudgroupManager.newCloudgroup(ctx, userCred, iGroup, self.Provider, self.DomainId)
	if err != nil {
		return nil, errors.Wrap(err, "newCloudgroup")
	}
	for _, policyId := range policyIds {
		group.attachPolicy(policyId)
	}
	return group, nil
}

func (self *SCloudaccount) getOrCacheCustomCloudpolicy(ctx context.Context, providerId, policyId string) error {
	cache, err := CloudpolicycacheManager.Register(ctx, self.Id, providerId, policyId)
	if err != nil {
		return errors.Wrapf(err, "Register")
	}
	return cache.cacheCustomCloudpolicy()
}

func (self *SCloudaccount) SyncCustomCloudpoliciesForCloud(ctx context.Context, userCred mcclient.TokenCredential, clouduser *SClouduser) error {
	factory, err := self.GetProviderFactory()
	if err != nil {
		return errors.Wrap(err, "GetProviderFactory")
	}

	if factory.IsClouduserpolicyWithSubscription() {
		providers, err := self.GetCloudproviders()
		if err != nil {
			return errors.Wrap(err, "GetCloudproviders")
		}
		for i := range providers {
			err = providers[i].SyncCustomCloudpoliciesForCloud(ctx, clouduser)
			if err != nil {
				return errors.Wrapf(err, "SyncSystemCloudpoliciesForCloud for cloudprovider %s", providers[i].Name)
			}
		}
		return nil
	}

	policyIds := []string{}
	policies, err := clouduser.GetCustomCloudpolicies("")
	if err != nil {
		return errors.Wrap(err, "GetSystemCloudpolicies")
	}
	for i := range policies {
		err = self.getOrCacheCustomCloudpolicy(ctx, "", policies[i].Id)
		if err != nil {
			return errors.Wrapf(err, "getOrCacheCloudpolicy %s(%s) for account %s", policies[i].Name, policies[i].Provider, self.Name)
		}
		policyIds = append(policyIds, policies[i].Id)
	}

	if !factory.IsSupportCreateCloudgroup() {
		policies, err = clouduser.GetCustomCloudgroupPolicies()
		if err != nil {
			return errors.Wrap(err, "GetSystemCloudgroupPolicies")
		}
		for i := range policies {
			err = self.getOrCacheCustomCloudpolicy(ctx, "", policies[i].Id)
			if err != nil {
				return errors.Wrapf(err, "getOrCacheCloudpolicy %s(%s) for account %s", policies[i].Name, policies[i].Provider, self.Name)
			}
			policyIds = append(policyIds, policies[i].Id)
		}
	}

	dbCaches, err := self.GetCloudpolicycaches(policyIds, "")
	if err != nil {
		return errors.Wrapf(err, "GetCloudpolicycaches")
	}

	iUser, err := clouduser.GetIClouduser()
	if err != nil {
		return errors.Wrapf(err, "GetIClouduser")
	}

	iPolicies, err := iUser.GetICustomCloudpolicies()
	if err != nil {
		return errors.Wrap(err, "GetISystemCloudpolicies")
	}

	added := make([]SCloudpolicycache, 0)
	commondb := make([]SCloudpolicycache, 0)
	commonext := make([]cloudprovider.ICloudpolicy, 0)
	removed := make([]cloudprovider.ICloudpolicy, 0)

	err = compare.CompareSets(dbCaches, iPolicies, &added, &commondb, &commonext, &removed)
	if err != nil {
		return errors.Wrap(err, "compare.CompareSets")
	}

	result := compare.SyncResult{}
	for i := 0; i < len(removed); i++ {
		err = iUser.DetachCustomPolicy(removed[i].GetGlobalId())
		if err != nil {
			result.DeleteError(err)
			logclient.AddSimpleActionLog(self, logclient.ACT_DETACH_POLICY, err, userCred, false)
			continue
		}
		result.Delete()
	}

	result.UpdateCnt = len(commondb)

	for i := 0; i < len(added); i++ {
		err = iUser.AttachCustomPolicy(added[i].ExternalId)
		if err != nil {
			result.AddError(err)
			logclient.AddSimpleActionLog(self, logclient.ACT_DETACH_POLICY, err, userCred, false)
			continue
		}
		result.Add()
	}

	if result.IsError() {
		return result.AllError()
	}

	log.Infof("Sync %s(%s) custom policies for user %s result: %s", self.Name, self.Provider, clouduser.Name, result.Result())
	return nil
}

func (self *SCloudaccount) SyncSystemCloudpoliciesForCloud(ctx context.Context, userCred mcclient.TokenCredential, clouduser *SClouduser) error {
	factory, err := self.GetProviderFactory()
	if err != nil {
		return errors.Wrap(err, "GetProviderFactory")
	}

	if factory.IsClouduserpolicyWithSubscription() {
		providers, err := self.GetCloudproviders()
		if err != nil {
			return errors.Wrap(err, "GetCloudproviders")
		}
		for i := range providers {
			err = providers[i].SyncSystemCloudpoliciesForCloud(ctx, clouduser)
			if err != nil {
				return errors.Wrapf(err, "SyncSystemCloudpoliciesForCloud for cloudprovider %s", providers[i].Name)
			}
		}
		return nil
	}

	dbPolicies, err := clouduser.GetSystemCloudpolicies("")
	if err != nil {
		return errors.Wrap(err, "GetSystemCloudpolicies")
	}

	if !factory.IsSupportCreateCloudgroup() {
		policyMaps := map[string]SCloudpolicy{}
		for i := range dbPolicies {
			policyMaps[dbPolicies[i].Id] = dbPolicies[i]
		}
		policies, err := clouduser.GetSystemCloudgroupPolicies()
		if err != nil {
			return errors.Wrap(err, "GetSystemCloudgroupPolicies")
		}
		for i := range policies {
			_, ok := policyMaps[policies[i].Id]
			if !ok {
				policyMaps[policies[i].Id] = policies[i]
				dbPolicies = append(dbPolicies, policies[i])
			}
		}
	}

	iUser, err := clouduser.GetIClouduser()
	if err != nil {
		return errors.Wrapf(err, "GetIClouduser")
	}

	iPolicies, err := iUser.GetISystemCloudpolicies()
	if err != nil {
		return errors.Wrap(err, "GetISystemCloudpolicies")
	}

	added := make([]SCloudpolicy, 0)
	commondb := make([]SCloudpolicy, 0)
	commonext := make([]cloudprovider.ICloudpolicy, 0)
	removed := make([]cloudprovider.ICloudpolicy, 0)

	err = compare.CompareSets(dbPolicies, iPolicies, &added, &commondb, &commonext, &removed)
	if err != nil {
		return errors.Wrap(err, "compare.CompareSets")
	}

	result := compare.SyncResult{}
	for i := 0; i < len(removed); i++ {
		err = iUser.DetachSystemPolicy(removed[i].GetGlobalId())
		if err != nil {
			result.DeleteError(err)
			logclient.AddSimpleActionLog(self, logclient.ACT_DETACH_POLICY, err, userCred, false)
			continue
		}
		result.Delete()
	}

	result.UpdateCnt = len(commondb)

	for i := 0; i < len(added); i++ {
		err = iUser.AttachSystemPolicy(added[i].ExternalId)
		if err != nil {
			result.AddError(err)
			logclient.AddSimpleActionLog(self, logclient.ACT_ATTACH_POLICY, err, userCred, false)
			continue
		}
		result.Add()
	}

	if result.IsError() {
		return result.AllError()
	}

	log.Infof("Sync %s(%s) system policies for user %s result: %s", self.Name, self.Provider, clouduser.Name, result.Result())
	return nil
}
