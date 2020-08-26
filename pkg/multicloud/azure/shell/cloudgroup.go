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
	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type CloudgroupListOptions struct {
		Name string
	}
	shellutils.R(&CloudgroupListOptions{}, "cloud-group-list", "List cloudgroups", func(cli *azure.SRegion, args *CloudgroupListOptions) error {
		groups, err := cli.GetClient().GetCloudgroups(args.Name)
		if err != nil {
			return err
		}
		printList(groups, 0, 0, 0, nil)
		return nil
	})

	type CloudgroupUserListOptions struct {
		ID string
	}

	shellutils.R(&CloudgroupUserListOptions{}, "cloud-group-user-list", "List group cloudusers", func(cli *azure.SRegion, args *CloudgroupUserListOptions) error {
		users, err := cli.GetClient().ListGroupMemebers(args.ID)
		if err != nil {
			return err
		}
		printList(users, 0, 0, 0, nil)
		return nil
	})

	type CloudgroupIdOptions struct {
		ID string
	}

	shellutils.R(&CloudgroupIdOptions{}, "cloud-group-delete", "Delete cloudgroup", func(cli *azure.SRegion, args *CloudgroupIdOptions) error {
		return cli.GetClient().DeleteGroup(args.ID)
	})

	type CloudgroupCreateOptions struct {
		NAME string
		Desc string
	}

	shellutils.R(&CloudgroupCreateOptions{}, "cloud-group-create", "Create cloudgroup", func(cli *azure.SRegion, args *CloudgroupCreateOptions) error {
		group, err := cli.GetClient().CreateGroup(args.NAME, args.Desc)
		if err != nil {
			return err
		}
		printObject(group)
		return nil
	})

	type CloudgroupUserOptions struct {
		ID        string
		USER_NAME string
	}

	shellutils.R(&CloudgroupUserOptions{}, "cloud-group-remove-user", "Remove user from cloudgroup", func(cli *azure.SRegion, args *CloudgroupUserOptions) error {
		return cli.GetClient().RemoveGroupUser(args.ID, args.USER_NAME)
	})

	shellutils.R(&CloudgroupUserOptions{}, "cloud-group-add-user", "Add user to cloudgroup", func(cli *azure.SRegion, args *CloudgroupUserOptions) error {
		return cli.GetClient().AddGroupUser(args.ID, args.USER_NAME)
	})

}
