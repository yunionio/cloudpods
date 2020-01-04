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
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
)

type SQcloudProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactor
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

func (self *SQcloudProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	secretId := account
	appId := ""
	if tmp := strings.Split(account, "/"); len(tmp) == 2 {
		secretId = tmp[0]
		appId = tmp[1]
	}
	client, err := qcloud.NewQcloudClient(providerId, providerName, secretId, secret, appId, false)
	if err != nil {
		return nil, err
	}
	return &SQcloudProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SQcloudProviderFactory) GetClientRC(url, account, secret string) (map[string]string, error) {
	secretId := account
	appId := ""
	if tmp := strings.Split(account, "/"); len(tmp) == 2 {
		secretId = tmp[0]
		appId = tmp[1]
	}
	return map[string]string{
		"QCLOUD_APPID":      appId,
		"QCLOUD_SECRET_ID":  secretId,
		"QCLOUD_SECRET_KEY": secret,
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
	if balance.AvailableAmount <= 0.0 {
		status = api.CLOUD_PROVIDER_HEALTH_ARREARS
	}
	return balance.AvailableAmount, status, nil
}

func (self *SQcloudProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SQcloudProvider) GetStorageClasses(regionId string) []string {
	return []string{
		"STANDARD", "STANDARD_IA", "ARCHIVE",
	}
}

func (self *SQcloudProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}
