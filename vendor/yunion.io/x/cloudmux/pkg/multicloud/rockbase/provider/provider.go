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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/rockbase"
)

// tag:finished
type SRockbaseProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactory
}

func (self *SRockbaseProviderFactory) GetId() string {
	return rockbase.CLOUD_PROVIDER_ROCKBASE
}

func (self *SRockbaseProviderFactory) GetName() string {
	return rockbase.CLOUD_PROVIDER_ROCKBASE_CN
}

func (self *SRockbaseProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_secret")
	}
	output.Account = input.AccessKeyId
	output.Secret = input.AccessKeySecret
	return output, nil
}

func (self *SRockbaseProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "access_key_secret")
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

func (self *SRockbaseProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	accessKey, projectId := parseAccount(cfg.Account)
	client, err := rockbase.NewRockbaseClient(
		rockbase.NewRockbaseClientConfig(
			accessKey, cfg.Secret,
		).ProjectId(projectId).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SRockbaseProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SRockbaseProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	accessKey, projectId := parseAccount(info.Account)
	return map[string]string{
		"ROCKBASE_ACCESS_KEY_ID":     accessKey,
		"ROCKBASE_ACCESS_KEY_SECRET": info.Secret,
		"ROCKBASE_REGION_ID":         rockbase.ROCKBASE_DEFAULT_REGION,
		"ROCKBASE_PROJECT_ID":        projectId,
	}, nil
}

func init() {
	factory := SRockbaseProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SRockbaseProvider struct {
	cloudprovider.SBaseProvider
	client *rockbase.SRockbaseClient
}

func (self *SRockbaseProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
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

func (self *SRockbaseProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions, _ := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(rockbase.ROCKBASE_API_VERSION), "api_version")
	return info, nil
}

func (self *SRockbaseProvider) GetVersion() string {
	return rockbase.ROCKBASE_API_VERSION
}

func (self *SRockbaseProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SRockbaseProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SRockbaseProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegions()
}

func (self *SRockbaseProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SRockbaseProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	return self.client.GetBalance()
}

func (self *SRockbaseProvider) GetOnPremiseIRegion() (cloudprovider.ICloudRegion, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRockbaseProvider) GetStorageClasses(regionId string) []string {
	return []string{
		"STANDARD", "IA", "ARCHIVE",
	}
}

func (self *SRockbaseProvider) GetBucketCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLPublicRead),
	}
}

func (self *SRockbaseProvider) GetObjectCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLPublicRead),
	}
}

func (self *SRockbaseProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SRockbaseProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return self.client.GetMetrics(opts)
}
