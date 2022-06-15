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

import "yunion.io/x/onecloud/pkg/apis"

const (
	TapFlowVSwitch  = "vswitch"
	TapFlowGuestNic = "vnic"

	TapFlowDirectionIn   = "IN"
	TapFlowDirectionOut  = "OUT"
	TapFlowDirectionBoth = "BOTH"

	TapFlowIdMin = 0x10
	TapFlowIdMax = 0x7fff
)

var (
	TapFlowDirections = []string{
		TapFlowDirectionIn,
		TapFlowDirectionOut,
		TapFlowDirectionBoth,
	}
)

type NetTapFlowListInput struct {
	apis.EnabledStatusStandaloneResourceListInput

	TapId string `json:"tap_id"`

	HostId string `json:"host_id" help:"filter by host id or name"`
}

type NetTapFlowDetails struct {
	apis.EnabledStatusStandaloneResourceDetails

	// 关联的tap服务名称
	Tap string `json:"tap"`

	Source string `json:"source"`

	SourceIps string `json:"source_ips"`

	Net string `json:"net"`
}

type NetTapFlowCreateInput struct {
	apis.EnabledStatusStandaloneResourceCreateInput

	TapId string `json:"tap_id" required:"true" help:"tap service id or name that this flow belongs to"`

	Type string `json:"type" required:"true" choices:"vswitch|vnic" help:"type of tap flow"`

	HostId string `json:"host_id" help:"id or name of host to tap with"`

	WireId string `json:"wire_id" help:"id or name of wire to tap with"`

	VlanId *int `json:"vlan_id" help:"vlan id of vswitch to tap with"`

	GuestId string `json:"guest_id" help:"id or name of vm to tap with"`

	// swagger:ignore
	NetId string `json:"net_id" ignore:"true"`

	MacAddr string `json:"mac_addr" help:"mac address of guest nic to tap with"`

	IpAddr string `json:"ip_addr" help:"ip address of guest nic to tap with"`

	// swagger:ignore
	SourceId string `json:"source_id" ignore:"true"`

	Direction string `json:"direction" help:"flow direction" choices:"IN|OUT|BOTH"`
}
