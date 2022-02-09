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

package ovn

import (
	computeapis "yunion.io/x/onecloud/pkg/apis/compute"
	agentmodels "yunion.io/x/onecloud/pkg/vpcagent/models"
)

type resolvedRoutes []resolvedRoute

type resolvedRoute struct {
	Cidr         string
	NextHop      string
	Network      *agentmodels.Network
	Guestnetwork *agentmodels.Guestnetwork
}

func resolveRoutes(vpc *agentmodels.Vpc, mss *agentmodels.ModelSets) resolvedRoutes {
	if vpc.RouteTable == nil || vpc.RouteTable.Routes == nil {
		return nil
	}
	var r resolvedRoutes

	routesModel := *vpc.RouteTable.Routes
	for _, routeModel := range routesModel {
		switch routeModel.NextHopType {
		case computeapis.NEXT_HOP_TYPE_IP:
			r = append(r, resolvedRoute{
				Cidr:    routeModel.Cidr,
				NextHop: routeModel.NextHopId,
			})
		case computeapis.NEXT_HOP_TYPE_INSTANCE:
			guestId := routeModel.NextHopId
			guest, ok := mss.Guests[guestId]
			if !ok {
				break
			}
			for _, gn := range guest.Guestnetworks {
				network := gn.Network
				gnVpc := network.Vpc
				if gnVpc.Id != vpc.Id {
					continue
				}
				r = append(r, resolvedRoute{
					Cidr:         routeModel.Cidr,
					NextHop:      gn.IpAddr,
					Network:      network,
					Guestnetwork: gn,
				})
			}
		default:
			return nil
		}
	}
	return r
}
