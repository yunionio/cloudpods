package k8s

import (
	"yunion.io/x/jsonutils"
)

type ChartListOptions struct {
	BaseListOptions
	Repo       string `help:"Repository name"`
	RepoUrl    string `help:"Repository url"`
	AllVersion bool   `json:"Get Chart all history versions"`
	Keyword    string `json:"Chart keyword"`
}

func (o ChartListOptions) Params() *jsonutils.JSONDict {
	params := o.BaseListOptions.Params()
	if len(o.Name) != 0 {
		params.Add(jsonutils.NewString(o.Name), "name")
	}
	if len(o.Repo) != 0 {
		params.Add(jsonutils.NewString(o.Repo), "repo")
	}
	if len(o.RepoUrl) != 0 {
		params.Add(jsonutils.NewString(o.RepoUrl), "repo_url")
	}
	if o.AllVersion {
		params.Add(jsonutils.JSONTrue, "all_version")
	}
	if len(o.Keyword) != 0 {
		params.Add(jsonutils.NewString(o.Keyword), "keyword")
	}
	return params
}

type ChartGetOptions struct {
	REPO    string `help:"Repo of the chart"`
	NAME    string `help:"Chart name"`
	Version string `help:"Chart version"`
}

func (o ChartGetOptions) Params() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.REPO), "repo")
	if o.Version != "" {
		params.Add(jsonutils.NewString(o.Version), "version")
	}
	return params
}
