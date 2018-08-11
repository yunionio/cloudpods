package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {

	/**
	 * 列出所有监控指标
	 */
	type ProjectAdminCandidateListOptions struct {
		BaseListOptions
	}
	R(&ProjectAdminCandidateListOptions{}, "projectadmincandidate-list", "List all Project Admin Candidates", func(s *mcclient.ClientSession, args *ProjectAdminCandidateListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)

		result, err := modules.ProjectAdminCandidate.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.ProjectAdminCandidate.GetColumns(s))
		return nil
	})

}
