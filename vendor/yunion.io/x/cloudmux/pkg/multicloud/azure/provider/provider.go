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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/azure"
)

type SAzureProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactory
}

func (self *SAzureProviderFactory) GetId() string {
	return azure.CLOUD_PROVIDER_AZURE
}

func (self *SAzureProviderFactory) GetName() string {
	return azure.CLOUD_PROVIDER_AZURE_CN
}

func (self *SAzureProviderFactory) GetMaxCloudEventKeepDays() int {
	return 90
}

func (self *SAzureProviderFactory) GetMaxCloudEventSyncDays() int {
	return 7
}

func (self *SAzureProviderFactory) IsCloudeventRegional() bool {
	return false
}

func (self *SAzureProviderFactory) IsSupportSAMLAuth() bool {
	return true
}

func (self *SAzureProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return fmt.Errorf("Changing %s bandwidth is not supported", azure.CLOUD_PROVIDER_AZURE)
}

func (self *SAzureProviderFactory) IsSupportPrepaidResources() bool {
	return false
}

func (self *SAzureProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.DirectoryId) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "directory_id")
	}
	if len(input.ClientId) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "client_id")
	}
	if len(input.ClientSecret) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "client_secret")
	}
	if len(input.Environment) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "environment")
	}
	output.Account = input.DirectoryId
	output.Secret = fmt.Sprintf("%s/%s", input.ClientId, input.ClientSecret)
	output.AccessUrl = input.Environment
	return output, nil
}

func (self *SAzureProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.ClientId) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "client_id")
	}
	if len(input.ClientSecret) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "client_secret")
	}
	output = cloudprovider.SCloudaccount{
		Account: cloudaccount,
		Secret:  fmt.Sprintf("%s/%s", input.ClientId, input.ClientSecret),
	}
	return output, nil
}

func parseAccount(account, secret string) (tenantId string, appId string, appKey string, subId string) {
	clientInfo := strings.Split(secret, "/")
	accountInfo := strings.Split(account, "/")
	tenantId = accountInfo[0]
	if len(accountInfo) > 1 {
		subId = strings.Join(accountInfo[1:], "/")
	}
	appId = clientInfo[0]
	if len(clientInfo) > 1 {
		appKey = strings.Join(clientInfo[1:], "/")
	}
	return
}

func (self *SAzureProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	tenantId, appId, appKey, subId := parseAccount(cfg.Account, cfg.Secret)
	if client, err := azure.NewAzureClient(
		azure.NewAzureClientConfig(
			cfg.URL, tenantId, appId, appKey,
		).SubscriptionId(subId).CloudproviderConfig(cfg),
	); err != nil {
		return nil, err
	} else {
		return &SAzureProvider{
			SBaseProvider: cloudprovider.NewBaseProvider(self),
			client:        client,
		}, nil
	}
}

func (self *SAzureProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	tenantId, appId, appKey, subId := parseAccount(info.Account, info.Secret)
	return map[string]string{
		"AZURE_DIRECTORY_ID":    tenantId,
		"AZURE_SUBSCRIPTION_ID": subId,
		"AZURE_APPLICATION_ID":  appId,
		"AZURE_APPLICATION_KEY": appKey,
		"AZURE_REGION_ID":       info.Region,
		"AZURE_CLOUD_ENV":       info.Url,
	}, nil
}

func init() {
	factory := SAzureProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SAzureProvider struct {
	cloudprovider.SBaseProvider
	client *azure.SAzureClient
}

func (self *SAzureProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions, _ := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(azure.AZURE_API_VERSION), "api_version")
	return info, nil
}

func (self *SAzureProvider) GetVersion() string {
	return azure.AZURE_API_VERSION
}

func (self *SAzureProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SAzureProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SAzureProvider) GetIamLoginUrl() string {
	return self.client.GetIamLoginUrl()
}

func (self *SAzureProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegions()
}

func (self *SAzureProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(id)
}

func (self *SAzureProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	ret := &cloudprovider.SBalanceInfo{
		Amount:   0.0,
		Currency: "CNY",
		Status:   api.CLOUD_PROVIDER_HEALTH_UNKNOWN,
	}
	if self.client.GetAccessEnv() == api.CLOUD_ACCESS_ENV_AZURE_GLOBAL {
		ret.Currency = "USD"
	}
	return ret, cloudprovider.ErrNotSupported
}

func (self *SAzureProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SAzureProvider) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return self.client.CreateIProject(name)
}

func (self *SAzureProvider) GetStorageClasses(regionId string) []string {
	sc, err := self.client.GetStorageClasses(regionId)
	if err != nil {
		log.Errorf("Fail to find storage classes: %s", err)
		return nil
	}
	return sc
}

func (self *SAzureProvider) GetBucketCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLPublicRead),
	}
}

func (self *SAzureProvider) GetObjectCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLPublicRead),
	}
}

func (self *SAzureProvider) GetCloudRegionExternalIdPrefix() string {
	return self.client.GetAccessEnv() + "/"
}

func (self *SAzureProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SAzureProvider) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	return self.client.CreateIClouduser(conf)
}

func (self *SAzureProvider) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	return self.client.GetICloudusers()
}

func (self *SAzureProvider) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	return self.client.GetIClouduserByName(name)
}

func (self *SAzureProvider) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	return self.client.GetICloudgroups()
}

func (self *SAzureProvider) CreateICloudgroup(name, desc string) (cloudprovider.ICloudgroup, error) {
	return self.client.CreateICloudgroup(name, desc)
}

func (self *SAzureProvider) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	return self.client.GetICloudgroupByName(name)
}

func (self *SAzureProvider) GetEnrollmentAccounts() ([]cloudprovider.SEnrollmentAccount, error) {
	return self.client.GetEnrollmentAccounts()
}

func (self *SAzureProvider) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return self.client.GetICloudpolicies()
}

func (self *SAzureProvider) CreateSubscription(input cloudprovider.SubscriptionCreateInput) error {
	return self.client.CreateSubscription(input.Name, input.EnrollmentAccountId, input.OfferType)
}

// fake func
func (self *SAzureProvider) CreateICloudSAMLProvider(opts *cloudprovider.SAMLProviderCreateOptions) (cloudprovider.ICloudSAMLProvider, error) {
	return self.client.CreateSAMLProvider(opts)
}

func (self *SAzureProvider) GetSamlEntityId() string {
	return cloudprovider.SAML_ENTITY_ID_AZURE
}

func (self *SAzureProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return self.client.GetMetrics(opts)
}
