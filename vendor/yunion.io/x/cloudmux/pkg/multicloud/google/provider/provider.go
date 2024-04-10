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

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/google"
)

type SGoogleProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactory
}

func (self *SGoogleProviderFactory) GetId() string {
	return google.CLOUD_PROVIDER_GOOGLE
}

func (self *SGoogleProviderFactory) GetName() string {
	return google.CLOUD_PROVIDER_GOOGLE_CN
}

func (self *SGoogleProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return nil
}

func (self *SGoogleProviderFactory) IsSupportPrepaidResources() bool {
	return false
}

func (self *SGoogleProviderFactory) IsClouduserSupportPassword() bool {
	return false
}

func (self *SGoogleProviderFactory) IsClouduserBelongCloudprovider() bool {
	return true
}

func (self *SGoogleProviderFactory) NeedSyncSkuFromCloud() bool {
	return false
}

func (self *SGoogleProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	for key, value := range map[string]string{
		"client_email":   input.GCPClientEmail,
		"project_id":     input.GCPProjectId,
		"private_key_id": input.GCPPrivateKeyId,
		"private_key":    input.GCPPrivateKey,
	} {
		if len(value) == 0 {
			return output, errors.Wrap(cloudprovider.ErrMissingParameter, key)
		}
	}
	output.Account = fmt.Sprintf("%s/%s", input.GCPProjectId, input.GCPClientEmail)
	output.Secret = fmt.Sprintf("%s/%s", input.GCPPrivateKeyId, input.GCPPrivateKey)
	return output, nil
}

func (self *SGoogleProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	projectID, clientEmail := "", ""
	accountInfo := strings.Split(cloudaccount, "/")
	if len(accountInfo) == 2 {
		projectID, clientEmail = accountInfo[0], accountInfo[1]
	}

	for key, value := range map[string]string{
		"private_key_id": input.GCPPrivateKeyId,
		"private_key":    input.GCPPrivateKey,
	} {
		if len(value) == 0 {
			return output, errors.Wrap(cloudprovider.ErrMissingParameter, key)
		}
	}
	if len(input.GCPClientEmail) == 0 {
		input.GCPClientEmail = clientEmail
	}

	if len(input.GCPProjectId) == 0 {
		input.GCPProjectId = projectID
	}

	output = cloudprovider.SCloudaccount{
		Account: fmt.Sprintf("%s/%s", input.GCPProjectId, input.GCPClientEmail),
		Secret:  fmt.Sprintf("%s/%s", input.GCPPrivateKeyId, input.GCPPrivateKey),
	}
	return output, nil
}

func (self *SGoogleProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	privateKeyID, privateKey := "", ""
	privateKeyInfo := strings.Split(cfg.Secret, "/")
	if len(privateKeyInfo) < 2 {
		return nil, fmt.Errorf("Missing privateKeyID or privateKey for google cloud")
	}
	privateKeyID = privateKeyInfo[0]
	privateKey = strings.Join(privateKeyInfo[1:], "/")
	projectID, clientEmail := "", ""
	accountInfo := strings.Split(cfg.Account, "/")
	if len(accountInfo) < 2 {
		return nil, fmt.Errorf("Invalid projectID or client email for google cloud %s", cfg.Account)
	}
	projectID, clientEmail = accountInfo[0], accountInfo[1]

	client, err := google.NewGoogleClient(
		google.NewGoogleClientConfig(
			projectID, clientEmail, privateKeyID, privateKey,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SGoogleProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func parseAccount(account, secret string) (projectId string, clientEmail string, privateKey string, privateKeyId string) {
	accountInfo := strings.Split(account, "/")
	if len(accountInfo) == 2 {
		projectId, clientEmail = accountInfo[0], accountInfo[1]
	}
	secretInfo := strings.Split(secret, "/")
	if len(secretInfo) >= 2 {
		privateKeyId, privateKey = secretInfo[0], strings.Join(secretInfo[1:], "/")
	}
	return
}

func (self *SGoogleProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	projectId, clientEmail, privateKey, privateKeyId := parseAccount(info.Account, info.Secret)
	return map[string]string{
		"GOOGLE_CLIENT_EMAIL":   clientEmail,
		"GOOGLE_PROJECT_ID":     projectId,
		"GOOGLE_PRIVATE_KEY_ID": privateKeyId,
		"GOOGLE_PRIVATE_KEY":    privateKey,
	}, nil
}

func init() {
	factory := SGoogleProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SGoogleProvider struct {
	cloudprovider.SBaseProvider
	client *google.SGoogleClient
}

func (self *SGoogleProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions, _ := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(google.GOOGLE_API_VERSION), "api_version")
	return info, nil
}

func (self *SGoogleProvider) GetVersion() string {
	return google.GOOGLE_API_VERSION
}

func (self *SGoogleProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SGoogleProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SGoogleProvider) GetIamLoginUrl() string {
	return "https://console.cloud.google.com"
}

func (self *SGoogleProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegions()
}

func (self *SGoogleProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SGoogleProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	return &cloudprovider.SBalanceInfo{
		Amount:   0.0,
		Currency: "USD",
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}, cloudprovider.ErrNotSupported
}

func (self *SGoogleProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SGoogleProvider) GetStorageClasses(regionId string) []string {
	return []string{
		"STANDARD", "NEARLINE", "COLDLINE", "ARCHIVE",
	}
}

func (self *SGoogleProvider) GetBucketCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLAuthRead),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SGoogleProvider) GetObjectCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLAuthRead),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SGoogleProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SGoogleProvider) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	return self.client.GetICloudusers()
}

func (self *SGoogleProvider) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	return []cloudprovider.ICloudgroup{}, nil
}

func (self *SGoogleProvider) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SGoogleProvider) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return self.client.GetICloudpolicies()
}

func (self *SGoogleProvider) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	return self.client.CreateIClouduser(conf)
}

func (self *SGoogleProvider) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	return self.client.GetIClouduserByName(name)
}

func (self *SGoogleProvider) CreateICloudpolicy(opts *cloudprovider.SCloudpolicyCreateOptions) (cloudprovider.ICloudpolicy, error) {
	return self.client.CreateICloudpolicy(opts)
}

func (self *SGoogleProvider) GetSamlEntityId() string {
	return cloudprovider.SAML_ENTITY_ID_GOOGLE
}

func (self *SGoogleProvider) GetICloudGlobalVpcs() ([]cloudprovider.ICloudGlobalVpc, error) {
	return self.client.GetICloudGlobalVpcs()
}

func (self *SGoogleProvider) GetICloudGlobalVpcById(id string) (cloudprovider.ICloudGlobalVpc, error) {
	return self.client.GetICloudGlobalVpcById(id)
}

func (self *SGoogleProvider) CreateICloudGlobalVpc(opts *cloudprovider.GlobalVpcCreateOptions) (cloudprovider.ICloudGlobalVpc, error) {
	return self.client.CreateICloudGlobalVpc(opts)
}

func (self *SGoogleProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return self.client.GetMetrics(opts)
}
