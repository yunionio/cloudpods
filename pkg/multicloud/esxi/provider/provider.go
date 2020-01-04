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
	"net/url"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/multicloud/esxi"
)

type SESXiProviderFactory struct {
	cloudprovider.SPremiseBaseProviderFactory
}

func (self *SESXiProviderFactory) GetId() string {
	return esxi.CLOUD_PROVIDER_VMWARE
}

func (self *SESXiProviderFactory) GetName() string {
	return esxi.CLOUD_PROVIDER_VMWARE
}

func (self *SESXiProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return fmt.Errorf("Changing %s bandwidth is not supported", esxi.CLOUD_PROVIDER_VMWARE)
}

func (self *SESXiProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.Username) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "username")
	}
	if len(input.Password) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "password")
	}
	if len(input.Host) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "host")
	}
	output.AccessUrl = fmt.Sprintf("https://%s:%d/sdk", input.Host, input.Port)
	if input.Port == 0 || input.Port == 443 {
		output.AccessUrl = fmt.Sprintf("https://%s/sdk", input.Host)
	}
	output.Account = input.Username
	output.Secret = input.Password
	return output, nil
}

func (self *SESXiProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.Username) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "username")
	}
	if len(input.Password) == 0 {
		return output, errors.Wrap(httperrors.ErrMissingParameter, "password")
	}
	output = cloudprovider.SCloudaccount{
		Account: input.Username,
		Secret:  input.Password,
	}
	return output, nil
}

func parseHostPort(host string, defPort int) (string, int, error) {
	colonPos := strings.IndexByte(host, ':')
	if colonPos > 0 {
		h := host[:colonPos]
		p, err := strconv.Atoi(host[colonPos+1:])
		if err != nil {
			log.Errorf("Invalid host %s", host)
			return "", 0, err
		}
		if p == 0 {
			p = defPort
		}
		return h, p, nil
	} else {
		return host, defPort, nil
	}
}

func (self *SESXiProviderFactory) GetProvider(providerId, providerName, urlStr, account, secret string) (cloudprovider.ICloudProvider, error) {
	parts, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	host, port, err := parseHostPort(parts.Host, 443)
	if err != nil {
		return nil, err
	}

	client, err := esxi.NewESXiClient(providerId, providerName, host, port, account, secret)
	if err != nil {
		return nil, err
	}
	return &SESXiProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SESXiProviderFactory) GetClientRC(urlStr, account, secret string) (map[string]string, error) {
	parts, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	host, port, err := parseHostPort(parts.Host, 443)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"VMWARE_HOST":     host,
		"VMWARE_PORT":     fmt.Sprintf("%d", port),
		"VMWARE_ACCOUNT":  account,
		"VMWARE_PASSWORD": secret,
	}, nil
}

func init() {
	factory := SESXiProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SESXiProvider struct {
	cloudprovider.SBaseProvider
	client *esxi.SESXiClient
}

func (self *SESXiProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return self.client.About(), nil
}

func (self *SESXiProvider) GetVersion() string {
	return self.client.GetVersion()
}

func (self *SESXiProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SESXiProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SESXiProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return nil
}

func (self *SESXiProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SESXiProvider) GetBalance() (float64, string, error) {
	return 0.0, api.CLOUD_PROVIDER_HEALTH_NORMAL, cloudprovider.ErrNotSupported
}

func (self *SESXiProvider) GetOnPremiseIRegion() (cloudprovider.ICloudRegion, error) {
	return self.client, nil
}

func (self *SESXiProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SESXiProvider) GetStorageClasses(regionId string) []string {
	return nil
}

func (self *SESXiProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}
