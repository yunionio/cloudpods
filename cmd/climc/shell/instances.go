package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type InstanceListOptions struct {
		options.BaseListOptions
	}
	R(&InstanceListOptions{}, "instance-list", "List instances", func(s *mcclient.ClientSession, suboptions *InstanceListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = suboptions.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		result, err := modules.Instances.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Instances.GetColumns(s))
		return nil
	})
}
