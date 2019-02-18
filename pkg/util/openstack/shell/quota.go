package shell

import (
	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type QuotaOptions struct {
	}
	shellutils.R(&QuotaOptions{}, "quota-show", "Show quota", func(cli *openstack.SRegion, args *QuotaOptions) error {
		quota, err := cli.GetQuota()
		if err != nil {
			return err
		}
		printObject(quota)
		return nil
	})

	shellutils.R(&openstack.SQuota{}, "quota-set", "Set quota", func(cli *openstack.SRegion, args *openstack.SQuota) error {
		return cli.SetQuota(args)
	})

}
