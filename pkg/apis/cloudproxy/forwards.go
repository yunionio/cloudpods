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

package cloudproxy

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis"
)

type ForwardCreateInput struct {
	apis.VirtualResourceCreateInput

	ProxyEndpointId string `json:"proxy_endpoint_id"`
	ProxyAgentId    string `json:"proxy_agent_id"`
	Type            string `json:"type"`
	BindPortReq     int    `json:"bind_port_req,omitzero"`
	RemoteAddr      string `json:"remote_addr"`
	RemotePort      string `json:"remote_port"`

	LastSeenTimeout int `json:"last_seen_timeout,omitzero"`

	Opaque string `json:"opaque"`
}

type ForwardCreateFromServerInput struct {
	ServerId string `json:"server_id"`

	Type        string `json:"type"`
	BindPortReq int    `json:"bind_port_req,omitzero"`
	RemotePort  int    `json:"remote_port,omitzero"`

	LastSeenTimeout int `json:"last_seen_timeout,omitzero"`
}

type ForwardHeartbeatInput struct{}

type ForwardListInput struct {
	ProxyAgentId    string `json:"proxy_agent_id"`
	ProxyEndpointId string `json:"proxy_endpoint_id"`

	Type        string `json:"type"`
	RemoteAddr  string `json:"remote_addr"`
	RemotePort  *int   `json:"remote_port"`
	BindPortReq *int   `json:"bind_port_req"`

	Opaque string `json:"opaque"`
}

type ForwardDetails struct {
	ProxyEndpoint   string `json:"proxy_endpoint"`
	ProxyEndpointId string `json:"proxy_endpoint_id"`
	ProxyAgent      string `json:"proxy_agent"`
	ProxyAgentId    string `json:"proxy_agent_id"`

	Type        string `json:"type"`
	BindPortReq int    `json:"bind_port_req"`
	BindPort    int    `json:"bind_port"`
	RemoteAddr  string `json:"remote_addr"`
	RemotePort  int    `json:"remote_port"`

	LastSeen        time.Time `json:"last_seen"`
	LastSeenTimeout int       `json:"last_seen_timeout"`

	Opaque string `json:"opaque"`

	BindAddr string `json:"bind_addr"`
}
