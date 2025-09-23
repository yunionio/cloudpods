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

type SAlbServerGroup struct {
	multicloud.SLoadbalancerBackendGroupBase
	AliyunTags
	alb *SAlb

	ServerGroupId          string                 `json:"ServerGroupId"`
	ServerGroupName        string                 `json:"ServerGroupName"`
	ServerGroupType        string                 `json:"ServerGroupType"`
	Protocol               string                 `json:"Protocol"`
	Scheduler              string                 `json:"Scheduler"`
	VpcId                  string                 `json:"VpcId"`
	ServerGroupStatus      string                 `json:"ServerGroupStatus"`
	StickySessionConfig    map[string]interface{} `json:"StickySessionConfig"`
	HealthCheckConfig      map[string]interface{} `json:"HealthCheckConfig"`
	UchConfig              map[string]interface{} `json:"UchConfig"`
	SlowStartConfig        map[string]interface{} `json:"SlowStartConfig"`
	ConnectionDrainConfig  map[string]interface{} `json:"ConnectionDrainConfig"`
	RelatedLoadBalancerIds []string               `json:"RelatedLoadBalancerIds"`
	Servers                []AlbServer            `json:"Servers"`
	CreateTime             string                 `json:"CreateTime"`
	RegionId               string                 `json:"RegionId"`
	ResourceGroupId        string                 `json:"ResourceGroupId"`
}

type AlbServer struct {
	ServerId    string `json:"ServerId"`
	ServerType  string `json:"ServerType"`
	ServerIp    string `json:"ServerIp"`
	Port        int    `json:"Port"`
	Weight      int    `json:"Weight"`
	Description string `json:"Description"`
	Status      string `json:"Status"`
}

func (group *SAlbServerGroup) GetILoadbalancer() cloudprovider.ICloudLoadbalancer {
	return group.alb
}

func (group *SAlbServerGroup) GetLoadbalancerId() string {
	return group.alb.GetId()
}

func (group *SAlbServerGroup) GetProtocolType() string {
	return group.Protocol
}

func (group *SAlbServerGroup) GetScheduler() string {
	return group.Scheduler
}

func (group *SAlbServerGroup) GetHealthCheck() (*cloudprovider.SLoadbalancerHealthCheck, error) {
	return nil, nil
}

func (group *SAlbServerGroup) GetStickySession() (*cloudprovider.SLoadbalancerStickySession, error) {
	return nil, nil
}

func (group *SAlbServerGroup) GetName() string {
	return group.ServerGroupName
}

func (group *SAlbServerGroup) GetId() string {
	return group.ServerGroupId
}

func (group *SAlbServerGroup) GetGlobalId() string {
	return group.ServerGroupId
}

