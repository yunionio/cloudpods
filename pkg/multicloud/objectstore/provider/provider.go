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
	"yunion.io/x/onecloud/pkg/multicloud/objectstore"
)

type SObjectStoreProviderFactory struct {
	cloudprovider.SPremiseBaseProviderFactory
}

func (self *SObjectStoreProviderFactory) GetId() string {
	return api.CLOUD_PROVIDER_GENERICS3
}

func (self *SObjectStoreProviderFactory) GetName() string {
	return api.CLOUD_PROVIDER_GENERICS3
}

func (self *SObjectStoreProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) error {
	accessKeyID, _ := data.GetString("access_key")
	if len(accessKeyID) == 0 {
		return httperrors.NewMissingParameterError("access_key")
	}
	accessKeySecret, _ := data.GetString("secret_key")
	if len(accessKeySecret) == 0 {
		return httperrors.NewMissingParameterError("secret_key")
	}
	endpointURL, _ := data.GetString("endpoint")
	if len(endpointURL) == 0 {
		return httperrors.NewMissingParameterError("endpoint")
	}
	data.Set("account", jsonutils.NewString(accessKeyID))
	data.Set("secret", jsonutils.NewString(accessKeySecret))
	data.Set("url", jsonutils.NewString(endpointURL))
	return nil
}

func (self *SObjectStoreProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, cloudaccount string) (*cloudprovider.SCloudaccount, error) {
	accessKeyID, _ := data.GetString("access_key")
	if len(accessKeyID) == 0 {
		return nil, httperrors.NewMissingParameterError("access_key")
	}
	accessKeySecret, _ := data.GetString("secret_key")
	if len(accessKeySecret) == 0 {
		return nil, httperrors.NewMissingParameterError("secret_key")
	}
	account := &cloudprovider.SCloudaccount{
		Account: accessKeyID,
		Secret:  accessKeySecret,
	}
	return account, nil
}

func (self *SObjectStoreProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	client, err := objectstore.NewObjectStoreClient(providerId, providerName, url, account, secret, false)
	if err != nil {
		return nil, err
	}
	client.SetVirtualObject(client)
	return &SObjectStoreProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SObjectStoreProviderFactory) GetClientRC(url, account, secret string) (map[string]string, error) {
	return map[string]string{
		"OBJECTSTORE_ACCESSKEY": account,
		"OBJECTSTORE_SECRET":    secret,
		"OBJECTSTORE_ENDPOINT":  url,
	}, nil
}

func init() {
	factory := SObjectStoreProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SObjectStoreProvider struct {
	cloudprovider.SBaseProvider
	client *objectstore.SObjectStoreClient
}

func (self *SObjectStoreProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return nil
}

func (self *SObjectStoreProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SObjectStoreProvider) GetBalance() (float64, string, error) {
	return 0.0, api.CLOUD_PROVIDER_HEALTH_NORMAL, cloudprovider.ErrNotSupported
}

func (self *SObjectStoreProvider) GetOnPremiseIRegion() (cloudprovider.ICloudRegion, error) {
	return self.client, nil
}

func (self *SObjectStoreProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SObjectStoreProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return self.client.About(), nil
}

func (self *SObjectStoreProvider) GetVersion() string {
	return self.client.GetVersion()
}

func (self *SObjectStoreProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}
