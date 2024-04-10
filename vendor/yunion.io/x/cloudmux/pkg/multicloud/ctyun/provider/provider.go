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
	"yunion.io/x/cloudmux/pkg/multicloud/ctyun"
)

type SCtyunProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactory
}

func (self *SCtyunProviderFactory) GetId() string {
	return ctyun.CLOUD_PROVIDER_CTYUN
}

func (self *SCtyunProviderFactory) GetName() string {
	return ctyun.CLOUD_PROVIDER_CTYUN_CN
}

func (self *SCtyunProviderFactory) IsSupportPrepaidResources() bool {
	return true
}

func (self *SCtyunProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_secret")
	}
	output.Account = input.AccessKeyId
	output.Secret = input.AccessKeySecret
	return output, nil
}

func (self *SCtyunProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func (self *SCtyunProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	client, err := ctyun.NewSCtyunClient(
		ctyun.NewSCtyunClientConfig(
			cfg.Account, cfg.Secret,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SCtyunProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SCtyunProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	ret := map[string]string{
		"CTYUN_ACCESS_KEY": info.Account,
		"CTYUN_SECRET":     info.Secret,
		"CTYUN_REGION":     "cn-beijing-5",
	}
	return ret, nil
}

func init() {
	factory := SCtyunProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SCtyunProvider struct {
	cloudprovider.SBaseProvider
	client *ctyun.SCtyunClient
}

func (self *SCtyunProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SCtyunProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SCtyunProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegions()
}

func (self *SCtyunProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions, _ := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	return info, nil
}

func (self *SCtyunProvider) GetVersion() string {
	return ""
}

func (self *SCtyunProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(id)
}

func (self *SCtyunProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	return &cloudprovider.SBalanceInfo{
		Amount:   0.0,
		Currency: "CNY",
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}, nil
}

func (self *SCtyunProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SCtyunProvider) GetStorageClasses(regionId string) []string {
	return []string{
		"STANDARD", "WARM", "COLD",
	}
}

func (self *SCtyunProvider) GetBucketCannedAcls(regionId string) []string {
	return nil
}

func (self *SCtyunProvider) GetObjectCannedAcls(regionId string) []string {
	return nil
}

func (self *SCtyunProvider) GetCloudRegionExternalIdPrefix() string {
	return self.client.GetCloudRegionExternalIdPrefix()
}

func (self *SCtyunProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SCtyunProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return self.client.GetMetrics(opts)
}
