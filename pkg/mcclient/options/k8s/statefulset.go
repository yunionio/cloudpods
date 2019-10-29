package k8s

import (
	"yunion.io/x/jsonutils"
)

type StatefulSetCreateOptions struct {
	NamespaceWithClusterOptions

	K8sLabelOptions
	K8sPodTemplateOptions
	ServiceSpecOptions

	NAME     string `help:"Name of deployment"`
	Replicas int64  `help:"Number of replicas for pods in this deployment"`

	K8sPVCTemplateOptions
}

func (o StatefulSetCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params := o.NamespaceWithClusterOptions.Params()
	pvcs, err := o.K8sPVCTemplateOptions.Parse()
	if err != nil {
		return nil, err
	}
	o.K8sPVCTemplateOptions.Attach(params, pvcs, &o.K8sPodTemplateOptions)

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
