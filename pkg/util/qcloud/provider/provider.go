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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/qcloud"
)

type SQcloudProviderFactory struct {
	// providerTable map[string]*SQcloudProvider
}

func (self *SQcloudProviderFactory) GetId() string {
	return qcloud.CLOUD_PROVIDER_QCLOUD
}

func (self *SQcloudProviderFactory) GetName() string {
	return qcloud.CLOUD_PROVIDER_QCLOUD_CN
}

func (self *SQcloudProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	if len(instanceId) == 0 {
		return fmt.Errorf("Only changes to the binding machine's EIP bandwidth are supported")
	}
	return nil
}

func (self *SQcloudProviderFactory) IsPublicCloud() bool {
	return true
}

func (self *SQcloudProviderFactory) IsOnPremise() bool {
	return false
}

func (self *SQcloudProviderFactory) IsSupportPrepaidResources() bool {
	return true
}

func (self *SQcloudProviderFactory) NeedSyncSkuFromCloud() bool {
	return false
}

func (self *SQcloudProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) error {
	appID, _ := data.GetString("app_id")
	if len(appID) == 0 {
		return httperrors.NewMissingParameterError("app_id")
	}
	secretID, _ := data.GetString("secret_id")
	if len(secretID) == 0 {
		return httperrors.NewMissingParameterError("secret_id")
	}
	secretKey, _ := data.GetString("secret_key")
	if len(secretKey) == 0 {
		return httperrors.NewMissingParameterError("secret_key")
	}
	data.Set("account", jsonutils.NewString(fmt.Sprintf("%s/%s", secretID, appID)))
	data.Set("secret", jsonutils.NewString(secretKey))
	return nil
}

func (self *SQcloudProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, cloudaccount string) (*cloudprovider.SCloudaccount, error) {
	appID, _ := data.GetString("app_id")
	if len(appID) == 0 {
		accountInfo := strings.Split(cloudaccount, "/")
		if len(accountInfo) < 2 {
			return nil, httperrors.NewMissingParameterError("app_id")
		}
		appID = accountInfo[1]
	}
	secretID, _ := data.GetString("secret_id")
	if len(secretID) == 0 {
		return nil, httperrors.NewMissingParameterError("secret_id")
	}
	secretKey, _ := data.GetString("secret_key")
	if len(secretKey) == 0 {
		return nil, httperrors.NewMissingParameterError("secret_key")
	}
	account := &cloudprovider.SCloudaccount{
		Account: fmt.Sprintf("%s/%s", secretID, appID),
		Secret:  secretKey,
	}
	return account, nil
}

func (self *SQcloudProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	client, err := qcloud.NewQcloudClient(providerId, providerName, account, secret, false)
	if err != nil {
		return nil, err
	}
	return &SQcloudProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
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
