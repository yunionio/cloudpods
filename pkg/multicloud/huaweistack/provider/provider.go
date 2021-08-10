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
	huawei "yunion.io/x/onecloud/pkg/multicloud/huaweistack"
)

type SHuaweiCloudStackProviderFactory struct {
	cloudprovider.SPrivateCloudBaseProviderFactory
}

func (self *SHuaweiCloudStackProviderFactory) GetId() string {
	return huawei.CLOUD_PROVIDER_HUAWEI
}

func (self *SHuaweiCloudStackProviderFactory) GetName() string {
	return huawei.CLOUD_PROVIDER_HUAWEI_CN
}

func (self *SHuaweiCloudStackProviderFactory) IsCloudeventRegional() bool {
	return true
}

func (self *SHuaweiCloudStackProviderFactory) GetMaxCloudEventSyncDays() int {
	return 7
}

func (self *SHuaweiCloudStackProviderFactory) GetMaxCloudEventKeepDays() int {
	return 7
}

func (self *SHuaweiCloudStackProviderFactory) IsSupportCloudIdService() bool {
	return true
}

func (self *SHuaweiCloudStackProviderFactory) IsSupportClouduserPolicy() bool {
	return false
}

func (self *SHuaweiCloudStackProviderFactory) IsSupportCreateCloudgroup() bool {
	return true
}

func (factory *SHuaweiCloudStackProviderFactory) IsSupportCrossCloudEnvVpcPeering() bool {
	return false
}

func (factory *SHuaweiCloudStackProviderFactory) IsSupportCrossRegionVpcPeering() bool {
	return false
}

func (factory *SHuaweiCloudStackProviderFactory) IsSupportVpcPeeringVpcCidrOverlap() bool {
	return true
}

func (factory *SHuaweiCloudStackProviderFactory) IsSupportModifyRouteTable() bool {
	return true
}

func (factory *SHuaweiCloudStackProviderFactory) IsSupportSAMLAuth() bool {
	return true
}

func (self *SHuaweiCloudStackProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_secret")
	}

	if input.SHuaweiCloudStackEndpoints == nil {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "cloud_stack_endpoints")
	}

	if len(input.SHuaweiCloudStackEndpoints.DefaultRegion) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "default_region")
	}

	if len(input.SHuaweiCloudStackEndpoints.EndpointDomain) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "endpoint_domain")
	}

	output.Account = input.AccessKeyId
	output.Secret = input.AccessKeySecret
	return output, nil
}

func (self *SHuaweiCloudStackProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_secret")
	}

	if input.SHuaweiCloudStackEndpoints != nil {
		if len(input.SHuaweiCloudStackEndpoints.DefaultRegion) == 0 {
			return output, errors.Wrap(httperrors.ErrMissingParameter, "default_region")
		}

		if len(input.SHuaweiCloudStackEndpoints.EndpointDomain) == 0 {
			return output, errors.Wrap(httperrors.ErrMissingParameter, "endpoint_domain")
		}
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

func (self *SHuaweiCloudStackProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	accessKey, project_id := parseAccount(cfg.Account)
	client, err := huawei.NewHuaweiClient(
		huawei.NewHuaweiClientConfig(
			accessKey, cfg.Secret, project_id, &cfg.SHuaweiCloudStackEndpoints,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SHuaweiCloudStackProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SHuaweiCloudStackProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	accessKey, projectId := parseAccount(info.Account)
	region := ""
	data := strings.Split(info.Name, "-")
	if len(data) >= 3 {
		region = strings.Join(data[2:], "-")
	}
	return map[string]string{
		"HUAWEI_CLOUD_ENV":  info.Url,
		"HUAWEI_ACCESS_KEY": accessKey,
		"HUAWEI_SECRET":     info.Secret,
		"HUAWEI_REGION":     region,
		"HUAWEI_PROJECT":    projectId,
	}, nil
}

func init() {
	factory := SHuaweiCloudStackProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SHuaweiCloudStackProvider struct {
	cloudprovider.SBaseProvider
	client *huawei.SHuaweiClient
}

func (self *SHuaweiCloudStackProvider) GetVersion() string {
	return self.client.GetVersion()
}

func (self *SHuaweiCloudStackProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(huawei.HUAWEI_API_VERSION), "api_version")
	return info, nil
}

func (self *SHuaweiCloudStackProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SHuaweiCloudStackProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SHuaweiCloudStackProvider) GetBalance() (float64, string, error) {
	return 0.0, api.CLOUD_PROVIDER_HEALTH_NORMAL, nil
}

func (self *SHuaweiCloudStackProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SHuaweiCloudStackProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SHuaweiCloudStackProvider) GetIamLoginUrl() string {
	return self.client.GetIamLoginUrl()
}

func (self *SHuaweiCloudStackProvider) GetCloudRegionExternalIdPrefix() string {
	return self.client.GetCloudRegionExternalIdPrefix()
}

func (self *SHuaweiCloudStackProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SHuaweiCloudStackProvider) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return self.client.CreateIProject(name)
}

func (self *SHuaweiCloudStackProvider) GetStorageClasses(regionId string) []string {
	return []string{
		"STANDARD", "WARM", "COLD",
	}
}

func (self *SHuaweiCloudStackProvider) GetBucketCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLAuthRead),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SHuaweiCloudStackProvider) GetObjectCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLAuthRead),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SHuaweiCloudStackProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SHuaweiCloudStackProvider) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	return self.client.CreateIClouduser(conf)
}

func (self *SHuaweiCloudStackProvider) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	return self.client.GetICloudusers()
}

func (self *SHuaweiCloudStackProvider) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	return self.client.GetICloudgroups()
}

func (self *SHuaweiCloudStackProvider) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	return self.client.GetICloudgroupByName(name)
}

func (self *SHuaweiCloudStackProvider) CreateICloudgroup(name, desc string) (cloudprovider.ICloudgroup, error) {
	return self.client.CreateICloudgroup(name, desc)
}

func (self *SHuaweiCloudStackProvider) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return self.client.GetISystemCloudpolicies()
}

func (self *SHuaweiCloudStackProvider) GetICustomCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return []cloudprovider.ICloudpolicy{}, nil
}

func (self *SHuaweiCloudStackProvider) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	return self.client.GetIClouduserByName(name)
}

func (self *SHuaweiCloudStackProvider) GetSamlEntityId() string {
	return cloudprovider.SAML_ENTITY_ID_HUAWEI_CLOUD
}

func (self *SHuaweiCloudStackProvider) GetICloudSAMLProviders() ([]cloudprovider.ICloudSAMLProvider, error) {
	return self.client.GetICloudSAMLProviders()
}

func (self *SHuaweiCloudStackProvider) CreateICloudSAMLProvider(opts *cloudprovider.SAMLProviderCreateOptions) (cloudprovider.ICloudSAMLProvider, error) {
	sp, err := self.client.CreateSAMLProvider(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSAMLProvider")
	}
	return sp, nil
}
