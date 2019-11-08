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
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
)

type ServiceSpecOptions struct {
	Port       []string `help:"Port for the service that is created, format is <protocol>:<service_port>:<container_port> e.g. tcp:80:3000"`
	IsExternal bool     `help:"Created service is external loadbalance"`
	LbNetwork  string   `help:"LoadBalancer service network id"`
}

func (o ServiceSpecOptions) Params() (*jsonutils.JSONDict, error) {
	if len(o.Port) == 0 {
		return nil, nil
	}
	params := jsonutils.NewDict()
	portMappings, err := parsePortMappings(o.Port)
	if err != nil {
		return nil, err
	}
	if o.IsExternal {
		params.Add(jsonutils.JSONTrue, "isExternal")
		if o.LbNetwork != "" {
			params.Add(jsonutils.NewString(o.LbNetwork), "loadBalancerNetwork")
		}
	}
	params.Add(portMappings, "portMappings")
	return params, nil
}

func (o ServiceSpecOptions) Attach(data *jsonutils.JSONDict) error {
	return attachData(o, data, "service")
}

type ServiceCreateOptions struct {
	NamespaceWithClusterOptions
	ServiceSpecOptions
	NAME     string   `help:"Name of deployment"`
	Selector []string `help:"Selectors are backends pods labels, e.g. 'run=app'"`
}

func (o ServiceCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params := o.NamespaceWithClusterOptions.Params()
	svcSpec, err := o.ServiceSpecOptions.Params()
	if err != nil {
		return nil, err
	}
	selector := jsonutils.NewDict()
	for _, s := range o.Selector {
		parts := strings.Split(s, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("Invalid selctor string: %s", s)
		}
		selector.Add(jsonutils.NewString(parts[1]), parts[0])
	}
	params.Update(svcSpec)
	params.Add(selector, "selector")
	params.Add(jsonutils.NewString(o.NAME), "name")
	return params, nil
}

type ServiceListOptions struct {
	NamespaceResourceListOptions
	Type string `help:"Service type" choices:"ClusterIP|LoadBalancer|NodePort|ExternalName"`
}

func (o ServiceListOptions) Params() *jsonutils.JSONDict {
	params := o.NamespaceResourceListOptions.Params()
	if o.Type != "" {
		params.Add(jsonutils.NewString(o.Type), "type")
	}
	return params
}
