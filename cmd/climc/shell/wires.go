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
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type WireListOptions struct {
		options.BaseListOptions

		Region string `help:"List hosts in region"`
		Zone   string `help:"list wires in zone" json:"-"`
		Vpc    string `help:"List wires in vpc"`
	}
	R(&WireListOptions{}, "wire-list", "List wires", func(s *mcclient.ClientSession, opts *WireListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		var result *modulebase.ListResult
		if len(opts.Zone) > 0 {
			result, err = modules.Wires.ListInContext(s, params, &modules.Zones, opts.Zone)
		} else {
			result, err = modules.Wires.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Wires.GetColumns(s))
		return nil
	})

	type WireUpdateOptions struct {
		ID   string `help:"ID or Name of zone to update"`
		Name string `help:"Name of wire"`
		Desc string `metavar:"<DESCRIPTION>" help:"Description"`
		Bw   int64  `help:"Bandwidth in mbps"`
		Mtu  int64  `help:"mtu in bytes"`
	}
	R(&WireUpdateOptions{}, "wire-update", "Update wire", func(s *mcclient.ClientSession, args *WireUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.Bw > 0 {
			params.Add(jsonutils.NewInt(args.Bw), "bandwidth")
		}
		if args.Mtu > 0 {
			params.Add(jsonutils.NewInt(args.Mtu), "mtu")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Wires.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type WireCreateOptions struct {
		ZONE string `help:"Zone ID or Name"`
		Vpc  string `help:"VPC ID or Name" default:"default"`
		NAME string `help:"Name of wire"`
		BW   int64  `help:"Bandwidth in mbps"`
		Mtu  int64  `help:"mtu in bytes"`
		Desc string `metavar:"<DESCRIPTION>" help:"Description"`
	}
	R(&WireCreateOptions{}, "wire-create", "Create a wire", func(s *mcclient.ClientSession, args *WireCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewInt(args.BW), "bandwidth")
		if args.Mtu > 0 {
			params.Add(jsonutils.NewInt(args.Mtu), "mtu")
		}
		if len(args.Vpc) > 0 {
			params.Add(jsonutils.NewString(args.Vpc), "vpc")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		result, err := modules.Wires.CreateInContext(s, params, &modules.Zones, args.ZONE)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type WireShowOptions struct {
		ID string `help:"ID or Name of the wire to show"`
	}
	R(&WireShowOptions{}, "wire-show", "Show wire details", func(s *mcclient.ClientSession, args *WireShowOptions) error {
		result, err := modules.Wires.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&WireShowOptions{}, "wire-delete", "Delete wire", func(s *mcclient.ClientSession, args *WireShowOptions) error {
		result, err := modules.Wires.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
