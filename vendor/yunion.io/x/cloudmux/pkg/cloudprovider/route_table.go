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

package cloudprovider

type RouteTableAssociationType string

type RouteTableType string

const (
	RouteTableAssociaToRouter = RouteTableAssociationType("Router")
	RouteTableAssociaToSubnet = RouteTableAssociationType("Subnet")
)

const (
	RouteTableTypeSystem = RouteTableType("System")
	RouteTableTypeCustom = RouteTableType("Custom")
)

type RouteTableAssociation struct {
	AssociationId        string
	AssociationType      RouteTableAssociationType
	AssociatedResourceId string
}

func (self RouteTableAssociation) GetGlobalId() string {
	return self.AssociationId
}

type RouteSet struct {
	RouteId     string
	Destination string // route destination
	NextHopType string // route next hop type
	NextHop     string // route next hop (ip or id)
}
