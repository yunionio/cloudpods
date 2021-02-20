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
	"fmt"

	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type AccountListOptions struct {
	}
	shellutils.R(&AccountListOptions{}, "account-list", "List accounts", func(cli *aws.SRegion, args *AccountListOptions) error {
		accounts, err := cli.ListAccounts()
		if err != nil {
			return err
		}
		printList(accounts, 0, 0, 0, []string{})
		return nil
	})

	type OrganizationPoliciesListOptions struct {
		FILTER string `json:"filter" choices:"SERVICE_CONTROL_POLICY|TAG_POLICY|BACKUP_POLICY|AISERVICES_OPT_OUT_POLICY"`
	}
	shellutils.R(&OrganizationPoliciesListOptions{}, "org-policy-list", "List policies of organizations", func(cli *aws.SRegion, args *OrganizationPoliciesListOptions) error {
		policies, err := cli.ListPolicies(args.FILTER)
		if err != nil {
			return err
		}
		printList(policies, 0, 0, 0, []string{})
		return nil
	})

	type OrganizationPolicyShowOptions struct {
		ID string `json:"id"`
	}
	shellutils.R(&OrganizationPolicyShowOptions{}, "org-policy-show", "Show details of an organizations policy", func(cli *aws.SRegion, args *OrganizationPolicyShowOptions) error {
		content, err := cli.DescribeOrgPolicy(args.ID)
		if err != nil {
			return err
		}
		fmt.Println(content.PrettyString())
		return nil
	})

	type OrganizationPoliciesListForTargetOptions struct {
		OrganizationPoliciesListOptions
		TARGET string `json:"target"`
	}
	shellutils.R(&OrganizationPoliciesListForTargetOptions{}, "org-target-policy-list", "List policies for target of organizations", func(cli *aws.SRegion, args *OrganizationPoliciesListForTargetOptions) error {
		policies, err := cli.ListPoliciesForTarget(args.FILTER, args.TARGET)
		if err != nil {
			return err
		}
		printList(policies, 0, 0, 0, []string{})
		return nil
	})

	type OrganizationParentsListOptions struct {
		ID string `json:"id"`
	}
	shellutils.R(&OrganizationParentsListOptions{}, "org-parent-list", "List parent nodes of a child node", func(cli *aws.SRegion, args *OrganizationParentsListOptions) error {
		err := cli.ListParents(args.ID)
		if err != nil {
			return err
		}
		return nil
	})

	type OrganizationalUnitShowOptions struct {
		ID string `help:"Id of organizational unit"`
	}
	shellutils.R(&OrganizationalUnitShowOptions{}, "org-ou-show", "Show details of organizational unit", func(cli *aws.SRegion, args *OrganizationalUnitShowOptions) error {
		err := cli.DescribeOrganizationalUnit(args.ID)
		if err != nil {
			return err
		}
		return nil
	})
}
