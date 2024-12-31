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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/objectstore"
)

type SObjectStoreProviderFactory struct {
	cloudprovider.SPremiseBaseProviderFactory
}

func (factory *SObjectStoreProviderFactory) GetId() string {
	return api.CLOUD_PROVIDER_GENERICS3
}

func (factory *SObjectStoreProviderFactory) GetName() string {
	return api.CLOUD_PROVIDER_GENERICS3
}

func (factory *SObjectStoreProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_secret")
	}
	if len(input.Endpoint) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "endpoint")
	}
	output.Account = input.AccessKeyId
	output.Secret = input.AccessKeySecret
	output.AccessUrl = input.Endpoint
	return output, nil
}

func (factory *SObjectStoreProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_secret")
	}
	output = cloudprovider.SCloudaccount{
		Account: input.AccessKeyId,
		Secret:  input.AccessKeySecret,
	}
	return output, nil
}

func (factory *SObjectStoreProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	storeCfg := objectstore.NewObjectStoreClientConfig(
		cfg.URL, cfg.Account, cfg.Secret,
	).CloudproviderConfig(cfg)
	var signVer string
	if cfg.Options != nil {
		signVer, _ = cfg.Options.GetString("sign_ver")
	}
	if len(signVer) > 0 {
		storeCfg = storeCfg.SignVersion(objectstore.S3SignVersion(signVer))
	}
	client, err := objectstore.NewObjectStoreClient(storeCfg)
	if err != nil {
		return nil, err
	}
	return NewObjectStoreProvider(factory, client, []string{
		string(cloudprovider.ACLPrivate),
	}), nil
}

func (factory *SObjectStoreProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	var signVer string
	if info.Options != nil {
		signVer, _ = info.Options.GetString("sign_ver")
	}
	return map[string]string{
		"S3_ACCESS_KEY": info.Account,
		"S3_SECRET":     info.Secret,
		"S3_ACCESS_URL": info.Url,
		"S3_SIGN_VER":   signVer,
		"S3_BACKEND":    api.CLOUD_PROVIDER_GENERICS3,
	}, nil
}

func init() {
	factory := SObjectStoreProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SObjectStoreProvider struct {
	cloudprovider.SBaseProvider
	client        objectstore.IBucketProvider
	supportedAcls []string
}

func NewObjectStoreProvider(factory cloudprovider.ICloudProviderFactory, client objectstore.IBucketProvider, acls []string) *SObjectStoreProvider {
	return &SObjectStoreProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(factory),
		client:        client,
		supportedAcls: acls,
	}
}

func (provider *SObjectStoreProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (provider *SObjectStoreProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (provider *SObjectStoreProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	return &cloudprovider.SBalanceInfo{
		Amount:   0.0,
		Currency: "CNY",
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}, cloudprovider.ErrNotSupported
}

func (provider *SObjectStoreProvider) GetOnPremiseIRegion() (cloudprovider.ICloudRegion, error) {
	return provider.client, nil
}

func (provider *SObjectStoreProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (provider *SObjectStoreProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return provider.client.About(), nil
}

func (provider *SObjectStoreProvider) GetVersion() string {
	return provider.client.GetVersion()
}

func (provider *SObjectStoreProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return provider.client.GetSubAccounts()
}

func (provider *SObjectStoreProvider) GetAccountId() string {
	return provider.client.GetAccountId()
}

func (provider *SObjectStoreProvider) GetStorageClasses(regionId string) []string {
	return []string{}
}

func (provider *SObjectStoreProvider) GetBucketCannedAcls(regionId string) []string {
	return provider.supportedAcls
}

func (provider *SObjectStoreProvider) GetObjectCannedAcls(regionId string) []string {
	return provider.supportedAcls
}

func (provider *SObjectStoreProvider) GetCapabilities() []string {
	return provider.client.GetCapabilities()
}
