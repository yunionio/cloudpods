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

type DeploymentCreateOptions struct {
	NamespaceWithClusterOptions

	K8sLabelOptions
	K8sPodTemplateOptions
	ServiceSpecOptions

	NAME     string `help:"Name of deployment"`
	Replicas int64  `help:"Number of replicas for pods in this deployment"`
}

func (o DeploymentCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params := o.NamespaceWithClusterOptions.Params()
	o.K8sPodTemplateOptions.setContainerName(o.NAME)
	if err := o.K8sPodTemplateOptions.Attach(params); err != nil {
		return nil, err
	}
	if err := o.K8sLabelOptions.Attach(params); err != nil {
		return nil, err
	}
	if err := o.ServiceSpecOptions.Attach(params); err != nil {
		return nil, err
	}
	params.Add(jsonutils.NewString(o.NAME), "name")
	if o.Replicas > 1 {
		params.Add(jsonutils.NewInt(o.Replicas), "replicas")
	}
	return params, nil
}

type DeploymentUpdateOptions struct {
	NamespaceWithClusterOptions
	NAME  string   `help:"Name of deployment"`
	Image []string `help:"Image of container to set, e.g. 'default=nginx:latest'"`
}

func (o DeploymentUpdateOptions) Params() (*jsonutils.JSONDict, error) {
	params := o.NamespaceWithClusterOptions.Params()
	containers := jsonutils.NewArray()
	for _, img := range o.Image {
		parts, err := parseImage(img)
		if err != nil {
			return nil, err
		}
		containers.Add(parts)
	}
	params.Add(containers, "containers")

	return params, nil
}
