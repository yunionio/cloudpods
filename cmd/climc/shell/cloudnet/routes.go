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

package cloudnet

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/cloudnet"
	base_options "yunion.io/x/onecloud/pkg/mcclient/options"
	options "yunion.io/x/onecloud/pkg/mcclient/options/cloudnet"
)

func init() {
	R(&options.RouteCreateOptions{}, "router-route-create", "Create router", func(s *mcclient.ClientSession, opts *options.RouteCreateOptions) error {
		params, err := base_options.StructToParams(opts)
		if err != nil {
			return err
		}
		router, err := modules.Routes.Create(s, params)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
	R(&options.RouteGetOptions{}, "router-route-show", "Show router", func(s *mcclient.ClientSession, opts *options.RouteGetOptions) error {
		router, err := modules.Routes.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
	R(&options.RouteListOptions{}, "router-route-list", "List routers", func(s *mcclient.ClientSession, opts *options.RouteListOptions) error {
		params, err := base_options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.Routes.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Routes.GetColumns(s))
		return nil
	})
	R(&options.RouteUpdateOptions{}, "router-route-update", "Update router", func(s *mcclient.ClientSession, opts *options.RouteUpdateOptions) error {
		params, err := base_options.StructToParams(opts)
		router, err := modules.Routes.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
	R(&options.RouteDeleteOptions{}, "router-route-delete", "Delete router", func(s *mcclient.ClientSession, opts *options.RouteDeleteOptions) error {
		router, err := modules.Routes.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
}
