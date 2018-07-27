package shell

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/mcclient/modules"
)

func init() {
	/**
	 * 列出列表
	 */
	type ResourceFeeListOptions struct {
		BaseListOptions
		STATTYPE  string `"help":"stat type of the resource_fee"`
		STATMONTH string `"help":"stat month of the resource_fee"`
		ProjectId string `"help":"project id of the resource_fee"`
		StartDay  string `"help":"start day of the resource_fee"`
		EndDay    string `"help":"end day of the resource_fee"`
	}
	R(&ResourceFeeListOptions{}, "resourcefee-list", "List all resource fees", func(s *mcclient.ClientSession, args *ResourceFeeListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		if len(args.STATTYPE) > 0 {
			params.Add(jsonutils.NewString(args.STATTYPE), "stat_type")
		}
		if len(args.STATMONTH) > 0 {
			params.Add(jsonutils.NewString(args.STATMONTH), "stat_month")
		}

		if len(args.ProjectId) > 0 {
			params.Add(jsonutils.NewString(args.ProjectId), "project_id")
		}

		if len(args.StartDay) > 0 {
			params.Add(jsonutils.NewString(args.StartDay), "start_day")
		}

		if len(args.EndDay) > 0 {
			params.Add(jsonutils.NewString(args.EndDay), "end_day")
		}

		result, err := modules.ResourceFees.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.ResourceFees.GetColumns(s))
		return nil
	})
}
