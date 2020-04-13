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

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudid/options"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
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

func (self *SCloudprovider) syncRemoveClouduser(ctx context.Context, userCred mcclient.TokenCredential) error {
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
	return self.Delete(ctx, userCred)
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

func (self SCloudprovider) GetCloudproviderId() string {
	return self.Id
}

func (manager *SCloudproviderManager) FetchProvider(ctx context.Context, id string) (*SCloudprovider, error) {
	provider, err := manager.FetchById(id)
	if err != nil {
		if err == sql.ErrNoRows {
			session := auth.GetAdminSession(context.Background(), options.Options.Region, "")
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
	s := auth.GetAdminSession(ctx, options.Options.Region, "")
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

func (self *SCloudprovider) getCloudusers() ([]SClouduser, error) {
	users := []SClouduser{}
	q := ClouduserManager.Query().Equals("cloudprovider_id", self.Id)
	err := db.FetchModelObjects(ClouduserManager, q, &users)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return users, nil
}

func (self *SCloudprovider) getAvailableUsers(cloudproviderId string) ([]SClouduser, error) {
	users := []SClouduser{}
	q := ClouduserManager.Query().Equals("status", api.CLOUD_USER_STATUS_AVAILABLE).Equals("cloudprovider_id", self.Id)
	err := db.FetchModelObjects(ClouduserManager, q, &users)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return users, nil
}
