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

package jdcloud

import (
	"github.com/jdcloud-api/jdcloud-sdk-go/core"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SJDCloudClient struct {
	*JDCloudClientConfig

	iregion []cloudprovider.ICloudRegion
}

type JDCloudClientConfig struct {
	cpcfg        cloudprovider.ProviderConfig
	accessKey    string
	accessSecret string
	debug        bool
}

func NewJDCloudClientConfig(accessKey, accessSecret string) *JDCloudClientConfig {
	cfg := &JDCloudClientConfig{
		accessKey:    accessKey,
		accessSecret: accessSecret,
	}
	return cfg
}

func (cfg *JDCloudClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *JDCloudClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *JDCloudClientConfig) Debug(debug bool) *JDCloudClientConfig {
	cfg.debug = debug
	return cfg
}

func (self *SJDCloudClient) getCredential() *core.Credential {
	return core.NewCredentials(self.accessKey, self.accessSecret)
}

func NewJDCloudClient(cfg *JDCloudClientConfig) (*SJDCloudClient, error) {
	client := SJDCloudClient{
		JDCloudClientConfig: cfg,
	}
	_, err := client.DescribeAccountAmount()
	return &client, err
}

func (self *SJDCloudClient) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	if self.iregion != nil {
		return self.iregion, nil
	}

	self.iregion = []cloudprovider.ICloudRegion{}
	for id, name := range regionList {
		self.iregion = append(self.iregion, &SRegion{
			ID:     id,
			Name:   name,
			client: self,
		})
	}
	return self.iregion, nil
}

func (self *SJDCloudClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Id = self.GetAccountId()
	subAccount.Name = self.cpcfg.Name
	subAccount.Account = self.accessKey
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SJDCloudClient) GetAccountId() string {
	return self.accessKey
}

func (self *SJDCloudClient) GetRegion(regionId string) (*SRegion, error) {
	iregions, err := self.GetIRegions()
	if err != nil {
		return nil, err
	}
	for i := range iregions {
		if len(regionId) == 0 || iregions[i].GetId() == regionId {
			return iregions[i].(*SRegion), nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}
