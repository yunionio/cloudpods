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
)

const (
	RESERVEDIP_TABLE          = "reservedips_tbl"
	RESERVEDIP_RESOURCE_TYPE  = "reservedip"
	RESERVEDIP_RESOURCE_TYPES = "reservedips"

	RESERVEDIP_STATUS_ONLINE  = "online"
	RESERVEDIP_STATUS_OFFLINE = "offline"
	RESERVEDIP_STATUS_UNKNOWN = "unknown"
)

type ReservedipListInput struct {
	apis.ResourceBaseListInput

	NetworkFilterListInput

	// list all reserved ips, including expired ones
	All *bool `json:"all"`

	// ip_addr
	IpAddr []string `json:"ip_addr"`
	// 状态
	Status []string `json:"status"`
}

type ReservedipDetails struct {
	apis.ResourceBaseDetails
	NetworkResourceInfo

	SReservedip

	// 是否过期
	Expired bool `json:"expired"`
}
