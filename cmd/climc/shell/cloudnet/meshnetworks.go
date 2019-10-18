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
	R(&options.MeshNetworkCreateOptions{}, "meshnetwork-create", "Create mesh network", func(s *mcclient.ClientSession, opts *options.MeshNetworkCreateOptions) error {
		params, err := base_options.StructToParams(opts)
		if err != nil {
			return err
		}
		mn, err := modules.MeshNetworks.Create(s, params)
		if err != nil {
			return err
		}
		printObject(mn)
		return nil
	})
	R(&options.MeshNetworkGetOptions{}, "meshnetwork-show", "Show mesh network", func(s *mcclient.ClientSession, opts *options.MeshNetworkGetOptions) error {
		mn, err := modules.MeshNetworks.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(mn)
		return nil
	})
	R(&options.MeshNetworkListOptions{}, "meshnetwork-list", "List mesh networks", func(s *mcclient.ClientSession, opts *options.MeshNetworkListOptions) error {
		params, err := base_options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.MeshNetworks.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.MeshNetworks.GetColumns(s))
		return nil
	})
	R(&options.MeshNetworkUpdateOptions{}, "meshnetwork-update", "Update mesh network", func(s *mcclient.ClientSession, opts *options.MeshNetworkUpdateOptions) error {
		params, err := base_options.StructToParams(opts)
		mn, err := modules.MeshNetworks.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(mn)
		return nil
	})
	R(&options.MeshNetworkDeleteOptions{}, "meshnetwork-delete", "Delete mesh network", func(s *mcclient.ClientSession, opts *options.MeshNetworkDeleteOptions) error {
		mn, err := modules.MeshNetworks.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(mn)
		return nil
	})
	R(&options.MeshNetworkActionRealizeOptions{}, "meshnetwork-realize", "Realize mesh network", func(s *mcclient.ClientSession, opts *options.MeshNetworkActionRealizeOptions) error {
		mn, err := modules.MeshNetworks.PerformAction(s, opts.ID, "realize", nil)
		if err != nil {
			return err
		}
		printObject(mn)
		return nil
	})
}
