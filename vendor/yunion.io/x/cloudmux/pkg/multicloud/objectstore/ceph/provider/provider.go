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
	"yunion.io/x/cloudmux/pkg/multicloud/objectstore/ceph"
	s3provider "yunion.io/x/cloudmux/pkg/multicloud/objectstore/provider"
)

type SCephRadosProviderFactory struct {
	s3provider.SObjectStoreProviderFactory
}

func (self *SCephRadosProviderFactory) GetId() string {
	return api.CLOUD_PROVIDER_CEPH
}

func (self *SCephRadosProviderFactory) GetName() string {
	return api.CLOUD_PROVIDER_CEPH
}

func (self *SCephRadosProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	client, err := ceph.NewCephRados(
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

func (self *SCephRadosProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	return map[string]string{
		"S3_ACCESS_KEY": info.Account,
		"S3_SECRET":     info.Secret,
		"S3_ACCESS_URL": info.Url,
		"S3_BACKEND":    api.CLOUD_PROVIDER_CEPH,
	}, nil
}

func init() {
	factory := SCephRadosProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}
