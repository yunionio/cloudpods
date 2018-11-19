package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	type UnderutilizedInstancesListOptions struct {
		options.BaseListOptions
	}
	R(&UnderutilizedInstancesListOptions{}, "underutilized-instances-list", "List underutilized instances", func(s *mcclient.ClientSession, args *UnderutilizedInstancesListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		result, err := modules.UnderutilizedInstances.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.UnderutilizedInstances.GetColumns(s))
		return nil
	})

}
