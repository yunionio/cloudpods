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
	Type   string `help:"Repository type" choices:"internal|external"`
	URL    string `help:"Repository url"`
	Public bool   `help:"Make repostitory public"`
}

func (o RepoCreateOptions) Params() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.NAME), "name")
	params.Add(jsonutils.NewString(o.URL), "url")
	if o.Type != "" {
		params.Add(jsonutils.NewString(o.Type), "type")
	}
	if o.Public {
		params.Add(jsonutils.JSONTrue, "is_public")
	}
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
