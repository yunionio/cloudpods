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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IProviderDriver interface {
	GetProvider() string

	RequestSyncCloudaccountResources(ctx context.Context, userCred mcclient.TokenCredential, account *SCloudaccount, provider cloudprovider.ICloudProvider) error
	RequestSyncCloudproviderResources(ctx context.Context, userCred mcclient.TokenCredential, cp *SCloudprovider, provider cloudprovider.ICloudProvider) error
	ValidateCreateCloudgroup(ctx context.Context, userCred mcclient.TokenCredential, cp *SCloudprovider, input *api.CloudgroupCreateInput) (*api.CloudgroupCreateInput, error)
	RequestCreateCloudgroup(ctx context.Context, userCred mcclient.TokenCredential, cp *SCloudprovider, group *SCloudgroup) error

	ValidateCreateClouduser(ctx context.Context, userCred mcclient.TokenCredential, cp *SCloudprovider, input *api.ClouduserCreateInput) (*api.ClouduserCreateInput, error)
	RequestCreateClouduser(ctx context.Context, userCred mcclient.TokenCredential, cp *SCloudprovider, user *SClouduser) error

	RequestCreateSAMLProvider(ctx context.Context, userCred mcclient.TokenCredential, account *SCloudaccount) error
	RequestCreateRoleForSamlUser(ctx context.Context, userCred mcclient.TokenCredential, account *SCloudaccount, group *SCloudgroup, user *SSamluser) error
}

var providerDrivers map[string]IProviderDriver

func init() {
	providerDrivers = make(map[string]IProviderDriver)
}

func RegisterProviderDriver(driver IProviderDriver) {
	providerDrivers[driver.GetProvider()] = driver
}

func GetProviderDriver(provider string) (IProviderDriver, error) {
	driver, ok := providerDrivers[provider]
	if ok {
		return driver, nil
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "provider: [%s]", provider)
}
