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

package types

type SNetworkConfig struct {
	GuestDhcp    string `json:"guest_dhcp"`
	GuestGateway string `json:"guest_gateway"`
	GuestIpStart string `json:"guest_ip_start"`
	GuestIpEnd   string `json:"guest_ip_end"`
	GuestIpMask  int    `json:"guest_ip_mask"`
	Id           string `json:"id"`
	IsEmulated   bool   `json:"is_emulated"`
	IsPublic     bool   `json:"is_public"`
	IsSystem     bool   `json:"is_system"`
	Name         string `json:"name"`
	ServerType   string `json:"server_type"`
	Status       string `json:"status"`
	ProjectId    string `json:"tenant_id"`
	VlanId       int    `json:"vlan_id"`
	WireId       string `json:"wire_id"`
}
