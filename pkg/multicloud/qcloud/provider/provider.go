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
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
)

type SQcloudProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactory
}

func (self *SQcloudProviderFactory) GetId() string {
	return qcloud.CLOUD_PROVIDER_QCLOUD
}

func (self *SQcloudProviderFactory) GetName() string {
	return qcloud.CLOUD_PROVIDER_QCLOUD_CN
}

func (self *SQcloudProviderFactory) IsCloudeventRegional() bool {
	return true
}

func (self *SQcloudProviderFactory) GetMaxCloudEventSyncDays() int {
	return 7
}

func (self *SQcloudProviderFactory) GetMaxCloudEventKeepDays() int {
	return 30
}

func (self *SQcloudProviderFactory) IsSupportCloudIdService() bool {
	return true
}

func (self *SQcloudProviderFactory) IsSupportCreateCloudgroup() bool {
	return true
}

func (self *SQcloudProviderFactory) IsSupportSAMLAuth() bool {
	return true
}

func (self *SQcloudProviderFactory) IsSupportCrossCloudEnvVpcPeering() bool {
	return false
}

func (self *SQcloudProviderFactory) IsSupportCrossRegionVpcPeering() bool {
	return true
}

func (self *SQcloudProviderFactory) IsSupportVpcPeeringVpcCidrOverlap() bool {
	return false
}

func (self *SQcloudProviderFactory) ValidateCrossRegionVpcPeeringBandWidth(bandwidth int) error {
	validatedBandwidths := []int{10, 20, 50, 100, 200, 500, 1000}
	ok, _ := utils.InArray(bandwidth, validatedBandwidths)
	if ok {
		return nil
	}
	return httperrors.NewInputParameterError("require validated qcloud cross region vpcPeering bandwidth values:[10, 20, 50, 100, 200, 500, 1000],unit Mbps")
}

func (self *SQcloudProviderFactory) GetSupportedDnsZoneTypes() []cloudprovider.TDnsZoneType {
	return []cloudprovider.TDnsZoneType{
		cloudprovider.PublicZone,
	}
}

func (self *SQcloudProviderFactory) GetSupportedDnsTypes() map[cloudprovider.TDnsZoneType][]cloudprovider.TDnsType {
	return map[cloudprovider.TDnsZoneType][]cloudprovider.TDnsType{
		cloudprovider.PublicZone: []cloudprovider.TDnsType{
			cloudprovider.DnsTypeA,
			cloudprovider.DnsTypeAAAA,
			cloudprovider.DnsTypeCNAME,
			cloudprovider.DnsTypeMX,
			cloudprovider.DnsTypeNS,
			cloudprovider.DnsTypeSRV,
			cloudprovider.DnsTypeTXT,
			cloudprovider.DnsTypePTR,
		},
	}
}

func (self *SQcloudProviderFactory) GetSupportedDnsPolicyTypes() map[cloudprovider.TDnsZoneType][]cloudprovider.TDnsPolicyType {
	return map[cloudprovider.TDnsZoneType][]cloudprovider.TDnsPolicyType{
		cloudprovider.PublicZone: []cloudprovider.TDnsPolicyType{
			cloudprovider.DnsPolicyTypeSimple,
			cloudprovider.DnsPolicyTypeByCarrier,
			cloudprovider.DnsPolicyTypeByGeoLocation,
			cloudprovider.DnsPolicyTypeBySearchEngine,
			cloudprovider.DnsPolicyTypeWeighted,
		},
	}
}

func (self *SQcloudProviderFactory) GetSupportedDnsPolicyValues() map[cloudprovider.TDnsPolicyType][]cloudprovider.TDnsPolicyValue {
	return map[cloudprovider.TDnsPolicyType][]cloudprovider.TDnsPolicyValue{
		cloudprovider.DnsPolicyTypeByCarrier: []cloudprovider.TDnsPolicyValue{
			cloudprovider.DnsPolicyValueUnicom,
			cloudprovider.DnsPolicyValueTelecom,
			cloudprovider.DnsPolicyValueChinaMobile,
			cloudprovider.DnsPolicyValueCernet,
		},
		cloudprovider.DnsPolicyTypeByGeoLocation: []cloudprovider.TDnsPolicyValue{
			cloudprovider.DnsPolicyValueOversea,
			cloudprovider.DnsPolicyValueMainland,
		},
		cloudprovider.DnsPolicyTypeBySearchEngine: []cloudprovider.TDnsPolicyValue{
			cloudprovider.DnsPolicyValueBaidu,
			cloudprovider.DnsPolicyValueBing,
			cloudprovider.DnsPolicyValueGoogle,
			cloudprovider.DnsPolicyValueYoudao,
			cloudprovider.DnsPolicyValueSousou,
			cloudprovider.DnsPolicyValueSougou,
			cloudprovider.DnsPolicyValueQihu360,
		},
	}
}

