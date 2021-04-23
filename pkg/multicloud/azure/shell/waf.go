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
	type RuleGroupListOptions struct {
	}
	shellutils.R(&RuleGroupListOptions{}, "waf-rule-group-list", "List waf rule groups", func(cli *azure.SRegion, args *RuleGroupListOptions) error {
		groups, err := cli.ListAppWafManagedRuleGroup()
		if err != nil {
			return err
		}
		printList(groups, len(groups), 0, 0, []string{})
		return nil
	})

	type FrontDoorPolicyListOptions struct {
		RESOURCE_GROUP string
	}
	shellutils.R(&FrontDoorPolicyListOptions{}, "front-door-policy-list", "List front door policies", func(cli *azure.SRegion, args *FrontDoorPolicyListOptions) error {
		policies, err := cli.ListFrontDoorWafs(args.RESOURCE_GROUP)
		if err != nil {
			return err
		}
		printList(policies, 0, 0, 0, []string{})
		return nil
	})

	type AppGatewayWafListOptions struct {
	}

	shellutils.R(&AppGatewayWafListOptions{}, "app-gateway-waf-list", "List app gateway wafs", func(cli *azure.SRegion, args *AppGatewayWafListOptions) error {
		wafs, err := cli.ListAppWafs()
		if err != nil {
			return err
		}
		printList(wafs, 0, 0, 0, []string{})
		return nil
	})

	type AppGatewayWafRuleGroupListOptions struct {
	}

	shellutils.R(&AppGatewayWafRuleGroupListOptions{}, "app-gateway-waf-rule-group-list", "List app gateway wafs", func(cli *azure.SRegion, args *AppGatewayWafRuleGroupListOptions) error {
		group, err := cli.ListAppWafManagedRuleGroup()
		if err != nil {
			return err
		}
		printList(group, 0, 0, 0, []string{})
		return nil
	})

}
