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
	type ClouduserPolicyListOptions struct {
		options.BaseListOptions
		Clouduser   string `help:"ID or Name of Clouduser"`
		Cloudpolicy string `help:"Policy ID or name"`
	}
	R(&ClouduserPolicyListOptions{}, "cloud-user-policy-list", "List clouduser cloudpolicy pairs", func(s *mcclient.ClientSession, args *ClouduserPolicyListOptions) error {
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
		if len(args.Clouduser) > 0 {
			result, err = modules.Clouduserpolicies.ListDescendent(s, args.Clouduser, params)
		} else if len(args.Cloudpolicy) > 0 {
			result, err = modules.Clouduserpolicies.ListDescendent2(s, args.Cloudpolicy, params)
		} else {
			result, err = modules.Clouduserpolicies.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Clouduserpolicies.GetColumns(s))
		return nil
	})

	type ClouduserPolicyDetailOptions struct {
		CLOUDUSER   string `help:"ID or Name of Clouduser"`
		CLOUDPOLICY string `help:"ID or Name of Cloudpolicy"`
	}
	R(&ClouduserPolicyDetailOptions{}, "cloud-user-policy-show", "Show clouduserpolicy details", func(s *mcclient.ClientSession, args *ClouduserPolicyDetailOptions) error {
		query := jsonutils.NewDict()
		result, err := modules.Clouduserpolicies.Get(s, args.CLOUDUSER, args.CLOUDPOLICY, query)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