// https://buy.cloud.tencent.com/cns?from=gobuy&domain=example4.com
func (self *SQcloudProviderFactory) GetTTLRange(zoneType cloudprovider.TDnsZoneType, productType cloudprovider.TDnsProductType) cloudprovider.TTlRange {
	if len(productType) > 0 {
		switch productType {
		case cloudprovider.DnsProductEnterpriseUltimate:
			return cloudprovider.TtlRangeQcloudEnterpriseUltimate
		case cloudprovider.DnsProductEnterpriseStandard:
			return cloudprovider.TtlRangeQcloudEnterpriseStandard
		case cloudprovider.DnsProductEnterpriseBasic:
			return cloudprovider.TtlRangeQcloudEnterpriseBasic
		case cloudprovider.DnsProductPersonalProfessional:
			return cloudprovider.TtlRangeQcloudPersonalProfessional
		case cloudprovider.DnsProductFree:
			return cloudprovider.TtlRangeQcloudFree
		default:
			return cloudprovider.TtlRangeQcloudFree
		}
	}
	return cloudprovider.TtlRangeQcloudFree
}

func (self *SQcloudProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	if len(instanceId) == 0 {
		return fmt.Errorf("Only changes to the binding machine's EIP bandwidth are supported")
	}
	return nil
}

func (self *SQcloudProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AppId) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "app_id")
	}
	if len(input.SecretId) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "secret_id")
	}
	if len(input.SecretKey) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "secret_key")
	}
	output.Account = fmt.Sprintf("%s/%s", input.SecretId, input.AppId)
	output.Secret = input.SecretKey
	return output, nil
}

func (self *SQcloudProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AppId) == 0 {
		accountInfo := strings.Split(cloudaccount, "/")
		if len(accountInfo) < 2 {
			return output, errors.Wrap(httperrors.ErrMissingParameter, "app_id")
		}
		input.AppId = accountInfo[1]
	}
	if len(input.SecretId) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "secret_id")
	}
	if len(input.SecretKey) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "secret_key")
	}
	output = cloudprovider.SCloudaccount{
		Account: fmt.Sprintf("%s/%s", input.SecretId, input.AppId),
		Secret:  input.SecretKey,
	}
	return output, nil
}

func (self *SQcloudProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	secretId := cfg.Account
	appId := ""
	if tmp := strings.Split(cfg.Account, "/"); len(tmp) == 2 {
		secretId = tmp[0]
		appId = tmp[1]
	}
	client, err := qcloud.NewQcloudClient(
		qcloud.NewQcloudClientConfig(
			secretId, cfg.Secret,
		).AppId(appId).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SQcloudProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SQcloudProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	secretId := info.Account
	appId := ""
	if tmp := strings.Split(info.Account, "/"); len(tmp) == 2 {
		secretId = tmp[0]
		appId = tmp[1]
	}
	return map[string]string{
		"QCLOUD_APPID":      appId,
		"QCLOUD_SECRET_ID":  secretId,
		"QCLOUD_SECRET_KEY": info.Secret,
		"QCLOUD_REGION":     qcloud.QCLOUD_DEFAULT_REGION,
	}, nil
}

func init() {
	factory := SQcloudProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SQcloudProvider struct {
	cloudprovider.SBaseProvider
	client *qcloud.SQcloudClient
}

func (self *SQcloudProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(qcloud.QCLOUD_API_VERSION), "api_version")
	return info, nil
}

func (self *SQcloudProvider) GetVersion() string {
	return qcloud.QCLOUD_API_VERSION
}

func (self *SQcloudProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SQcloudProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SQcloudProvider) GetIamLoginUrl() string {
	return self.client.GetIamLoginUrl()
}

func (self *SQcloudProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SQcloudProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(id)
}

func (self *SQcloudProvider) GetBalance() (float64, string, error) {
	balance, err := self.client.QueryAccountBalance()
	if err != nil {
		return 0.0, api.CLOUD_PROVIDER_HEALTH_UNKNOWN, err
	}
	status := api.CLOUD_PROVIDER_HEALTH_NORMAL
	if balance.AvailableAmount < 0.0 {
		status = api.CLOUD_PROVIDER_HEALTH_ARREARS
	}
	return balance.AvailableAmount, status, nil
}

func (self *SQcloudProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SQcloudProvider) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return self.client.CreateIProject(name)
}

func (self *SQcloudProvider) GetStorageClasses(regionId string) []string {
	return []string{
		"STANDARD", "STANDARD_IA", "ARCHIVE",
	}
}

