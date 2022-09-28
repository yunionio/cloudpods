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
	type ClouduserListOptions struct {
	}
	shellutils.R(&ClouduserListOptions{}, "cloud-user-list", "List cloudusers", func(cli *azure.SRegion, args *ClouduserListOptions) error {
		users, err := cli.GetClient().GetCloudusers()
		if err != nil {
			return err
		}
		printList(users, 0, 0, 0, nil)
		return nil
	})

	type ClouduserIdOptions struct {
		ID string
	}

	shellutils.R(&ClouduserIdOptions{}, "cloud-user-delete", "Delete clouduser", func(cli *azure.SRegion, args *ClouduserIdOptions) error {
		return cli.GetClient().DeleteClouduser(args.ID)
	})

	shellutils.R(&ClouduserIdOptions{}, "cloud-user-group-list", "List clouduser groups", func(cli *azure.SRegion, args *ClouduserIdOptions) error {
		groups, err := cli.GetClient().GetUserGroups(args.ID)
		if err != nil {
			return err
		}
		printList(groups, 0, 0, 0, nil)
		return nil
	})

	type ClouduserCreateOptions struct {
		NAME     string
		Password string
	}

	shellutils.R(&ClouduserCreateOptions{}, "cloud-user-create", "Create clouduser", func(cli *azure.SRegion, args *ClouduserCreateOptions) error {
		user, err := cli.GetClient().CreateClouduser(args.NAME, args.Password)
		if err != nil {
			return err
		}
		printObject(user)
		return nil
	})

	type ClouduserResetPasswordOptions struct {
		NAME     string
		PASSWORD string
	}

	shellutils.R(&ClouduserResetPasswordOptions{}, "cloud-user-reset-password", "Reset clouduser password", func(cli *azure.SRegion, args *ClouduserResetPasswordOptions) error {
		return cli.GetClient().ResetClouduserPassword(args.NAME, args.PASSWORD)
	})

	type DomainListOptions struct {
	}

	shellutils.R(&DomainListOptions{}, "domain-list", "List domains", func(cli *azure.SRegion, args *DomainListOptions) error {
		domains, err := cli.GetClient().GetDomains()
		if err != nil {
			return err
		}
		printList(domains, 0, 0, 0, nil)
		return nil
	})

	type GroupUserList struct {
	}

	shellutils.R(&GroupUserList{}, "graph-user-list", "List graph users", func(cli *azure.SRegion, args *GroupUserList) error {
		users, err := cli.GetClient().ListGraphUsers()
		if err != nil {
			return err
		}
		printObject(users)
		return nil
	})

}
