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

const (
	NetworkAddressParentTypeGuestnetwork = "guestnetwork"
)

var NetworkAddressParentTypes = choices.NewChoices(
	NetworkAddressParentTypeGuestnetwork,
)

type NetworkAddressCreateInput struct {
	apis.StandaloneAnonResourceCreateInput

	ParentType        string
	ParentId          int64
	GuestId           string
	GuestnetworkIndex int8

	Type      string
	NetworkId string
	IPAddr    string
}

type NetworkAddressListInput struct {
	apis.StandaloneAnonResourceListInput
	NetworkFilterListInput

	GuestId []string
}

type NetworkAddressDetails struct {
	apis.StandaloneAnonResourceDetails
	NetworkResourceInfo

	Type       string
	ParentType string
	ParentId   string
	NetworkId  string
	IpAddr     string

	SubCtrVid int

	Guestnetwork GuestnetworkDetails
}
