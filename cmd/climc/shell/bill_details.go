package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	type BillDetailListOptions struct {
		options.BaseListOptions
		STARTDATE string `help:"start date of the bill_detail"`
		ENDDATE   string `help:"end date of the bill_detail"`
		ProjectId string `help:"project id of the bill_detail"`
	}
	R(&BillDetailListOptions{}, "billdetail-list", "List all bill details", func(s *mcclient.ClientSession, args *BillDetailListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if len(args.STARTDATE) > 0 {
			params.Add(jsonutils.NewString(args.STARTDATE), "start_date")
		}
		if len(args.ENDDATE) > 0 {
			params.Add(jsonutils.NewString(args.ENDDATE), "end_date")
		}

		if len(args.ProjectId) > 0 {
			params.Add(jsonutils.NewString(args.ProjectId), "project_id")
		}

		result, err := modules.BillDetails.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.BillDetails.GetColumns(s))
		return nil
	})
}
