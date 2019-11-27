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
	"yunion.io/x/onecloud/pkg/multicloud/google"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ResourcePolicyListOptions struct {
		Disk       string
		MaxResults int
		PageToken  string
	}
	shellutils.R(&ResourcePolicyListOptions{}, "resource-policy-list", "List resourcepolicys", func(cli *google.SRegion, args *ResourcePolicyListOptions) error {
		resourcepolicys, err := cli.GetResourcePolicies(args.MaxResults, args.PageToken)
		if err != nil {
			return err
		}
		printList(resourcepolicys, 0, 0, 0, nil)
		return nil
	})

	type ResourcePolicyShowOptions struct {
		ID string
	}
	shellutils.R(&ResourcePolicyShowOptions{}, "resource-policy-show", "Show resourcepolicy", func(cli *google.SRegion, args *ResourcePolicyShowOptions) error {
		resourcepolicy, err := cli.GetResourcePolicy(args.ID)
		if err != nil {
			return err
		}
		printObject(resourcepolicy)
		return nil
	})

}
