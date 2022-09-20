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
	huawei "yunion.io/x/onecloud/pkg/multicloud/hcso"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ClouduserListOptions struct {
		Name string
	}
	shellutils.R(&ClouduserListOptions{}, "cloud-user-list", "List cloudusers", func(cli *huawei.SRegion, args *ClouduserListOptions) error {
		users, err := cli.GetClient().GetCloudusers(args.Name)
		if err != nil {
			return err
		}
		printList(users, 0, 0, 0, nil)
		return nil
	})

	type ClouduserIdOptions struct {
		ID string
	}

	shellutils.R(&ClouduserIdOptions{}, "cloud-user-delete", "Delete clouduser", func(cli *huawei.SRegion, args *ClouduserIdOptions) error {
		return cli.GetClient().DeleteClouduser(args.ID)
	})

	shellutils.R(&ClouduserIdOptions{}, "cloud-user-group-list", "List clouduser groups", func(cli *huawei.SRegion, args *ClouduserIdOptions) error {
		groups, err := cli.GetClient().ListUserGroups(args.ID)
		if err != nil {
			return err
		}
		printList(groups, 0, 0, 0, nil)
		return nil
	})

	type ClouduserCreateOptions struct {
		NAME     string
		Password string
		Desc     string
	}

	shellutils.R(&ClouduserCreateOptions{}, "cloud-user-create", "Create clouduser", func(cli *huawei.SRegion, args *ClouduserCreateOptions) error {
		user, err := cli.GetClient().CreateClouduser(args.NAME, args.Password, args.Desc)
		if err != nil {
			return err
		}
		printObject(user)
		return nil
	})

	type RoleListOptions struct {
		DomainId string
		Name     string
	}

	shellutils.R(&RoleListOptions{}, "cloud-policy-list", "List role", func(cli *huawei.SRegion, args *RoleListOptions) error {
		roles, err := cli.GetClient().GetRoles(args.DomainId, args.Name)
		if err != nil {
			return err
		}
		printList(roles, 0, 0, 0, nil)
		return nil
	})

	type ClouduserResetPassword struct {
		ID       string
		PASSWORD string
	}

	shellutils.R(&ClouduserResetPassword{}, "cloud-user-reset-password", "Reset clouduser password", func(cli *huawei.SRegion, args *ClouduserResetPassword) error {
		return cli.GetClient().ResetClouduserPassword(args.ID, args.PASSWORD)
	})

	type ClouduserAKSKOptions struct {
		Id string
	}
	shellutils.R(&ClouduserAKSKOptions{}, "aksk-list", "List AKSK", func(cli *huawei.SRegion, args *ClouduserAKSKOptions) error {
		res, err := cli.GetClient().GetAKSK(args.Id)
		if err != nil {
			return err
		}
		printList(res, len(res), 0, 0, []string{})
		return nil
	})
	type ClouduserAKSKDeleteOptions struct {
		AccessKey string
	}
	shellutils.R(&ClouduserAKSKDeleteOptions{}, "aksk-delete", "Delete AKSK", func(cli *huawei.SRegion, args *ClouduserAKSKDeleteOptions) error {
		err := cli.GetClient().DeleteAKSK(args.AccessKey)
		return err
	})

	type AKSKCreateOption struct {
		UserId      string `help:"user id"`
		Description string `help:"description"`
	}

	shellutils.R(&AKSKCreateOption{}, "aksk-create", "Create aksk", func(cli *huawei.SRegion, args *AKSKCreateOption) error {
		res, err := cli.GetClient().CreateAKSK(args.UserId, args.Description)
		if err != nil {
			return err
		}
		printObject(res)
		return nil
	})
}
