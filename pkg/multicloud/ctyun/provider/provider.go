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
	"yunion.io/x/onecloud/pkg/multicloud/ctyun"
)

type SCtyunProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactor
}

func (self *SCtyunProviderFactory) GetId() string {
	return ctyun.CLOUD_PROVIDER_CTYUN
}

func (self *SCtyunProviderFactory) GetName() string {
	return ctyun.CLOUD_PROVIDER_CTYUN_CN
}

func (self *SCtyunProviderFactory) IsSupportPrepaidResources() bool {
	return true
}

func (self *SCtyunProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
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

func (self *SCtyunProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func (self *SCtyunProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	segs := strings.Split(account, "/")
	projectId := ""
	if len(segs) == 2 {
		projectId = segs[1]
		account = segs[0]
	}

	client, err := ctyun.NewSCtyunClient(providerId, providerName, projectId, account, secret, false)
	if err != nil {
		return nil, err
	}
	return &SCtyunProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SCtyunProviderFactory) GetClientRC(url, account, secret string) (map[string]string, error) {
	return map[string]string{
		"CTYUN_ACCESS_URL": url,
		"CTYUN_ACCESS_KEY": account,
		"CTYUN_SECRET":     secret,
		"CTYUN_REGION":     ctyun.CTYUN_DEFAULT_REGION,
	}, nil
}

func init() {
	factory := SCtyunProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SCtyunProvider struct {
	cloudprovider.SBaseProvider
	client *ctyun.SCtyunClient
}

func (self *SCtyunProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SCtyunProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SCtyunProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SCtyunProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(ctyun.CTYUN_API_VERSION), "api_version")
	return info, nil
}

func (self *SCtyunProvider) GetVersion() string {
	return ctyun.CTYUN_API_VERSION
}

func (self *SCtyunProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(id)
}

func (self *SCtyunProvider) GetBalance() (float64, string, error) {
	return 0.0, api.CLOUD_PROVIDER_HEALTH_NORMAL, cloudprovider.ErrNotSupported
}

func (self *SCtyunProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SCtyunProvider) GetStorageClasses(regionId string) []string {
	return []string{
		"STANDARD", "WARM", "COLD",
	}
}

func (self *SCtyunProvider) GetCloudRegionExternalIdPrefix() string {
	return self.client.GetCloudRegionExternalIdPrefix()
}

func (self *SCtyunProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}
