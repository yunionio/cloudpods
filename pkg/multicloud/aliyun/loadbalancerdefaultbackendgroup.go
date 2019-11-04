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

package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLoadbalancerDefaultBackendGroup struct {
	lb *SLoadbalancer
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetILoadbalancer() cloudprovider.ICloudLoadbalancer {
	return backendgroup.lb
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetLoadbalancerId() string {
	return backendgroup.lb.GetId()
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetProtocolType() string {
	return ""
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetScheduler() string {
	return ""
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetHealthCheck() (*cloudprovider.SLoadbalancerHealthCheck, error) {
	return nil, nil
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetStickySession() (*cloudprovider.SLoadbalancerStickySession, error) {
	return nil, nil
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetName() string {
	return fmt.Sprintf("%s(%s)-default", backendgroup.lb.LoadBalancerName, backendgroup.lb.Address)
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetId() string {
	return fmt.Sprintf("%s/default", backendgroup.lb.LoadBalancerId)
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetGlobalId() string {
	return backendgroup.GetId()
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) IsDefault() bool {
	return true
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetType() string {
	return api.LB_BACKENDGROUP_TYPE_DEFAULT
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) IsEmulated() bool {
	return false
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) Refresh() error {
	return nil
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	loadbalancer, err := backendgroup.lb.region.GetLoadbalancerDetail(backendgroup.lb.LoadBalancerId)
	if err != nil {
		return nil, err
	}
	ibackends := []cloudprovider.ICloudLoadbalancerBackend{}
	for i := 0; i < len(loadbalancer.BackendServers.BackendServer); i++ {
		loadbalancer.BackendServers.BackendServer[i].lbbg = backendgroup
		ibackends = append(ibackends, &loadbalancer.BackendServers.BackendServer[i])
	}
	return ibackends, nil
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetILoadbalancerBackendById(backendId string) (cloudprovider.ICloudLoadbalancerBackend, error) {
	backends, err := backendgroup.GetILoadbalancerBackends()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(backends); i++ {
		if backends[i].GetGlobalId() == backendId {
			return backends[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) Sync(group *cloudprovider.SLoadbalancerBackendGroup) error {
	return cloudprovider.ErrNotSupported
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (region *SRegion) AddBackendServer(loadbalancerId, serverId string, weight, port int) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	servers := jsonutils.NewArray()
	servers.Add(jsonutils.Marshal(map[string]string{"ServerId": serverId, "Weight": fmt.Sprintf("%d", weight)}))
	params["BackendServers"] = servers.String()
	_, err := region.lbRequest("AddBackendServers", params)
	return err
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) AddBackendServer(serverId string, weight, port int) (cloudprovider.ICloudLoadbalancerBackend, error) {
	if err := backendgroup.lb.region.AddBackendServer(backendgroup.lb.LoadBalancerId, serverId, weight, port); err != nil {
		return nil, err
	}
	return &SLoadbalancerDefaultBackend{lbbg: backendgroup, ServerId: serverId, Weight: weight}, nil
}

func (region *SRegion) RemoveBackendServer(loadbalancerId, serverId string) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	servers := jsonutils.NewArray()
	servers.Add(jsonutils.NewString(serverId))
	params["BackendServers"] = servers.String()
	_, err := region.lbRequest("RemoveBackendServers", params)
	return err
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) RemoveBackendServer(serverId string, weight, port int) error {
	return backendgroup.lb.region.RemoveBackendServer(backendgroup.lb.LoadBalancerId, serverId)
}

func (backendgroup *SLoadbalancerDefaultBackendGroup) GetProjectId() string {
	return ""
}
