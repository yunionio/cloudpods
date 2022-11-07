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

func (self *SObjectStoreProviderFactory) GetId() string {
	return api.CLOUD_PROVIDER_GENERICS3
}

func (self *SObjectStoreProviderFactory) GetName() string {
	return api.CLOUD_PROVIDER_GENERICS3
}

func (self *SObjectStoreProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
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

func (self *SObjectStoreProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func (self *SObjectStoreProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	client, err := objectstore.NewObjectStoreClient(
		objectstore.NewObjectStoreClientConfig(
			cfg.URL, cfg.Account, cfg.Secret,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return NewObjectStoreProvider(self, client, []string{
		string(cloudprovider.ACLPrivate),
	}), nil
}

func (self *SObjectStoreProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	return map[string]string{
		"S3_ACCESS_KEY": info.Account,
		"S3_SECRET":     info.Secret,
		"S3_ACCESS_URL": info.Url,
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

func (self *SObjectStoreProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SObjectStoreProvider) GetStorageClasses(regionId string) []string {
	return []string{}
}

func (self *SObjectStoreProvider) GetBucketCannedAcls(regionId string) []string {
	return self.supportedAcls
}

func (self *SObjectStoreProvider) GetObjectCannedAcls(regionId string) []string {
	return self.supportedAcls
}

func (self *SObjectStoreProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}
