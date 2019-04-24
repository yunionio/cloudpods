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

type SLoadbalancerMasterSlaveBackendGroup struct {
	lb *SLoadbalancer

	MasterSlaveServerGroupId   string
	MasterSlaveServerGroupName string
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) GetName() string {
	return backendgroup.MasterSlaveServerGroupName
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) GetId() string {
	return backendgroup.MasterSlaveServerGroupId
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) GetGlobalId() string {
	return backendgroup.GetId()
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) IsEmulated() bool {
	return false
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) Refresh() error {
	return nil
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) IsDefault() bool {
	return false
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) GetType() string {
	return api.LB_BACKENDGROUP_TYPE_MASTER_SLAVE
}

func (region *SRegion) GetLoadbalancerMasterSlaveBackendgroups(loadbalancerId string) ([]SLoadbalancerMasterSlaveBackendGroup, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	body, err := region.lbRequest("DescribeMasterSlaveServerGroups", params)
	if err != nil {
		return nil, err
	}
	backendgroups := []SLoadbalancerMasterSlaveBackendGroup{}
	return backendgroups, body.Unmarshal(&backendgroups, "MasterSlaveServerGroups", "MasterSlaveServerGroup")
}

func (region *SRegion) GetLoadbalancerMasterSlaveBackends(backendgroupId string) ([]SLoadbalancerMasterSlaveBackend, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["MasterSlaveServerGroupId"] = backendgroupId
	body, err := region.lbRequest("DescribeMasterSlaveServerGroupAttribute", params)
	if err != nil {
		return nil, err
	}
	backends := []SLoadbalancerMasterSlaveBackend{}
	return backends, body.Unmarshal(&backends, "MasterSlaveBackendServers", "MasterSlaveBackendServer")
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	backends, err := backendgroup.lb.region.GetLoadbalancerMasterSlaveBackends(backendgroup.MasterSlaveServerGroupId)
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

func (region *SRegion) CreateLoadbalancerMasterSlaveBackendGroup(name, loadbalancerId string, backends []cloudprovider.SLoadbalancerBackend) (*SLoadbalancerMasterSlaveBackendGroup, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["MasterSlaveServerGroupName"] = name
	params["LoadBalancerId"] = loadbalancerId
	if len(backends) != 2 {
		return nil, fmt.Errorf("master slave backendgorup must contain two backend")
	}
	servers := jsonutils.NewArray()
	for _, backend := range backends {
		serverType := "Slave"
		if backend.Index == 0 {
			serverType = "Master"
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
	params["MasterSlaveBackendServers"] = servers.String()
	body, err := region.lbRequest("CreateMasterSlaveServerGroup", params)
	if err != nil {
		return nil, err
	}
	groupId, err := body.GetString("MasterSlaveServerGroupId")
	if err != nil {
		return nil, err
	}
	return region.GetLoadbalancerMasterSlaveBackendgroupById(groupId)
}

func (region *SRegion) GetLoadbalancerMasterSlaveBackendgroupById(groupId string) (*SLoadbalancerMasterSlaveBackendGroup, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["MasterSlaveServerGroupId"] = groupId
	params["NeedInstanceDetail"] = "true"
	body, err := region.lbRequest("DescribeMasterSlaveServerGroupAttribute", params)
	if err != nil {
		return nil, err
	}
	group := &SLoadbalancerMasterSlaveBackendGroup{}
	return group, body.Unmarshal(group)
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) Sync(name string) error {
	return nil
}

func (region *SRegion) DeleteLoadbalancerMasterSlaveBackendgroup(groupId string) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["MasterSlaveServerGroupId"] = groupId
	_, err := region.lbRequest("DeleteMasterSlaveServerGroup", params)
	return err
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) Delete() error {
	return backendgroup.lb.region.DeleteLoadbalancerMasterSlaveBackendgroup(backendgroup.MasterSlaveServerGroupId)
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) AddBackendServer(serverId string, weight, port int) (cloudprovider.ICloudLoadbalancerBackend, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) RemoveBackendServer(serverId string, weight, port int) error {
	return cloudprovider.ErrNotSupported
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) GetProjectId() string {
	return ""
}
