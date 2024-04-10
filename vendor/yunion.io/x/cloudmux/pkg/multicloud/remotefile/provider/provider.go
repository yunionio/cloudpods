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
	"yunion.io/x/cloudmux/pkg/multicloud/remotefile"
)

type SRemoteFileProviderFactory struct {
	cloudprovider.SPrivateCloudBaseProviderFactory
}

func (self *SRemoteFileProviderFactory) GetId() string {
	return remotefile.CLOUD_PROVIDER_REMOTEFILE
}

func (self *SRemoteFileProviderFactory) GetName() string {
	return remotefile.CLOUD_PROVIDER_REMOTEFILE
}

func (self *SRemoteFileProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AuthUrl) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "auth_url")
	}
	if len(input.Username) == 0 {
		input.Username = "root"
	}
	if len(input.Password) == 0 {
		input.Password = "password"
	}
	output.Account = input.Username
	output.Secret = input.Password
	output.AccessUrl = input.AuthUrl
	return output, nil
}

func (self *SRemoteFileProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	return output, cloudprovider.ErrNotSupported
}

func (self *SRemoteFileProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	client, err := remotefile.NewRemoteFileClient(
		remotefile.NewRemoteFileClientConfig(
			cfg.URL,
			cfg.Account,
			cfg.Secret,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SRemoteFileProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SRemoteFileProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	return map[string]string{
		"REMOTEFILE_AUTH_URL": info.Url,
	}, nil
}

func init() {
	factory := SRemoteFileProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SRemoteFileProvider struct {
	cloudprovider.SBaseProvider
	client *remotefile.SRemoteFileClient
}

func (self *SRemoteFileProvider) GetVersion() string {
	return ""
}

func (self *SRemoteFileProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return jsonutils.NewDict(), nil
}

func (self *SRemoteFileProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SRemoteFileProvider) GetAccountId() string {
	return ""
}

func (self *SRemoteFileProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegions()
}

func (self *SRemoteFileProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(id)
}

func (self *SRemoteFileProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	return &cloudprovider.SBalanceInfo{
		Amount:   0.0,
		Currency: "CNY",
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}, cloudprovider.ErrNotSupported
}

func (self *SRemoteFileProvider) GetCloudRegionExternalIdPrefix() string {
	return self.client.GetCloudRegionExternalIdPrefix()
}

func (self *SRemoteFileProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SRemoteFileProvider) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRemoteFileProvider) GetStorageClasses(regionId string) []string {
	return nil
}

func (self *SRemoteFileProvider) GetBucketCannedAcls(regionId string) []string {
	return nil
}

func (self *SRemoteFileProvider) GetObjectCannedAcls(regionId string) []string {
	return nil
}

func (self *SRemoteFileProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SRemoteFileProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return self.client.GetMetrics(opts)
}