func (self *SQcloudProvider) GetBucketCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLAuthRead),
		string(cloudprovider.ACLPublicRead),
	}
}

func (self *SQcloudProvider) GetObjectCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLAuthRead),
		string(cloudprovider.ACLPublicRead),
	}
}

func (self *SQcloudProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SQcloudProvider) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	return self.client.CreateIClouduser(conf)
}

func (self *SQcloudProvider) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	return self.client.GetICloudusers()
}

func (self *SQcloudProvider) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	return self.client.GetICloudgroups()
}

func (self *SQcloudProvider) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	return self.client.GetICloudgroupByName(name)
}

func (self *SQcloudProvider) CreateICloudgroup(name, desc string) (cloudprovider.ICloudgroup, error) {
	return self.client.CreateICloudgroup(name, desc)
}

func (self *SQcloudProvider) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return self.client.GetISystemCloudpolicies()
}

func (self *SQcloudProvider) GetICustomCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return self.client.GetICustomCloudpolicies()
}

func (self *SQcloudProvider) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	return self.client.GetIClouduserByName(name)
}

func (self *SQcloudProvider) CreateICloudpolicy(opts *cloudprovider.SCloudpolicyCreateOptions) (cloudprovider.ICloudpolicy, error) {
	return self.client.CreateICloudpolicy(opts)
}

func (self *SQcloudProvider) GetSamlEntityId() string {
	return cloudprovider.SAML_ENTITY_ID_QCLOUD
}

func (self *SQcloudProvider) GetICloudDnsZones() ([]cloudprovider.ICloudDnsZone, error) {
	return self.client.GetICloudDnsZones()
}

func (self *SQcloudProvider) GetICloudDnsZoneById(id string) (cloudprovider.ICloudDnsZone, error) {
	return self.client.GetDomainById(id)
}

func (self *SQcloudProvider) CreateICloudDnsZone(opts *cloudprovider.SDnsZoneCreateOptions) (cloudprovider.ICloudDnsZone, error) {
	return self.client.CreateICloudDnsZone(opts)
}

func (self *SQcloudProvider) CreateICloudSAMLProvider(opts *cloudprovider.SAMLProviderCreateOptions) (cloudprovider.ICloudSAMLProvider, error) {
	saml, err := self.client.CreateSAMLProvider(opts.Name, opts.Metadata.String(), "")
	if err != nil {
		return nil, errors.Wrap(err, "CreateSAMLProvider")
	}
	return saml, nil
}

func (self *SQcloudProvider) GetICloudSAMLProviders() ([]cloudprovider.ICloudSAMLProvider, error) {
	return self.client.GetICloudSAMLProviders()
}

func (self *SQcloudProvider) CreateICloudrole(opts *cloudprovider.SRoleCreateOptions) (cloudprovider.ICloudrole, error) {
	if len(opts.SAMLProvider) > 0 {
		document := fmt.Sprintf(`{"version":"2.0","statement":[{"action":"name/sts:AssumeRoleWithSAML","effect":"allow","principal":{"federated":["qcs::cam::uin/%s:saml-provider/%s"]},"condition":{}}]}`, self.client.GetAccountId(), opts.SAMLProvider)
		role, err := self.client.CreateRole(opts.Name, document, opts.Desc)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateRole")
		}
		return role, nil
	}
	role, err := self.client.CreateRole(opts.Name, "", opts.Desc)
	if err != nil {
		return nil, errors.Wrapf(err, "")
	}
	return role, nil
}

func (self *SQcloudProvider) GetICloudroles() ([]cloudprovider.ICloudrole, error) {
	return self.client.GetICloudroles()
}

func (self *SQcloudProvider) GetICloudroleByName(name string) (cloudprovider.ICloudrole, error) {
	role, err := self.client.GetRole(name)
	if err != nil {
		return nil, errors.Wrapf(err, "GetRole(%s)", name)
	}
	return role, nil
}

func (self *SQcloudProvider) GetICloudroleById(id string) (cloudprovider.ICloudrole, error) {
	return self.GetICloudroleByName(id)
}

func (self *SQcloudProvider) GetICloudInterVpcNetworks() ([]cloudprovider.ICloudInterVpcNetwork, error) {
	return self.client.GetICloudInterVpcNetworks()
}

func (self *SQcloudProvider) GetICloudInterVpcNetworkById(id string) (cloudprovider.ICloudInterVpcNetwork, error) {
	return self.client.GetICloudInterVpcNetworkById(id)
}

func (self *SQcloudProvider) CreateICloudInterVpcNetwork(opts *cloudprovider.SInterVpcNetworkCreateOptions) (cloudprovider.ICloudInterVpcNetwork, error) {
	return self.client.CreateICloudInterVpcNetwork(opts)
}
