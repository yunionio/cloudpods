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

package options

import (
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SchedtagModelListOptions struct {
	BaseListOptions
	Schedtag string `help:"ID or Name of schedtag"`
}

func (o SchedtagModelListOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := o.BaseListOptions.Params()
	if err != nil {
		return nil, err
	}
	return params, nil
}

type SchedtagModelPairOptions struct {
	SCHEDTAG string `help:"Scheduler tag"`
	OBJECT   string `help:"Object id"`
}

type SchedtagSetOptions struct {
	ID       string   `help:"Id or name of resource"`
	Schedtag []string `help:"Ids of schedtag"`
}

func (o SchedtagSetOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	for idx, tag := range o.Schedtag {
		params.Add(jsonutils.NewString(tag), fmt.Sprintf("schedtag.%d", idx))
	}
	return params, nil
}

type SchedtagListOptions struct {
	BaseListOptions
	Type            string `help:"Filter by resource type"`
	CloudproviderId string `help:"Filter by cloudprovider id"`
}

func (o SchedtagListOptions) Params() (jsonutils.JSONObject, error) {
	params, err := o.BaseListOptions.Params()
	if err != nil {
		return nil, err
	}

	if len(o.Type) > 0 {
		params.Add(jsonutils.NewString(o.Type), "resource_type")
	}
	if len(o.CloudproviderId) > 0 {
		params.Add(jsonutils.NewString(o.CloudproviderId), "cloudprovider_id")
	}

	return params, nil
}

type SchedtagShowOptions struct {
	ID string `help:"ID or Name of the scheduler tag to show"`
}

func (o SchedtagShowOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

func (o SchedtagShowOptions) GetId() string {
	return o.ID
}

type SchedtagCreateOptions struct {
	NAME     string `help:"Name of new schedtag"`
	Strategy string `help:"Policy" choices:"require|exclude|prefer|avoid"`
	Desc     string `help:"Description"`
	Scope    string `help:"Resource scope" choices:"system|domain|project"`
	Type     string `help:"Resource type" choices:"hosts|storages|networks|cloudproviders|cloudregions|zones"`
}

func (o SchedtagCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.NAME), "name")
	if len(o.Strategy) > 0 {
		params.Add(jsonutils.NewString(o.Strategy), "default_strategy")
	}
	if len(o.Desc) > 0 {
		params.Add(jsonutils.NewString(o.Desc), "description")
	}
	if len(o.Type) > 0 {
		params.Add(jsonutils.NewString(o.Type), "resource_type")
	}
	if len(o.Scope) > 0 {
		params.Add(jsonutils.NewString(o.Scope), "scope")
	}

	return params, nil
}

type SchedtagUpdateOptions struct {
	ID            string `help:"ID or Name of schetag"`
	Name          string `help:"New name of schedtag"`
	Strategy      string `help:"Policy" choices:"require|exclude|prefer|avoid"`
	Desc          string `help:"Description"`
	ClearStrategy bool   `help:"Clear default schedule policy"`
}

func (o SchedtagUpdateOptions) GetId() string {
	return o.ID
}

func (o SchedtagUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if len(o.Name) > 0 {
		params.Add(jsonutils.NewString(o.Name), "name")
	}
	if len(o.Strategy) > 0 {
		params.Add(jsonutils.NewString(o.Strategy), "default_strategy")
	}
	if len(o.Desc) > 0 {
		params.Add(jsonutils.NewString(o.Desc), "description")
	}
	if o.ClearStrategy {
		params.Add(jsonutils.NewString(""), "default_strategy")
	}
	if params.Size() == 0 {
		return nil, fmt.Errorf("No valid data to update")
	}

	return params, nil
}

type SchedtagSetScopeOptions struct {
	ID      []string `help:"ID or Name of schetag"`
	Project string   `help:"ID or Name of project"`
	Domain  string   `help:"ID or Name of domain"`
	System  bool     `help:"Set to system scope"`
}

func (o SchedtagSetScopeOptions) GetIds() []string {
	return o.ID
}

func (o SchedtagSetScopeOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	domainId := o.Domain
	projectId := o.Project
	if o.System {
		domainId = ""
		projectId = ""
	}
	params.Add(jsonutils.NewString(domainId), "domain")
	params.Add(jsonutils.NewString(projectId), "project")
	return params, nil
}

type SchedtagSetResource struct {
	ID        string   `help:"ID or Name of schetag"`
	Resource  []string `help:"Resource id or name"`
	UnbindAll bool     `help:"Unbind all attached resources"`
}

func (o SchedtagSetResource) GetId() string {
	return o.ID
}

func (o SchedtagSetResource) Params() (jsonutils.JSONObject, error) {
	if o.UnbindAll && len(o.Resource) != 0 {
		return nil, fmt.Errorf("Can not use --unbind-all and --resource at same time")
	}

	input := new(api.SchedtagSetResourceInput)

	if !o.UnbindAll {
		input.ResourceIds = o.Resource
	}

	return jsonutils.Marshal(input), nil
}
