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
	R(&options.RouterCreateOptions{}, "router-create", "Create router", func(s *mcclient.ClientSession, opts *options.RouterCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		router, err := modules.Routers.Create(s, params)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
	R(&options.RouterGetOptions{}, "router-show", "Show router", func(s *mcclient.ClientSession, opts *options.RouterGetOptions) error {
		router, err := modules.Routers.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
	R(&options.RouterListOptions{}, "router-list", "List routers", func(s *mcclient.ClientSession, opts *options.RouterListOptions) error {
		params, err := base_options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.Routers.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Routers.GetColumns(s))
		return nil
	})
	R(&options.RouterUpdateOptions{}, "router-update", "Update router", func(s *mcclient.ClientSession, opts *options.RouterUpdateOptions) error {
		params, err := base_options.StructToParams(opts)
		router, err := modules.Routers.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
	R(&options.RouterDeleteOptions{}, "router-delete", "Delete router", func(s *mcclient.ClientSession, opts *options.RouterDeleteOptions) error {
		router, err := modules.Routers.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
	R(&options.RouterActionJoinMeshNetworkOptions{}, "router-join-meshnetwork", "Router join meshnetwork", func(s *mcclient.ClientSession, opts *options.RouterActionJoinMeshNetworkOptions) error {
		params, err := base_options.StructToParams(opts)
		if err != nil {
			return err
		}
		router, err := modules.Routers.PerformAction(s, opts.ID, "join-mesh-network", params)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
	R(&options.RouterActionLeaveMeshNetworkOptions{}, "router-leave-meshnetwork", "Router leave meshnetwork", func(s *mcclient.ClientSession, opts *options.RouterActionLeaveMeshNetworkOptions) error {
		params, err := base_options.StructToParams(opts)
		if err != nil {
			return err
		}
		router, err := modules.Routers.PerformAction(s, opts.ID, "leave-mesh-network", params)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
	R(&options.RouterActionRegisterIfnameOptions{}, "router-register-ifname", "Router register new ifname", func(s *mcclient.ClientSession, opts *options.RouterActionRegisterIfnameOptions) error {
		params, err := base_options.StructToParams(opts)
		if err != nil {
			return err
		}
		router, err := modules.Routers.PerformAction(s, opts.ID, "register-ifname", params)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
	R(&options.RouterActionUnregisterIfnameOptions{}, "router-unregister-ifname", "Router unregister ifname", func(s *mcclient.ClientSession, opts *options.RouterActionUnregisterIfnameOptions) error {
		params, err := base_options.StructToParams(opts)
		if err != nil {
			return err
		}
		router, err := modules.Routers.PerformAction(s, opts.ID, "unregister-ifname", params)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
	R(&options.RouterActionRealizeOptions{}, "router-realize", "Router realize", func(s *mcclient.ClientSession, opts *options.RouterActionRealizeOptions) error {
		params, err := base_options.StructToParams(opts)
		if err != nil {
			return err
		}
		router, err := modules.Routers.PerformAction(s, opts.ID, "realize", params)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
}
