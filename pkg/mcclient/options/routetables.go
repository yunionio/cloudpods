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

import (
	"fmt"

	"yunion.io/x/jsonutils"
)

type Route struct {
	Type        string
	Cidr        string
	NextHopType string
	NextHopId   string
}

type Routes []*Route

type RoutesOptions struct {
	Type        []string
	Cidr        []string
	NextHopType []string
	NextHopId   []string
}

func (opts *RoutesOptions) Params() (jsonutils.JSONObject, error) {
	len0 := len(opts.Type)
	len1 := len(opts.Cidr)
	if len0 != len1 || len0 != len(opts.NextHopType) || len1 != len(opts.NextHopId) {
		return nil, fmt.Errorf("there must be equal number of options for each route")
	}
	routes := []*Route{}
	for i := 0; i < len0; i++ {
		routes = append(routes, &Route{
			Type:        opts.Type[i],
			Cidr:        opts.Cidr[i],
			NextHopType: opts.NextHopType[i],
			NextHopId:   opts.NextHopId[i],
		})
	}
	routesJson := jsonutils.Marshal(routes)
	return routesJson, nil
}

type RouteTableCreateOptions struct {
	NAME string
	Vpc  string `required:"true"`

	RoutesOptions
}

func (opts *RouteTableCreateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := optionsStructToParams(opts)
	if err != nil {
		return nil, err
	}
	routesJson, err := opts.RoutesOptions.Params()
	if err != nil {
		return nil, err
	}
	params.Set("routes", routesJson)
	return params, nil
}

type RouteTableGetOptions struct {
	ID string
}

type RouteTableIdOptions struct {
	ID string
}

func (opts *RouteTableIdOptions) GetId() string {
	return opts.ID
}

func (opts *RouteTableIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type RouteTableUpdateOptions struct {
	ID   string `json:"-"`
	Name string

	RoutesOptions
}

func (opts *RouteTableUpdateOptions) GetId() string {
	return opts.ID
}

func (opts *RouteTableUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := optionsStructToParams(opts)
	if err != nil {
		return nil, err
	}
	if len(opts.Cidr) != 0 {
		routesJson, err := opts.RoutesOptions.Params()
		if err != nil {
			return nil, err
		}
		params.Set("routes", routesJson)
	}
	return params, nil
}

type RouteTableAddRoutesOptions struct {
	ID string `json:"-"`

	RoutesOptions
}

func (opts *RouteTableAddRoutesOptions) GetId() string {
	return opts.ID
}

func (opts *RouteTableAddRoutesOptions) Params() (jsonutils.JSONObject, error) {
	if len(opts.Cidr) == 0 {
		return nil, fmt.Errorf("nothing to add")
	}
	routesJson, err := opts.RoutesOptions.Params()
	if err != nil {
		return nil, err
	}
	params := jsonutils.NewDict()
	params.Set("routes", routesJson)
	return params, nil
}

type RouteTableDelRoutesOptions struct {
	ID string `json:"-"`

	Cidr []string
}

func (opts *RouteTableDelRoutesOptions) GetId() string {
	return opts.ID
}

func (opts *RouteTableDelRoutesOptions) Params() (jsonutils.JSONObject, error) {
	if len(opts.Cidr) == 0 {
		return nil, fmt.Errorf("nothing to del")
	}
	params := jsonutils.NewDict()
	params.Set("cidrs", jsonutils.Marshal(opts.Cidr))
	return params, nil
}

type RouteTableDeleteOptions struct {
	ID string
}

type RouteTablePurgeOptions struct {
	ID string
}

type RouteTableListOptions struct {
	Vpc         string
	Cloudregion string

	BaseListOptions
}

func (opts *RouteTableListOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}

type RouteTableSyncstatusOptions struct {
	ID string
}
