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
	"net"
	"reflect"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	ROUTE_TABLE_UPDATING     = "updating"
	ROUTE_TABLE_UPDATEFAILED = "update_falied"
	ROUTE_TABLE_AVAILABLE    = "available"
	ROUTE_TABLE_UNKNOWN      = "unknown"
)

type RouteTableDetails struct {
	apis.StatusInfrasResourceBaseDetails
	VpcResourceInfo

	SRouteTable

	RouteSetCount    int
	AccociationCount int
}

type SRoute struct {
	Type        string `json:"type"`
	Cidr        string `json:"cidr"`
	NextHopType string `json:"next_hop_type"`
	NextHopId   string `json:"next_hop_id"`
}

func (route *SRoute) Validate() error {
	if strings.Index(route.Cidr, "/") > 0 {
		_, ipNet, err := net.ParseCIDR(route.Cidr)
		if err != nil {
			return errors.Wrapf(httperrors.ErrInputParameter, "net.ParseCIDR %s", err)
		}
		// normalize from 192.168.1.3/24 to 192.168.1.0/24
		route.Cidr = ipNet.String()
	} else {
		ip := net.ParseIP(route.Cidr).To4()
		if ip == nil {
			return errors.Wrapf(httperrors.ErrInputParameter, "invalid addr %s", route.Cidr)
		}
	}
	return nil
}

type SRoutes []*SRoute

func (routes SRoutes) String() string {
	return jsonutils.Marshal(routes).String()
}

func (routes SRoutes) IsZero() bool {
	if len(routes) == 0 {
		return true
	}
	return false
}

func (routes *SRoutes) Validate() error {
	if routes == nil {
		*routes = SRoutes{}
		return nil
	}

	found := map[string]struct{}{}
	for _, route := range *routes {
		if err := route.Validate(); err != nil {
			return err
		}
		if _, ok := found[route.Cidr]; ok {
			// error so that the user has a chance to deal with comments
			return httperrors.NewInputParameterError("duplicate route cidr %s", route.Cidr)
		}
		// TODO aliyun: check overlap with System type route
		found[route.Cidr] = struct{}{}
	}
	return nil
}

type RouteTableCreateInput struct {
	apis.StatusInfrasResourceBaseCreateInput

	VpcResourceInput

	Type   string   `json:"type"`
	Routes *SRoutes `json:"routes"`
}

type RouteTableUpdateInput struct {
	apis.StatusInfrasResourceBaseUpdateInput

	Routes *SRoutes `json:"routes"`
}

type RouteTableFilterListBase struct {
	RouteTableId string `json:"RouteTableId"`
}

type RouteTableFilterList struct {
	RouteTableFilterListBase
	VpcFilterListInput
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SRoutes{}), func() gotypes.ISerializable {
		return &SRoutes{}
	})
}
