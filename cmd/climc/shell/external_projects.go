package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ExternalProjectListOptions struct {
		options.BaseListOptions
	}
	R(&ExternalProjectListOptions{}, "external-project-list", "List public cloud projects", func(s *mcclient.ClientSession, opts *ExternalProjectListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.ExternalProjects.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Disks.GetColumns(s))
		return nil
	})

	type ExternalProjectUpdateOptions struct {
		ID      string `help:"ExternalProject ID or Name"`
		PROJECT string `help:"Local project ID or Name"`
	}

	R(&ExternalProjectUpdateOptions{}, "external-project-update", "Update external project point to local project", func(s *mcclient.ClientSession, args *ExternalProjectUpdateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.PROJECT), "project")
		result, err := modules.ExternalProjects.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
