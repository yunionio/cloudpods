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

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SLoadbalancerBackendGroup struct {
	multicloud.SResourceBase
	AzureTags
	lb *SLoadbalancer

	Id string

	Properties              *SLoadbalancerBackend
	BackendIPConfigurations []struct {
		Id string
	}
}

func (self *SLoadbalancerBackendGroup) GetId() string {
	return self.Id
}

func (self *SLoadbalancerBackendGroup) GetName() string {
	info := strings.Split(self.Id, "/")
	if len(info) > 0 {
		return info[len(info)-1]
	}
	return ""
}

func (self *SLoadbalancerBackendGroup) GetGlobalId() string {
	return strings.ToLower(self.GetId())
}

func (self *SLoadbalancerBackendGroup) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SLoadbalancerBackendGroup) Refresh() error {
	lbbg, err := self.lb.GetILoadBalancerBackendGroupById(self.GetId())
	if err != nil {
		return errors.Wrap(err, "GetILoadBalancerBackendGroupById")
	}

	return jsonutils.Update(self, lbbg)
}

func (self *SLoadbalancerBackendGroup) GetProjectId() string {
	return getResourceGroup(self.GetId())
}

func (self *SLoadbalancerBackendGroup) IsDefault() bool {
	return false
}

func (self *SLoadbalancerBackendGroup) GetType() string {
	return api.LB_BACKENDGROUP_TYPE_NORMAL
}

func (self *SLoadbalancerBackendGroup) GetLoadbalancerId() string {
	return self.lb.GetId()
}

func (self *SLoadbalancerBackendGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	ret := []cloudprovider.ICloudLoadbalancerBackend{}
	if len(self.BackendIPConfigurations) > 0 {
		for _, ipConf := range self.BackendIPConfigurations {
			apiVerion := "2024-03-01"
			if strings.Contains(strings.ToLower(ipConf.Id), "microsoft.network/networkinterfaces") {
				apiVerion = "2023-11-01"
			}
			resp, err := self.lb.region.show(ipConf.Id, apiVerion)
			if err != nil {
				return nil, err
			}
			backend := &SLoadbalancerBackend{
				lbbg: self,
			}
			err = resp.Unmarshal(backend, "properties")
			if err != nil {
				return nil, errors.Wrapf(err, "Unmarshal")
			}
			ret = append(ret, backend)
		}
		return ret, nil
	}
	resp, err := self.lb.region.show(self.Id, "2021-02-01")
	if err != nil {
		return nil, err
	}
	err = resp.Unmarshal(self)
	if err != nil {
		return nil, err
	}
	if self.Properties != nil && (len(self.Properties.PrivateIPAddress) > 0 || len(self.Properties.LoadBalancerBackendAddresses) > 0) {
		self.Properties.lbbg = self
		ret = append(ret, self.Properties)
	}
	return ret, nil
}

func (self *SLoadbalancerBackendGroup) GetILoadbalancerBackendById(backendId string) (cloudprovider.ICloudLoadbalancerBackend, error) {
	lbbs, err := self.GetILoadbalancerBackends()
	if err != nil {
		return nil, errors.Wrap(err, "GetILoadbalancerBackends")
	}

	for i := range lbbs {
		if lbbs[i].GetId() == backendId {
			return lbbs[i], nil
		}
	}

	return nil, errors.Wrapf(cloudprovider.ErrNotFound, backendId)
}

func (self *SLoadbalancerBackendGroup) GetProtocolType() string {
	return ""
}

func (self *SLoadbalancerBackendGroup) GetScheduler() string {
	return ""
}

func (self *SLoadbalancerBackendGroup) GetHealthCheck() (*cloudprovider.SLoadbalancerHealthCheck, error) {
	return nil, nil
}

func (self *SLoadbalancerBackendGroup) GetStickySession() (*cloudprovider.SLoadbalancerStickySession, error) {
	return nil, nil
}

func (self *SLoadbalancerBackendGroup) AddBackendServer(serverId string, weight int, port int) (cloudprovider.ICloudLoadbalancerBackend, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "AddBackendServer")
}

func (self *SLoadbalancerBackendGroup) RemoveBackendServer(serverId string, weight int, port int) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "RemoveBackendServer")
}

func (self *SLoadbalancerBackendGroup) Delete(ctx context.Context) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Delete")
}

func (self *SLoadbalancerBackendGroup) Sync(ctx context.Context, group *cloudprovider.SLoadbalancerBackendGroup) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Sync")
}
