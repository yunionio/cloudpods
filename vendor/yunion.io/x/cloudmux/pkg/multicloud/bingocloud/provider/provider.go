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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/bingocloud"
)

type SBingoCloudProviderFactory struct {
	cloudprovider.SPrivateCloudBaseProviderFactory
}

func (self *SBingoCloudProviderFactory) GetId() string {
	return bingocloud.CLOUD_PROVIDER_BINGO_CLOUD
}

func (self *SBingoCloudProviderFactory) GetName() string {
	return bingocloud.CLOUD_PROVIDER_BINGO_CLOUD
}

func (self *SBingoCloudProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return fmt.Errorf("Changing %s bandwidth is not supported", bingocloud.CLOUD_PROVIDER_BINGO_CLOUD)
}

func (self *SBingoCloudProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
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
	output.AccessUrl = input.Endpoint
	output.Account = input.AccessKeyId
	output.Secret = input.AccessKeySecret
	return output, nil
}

func (self *SBingoCloudProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func (self *SBingoCloudProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	client, err := bingocloud.NewBingoCloudClient(
		bingocloud.NewBingoCloudClientConfig(
			cfg.URL, cfg.Account, cfg.Secret,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SBingoCloudProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SBingoCloudProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	return map[string]string{
		"BINGO_CLOUD_ENDPOINT":   info.Url,
		"BINGO_CLOUD_ACCESS_KEY": info.Account,
		"BINGO_CLOUD_SECRET_KEY": info.Secret,
	}, nil
}

func init() {
	factory := SBingoCloudProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SBingoCloudProvider struct {
	cloudprovider.SBaseProvider
	client *bingocloud.SBingoCloudClient
}

func (self *SBingoCloudProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return jsonutils.NewDict(), nil
}

func (self *SBingoCloudProvider) GetVersion() string {
	return "2009-08-15"
}

func (self *SBingoCloudProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	self.GetAccountId()
	return self.client.GetSubAccounts()
}

func (self *SBingoCloudProvider) CreateSubscription(cloudprovider.SubscriptionCreateInput) error {
	return nil
}

func (self *SBingoCloudProvider) GetEnrollmentAccounts() ([]cloudprovider.SEnrollmentAccount, error) {
	return self.client.GetEnrollmentAccounts()
}

func (self *SBingoCloudProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SBingoCloudProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegions()
}

func (self *SBingoCloudProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(id)
}

func (self *SBingoCloudProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	return &cloudprovider.SBalanceInfo{
		Amount:   0.0,
		Currency: "CNY",
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}, cloudprovider.ErrNotSupported
}

func (self *SBingoCloudProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return []cloudprovider.ICloudProject{}, nil
}

func (self *SBingoCloudProvider) GetStorageClasses(regionId string) []string {
	return nil
}

func (self *SBingoCloudProvider) GetBucketCannedAcls(regionId string) []string {
	return nil
}

func (self *SBingoCloudProvider) GetObjectCannedAcls(regionId string) []string {
	return nil
}

func (self *SBingoCloudProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SBingoCloudProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return self.client.GetMetrics(opts)
}
