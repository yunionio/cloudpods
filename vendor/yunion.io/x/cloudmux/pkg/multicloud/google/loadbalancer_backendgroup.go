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

type SLoadBalancerBackendGroup struct {
	lb       *SLoadbalancer
	backends []SLoadbalancerBackend

	backendService SBackendServices //

	Id   string `json:"id"`
	Name string `json:"name"`
}

func (self *SLoadBalancerBackendGroup) GetId() string {
	return self.Id
}

func (self *SLoadBalancerBackendGroup) GetName() string {
	return self.Name
}

func (self *SLoadBalancerBackendGroup) GetDescription() string {
	return ""
}

func (self *SLoadBalancerBackendGroup) GetGlobalId() string {
	return self.backendService.GetGlobalId()
}

func (self *SLoadBalancerBackendGroup) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SLoadBalancerBackendGroup) Refresh() error {
	return nil
}

func (self *SLoadBalancerBackendGroup) IsEmulated() bool {
	return true
}

func (self *SLoadBalancerBackendGroup) GetSysTags() map[string]string {
	return nil
}

func (self *SLoadBalancerBackendGroup) GetTags() (map[string]string, error) {
	return nil, nil
}

func (self *SLoadBalancerBackendGroup) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadBalancerBackendGroup) GetProjectId() string {
	return self.lb.GetProjectId()
}

func (self *SLoadBalancerBackendGroup) IsDefault() bool {
	return false
}

func (self *SLoadBalancerBackendGroup) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SLoadBalancerBackendGroup) GetType() string {
	return api.LB_BACKENDGROUP_TYPE_NORMAL
}

func (self *SLoadBalancerBackendGroup) GetLoadbalancerId() string {
	return self.lb.GetGlobalId()
}

func (self *SLoadBalancerBackendGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	backends, err := self.GetLoadbalancerBackends()
	if err != nil {
		return nil, errors.Wrap(err, "GetLoadbalancerBackends")
	}

	ibackends := make([]cloudprovider.ICloudLoadbalancerBackend, len(backends))
	for i := range backends {
		ibackends[i] = &backends[i]
	}

	return ibackends, nil
}

func (self *SLoadBalancerBackendGroup) GetILoadbalancerBackendById(backendId string) (cloudprovider.ICloudLoadbalancerBackend, error) {
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

func (self *SLoadBalancerBackendGroup) GetProtocolType() string {
	return ""
}

func (self *SLoadBalancerBackendGroup) GetScheduler() string {
	return ""
}

func (self *SLoadBalancerBackendGroup) GetHealthCheck() (*cloudprovider.SLoadbalancerHealthCheck, error) {
	return nil, nil
}

func (self *SLoadBalancerBackendGroup) GetStickySession() (*cloudprovider.SLoadbalancerStickySession, error) {
	return nil, nil
}

func (self *SLoadBalancerBackendGroup) AddBackendServer(serverId string, weight int, port int) (cloudprovider.ICloudLoadbalancerBackend, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SLoadBalancerBackendGroup) RemoveBackendServer(serverId string, weight int, port int) error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadBalancerBackendGroup) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadBalancerBackendGroup) Sync(ctx context.Context, group *cloudprovider.SLoadbalancerBackendGroup) error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) GetLoadbalancerBackendGroups() ([]SLoadBalancerBackendGroup, error) {
	bss, err := self.GetBackendServices()
	if err != nil {
		return nil, errors.Wrap(err, "GetBackendServices")
	}

	ret := make([]SLoadBalancerBackendGroup, 0)
	for i := range bss {
		group := SLoadBalancerBackendGroup{
			lb:             self,
			backendService: bss[i],
			Id:             bss[i].GetId(),
			Name:           bss[i].GetName(),
		}

		ret = append(ret, group)
	}

	return ret, nil
}
