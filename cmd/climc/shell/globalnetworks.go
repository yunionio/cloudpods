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
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type GlobalNetworkListOptions struct {
		options.BaseListOptions
	}
	R(&GlobalNetworkListOptions{}, "global-network-list", "List global networks", func(s *mcclient.ClientSession, args *GlobalNetworkListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.GlobalNetworks.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.GlobalNetworks.GetColumns(s))
		return nil
	})

	type GlobalNetworkShowOptions struct {
		ID string `help:"ID or Name of globalnetwork"`
	}
	R(&GlobalNetworkShowOptions{}, "global-network-show", "Show details of a global network", func(s *mcclient.ClientSession, args *GlobalNetworkShowOptions) error {
		result, err := modules.GlobalNetworks.GetById(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
