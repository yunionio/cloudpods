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

package proxmox

import "yunion.io/x/onecloud/pkg/cloudprovider"

type ProxmoxClient struct {
	*ProxmoxClientConfig
}

type ProxmoxClientConfig struct {
	cpcfg    cloudprovider.ProviderConfig
	username string
	password string
	host     string
	port     int
	debug    bool
}

func NewProxmoxClientConfig(host, username, password string, port int) *ProxmoxClientConfig {
	cfg := &ProxmoxClientConfig{
		host:     host,
		username: username,
		password: password,
		port:     port,
	}
	return cfg
}

func (self *ProxmoxClientConfig) Debug(debug bool) *ProxmoxClientConfig {
	self.debug = debug
	return self
}

func (self *ProxmoxClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *ProxmoxClientConfig {
	self.cpcfg = cpcfg
	return self
}

func NewProxmoxClient(cfg *ProxmoxClientConfig) (*ProxmoxClient, error) {
	client := &ProxmoxClient{
		ProxmoxClientConfig: cfg,
	}

	return client, client.auth()
}

func (self *ProxmoxClient) auth() error {
	return cloudprovider.ErrNotImplemented
}

func (self *ProxmoxClient) GetRegion() (*SRegion, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *ProxmoxClient) GetRegions() ([]SRegion, error) {
	return nil, cloudprovider.ErrNotImplemented
}
