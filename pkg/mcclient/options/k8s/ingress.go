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

type IngressCreateOptions struct {
	NamespaceWithClusterOptions
	NAME    string `help:"Name of ingress"`
	SERVICE string `help:"Service name"`
	PORT    int    `help:"Service port"`
	Path    string `help:"HTTP path"`
	Host    string `help:"Fuly qualified domain name of a network host" required:"true"`
}

func (o IngressCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := o.NamespaceWithClusterOptions.Params()
	params.Add(jsonutils.NewString(o.NAME), "name")
	path := jsonutils.NewDict()
	path.Add(jsonutils.NewString(o.SERVICE), "backend", "serviceName")
	path.Add(jsonutils.NewInt(int64(o.PORT)), "backend", "servicePort")
	if o.Path != "" {
		path.Add(jsonutils.NewString(o.Path), "path")
	}
	paths := jsonutils.NewArray()
	paths.Add(path)
	rule := jsonutils.NewDict()
	rule.Add(paths, "http", "paths")
	rule.Add(jsonutils.NewString(o.Host), "host")
	rules := jsonutils.NewArray()
	rules.Add(rule)
	params.Add(rules, "rules")
	return params, nil
}
