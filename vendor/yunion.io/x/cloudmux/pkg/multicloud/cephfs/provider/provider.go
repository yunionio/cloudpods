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
	"yunion.io/x/pkg/util/regutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/cephfs"
)

type SCephFSProviderFactory struct {
	cloudprovider.SPremiseBaseProviderFactory
}

func (self *SCephFSProviderFactory) GetId() string {
	return cephfs.CLOUD_PROVIDER_CEPHFS
}

func (self *SCephFSProviderFactory) GetName() string {
	return cephfs.CLOUD_PROVIDER_CEPHFS
}

func (self *SCephFSProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return fmt.Errorf("Changing %s bandwidth is not supported", cephfs.CLOUD_PROVIDER_CEPHFS)
}

func (self *SCephFSProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
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
	output.AccessUrl = fmt.Sprintf("https://%s:%d/api", input.Host, input.Port)
	if input.Port == 443 {
		output.AccessUrl = fmt.Sprintf("https://%s/api", input.Host)
	}
	output.Account = input.Username
	output.Secret = input.Password
	return output, nil
}

func (self *SCephFSProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
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
	if len(input.Host) > 0 {
		if !regutils.MatchIPAddr(input.Host) && !regutils.MatchDomainName(input.Host) {
			return output, errors.Wrap(cloudprovider.ErrInputParameter, "host should be ip or domain name")
		}
		output.AccessUrl = fmt.Sprintf("https://%s:%d/api", input.Host, input.Port)
		if input.Port == 443 {
			output.AccessUrl = fmt.Sprintf("https://%s/api", input.Host)
		}
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

func (self *SCephFSProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	parts, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, err
	}
	host, port, err := parseHostPort(parts.Host, 8443)
	if err != nil {
		return nil, err
	}

	account, fsId := cfg.Account, ""
	if idx := strings.Index(account, "/"); idx > 0 {
		account, fsId = cfg.Account[:idx], cfg.Account[idx+1:]
	}

	client, err := cephfs.NewCephFSClient(
		cephfs.NewCephFSClientConfig(
			host, port, account, cfg.Secret, fsId,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return &SCephFSProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SCephFSProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	parts, err := url.Parse(info.Url)
	if err != nil {
		return nil, err
	}
	host, port, err := parseHostPort(parts.Host, 443)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"CEPHFS_HOST":     host,
		"CEPHFS_PORT":     fmt.Sprintf("%d", port),
		"CEPHFS_USERNAME": info.Account,
		"CEPHFS_PASSWORD": info.Secret,
	}, nil
}

func (self *SCephFSProviderFactory) GetAccountIdEqualizer() func(origin, now string) bool {
	return func(origin, now string) bool {
		if len(now) == 0 {
			return true
		}
		originUserName, nowUserName := origin, now
		index1 := strings.Index(origin, "@")
		index2 := strings.Index(now, "@")
		if index1 != -1 {
			originUserName = originUserName[:index1]
		}
		if index2 != -1 {
			nowUserName = nowUserName[:index2]
		}
		return originUserName == nowUserName
	}
}

func init() {
	factory := SCephFSProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SCephFSProvider struct {
	cloudprovider.SBaseProvider
	client *cephfs.SCephFSClient
}

func (self *SCephFSProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return jsonutils.NewDict(), nil
}

func (self *SCephFSProvider) GetVersion() string {
	return "v1.0"
}

func (self *SCephFSProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SCephFSProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SCephFSProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SCephFSProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SCephFSProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	return &cloudprovider.SBalanceInfo{
		Amount:   0.0,
		Currency: "CNY",
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}, cloudprovider.ErrNotSupported
}

func (self *SCephFSProvider) GetOnPremiseIRegion() (cloudprovider.ICloudRegion, error) {
	return self.client, nil
}

func (self *SCephFSProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return []cloudprovider.ICloudProject{}, nil
}

func (self *SCephFSProvider) GetStorageClasses(regionId string) []string {
	return nil
}

func (self *SCephFSProvider) GetBucketCannedAcls(regionId string) []string {
	return nil
}

func (self *SCephFSProvider) GetObjectCannedAcls(regionId string) []string {
	return nil
}

func (self *SCephFSProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SCephFSProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return nil, cloudprovider.ErrNotSupported
}
