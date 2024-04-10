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
	"yunion.io/x/cloudmux/pkg/multicloud/qingcloud"
)

type SQingCloudProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactory
}

func (self *SQingCloudProviderFactory) GetId() string {
	return api.CLOUD_PROVIDER_QINGCLOUD
}

func (self *SQingCloudProviderFactory) GetName() string {
	return qingcloud.CLOUD_PROVIDER_QINGCLOUD_CN
}

func (self *SQingCloudProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
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

func (self *SQingCloudProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func (self *SQingCloudProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	client, err := qingcloud.NewQingCloudClient(
		qingcloud.NewQingCloudClientConfig(
			cfg.Account,
			cfg.Secret,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}

	return &SQingCloudProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SQingCloudProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	return map[string]string{
		"QINGCLOUD_ACCESS_KEY_ID":     info.Account,
		"QINGCLOUD_ACCESS_KEY_SECRET": info.Secret,
		"QINGCLOUD_REGION_ID":         qingcloud.QINGCLOUD_DEFAULT_REGION,
	}, nil
}

func init() {
	factory := SQingCloudProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SQingCloudProvider struct {
	cloudprovider.SBaseProvider
	client *qingcloud.SQingCloudClient
}

func (self *SQingCloudProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	return info, nil
}

func (self *SQingCloudProvider) GetVersion() string {
	return ""
}

func (self *SQingCloudProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SQingCloudProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SQingCloudProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	regions := self.client.GetRegions()
	ret := []cloudprovider.ICloudRegion{}
	for i := range regions {
		ret = append(ret, &regions[i])
	}
	return ret, nil
}

func (self *SQingCloudProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	region, err := self.client.GetRegion(extId)
	if err != nil {
		return nil, err
	}
	return region, nil
}

func (self *SQingCloudProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	ret := &cloudprovider.SBalanceInfo{Currency: "CNY", Status: api.CLOUD_PROVIDER_HEALTH_UNKNOWN}
	balance, err := self.client.QueryBalance()
	if err != nil {
		return ret, err
	}
	ret.Status = api.CLOUD_PROVIDER_HEALTH_NORMAL
	if balance.Balance <= 0 {
		if balance.Balance < 0 {
			ret.Status = api.CLOUD_PROVIDER_HEALTH_ARREARS
		}
	}
	ret.Amount = balance.Balance
	return ret, nil
}

func (self *SQingCloudProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return []cloudprovider.ICloudProject{}, nil
}

func (self *SQingCloudProvider) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SQingCloudProvider) GetStorageClasses(regionId string) []string {
	return []string{}
}

func (self *SQingCloudProvider) GetBucketCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SQingCloudProvider) GetObjectCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SQingCloudProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SQingCloudProvider) GetIamLoginUrl() string {
	return ""
}

func (self *SQingCloudProvider) GetCloudRegionExternalIdPrefix() string {
	return api.CLOUD_PROVIDER_QINGCLOUD + "/"
}

func (self *SQingCloudProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return nil, cloudprovider.ErrNotImplemented
}
