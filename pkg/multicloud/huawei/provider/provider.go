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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/multicloud/huawei"
)

type SHuaweiProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactor
}

func (self *SHuaweiProviderFactory) GetId() string {
	return huawei.CLOUD_PROVIDER_HUAWEI
}

func (self *SHuaweiProviderFactory) GetName() string {
	return huawei.CLOUD_PROVIDER_HUAWEI_CN
}

func (self *SHuaweiProviderFactory) IsCloudeventRegional() bool {
	return true
}

func (self *SHuaweiProviderFactory) GetMaxCloudEventSyncDays() int {
	return 7
}

func (self *SHuaweiProviderFactory) GetMaxCloudEventKeepDays() int {
	return 7
}

func (self *SHuaweiProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
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

func (self *SHuaweiProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func (self *SHuaweiProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	accessKey, projectId := parseAccount(account)
	client, err := huawei.NewHuaweiClient(providerId, providerName, url, accessKey, secret, projectId, false)
	if err != nil {
		return nil, err
	}
	return &SHuaweiProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SHuaweiProviderFactory) GetClientRC(url, account, secret string) (map[string]string, error) {
	accessKey, projectId := parseAccount(account)
	return map[string]string{
		"HUAWEI_CLOUD_ENV":  url,
		"HUAWEI_ACCESS_KEY": accessKey,
		"HUAWEI_SECRET":     secret,
		"HUAWEI_REGION":     huawei.HUAWEI_DEFAULT_REGION,
		"HUAWEI_PROJECT":    projectId,
	}, nil
}

func init() {
	factory := SHuaweiProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SHuaweiProvider struct {
	cloudprovider.SBaseProvider
	client *huawei.SHuaweiClient
}

func (self *SHuaweiProvider) GetVersion() string {
	return self.client.GetVersion()
}

func (self *SHuaweiProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(huawei.HUAWEI_API_VERSION), "api_version")
	return info, nil
}

func (self *SHuaweiProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SHuaweiProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SHuaweiProvider) GetBalance() (float64, string, error) {
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

func (self *SHuaweiProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SHuaweiProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SHuaweiProvider) GetCloudRegionExternalIdPrefix() string {
	return self.client.GetCloudRegionExternalIdPrefix()
}

func (self *SHuaweiProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SHuaweiProvider) GetStorageClasses(regionId string) []string {
	return []string{
		"STANDARD", "WARM", "COLD",
	}
}

func (self *SHuaweiProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}
