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
	type CloudpolicyListOptions struct {
		Name       string
		PolicyType string `choices:"CustomRole|BuiltInRole"`
	}
	shellutils.R(&CloudpolicyListOptions{}, "cloud-policy-list", "List cloudpolicies", func(cli *azure.SRegion, args *CloudpolicyListOptions) error {
		roles, err := cli.GetClient().GetRoles(args.Name, args.PolicyType)
		if err != nil {
			return err
		}
		printList(roles, 0, 0, 0, nil)
		return nil
	})

	type CloudpolicyAssignOption struct {
		OBJECT         string
		ROLE           string
		SubscriptionId string
	}

	shellutils.R(&CloudpolicyAssignOption{}, "cloud-policy-assign-object", "Assign cloudpolicy for object", func(cli *azure.SRegion, args *CloudpolicyAssignOption) error {
		return cli.GetClient().AssignPolicy(args.OBJECT, args.ROLE, args.SubscriptionId)
	})

	type AssignmentListOption struct {
		ObjectId string
	}

	shellutils.R(&AssignmentListOption{}, "assignment-list", "List role assignments", func(cli *azure.SRegion, args *AssignmentListOption) error {
		assignments, err := cli.GetClient().GetAssignments(args.ObjectId)
		if err != nil {
			return err
		}
		printList(assignments, 0, 0, 0, nil)
		return nil
	})

	type AssignmentIdOption struct {
		ID string
	}

	shellutils.R(&AssignmentIdOption{}, "assignment-delete", "Delete role assignment", func(cli *azure.SRegion, args *AssignmentIdOption) error {
		return cli.GetClient().Delete(args.ID)
	})

	type ObjectPolicyListOptions struct {
		OBJECT string
	}

	shellutils.R(&ObjectPolicyListOptions{}, "object-policy-list", "List object policies", func(cli *azure.SRegion, args *ObjectPolicyListOptions) error {
		policies, err := cli.GetClient().GetCloudpolicies(args.OBJECT)
		if err != nil {
			return err
		}
		printList(policies, 0, 0, 0, nil)
		return nil
	})

}
