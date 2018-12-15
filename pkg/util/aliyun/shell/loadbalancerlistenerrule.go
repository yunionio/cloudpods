package shell

import (
	"yunion.io/x/onecloud/pkg/util/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type LoadbalancerListenerRuleListOptions struct {
		ID   string `help:"ID of loadbalaner"`
		PORT int    `help:"Port of listener port"`
	}
	shellutils.R(&LoadbalancerListenerRuleListOptions{}, "lb-listener-rule-list", "List LoadbalancerListenerRules", func(cli *aliyun.SRegion, args *LoadbalancerListenerRuleListOptions) error {
		rules, err := cli.GetLoadbalancerListenerRules(args.ID, args.PORT)
		if err != nil {
			return err
		}
		printList(rules, len(rules), 0, 0, []string{})
		return nil
	})
}
