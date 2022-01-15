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

package options

import "yunion.io/x/jsonutils"

type RouteTableRouteSetListOptions struct {
	BaseListOptions
	RouteTableId string
	VpcId        string
	Type         string
	NextHopType  string
	NextHopId    string
	Cidr         string
}

func (opts *RouteTableRouteSetListOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}

type RouteTableRouteSetIdOptions struct {
	ID string `json:"route table routeset ID"`
}

func (opts *RouteTableRouteSetIdOptions) GetId() string {
	return opts.ID
}

func (opts *RouteTableRouteSetIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type RouteTableRouteSetCreateOptions struct {
	RouteTableId string
	Type         string
	Cidr         string
	NextHopType  string
	NextHopId    string
}

func (opts *RouteTableRouteSetCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts).(*jsonutils.JSONDict), nil
}

type RouteTableRouteSetUpdateOptions struct {
	ID           string `json:"route table routeset ID"`
	RouteTableId string
	Type         string
	Cidr         string
	NextHopType  string
	NextHopId    string
}

func (opts *RouteTableRouteSetUpdateOptions) GetId() string {
	return opts.ID
}

func (opts *RouteTableRouteSetUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts).(*jsonutils.JSONDict), nil
}
