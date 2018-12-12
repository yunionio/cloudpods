package shell

import (
	"yunion.io/x/onecloud/pkg/util/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type LoadbalancerBackendgroupListOptions struct {
		ID string `help:"ID of Loadbalancer"`
	}
	shellutils.R(&LoadbalancerBackendgroupListOptions{}, "lb-backendgroup-list", "List loadbalancerBackendgroups", func(cli *aliyun.SRegion, args *LoadbalancerBackendgroupListOptions) error {
		backendgroups, err := cli.GetLoadbalancerBackendgroups(args.ID)
		if err != nil {
			return err
		}
		printList(backendgroups, len(backendgroups), 0, 0, []string{})
		return nil
	})

	shellutils.R(&LoadbalancerBackendgroupListOptions{}, "lb-master-slave-backendgroup-list", "List loadbalancerMasterSlaveBackendgroups", func(cli *aliyun.SRegion, args *LoadbalancerBackendgroupListOptions) error {
		backendgroups, err := cli.GetLoadbalancerMasterSlaveBackendgroups(args.ID)
		if err != nil {
			return err
		}
		printList(backendgroups, len(backendgroups), 0, 0, []string{})
		return nil
	})

}
