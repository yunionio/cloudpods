package shell

import (
	"yunion.io/x/onecloud/pkg/util/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type LoadbalancerListOptions struct {
		Ids []string `help:"Loadbalancer ids"`
	}
	shellutils.R(&LoadbalancerListOptions{}, "lb-list", "List loadbalancers", func(cli *aliyun.SRegion, args *LoadbalancerListOptions) error {
		lbs, err := cli.GetLoadbalancers(args.Ids)
		if err != nil {
			return err
		}
		printList(lbs, len(lbs), 0, 0, []string{})
		return nil
	})

	type LoadbalancerOptions struct {
		ID string `help:"ID of loadbalancer"`
	}
	shellutils.R(&LoadbalancerOptions{}, "lb-show", "Show loadbalancer", func(cli *aliyun.SRegion, args *LoadbalancerOptions) error {
		lb, err := cli.GetLoadbalancerDetail(args.ID)
		if err != nil {
			return err
		}
		printObject(lb)
		return nil
	})

}
