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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/multicloud/openstack"
)

type SOpenStackProviderFactory struct {
	cloudprovider.SPrivateCloudBaseProviderFactor
}

var EndpointTypes = []string{"admin", "internal", "public"}

func (self *SOpenStackProviderFactory) GetId() string {
	return openstack.CLOUD_PROVIDER_OPENSTACK
}

func (self *SOpenStackProviderFactory) GetName() string {
	return openstack.CLOUD_PROVIDER_OPENSTACK
}

func (self *SOpenStackProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, input *api.CloudaccountCreateInput) error {
	if len(input.ProjectName) == 0 {
		return httperrors.NewMissingParameterError("project_name")
	}
	if len(input.Username) == 0 {
		return httperrors.NewMissingParameterError("username")
	}
	if len(input.Password) == 0 {
		return httperrors.NewMissingParameterError("password")
	}
	if len(input.AuthUrl) == 0 {
		return httperrors.NewMissingParameterError("auth_url")
	}

	input.Account = fmt.Sprintf("%s/%s", input.ProjectName, input.Username)
	if len(input.DomainName) > 0 {
		input.Account = fmt.Sprintf("%s/%s", input.Account, input.DomainName)
	}

	input.Secret = input.Password
	input.AccessUrl = input.AuthUrl
	return nil
}

func (self *SOpenStackProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, input *api.CloudaccountCredentialInput, cloudaccount string) (*cloudprovider.SCloudaccount, error) {
	if len(input.ProjectName) == 0 {
		accountInfo := strings.Split(cloudaccount, "/")
		if len(accountInfo) < 2 {
			return nil, httperrors.NewMissingParameterError("project_name")
		}
		input.ProjectName = accountInfo[0]
	}
	if len(input.Username) == 0 {
		return nil, httperrors.NewMissingParameterError("username")
	}
	if len(input.Password) == 0 {
		return nil, httperrors.NewMissingParameterError("password")
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

	account := &cloudprovider.SCloudaccount{
		Account: _account,
		Secret:  input.Password,
	}
	return account, nil
}

func (self *SOpenStackProviderFactory) GetProvider(providerId, providerName, url, account, password string) (cloudprovider.ICloudProvider, error) {
	accountInfo := strings.Split(account, "/")
	if len(accountInfo) < 2 {
		return nil, fmt.Errorf("Missing username or project name %s", account)
	}
	project, username, endpointType, domainName, projectDomainName := accountInfo[0], accountInfo[1], "internal", "Default", "Default"
	if len(accountInfo) == 3 {
		domainName, projectDomainName = accountInfo[2], accountInfo[2]
	}
	client, err := openstack.NewOpenStackClient(providerId, providerName, url, username, password, project, endpointType, domainName, projectDomainName, false)
	if err != nil {
		return nil, err
	}
	return &SOpenStackProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SOpenStackProviderFactory) GetClientRC(url, account, secret string) (map[string]string, error) {
	accountInfo := strings.Split(account, "/")
	if len(accountInfo) < 2 {
		return nil, fmt.Errorf("Missing username or project name %s", account)
	}
	project, username, endpointType, domainName, projectDomainName := accountInfo[0], accountInfo[1], "internal", "Default", "Default"
	if len(accountInfo) == 3 {
		domainName, projectDomainName = accountInfo[2], accountInfo[2]
	}

	return map[string]string{
		"OPENSTACK_AUTH_URL":       url,
		"OPENSTACK_USERNAME":       username,
		"OPENSTACK_PASSWORD":       secret,
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

func (self *SOpenStackProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SOpenStackProvider) GetIGlobalnetworks() ([]cloudprovider.ICloudGlobalnetwork, error) {
	return []cloudprovider.ICloudGlobalnetwork{}, nil
}

func (self *SOpenStackProvider) GetIGlobalnetworkById(id string) (cloudprovider.ICloudGlobalnetwork, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SOpenStackProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SOpenStackProvider) GetBalance() (float64, string, error) {
	return 0.0, api.CLOUD_PROVIDER_HEALTH_UNKNOWN, cloudprovider.ErrNotSupported
}

func (self *SOpenStackProvider) GetCloudRegionExternalIdPrefix() string {
	return self.client.GetCloudRegionExternalIdPrefix()
}

func (self *SOpenStackProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}

func (self *SOpenStackProvider) GetStorageClasses(regionId string) []string {
	return nil
}
