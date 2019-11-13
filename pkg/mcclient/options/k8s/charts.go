// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8s

import (
	"yunion.io/x/jsonutils"
)

type ChartListOptions struct {
	BaseListOptions
	Repo       string `help:"Repository name"`
	RepoUrl    string `help:"Repository url"`
	AllVersion bool   `help:"Get Chart all history versions"`
	Keyword    string `help:"Chart keyword"`
	Version    string `help:"Chart semver version filter"`
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
	if len(o.Version) != 0 {
		params.Add(jsonutils.NewString(o.Version), "version")
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
