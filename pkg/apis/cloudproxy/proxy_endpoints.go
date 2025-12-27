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
	"yunion.io/x/onecloud/pkg/apis"
)

type ProxyEndpointCreateInput struct {
	apis.VirtualResourceCreateInput

	User       string `json:"user"`
	Host       string `json:"host"`
	Port       int    `json:"port,omitzero"`
	PrivateKey string `json:"private_key"`

	IntranetIpAddr string `json:"intranet_ip_addr"`
}

type ProxyEndpointCreateFromServerInput struct {
	Name         string `json:"name"`
	GenerateName string `json:"generate_name"`

	ServerId string `json:"server_id"`
}

type ProxyEndpointUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput

	User       string `json:"user"`
	Host       string `json:"host"`
	Port       *int   `json:"port"`
	PrivateKey string `json:"private_key"`
}

type ProxyEndpointListInput struct {
	apis.VirtualResourceListInput

	VpcId     string `json:"vpc_id"`
	NetworkId string `json:"network_id"`
}
