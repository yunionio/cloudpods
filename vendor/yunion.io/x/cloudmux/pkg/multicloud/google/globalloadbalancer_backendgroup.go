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

package google

import (
	"context"
	"time"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SGlobalLoadBalancerBackendGroup struct {
	lb       *SGlobalLoadbalancer
	backends []SGlobalLoadbalancerBackend

	backendService SBackendServices //

	Id   string `json:"id"`
	Name string `json:"name"`
}

func (self *SGlobalLoadBalancerBackendGroup) GetId() string {
	return self.Id
}

func (self *SGlobalLoadBalancerBackendGroup) GetName() string {
	return self.Name
}

func (self *SGlobalLoadBalancerBackendGroup) GetGlobalId() string {
	return self.backendService.GetGlobalId()
}

func (self *SGlobalLoadBalancerBackendGroup) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SGlobalLoadBalancerBackendGroup) GetDescription() string {
	return ""
}

func (self *SGlobalLoadBalancerBackendGroup) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SGlobalLoadBalancerBackendGroup) Refresh() error {
	return nil
}

func (self *SGlobalLoadBalancerBackendGroup) IsEmulated() bool {
	return true
}

func (self *SGlobalLoadBalancerBackendGroup) GetSysTags() map[string]string {
	return nil
}

func (self *SGlobalLoadBalancerBackendGroup) GetTags() (map[string]string, error) {
	return nil, nil
}

func (self *SGlobalLoadBalancerBackendGroup) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotSupported
}

func (self *SGlobalLoadBalancerBackendGroup) IsDefault() bool {
	return false
}

func (self *SGlobalLoadBalancerBackendGroup) GetType() string {
	return api.LB_BACKENDGROUP_TYPE_NORMAL
}

func (self *SGlobalLoadBalancerBackendGroup) GetILoadbalancerBackendById(backendId string) (cloudprovider.ICloudLoadbalancerBackend, error) {
	backends, err := self.GetLoadbalancerBackends()
	if err != nil {
		return nil, errors.Wrap(err, "GetLoadbalancerBackends")
	}

	for i := range backends {
		if backends[i].GetGlobalId() == backendId {
			return &backends[i], nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (self *SGlobalLoadBalancerBackendGroup) AddBackendServer(serverId string, weight int, port int) (cloudprovider.ICloudLoadbalancerBackend, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SGlobalLoadBalancerBackendGroup) RemoveBackendServer(serverId string, weight int, port int) error {
	return cloudprovider.ErrNotSupported
}

func (self *SGlobalLoadBalancerBackendGroup) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SGlobalLoadBalancerBackendGroup) Sync(ctx context.Context, group *cloudprovider.SLoadbalancerBackendGroup) error {
	return cloudprovider.ErrNotSupported
}

func (self *SGlobalLoadbalancer) GetLoadbalancerBackendGroups() ([]SGlobalLoadBalancerBackendGroup, error) {
	bss, err := self.GetBackendServices()
	if err != nil {
		return nil, errors.Wrap(err, "GetBackendServices")
	}

	ret := make([]SGlobalLoadBalancerBackendGroup, 0)
	for i := range bss {
		group := SGlobalLoadBalancerBackendGroup{
			backendService: bss[i],
			Id:             bss[i].GetId(),
			Name:           bss[i].GetName(),
			lb:             self,
		}

		ret = append(ret, group)
	}

	return ret, nil
}
