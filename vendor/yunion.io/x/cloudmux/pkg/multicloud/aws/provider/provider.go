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
	"yunion.io/x/cloudmux/pkg/multicloud/aws"
)

type SAwsProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactory
}

func (self *SAwsProviderFactory) GetId() string {
	return aws.CLOUD_PROVIDER_AWS
}

func (self *SAwsProviderFactory) GetName() string {
	return aws.CLOUD_PROVIDER_AWS_CN
}

func (self *SAwsProviderFactory) IsCloudeventRegional() bool {
	return true
}

func (self *SAwsProviderFactory) GetMaxCloudEventSyncDays() int {
	return 1
}

func (self *SAwsProviderFactory) GetMaxCloudEventKeepDays() int {
	return 90
}

func (self *SAwsProviderFactory) IsSupportPrepaidResources() bool {
	return false
}

func (factory *SAwsProviderFactory) IsSupportSAMLAuth() bool {
	return true
}

func (self *SAwsProviderFactory) GetSupportedDnsZoneTypes() []cloudprovider.TDnsZoneType {
	return []cloudprovider.TDnsZoneType{
		cloudprovider.PublicZone,
		cloudprovider.PrivateZone,
	}
}

func (self *SAwsProviderFactory) GetSupportedDnsTypes() map[cloudprovider.TDnsZoneType][]cloudprovider.TDnsType {
	return map[cloudprovider.TDnsZoneType][]cloudprovider.TDnsType{
		cloudprovider.PublicZone: {
			cloudprovider.DnsTypeA,
			cloudprovider.DnsTypeAAAA,
			cloudprovider.DnsTypeCAA,
			cloudprovider.DnsTypeCNAME,
			cloudprovider.DnsTypeMX,
			cloudprovider.DnsTypeNS,
			cloudprovider.DnsTypeSRV,
			cloudprovider.DnsTypeSOA,
			cloudprovider.DnsTypeTXT,
			cloudprovider.DnsTypePTR,
			cloudprovider.DnsTypeNAPTR,
			cloudprovider.DnsTypeSPF,
		},
		cloudprovider.PrivateZone: {
			cloudprovider.DnsTypeA,
			cloudprovider.DnsTypeAAAA,
			cloudprovider.DnsTypeCAA,
			cloudprovider.DnsTypeCNAME,
			cloudprovider.DnsTypeMX,
			cloudprovider.DnsTypeNS,
			cloudprovider.DnsTypeSRV,
			cloudprovider.DnsTypeSOA,
			cloudprovider.DnsTypeTXT,
			cloudprovider.DnsTypePTR,
			cloudprovider.DnsTypeNAPTR,
			cloudprovider.DnsTypeSPF,
		},
	}
}

func (self *SAwsProviderFactory) GetSupportedDnsPolicyTypes() map[cloudprovider.TDnsZoneType][]cloudprovider.TDnsPolicyType {
	return map[cloudprovider.TDnsZoneType][]cloudprovider.TDnsPolicyType{
		cloudprovider.PublicZone: {
			cloudprovider.DnsPolicyTypeSimple,
			cloudprovider.DnsPolicyTypeByGeoLocation,
			cloudprovider.DnsPolicyTypeWeighted,
			cloudprovider.DnsPolicyTypeFailover,
			cloudprovider.DnsPolicyTypeMultiValueAnswer,
			cloudprovider.DnsPolicyTypeLatency,
		},
		cloudprovider.PrivateZone: {
			cloudprovider.DnsPolicyTypeSimple,
			cloudprovider.DnsPolicyTypeWeighted,
			cloudprovider.DnsPolicyTypeFailover,
			cloudprovider.DnsPolicyTypeMultiValueAnswer,
			cloudprovider.DnsPolicyTypeLatency,
		},
	}
}

