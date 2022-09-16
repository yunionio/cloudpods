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
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type WafRuleGroupListOptions struct {
		Scope string `choices:"CLOUDFRONT|REGIONAL" default:"REGIONAL"`
	}

	shellutils.R(&WafRuleGroupListOptions{}, "waf-managed-rule-group-list", "List waf managed rule group", func(cli *aws.SRegion, args *WafRuleGroupListOptions) error {
		groups, err := cli.ListAvailableManagedRuleGroups(args.Scope)
		if err != nil {
			return err
		}
		printList(groups, 0, 0, 0, []string{})
		return nil
	})

	shellutils.R(&WafRuleGroupListOptions{}, "waf-rule-group-list", "List waf rule group", func(cli *aws.SRegion, args *WafRuleGroupListOptions) error {
		groups, err := cli.ListRuleGroups(args.Scope)
		if err != nil {
			return err
		}
		printList(groups, 0, 0, 0, []string{})
		return nil
	})

	type WafRuleGroupShowOptions struct {
		ID    string
		NAME  string
		SCOPE string
	}

	shellutils.R(&WafRuleGroupShowOptions{}, "waf-rule-group-show", "Show waf rule group", func(cli *aws.SRegion, args *WafRuleGroupShowOptions) error {
		group, err := cli.GetRuleGroup(args.ID, args.NAME, args.SCOPE)
		if err != nil {
			return err
		}
		printObject(group)
		return nil
	})

	type WafManagedRuleGroupShowOptions struct {
		NAME       string
		SCOPE      string
		VendorName string `default:"AWS"`
	}

	shellutils.R(&WafManagedRuleGroupShowOptions{}, "waf-managed-rule-group-show", "Show waf rule group", func(cli *aws.SRegion, args *WafManagedRuleGroupShowOptions) error {
		group, err := cli.DescribeManagedRuleGroup(args.NAME, args.SCOPE, args.VendorName)
		if err != nil {
			return err
		}
		printObject(group)
		return nil
	})

	type RuleGroupDeleteOptions struct {
		ID         string
		NAME       string
		SCOPE      string
		LOCK_TOKEN string
	}

	shellutils.R(&RuleGroupDeleteOptions{}, "waf-rule-group-delete", "Delete waf ip set", func(cli *aws.SRegion, args *RuleGroupDeleteOptions) error {
		return cli.DeleteRuleGroup(args.ID, args.NAME, args.SCOPE, args.LOCK_TOKEN)
	})

	type IPSetListOptions struct {
		Scope string `choices:"CLOUDFRONT|REGIONAL" default:"REGIONAL"`
	}

	shellutils.R(&IPSetListOptions{}, "waf-ipset-list", "List waf ip sets", func(cli *aws.SRegion, args *IPSetListOptions) error {
		ipsets, err := cli.ListIPSets(args.Scope)
		if err != nil {
			return err
		}
		printList(ipsets, 0, 0, 0, []string{})
		return nil
	})

	type WafIPSetShowOptions struct {
		ID    string
		NAME  string
		SCOPE string
	}

	shellutils.R(&WafIPSetShowOptions{}, "waf-ipset-show", "Show waf ip sets", func(cli *aws.SRegion, args *WafIPSetShowOptions) error {
		ipset, err := cli.GetIPSet(args.ID, args.NAME, args.SCOPE)
		if err != nil {
			return err
		}
		printObject(ipset)
		return nil
	})

	type WafIPSetDeleteOptions struct {
		ID         string
		NAME       string
		SCOPE      string
		LOCK_TOKEN string
	}

	shellutils.R(&WafIPSetDeleteOptions{}, "waf-ipset-delete", "Delete waf ip set", func(cli *aws.SRegion, args *WafIPSetDeleteOptions) error {
		return cli.DeleteIPSet(args.ID, args.NAME, args.SCOPE, args.LOCK_TOKEN)
	})

	type WafListOptions struct {
		Scope string `choices:"CLOUDFRONT|REGIONAL" default:"REGIONAL"`
	}

	shellutils.R(&WafListOptions{}, "waf-list", "List web acls", func(cli *aws.SRegion, args *WafListOptions) error {
		acls, err := cli.ListWebACLs(args.Scope)
		if err != nil {
			return err
		}
		printList(acls, 0, 0, 0, []string{})
		return nil
	})

	type WafShowOptions struct {
		ID    string
		NAME  string
		SCOPE string
	}

	shellutils.R(&WafShowOptions{}, "waf-show", "Show web acl", func(cli *aws.SRegion, args *WafShowOptions) error {
		webAcl, err := cli.GetWebAcl(args.ID, args.NAME, args.SCOPE)
		if err != nil {
			return err
		}
		printObject(webAcl)
		return nil
	})

	type WafDeleteOptions struct {
		ID         string
		NAME       string
		SCOPE      string
		LOCK_TOKEN string
	}

	shellutils.R(&WafDeleteOptions{}, "waf-delete", "Delete web acl", func(cli *aws.SRegion, args *WafDeleteOptions) error {
		return cli.DeleteWebAcl(args.ID, args.NAME, args.SCOPE, args.LOCK_TOKEN)
	})

	type WafResourceListOptions struct {
		ResType string `choices:"APPLICATION_LOAD_BALANCER|API_GATEWAY|APPSYNC"`
		ARN     string
	}

	shellutils.R(&WafResourceListOptions{}, "waf-res-list", "List web acl resource", func(cli *aws.SRegion, args *WafResourceListOptions) error {
		res, err := cli.ListResourcesForWebACL(args.ResType, args.ARN)
		if err != nil {
			return err
		}
		fmt.Println("res:", res)
		return nil
	})

	type WafAddRuleOptions struct {
		WafShowOptions

		RULE_FILE string
	}

	shellutils.R(&WafAddRuleOptions{}, "waf-add-rule", "Add web acl rule", func(cli *aws.SRegion, args *WafAddRuleOptions) error {
		waf, err := cli.GetWebAcl(args.ID, args.NAME, args.SCOPE)
		if err != nil {
			return errors.Wrapf(err, "GetWebAcl")
		}
		data, err := os.ReadFile(args.RULE_FILE)
		if err != nil {
			return errors.Wrapf(err, "ReadFile")
		}
		params, err := jsonutils.Parse(data)
		if err != nil {
			return errors.Wrapf(err, "Parse")
		}
		rule := &cloudprovider.SWafRule{}
		params.Unmarshal(rule)
		_, err = waf.AddRule(rule)
		return err
	})

}
