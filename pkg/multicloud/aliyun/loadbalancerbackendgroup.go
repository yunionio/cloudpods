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

type Rule struct {
	RuleId   string
	RuleName string
	Domain   string
	Url      string
}

type Rules struct {
	Rule []Rule
}

type Listener struct {
	Protocol string
	Port     int
}

type Listeners struct {
	Listener []Listener
}

type AssociatedObjects struct {
	Rules     Rules
	Listeners Listeners
}

type SLoadbalancerBackendGroup struct {
	lb *SLoadbalancer

	VServerGroupId    string
	VServerGroupName  string
	AssociatedObjects AssociatedObjects
}

func (backendgroup *SLoadbalancerBackendGroup) GetILoadbalancer() cloudprovider.ICloudLoadbalancer {
	return backendgroup.lb
}

func (backendgroup *SLoadbalancerBackendGroup) GetLoadbalancerId() string {
	return backendgroup.lb.GetId()
}

func (backendgroup *SLoadbalancerBackendGroup) GetProtocolType() string {
	return ""
}

func (backendgroup *SLoadbalancerBackendGroup) GetScheduler() string {
	return ""
}

func (backendgroup *SLoadbalancerBackendGroup) GetHealthCheck() (*cloudprovider.SLoadbalancerHealthCheck, error) {
	return nil, nil
}

func (backendgroup *SLoadbalancerBackendGroup) GetStickySession() (*cloudprovider.SLoadbalancerStickySession, error) {
	return nil, nil
}

func (backendgroup *SLoadbalancerBackendGroup) GetName() string {
	return backendgroup.VServerGroupName
}

func (backendgroup *SLoadbalancerBackendGroup) GetId() string {
	return backendgroup.VServerGroupId
}

func (backendgroup *SLoadbalancerBackendGroup) GetGlobalId() string {
	return backendgroup.VServerGroupId
}

func (backendgroup *SLoadbalancerBackendGroup) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (backendgroup *SLoadbalancerBackendGroup) IsDefault() bool {
	return false
}

func (backendgroup *SLoadbalancerBackendGroup) GetType() string {
	return api.LB_BACKENDGROUP_TYPE_NORMAL
}

func (backendgroup *SLoadbalancerBackendGroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (backendgroup *SLoadbalancerBackendGroup) IsEmulated() bool {
	return false
}

func (backendgroup *SLoadbalancerBackendGroup) Refresh() error {
	loadbalancerBackendgroups, err := backendgroup.lb.region.GetLoadbalancerBackendgroups(backendgroup.lb.LoadBalancerId)
	if err != nil {
		return err
	}
	for _, loadbalancerBackendgroup := range loadbalancerBackendgroups {
		if loadbalancerBackendgroup.VServerGroupId == backendgroup.VServerGroupId {
			return jsonutils.Update(backendgroup, loadbalancerBackendgroup)
		}
	}
	return cloudprovider.ErrNotFound
}

func (region *SRegion) GetLoadbalancerBackendgroups(loadbalancerId string) ([]SLoadbalancerBackendGroup, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	body, err := region.lbRequest("DescribeVServerGroups", params)
	if err != nil {
		return nil, err
	}
	backendgroups := []SLoadbalancerBackendGroup{}
	return backendgroups, body.Unmarshal(&backendgroups, "VServerGroups", "VServerGroup")
}

func (backendgroup *SLoadbalancerBackendGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	backends, err := backendgroup.lb.region.GetLoadbalancerBackends(backendgroup.VServerGroupId)
	if err != nil {
		return nil, err
	}
	ibackends := []cloudprovider.ICloudLoadbalancerBackend{}
	for i := 0; i < len(backends); i++ {
		backends[i].lbbg = backendgroup
		ibackends = append(ibackends, &backends[i])
	}
	return ibackends, nil
}

func (backendgroup *SLoadbalancerBackendGroup) GetILoadbalancerBackendById(backendId string) (cloudprovider.ICloudLoadbalancerBackend, error) {
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

func (region *SRegion) CreateLoadbalancerBackendGroup(name, loadbalancerId string, backends []cloudprovider.SLoadbalancerBackend) (*SLoadbalancerBackendGroup, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["VServerGroupName"] = name
	params["LoadBalancerId"] = loadbalancerId
	if len(backends) > 0 {
		servers := jsonutils.NewArray()
		for _, backend := range backends {
			servers.Add(
				jsonutils.Marshal(
					map[string]string{
						"ServerId": backend.ExternalID,
						"Port":     fmt.Sprintf("%d", backend.Port),
						"Weight":   fmt.Sprintf("%d", backend.Weight),
					},
				))
		}
		params["BackendServers"] = servers.String()
	}
	body, err := region.lbRequest("CreateVServerGroup", params)
	if err != nil {
		return nil, err
	}
	groupId, err := body.GetString("VServerGroupId")
	if err != nil {
		return nil, err
	}
	return region.GetLoadbalancerBackendgroupById(groupId)
}

func (region *SRegion) GetLoadbalancerBackendgroupById(groupId string) (*SLoadbalancerBackendGroup, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["VServerGroupId"] = groupId
	body, err := region.lbRequest("DescribeVServerGroupAttribute", params)
	if err != nil {
		return nil, err
	}
	group := &SLoadbalancerBackendGroup{}
	return group, body.Unmarshal(group)
}

func (region *SRegion) UpdateLoadBalancerBackendGroupName(name, groupId string) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["VServerGroupId"] = groupId
	params["VServerGroupName"] = name
	_, err := region.lbRequest("SetVServerGroupAttribute", params)
	return err
}

func (backendgroup *SLoadbalancerBackendGroup) Sync(group *cloudprovider.SLoadbalancerBackendGroup) error {
	if group == nil {
		return nil
	}

	if backendgroup.VServerGroupName != group.Name {
		return backendgroup.lb.region.UpdateLoadBalancerBackendGroupName(backendgroup.VServerGroupId, group.Name)
	}
	return nil
}

func (region *SRegion) DeleteLoadBalancerBackendGroup(groupId string) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["VServerGroupId"] = groupId
	_, err := region.lbRequest("DeleteVServerGroup", params)
	return err
}

func (backendgroup *SLoadbalancerBackendGroup) Delete() error {
	return backendgroup.lb.region.DeleteLoadBalancerBackendGroup(backendgroup.VServerGroupId)
}

func (region *SRegion) AddBackendVServer(loadbalancerId, backendGroupId, serverId string, weight, port int) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	params["VServerGroupId"] = backendGroupId
	servers := jsonutils.NewArray()
	servers.Add(jsonutils.Marshal(map[string]string{"ServerId": serverId, "Weight": fmt.Sprintf("%d", weight), "Port": fmt.Sprintf("%d", port)}))
	params["BackendServers"] = servers.String()
	_, err := region.lbRequest("AddVServerGroupBackendServers", params)
	return err
}

func (region *SRegion) RemoveBackendVServer(loadbalancerId, backendgroupId, serverId string, port int) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	params["VServerGroupId"] = backendgroupId
	servers := jsonutils.NewArray()
	servers.Add(jsonutils.Marshal(map[string]string{"ServerId": serverId, "Port": fmt.Sprintf("%d", port)}))
	params["BackendServers"] = servers.String()
	_, err := region.lbRequest("RemoveVServerGroupBackendServers", params)
	return err
}

func (backendgroup *SLoadbalancerBackendGroup) AddBackendServer(serverId string, weight, port int) (cloudprovider.ICloudLoadbalancerBackend, error) {
	if err := backendgroup.lb.region.AddBackendVServer(backendgroup.lb.LoadBalancerId, backendgroup.VServerGroupId, serverId, weight, port); err != nil {
		return nil, err
	}
	return &SLoadbalancerBackend{lbbg: backendgroup, ServerId: serverId, Weight: weight, Port: port}, nil
}

func (backendgroup *SLoadbalancerBackendGroup) RemoveBackendServer(serverId string, weight, port int) error {
	return backendgroup.lb.region.RemoveBackendVServer(backendgroup.lb.LoadBalancerId, backendgroup.VServerGroupId, serverId, port)
}

func (backendgroup *SLoadbalancerBackendGroup) GetProjectId() string {
	return ""
}
