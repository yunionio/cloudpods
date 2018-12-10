package shell

import (
	"yunion.io/x/onecloud/pkg/util/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type LoadbalancerListOptions struct {
		Ids []string `help:"Loadbalancer ids"`
	}
	shellutils.R(&LoadbalancerListOptions{}, "lb-list", "List loadbalance", func(cli *aliyun.SRegion, args *LoadbalancerListOptions) error {
		lbs, err := cli.GetLoadbalancers(args.Ids)
		if err != nil {
			return err
		}
		printList(lbs, len(lbs), 0, 0, []string{})
		return nil
	})
}
