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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/nutanix"
)

type SNutanixProviderFactory struct {
	cloudprovider.SPrivateCloudBaseProviderFactory
}

func (self *SNutanixProviderFactory) GetId() string {
	return nutanix.CLOUD_PROVIDER_NUTANIX
}

func (self *SNutanixProviderFactory) GetName() string {
	return nutanix.CLOUD_PROVIDER_NUTANIX
}

func (self *SNutanixProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return fmt.Errorf("Changing %s bandwidth is not supported", nutanix.CLOUD_PROVIDER_NUTANIX)
}

func (self *SNutanixProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.Username) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "username")
	}
	if len(input.Password) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "password")
	}
	if len(input.Host) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "host")
	}
	if !regutils.MatchIPAddr(input.Host) && !regutils.MatchDomainName(input.Host) {
		return output, errors.Wrap(cloudprovider.ErrInputParameter, "host should be ip or domain name")
	}
	if input.Port == 0 {
		input.Port = 9440
	}
	output.AccessUrl = fmt.Sprintf("https://%s:%d", input.Host, input.Port)
	output.Account = input.Username
	output.Secret = input.Password
	return output, nil
}

func (self *SNutanixProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.Username) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "username")
	}
	if len(input.Password) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "password")
	}
	output = cloudprovider.SCloudaccount{
		Account: input.Username,
		Secret:  input.Password,
	}
	return output, nil
}

func parseHostPort(_url string) (string, int, error) {
	urlParse, err := url.Parse(_url)
	if err != nil {
		return "", 0, errors.Wrapf(err, "parse %s", _url)
	}
	port := func() int {
		if len(urlParse.Port()) > 0 {
			_port, _ := strconv.Atoi(urlParse.Port())
			return _port
		}
		return 9440
	}()
	return strings.TrimSuffix(urlParse.Host, fmt.Sprintf(":%d", port)), port, nil
}

func (self *SNutanixProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	host, port, err := parseHostPort(cfg.URL)
	if err != nil {
		return nil, errors.Wrapf(err, "parseHostPort")
	}

	client, err := nutanix.NewNutanixClient(
		nutanix.NewNutanixClientConfig(
			host, cfg.Account, cfg.Secret, port,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SNutanixProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SNutanixProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	host, port, err := parseHostPort(info.Url)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"NUTANIX_HOST":     host,
		"NUTANIX_PORT":     fmt.Sprintf("%d", port),
		"NUTANIX_USERNAME": info.Account,
		"NUTANIX_PASSWORD": info.Secret,
	}, nil
}

func init() {
	factory := SNutanixProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SNutanixProvider struct {
	cloudprovider.SBaseProvider
	client *nutanix.SNutanixClient
}

func (self *SNutanixProvider) GetCloudRegionExternalIdPrefix() string {
	return self.client.GetCloudRegionExternalIdPrefix()
}

func (self *SNutanixProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return jsonutils.NewDict(), nil
}

func (self *SNutanixProvider) GetVersion() string {
	return nutanix.NUTANIX_VERSION_V2
}

func (self *SNutanixProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SNutanixProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SNutanixProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegions()
}

func (self *SNutanixProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	regions, err := self.GetIRegions()
	if err != nil {
		return nil, err
	}
	for i := range regions {
		if regions[i].GetGlobalId() == id {
			return regions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SNutanixProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	return &cloudprovider.SBalanceInfo{
		Amount:   0.0,
		Currency: "CNY",
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}, cloudprovider.ErrNotSupported
}

func (self *SNutanixProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return []cloudprovider.ICloudProject{}, nil
}

func (self *SNutanixProvider) GetStorageClasses(regionId string) []string {
	return nil
}

func (self *SNutanixProvider) GetBucketCannedAcls(regionId string) []string {
	return nil
}

func (self *SNutanixProvider) GetObjectCannedAcls(regionId string) []string {
	return nil
}

func (self *SNutanixProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}
