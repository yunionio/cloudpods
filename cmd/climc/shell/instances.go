package shell

import (
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

func init() {
	type InstanceListOptions struct {
		BaseListOptions
	}
	R(&InstanceListOptions{}, "instance-list", "List instances", func(s *mcclient.ClientSession, suboptions *InstanceListOptions) error {
		params := FetchPagingParams(suboptions.BaseListOptions)
		result, err := modules.Instances.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Instances.GetColumns(s))
		return nil
	})
}
