package shell

import (
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type AccountListOptions struct {
	}
	shellutils.R(&AccountListOptions{}, "account-list", "List sub account", func(cli *azure.SRegion, args *AccountListOptions) error {
		if accounts, err := cli.GetClient().GetSubAccounts(); err != nil {
			return err
		} else {
			printObject(accounts)
			return nil
		}
	})
}
