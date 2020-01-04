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
	"yunion.io/x/onecloud/pkg/multicloud/google"
)

type SGoogleProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactor
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
	return true
}

func (self *SGoogleProviderFactory) NeedSyncSkuFromCloud() bool {
	return false
}

func (self *SGoogleProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	for key, value := range map[string]string{
		"client_email":   input.ClientEmail,
		"project_id":     input.ProjectId,
		"private_key_id": input.PrivateKeyId,
		"private_key":    input.PrivateKey,
	} {
		if len(value) == 0 {
			return output, errors.Wrap(httperrors.ErrMissingParameter, key)
		}
	}
	output.Account = fmt.Sprintf("%s/%s", input.ProjectId, input.ClientEmail)
	output.Secret = fmt.Sprintf("%s/%s", input.PrivateKeyId, input.PrivateKey)
	return output, nil
}

func (self *SGoogleProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	projectID, clientEmail := "", ""
	accountInfo := strings.Split(cloudaccount, "/")
	if len(accountInfo) == 2 {
		projectID, clientEmail = accountInfo[0], accountInfo[1]
	}

	for key, value := range map[string]string{
		"private_key_id": input.PrivateKeyId,
		"private_key":    input.PrivateKey,
	} {
		if len(value) == 0 {
			return output, errors.Wrap(httperrors.ErrMissingParameter, key)
		}
	}
	if len(input.ClientEmail) == 0 {
		input.ClientEmail = clientEmail
	}

	if len(input.ProjectId) == 0 {
		input.ProjectId = projectID
	}

	output = cloudprovider.SCloudaccount{
		Account: fmt.Sprintf("%s/%s", input.ProjectId, input.ClientEmail),
		Secret:  fmt.Sprintf("%s/%s", input.PrivateKeyId, input.PrivateKey),
	}
	return output, nil
}

func (self *SGoogleProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	privateKeyID, privateKey := "", ""
	privateKeyInfo := strings.Split(secret, "/")
	if len(privateKeyInfo) < 2 {
		return nil, fmt.Errorf("Missing privateKeyID or privateKey for google cloud")
	}
	privateKeyID = privateKeyInfo[0]
	privateKey = strings.Join(privateKeyInfo[1:], "/")
	projectID, clientEmail := "", ""
	accountInfo := strings.Split(account, "/")
	if len(accountInfo) < 2 {
		return nil, fmt.Errorf("Invalid projectID or client email for google cloud %s", account)
	}
	projectID, clientEmail = accountInfo[0], accountInfo[1]
	client, err := google.NewGoogleClient(providerId, providerName, projectID, clientEmail, privateKeyID, privateKey, false)
	if err != nil {
		return nil, err
	}
	return &SGoogleProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SGoogleProviderFactory) GetClientRC(url, account, secret string) (map[string]string, error) {
	return map[string]string{
		"ALIYUN_ACCESS_KEY": account,
		"ALIYUN_SECRET":     secret,
		"ALIYUN_REGION":     google.GOOGLE_DEFAULT_REGION,
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
	regions := self.client.GetIRegions()
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

func (self *SGoogleProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SGoogleProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SGoogleProvider) GetBalance() (float64, string, error) {
	return 0.0, api.CLOUD_PROVIDER_HEALTH_NORMAL, cloudprovider.ErrNotSupported
}

func (self *SGoogleProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SGoogleProvider) GetStorageClasses(regionId string) []string {
	return []string{
		"Standard", "IA", "Archive",
	}
}

func (self *SGoogleProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}
