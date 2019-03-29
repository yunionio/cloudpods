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

type AppBaseCreateOptions struct {
	NamespaceWithClusterOptions
	ReleaseCreateUpdateOptions
	Name string `help:"Release name, If unspecified, it will autogenerate one for you"`
}

func (o AppBaseCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := o.ReleaseCreateUpdateOptions.Params()
	if err != nil {
		return nil, err
	}
	params.Update(o.NamespaceWithClusterOptions.Params())
	if o.Name != "" {
		params.Add(jsonutils.NewString(o.Name), "release_name")
	}
	return params, nil
}

type AppCreateOptions struct {
	AppBaseCreateOptions
	ChartName string `help:"Helm release app chart name, e.g yunion/meter, yunion/monitor-stack"`
}

func (o AppCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := o.AppBaseCreateOptions.Params()
	if err != nil {
		return nil, err
	}
	if o.ChartName != "" {
		params.Add(jsonutils.NewString(o.ChartName), "chart_name")
	}
	return params, nil
}
