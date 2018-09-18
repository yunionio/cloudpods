package k8s

import (
	"yunion.io/x/jsonutils"
)

type ClusterBaseOptions struct {
	Cluster string `default:"$K8S_CLUSTER|default" help:"Kubernetes cluster name"`
}

func (o ClusterBaseOptions) Params() *jsonutils.JSONDict {
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(o.Cluster), "cluster")
	return ret
}

type BaseListOptions struct {
	Limit  int    `default:"20" help:"Page limit"`
	Offset int    `default:"0" help:"Page offset"`
	Name   string `help:"Search by name"`
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
