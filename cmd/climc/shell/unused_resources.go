package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	type UnusedResourceListOptions struct {
		options.BaseListOptions
		ProjectId string `help:"project id of the unused resources"`
		StartDate string `help:"start_date of the unused resources"`
		EndDate   string `help:"end_date of the unused resources"`
		ResType   string `help:"res_type of the unused resources"`
		Platform  string `help:"platform of the unused resources"`
	}
	R(&UnusedResourceListOptions{}, "unusedresources-list", "List all unused resources", func(s *mcclient.ClientSession, args *UnusedResourceListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if len(args.ProjectId) > 0 {
			params.Add(jsonutils.NewString(args.ProjectId), "project_id")
		}
		if len(args.StartDate) > 0 {
			params.Add(jsonutils.NewString(args.StartDate), "start_date")
		}
		if len(args.EndDate) > 0 {
			params.Add(jsonutils.NewString(args.EndDate), "end_date")
		}
		if len(args.ResType) > 0 {
			params.Add(jsonutils.NewString(args.ResType), "res_type")
		}
		if len(args.Platform) > 0 {
			params.Add(jsonutils.NewString(args.Platform), "platform")
		}

		result, err := modules.UnusedResources.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.UnusedResources.GetColumns(s))
		return nil
	})
}
