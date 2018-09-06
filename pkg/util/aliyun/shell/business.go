package shell

import (
	"yunion.io/x/onecloud/pkg/util/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type AccountBalanceOptions struct {
	}
	shellutils.R(&AccountBalanceOptions{}, "balance", "Get account balance", func(cli *aliyun.SRegion, args *AccountBalanceOptions) error {
		result1, err := cli.GetClient().QueryAccountBalance()
		if err != nil {
			return err
		}
		printObject(result1)

		result2, err := cli.GetClient().QueryCashCoupons()
		if err != nil {
			return err
		}
		printList(result2, len(result2), 0, 0, nil)

		result3, err := cli.GetClient().QueryPrepaidCards()
		if err != nil {
			return err
		}
		printList(result3, len(result3), 0, 0, nil)
		return nil
	})
}
