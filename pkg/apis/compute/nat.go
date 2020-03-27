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

type SNatSCreateInput struct {
	apis.Meta

	Name         string
	NatgatewayId string
	NetworkId    string
	Ip           string
	ExternalIpId string
	SourceCidr   string
}

type SNatDCreateInput struct {
	apis.Meta

	Name         string
	NatgatewayId string
	InternalIp   string
	InternalPort int
	ExternalIp   string
	ExternalIpId string
	ExternalPort int
	IpProtocol   string
}

type NatDEntryDetails struct {
	NatEntryDetails

	// SNatDEntry
}

type NatSEntryDetails struct {
	NatEntryDetails

	// SNatSEntry

	// SNAT归属网络
	Network SimpleNetwork `json:"network"`
}

type SimpleNetwork struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	GuestIpStart  string `json:"guest_ip_start"`
	GuestIpEnd    string `json:"guest_ip_end"`
	GuestIp6Start string `json:"guest_ip6_start"`
	GuestIp6End   string `json:"guest_ip6_end"`
}

type NatgatewayDetails struct {
	apis.StatusInfrasResourceBaseDetails

	VpcResourceInfo

	SNatGateway

	NatSpec string `json:"nat_spec"`
}
