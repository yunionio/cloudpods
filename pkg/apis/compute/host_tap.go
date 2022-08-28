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

const (
	HostVpcBridge = "__vpc_bridge__"
	HostTapBridge = "__tap_bridge__"
)

type SMirrorConfig struct {
	TapHostIp string `json:"tap_host_ip"`

	HostIp string `json:"host_ip"`

	Port string `json:"port"`

	Bridge string `json:"bridge"`

	FlowId uint16 `json:"flow_id"`

	VlanId int `json:"vlan_id"`

	Direction string `json:"direction"`
}

/*
type SHostBridgeMirrorConfig struct {
	TapHostIp string `json:"tap_host_ip"`

	HostIp string `json:"host_ip"`

	Bridge string `json:"bridge"`

	Direction string `json:"direction"`

	FlowId uint16 `json:"flow_id"`

	VlanId []int `json:"vlan_id"`

	Port []string `json:"port"`
}
*/

type STapServiceConfig struct {
	TapHostIp string `json:"tap_host_ip"`

	MacAddr string `json:"mac_addr"`

	Ifname string `json:"ifname"`

	Mirrors []SMirrorConfig
}

type SHostTapConfig struct {
	Taps []STapServiceConfig `json:"taps"`

	Mirrors []SMirrorConfig `json:"mirrors"`
}
