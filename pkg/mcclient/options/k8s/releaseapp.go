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
	CHARTNAME string `help:"Helm release app chart name, e.g yunion/meter"`
}

func (o AppCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := o.AppBaseCreateOptions.Params()
	if err != nil {
		return nil, err
	}
	params.Add(jsonutils.NewString(o.CHARTNAME), "chart_name")
	return params, nil
}
