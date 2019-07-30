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

	R(&options.NatGatewayListOptions{}, "natgateway-list", "List NAT gateways", func(s *mcclient.ClientSession, opts *options.NatGatewayListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.NatGateways.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.NatGateways.GetColumns(s))
		return nil
	})

	R(&options.NatGatewayShowOptions{}, "natgateway-show", "Show a NAT gateway", func(s *mcclient.ClientSession, args *options.NatGatewayShowOptions) error {
		results, err := modules.NatGateways.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})
}
