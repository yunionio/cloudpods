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
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ServerSecgroupListOptions struct {
		options.BaseListOptions
		Server   string `help:"ID or Name of Server"`
		Secgroup string `help:"Secgroup ID or name"`
	}
	R(&ServerSecgroupListOptions{}, "server-secgroup-list", "List server secgroup pairs", func(s *mcclient.ClientSession, args *ServerSecgroupListOptions) error {
		var params *jsonutils.JSONDict
		{
			param, err := args.BaseListOptions.Params()
			if err != nil {
				return err
			}
			params = param.(*jsonutils.JSONDict)
		}
		var result *printutils.ListResult
		var err error
		if len(args.Server) > 0 {
			result, err = modules.Serversecgroups.ListDescendent(s, args.Server, params)
		} else if len(args.Secgroup) > 0 {
			result, err = modules.Serversecgroups.ListDescendent2(s, args.Secgroup, params)
		} else {
			result, err = modules.Serversecgroups.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Serversecgroups.GetColumns(s))
		return nil
	})

	type ServerSecgroupDetailOptions struct {
		SERVER   string `help:"ID or Name of Server"`
		SECGROUP string `help:"ID or Name of Security Group"`
	}
	R(&ServerSecgroupDetailOptions{}, "server-secgroup-show", "Show server security group details", func(s *mcclient.ClientSession, args *ServerSecgroupDetailOptions) error {
		query := jsonutils.NewDict()
		result, err := modules.Serversecgroups.Get(s, args.SERVER, args.SECGROUP, query)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
