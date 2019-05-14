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
	type ServiceListOptions struct {
		Limit  int64  `help:"Limit, default 0, i.e. no limit" default:"20"`
		Offset int64  `help:"Offset, default 0, i.e. no offset"`
		Name   string `help:"Search by name"`
		Type   string `help:"Search by type"`
		Search string `help:"search any fields"`
	}
	R(&ServiceListOptions{}, "service-list", "List services", func(s *mcclient.ClientSession, args *ServiceListOptions) error {
		query := jsonutils.NewDict()
		if args.Limit > 0 {
			query.Add(jsonutils.NewInt(args.Limit), "limit")
		}
		if args.Offset > 0 {
			query.Add(jsonutils.NewInt(args.Offset), "offset")
		}
		if len(args.Name) > 0 {
			query.Add(jsonutils.NewString(args.Name), "name__icontains")
		}
		if len(args.Type) > 0 {
			query.Add(jsonutils.NewString(args.Type), "type__icontains")
		}
		if len(args.Search) > 0 {
			query.Add(jsonutils.NewString(args.Search), "search")
		}
		result, err := modules.ServicesV3.List(s, query)
		if err != nil {
			return err
		}
		printList(result, modules.ServicesV3.GetColumns(s))
		return nil
	})

	type ServiceShowOptions struct {
		ID string `help:"ID of service"`
	}
	R(&ServiceShowOptions{}, "service-show", "Show details of a service", func(s *mcclient.ClientSession, args *ServiceShowOptions) error {
		srvId, err := modules.ServicesV3.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		result, err := modules.ServicesV3.Get(s, srvId, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	R(&ServiceShowOptions{}, "service-delete", "Delete a service", func(s *mcclient.ClientSession, args *ServiceShowOptions) error {
		srvId, err := modules.ServicesV3.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		result, err := modules.ServicesV3.Delete(s, srvId, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServiceCreateOptions struct {
		TYPE     string `help:"Service type"`
		NAME     string `help:"Service name"`
		Desc     string `help:"Description"`
		Enabled  bool   `help:"Enabeld"`
		Disabled bool   `help:"Disabled"`
	}
	R(&ServiceCreateOptions{}, "service-create", "Create a service", func(s *mcclient.ClientSession, args *ServiceCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.TYPE), "type")
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.Enabled && !args.Disabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if !args.Enabled && args.Disabled {
			params.Add(jsonutils.JSONFalse, "enabled")
		}
		srv, err := modules.ServicesV3.Create(s, params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ServiceUpdateOptions struct {
		ID       string `help:"ID or name of the service"`
		Type     string `help:"Service type"`
		Name     string `help:"Service name"`
		Desc     string `help:"Description"`
		Enabled  bool   `help:"Enabeld"`
		Disabled bool   `help:"Disabled"`
	}
	R(&ServiceUpdateOptions{}, "service-update", "Update a service", func(s *mcclient.ClientSession, args *ServiceUpdateOptions) error {
		srvId, err := modules.ServicesV3.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		if len(args.Type) > 0 {
			params.Add(jsonutils.NewString(args.Type), "type")
		}
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.Enabled && !args.Disabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if !args.Enabled && args.Disabled {
			params.Add(jsonutils.JSONFalse, "enabled")
		}
		srv, err := modules.ServicesV3.Patch(s, srvId, params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})
}
