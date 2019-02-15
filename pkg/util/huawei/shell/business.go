package shell

import (
	"yunion.io/x/onecloud/pkg/util/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type AccountBalanceOptions struct {
	}
	shellutils.R(&AccountBalanceOptions{}, "balance", "Get account balance", func(cli *huawei.SRegion, args *AccountBalanceOptions) error {
		result, err := cli.GetClient().QueryAccountBalance()
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
