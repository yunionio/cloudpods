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

package openstack

type SExternalFixedIP struct {
	IPAddress string `json:"ip_address"`
	SubnetId  string `json:"subnet_id"`
}
type SExternalGatewayInfo struct {
	EnableSnat       bool               `json:"enable_snat"`
	ExtrernalFiedIps []SExternalFixedIP `json:"external_fixed_ips"`
	NetworkId        string             `json:"network_id"`
}

type SConntrackHelper struct {
	Protocol string `json:"protocol"`
	Helper   string `json:"helper"`
	Port     int    `json:"port"`
}
type SRouter struct {
	ports               []SPort
	AdminStateUp        bool                 `json:"admin_state_up"`
	Description         string               `json:"description"`
	FlavorId            string               `json:"flavor_id"`
	Id                  string               `json:"id"`
	Name                string               `json:"name"`
	Routes              []SRouteEntry        `json:"routes"`
	Status              string               `json:"status"`
	ProjectId           string               `json:"project_id"`
	TenantId            string               `json:"tenant_id"`
	Tags                []string             `json:"tags"`
	ConntrackHelpers    []SConntrackHelper   `json:"conntrack_helpers"`
	ExternalGatewayInfo SExternalGatewayInfo `json:"external_gateway_info"`
}
