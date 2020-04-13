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

package cloudid

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type CloudgroupUserListOptions struct {
		options.BaseListOptions
		Cloudgroup string `help:"ID or Name of Cloudgroup"`
		Clouduser  string `help:"User ID or name"`
	}
	R(&CloudgroupUserListOptions{}, "cloud-group-user-list", "List cloudgroup clouduser pairs", func(s *mcclient.ClientSession, args *CloudgroupUserListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		var result *modulebase.ListResult
		var err error
		if len(args.Cloudgroup) > 0 {
			result, err = modules.Cloudgroupusers.ListDescendent(s, args.Cloudgroup, params)
		} else if len(args.Clouduser) > 0 {
			result, err = modules.Cloudgroupusers.ListDescendent2(s, args.Clouduser, params)
		} else {
			result, err = modules.Cloudgroupusers.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Cloudgroupusers.GetColumns(s))
		return nil
	})

	type CloudgroupUserDetailOptions struct {
		CLOUDUSER   string `help:"ID or Name of Cloudgroup"`
		CLOUDPOLICY string `help:"ID or Name of Clouduser"`
	}
	R(&CloudgroupUserDetailOptions{}, "cloud-group-user-show", "Show cloudgroupuser details", func(s *mcclient.ClientSession, args *CloudgroupUserDetailOptions) error {
		query := jsonutils.NewDict()
		result, err := modules.Cloudgroupusers.Get(s, args.CLOUDUSER, args.CLOUDPOLICY, query)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
