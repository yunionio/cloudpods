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

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNlbServerGroupServer struct {
	multicloud.SResourceBase
	AliyunTags
	nlbServerGroup *SNlbServerGroup
	NlbServer
}

func (server *SNlbServerGroupServer) GetName() string {
	return server.ServerId
}

func (server *SNlbServerGroupServer) GetId() string {
	return fmt.Sprintf("%s/%s/%d", server.nlbServerGroup.ServerGroupId, server.ServerId, server.Port)
}

func (server *SNlbServerGroupServer) GetGlobalId() string {
	return server.GetId()
}

func (server *SNlbServerGroupServer) GetStatus() string {
	switch server.Status {
	case "Available":
		return api.LB_STATUS_ENABLED
	case "Configuring":
		return api.LB_STATUS_UNKNOWN
	default:
		return api.LB_STATUS_ENABLED
	}
}

func (server *SNlbServerGroupServer) IsEmulated() bool {
	return false
}

func (server *SNlbServerGroupServer) Refresh() error {
	serverGroup, err := server.nlbServerGroup.nlb.region.GetNlbServerGroup(server.nlbServerGroup.ServerGroupId)
	if err != nil {
		return err
	}

	for _, s := range serverGroup.Servers {
		if s.ServerId == server.ServerId && s.Port == server.Port {
			return jsonutils.Update(server, &s)
		}
	}
	return cloudprovider.ErrNotFound
}

func (server *SNlbServerGroupServer) GetWeight() int {
	return server.Weight
}

func (server *SNlbServerGroupServer) GetPort() int {
	return server.Port
}

func (server *SNlbServerGroupServer) GetBackendType() string {
	return api.LB_BACKEND_GUEST
}

func (server *SNlbServerGroupServer) GetBackendRole() string {
	return api.LB_BACKEND_ROLE_DEFAULT
}

func (server *SNlbServerGroupServer) GetBackendId() string {
	return server.ServerId
}

func (server *SNlbServerGroupServer) GetIpAddress() string {
	return server.ServerIp
}

func (server *SNlbServerGroupServer) GetProjectId() string {
	return server.nlbServerGroup.GetProjectId()
}

func (server *SNlbServerGroupServer) Update(ctx context.Context, opts *cloudprovider.SLoadbalancerBackend) error {
	return server.nlbServerGroup.nlb.region.UpdateNlbServerGroupServerAttribute(
		server.nlbServerGroup.ServerGroupId,
		server.ServerId,
		opts.Port,
		opts.Weight,
	)
}

// region methods for NLB server group server operations
func (region *SRegion) UpdateNlbServerGroupServerAttribute(serverGroupId, serverId string, port, weight int) error {
	params := map[string]string{
		"RegionId":      region.RegionId,
		"ServerGroupId": serverGroupId,
	}

	servers := jsonutils.NewArray()
	servers.Add(jsonutils.Marshal(map[string]interface{}{
		"ServerId":   serverId,
		"ServerType": "Ecs",
		"Port":       port,
		"Weight":     weight,
	}))
	params["Servers"] = servers.String()

	_, err := region.nlbRequest("UpdateServerGroupServersAttribute", params)
	return err
}
