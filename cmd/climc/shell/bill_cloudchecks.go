package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	type BillCloudCheckListOptions struct {
		options.BaseListOptions
		ACCOUNTID string `help:"accountId of the bill_cloudcheck"`
		SUMMONTH  string `help:"sum_month of the bill_cloudcheck"`
		QUERYTYPE string `help:"query_type of the bill_cloudcheck"`
		QueryItem string `help:"query_item of the bill_cloudcheck"`
	}
	R(&BillCloudCheckListOptions{}, "billcloudcheck-list", "List all BillCloudChecks ", func(s *mcclient.ClientSession, args *BillCloudCheckListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err
			}
		}

		params.Add(jsonutils.NewString(args.ACCOUNTID), "account_id")
		params.Add(jsonutils.NewString(args.SUMMONTH), "sum_month")
		params.Add(jsonutils.NewString(args.QUERYTYPE), "query_type")
		if len(args.QueryItem) > 0 {
			params.Add(jsonutils.NewString(args.QueryItem), "query_item")
		}

		result, err := modules.BillCloudChecks.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.BillCloudChecks.GetColumns(s))
		return nil
	})
}
