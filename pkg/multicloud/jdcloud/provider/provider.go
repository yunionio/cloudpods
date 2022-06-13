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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/multicloud/jdcloud"
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

func (f *SJdcloudProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_secret")
	}
	output.Account = input.AccessKeyId
	output.Secret = input.AccessKeySecret
	return output, nil
}

func (f *SJdcloudProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_secret")
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

	p := &SJdcloudProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(f),
		accessKey:     account,
		secret:        cfg.Secret,
		cfg:           cfg,
	}
	err := p.TryConnect()
	if err != nil {
		return nil, err
	}
	return p, nil
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
	accessKey string
	secret    string
	cfg       cloudprovider.ProviderConfig

	iregion []cloudprovider.ICloudRegion
}

func (p *SJdcloudProvider) TryConnect() error {
	iregions := p.GetIRegions()
	if len(iregions) == 0 {
		return fmt.Errorf("no invalid region for ecloud")
	}
	region := iregions[0].(*jdcloud.SRegion)
	_, _, err := region.GetImages(nil, "private", 1, 10)
	return err
}

func (p *SJdcloudProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Name = p.cfg.Name
	subAccount.Account = p.accessKey
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (p *SJdcloudProvider) GetAccountId() string {
	return p.accessKey
}

func (p *SJdcloudProvider) GetIRegions() []cloudprovider.ICloudRegion {
	if p.iregion != nil {
		return p.iregion
	}
	regions := jdcloud.AllRegions(p.accessKey, p.secret, &p.cfg, false)
	iregions := make([]cloudprovider.ICloudRegion, len(regions))
	for i := range iregions {
		iregions[i] = &regions[i]
	}
	return iregions
}

func (p *SJdcloudProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	iregions := p.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(iregions))), "region_count")
	return info, nil
}

func (p *SJdcloudProvider) GetVersion() string {
	return ""
}

func (p *SJdcloudProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	iregions := p.GetIRegions()
	for i := range iregions {
		if iregions[i].GetGlobalId() == id {
			return iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (p *SJdcloudProvider) GetBalance() (float64, string, error) {
	regions := p.GetIRegions()
	if len(regions) > 0 {
		region := regions[0].(*jdcloud.SRegion)
		balance, err := region.DescribeAccountAmount()
		if err != nil {
			return 0.0, api.CLOUD_PROVIDER_HEALTH_NO_PERMISSION, errors.Wrap(err, "DescribeAccountAmount")
		}
		amount, _ := jsonutils.Marshal(balance).Float("totalAmount")
		if amount < 0 {
			return amount, api.CLOUD_PROVIDER_HEALTH_ARREARS, nil
		} else if amount < 50 {
			return amount, api.CLOUD_PROVIDER_HEALTH_INSUFFICIENT, nil
		}
		return amount, api.CLOUD_PROVIDER_HEALTH_NORMAL, nil
	}
	return 0.0, api.CLOUD_PROVIDER_HEALTH_NORMAL, cloudprovider.ErrNotSupported
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
	iRegions := p.GetIRegions()
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
