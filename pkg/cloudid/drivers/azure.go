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

package drivers

import (
	"context"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SAzureDriver struct {
	SAccountBaseProviderDriver
}

func (driver SAzureDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_AZURE
}

func init() {
	models.RegisterProviderDriver(&SAzureDriver{})
}

func (base SAzureDriver) RequestSyncCloudaccountResources(ctx context.Context, userCred mcclient.TokenCredential, account *models.SCloudaccount, provider cloudprovider.ICloudProvider) error {

	func() {
		lockman.LockRawObject(ctx, account.Id, models.SAMLProviderManager.Keyword())
		defer lockman.ReleaseRawObject(ctx, account.Id, models.SAMLProviderManager.Keyword())

		samls, err := provider.GetICloudSAMLProviders()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				log.Errorf("get saml providers for account %s error: %v", account.Name, err)
			}
			return
		}
		result := account.SyncSAMLProviders(ctx, userCred, samls, "")
		log.Infof("Sync SAMLProviders for account %s(%s) result: %s", account.Name, account.Provider, result.Result())
	}()

	func() {
		policies, err := provider.GetICloudpolicies()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				log.Errorf("get system policies for account %s error: %v", account.Name, err)
			}
			return
		}
		result := account.SyncPolicies(ctx, userCred, policies, "")
		log.Infof("Sync policies for account %s(%s) result: %s", account.Name, account.Provider, result.Result())
	}()

	func() {
		iGroups, err := provider.GetICloudgroups()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				log.Errorf("get groups for account %s error: %v", account.Name, err)
			}
			return
		}
		localGroups, remoteGroups, result := account.SyncCloudgroups(ctx, userCred, iGroups, "")
		log.Infof("SyncCloudgroups for account %s(%s) result: %s", account.Name, account.Provider, result.Result())
		for i := 0; i < len(localGroups); i += 1 {
			func() {
				// lock cloudgroup
				lockman.LockObject(ctx, &localGroups[i])
				defer lockman.ReleaseObject(ctx, &localGroups[i])

				localGroups[i].SyncCloudpolicies(ctx, userCred, remoteGroups[i])
			}()
		}
	}()

	func() {
		iUsers, err := provider.GetICloudusers()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				log.Errorf("get users for account %s error: %v", account.Name, err)
			}
			return
		}
		localUsers, remoteUsers, result := account.SyncCloudusers(ctx, userCred, iUsers, "")
		log.Infof("SyncCloudusers for account %s(%s) result: %s", account.Name, account.Provider, result.Result())
		for i := 0; i < len(localUsers); i += 1 {
			func() {
				// lock clouduser
				lockman.LockObject(ctx, &localUsers[i])
				defer lockman.ReleaseObject(ctx, &localUsers[i])

				localUsers[i].SyncCloudpolicies(ctx, userCred, remoteUsers[i])
				localUsers[i].SyncCloudgroups(ctx, userCred, remoteUsers[i])
			}()
		}
	}()

	return nil
}

func (base SAzureDriver) RequestSyncCloudproviderResources(ctx context.Context, userCred mcclient.TokenCredential, cp *models.SCloudprovider, provider cloudprovider.ICloudProvider) error {
	return nil
}
