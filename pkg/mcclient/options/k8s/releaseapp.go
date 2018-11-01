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
	ChartName string `help:"Meter helm chart name, e.g infra/meter" default:"infra/meter"`
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
