package k8s

import (
	json "yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initChart() {
	cmdN := func(suffix string) string {
		return resourceCmdN("chart", suffix)
	}

	type listOpt struct {
		baseListOptions
		Name       string `help:"Chart name"`
		Repo       string `help:"Repository name"`
		RepoUrl    string `help:"Repository url"`
		AllVersion bool   `json:"Get Chart all history versions"`
		Keyword    string `json:"Chart keyword"`
	}
	R(&listOpt{}, cmdN("list"), "List k8s helm global charts", func(s *mcclient.ClientSession, args *listOpt) error {
		params := fetchPagingParams(args.baseListOptions)
		if len(args.Name) != 0 {
			params.Add(json.NewString(args.Name), "name")
		}
		if len(args.Repo) != 0 {
			params.Add(json.NewString(args.Repo), "repo")
		}
		if len(args.RepoUrl) != 0 {
			params.Add(json.NewString(args.RepoUrl), "repo_url")
		}
		if args.AllVersion {
			params.Add(json.JSONTrue, "all_version")
		}
		if len(args.Keyword) != 0 {
			params.Add(json.NewString(args.Keyword), "keyword")
		}
		charts, err := k8s.Charts.List(s, params)
		if err != nil {
			return err
		}

		PrintListResultTable(charts, k8s.Charts, s)
		return nil
	})
}
