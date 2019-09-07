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
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {

	type NetworkinterfaceNetworkListOptions struct {
		Networkinterface string `help:"ID or Name of Server"`
		Network          string `help:"Network ID or name"`
	}
	R(&NetworkinterfaceNetworkListOptions{}, "networkinterface-network-list", "List server network pairs", func(s *mcclient.ClientSession, args *NetworkinterfaceNetworkListOptions) error {
		params := jsonutils.NewDict()
		var result *modulebase.ListResult
		var err error
		if len(args.Networkinterface) > 0 {
			result, err = modules.Networkinterfacenetworks.ListDescendent(s, args.Networkinterface, params)
		} else if len(args.Network) > 0 {
			result, err = modules.Networkinterfacenetworks.ListDescendent2(s, args.Network, params)
		} else {
			result, err = modules.Networkinterfacenetworks.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Networkinterfacenetworks.GetColumns(s))
		return nil
	})
}
