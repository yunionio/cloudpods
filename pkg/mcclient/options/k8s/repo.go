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
	Type string `help:"Helm repostitory type" json:"type" choices:"internal|external"`
}

func (o *RepoListOptions) Params() (jsonutils.JSONObject, error) {
	param, err := o.BaseListOptions.Params()
	if err != nil {
		return nil, err
	}
	params := param.(*jsonutils.JSONDict)
	if o.Type != "" {
		params.Add(jsonutils.NewString(o.Type), "type")
	}
	return params, nil
}

type RepoGetOptions struct {
	NAME string `help:"ID or name of the repo"`
}

func (o *RepoGetOptions) GetId() string {
	return o.NAME
}

func (o *RepoGetOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type RepoCreateOptions struct {
	RepoGetOptions
	Type     string `help:"Repository type" choices:"internal|external"`
	URL      string `help:"Repository url"`
	Public   bool   `help:"Make repostitory public"`
	Backend  string `help:"Repository backend" choices:"common|nexus"`
	Username string `help:"Repository auth username"`
	Password string `help:"Repository auth password"`
}

func (o RepoCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.NAME), "name")
	params.Add(jsonutils.NewString(o.URL), "url")
	if o.Type != "" {
		params.Add(jsonutils.NewString(o.Type), "type")
	}
	if o.Public {
		params.Add(jsonutils.JSONTrue, "is_public")
	}
	if o.Backend != "" {
		params.Add(jsonutils.NewString(o.Backend), "backend")
	}
	if o.Username != "" {
		params.Add(jsonutils.NewString(o.Username), "username")
	}
	if o.Password != "" {
		params.Add(jsonutils.NewString(o.Password), "password")
	}
	return params, nil
}

type RepoUpdateOptions struct {
	RepoGetOptions
	Name     string `help:"Repository name to change"`
	Url      string `help:"Repository url to change"`
	Username string `help:"Repository auth username"`
	Password string `help:"Repository auth password"`
}

func (o RepoUpdateOptions) GetId() string {
	return o.NAME
}

func (o RepoUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if o.Name != "" {
		params.Add(jsonutils.NewString(o.Name), "name")
	}
	if o.Url != "" {
		params.Add(jsonutils.NewString(o.Url), "url")
	}
	if o.Username != "" {
		params.Add(jsonutils.NewString(o.Username), "username")
	}
	if o.Password != "" {
		params.Add(jsonutils.NewString(o.Password), "password")
	}
	return params, nil
}

type RepoPublicOptions struct {
	ID            string   `help:"ID or name of repo" json:"-"`
	Scope         string   `help:"sharing scope" choices:"system|domain"`
	SharedDomains []string `help:"share to domains"`
}

func (o *RepoPublicOptions) GetId() string {
	return o.ID
}

func (o *RepoPublicOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}
