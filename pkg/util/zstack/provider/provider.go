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

package provider

import (
	"context"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

type SZStackProviderFactory struct {
	cloudprovider.SPrivateCloudBaseProviderFactor
}

func (self *SZStackProviderFactory) GetId() string {
	return zstack.CLOUD_PROVIDER_ZSTACK
}

func (self *SZStackProviderFactory) GetName() string {
	return zstack.CLOUD_PROVIDER_ZSTACK
}

func (self *SZStackProviderFactory) GetSupportedBrands() []string {
	return []string{api.ZSTACK_BRAND_DSTACK}
}

func (self *SZStackProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) error {
	username, _ := data.GetString("username")
	if len(username) == 0 {
		return httperrors.NewMissingParameterError("username")
	}
	password, _ := data.GetString("password")
	if len(password) == 0 {
		return httperrors.NewMissingParameterError("password")
	}
	authURL, _ := data.GetString("auth_url")
	if len(authURL) == 0 {
		return httperrors.NewMissingParameterError("auth_url")
	}
	data.Set("account", jsonutils.NewString(username))
	data.Set("secret", jsonutils.NewString(password))
	data.Set("access_url", jsonutils.NewString(authURL))
	return nil
}

func (self *SZStackProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, cloudaccount string) (*cloudprovider.SCloudaccount, error) {
	if username, _ := data.GetString("username"); len(username) > 0 {
		cloudaccount = username
	}
	password, _ := data.GetString("password")
	if len(password) == 0 {
		return nil, httperrors.NewMissingParameterError("password")
	}
	account := &cloudprovider.SCloudaccount{
		Account: cloudaccount,
		Secret:  password,
	}
	return account, nil
}

func (self *SZStackProviderFactory) GetProvider(providerId, providerName, url, username, password string) (cloudprovider.ICloudProvider, error) {
	client, err := zstack.NewZStackClient(providerId, providerName, url, username, password, false)
	if err != nil {
		return nil, err
	}
	return &SZStackProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SZStackProviderFactory) GetClientRC(url, account, secret string) (map[string]string, error) {
	return map[string]string{
		"ZSTACK_AUTH_URL":  url,
		"ZSTACK_USERNAME":  account,
		"ZSTACK_PASSWORD":  secret,
		"ZSTACK_REGION_ID": zstack.ZSTACK_DEFAULT_REGION,
	}, nil
}

func init() {
	factory := SZStackProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SZStackProvider struct {
	cloudprovider.SBaseProvider
	client *zstack.SZStackClient
}

func (self *SZStackProvider) GetVersion() string {
	return ""
}

func (self *SZStackProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return jsonutils.NewDict(), nil
}

func (self *SZStackProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SZStackProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SZStackProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SZStackProvider) GetBalance() (float64, string, error) {
	return 0.0, api.CLOUD_PROVIDER_HEALTH_UNKNOWN, cloudprovider.ErrNotSupported
}

func (self *SZStackProvider) GetCloudRegionExternalIdPrefix() string {
	return self.client.GetCloudRegionExternalIdPrefix()
}

func (self *SZStackProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}
