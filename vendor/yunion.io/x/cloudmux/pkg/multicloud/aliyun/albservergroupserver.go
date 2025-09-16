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

type SAlbServerGroupServer struct {
	multicloud.SResourceBase
	AliyunTags
	albServerGroup *SAlbServerGroup
	AlbServer
}

func (server *SAlbServerGroupServer) GetName() string {
	return server.ServerId
}

func (server *SAlbServerGroupServer) GetId() string {
	return fmt.Sprintf("%s/%s/%s/%d", server.albServerGroup.ServerGroupId, server.ServerId, server.ServerIp, server.Port)
}

func (server *SAlbServerGroupServer) GetGlobalId() string {
	return server.GetId()
}

func (server *SAlbServerGroupServer) GetStatus() string {
	switch server.Status {
	case "Available":
		return api.LB_STATUS_ENABLED
	case "Configuring":
		return api.LB_STATUS_UNKNOWN
	default:
		return api.LB_STATUS_ENABLED
	}
}

func (server *SAlbServerGroupServer) IsEmulated() bool {
	return false
}

func (server *SAlbServerGroupServer) Refresh() error {
	servers, err := server.albServerGroup.alb.region.ListServerGroupServers(server.albServerGroup.ServerGroupId)
	if err != nil {
		return err
	}

	for _, s := range servers {
		if s.ServerId == server.ServerId && s.Port == server.Port {
			return jsonutils.Update(server, &s)
		}
	}
	return cloudprovider.ErrNotFound
}

func (server *SAlbServerGroupServer) GetWeight() int {
	return server.Weight
}

func (server *SAlbServerGroupServer) GetPort() int {
	return server.Port
}

func (server *SAlbServerGroupServer) GetBackendType() string {
	return api.LB_BACKEND_GUEST
}

func (server *SAlbServerGroupServer) GetBackendRole() string {
	return api.LB_BACKEND_ROLE_DEFAULT
}

func (server *SAlbServerGroupServer) GetBackendId() string {
	return server.ServerId
}

func (server *SAlbServerGroupServer) GetIpAddress() string {
	return server.ServerIp
}

func (server *SAlbServerGroupServer) GetProjectId() string {
	return server.albServerGroup.GetProjectId()
}

func (server *SAlbServerGroupServer) Update(ctx context.Context, opts *cloudprovider.SLoadbalancerBackend) error {
	return server.albServerGroup.alb.region.UpdateAlbServerGroupServerAttribute(
		server.albServerGroup.ServerGroupId,
		server.ServerId,
		opts.Port,
		opts.Weight,
	)
}

// region methods for ALB server group server operations
func (region *SRegion) UpdateAlbServerGroupServerAttribute(serverGroupId, serverId string, port, weight int) error {
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

	_, err := region.albRequest("UpdateServerGroupServersAttribute", params)
	return err
}