func (self *SAwsProviderFactory) GetSupportedDnsPolicyValues() map[cloudprovider.TDnsPolicyType][]cloudprovider.TDnsPolicyValue {
	return map[cloudprovider.TDnsPolicyType][]cloudprovider.TDnsPolicyValue{
		cloudprovider.DnsPolicyTypeByGeoLocation: cloudprovider.AwsGeoLocations,
		cloudprovider.DnsPolicyTypeLatency:       cloudprovider.AwsRegions,
		cloudprovider.DnsPolicyTypeFailover:      cloudprovider.AwsFailovers,
	}
}

func (factory *SAwsProviderFactory) IsSupportCrossCloudEnvVpcPeering() bool {
	return false
}

func (factory *SAwsProviderFactory) IsSupportCrossRegionVpcPeering() bool {
	return true
}

func (factory *SAwsProviderFactory) IsSupportVpcPeeringVpcCidrOverlap() bool {
	return false
}

func (factory *SAwsProviderFactory) IsSupportModifyRouteTable() bool {
	return true
}

func (self *SAwsProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_secret")
	}
	if len(input.Environment) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "environment")
	}
	output.Account = input.AccessKeyId
	output.Secret = input.AccessKeySecret
	output.AccessUrl = input.Environment
	return output, nil
}

func (self *SAwsProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func parseAccount(account, secret string) (accessKey string, secretKey string, accountId string) {
	slash := strings.Index(account, "/")
	if slash > 0 {
		accessKey = account[:slash]
		accountId = account[slash+1:]
	} else {
		accessKey = account
	}
	secretKey = secret
	return
}

func (self *SAwsProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	extra := cloudprovider.SAWSExtraOptions{}
	if cfg.Options != nil {
		cfg.Options.Unmarshal(&extra)
	}
	accessKey, secret, accountId := parseAccount(cfg.Account, cfg.Secret)
	client, err := aws.NewAwsClient(
		aws.NewAwsClientConfig(
			cfg.URL, accessKey, secret, accountId,
		).SetAssumeRole(extra.AWSAssumeRoleName).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, errors.Wrap(err, "NewAwsClient")
	}
	return &SAwsProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SAwsProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	accessKey, secret, accountId := parseAccount(info.Account, info.Secret)
	return map[string]string{
		"AWS_ACCESS_URL": info.Url,
		"AWS_ACCESS_KEY": accessKey,
		"AWS_SECRET":     secret,
		"AWS_REGION":     aws.GetDefaultRegionId(info.Url),
		"AWS_ACCOUNT_ID": accountId,
	}, nil
}

func init() {
	factory := SAwsProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SAwsProvider struct {
	cloudprovider.SBaseProvider
	client *aws.SAwsClient
}

func (self *SAwsProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SAwsProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SAwsProvider) GetIamLoginUrl() string {
	return self.client.GetIamLoginUrl()
}

func (self *SAwsProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegions()
}

func (self *SAwsProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions, _ := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(aws.AWS_API_VERSION), "api_version")
	return info, nil
}

func (self *SAwsProvider) GetVersion() string {
	return aws.AWS_API_VERSION
}

func (self *SAwsProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(id)
}

func (self *SAwsProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	ret := &cloudprovider.SBalanceInfo{
		Amount:   0.0,
		Currency: "CNY",
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}
	if self.client.GetAccessEnv() == api.CLOUD_ACCESS_ENV_AWS_GLOBAL {
		ret.Currency = "USD"
	}
	return ret, cloudprovider.ErrNotSupported
}

func (self *SAwsProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SAwsProvider) GetStorageClasses(regionId string) []string {
	return []string{
		"STANDARD",
		"STANDARD_IA",
		"ONEZONE_IA",
		"GLACIER",
		"DEEP_ARCHIVE",
		"INTELLIGENT_TIERING",
	}
}

func (self *SAwsProvider) GetBucketCannedAcls(regionId string) []string {
	return self.client.GetBucketCannedAcls()
}

func (self *SAwsProvider) GetObjectCannedAcls(regionId string) []string {
	return self.client.GetObjectCannedAcls()
}

func (self *SAwsProvider) GetCloudRegionExternalIdPrefix() string {
	return self.client.GetAccessEnv() + "/"
}

func (self *SAwsProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SAwsProvider) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	return self.client.CreateIClouduser(conf)
}

