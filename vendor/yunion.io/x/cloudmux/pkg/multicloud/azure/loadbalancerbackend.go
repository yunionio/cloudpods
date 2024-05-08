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

package azure

import (
	"context"
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SLoadbalancerBackend struct {
	multicloud.SResourceBase
	AzureTags

	lbbg                         *SLoadbalancerBackendGroup
	PrivateIPAddress             string
	LoadBalancerBackendAddresses []string
}

func (self *SLoadbalancerBackend) GetId() string {
	return self.PrivateIPAddress + strings.Join(self.LoadBalancerBackendAddresses, ",")
}

func (self *SLoadbalancerBackend) GetName() string {
	return self.PrivateIPAddress + strings.Join(self.LoadBalancerBackendAddresses, ",")
}

func (self *SLoadbalancerBackend) GetGlobalId() string {
	return strings.ToLower(self.GetId())
}

func (self *SLoadbalancerBackend) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SLoadbalancerBackend) GetProjectId() string {
	return ""
}

func (self *SLoadbalancerBackend) GetWeight() int {
	return 0
}

func (self *SLoadbalancerBackend) GetPort() int {
	return 0
}

func (self *SLoadbalancerBackend) GetBackendType() string {
	return api.LB_BACKEND_IP
}

func (self *SLoadbalancerBackend) GetBackendRole() string {
	return api.LB_BACKEND_ROLE_DEFAULT
}

func (self *SLoadbalancerBackend) GetBackendId() string {
	return ""
}

func (self *SLoadbalancerBackend) GetIpAddress() string {
	return self.PrivateIPAddress + strings.Join(self.LoadBalancerBackendAddresses, ",")
}

func (self *SLoadbalancerBackend) SyncConf(ctx context.Context, port, weight int) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SyncConf")
}
