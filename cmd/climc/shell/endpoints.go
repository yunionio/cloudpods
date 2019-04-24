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
)

func init() {
	type EndpointListOptions struct {
		Limit     int64  `help:"Limit, default 0, i.e. no limit" default:"20"`
		Offset    int64  `help:"Offset, default 0, i.e. no offset"`
		Region    string `help:"Search by region"`
		Service   string `help:"Search by service id or name"`
		Interface string `help:"Search by interface"`
	}
	R(&EndpointListOptions{}, "endpoint-list", "List service endpoints", func(s *mcclient.ClientSession, args *EndpointListOptions) error {
		mod, err := modules.GetModule(s, "endpoints")
		if err != nil {
			return err
		}
		query := jsonutils.NewDict()
		if args.Limit > 0 {
			query.Add(jsonutils.NewInt(args.Limit), "limit")
		}
		if args.Offset > 0 {
			query.Add(jsonutils.NewInt(args.Offset), "offset")
		}
		if len(args.Region) > 0 {
			query.Add(jsonutils.NewString(args.Region), "region_id")
		}
		if len(args.Service) > 0 {
			srvMod, err := modules.GetModule(s, "services")
			if err != nil {
				return err
			}
			srvId, err := srvMod.GetId(s, args.Service, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(srvId), "service_id")
		}
		if len(args.Interface) > 0 {
			query.Add(jsonutils.NewString(args.Interface), "interface")
		}
		result, err := mod.List(s, query)
		if err != nil {
			return err
		}
		printList(result, mod.GetColumns(s))
		return nil
	})

	type EndpointDetailOptions struct {
		ID string `help:"ID or name of endpoints"`
	}
	R(&EndpointDetailOptions{}, "endpoint-show", "Show details of an endpoint", func(s *mcclient.ClientSession, args *EndpointDetailOptions) error {
		mod, err := modules.GetModule(s, "endpoints")
		if err != nil {
			return err
		}
		ep, err := mod.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(ep)
		return nil
	})

	R(&EndpointDetailOptions{}, "endpoint-delete", "Delete an endpoint", func(s *mcclient.ClientSession, args *EndpointDetailOptions) error {
		mod, err := modules.GetModule(s, "endpoints")
		if err != nil {
			return err
		}
		ep, err := mod.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(ep)
		return nil
	})

	type EndpointCreateOptions struct {
		SERVICE   string `help:"Service ID or Name"`
		REGION    string `help:"Region"`
		INTERFACE string `help:"Interface types" choices:"internal|public|admin"`
		URL       string `help:"URL"`
		Zone      string `help:"Zone"`
		Name      string `help:"Name"`
		Enabled   bool   `help:"Enabled"`
		Disabled  bool   `help:"Disabled"`
	}
	R(&EndpointCreateOptions{}, "endpoint-create", "Create endpoint", func(s *mcclient.ClientSession, args *EndpointCreateOptions) error {
		params := jsonutils.NewDict()
		srvId, err := modules.ServicesV3.GetId(s, args.SERVICE, nil)
		if err != nil {
			return err
		}
		params.Add(jsonutils.NewString(srvId), "service_id")
		regionId, err := modules.Regions.GetId(s, mcclient.RegionID(args.REGION, args.Zone), nil)
		if err != nil {
			return err
		}
		params.Add(jsonutils.NewString(regionId), "region_id")
		params.Add(jsonutils.NewString(args.INTERFACE), "interface")
		params.Add(jsonutils.NewString(args.URL), "url")
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if args.Enabled && !args.Disabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if !args.Enabled && args.Disabled {
			params.Add(jsonutils.JSONFalse, "enabled")
		}
		ep, err := modules.EndpointsV3.Create(s, params)
		if err != nil {
			return err
		}
		printObject(ep)
		return nil
	})

	type EndpointUpdateOptions struct {
		ID       string `help:"ID or name of endpoint"`
		Url      string `help:"URL"`
		Name     string `help:"Name"`
		Enabled  bool   `help:"Enabled"`
		Disabled bool   `help:"Disabled"`
	}
	R(&EndpointUpdateOptions{}, "endpoint-update", "Update a endpoint", func(s *mcclient.ClientSession, args *EndpointUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Url) > 0 {
			params.Add(jsonutils.NewString(args.Url), "url")
		}
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if args.Enabled && !args.Disabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if !args.Enabled && args.Disabled {
			params.Add(jsonutils.JSONFalse, "enabled")
		}
		ep, err := modules.EndpointsV3.Patch(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(ep)
		return nil
	})
}
