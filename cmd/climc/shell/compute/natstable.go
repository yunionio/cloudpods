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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {

	R(&compute.NatSTableListOptions{}, "snat-list", "List SNAT entries", func(s *mcclient.ClientSession, opts *compute.NatSTableListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.NatSTable.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.NatSTable.GetColumns(s))
		return nil
	})
	R(&compute.NatSDeleteShowOptions{}, "snat-delete", "Delete a SNAT", func(s *mcclient.ClientSession, args *compute.NatSDeleteShowOptions) error {
		results, err := modules.NatSTable.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})

	R(&compute.NatSDeleteShowOptions{}, "snat-show", "Show a SNAT", func(s *mcclient.ClientSession, args *compute.NatSDeleteShowOptions) error {
		results, err := modules.NatSTable.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})

	R(&compute.NatSCreateOptions{}, "snat-create", "Create a SNAT", func(s *mcclient.ClientSession, args *compute.NatSCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.NATGATEWAYID), "natgateway_id")
		params.Add(jsonutils.NewString(args.IP), "ip")
		params.Add(jsonutils.NewString(args.EXTERNALIPID), "external_ip_id")
		params.Add(jsonutils.NewString(args.SourceCIDR), "source_cidr")
		params.Add(jsonutils.NewString(args.NetworkID), "network_id")

		result, err := modules.NatSTable.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
