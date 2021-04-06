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

	ProxyEndpointId string
	ProxyAgentId    string
	Type            string
	BindPortReq     int `json:",omitzero"`
	RemoteAddr      string
	RemotePort      string

	LastSeenTimeout int `json:",omitzero"`

	Opaque string
}

type ForwardCreateFromServerInput struct {
	ServerId string

	Type        string
	BindPortReq int `json:",omitzero"`
	RemotePort  int `json:",omitzero"`

	LastSeenTimeout int `json:",omitzero"`
}

type ForwardHeartbeatInput struct{}

type ForwardListInput struct {
	ProxyAgentId    string
	ProxyEndpointId string

	Type        string
	RemoteAddr  string
	RemotePort  *int
	BindPortReq *int

	Opaque string
}

type ForwardDetails struct {
	ProxyEndpoint   string
	ProxyEndpointId string
	ProxyAgent      string
	ProxyAgentId    string

	Type        string
	BindPortReq int
	BindPort    int
	RemoteAddr  string
	RemotePort  int

	LastSeen        time.Time
	LastSeenTimeout int

	Opaque string

	BindAddr string
}
