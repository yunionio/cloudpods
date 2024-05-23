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
	"yunion.io/x/cloudmux/pkg/multicloud/ksyun"
)

type SKsyunProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactory
}

func (self *SKsyunProviderFactory) GetId() string {
	return api.CLOUD_PROVIDER_KSYUN
}

func (self *SKsyunProviderFactory) GetName() string {
	return ksyun.CLOUD_PROVIDER_KSYUN_CN
}

func (self *SKsyunProviderFactory) IsReadOnly() bool {
	return true
}

func (self *SKsyunProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
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

func (self *SKsyunProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func (self *SKsyunProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	client, err := ksyun.NewKsyunClient(
		ksyun.NewKsyunClientConfig(
			cfg.Account,
			cfg.Secret,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}

	return &SKsyunProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SKsyunProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	return map[string]string{
		"KSYUN_ACCESS_KEY_ID":     info.Account,
		"KSYUN_ACCESS_KEY_SECRET": info.Secret,
		"KSYUN_REGION_ID":         ksyun.KSYUN_DEFAULT_REGION,
	}, nil
}

func init() {
	factory := SKsyunProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SKsyunProvider struct {
	cloudprovider.SBaseProvider
	client *ksyun.SKsyunClient
}

func (self *SKsyunProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions, err := self.client.GetRegions()
	if err != nil {
		return nil, err
	}
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	return info, nil
}

func (self *SKsyunProvider) GetVersion() string {
	return ""
}

func (self *SKsyunProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SKsyunProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SKsyunProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
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

func (self *SKsyunProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	region, err := self.client.GetRegion(extId)
	if err != nil {
		return nil, err
	}
	return region, nil
}

func (self *SKsyunProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	ret := &cloudprovider.SBalanceInfo{Currency: "CNY", Status: api.CLOUD_PROVIDER_HEALTH_UNKNOWN}
	balance, err := self.client.QueryCashWalletAction()
	if err != nil {
		return ret, err
	}
	ret.Status = api.CLOUD_PROVIDER_HEALTH_NORMAL
	ret.Currency = balance.Currency
	if balance.AvailableAmount <= 0 {
		if balance.AvailableAmount < 0 {
			ret.Status = api.CLOUD_PROVIDER_HEALTH_ARREARS
		} else if balance.AvailableAmount < 100 {
			ret.Status = api.CLOUD_PROVIDER_HEALTH_INSUFFICIENT
		}
	}
	ret.Amount = balance.AvailableAmount
	return ret, nil
}

func (self *SKsyunProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SKsyunProvider) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SKsyunProvider) GetStorageClasses(regionId string) []string {
	return []string{}
}

func (self *SKsyunProvider) GetBucketCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SKsyunProvider) GetObjectCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SKsyunProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SKsyunProvider) GetIamLoginUrl() string {
	return ""
}

func (self *SKsyunProvider) GetCloudRegionExternalIdPrefix() string {
	return api.CLOUD_PROVIDER_KSYUN + "/"
}

func (self *SKsyunProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SKsyunProvider) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	return self.client.CreateIClouduser(conf)
}

func (self *SKsyunProvider) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	return self.client.GetICloudusers()
}

func (self *SKsyunProvider) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	return self.client.GetIClouduserByName(name)
}

func (self *SKsyunProvider) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	return self.client.GetICloudgroups()
}

func (self *SKsyunProvider) CreateICloudgroup(name, desc string) (cloudprovider.ICloudgroup, error) {
	return self.client.CreateICloudgroup(name, desc)
}

func (self *SKsyunProvider) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	return self.client.GetICloudgroupByName(name)
}

func (self *SKsyunProvider) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return self.client.GetICloudpolicies()
}
