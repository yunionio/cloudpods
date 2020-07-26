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
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLoadbalancerMainSubordinateBackendGroup struct {
	lb *SLoadbalancer

	MainSubordinateServerGroupId   string
	MainSubordinateServerGroupName string
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) GetLoadbalancerId() string {
	return backendgroup.lb.GetId()
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) GetProtocolType() string {
	return ""
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) GetScheduler() string {
	return ""
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) GetHealthCheck() (*cloudprovider.SLoadbalancerHealthCheck, error) {
	return nil, nil
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) GetStickySession() (*cloudprovider.SLoadbalancerStickySession, error) {
	return nil, nil
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) GetName() string {
	return backendgroup.MainSubordinateServerGroupName
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) GetId() string {
	return backendgroup.MainSubordinateServerGroupId
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) GetGlobalId() string {
	return backendgroup.GetId()
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) IsEmulated() bool {
	return false
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) Refresh() error {
	return nil
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) IsDefault() bool {
	return false
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) GetType() string {
	return api.LB_BACKENDGROUP_TYPE_MASTER_SLAVE
}

func (region *SRegion) GetLoadbalancerMainSubordinateBackendgroups(loadbalancerId string) ([]SLoadbalancerMainSubordinateBackendGroup, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	body, err := region.lbRequest("DescribeMainSubordinateServerGroups", params)
	if err != nil {
		return nil, err
	}
	backendgroups := []SLoadbalancerMainSubordinateBackendGroup{}
	return backendgroups, body.Unmarshal(&backendgroups, "MainSubordinateServerGroups", "MainSubordinateServerGroup")
}

func (region *SRegion) GetLoadbalancerMainSubordinateBackends(backendgroupId string) ([]SLoadbalancerMainSubordinateBackend, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["MainSubordinateServerGroupId"] = backendgroupId
	body, err := region.lbRequest("DescribeMainSubordinateServerGroupAttribute", params)
	if err != nil {
		return nil, err
	}
	backends := []SLoadbalancerMainSubordinateBackend{}
	return backends, body.Unmarshal(&backends, "MainSubordinateBackendServers", "MainSubordinateBackendServer")
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	backends, err := backendgroup.lb.region.GetLoadbalancerMainSubordinateBackends(backendgroup.MainSubordinateServerGroupId)
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

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) GetILoadbalancerBackendById(backendId string) (cloudprovider.ICloudLoadbalancerBackend, error) {
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

func (region *SRegion) CreateLoadbalancerMainSubordinateBackendGroup(name, loadbalancerId string, backends []cloudprovider.SLoadbalancerBackend) (*SLoadbalancerMainSubordinateBackendGroup, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["MainSubordinateServerGroupName"] = name
	params["LoadBalancerId"] = loadbalancerId
	if len(backends) != 2 {
		return nil, fmt.Errorf("main subordinate backendgorup must contain two backend")
	}
	servers := jsonutils.NewArray()
	for _, backend := range backends {
		serverType := "Subordinate"
		if backend.Index == 0 {
			serverType = "Main"
		}
		servers.Add(
			jsonutils.Marshal(
				map[string]string{
					"ServerId":   backend.ExternalID,
					"Port":       fmt.Sprintf("%d", backend.Port),
					"Weight":     fmt.Sprintf("%d", backend.Weight),
					"ServerType": serverType,
				},
			))
	}
	params["MainSubordinateBackendServers"] = servers.String()
	body, err := region.lbRequest("CreateMainSubordinateServerGroup", params)
	if err != nil {
		return nil, err
	}
	groupId, err := body.GetString("MainSubordinateServerGroupId")
	if err != nil {
		return nil, err
	}
	return region.GetLoadbalancerMainSubordinateBackendgroupById(groupId)
}

func (region *SRegion) GetLoadbalancerMainSubordinateBackendgroupById(groupId string) (*SLoadbalancerMainSubordinateBackendGroup, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["MainSubordinateServerGroupId"] = groupId
	params["NeedInstanceDetail"] = "true"
	body, err := region.lbRequest("DescribeMainSubordinateServerGroupAttribute", params)
	if err != nil {
		return nil, err
	}
	group := &SLoadbalancerMainSubordinateBackendGroup{}
	return group, body.Unmarshal(group)
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) Sync(ctx context.Context, group *cloudprovider.SLoadbalancerBackendGroup) error {
	return nil
}

func (region *SRegion) DeleteLoadbalancerMainSubordinateBackendgroup(groupId string) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["MainSubordinateServerGroupId"] = groupId
	_, err := region.lbRequest("DeleteMainSubordinateServerGroup", params)
	return err
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) Delete(ctx context.Context) error {
	return backendgroup.lb.region.DeleteLoadbalancerMainSubordinateBackendgroup(backendgroup.MainSubordinateServerGroupId)
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) AddBackendServer(serverId string, weight, port int) (cloudprovider.ICloudLoadbalancerBackend, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) RemoveBackendServer(serverId string, weight, port int) error {
	return cloudprovider.ErrNotSupported
}

func (backendgroup *SLoadbalancerMainSubordinateBackendGroup) GetProjectId() string {
	return ""
}
