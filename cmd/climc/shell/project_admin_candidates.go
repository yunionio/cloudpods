package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 列出所有监控指标
	 */
	type ProjectAdminCandidateListOptions struct {
		options.BaseListOptions
	}
	R(&ProjectAdminCandidateListOptions{}, "projectadmincandidate-list", "List all Project Admin Candidates", func(s *mcclient.ClientSession, args *ProjectAdminCandidateListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}

		result, err := modules.ProjectAdminCandidate.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.ProjectAdminCandidate.GetColumns(s))
		return nil
	})

}
