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
	type ZoneListOptions struct {
		options.BaseListOptions
		Region    string `help:"cloud region ID or Name" json:"-"`
		City      string `help:"Filter zone by city of cloudregions"`
		Usable    *bool  `help:"List all zones where networks are usable"`
		UsableVpc *bool  `help:"List all zones where vpc are usable"`
	}
	R(&ZoneListOptions{}, "zone-list", "List zones", func(s *mcclient.ClientSession, args *ZoneListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		var result *modulebase.ListResult
		if len(args.Region) > 0 {
			result, err = modules.Zones.ListInContext(s, params, &modules.Cloudregions, args.Region)
		} else {
			result, err = modules.Zones.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Zones.GetColumns(s))
		return nil
	})

	type ZoneUpdateOptions struct {
		ID       string `help:"ID or Name of zone to update"`
		Name     string `help:"Name of zone"`
		NameCN   string `help:"Name in Chinese"`
		Desc     string `metavar:"<DESCRIPTION>" help:"Description"`
		Location string `help:"Location"`
	}
	R(&ZoneUpdateOptions{}, "zone-update", "Update zone", func(s *mcclient.ClientSession, args *ZoneUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.NameCN) > 0 {
			params.Add(jsonutils.NewString(args.NameCN), "name_cn")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.Location) > 0 {
			params.Add(jsonutils.NewString(args.Location), "location")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Zones.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ZoneShowOptions struct {
		ID string `help:"ID or Name of the zone to show"`
	}
	R(&ZoneShowOptions{}, "zone-show", "Show zone details", func(s *mcclient.ClientSession, args *ZoneShowOptions) error {
		result, err := modules.Zones.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ZoneShowOptions{}, "zone-delete", "Delete zone", func(s *mcclient.ClientSession, args *ZoneShowOptions) error {
		result, err := modules.Zones.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ZoneCapabilityOptions struct {
		ID     string `help:"Zone ID or Name" json:"-"`
		Domain string `help:"domain Id or name"`
	}
	R(&ZoneCapabilityOptions{}, "zone-capability", "Show zone's capacibilities", func(s *mcclient.ClientSession, args *ZoneCapabilityOptions) error {
		query, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Zones.GetSpecific(s, args.ID, "capability", query)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ZoneCreateOptions struct {
		NAME     string `help:"Name of zone"`
		NameCN   string `help:"Name in Chinese"`
		Desc     string `metavar:"<DESCRIPTION>" help:"Description"`
		Location string `help:"Location"`
		Region   string `help:"Cloudregion in which zone created"`
	}
	R(&ZoneCreateOptions{}, "zone-create", "Create a zone", func(s *mcclient.ClientSession, args *ZoneCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.NameCN) > 0 {
			params.Add(jsonutils.NewString(args.NameCN), "name_cn")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.Location) > 0 {
			params.Add(jsonutils.NewString(args.Location), "location")
		}
		if len(args.Region) > 0 {
			params.Add(jsonutils.NewString(args.Region), "region")
		}
		zone, err := modules.Zones.Create(s, params)
		if err != nil {
			return err
		}
		printObject(zone)
		return nil
	})

	type ZoneStatusOptions struct {
		ID     string `help:"ID or name of zone"`
		STATUS string `help:"zone status" choices:"enable|disable|soldout"`
		REASON string `help:"why update status"`
	}
	R(&ZoneStatusOptions{}, "zone-update-status", "Update zone status", func(s *mcclient.ClientSession, args *ZoneStatusOptions) error {
		params := jsonutils.NewDict()
		params.Set("status", jsonutils.NewString(args.STATUS))
		params.Set("reason", jsonutils.NewString(args.REASON))
		result, err := modules.Zones.PerformAction(s, args.ID, "status", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
