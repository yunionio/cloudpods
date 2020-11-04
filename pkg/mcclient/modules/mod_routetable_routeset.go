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

package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

type RouteTableRoutesetManager struct {
	modulebase.ResourceManager
}

var (
	RouteTableRouteSets RouteTableRoutesetManager
)

func init() {
	RouteTableRouteSets = RouteTableRoutesetManager{
		NewComputeManager(
			"route_table_route_set",
			"route_table_route_sets",
			[]string{
				"id",
				"name",
				"type",
				"route_table_id",
				"type",
				"cidr",
				"next_hop_type",
				"next_hop_id",
				"ext_next_hop_id",
			},
			[]string{},
		),
	}
	registerCompute(&RouteTableRouteSets)
}
