package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	type BillConditionListOptions struct {
		options.BaseListOptions
		QUERYTYPE string `help:"query type of the bill_condition"`
		ParentId  string `help:"parent id of the bill_condition"`
	}
	R(&BillConditionListOptions{}, "billcondition-list", "List all bill conditions", func(s *mcclient.ClientSession, args *BillConditionListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if len(args.QUERYTYPE) > 0 {
			params.Add(jsonutils.NewString(args.QUERYTYPE), "query_type")
		}
		if len(args.ParentId) > 0 {
			params.Add(jsonutils.NewString(args.ParentId), "parent_id")
		}

		result, err := modules.BillConditions.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.BillConditions.GetColumns(s))
		return nil
	})
}
