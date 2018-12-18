package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	type BillBalanceListOptions struct {
		options.BaseListOptions
		ACCOUNTID string `help:"accountId of the BillBalance"`
	}
	R(&BillBalanceListOptions{}, "billbalance-list", "List all BillBalances ", func(s *mcclient.ClientSession, args *BillBalanceListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err
			}
		}

		params.Add(jsonutils.NewString(args.ACCOUNTID), "account_id")

		result, err := modules.BillBalances.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.BillBalances.GetColumns(s))
		return nil
	})

	type BillBalanceShowOptions struct {
		ACCOUNTID string `help:"accountId of the BillBalance to show"`
	}
	R(&BillBalanceShowOptions{}, "billbalance-show", "Show BillBalance details", func(s *mcclient.ClientSession, args *BillBalanceShowOptions) error {
		result, err := modules.BillBalances.Get(s, args.ACCOUNTID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
