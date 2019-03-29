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

	/**
	 * 获得一个全部通信地址组
	 */
	type ContactGroupsListOptions struct {
		options.BaseListOptions
	}
	R(&ContactGroupsListOptions{}, "contact-group-list", "List all contact groups for all the domainsconta", func(s *mcclient.ClientSession, args *ContactGroupsListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}

		result, err := modules.ContactGroups.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.ContactGroups.GetColumns(s))
		return nil
	})

}
