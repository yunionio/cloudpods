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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudid/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

// +onecloud:swagger-gen-ignore
type SCloudproviderManager struct {
	db.SStandaloneResourceBaseManager
}

var CloudproviderManager *SCloudproviderManager

func init() {
	CloudproviderManager = &SCloudproviderManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SCloudprovider{},
			"cloudproviders_tbl",
			"cloudprovider",
			"cloudproviders",
		),
	}
	CloudproviderManager.SetVirtualObject(CloudproviderManager)
}

type SCloudprovider struct {
	db.SStandaloneResourceBase

	Provider       string `width:"64" charset:"ascii" list:"domain"`
	CloudaccountId string `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (manager *SCloudproviderManager) newFromRegionProvider(ctx context.Context, userCred mcclient.TokenCredential, provider SCloudprovider) error {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	return manager.TableSpec().Insert(ctx, &provider)
}

func (self *SCloudprovider) syncWithRegionProvider(ctx context.Context, userCred mcclient.TokenCredential, provider SCloudprovider) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.Name = provider.Name
		return nil
	})
	return err
}

func (self SCloudprovider) GetGlobalId() string {
	return self.Id
}

func (self SCloudprovider) GetExternalId() string {
	return self.Id
}

func (manager *SCloudproviderManager) FetchProvider(ctx context.Context, id string) (*SCloudprovider, error) {
	provider, err := manager.FetchById(id)
	if err != nil {
		if err == sql.ErrNoRows {
			session := auth.GetAdminSession(context.Background(), options.Options.Region)
			result, err := modules.Cloudproviders.Get(session, id, nil)
			if err != nil {
				return nil, errors.Wrap(err, "Cloudproviders.Get")
			}
			_provider := &SCloudprovider{}
			_provider.SetModelManager(manager, _provider)
			err = result.Unmarshal(_provider)
			if err != nil {
				return nil, errors.Wrap(err, "result.Unmarshal")
			}

			lockman.LockRawObject(ctx, manager.KeywordPlural(), id)
			defer lockman.ReleaseRawObject(ctx, manager.KeywordPlural(), id)
			return _provider, manager.TableSpec().InsertOrUpdate(ctx, _provider)
		}
		return nil, errors.Wrap(err, "manager.FetchById")
	}
	return provider.(*SCloudprovider), nil
}

func (self *SCloudprovider) getCloudDelegate(ctx context.Context) (*SCloudDelegate, error) {
	s := auth.GetAdminSession(ctx, options.Options.Region)
	result, err := modules.Cloudproviders.Get(s, self.Id, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Cloudproviders.Get")
	}
	delegate := &SCloudDelegate{}
	err = result.Unmarshal(delegate)
	if err != nil {
		return nil, errors.Wrap(err, "result.Unmarshal")
	}
	return delegate, nil
}

func (self *SCloudprovider) GetProvider() (cloudprovider.ICloudProvider, error) {
	delegate, err := self.getCloudDelegate(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "getCloudproviderDelegate")
	}
	return delegate.GetProvider()
}

func (self *SCloudprovider) GetProviderFactory() (cloudprovider.ICloudProviderFactory, error) {
	return cloudprovider.GetProviderFactory(self.Provider)
}

func (self *SCloudprovider) SyncCustomCloudpoliciesForCloud(ctx context.Context, clouduser *SClouduser) error {
	account, err := self.GetCloudaccount()
	if err != nil {
		return errors.Wrap(err, "GetCloudaccount")
	}

	factory, err := self.GetProviderFactory()
	if err != nil {
		return errors.Wrapf(err, "GetProviderFactory")
	}

	policyIds := []string{}
	policies, err := clouduser.GetCustomCloudpolicies("")
	if err != nil {
		return errors.Wrap(err, "GetSystemCloudpolicies")
	}
	for i := range policies {
		err = account.getOrCacheCustomCloudpolicy(ctx, self.Id, policies[i].Id)
		if err != nil {
			return errors.Wrapf(err, "getOrCacheCloudpolicy %s(%s) for cloudprovider %s", policies[i].Name, policies[i].Provider, self.Name)
		}
		policyIds = append(policyIds, policies[i].Id)
	}

	if !factory.IsSupportCreateCloudgroup() {
		policies, err = clouduser.GetCustomCloudgroupPolicies()
		if err != nil {
			return errors.Wrap(err, "GetSystemCloudgroupPolicies")
		}
		for i := range policies {
			err = account.getOrCacheCustomCloudpolicy(ctx, self.Id, policies[i].Id)
			if err != nil {
				return errors.Wrapf(err, "getOrCacheCloudpolicy %s(%s) for cloudprovider %s", policies[i].Name, policies[i].Provider, self.Name)
			}
			policyIds = append(policyIds, policies[i].Id)
		}
	}

	dbCaches := []SCloudpolicycache{}
	if len(policyIds) > 0 {
		dbCaches, err = account.GetCloudpolicycaches(policyIds, self.Id)
		if err != nil {
			return errors.Wrapf(err, "GetCloudpolicycaches")
		}
	}

	provider, err := self.GetProvider()
	if err != nil {
		return errors.Wrapf(err, "GetProvider")
	}

	iUser, err := provider.GetIClouduserByName(clouduser.Name)
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
			continue
		}
		result.Delete()
	}

	result.UpdateCnt = len(commondb)

	for i := 0; i < len(added); i++ {
		err = iUser.AttachCustomPolicy(added[i].ExternalId)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	log.Infof("Sync %s(%s) custom policies for user %s result: %s", self.Name, self.Provider, clouduser.Name, result.Result())
	return nil
}

func (self *SCloudprovider) SyncSystemCloudpoliciesForCloud(ctx context.Context, clouduser *SClouduser) error {
	dbPolicies, err := clouduser.GetSystemCloudpolicies(self.Id)
	if err != nil {
		return errors.Wrap(err, "GetSystemCloudpolicies")
	}
	factory, err := self.GetProviderFactory()
	if err != nil {
		return errors.Wrap(err, "GetProviderFactory")
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
	provider, err := self.GetProvider()
	if err != nil {
		return errors.Wrapf(err, "GetProvider")
	}
	iUser, err := provider.GetIClouduserByName(clouduser.Name)
	if err != nil {
		return errors.Wrapf(err, "GetIClouduserByName(%s)", clouduser.Name)
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
			continue
		}
		result.Delete()
	}

	result.UpdateCnt = len(commondb)

	for i := 0; i < len(added); i++ {
		err = iUser.AttachSystemPolicy(added[i].ExternalId)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	log.Infof("Sync %s(%s) system policies for user %s in provider %s result: %s", self.Name, self.Provider, clouduser.Name, self.Name, result.Result())
	return nil
}

func (self *SCloudprovider) GetCloudaccount() (*SCloudaccount, error) {
	account, err := CloudaccountManager.FetchById(self.CloudaccountId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById(%s)", self.CloudaccountId)
	}
	return account.(*SCloudaccount), nil
}

func (self *SCloudprovider) SyncCustomCloudpoliciesFromCloud(ctx context.Context, userCred mcclient.TokenCredential) error {
	provider, err := self.GetProvider()
	if err != nil {
		return errors.Wrapf(err, "GetProvider")
	}

	account, err := self.GetCloudaccount()
	if err != nil {
		return errors.Wrapf(err, "GetCloudaccount")
	}

	policies, err := provider.GetICustomCloudpolicies()
	if err != nil {
		return errors.Wrapf(err, "GetICustomCloudpolicies for account %s(%s)", self.Name, self.Provider)
	}

	result := account.SyncCustomCloudpoliciesToLocal(ctx, userCred, policies, self.Id)
	log.Infof("Sync %s custom policies for cloudprovider %s result: %s", self.Provider, self.Name, result.Result())
	return nil
}
