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
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ClouduserListOptions struct {
	}
	shellutils.R(&ClouduserListOptions{}, "cloud-user-list", "List cloudusers", func(cli *qcloud.SRegion, args *ClouduserListOptions) error {
		users, err := cli.GetClient().ListUsers()
		if err != nil {
			return err
		}
		printList(users, 0, 0, 0, nil)
		return nil
	})

	type ClouduserOptions struct {
		USER string
	}
	shellutils.R(&ClouduserOptions{}, "cloud-user-delete", "Delete clouduser", func(cli *qcloud.SRegion, args *ClouduserOptions) error {
		return cli.GetClient().DeleteUser(args.USER)
	})

	shellutils.R(&ClouduserOptions{}, "cloud-user-show", "Show clouduser", func(cli *qcloud.SRegion, args *ClouduserOptions) error {
		user, err := cli.GetClient().GetUser(args.USER)
		if err != nil {
			return err
		}
		printObject(user)
		return nil
	})

	type ClouduserCreateOptions struct {
		NAME         string
		Password     string
		Desc         string
		ConsoleLogin bool
	}

	shellutils.R(&ClouduserCreateOptions{}, "cloud-user-create", "Create clouduser", func(cli *qcloud.SRegion, args *ClouduserCreateOptions) error {
		user, err := cli.GetClient().AddUser(args.NAME, args.Password, args.Desc, args.ConsoleLogin)
		if err != nil {
			return err
		}
		printObject(user)
		return nil
	})

	type ClouduserPolicyOptions struct {
		UIN       string
		POLICY_ID string
	}

	shellutils.R(&ClouduserPolicyOptions{}, "cloud-user-attach-policy", "Attach policy for clouduser", func(cli *qcloud.SRegion, args *ClouduserPolicyOptions) error {
		return cli.GetClient().AttachUserPolicy(args.UIN, args.POLICY_ID)
	})

	shellutils.R(&ClouduserPolicyOptions{}, "cloud-user-detach-policy", "Detach policy from clouduser", func(cli *qcloud.SRegion, args *ClouduserPolicyOptions) error {
		return cli.GetClient().DetachUserPolicy(args.UIN, args.POLICY_ID)
	})

	type ClouduserPolicyListOptions struct {
		UIN    string
		Offset int
		Limit  int
	}

	shellutils.R(&ClouduserPolicyListOptions{}, "cloud-user-policy-list", "List policy from clouduser", func(cli *qcloud.SRegion, args *ClouduserPolicyListOptions) error {
		policies, _, err := cli.GetClient().ListAttachedUserPolicies(args.UIN, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(policies, 0, 0, 0, nil)
		return nil
	})

	type ClouduserGroupListOptions struct {
		UIN    int
		Offset int
		Limit  int
	}

	shellutils.R(&ClouduserGroupListOptions{}, "cloud-user-group-list", "List clouduser groups", func(cli *qcloud.SRegion, args *ClouduserGroupListOptions) error {
		groups, _, err := cli.GetClient().ListGroupsForUser(args.UIN, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(groups, 0, 0, 0, nil)
		return nil
	})

}
