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

package compute

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	type NetworkinterfaceListOptions struct {
		options.BaseListOptions
	}

	R(&NetworkinterfaceListOptions{}, "network-interface-list", "List networkinterfaces", func(s *mcclient.ClientSession, opts *NetworkinterfaceListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.Networkinterfaces.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Networkinterfaces.GetColumns(s))
		return nil
	})

	type NetworkinterfaceIdOptions struct {
		ID string `help:"Id or name of networkinterface"`
	}

	R(&NetworkinterfaceIdOptions{}, "network-interface-show", "Show networkinterface detail", func(s *mcclient.ClientSession, args *NetworkinterfaceIdOptions) error {
		result, err := modules.Networkinterfaces.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
