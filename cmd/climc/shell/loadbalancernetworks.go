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
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type LoadbalancerNetworkListOptions struct {
		options.BaseListOptions
		Loadbalancer string `help:"ID or Name of Loadbalancer"`
		Network      string `help:"ID or Name of network"`
		Ip           string `help:"search the IP address"`
	}
	R(&LoadbalancerNetworkListOptions{}, "lbnetwork-list", "List loadbalancer network pairs", func(s *mcclient.ClientSession, args *LoadbalancerNetworkListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if len(args.Ip) > 0 {
			params.Add(jsonutils.NewString(args.Ip), "ip_addr")
		}
		var result *modulebase.ListResult
		var err error
		if len(args.Loadbalancer) > 0 {
			result, err = modules.Loadbalancernetworks.ListDescendent(s, args.Loadbalancer, params)
		} else if len(args.Network) > 0 {
			result, err = modules.Loadbalancernetworks.ListDescendent2(s, args.Network, params)
		} else {
			result, err = modules.Loadbalancernetworks.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Loadbalancernetworks.GetColumns(s))
		return nil
	})
}
