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
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	R(&options.NatDTableListOptions{}, "dnat-list", "List DNAT entries", func(s *mcclient.ClientSession, opts *options.NatDTableListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.NatDTable.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.NatDTable.GetColumns(s))
		return nil
	})
	R(&options.NatDDeleteShowOptions{}, "dnat-delete", "Delete a DNAT", func(s *mcclient.ClientSession, args *options.NatDDeleteShowOptions) error {
		results, err := modules.NatDTable.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})
	R(&options.NatDDeleteShowOptions{}, "dnat-show", "Show a DNAT", func(s *mcclient.ClientSession, args *options.NatDDeleteShowOptions) error {
		results, err := modules.NatDTable.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})

	R(&options.NatDCreateOptions{}, "dnat-create", "Create a DNAT", func(s *mcclient.ClientSession, args *options.NatDCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.NATGATEWAYID), "natgateway_id")
		params.Add(jsonutils.NewString(args.INTERNALIP), "internal_ip")
		params.Add(jsonutils.NewString(args.INTERNALPORT), "internal_port")
		params.Add(jsonutils.NewString(args.EXTERNALIP), "external_ip")
		params.Add(jsonutils.NewString(args.EXTERNALIPID), "external_ip_id")
		params.Add(jsonutils.NewString(args.EXTERNALPORT), "external_port")
		params.Add(jsonutils.NewString(args.IPPROTOCOL), "ip_protocol")

		result, err := modules.NatDTable.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
