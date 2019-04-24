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
	"yunion.io/x/onecloud/pkg/util/ucloud"
)

// tag:finished
type SUcloudProviderFactory struct {
}

func (self *SUcloudProviderFactory) GetId() string {
	return ucloud.CLOUD_PROVIDER_UCLOUD
}

func (self *SUcloudProviderFactory) GetName() string {
	return ucloud.CLOUD_PROVIDER_UCLOUD_CN
}

func (self *SUcloudProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return nil
}

func (self *SUcloudProviderFactory) IsPublicCloud() bool {
	return true
}

func (self *SUcloudProviderFactory) IsOnPremise() bool {
	return false
}

func (self *SUcloudProviderFactory) IsSupportPrepaidResources() bool {
	return true
}

func (self *SUcloudProviderFactory) NeedSyncSkuFromCloud() bool {
	return false
}

func (self *SUcloudProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) error {
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

func (self *SUcloudProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, cloudaccount string) (*cloudprovider.SCloudaccount, error) {

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

func (self *SUcloudProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	client, err := ucloud.NewUcloudClient(providerId, providerName, account, secret, false)
	if err != nil {
		return nil, err
	}
	return &SUcloudProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func init() {
	factory := SUcloudProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SUcloudProvider struct {
	cloudprovider.SBaseProvider
	client *ucloud.SUcloudClient
}

func (self *SUcloudProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	projects, err := self.client.FetchProjects()
	if err != nil {
		return nil, err
	}

	iprojects := make([]cloudprovider.ICloudProject, len(projects))
	for i := range projects {
		iprojects[i] = &projects[i]
	}

	return iprojects, nil
}

func (self *SUcloudProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(ucloud.UCLOUD_API_VERSION), "api_version")
	return info, nil
}

func (self *SUcloudProvider) GetVersion() string {
	return ucloud.UCLOUD_API_VERSION
}

func (self *SUcloudProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SUcloudProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SUcloudProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SUcloudProvider) GetBalance() (float64, string, error) {
	return 0.0, api.CLOUD_PROVIDER_HEALTH_NORMAL, cloudprovider.ErrNotSupported
}

func (self *SUcloudProvider) GetOnPremiseIRegion() (cloudprovider.ICloudRegion, error) {
	return nil, cloudprovider.ErrNotImplemented
}
