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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/multicloud/aws"
)

type SAwsProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactor
}

func (self *SAwsProviderFactory) GetId() string {
	return aws.CLOUD_PROVIDER_AWS
}

func (self *SAwsProviderFactory) GetName() string {
	return aws.CLOUD_PROVIDER_AWS_CN
}

func (self *SAwsProviderFactory) IsSupportPrepaidResources() bool {
	return false
}

func (self *SAwsProviderFactory) IsSupportCloudIdService() bool {
	return true
}

func (self *SAwsProviderFactory) IsSupportCreateCloudgroup() bool {
	return true
}

func (factory *SAwsProviderFactory) IsSystemCloudpolicyUnified() bool {
	return false
}

func (self *SAwsProviderFactory) GetSupportedDnsZoneTypes() []cloudprovider.TDnsZoneType {
	return []cloudprovider.TDnsZoneType{
		cloudprovider.PublicZone,
		cloudprovider.PrivateZone,
	}
}

func (self *SAwsProviderFactory) GetSupportedDnsTypes() map[cloudprovider.TDnsZoneType][]cloudprovider.TDnsType {
	return map[cloudprovider.TDnsZoneType][]cloudprovider.TDnsType{
		cloudprovider.PublicZone: []cloudprovider.TDnsType{
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
		cloudprovider.PrivateZone: []cloudprovider.TDnsType{
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
		cloudprovider.PublicZone: []cloudprovider.TDnsPolicyType{
			cloudprovider.DnsPolicyTypeSimple,
			cloudprovider.DnsPolicyTypeByGeoLocation,
			cloudprovider.DnsPolicyTypeWeighted,
			cloudprovider.DnsPolicyTypeFailover,
			cloudprovider.DnsPolicyTypeMultiValueAnswer,
			cloudprovider.DnsPolicyTypeLatency,
		},
		cloudprovider.PrivateZone: []cloudprovider.TDnsPolicyType{
			cloudprovider.DnsPolicyTypeSimple,
			cloudprovider.DnsPolicyTypeWeighted,
			cloudprovider.DnsPolicyTypeFailover,
			cloudprovider.DnsPolicyTypeMultiValueAnswer,
			cloudprovider.DnsPolicyTypeLatency,
		},
	}
}

func (self *SAwsProviderFactory) GetSupportedDnsPolicyTypeValues() map[cloudprovider.TDnsPolicyType][]cloudprovider.TDnsPolicyValue {
	return map[cloudprovider.TDnsPolicyType][]cloudprovider.TDnsPolicyValue{}
}

func (self *SAwsProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_secret")
	}
	if len(input.Environment) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "environment")
	}
	output.Account = input.AccessKeyId
	output.Secret = input.AccessKeySecret
	output.AccessUrl = input.Environment
	return output, nil
}

func (self *SAwsProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func (self *SAwsProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	client, err := aws.NewAwsClient(
		aws.NewAwsClientConfig(
			cfg.URL, cfg.Account, cfg.Secret,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SAwsProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SAwsProviderFactory) GetClientRC(url, account, secret string) (map[string]string, error) {
	return map[string]string{
		"AWS_ACCESS_URL": url,
		"AWS_ACCESS_KEY": account,
		"AWS_SECRET":     secret,
		"AWS_REGION":     aws.GetDefaultRegionId(url),
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

func (self *SAwsProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SAwsProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
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

func (self *SAwsProvider) GetBalance() (float64, string, error) {
	return 0.0, api.CLOUD_PROVIDER_HEALTH_NORMAL, cloudprovider.ErrNotSupported
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

func (self *SAwsProvider) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return self.client.GetISystemCloudpolicies()
}

func (self *SAwsProvider) GetICustomCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return self.client.GetICustomCloudpolicies()
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

func (self *SAwsProvider) GetSamlSpInitiatedLoginUrl(idpName string) string {
	return ""
}

func (self *SAwsProvider) GetICloudDnsZones() ([]cloudprovider.ICloudDnsZone, error) {
	return self.client.GetICloudDnsZones()
}
func (self *SAwsProvider) GetICloudDnsZoneById(id string) (cloudprovider.ICloudDnsZone, error) {
	return self.client.GetHostedZoneById(id)
}
func (self *SAwsProvider) CreateICloudDnsZone(opts *cloudprovider.SDnsZoneCreateOptions) (cloudprovider.ICloudDnsZone, error) {
	return self.client.CreateHostedZone(opts)
}
