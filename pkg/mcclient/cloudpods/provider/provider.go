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

func (factory *SCloudpodsProviderFactory) GetId() string {
	return cloudpods.CLOUD_PROVIDER_CLOUDPODS
}

func (factory *SCloudpodsProviderFactory) GetName() string {
	return cloudpods.CLOUD_PROVIDER_CLOUDPODS
}

func (factory *SCloudpodsProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
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

func (factory *SCloudpodsProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func (factory *SCloudpodsProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
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
		SBaseProvider: cloudprovider.NewBaseProvider(factory),
		client:        client,
	}, nil
}

func (factory *SCloudpodsProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
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

func (provider *SCloudpodsProvider) GetAccountId() string {
	return provider.client.GetAccountId()
}

func (provider *SCloudpodsProvider) GetCloudRegionExternalIdPrefix() string {
	return provider.client.GetCloudRegionExternalIdPrefix()
}

func (provider *SCloudpodsProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	return &cloudprovider.SBalanceInfo{
		Currency: "CNY",
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}, cloudprovider.ErrNotSupported
}

func (provider *SCloudpodsProvider) GetBucketCannedAcls(regionId string) []string {
	return nil
}

func (provider *SCloudpodsProvider) GetCapabilities() []string {
	return provider.client.GetCapabilities()
}

func (provider *SCloudpodsProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return provider.client.GetIProjects()
}

func (provider *SCloudpodsProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return provider.client.GetIRegionById(extId)
}

func (provider *SCloudpodsProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return provider.client.GetIRegions()
}

func (provider *SCloudpodsProvider) GetObjectCannedAcls(regionId string) []string {
	return nil
}

func (provider *SCloudpodsProvider) GetStorageClasses(regionId string) []string {
	return nil
}

func (provider *SCloudpodsProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return provider.client.GetSubAccounts()
}

func (provider *SCloudpodsProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return jsonutils.NewDict(), nil
}

func (provider *SCloudpodsProvider) GetVersion() string {
	return provider.client.GetVersion()
}

func (provider *SCloudpodsProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return provider.client.GetMetrics(opts)
}
