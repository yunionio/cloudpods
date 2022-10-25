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
	"yunion.io/x/onecloud/pkg/multicloud/hcs"
)

type SHcsProviderFactory struct {
	cloudprovider.SPrivateCloudBaseProviderFactory
}

func (self *SHcsProviderFactory) GetId() string {
	return hcs.CLOUD_PROVIDER_HCS
}

func (self *SHcsProviderFactory) GetName() string {
	return hcs.CLOUD_PROVIDER_HCS_CN
}

func (self *SHcsProviderFactory) IsCloudeventRegional() bool {
	return true
}

func (self *SHcsProviderFactory) GetMaxCloudEventSyncDays() int {
	return 7
}

func (self *SHcsProviderFactory) GetMaxCloudEventKeepDays() int {
	return 7
}

func (factory *SHcsProviderFactory) IsSupportModifyRouteTable() bool {
	return true
}

func (factory *SHcsProviderFactory) IsSupportSAMLAuth() bool {
	return false
}

func (self *SHcsProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_secret")
	}
	if len(input.AuthUrl) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "auth_url")
	}
	output.Account = input.AccessKeyId
	output.Secret = input.AccessKeySecret
	output.AccessUrl = input.AuthUrl
	return output, nil
}

func (self *SHcsProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func (self *SHcsProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	accessKey, projectId := parseAccount(cfg.Account)
	client, err := hcs.NewHcsClient(
		hcs.NewHcsConfig(
			accessKey, cfg.Secret, projectId, cfg.URL,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SHcsProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SHcsProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	accessKey, projectId := parseAccount(info.Account)
	return map[string]string{
		"HCS_AUTH_URL":   info.Url,
		"HCS_ACCESS_KEY": accessKey,
		"HCS_SECRET":     info.Secret,
		"HCS_PROJECT_ID": projectId,
	}, nil
}

func init() {
	factory := SHcsProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SHcsProvider struct {
	cloudprovider.SBaseProvider
	client *hcs.SHcsClient
}

func (self *SHcsProvider) GetVersion() string {
	return ""
}

func (self *SHcsProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(hcs.HCS_API_VERSION), "api_version")
	return info, nil
}

func (self *SHcsProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SHcsProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	region, err := self.client.GetRegion(id)
	if err != nil {
		return nil, err
	}
	return region, nil
}

func (self *SHcsProvider) GetBalance() (float64, string, error) {
	return 0, api.CLOUD_PROVIDER_HEALTH_NORMAL, nil
}

func (self *SHcsProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SHcsProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SHcsProvider) GetCloudRegionExternalIdPrefix() string {
	return self.client.GetCloudRegionExternalIdPrefix()
}

func (self *SHcsProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return []cloudprovider.ICloudProject{}, nil
}

func (self *SHcsProvider) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SHcsProvider) GetStorageClasses(regionId string) []string {
	return []string{
		"STANDARD", "WARM", "COLD",
	}
}

func (self *SHcsProvider) GetBucketCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLAuthRead),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SHcsProvider) GetObjectCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLAuthRead),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SHcsProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}
