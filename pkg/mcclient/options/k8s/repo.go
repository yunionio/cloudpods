package k8s

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type RepoListOptions struct {
	options.BaseListOptions
}

type RepoGetOptions struct {
	NAME string `help:"ID or name of the repo"`
}

type RepoCreateOptions struct {
	RepoGetOptions
	URL string `help:"Repository url"`
}

func (o RepoCreateOptions) Params() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.NAME), "name")
	params.Add(jsonutils.NewString(o.URL), "url")
	return params
}

type RepoUpdateOptions struct {
	RepoGetOptions
	Name string `help:"Repository name to change"`
	Url  string `help:"Repository url to change"`
}

func (o RepoUpdateOptions) Params() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	if o.Name != "" {
		params.Add(jsonutils.NewString(o.Name), "name")
	}
	if o.Url != "" {
		params.Add(jsonutils.NewString(o.Url), "url")
	}
	return params
}
