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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type NasAccessGroupListOptions struct {
		FileSystemType string `choices:"standard|extreme" default:"standard"`
	}
	shellutils.R(&NasAccessGroupListOptions{}, "access-group-list", "List Nas AccessGroups", func(cli *aliyun.SRegion, args *NasAccessGroupListOptions) error {
		ags, err := cli.GetAccessGroups(args.FileSystemType)
		if err != nil {
			return err
		}
		printList(ags, 0, 0, 0, []string{})
		return nil
	})

	shellutils.R(&cloudprovider.SAccessGroup{}, "access-group-create", "Create Nas AccessGroup", func(cli *aliyun.SRegion, args *cloudprovider.SAccessGroup) error {
		return cli.CreateAccessGroup(args)
	})

	type AccessGroupDeleteOptions struct {
		FileSystemType string
		NAME           string
	}

	shellutils.R(&AccessGroupDeleteOptions{}, "access-group-delete", "Delete AccessGroup", func(cli *aliyun.SRegion, args *AccessGroupDeleteOptions) error {
		return cli.DeleteAccessGroup(args.FileSystemType, args.NAME)
	})

	type NasAccessGroupRuleListOptions struct {
		GROUP    string
		PageSize int `help:"page size"`
		PageNum  int `help:"page num"`
	}

	shellutils.R(&NasAccessGroupRuleListOptions{}, "access-group-rule-list", "List Nas AccessGroup Rules", func(cli *aliyun.SRegion, args *NasAccessGroupRuleListOptions) error {
		rules, _, err := cli.GetAccessGroupRules(args.GROUP, args.PageSize, args.PageNum)
		if err != nil {
			return err
		}
		printList(rules, 0, 0, 0, []string{})
		return nil
	})

	type AccessRuleDeleteOptions struct {
		FileSystemType string
		GROUP          string
		RULE_ID        string
	}

	shellutils.R(&AccessRuleDeleteOptions{}, "access-group-rule-delete", "Delete AccessGroup Rule", func(cli *aliyun.SRegion, args *AccessRuleDeleteOptions) error {
		return cli.DeleteAccessGroupRule(args.FileSystemType, args.GROUP, args.RULE_ID)
	})

	type AccessRuleCreateOptions struct {
		SOURCE         string
		FileSystemType string
		GroupName      string
		RwType         cloudprovider.TRWAccessType
		UserType       cloudprovider.TUserAccessType
		Priority       int
	}

	shellutils.R(&AccessRuleCreateOptions{}, "access-group-rule-create", "Delete AccessGroup Rule", func(cli *aliyun.SRegion, args *AccessRuleCreateOptions) error {
		return cli.CreateAccessGroupRule(args.SOURCE, args.FileSystemType, args.GroupName, args.RwType, args.UserType, args.Priority)
	})

}
