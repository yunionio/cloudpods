package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type MetadataListOptions struct {
		Resources []string `help:"list of resource e.g server、disk"`
		IsSys     bool     `help:"list sys metadata"`
	}

	R(&MetadataListOptions{}, "metadata-list", "List metadatas", func(s *mcclient.ClientSession, opts *MetadataListOptions) error {
		params := jsonutils.NewDict()
		resources := jsonutils.NewArray()
		for _, resource := range opts.Resources {
			resources.Add(jsonutils.NewString(resource))
		}
		if resources.Length() > 0 {
			params.Add(resources, "resources")
		}
		if opts.IsSys {
			params.Add(jsonutils.JSONTrue, "is_sys")
		}
		result, err := modules.Metadatas.List(s, params)
		if err != nil {
			return err
		}
		printList(result, []string{})
		return nil
	})
}
