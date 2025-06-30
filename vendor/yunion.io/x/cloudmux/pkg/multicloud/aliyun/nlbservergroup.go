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

	"yunion.io/x/jsonutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNlbServerGroup struct {
	multicloud.SResourceBase
	AliyunTags
	nlb *SNlb

	ServerGroupId           string                 `json:"ServerGroupId"`
	ServerGroupName         string                 `json:"ServerGroupName"`
	ServerGroupType         string                 `json:"ServerGroupType"`
	Protocol                string                 `json:"Protocol"`
	Scheduler               string                 `json:"Scheduler"`
	VpcId                   string                 `json:"VpcId"`
	ServerGroupStatus       string                 `json:"ServerGroupStatus"`
	AddressIPVersion        string                 `json:"AddressIPVersion"`
	AnyPortEnabled          bool                   `json:"AnyPortEnabled"`
	ConnectionDrainEnabled  bool                   `json:"ConnectionDrainEnabled"`
	ConnectionDrainTimeout  int                    `json:"ConnectionDrainTimeout"`
	PreserveClientIpEnabled bool                   `json:"PreserveClientIpEnabled"`
	PersistenceEnabled      bool                   `json:"PersistenceEnabled"`
	PersistenceTimeout      int                    `json:"PersistenceTimeout"`
	HealthCheckConfig       map[string]interface{} `json:"HealthCheckConfig"`
	RelatedLoadBalancerIds  []string               `json:"RelatedLoadBalancerIds"`
	Servers                 []NlbServer            `json:"Servers"`
	CreateTime              string                 `json:"CreateTime"`
	RegionId                string                 `json:"RegionId"`
	ResourceGroupId         string                 `json:"ResourceGroupId"`
}

type NlbServer struct {
	ServerId    string `json:"ServerId"`
	ServerType  string `json:"ServerType"`
	ServerIp    string `json:"ServerIp"`
	Port        int    `json:"Port"`
	Weight      int    `json:"Weight"`
	Description string `json:"Description"`
	Status      string `json:"Status"`
	ZoneId      string `json:"ZoneId"`
}

func (group *SNlbServerGroup) GetILoadbalancer() cloudprovider.ICloudLoadbalancer {
	return group.nlb
}

func (group *SNlbServerGroup) GetLoadbalancerId() string {
	return group.nlb.GetId()
}

func (group *SNlbServerGroup) GetProtocolType() string {
	return group.Protocol
}

func (group *SNlbServerGroup) GetScheduler() string {
	return group.Scheduler
}

func (group *SNlbServerGroup) GetHealthCheck() (*cloudprovider.SLoadbalancerHealthCheck, error) {
	return nil, nil
}

func (group *SNlbServerGroup) GetStickySession() (*cloudprovider.SLoadbalancerStickySession, error) {
	return nil, nil
}

func (group *SNlbServerGroup) GetName() string {
	return group.ServerGroupName
}

func (group *SNlbServerGroup) GetId() string {
	return group.ServerGroupId
}

func (group *SNlbServerGroup) GetGlobalId() string {
	return group.ServerGroupId
}

