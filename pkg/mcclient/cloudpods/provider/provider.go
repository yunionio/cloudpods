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

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient/cloudpods"
)

type SCloudpodsProviderFactory struct {
	cloudprovider.SPrivateCloudBaseProviderFactory
}

func (self *SCloudpodsProviderFactory) GetId() string {
	return cloudpods.CLOUD_PROVIDER_CLOUDPODS
}

func (self *SCloudpodsProviderFactory) GetName() string {
	return cloudpods.CLOUD_PROVIDER_CLOUDPODS
}

func (self *SCloudpodsProviderFactory) IsNeedForceAutoCreateProject() bool {
	return true
}

func (self *SCloudpodsProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	ret := cloudprovider.SCloudaccount{}
	if len(input.AuthUrl) == 0 {
		return ret, errors.Wrapf(cloudprovider.ErrMissingParameter, "auth_url")
	}
	ret.AccessUrl = input.AuthUrl
	if len(input.AccessKeyId) == 0 {
		return ret, errors.Wrapf(cloudprovider.ErrMissingParameter, "access_key_id")
	}
	ret.Account = input.AccessKeyId
	if len(input.AccessKeySecret) == 0 {
		return ret, errors.Wrapf(cloudprovider.ErrMissingParameter, "access_key_secret")
	}
	ret.Secret = input.AccessKeySecret
	return ret, nil
}

func (self *SCloudpodsProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
	ret := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return ret, errors.Wrapf(cloudprovider.ErrMissingParameter, "access_key_id")
	}
	ret.Account = input.AccessKeyId
	if len(input.AccessKeySecret) == 0 {
		return ret, errors.Wrapf(cloudprovider.ErrMissingParameter, "access_key_secret")
	}
	ret.Secret = input.AccessKeySecret
	return ret, nil
}

func (self *SCloudpodsProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	client, err := cloudpods.NewCloudpodsClient(
		cloudpods.NewCloudpodsClientConfig(
			cfg.URL,
			cfg.Account,
			cfg.Secret,
		).Debug(cfg.Debug).
			CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SCloudpodsProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SCloudpodsProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	return map[string]string{
		"CLOUDPODS_AUTH_URL":      info.Url,
		"CLOUDPODS_ACCESS_KEY":    info.Account,
		"CLOUDPODS_ACCESS_SECRET": info.Secret,
	}, nil
}

func init() {
	factory := SCloudpodsProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SCloudpodsProvider struct {
	cloudprovider.SBaseProvider
	client *cloudpods.SCloudpodsClient
}

func (self *SCloudpodsProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SCloudpodsProvider) GetCloudRegionExternalIdPrefix() string {
	return self.client.GetCloudRegionExternalIdPrefix()
}

func (self *SCloudpodsProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	return &cloudprovider.SBalanceInfo{
		Currency: "CNY",
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}, cloudprovider.ErrNotSupported
}

func (self *SCloudpodsProvider) GetBucketCannedAcls(regionId string) []string {
	return nil
}

func (self *SCloudpodsProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SCloudpodsProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SCloudpodsProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SCloudpodsProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegions()
}

func (self *SCloudpodsProvider) GetObjectCannedAcls(regionId string) []string {
	return nil
}

func (self *SCloudpodsProvider) GetStorageClasses(regionId string) []string {
	return nil
}

func (self *SCloudpodsProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SCloudpodsProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return jsonutils.NewDict(), nil
}

func (self *SCloudpodsProvider) GetVersion() string {
	return self.client.GetVersion()
}

func (provider *SCloudpodsProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return provider.client.GetMetrics(opts)
}
