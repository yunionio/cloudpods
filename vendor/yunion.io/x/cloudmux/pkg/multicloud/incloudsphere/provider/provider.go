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
	"yunion.io/x/pkg/util/regutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/cloudmux/pkg/multicloud/incloudsphere"
)

type SInCloudSphereProviderFactory struct {
	cloudprovider.SPrivateCloudBaseProviderFactory
}

func (self *SInCloudSphereProviderFactory) GetId() string {
	return incloudsphere.CLOUD_PROVIDER_INCLOUD_SPHERE
}

func (self *SInCloudSphereProviderFactory) GetName() string {
	return incloudsphere.CLOUD_PROVIDER_INCLOUD_SPHERE
}

func (self *SInCloudSphereProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return fmt.Errorf("Changing %s bandwidth is not supported", incloudsphere.CLOUD_PROVIDER_INCLOUD_SPHERE)
}

func (self *SInCloudSphereProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.AccessKeyId) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_id")
	}
	if len(input.AccessKeySecret) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "access_key_secret")
	}
	if len(input.Host) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "host")
	}
	input.Host = strings.TrimPrefix(input.Host, "https://")
	input.Host = strings.TrimPrefix(input.Host, "http://")
	if !regutils.MatchIPAddr(input.Host) && !regutils.MatchDomainName(input.Host) {
		return output, errors.Wrap(httperrors.ErrInputParameter, "host should be ip or domain name")
	}
	output.AccessUrl = input.Host
	output.Account = input.AccessKeyId
	output.Secret = input.AccessKeySecret
	return output, nil
}

func (self *SInCloudSphereProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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

func (self *SInCloudSphereProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	client, err := incloudsphere.NewSphereClient(
		incloudsphere.NewSphereClientConfig(
			cfg.URL, cfg.Account, cfg.Secret,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SInCloudSphereProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SInCloudSphereProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	return map[string]string{
		"INCLOUD_SPHERE_HOST":              info.Url,
		"INCLOUD_SPHERE_ACCESS_KEY_ID":     info.Account,
		"INCLOUD_SPHERE_ACCESS_KEY_SECRET": info.Secret,
	}, nil
}

func init() {
	factory := SInCloudSphereProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SInCloudSphereProvider struct {
	cloudprovider.SBaseProvider
	client *incloudsphere.SphereClient
}

func (self *SInCloudSphereProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return jsonutils.NewDict(), nil
}

func (self *SInCloudSphereProvider) GetVersion() string {
	return "5.8"
}

func (self *SInCloudSphereProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SInCloudSphereProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SInCloudSphereProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SInCloudSphereProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	regions := self.GetIRegions()
	for i := range regions {
		if regions[i].GetGlobalId() == id {
			return regions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SInCloudSphereProvider) GetBalance() (float64, string, error) {
	return 0.0, api.CLOUD_PROVIDER_HEALTH_NORMAL, cloudprovider.ErrNotSupported
}

func (self *SInCloudSphereProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return []cloudprovider.ICloudProject{}, nil
}

func (self *SInCloudSphereProvider) GetStorageClasses(regionId string) []string {
	return nil
}

func (self *SInCloudSphereProvider) GetBucketCannedAcls(regionId string) []string {
	return nil
}

func (self *SInCloudSphereProvider) GetObjectCannedAcls(regionId string) []string {
	return nil
}

func (self *SInCloudSphereProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}
