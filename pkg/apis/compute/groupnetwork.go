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

package compute

type GroupnetworkDetails struct {
	GroupJointResourceDetails

	SGroupnetwork

	// IP子网名称
	Network string `json:"network"`

	// EipAddr if eip is associated with this groupnetwork
	EipAddr string `json:"eip_addr"`
}

type GroupnetworkListInput struct {
	GroupJointsListInput

	NetworkFilterListInput

	// IP地址
	IpAddr []string `json:"ip_addr"`
}

type GroupAttachNetworkInput struct {
	// network id or name
	NetworkId string `json:"network_id" help:"The network to attach, optional"`

	// candidate IPaddr
	IpAddr string `json:"ip_addr" help:"The ip address to use, optional"`

	// Allocation direction
	AllocDir IPAllocationDirection `json:"alloc_dir" help:"ip allocation direction, optional"`

	// Reserved
	Reserved *bool `json:"reserved" help:"the address is allocated from reserved addresses"`

	// Required Designed IP
	RequireDesignatedIp *bool `json:"require_designated_ip" help:"fail if the designed ip is not available"`
}

type GroupDetachNetworkInput struct {
	// candidate IPaddr
	IpAddr string `json:"ip_addr" help:"Ip address to detach, empty if detach all networks"`
}
