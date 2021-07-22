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

package itsm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type ExtraUserCreateOptions struct {
		UserName string `help:"user name" required:"true"`
		Pwd      string `help:"password" required:"true"`
		Url      string `help:"url" required:"true"`
		Type     string `help:"extra order type" required:"true"`
	}
	R(&ExtraUserCreateOptions{}, "itsm_extra_user_create", "extra user create",
		func(s *mcclient.ClientSession, args *ExtraUserCreateOptions) error {
			params := jsonutils.Marshal(args)
			ret, err := modules.ExtraUsers.Create(s, params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})
	type ExtraUserListOptions struct {
		ID string `help:"ID of user"`
	}
	R(&ExtraUserListOptions{}, "itsm_extra_user_list", "extra user list",
		func(s *mcclient.ClientSession, args *ExtraUserListOptions) error {
			params := jsonutils.NewDict()

			if len(args.ID) > 0 {
				params.Add(jsonutils.NewString(args.ID), "id")
			}
			ret, err := modules.ExtraUsers.List(s, params)
			if err != nil {
				return err
			}
			printList(ret, modules.ExtraUsers.GetColumns(s))
			return nil
		})
}
