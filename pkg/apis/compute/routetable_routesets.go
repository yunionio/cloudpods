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
	"yunion.io/x/cloudmux/pkg/apis/compute"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	NEXT_HOP_TYPE_INSTANCE   = compute.NEXT_HOP_TYPE_INSTANCE   // ECS实例。
	NEXT_HOP_TYPE_VPCPEERING = compute.NEXT_HOP_TYPE_VPCPEERING // vpc对等连接

	NEXT_HOP_TYPE_IP = compute.NEXT_HOP_TYPE_IP
)

type RouteTableRouteSetCreateInput struct {
	apis.StatusStandaloneResourceCreateInput
	RouteTableId string
	Cidr         string `json:"cidr"`
	NextHopType  string `json:"next_hop_type"`
	NextHopId    string `json:"next_hop_id"`
	ExtNextHopId string `json:"ext_next_hop_id"`
}

type RouteTableRouteSetUpdateInput struct {
	apis.StatusStandaloneResourceBaseUpdateInput
	Cidr         string `json:"cidr"`
	NextHopType  string `json:"next_hop_type"`
	NextHopId    string `json:"next_hop_id"`
	ExtNextHopId string `json:"ext_next_hop_id"`
}

type RouteTableRouteSetListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput
	RouteTableFilterList
	Type        string `json:"type"`
	NextHopType string `json:"next_hop_type"`
	NextHopId   string `json:"next_hop_id"`
	Cidr        string `json:"cidr"`
}
