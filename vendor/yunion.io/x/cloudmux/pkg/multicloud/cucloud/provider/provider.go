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
	"yunion.io/x/cloudmux/pkg/multicloud/cucloud"
)

type SChinaUnionProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactory
}

func (self *SChinaUnionProviderFactory) GetId() string {
	return api.CLOUD_PROVIDER_CUCLOUD
}

func (self *SChinaUnionProviderFactory) GetName() string {
	return cucloud.CLOUD_PROVIDER_CUCLOUD_CN
}

func (self *SChinaUnionProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
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

func (self *SChinaUnionProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func (self *SChinaUnionProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	client, err := cucloud.NewChinaUnionClient(
		cucloud.NewChinaUnionClientConfig(
			cfg.Account,
			cfg.Secret,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}

	return &SChinaUnionProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SChinaUnionProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	return map[string]string{
		"CUCLOUD_ACCESS_KEY_ID":     info.Account,
		"CUCLOUD_ACCESS_KEY_SECRET": info.Secret,
		"CUCLOUD_REGION_ID":         cucloud.CUCLOUD_DEFAULT_REGION,
	}, nil
}

func init() {
	factory := SChinaUnionProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SChinaUnionProvider struct {
	cloudprovider.SBaseProvider
	client *cucloud.SChinaUnionClient
}

func (self *SChinaUnionProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions, err := self.client.GetRegions()
	if err != nil {
		return nil, err
	}
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	return info, nil
}

func (self *SChinaUnionProvider) GetVersion() string {
	return ""
}

func (self *SChinaUnionProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SChinaUnionProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SChinaUnionProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	regions, err := self.client.GetRegions()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudRegion{}
	for i := range regions {
		ret = append(ret, &regions[i])
	}
	return ret, nil
}

func (self *SChinaUnionProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	region, err := self.client.GetRegion(extId)
	if err != nil {
		return nil, err
	}
	return region, nil
}

func (self *SChinaUnionProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	ret := &cloudprovider.SBalanceInfo{Currency: "CNY", Status: api.CLOUD_PROVIDER_HEALTH_UNKNOWN}
	balance, err := self.client.QueryBalance()
	if err != nil {
		return ret, err
	}
	ret.Status = api.CLOUD_PROVIDER_HEALTH_NORMAL
	if balance.CashBalance <= 0 {
		if balance.CashBalance < 0 {
			ret.Status = api.CLOUD_PROVIDER_HEALTH_ARREARS
		} else if balance.CashBalance < 100 {
			ret.Status = api.CLOUD_PROVIDER_HEALTH_INSUFFICIENT
		}
	}
	ret.Amount = balance.CashBalance
	return ret, nil
}

func (self *SChinaUnionProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return []cloudprovider.ICloudProject{}, nil
}

func (self *SChinaUnionProvider) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SChinaUnionProvider) GetStorageClasses(regionId string) []string {
	return []string{}
}

func (self *SChinaUnionProvider) GetBucketCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SChinaUnionProvider) GetObjectCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SChinaUnionProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SChinaUnionProvider) GetIamLoginUrl() string {
	return ""
}

func (self *SChinaUnionProvider) GetCloudRegionExternalIdPrefix() string {
	return api.CLOUD_PROVIDER_CUCLOUD + "/"
}

func (self *SChinaUnionProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return nil, cloudprovider.ErrNotImplemented
}
