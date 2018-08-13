package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {

	/**
	 * 列出所有监控指标
	 */
	type ProjectAdminListOptions struct {
		BaseListOptions
	}
	R(&ProjectAdminListOptions{}, "projectadmin-list", "List all Project Admins", func(s *mcclient.ClientSession, args *ProjectAdminListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)

		result, err := modules.ProjectAdmin.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.ProjectAdmin.GetColumns(s))
		return nil
	})

}
