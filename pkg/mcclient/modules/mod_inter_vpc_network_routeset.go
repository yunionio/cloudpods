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

type InterVpcNetworkRouteSetManager struct {
	modulebase.ResourceManager
}

var (
	InterVpcNetworkRouteSets InterVpcNetworkRouteSetManager
)

func init() {
	InterVpcNetworkRouteSets = InterVpcNetworkRouteSetManager{
		NewComputeManager(
			"inter_vpc_network_route_set",
			"inter_vpc_network_route_sets",
			[]string{
				"id",
				"inter_vpc_network_id",
				"name",
				"enabled",
				"status",
				"cidr",
				"vpc_id",
				"ext_instance_id",
				"ext_instance_type",
				"ext_instance_region_id",
			},
			[]string{},
		),
	}
	registerCompute(&InterVpcNetworkRouteSets)
}
