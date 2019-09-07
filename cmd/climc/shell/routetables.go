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

package shell

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	printRouteTableList := func(list *modulebase.ListResult, columns []string) {
		data := list.Data
		for _, jsonObj := range data {
			jd := jsonObj.(*jsonutils.JSONDict)
			routesObj, err := jd.GetArray("routes")
			if err != nil {
				continue
			}
			routes := []string{}
			for _, routeObj := range routesObj {
				typ, _ := routeObj.GetString("type")
				cidr, _ := routeObj.GetString("cidr")
				next_hop_type, _ := routeObj.GetString("next_hop_type")
				next_hop_id, _ := routeObj.GetString("next_hop_id")
				route := fmt.Sprintf("%8s: %18s %s", typ, cidr, next_hop_type)
				if next_hop_id != "" {
					route += fmt.Sprintf(":%s", next_hop_id)
				}
				routes = append(routes, route)
			}
			s := strings.Join(routes, "\n")
			jd.Set("routes", jsonutils.NewString(s))
		}
		printList(list, columns)
	}

	R(&options.RouteTableCreateOptions{}, "routetable-create", "Create routetable", func(s *mcclient.ClientSession, opts *options.RouteTableCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		routetable, err := modules.RouteTables.Create(s, params)
		if err != nil {
			return err
		}
		printObjectRecursive(routetable)
		return nil
	})
	R(&options.RouteTableGetOptions{}, "routetable-show", "Show routetable", func(s *mcclient.ClientSession, opts *options.RouteTableGetOptions) error {
		routetable, err := modules.RouteTables.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObjectRecursive(routetable)
		return nil
	})
	R(&options.RouteTableListOptions{}, "routetable-list", "List routetables", func(s *mcclient.ClientSession, opts *options.RouteTableListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.RouteTables.List(s, params)
		if err != nil {
			return err
		}
		printRouteTableList(result, modules.RouteTables.GetColumns(s))
		return nil
	})
	R(&options.RouteTableUpdateOptions{}, "routetable-update", "Update routetable", func(s *mcclient.ClientSession, opts *options.RouteTableUpdateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		routetable, err := modules.RouteTables.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObjectRecursive(routetable)
		return nil
	})
	R(&options.RouteTableAddRoutesOptions{}, "routetable-add-routes", "Add routes to routetable", func(s *mcclient.ClientSession, opts *options.RouteTableAddRoutesOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		routetable, err := modules.RouteTables.PerformAction(s, opts.ID, "add-routes", params)
		if err != nil {
			return err
		}
		printObjectRecursive(routetable)
		return nil
	})
	R(&options.RouteTableDelRoutesOptions{}, "routetable-del-routes", "Del routes to routetable", func(s *mcclient.ClientSession, opts *options.RouteTableDelRoutesOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		routetable, err := modules.RouteTables.PerformAction(s, opts.ID, "del-routes", params)
		if err != nil {
			return err
		}
		printObjectRecursive(routetable)
		return nil
	})
	R(&options.RouteTableDeleteOptions{}, "routetable-delete", "Show routetable", func(s *mcclient.ClientSession, opts *options.RouteTableDeleteOptions) error {
		routetable, err := modules.RouteTables.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObjectRecursive(routetable)
		return nil
	})
	R(&options.RouteTablePurgeOptions{}, "routetable-purge", "Purge routetable", func(s *mcclient.ClientSession, opts *options.RouteTablePurgeOptions) error {
		routetable, err := modules.RouteTables.PerformAction(s, opts.ID, "purge", nil)
		if err != nil {
			return err
		}
		printObjectRecursive(routetable)
		return nil
	})
}
