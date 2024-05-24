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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/jdcloud"
)

type SJdcloudProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactory
}

func (f *SJdcloudProviderFactory) GetId() string {
	return jdcloud.CLOUD_PROVIDER_JDCLOUD
}

func (f *SJdcloudProviderFactory) GetName() string {
	return jdcloud.CLOUD_PROVIDER_JDCLOUD_CN
}

func (f *SJdcloudProviderFactory) IsSupportPrepaidResources() bool {
	return true
}

func (f *SJdcloudProviderFactory) IsReadOnly() bool {
	return true
}

func (f *SJdcloudProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
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

func (f *SJdcloudProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func (f *SJdcloudProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	segs := strings.Split(cfg.Account, "/")
	account := cfg.Account
	if len(segs) == 2 {
		account = segs[0]
	}
	client, err := jdcloud.NewJDCloudClient(
		jdcloud.NewJDCloudClientConfig(
			account,
			cfg.Secret,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}

	return &SJdcloudProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(f),
		client:        client,
	}, nil
}

func (f *SJdcloudProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	return map[string]string{
		"JDCLOUD_ACCESS_KEY": info.Account,
		"JDCLOUD_SECRET":     info.Secret,
		"JDCLOUD_REGION":     jdcloud.JDCLOUD_DEFAULT_REGION,
	}, nil
}

func init() {
	factory := SJdcloudProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SJdcloudProvider struct {
	cloudprovider.SBaseProvider

	client *jdcloud.SJDCloudClient
}

func (p *SJdcloudProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return p.client.GetSubAccounts()
}

func (p *SJdcloudProvider) GetAccountId() string {
	return p.client.GetAccountId()
}

func (p *SJdcloudProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return p.client.GetIRegions()
}

func (p *SJdcloudProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	iregions, _ := p.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(iregions))), "region_count")
	return info, nil
}

func (p *SJdcloudProvider) GetVersion() string {
	return ""
}

func (p *SJdcloudProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	iregions, err := p.GetIRegions()
	if err != nil {
		return nil, err
	}
	for i := range iregions {
		if iregions[i].GetGlobalId() == id {
			return iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (p *SJdcloudProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	ret := &cloudprovider.SBalanceInfo{
		Amount:   0.0,
		Currency: "CNY",
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}
	balance, err := p.client.DescribeAccountAmount()
	if err != nil {
		ret.Status = api.CLOUD_PROVIDER_HEALTH_NO_PERMISSION
		return ret, errors.Wrap(err, "DescribeAccountAmount")
	}
	ret.Amount, _ = jsonutils.Marshal(balance).Float("totalAmount")
	if ret.Amount < 0 {
		ret.Status = api.CLOUD_PROVIDER_HEALTH_ARREARS
	} else if ret.Amount < 50 {
		ret.Status = api.CLOUD_PROVIDER_HEALTH_INSUFFICIENT
	}
	return ret, nil
}

func (p *SJdcloudProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (p *SJdcloudProvider) GetStorageClasses(regionId string) []string {
	// TODO
	return nil
}

func (p *SJdcloudProvider) GetBucketCannedAcls(regionId string) []string {
	return nil
}

func (p *SJdcloudProvider) GetObjectCannedAcls(regionId string) []string {
	return nil
}

func (p *SJdcloudProvider) GetCloudRegionExternalIdPrefix() string {
	return api.CLOUD_PROVIDER_JDCLOUD
}

func (p *SJdcloudProvider) GetCapabilities() []string {
	iRegions, _ := p.GetIRegions()
	if len(iRegions) > 0 {
		return iRegions[0].GetCapabilities()
	}
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_NETWORK + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_EIP + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_RDS + cloudprovider.READ_ONLY_SUFFIX,
	}
	return caps
}

func (self *SJdcloudProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return self.client.GetMetrics(opts)
}
