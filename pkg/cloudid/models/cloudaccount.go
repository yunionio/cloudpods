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
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"

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
)

// +onecloud:swagger-gen-ignore
type SCloudaccountManager struct {
	db.SDomainLevelResourceBaseManager
}

var CloudaccountManager *SCloudaccountManager
var isCloudacountSynced bool

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
}

type SCloudaccount struct {
	db.SStandaloneResourceBase
	db.SDomainizedResourceBase

	Provider    string `width:"64" charset:"ascii" list:"domain"`
	Brand       string `width:"64" charset:"utf8" nullable:"true" list:"domain"`
	IamLoginUrl string `width:"512" charset:"ascii"`
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
			result.AddError(err)
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

		factory, err := account.GetProviderFactory()
		if err != nil {
			log.Errorf("GetProviderFactory: %v", err)
			continue
		}
		if !factory.IsClouduserBelongCloudprovider() {
			continue
		}
		result = account.syncCloudprovider(ctx, userCred)
		log.Infof("sync cloudprovider for cloudaccount %s(%s) result: %s", account.Name, account.Id, result.Result())
	}
	isCloudacountSynced = true
}

func waitForSync() {
	for !isCloudacountSynced {
		log.Infof("cloudaccount not sync, wait for 10 seconds")
		time.Sleep(time.Second * 10)
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
		return nil, errors.Errorf("Cloud account %s is not enabled", account.Name)
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

func (self *SCloudaccount) GetCloudusersByProviderId(cloudproviderId string) ([]SClouduser, error) {
	users := []SClouduser{}
	q := ClouduserManager.Query().Equals("status", api.CLOUD_USER_STATUS_AVAILABLE).Equals("cloudaccount_id", self.Id)
	if len(cloudproviderId) > 0 {
		q = q.Equals("cloudprovider_id", cloudproviderId)
	}
	err := db.FetchModelObjects(ClouduserManager, q, &users)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return users, nil
}

func (self *SCloudaccount) SyncCloudusers(ctx context.Context, userCred mcclient.TokenCredential, cloudproviderId string, iUsers []cloudprovider.IClouduser) ([]SClouduser, []cloudprovider.IClouduser, compare.SyncResult) {
	result := compare.SyncResult{}
	dbUsers, err := self.GetCloudusersByProviderId(cloudproviderId)
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
				result.AddError(err)
				continue
			}
			result.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithClouduser(ctx, userCred, commonext[i], cloudproviderId)
		if err != nil {
			result.UpdateError(err)
			continue
		}
		localUsers = append(localUsers, commondb[i])
		remoteUsers = append(remoteUsers, commonext[i])
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		user, err := ClouduserManager.newFromClouduser(ctx, userCred, added[i], self.Id, cloudproviderId)
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

func (self *SCloudaccount) GetCloudpolicies() ([]SCloudpolicy, error) {
	q := CloudpolicyManager.Query().Equals("provider", self.Provider)
	policies := []SCloudpolicy{}
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SCloudaccount) SyncCloudpolicies(ctx context.Context, userCred mcclient.TokenCredential, iPolicies []cloudprovider.ICloudpolicy) compare.SyncResult {
	result := compare.SyncResult{}
	dbPolicies, err := self.GetCloudpolicies()
	if err != nil {
		result.Error(errors.Wrap(err, "GetCloudpolicies"))
		return result
	}

	removed := make([]SCloudpolicy, 0)
	commondb := make([]SCloudpolicy, 0)
	commonext := make([]cloudprovider.ICloudpolicy, 0)
	added := make([]cloudprovider.ICloudpolicy, 0)

	err = compare.CompareSets(dbPolicies, iPolicies, &removed, &commondb, &commonext, &added)
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
		err = commondb[i].SyncWithCloudpolicy(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		_, err := CloudpolicyManager.newFromCloudpolicy(ctx, userCred, added[i], self.Provider)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
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
		err = removed[i].syncRemoveClouduser(ctx, userCred)
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

func (manager *SCloudaccountManager) SyncCloudidResources(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	waitForSync()
	accounts, err := manager.GetCloudaccounts()
	if err != nil {
		log.Errorf("GetCloudaccounts error: %v", err)
		return
	}
	for i := range accounts {
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

func (self *SCloudaccount) SyncCloudgroupcaches(ctx context.Context, userCred mcclient.TokenCredential, iGroups []cloudprovider.ICloudgroup) compare.SyncResult {
	result := compare.SyncResult{}

	dbCaches, err := self.GetCloudgroupcaches()
	if err != nil {
		result.Error(errors.Wrap(err, "GetCloudgroupcaches"))
		return result
	}

	removed := make([]SCloudgroupcache, 0)
	commondb := make([]SCloudgroupcache, 0)
	commonext := make([]cloudprovider.ICloudgroup, 0)
	added := make([]cloudprovider.ICloudgroup, 0)

	err = compare.CompareSets(dbCaches, iGroups, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrap(err, "compare.CompareSets"))
		return result
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
		err = commondb[i].syncWithCloudgrup(ctx, userCred, commonext[i])
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
	return result
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

func (self *SCloudaccount) GetOrCreateCloudgroup(ctx context.Context, userCred mcclient.TokenCredential, iGroup cloudprovider.ICloudgroup) (*SCloudgroup, error) {
	groups, err := self.GetCloudgroups()
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudgroups")
	}
	iPolicies, err := iGroup.GetISystemCloudpolicies()
	if err != nil {
		return nil, errors.Wrap(err, "GetICloudpolicies")
	}

	for i := range iPolicies {
		_, err := db.FetchByExternalId(CloudpolicyManager, iPolicies[i].GetGlobalId())
		if err == nil {
			continue
		}
		if errors.Cause(err) != sql.ErrNoRows {
			return nil, errors.Wrapf(err, "db.FetchByExternalId(%s)", iPolicies[i].GetGlobalId())
		}
		_, err = CloudpolicyManager.newFromCloudpolicy(ctx, userCred, iPolicies[i], self.Provider)
		if err != nil {
			return nil, errors.Wrap(err, "newFromCloudpolicy")
		}
	}
	for i := range groups {
		isEqual, err := groups[i].IsEqual(iPolicies)
		if err != nil {
			return nil, errors.Wrap(err, "IsEqual")
		}
		if isEqual {
			return &groups[i], nil
		}
	}
	group, err := CloudgroupManager.newCloudgroup(ctx, userCred, iGroup, self.Provider, self.DomainId)
	if err != nil {
		return nil, errors.Wrap(err, "newCloudgroup")
	}
	for i := range iPolicies {
		err = group.attachPolicyFromCloudpolicy(ctx, userCred, iPolicies[i])
		if err != nil {
			return nil, errors.Wrap(err, "attachPolicyFromCloudpolicy")
		}
	}
	return group, nil
}
