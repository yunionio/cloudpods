package shell

import (
	"yunion.io/x/onecloud/pkg/util/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type LoadbalancerACLListOptions struct {
	}
	shellutils.R(&LoadbalancerACLListOptions{}, "lb-acl-list", "List loadbalanceAcls", func(cli *aliyun.SRegion, args *LoadbalancerACLListOptions) error {
		acls, err := cli.GetLoadBalancerAcls()
		if err != nil {
			return err
		}
		printList(acls, len(acls), 0, 0, []string{})
		return nil
	})
}