func (group *SAlbServerGroup) GetStatus() string {
	switch group.ServerGroupStatus {
	case "Available":
		return api.LB_STATUS_ENABLED
	case "Configuring":
		return api.LB_STATUS_UNKNOWN
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (group *SAlbServerGroup) IsDefault() bool {
	return false
}

func (group *SAlbServerGroup) GetType() string {
	return api.LB_BACKENDGROUP_TYPE_NORMAL
}

func (group *SAlbServerGroup) IsEmulated() bool {
	return false
}

func (group *SAlbServerGroup) Refresh() error {
	serverGroup, err := group.alb.region.GetAlbServerGroup(group.ServerGroupId)
	if err != nil {
		return err
	}
	return jsonutils.Update(group, serverGroup)
}

func (group *SAlbServerGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	backends := []cloudprovider.ICloudLoadbalancerBackend{}
	for i := 0; i < len(group.Servers); i++ {
		server := &SAlbServerGroupServer{
			albServerGroup: group,
			AlbServer:      group.Servers[i],
		}
		backends = append(backends, server)
	}
	return backends, nil
}

func (group *SAlbServerGroup) GetILoadbalancerBackendById(backendId string) (cloudprovider.ICloudLoadbalancerBackend, error) {
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

func (group *SAlbServerGroup) Update(ctx context.Context, opts *cloudprovider.SLoadbalancerBackendGroup) error {
	return cloudprovider.ErrNotImplemented
}

func (group *SAlbServerGroup) Delete(ctx context.Context) error {
	return group.alb.region.DeleteAlbServerGroup(group.ServerGroupId)
}

func (group *SAlbServerGroup) AddBackendServer(opts *cloudprovider.SLoadbalancerBackend) (cloudprovider.ICloudLoadbalancerBackend, error) {
	err := group.alb.region.AddServersToAlbServerGroup(group.ServerGroupId, []cloudprovider.SLoadbalancerBackend{*opts})
	if err != nil {
		return nil, err
	}

	return &SAlbServerGroupServer{
		albServerGroup: group,
		AlbServer: AlbServer{
			ServerId: opts.ExternalId,
			Weight:   opts.Weight,
			Port:     opts.Port,
		},
	}, nil
}

func (group *SAlbServerGroup) RemoveBackendServer(opts *cloudprovider.SLoadbalancerBackend) error {
	return group.alb.region.RemoveServersFromAlbServerGroup(group.ServerGroupId, []string{opts.ExternalId})
}

func (group *SAlbServerGroup) GetProjectId() string {
	return group.ResourceGroupId
}

// region methods
func (region *SRegion) GetAlbServerGroups() ([]SAlbServerGroup, error) {
	params := map[string]string{
		"RegionId":   region.RegionId,
		"MaxResults": "100",
	}

	groups := []SAlbServerGroup{}
	nextToken := ""

	for {
		if nextToken != "" {
			params["NextToken"] = nextToken
		}

		body, err := region.albRequest("ListServerGroups", params)
		if err != nil {
			return nil, err
		}

		pageGroups := []SAlbServerGroup{}
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

func (region *SRegion) GetAlbServerGroup(serverGroupId string) (*SAlbServerGroup, error) {
	params := map[string]string{
		"RegionId":      region.RegionId,
		"ServerGroupId": serverGroupId,
	}

	body, err := region.albRequest("ListServerGroupServers", params)
	if err != nil {
		return nil, err
	}

	group := &SAlbServerGroup{}
	err = body.Unmarshal(group)
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (region *SRegion) CreateAlbServerGroup(group *cloudprovider.SLoadbalancerBackendGroup, vpcId string) (*SAlbServerGroup, error) {
	params := map[string]string{
		"RegionId":        region.RegionId,
		"ServerGroupName": group.Name,
		"ServerGroupType": "Instance",
		"VpcId":           vpcId,
		"Protocol":        "HTTP",
		"Scheduler":       "Wrr",
	}

	body, err := region.albRequest("CreateServerGroup", params)
	if err != nil {
		return nil, err
	}

	serverGroupId, err := body.GetString("ServerGroupId")
	if err != nil {
		return nil, err
	}

	return region.GetAlbServerGroup(serverGroupId)
}

func (region *SRegion) DeleteAlbServerGroup(serverGroupId string) error {
	params := map[string]string{
		"RegionId":      region.RegionId,
		"ServerGroupId": serverGroupId,
	}

	_, err := region.albRequest("DeleteServerGroup", params)
	return err
}

func (region *SRegion) AddServersToAlbServerGroup(serverGroupId string, backends []cloudprovider.SLoadbalancerBackend) error {
	params := map[string]string{
		"RegionId":      region.RegionId,
		"ServerGroupId": serverGroupId,
	}

	servers := jsonutils.NewArray()
	for _, backend := range backends {
		servers.Add(jsonutils.Marshal(map[string]interface{}{
			"ServerId":   backend.ExternalId,
			"ServerType": "Ecs",
			"Port":       backend.Port,
			"Weight":     backend.Weight,
		}))
	}
	params["Servers"] = servers.String()

	_, err := region.albRequest("AddServersToServerGroup", params)
	return err
}

func (region *SRegion) RemoveServersFromAlbServerGroup(serverGroupId string, serverIds []string) error {
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

	_, err := region.albRequest("RemoveServersFromServerGroup", params)
	return err
}
