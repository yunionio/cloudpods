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
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/objectstore/ceph"
	s3provider "yunion.io/x/onecloud/pkg/multicloud/objectstore/provider"
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

func (self *SCephRadosProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	client, err := ceph.NewCephRados(providerId, providerName, url, account, secret, false)
	if err != nil {
		return nil, err
	}
	return s3provider.NewObjectStoreProvider(self, client), nil
}

func (self *SCephRadosProviderFactory) GetClientRC(url, account, secret string) (map[string]string, error) {
	return map[string]string{
		"S3_ACCESS_KEY": account,
		"S3_SECRET":     secret,
		"S3_ACCESS_URL": url,
		"S3_BACKEND":    api.CLOUD_PROVIDER_CEPH,
	}, nil
}

func init() {
	factory := SCephRadosProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}
