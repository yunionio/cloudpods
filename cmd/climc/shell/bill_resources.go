package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	type BillResourceListOptions struct {
		options.BaseListOptions
		STARTDATE string `help:"start date of the bill_resource"`
		ENDDATE   string `help:"end date of the bill_resource"`
		ProjectId string `help:"project id of the bill_resource"`
	}
	R(&BillResourceListOptions{}, "billresource-list", "List all bill resources", func(s *mcclient.ClientSession, args *BillResourceListOptions) error {
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

		result, err := modules.BillResources.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.BillResources.GetColumns(s))
		return nil
	})
}
