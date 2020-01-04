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
	"yunion.io/x/onecloud/pkg/multicloud/ucloud"
)

// tag:finished
type SUcloudProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactor
}

func (self *SUcloudProviderFactory) GetId() string {
	return ucloud.CLOUD_PROVIDER_UCLOUD
}

func (self *SUcloudProviderFactory) GetName() string {
	return ucloud.CLOUD_PROVIDER_UCLOUD_CN
}

func (self *SUcloudProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_secret")
	}
	output.Account = input.AccessKeyId
	output.Secret = input.AccessKeySecret
	return output, nil
}

func (self *SUcloudProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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
	segs := strings.Split(account, "::")
	if len(segs) == 2 {
		accessKey = segs[0]
		projectId = segs[1]
	} else {
		accessKey = account
		projectId = ""
	}

	return
}

func (self *SUcloudProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	accessKey, projectId := parseAccount(account)
	client, err := ucloud.NewUcloudClient(providerId, providerName, accessKey, secret, projectId, false)
	if err != nil {
		return nil, err
	}
	return &SUcloudProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SUcloudProviderFactory) GetClientRC(url, account, secret string) (map[string]string, error) {
	accessKey, projectId := parseAccount(account)
	return map[string]string{
		"UCLOUD_ACCESS_KEY": accessKey,
		"UCLOUD_SECRET":     secret,
		"UCLOUD_REGION":     ucloud.UCLOUD_DEFAULT_REGION,
		"UCLOUD_PROJECT":    projectId,
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

func (self *SUcloudProvider) GetAccountId() string {
	return self.client.GetAccountId()
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

func (self *SUcloudProvider) GetStorageClasses(regionId string) []string {
	return []string{
		"STANDARD", "IA", "ARCHIVE",
	}
}

func (self *SUcloudProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}
