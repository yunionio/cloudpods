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
	type NetworkReserveIPOptions struct {
		NETWORK string   `help:"IP or name of network"`
		NOTES   string   `help:"Why reserve this IP"`
		IPS     []string `help:"IPs to reserve"`
	}
	R(&NetworkReserveIPOptions{}, "network-reserve-ip", "Reserve an IP address from pool", func(s *mcclient.ClientSession, args *NetworkReserveIPOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewStringArray(args.IPS), "ips")
		params.Add(jsonutils.NewString(args.NOTES), "notes")
		net, err := modules.Networks.PerformAction(s, args.NETWORK, "reserve-ip", params)
		if err != nil {
			return err
		}
		printObject(net)
		return nil
	})

	type NetworkReleaseReservedIPOptions struct {
		NETWORK string `help:"IP or name of network"`
		IP      string `help:"IP to release"`
	}
	R(&NetworkReleaseReservedIPOptions{}, "network-release-reserved-ip", "Release a reserved IP into pool", func(s *mcclient.ClientSession, args *NetworkReleaseReservedIPOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.IP), "ip")
		net, err := modules.Networks.PerformAction(s, args.NETWORK, "release-reserved-ip", params)
		if err != nil {
			return err
		}
		printObject(net)
		return nil
	})

	type ReservedIPListOptions struct {
		options.BaseListOptions
		Network string `help:"Network filter"`
	}
	R(&ReservedIPListOptions{}, "reserved-ip-list", "Show all reserved IPs for any network", func(s *mcclient.ClientSession, args *ReservedIPListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if len(args.Network) > 0 {
			params.Add(jsonutils.NewString(args.Network), "network")
		}
		result, err := modules.ReservedIPs.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.ReservedIPs.GetColumns(s))
		return nil
	})

	type ReservedIpUpdateOptions struct {
		ID    string `help:"ID of reserved ip" json:"-"`
		Notes string `help:"notes"`
	}
	R(&ReservedIpUpdateOptions{}, "reserved-ip-update", "update reserved ip notes", func(s *mcclient.ClientSession, args *ReservedIpUpdateOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.ReservedIPs.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
