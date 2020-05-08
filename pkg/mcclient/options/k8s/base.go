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

type ClusterBaseOptions struct {
	Cluster string `default:"$K8S_CLUSTER" help:"Kubernetes cluster name"`
}

func (o ClusterBaseOptions) Params() *jsonutils.JSONDict {
	ret := jsonutils.NewDict()
	if o.Cluster != "" {
		ret.Add(jsonutils.NewString(o.Cluster), "cluster")
	}
	return ret
}

type ClusterResourceBaseOptions struct {
	ClusterBaseOptions
	NAME string `help:"Name of resource"`
}

type ClusterResourceCreateOptions struct {
	ClusterResourceBaseOptions
}

func (o ClusterResourceCreateOptions) Params() *jsonutils.JSONDict {
	params := o.ClusterBaseOptions.Params()
	params.Add(jsonutils.NewString(o.NAME), "name")
	return params
}

type BaseListOptions struct {
	Limit  int    `default:"20" help:"Page limit"`
	Offset int    `default:"0" help:"Page offset"`
	Name   string `help:"Search by name"`
	System *bool  `help:"Show system resource"`
}

func (o BaseListOptions) Params() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	if o.Limit > 0 {
		params.Add(jsonutils.NewInt(int64(o.Limit)), "limit")
	}
	if o.Offset > 0 {
		params.Add(jsonutils.NewInt(int64(o.Offset)), "offset")
	}
	if o.Name != "" {
		params.Add(jsonutils.NewString(o.Name), "name")
	}
	if o.System != nil {
		params.Add(jsonutils.NewBool(*o.System), "system")
	}
	return params
}

type ResourceListOptions struct {
	ClusterBaseOptions
	BaseListOptions
}

func (o ResourceListOptions) Params() *jsonutils.JSONDict {
	params := o.BaseListOptions.Params()
	params.Update(o.ClusterBaseOptions.Params())
	return params
}

type ResourceGetOptions struct {
	ClusterBaseOptions
	NAME string `help:"Name ident of the resource"`
}

func (o ResourceGetOptions) Params() *jsonutils.JSONDict {
	params := o.ClusterBaseOptions.Params()
	return params
}

type ResourceDeleteOptions struct {
	ClusterBaseOptions
	NAME []string `help:"Name ident of the resources"`
}

func (o ResourceDeleteOptions) Params() *jsonutils.JSONDict {
	params := o.ClusterBaseOptions.Params()
	return params
}

type NamespaceResourceListOptions struct {
	ResourceListOptions
	Namespace    string `help:"Namespace of this resource"`
	AllNamespace bool   `help:"Show resource in all namespace"`
}

func (o NamespaceResourceListOptions) Params() *jsonutils.JSONDict {
	params := o.ResourceListOptions.Params()
	if o.AllNamespace {
		params.Add(jsonutils.JSONTrue, "all_namespace")
		return params
	}
	if o.Namespace != "" {
		params.Add(jsonutils.NewString(o.Namespace), "namespace")
	}
	return params
}

type NamespaceOptions struct {
	Namespace string `help:"Namespace of this resource"`
}

func (o NamespaceOptions) Params() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	if o.Namespace != "" {
		params.Add(jsonutils.NewString(o.Namespace), "namespace")
	}
	return params
}

type NamespaceResourceGetOptions struct {
	ResourceGetOptions
	NamespaceOptions
}

func (o NamespaceResourceGetOptions) Params() *jsonutils.JSONDict {
	params := o.ResourceGetOptions.Params()
	params.Update(o.NamespaceOptions.Params())
	return params
}

type NamespaceResourceDeleteOptions struct {
	ResourceDeleteOptions
	NamespaceOptions
}

func (o NamespaceResourceDeleteOptions) Params() *jsonutils.JSONDict {
	params := o.ResourceDeleteOptions.Params()
	params.Update(o.NamespaceOptions.Params())
	return params
}

type NamespaceWithClusterOptions struct {
	NamespaceOptions
	ClusterBaseOptions
}

func (o NamespaceWithClusterOptions) Params() *jsonutils.JSONDict {
	params := o.ClusterBaseOptions.Params()
	params.Update(o.NamespaceOptions.Params())
	return params
}
