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
	"yunion.io/x/cloudmux/pkg/multicloud/oceanbase"
)

type SOceanBaseProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactory
}

func (self *SOceanBaseProviderFactory) GetId() string {
	return api.CLOUD_PROVIDER_OCEANBASE
}

func (self *SOceanBaseProviderFactory) GetName() string {
	return api.CLOUD_PROVIDER_OCEANBASE
}

func (self *SOceanBaseProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_secret")
	}
	output.AccessUrl = input.Environment
	output.Account = input.AccessKeyId
	output.Secret = input.AccessKeySecret
	return output, nil
}

func (self *SOceanBaseProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func (self *SOceanBaseProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	client, err := oceanbase.NewOceanBaseClient(
		oceanbase.NewOceanBaseClientConfig(
			cfg.Account,
			cfg.Secret,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}

	return &SOceanBaseProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SOceanBaseProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	return map[string]string{
		"OCEANBASE_ACCESS_KEY_ID":     info.Account,
		"OCEANBASE_ACCESS_KEY_SECRET": info.Secret,
	}, nil
}

func init() {
	factory := SOceanBaseProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SOceanBaseProvider struct {
	cloudprovider.SBaseProvider
	client *oceanbase.SOceanBaseClient
}

func (self *SOceanBaseProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return jsonutils.NewDict(), nil
}

func (self *SOceanBaseProvider) GetVersion() string {
	return ""
}

func (self *SOceanBaseProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SOceanBaseProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SOceanBaseProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return []cloudprovider.ICloudRegion{
		self.client.GetRegion(),
	}, nil
}

func (self *SOceanBaseProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	regions, err := self.GetIRegions()
	if err != nil {
		return nil, err
	}
	for i := range regions {
		if regions[i].GetGlobalId() == extId {
			return regions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SOceanBaseProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	return &cloudprovider.SBalanceInfo{
		Currency: "CNY",
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}, cloudprovider.ErrNotSupported
}

func (self *SOceanBaseProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return []cloudprovider.ICloudProject{}, nil
}

func (self *SOceanBaseProvider) GetStorageClasses(regionId string) []string {
	return []string{}
}

func (self *SOceanBaseProvider) GetBucketCannedAcls(regionId string) []string {
	return []string{}
}

func (self *SOceanBaseProvider) GetObjectCannedAcls(regionId string) []string {
	return []string{}
}

func (self *SOceanBaseProvider) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SOceanBaseProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SOceanBaseProvider) GetIamLoginUrl() string {
	return ""
}

func (self *SOceanBaseProvider) GetCloudRegionExternalIdPrefix() string {
	return api.CLOUD_PROVIDER_OCEANBASE + "/"
}

func (self *SOceanBaseProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return nil, cloudprovider.ErrNotImplemented
}
