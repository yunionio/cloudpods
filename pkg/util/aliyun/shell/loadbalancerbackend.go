package shell

import (
	"yunion.io/x/onecloud/pkg/util/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type LoadbalancerBackendListOptions struct {
		GROUPID string `help:"LoadbalancerBackendgroup ID"`
	}
	shellutils.R(&LoadbalancerBackendListOptions{}, "lb-backend-list", "List loadbalanceBackends", func(cli *aliyun.SRegion, args *LoadbalancerBackendListOptions) error {
		backends, err := cli.GetLoadbalancerBackends(args.GROUPID)
		if err != nil {
			return err
		}
		printList(backends, len(backends), 0, 0, []string{})
		return nil
	})

	shellutils.R(&LoadbalancerBackendListOptions{}, "lb-master-slave-backend-list", "List loadbalanceMasterSlaveBackends", func(cli *aliyun.SRegion, args *LoadbalancerBackendListOptions) error {
		backends, err := cli.GetLoadbalancerMasterSlaveBackends(args.GROUPID)
		if err != nil {
			return err
		}
		printList(backends, len(backends), 0, 0, []string{})
		return nil
	})
}
