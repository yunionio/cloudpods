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
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/objectstore"
	s3provider "yunion.io/x/cloudmux/pkg/multicloud/objectstore/provider"
	"yunion.io/x/cloudmux/pkg/multicloud/objectstore/xsky"
)

type SXskyProviderFactory struct {
	s3provider.SObjectStoreProviderFactory
}

func (self *SXskyProviderFactory) GetId() string {
	return api.CLOUD_PROVIDER_XSKY
}

func (self *SXskyProviderFactory) GetName() string {
	return api.CLOUD_PROVIDER_XSKY
}

func (self *SXskyProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	client, err := xsky.NewXskyClient(
		objectstore.NewObjectStoreClientConfig(
			cfg.URL, cfg.Account, cfg.Secret,
		).CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}
	return s3provider.NewObjectStoreProvider(self, client, []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}), nil
}

func (self *SXskyProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	client, err := xsky.NewXskyClient(
		objectstore.NewObjectStoreClientConfig(
			info.Url, info.Account, info.Secret,
		),
	)
	if err != nil {
		return nil, err
	}
	return client.GetClientRC(), nil
}

func init() {
	factory := SXskyProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}
