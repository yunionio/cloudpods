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

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/openstack"
)

type SOpenStackProviderFactory struct {
	cloudprovider.SPrivateCloudBaseProviderFactory
}

var EndpointTypes = []string{"admin", "internal", "public"}

func (self *SOpenStackProviderFactory) GetId() string {
	return openstack.CLOUD_PROVIDER_OPENSTACK
}

func (self *SOpenStackProviderFactory) GetName() string {
	return openstack.CLOUD_PROVIDER_OPENSTACK
}

func (self *SOpenStackProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.ProjectName) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "project_name")
	}
	if len(input.Username) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "username")
	}
	if len(input.Password) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "password")
	}
	if len(input.AuthUrl) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "auth_url")
	}

	output.Account = fmt.Sprintf("%s/%s", input.ProjectName, input.Username)
	if len(input.DomainName) > 0 {
		output.Account = fmt.Sprintf("%s/%s", output.Account, input.DomainName)
	}

	output.Secret = input.Password
	output.AccessUrl = input.AuthUrl
	return output, nil
}

func (self *SOpenStackProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.ProjectName) == 0 {
		accountInfo := strings.Split(cloudaccount, "/")
		if len(accountInfo) < 2 {
			return output, errors.Wrap(cloudprovider.ErrMissingParameter, "project_name")
		}
		input.ProjectName = accountInfo[0]
	}
	if len(input.Username) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "username")
	}
	if len(input.Password) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "password")
	}

	_account := fmt.Sprintf("%s/%s", input.ProjectName, input.Username)
	if len(input.DomainName) == 0 {
		if accountInfo := strings.Split(cloudaccount, "/"); len(accountInfo) == 3 {
			input.DomainName = accountInfo[2]
		}
	}

	if len(input.DomainName) > 0 {
		_account = fmt.Sprintf("%s/%s", _account, input.DomainName)
	}

	output = cloudprovider.SCloudaccount{
		Account: _account,
		Secret:  input.Password,
	}
	return output, nil
}

func (self *SOpenStackProviderFactory) IsNeedForceAutoCreateProject() bool {
	return true
}

func (self *SOpenStackProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	accountInfo := strings.Split(cfg.Account, "/")
	if len(accountInfo) < 2 {
		return nil, fmt.Errorf("Missing username or project name %s", cfg.Account)
	}
	project, username, endpointType, domainName, projectDomainName := accountInfo[0], accountInfo[1], "internal", "Default", "Default"
	if len(accountInfo) == 3 {
		domainName, projectDomainName = accountInfo[2], accountInfo[2]
	}
	client, err := openstack.NewOpenStackClient(
		openstack.NewOpenstackClientConfig(
			cfg.URL,
			username,
			cfg.Secret,
			project,
			projectDomainName,
		).
			DomainName(domainName).
			EndpointType(endpointType).
			CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SOpenStackProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SOpenStackProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	accountInfo := strings.Split(info.Account, "/")
	if len(accountInfo) < 2 {
		return nil, fmt.Errorf("Missing username or project name %s", info.Account)
	}
	project, username, endpointType, domainName, projectDomainName := accountInfo[0], accountInfo[1], "internal", "Default", "Default"
	if len(accountInfo) == 3 {
		domainName, projectDomainName = accountInfo[2], accountInfo[2]
	}

	return map[string]string{
		"OPENSTACK_AUTH_URL":       info.Url,
		"OPENSTACK_USERNAME":       username,
		"OPENSTACK_PASSWORD":       info.Secret,
		"OPENSTACK_PROJECT":        project,
		"OPENSTACK_ENDPOINT_TYPE":  endpointType,
		"OPENSTACK_DOMAIN_NAME":    domainName,
		"OPENSTACK_PROJECT_DOMAIN": projectDomainName,
		"OPENSTACK_REGION_ID":      openstack.OPENSTACK_DEFAULT_REGION,
	}, nil
}

func init() {
	factory := SOpenStackProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SOpenStackProvider struct {
	cloudprovider.SBaseProvider
	client *openstack.SOpenStackClient
}

func (self *SOpenStackProvider) GetVersion() string {
	return ""
}

func (self *SOpenStackProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return jsonutils.NewDict(), nil
}

func (self *SOpenStackProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SOpenStackProvider) GetAccountId() string {
	return ""
}

func (self *SOpenStackProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegions()
}

func (self *SOpenStackProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SOpenStackProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	return &cloudprovider.SBalanceInfo{
		Amount:   0.0,
		Currency: "CNY",
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}, cloudprovider.ErrNotSupported
}

func (self *SOpenStackProvider) GetCloudRegionExternalIdPrefix() string {
	return self.client.GetCloudRegionExternalIdPrefix()
}

func (self *SOpenStackProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SOpenStackProvider) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return self.client.CreateIProject(name)
}

func (self *SOpenStackProvider) GetStorageClasses(regionId string) []string {
	return nil
}

func (self *SOpenStackProvider) GetBucketCannedAcls(regionId string) []string {
	return nil
}

func (self *SOpenStackProvider) GetObjectCannedAcls(regionId string) []string {
	return nil
}

func (self *SOpenStackProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}
