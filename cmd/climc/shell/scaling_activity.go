package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ScalingActivityListOptions struct {
		options.BaseListOptions
	}
	R(&ScalingActivityListOptions{}, "scaling-activity-list", "List Scaling Activity",
		func(s *mcclient.ClientSession, args *ScalingActivityListOptions) error {
			params, err := options.ListStructToParams(args)
			if err != nil {
				return err
			}
			list, err := modules.ScalingActivity.List(s, params)
			if err != nil {
				return err
			}
			printList(list, modules.ScalingActivity.GetColumns(s))
			return nil
		},
	)
}
