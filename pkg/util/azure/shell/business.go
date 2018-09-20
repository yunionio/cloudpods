package shell

import (
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type AccountBalanceOptions struct {
	}
	shellutils.R(&AccountBalanceOptions{}, "balance", "Get account balance", func(cli *azure.SRegion, args *AccountBalanceOptions) error {
		if result1, err := cli.GetClient().QueryAccountBalance(); err != nil {
			return err
		} else if result1 != nil {
			printObject(result1)
			return nil
		}

		// result2, err := cli.GetClient().QueryCashCoupons()
		// if err != nil {
		// 	return err
		// }
		// printList(result2, len(result2), 0, 0, nil)

		// result3, err := cli.GetClient().QueryPrepaidCards()
		// if err != nil {
		// 	return err
		// }
		// printList(result3, len(result3), 0, 0, nil)
		// return nil
		return nil
	})
}
