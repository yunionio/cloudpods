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
	huawei "yunion.io/x/cloudmux/pkg/multicloud/hcso"
)

type SHCSOProviderFactory struct {
	cloudprovider.SPrivateCloudBaseProviderFactory
}

func (self *SHCSOProviderFactory) GetId() string {
	return huawei.CLOUD_PROVIDER_HUAWEI
}

func (self *SHCSOProviderFactory) GetName() string {
	return huawei.CLOUD_PROVIDER_HUAWEI_CN
}

func (self *SHCSOProviderFactory) IsCloudeventRegional() bool {
	return true
}

func (self *SHCSOProviderFactory) GetMaxCloudEventSyncDays() int {
	return 7
}

func (self *SHCSOProviderFactory) GetMaxCloudEventKeepDays() int {
	return 7
}

func (factory *SHCSOProviderFactory) IsSupportCrossCloudEnvVpcPeering() bool {
	return false
}

func (factory *SHCSOProviderFactory) IsSupportCrossRegionVpcPeering() bool {
	return false
}

func (factory *SHCSOProviderFactory) IsSupportVpcPeeringVpcCidrOverlap() bool {
	return true
}

func (factory *SHCSOProviderFactory) IsSupportModifyRouteTable() bool {
	return true
}

func (factory *SHCSOProviderFactory) IsSupportSAMLAuth() bool {
	return true
}

func (self *SHCSOProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_secret")
	}

	if input.SHCSOEndpoints == nil {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "cloud_stack_endpoints")
	}

	if len(input.RegionId) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "region_id")
	}

	if len(input.SHCSOEndpoints.EndpointDomain) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "endpoint_domain")
	}

	output.Account = input.AccessKeyId
	output.Secret = input.AccessKeySecret
	return output, nil
}

func (self *SHCSOProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_secret")
	}

	if input.SHCSOEndpoints != nil {
		if len(input.RegionId) == 0 {
			return output, errors.Wrap(cloudprovider.ErrMissingParameter, "region_id")
		}

		if len(input.SHCSOEndpoints.EndpointDomain) == 0 {
			return output, errors.Wrap(cloudprovider.ErrMissingParameter, "endpoint_domain")
		}
	}

	output = cloudprovider.SCloudaccount{
		Account: input.AccessKeyId,
		Secret:  input.AccessKeySecret,
	}
	return output, nil
}

func parseAccount(account string) (accessKey string, projectId string) {
	segs := strings.Split(account, "/")
	if len(segs) == 2 {
		accessKey = segs[0]
		projectId = segs[1]
	} else {
		accessKey = account
		projectId = ""
	}

	return
}

func (self *SHCSOProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	hscsoEndpoints := cloudprovider.SHCSOEndpoints{}
	if cfg.Options != nil {
		cfg.Options.Unmarshal(&hscsoEndpoints)
	}
	accessKey, project_id := parseAccount(cfg.Account)
	client, err := huawei.NewHuaweiClient(
		huawei.NewHuaweiClientConfig(
			accessKey, cfg.Secret, project_id, &hscsoEndpoints,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SHCSOProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SHCSOProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	accessKey, projectId := parseAccount(info.Account)
	region := ""
	data := strings.Split(info.Name, "-")
	if len(data) >= 3 {
		region = strings.Join(data[2:], "-")
	}
	return map[string]string{
		"HUAWEI_CLOUD_ENV":  info.Url,
		"HUAWEI_ACCESS_KEY": accessKey,
		"HUAWEI_SECRET":     info.Secret,
		"HUAWEI_REGION":     region,
		"HUAWEI_PROJECT":    projectId,
	}, nil
}

func (self *SHCSOProviderFactory) IsMultiTenant() bool {
	return true
}

func init() {
	factory := SHCSOProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SHCSOProvider struct {
	cloudprovider.SBaseProvider
	client *huawei.SHuaweiClient
}

func (self *SHCSOProvider) GetVersion() string {
	return self.client.GetVersion()
}

func (self *SHCSOProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions, _ := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(huawei.HUAWEI_API_VERSION), "api_version")
	return info, nil
}

func (self *SHCSOProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegions()
}

func (self *SHCSOProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SHCSOProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	return &cloudprovider.SBalanceInfo{
		Amount:   0.0,
		Currency: "CNY",
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}, cloudprovider.ErrNotSupported
}

func (self *SHCSOProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SHCSOProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SHCSOProvider) GetIamLoginUrl() string {
	return self.client.GetIamLoginUrl()
}

func (self *SHCSOProvider) GetCloudRegionExternalIdPrefix() string {
	return self.client.GetCloudRegionExternalIdPrefix()
}

func (self *SHCSOProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SHCSOProvider) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return self.client.CreateIProject(name)
}

func (self *SHCSOProvider) GetStorageClasses(regionId string) []string {
	return []string{
		"STANDARD", "WARM", "COLD",
	}
}

func (self *SHCSOProvider) GetBucketCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLAuthRead),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SHCSOProvider) GetObjectCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLAuthRead),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SHCSOProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SHCSOProvider) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	return self.client.CreateIClouduser(conf)
}

func (self *SHCSOProvider) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	return self.client.GetICloudusers()
}

func (self *SHCSOProvider) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	return self.client.GetICloudgroups()
}

func (self *SHCSOProvider) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	return self.client.GetICloudgroupByName(name)
}

func (self *SHCSOProvider) CreateICloudgroup(name, desc string) (cloudprovider.ICloudgroup, error) {
	return self.client.CreateICloudgroup(name, desc)
}

func (self *SHCSOProvider) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return self.client.GetICloudpolicies()
}

func (self *SHCSOProvider) CreateICloudpolicy(opts *cloudprovider.SCloudpolicyCreateOptions) (cloudprovider.ICloudpolicy, error) {
	return self.client.CreateICloudpolicy(opts)
}

func (self *SHCSOProvider) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	return self.client.GetIClouduserByName(name)
}

func (self *SHCSOProvider) GetSamlEntityId() string {
	return self.client.GetSamlEntityId()
}

func (self *SHCSOProvider) GetICloudSAMLProviders() ([]cloudprovider.ICloudSAMLProvider, error) {
	return self.client.GetICloudSAMLProviders()
}

func (self *SHCSOProvider) CreateICloudSAMLProvider(opts *cloudprovider.SAMLProviderCreateOptions) (cloudprovider.ICloudSAMLProvider, error) {
	sp, err := self.client.CreateSAMLProvider(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSAMLProvider")
	}
	return sp, nil
}

func (self *SHCSOProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metrics, err := self.client.GetMetrics(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "GetMetrics")
	}
	return metrics, nil
}
