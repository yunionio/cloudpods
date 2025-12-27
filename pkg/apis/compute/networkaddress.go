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

import (
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/util/choices"
)

const (
	NetworkAddressTypeSubIP = "sub_ip"
)

var NetworkAddressTypes = choices.NewChoices(
	NetworkAddressTypeSubIP,
)

type TNetworkAddressParentType string

const (
	NetworkAddressParentTypeGuestnetwork = TNetworkAddressParentType("guestnetwork")
)

var NetworkAddressParentTypes = choices.NewChoices(
	string(NetworkAddressParentTypeGuestnetwork),
)

type NetworkAddressCreateInput struct {
	apis.StandaloneAnonResourceCreateInput

	ParentType        TNetworkAddressParentType `json:"parent_type"`
	ParentId          int64                     `json:"parent_id"`
	GuestId           string                    `json:"guest_id"`
	GuestnetworkIndex int8                      `json:"guestnetwork_index"`

	Type      string   `json:"type"`
	NetworkId string   `json:"network_id"`
	IPAddr    string   `json:"ip_addr"`
	IPAddrs   []string `json:"ip_addrs"`
}

type NetworkAddressListInput struct {
	apis.StandaloneAnonResourceListInput
	NetworkFilterListInput

	GuestId []string `json:"guest_id"`

	ManagedResourceListInput
}

type NetworkAddressDetails struct {
	apis.StandaloneAnonResourceDetails
	NetworkResourceInfo

	Type       string `json:"type"`
	ParentType string `json:"parent_type"`
	ParentId   string `json:"parent_id"`
	NetworkId  string `json:"network_id"`
	IpAddr     string `json:"ip_addr"`

	SubCtrVid int `json:"sub_ctr_vid"`

	Guestnetwork GuestnetworkDetails `json:"guestnetwork"`
}
