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
	type GlobalVpcListOptions struct {
		options.BaseListOptions
	}
	R(&GlobalVpcListOptions{}, "global-vpc-list", "List global vpcs", func(s *mcclient.ClientSession, args *GlobalVpcListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.GlobalVpcs.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.GlobalVpcs.GetColumns(s))
		return nil
	})

	type GlobalVpcShowOptions struct {
		ID string `help:"ID or Name of globalvpc"`
	}
	R(&GlobalVpcShowOptions{}, "global-vpc-show", "Show details of a global vpc", func(s *mcclient.ClientSession, args *GlobalVpcShowOptions) error {
		result, err := modules.GlobalVpcs.GetById(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
