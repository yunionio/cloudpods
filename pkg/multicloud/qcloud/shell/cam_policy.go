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
	type CloudpolicyListOption struct {
		Keyword string
		Scope   string `choices:"QCS|Local"`
		Offset  int
		Limit   int
	}

	shellutils.R(&CloudpolicyListOption{}, "cloud-policy-list", "List cloudpolicy", func(cli *qcloud.SRegion, args *CloudpolicyListOption) error {
		policies, _, err := cli.GetClient().ListPolicies(args.Keyword, args.Scope, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(policies, 0, 0, 0, nil)
		return nil
	})

	type CloudpolicyShowOption struct {
		POLICY_ID string
	}

	shellutils.R(&CloudpolicyShowOption{}, "cloud-policy-show", "Show cloudpolicy", func(cli *qcloud.SRegion, args *CloudpolicyShowOption) error {
		policy, err := cli.GetClient().GetPolicy(args.POLICY_ID)
		if err != nil {
			return err
		}
		printObject(policy)
		return nil
	})

	type CloudpolicyIdsOptions struct {
		IDS []int
	}

	shellutils.R(&CloudpolicyIdsOptions{}, "cloud-policy-delete", "Delete cloudpolicies", func(cli *qcloud.SRegion, args *CloudpolicyIdsOptions) error {
		return cli.GetClient().DeletePolicy(args.IDS)
	})

	type CloudpolicyCreateOptions struct {
		NAME     string
		DOCUMENT string
		Desc     string
	}

	shellutils.R(&CloudpolicyCreateOptions{}, "cloud-policy-create", "Create cloudpolicy", func(cli *qcloud.SRegion, args *CloudpolicyCreateOptions) error {
		policy, err := cli.GetClient().CreatePolicy(args.NAME, args.DOCUMENT, args.Desc)
		if err != nil {
			return err
		}
		printObject(policy)
		return nil
	})

	type CloudpolicyUpdateOptions struct {
		ID       int
		Document string
		Desc     string
	}

	shellutils.R(&CloudpolicyUpdateOptions{}, "cloud-policy-update", "Update cloudpolicy", func(cli *qcloud.SRegion, args *CloudpolicyUpdateOptions) error {
		return cli.GetClient().UpdatePolicy(args.ID, args.Document, args.Desc)
	})

}
