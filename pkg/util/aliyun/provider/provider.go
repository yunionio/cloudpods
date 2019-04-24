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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/aliyun"
)

type SAliyunProviderFactory struct {
}

func (self *SAliyunProviderFactory) GetId() string {
	return aliyun.CLOUD_PROVIDER_ALIYUN
}

func (self *SAliyunProviderFactory) GetName() string {
	return aliyun.CLOUD_PROVIDER_ALIYUN_CN
}

func (self *SAliyunProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return nil
}

func (self *SAliyunProviderFactory) IsPublicCloud() bool {
	return true
}

func (self *SAliyunProviderFactory) IsOnPremise() bool {
	return false
}

func (self *SAliyunProviderFactory) IsSupportPrepaidResources() bool {
	return true
}

func (self *SAliyunProviderFactory) NeedSyncSkuFromCloud() bool {
	return false
}

func (self *SAliyunProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) error {
	accessKeyID, _ := data.GetString("access_key_id")
	if len(accessKeyID) == 0 {
		return httperrors.NewMissingParameterError("access_key_id")
	}
	accessKeySecret, _ := data.GetString("access_key_secret")
	if len(accessKeySecret) == 0 {
		return httperrors.NewMissingParameterError("access_key_secret")
	}
	data.Set("account", jsonutils.NewString(accessKeyID))
	data.Set("secret", jsonutils.NewString(accessKeySecret))
	return nil
}

func (self *SAliyunProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, cloudaccount string) (*cloudprovider.SCloudaccount, error) {

	accessKeyID, _ := data.GetString("access_key_id")
	if len(accessKeyID) == 0 {
		return nil, httperrors.NewMissingParameterError("access_key_id")
	}
	accessKeySecret, _ := data.GetString("access_key_secret")
	if len(accessKeySecret) == 0 {
		return nil, httperrors.NewMissingParameterError("access_key_secret")
	}
	account := &cloudprovider.SCloudaccount{
		Account: accessKeyID,
		Secret:  accessKeySecret,
	}
	return account, nil
}

func (self *SAliyunProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	client, err := aliyun.NewAliyunClient(providerId, providerName, account, secret, false)
	if err != nil {
		return nil, err
	}
	return &SAliyunProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func init() {
	factory := SAliyunProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SAliyunProvider struct {
	cloudprovider.SBaseProvider
	client *aliyun.SAliyunClient
}

func (self *SAliyunProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(aliyun.ALIYUN_API_VERSION), "api_version")
	return info, nil
}

func (self *SAliyunProvider) GetVersion() string {
	return aliyun.ALIYUN_API_VERSION
}

func (self *SAliyunProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SAliyunProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SAliyunProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SAliyunProvider) GetBalance() (float64, string, error) {
	balance, err := self.client.QueryAccountBalance()
	if err != nil {
		return 0.0, api.CLOUD_PROVIDER_HEALTH_UNKNOWN, err
	}
	status := api.CLOUD_PROVIDER_HEALTH_NORMAL
	if balance.AvailableAmount <= 0 {
		status = api.CLOUD_PROVIDER_HEALTH_ARREARS
	} else if balance.AvailableAmount < 100 {
		status = api.CLOUD_PROVIDER_HEALTH_INSUFFICIENT
	}
	return balance.AvailableAmount, status, nil
}

func (self *SAliyunProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}
