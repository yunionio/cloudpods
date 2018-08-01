package shell

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

func init() {
	/**
	 * 列出列表
	 */
	type ResourceDetailListOptions struct {
		BaseListOptions
		QUERYTYPE string `"help":"query type of the resource_detail"`
		ProjectId string `"help":"project id of the resource_detail"`
	}
	R(&ResourceDetailListOptions{}, "resourcedetail-list", "List all resource details", func(s *mcclient.ClientSession, args *ResourceDetailListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		if len(args.QUERYTYPE) > 0 {
			params.Add(jsonutils.NewString(args.QUERYTYPE), "query_type")
		}
		if len(args.ProjectId) > 0 {
			params.Add(jsonutils.NewString(args.ProjectId), "project_Iid")
		}

		result, err := modules.ResourceDetails.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.ResourceDetails.GetColumns(s))
		return nil
	})
}