func (self *SAwsProvider) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	return self.client.GetICloudusers()
}

func (self *SAwsProvider) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	return self.client.GetICloudgroups()
}

func (self *SAwsProvider) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	return self.client.GetICloudgroupByName(name)
}

func (self *SAwsProvider) CreateICloudgroup(name, desc string) (cloudprovider.ICloudgroup, error) {
	return self.client.CreateICloudgroup(name, desc)
}

func (self *SAwsProvider) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return self.client.GetICloudpolicies()
}

func (self *SAwsProvider) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	return self.client.GetIClouduserByName(name)
}

func (self *SAwsProvider) CreateICloudpolicy(opts *cloudprovider.SCloudpolicyCreateOptions) (cloudprovider.ICloudpolicy, error) {
	return self.client.CreateICloudpolicy(opts)
}

func (self *SAwsProvider) GetSamlEntityId() string {
	return self.client.GetSamlEntityId()
}

func (self *SAwsProvider) GetICloudDnsZones() ([]cloudprovider.ICloudDnsZone, error) {
	return self.client.GetICloudDnsZones()
}

func (self *SAwsProvider) GetICloudDnsZoneById(id string) (cloudprovider.ICloudDnsZone, error) {
	zone, err := self.client.GetDnsZone(id)
	if err != nil {
		return nil, err
	}
	return &zone.HostedZone, nil
}

func (self *SAwsProvider) CreateICloudDnsZone(opts *cloudprovider.SDnsZoneCreateOptions) (cloudprovider.ICloudDnsZone, error) {
	return self.client.CreateDnsZone(opts)
}

func (self *SAwsProvider) GetICloudSAMLProviders() ([]cloudprovider.ICloudSAMLProvider, error) {
	return self.client.GetICloudSAMLProviders()
}

func (self *SAwsProvider) CreateICloudSAMLProvider(opts *cloudprovider.SAMLProviderCreateOptions) (cloudprovider.ICloudSAMLProvider, error) {
	sp, err := self.client.CreateSAMLProvider(opts.Name, opts.Metadata.String())
	if err != nil {
		return nil, errors.Wrap(err, "CreateSAMLProvider")
	}
	return sp, nil
}

func (self *SAwsProvider) GetICloudroles() ([]cloudprovider.ICloudrole, error) {
	return self.client.GetICloudroles()
}

func (self *SAwsProvider) GetICloudroleById(id string) (cloudprovider.ICloudrole, error) {
	roles, err := self.GetICloudroles()
	if err != nil {
		return nil, errors.Wrapf(err, "GetICloudroles")
	}
	for i := range roles {
		if roles[i].GetGlobalId() == id {
			return roles[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SAwsProvider) GetICloudroleByName(name string) (cloudprovider.ICloudrole, error) {
	role, err := self.client.GetRole(name)
	if err != nil {
		return nil, errors.Wrapf(err, "GetRole(%s)", name)
	}
	return role, nil
}

func (self *SAwsProvider) CreateICloudrole(opts *cloudprovider.SRoleCreateOptions) (cloudprovider.ICloudrole, error) {
	role, err := self.client.CreateRole(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateRole")
	}
	return role, nil
}

func (self *SAwsProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return self.client.GetMetrics(opts)
}

func (self *SAwsProvider) GetICloudCDNDomains() ([]cloudprovider.ICloudCDNDomain, error) {
	return self.client.GetICloudCDNDomains()
}

func (self *SAwsProvider) GetICloudCDNDomainByName(name string) (cloudprovider.ICloudCDNDomain, error) {
	return self.client.GetICloudCDNDomainByName(name)
}
