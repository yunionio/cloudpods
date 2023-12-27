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
	"yunion.io/x/cloudmux/pkg/multicloud/huawei"
)

type SHuaweiProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactory
}

func (self *SHuaweiProviderFactory) GetId() string {
	return huawei.CLOUD_PROVIDER_HUAWEI
}

func (self *SHuaweiProviderFactory) GetName() string {
	return huawei.CLOUD_PROVIDER_HUAWEI_CN
}

func (self *SHuaweiProviderFactory) IsCloudeventRegional() bool {
	return true
}

func (self *SHuaweiProviderFactory) GetMaxCloudEventSyncDays() int {
	return 7
}

func (self *SHuaweiProviderFactory) GetMaxCloudEventKeepDays() int {
	return 7
}

func (self *SHuaweiProviderFactory) IsSupportCloudIdService() bool {
	return true
}

func (self *SHuaweiProviderFactory) IsSupportClouduserPolicy() bool {
	return false
}

func (self *SHuaweiProviderFactory) IsSupportCreateCloudgroup() bool {
	return true
}

func (factory *SHuaweiProviderFactory) IsSupportCrossCloudEnvVpcPeering() bool {
	return false
}

func (factory *SHuaweiProviderFactory) IsSupportCrossRegionVpcPeering() bool {
	return false
}

func (factory *SHuaweiProviderFactory) IsSupportVpcPeeringVpcCidrOverlap() bool {
	return true
}

func (factory *SHuaweiProviderFactory) IsSupportModifyRouteTable() bool {
	return true
}

func (factory *SHuaweiProviderFactory) IsSupportSAMLAuth() bool {
	return true
}

func (self *SHuaweiProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
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

func (self *SHuaweiProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func (self *SHuaweiProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	client, err := huawei.NewHuaweiClient(
		huawei.NewHuaweiClientConfig(
			cfg.Account, cfg.Secret,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SHuaweiProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SHuaweiProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	return map[string]string{
		"HUAWEI_ACCESS_KEY": info.Account,
		"HUAWEI_SECRET":     info.Secret,
		"HUAWEI_REGION":     huawei.HUAWEI_DEFAULT_REGION,
	}, nil
}

func init() {
	factory := SHuaweiProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SHuaweiProvider struct {
	cloudprovider.SBaseProvider
	client *huawei.SHuaweiClient
}

func (self *SHuaweiProvider) GetVersion() string {
	return self.client.GetVersion()
}

func (self *SHuaweiProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(huawei.HUAWEI_API_VERSION), "api_version")
	return info, nil
}

func (self *SHuaweiProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SHuaweiProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SHuaweiProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	ret := &cloudprovider.SBalanceInfo{Currency: "CNY", Status: api.CLOUD_PROVIDER_HEALTH_UNKNOWN}
	balance, err := self.client.QueryAccountBalance()
	if err != nil {
		return ret, err
	}
	ret.Amount = balance.Amount
	ret.Status = api.CLOUD_PROVIDER_HEALTH_NORMAL
	if balance.Amount < 0.0 && balance.CreditAmount < 0.0 {
		ret.Status = api.CLOUD_PROVIDER_HEALTH_ARREARS
	}
	return ret, nil
}

func (self *SHuaweiProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SHuaweiProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SHuaweiProvider) GetIamLoginUrl() string {
	return self.client.GetIamLoginUrl()
}

func (self *SHuaweiProvider) GetCloudRegionExternalIdPrefix() string {
	return api.CLOUD_PROVIDER_HUAWEI + "/"
}

func (self *SHuaweiProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SHuaweiProvider) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return self.client.CreateIProject(name)
}

func (self *SHuaweiProvider) GetStorageClasses(regionId string) []string {
	return []string{
		"STANDARD", "WARM", "COLD",
	}
}

func (self *SHuaweiProvider) GetBucketCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLAuthRead),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SHuaweiProvider) GetObjectCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLAuthRead),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SHuaweiProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SHuaweiProvider) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	return self.client.CreateIClouduser(conf)
}

func (self *SHuaweiProvider) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	return self.client.GetICloudusers()
}

func (self *SHuaweiProvider) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	return self.client.GetICloudgroups()
}

func (self *SHuaweiProvider) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	return self.client.GetICloudgroupByName(name)
}

func (self *SHuaweiProvider) CreateICloudgroup(name, desc string) (cloudprovider.ICloudgroup, error) {
	return self.client.CreateICloudgroup(name, desc)
}

func (self *SHuaweiProvider) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return self.client.GetISystemCloudpolicies()
}

func (self *SHuaweiProvider) GetICustomCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return []cloudprovider.ICloudpolicy{}, nil
}

func (self *SHuaweiProvider) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	return self.client.GetIClouduserByName(name)
}

func (self *SHuaweiProvider) GetSamlEntityId() string {
	return cloudprovider.SAML_ENTITY_ID_HUAWEI_CLOUD
}

func (self *SHuaweiProvider) GetICloudSAMLProviders() ([]cloudprovider.ICloudSAMLProvider, error) {
	return self.client.GetICloudSAMLProviders()
}

func (self *SHuaweiProvider) CreateICloudSAMLProvider(opts *cloudprovider.SAMLProviderCreateOptions) (cloudprovider.ICloudSAMLProvider, error) {
	sp, err := self.client.CreateSAMLProvider(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSAMLProvider")
	}
	return sp, nil
}

func (self *SHuaweiProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metrics, err := self.client.GetMetrics(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "GetMetrics")
	}
	return metrics, nil
}
