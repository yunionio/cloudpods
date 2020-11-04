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

type RouteTableAssociationManager struct {
	modulebase.ResourceManager
}

var (
	RouteTableRouteAssociations RouteTableAssociationManager
)

func init() {
	RouteTableRouteAssociations = RouteTableAssociationManager{
		NewComputeManager(
			"route_table_association",
			"route_table_associations",
			[]string{
				"id",
				"name",
				"type",
				"route_table_id",
				"association_type",
				"associated_resource_id",
				"ext_associated_resource_id",
			},
			[]string{},
		),
	}
	registerCompute(&RouteTableRouteAssociations)
}
