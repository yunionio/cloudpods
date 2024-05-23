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
	"yunion.io/x/cloudmux/pkg/multicloud/ecloud"
)

type SEcloudProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactory
}

func (f *SEcloudProviderFactory) GetId() string {
	return ecloud.CLOUD_PROVIDER_ECLOUD
}

func (f *SEcloudProviderFactory) GetName() string {
	return ecloud.CLOUD_PROVIDER_ECLOUD_CN
}

func (f *SEcloudProviderFactory) IsSupportPrepaidResources() bool {
	return true
}

func (f *SEcloudProviderFactory) IsReadOnly() bool {
	return true
}

func (f *SEcloudProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
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

func (f *SEcloudProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func (f *SEcloudProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	segs := strings.Split(cfg.Account, "/")
	account := cfg.Account
	if len(segs) == 2 {
		account = segs[0]
	}

	client, err := ecloud.NewEcloudClient(
		ecloud.NewEcloudClientConfig(
			ecloud.NewRamRoleSigner(account, cfg.Secret),
		).SetCloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	err = client.TryConnect()
	if err != nil {
		return nil, err
	}
	return &SEcloudProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(f),
		client:        client,
	}, nil
}

func (f *SEcloudProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	return map[string]string{
		"ECLOUD_ACCESS_URL": info.Url,
		"ECLOUD_ACCESS_KEY": info.Account,
		"ECLOUD_SECRET":     info.Secret,
		"ECLOUD_REGION":     ecloud.ECLOUD_DEFAULT_REGION,
	}, nil
}

func init() {
	factory := SEcloudProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SEcloudProvider struct {
	cloudprovider.SBaseProvider
	client *ecloud.SEcloudClient
}

func (p *SEcloudProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return p.client.GetSubAccounts()
}

func (p *SEcloudProvider) GetAccountId() string {
	return p.client.GetAccountId()
}

func (p *SEcloudProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return p.client.GetIRegions()
}

func (p *SEcloudProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	iregions, _ := p.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(iregions))), "region_count")
	info.Add(jsonutils.NewString(ecloud.CLOUD_API_VERSION), "api_version")
	return info, nil
}

func (p *SEcloudProvider) GetVersion() string {
	return ecloud.CLOUD_API_VERSION
}

func (p *SEcloudProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	iregion, err := p.client.GetIRegionById(id)
	if err != nil {
		return nil, err
	}
	if iregion == nil {
		return nil, cloudprovider.ErrNotFound
	}
	return iregion, nil
}

func (p *SEcloudProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	return &cloudprovider.SBalanceInfo{
		Amount:   0.0,
		Currency: "CNY",
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}, cloudprovider.ErrNotSupported
}

func (p *SEcloudProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (p *SEcloudProvider) GetStorageClasses(regionId string) []string {
	// TODO
	return nil
}

func (p *SEcloudProvider) GetBucketCannedAcls(regionId string) []string {
	return nil
}

func (p *SEcloudProvider) GetObjectCannedAcls(regionId string) []string {
	return nil
}

func (p *SEcloudProvider) GetCloudRegionExternalIdPrefix() string {
	return p.client.GetCloudRegionExternalIdPrefix()
}

func (p *SEcloudProvider) GetCapabilities() []string {
	return p.client.GetCapabilities()
}

func (self *SEcloudProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return self.client.GetMetrics(opts)
}