func (group *SNlbServerGroup) GetStatus() string {
	switch group.ServerGroupStatus {
	case "Available":
		return api.LB_STATUS_ENABLED
	case "Configuring":
		return api.LB_STATUS_UNKNOWN
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (group *SNlbServerGroup) IsDefault() bool {
	return false
}

func (group *SNlbServerGroup) GetType() string {
	return api.LB_BACKENDGROUP_TYPE_NORMAL
}

func (group *SNlbServerGroup) IsEmulated() bool {
	return false
}

func (group *SNlbServerGroup) Refresh() error {
	serverGroup, err := group.nlb.region.GetNlbServerGroup(group.ServerGroupId)
	if err != nil {
		return err
	}
	return jsonutils.Update(group, serverGroup)
}

func (group *SNlbServerGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	backends := []cloudprovider.ICloudLoadbalancerBackend{}
	for i := 0; i < len(group.Servers); i++ {
		server := &SNlbServerGroupServer{
			nlbServerGroup: group,
			NlbServer:      group.Servers[i],
		}
		backends = append(backends, server)
	}
	return backends, nil
}

func (group *SNlbServerGroup) GetILoadbalancerBackendById(backendId string) (cloudprovider.ICloudLoadbalancerBackend, error) {
	backends, err := group.GetILoadbalancerBackends()
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

func (group *SNlbServerGroup) Sync(ctx context.Context, group2 *cloudprovider.SLoadbalancerBackendGroup) error {
	return nil
}

func (group *SNlbServerGroup) Delete(ctx context.Context) error {
	return group.nlb.region.DeleteNlbServerGroup(group.ServerGroupId)
}

func (group *SNlbServerGroup) AddBackendServer(serverId string, weight, port int) (cloudprovider.ICloudLoadbalancerBackend, error) {
	err := group.nlb.region.AddServersToNlbServerGroup(group.ServerGroupId, []cloudprovider.SLoadbalancerBackend{
		{
			ExternalID: serverId,
			Weight:     weight,
			Port:       port,
		},
	})
	if err != nil {
		return nil, err
	}

	return &SNlbServerGroupServer{
		nlbServerGroup: group,
		NlbServer: NlbServer{
			ServerId: serverId,
			Weight:   weight,
			Port:     port,
		},
	}, nil
}

func (group *SNlbServerGroup) RemoveBackendServer(serverId string, weight, port int) error {
	return group.nlb.region.RemoveServersFromNlbServerGroup(group.ServerGroupId, []string{serverId})
}

func (group *SNlbServerGroup) GetProjectId() string {
	return group.ResourceGroupId
}

// region methods
func (region *SRegion) GetNlbServerGroups() ([]SNlbServerGroup, error) {
	params := map[string]string{
		"RegionId":   region.RegionId,
		"MaxResults": "100",
	}

	groups := []SNlbServerGroup{}
	nextToken := ""

	for {
		if nextToken != "" {
			params["NextToken"] = nextToken
		}

		body, err := region.nlbRequest("ListServerGroups", params)
		if err != nil {
			return nil, err
		}

		pageGroups := []SNlbServerGroup{}
		err = body.Unmarshal(&pageGroups, "ServerGroups")
		if err != nil {
			return nil, err
		}

		groups = append(groups, pageGroups...)

		nextToken, _ = body.GetString("NextToken")
		if nextToken == "" {
			break
		}
	}

	return groups, nil
}

func (region *SRegion) GetNlbServerGroup(serverGroupId string) (*SNlbServerGroup, error) {
	params := map[string]string{
		"RegionId":      region.RegionId,
		"ServerGroupId": serverGroupId,
	}

	body, err := region.nlbRequest("ListServerGroupServers", params)
	if err != nil {
		return nil, err
	}

	group := &SNlbServerGroup{}
	err = body.Unmarshal(group)
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (region *SRegion) CreateNlbServerGroup(group *cloudprovider.SLoadbalancerBackendGroup, vpcId string) (*SNlbServerGroup, error) {
	params := map[string]string{
		"RegionId":        region.RegionId,
		"ServerGroupName": group.Name,
		"ServerGroupType": "Instance",
		"VpcId":           vpcId,
		"Protocol":        "TCP",
		"Scheduler":       "Wrr",
	}

	body, err := region.nlbRequest("CreateServerGroup", params)
	if err != nil {
		return nil, err
	}

	serverGroupId, err := body.GetString("ServerGroupId")
	if err != nil {
		return nil, err
	}

	return region.GetNlbServerGroup(serverGroupId)
}

func (region *SRegion) DeleteNlbServerGroup(serverGroupId string) error {
	params := map[string]string{
		"RegionId":      region.RegionId,
		"ServerGroupId": serverGroupId,
	}

	_, err := region.nlbRequest("DeleteServerGroup", params)
	return err
}

func (region *SRegion) AddServersToNlbServerGroup(serverGroupId string, backends []cloudprovider.SLoadbalancerBackend) error {
	params := map[string]string{
		"RegionId":      region.RegionId,
		"ServerGroupId": serverGroupId,
	}

	servers := jsonutils.NewArray()
	for _, backend := range backends {
		servers.Add(jsonutils.Marshal(map[string]interface{}{
			"ServerId":   backend.ExternalID,
			"ServerType": "Ecs",
			"Port":       backend.Port,
			"Weight":     backend.Weight,
		}))
	}
	params["Servers"] = servers.String()

	_, err := region.nlbRequest("AddServersToServerGroup", params)
	return err
}

func (region *SRegion) RemoveServersFromNlbServerGroup(serverGroupId string, serverIds []string) error {
	params := map[string]string{
		"RegionId":      region.RegionId,
		"ServerGroupId": serverGroupId,
	}

	servers := jsonutils.NewArray()
	for _, serverId := range serverIds {
		servers.Add(jsonutils.Marshal(map[string]interface{}{
			"ServerId":   serverId,
			"ServerType": "Ecs",
		}))
	}
	params["Servers"] = servers.String()

	_, err := region.nlbRequest("RemoveServersFromServerGroup", params)
	return err
}
