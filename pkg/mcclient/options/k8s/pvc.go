package k8s

import (
	"yunion.io/x/jsonutils"
)

type PVCListOptions struct {
	NamespaceResourceListOptions
	Unused bool `help:"Filter unused pvc"`
}

func (o PVCListOptions) Params() *jsonutils.JSONDict {
	params := o.NamespaceResourceListOptions.Params()
	if o.Unused {
		params.Add(jsonutils.JSONTrue, "unused")
	}
	return params
}

type PVCCreateOptions struct {
	NamespaceWithClusterOptions
	NAME         string `help:"Name of PVC"`
	SIZE         string `help:"Storage size, e.g. 10Gi"`
	StorageClass string `help:"PVC StorageClassName"`
}

func (o PVCCreateOptions) Params() *jsonutils.JSONDict {
	params := o.NamespaceWithClusterOptions.Params()
	params.Add(jsonutils.NewString(o.NAME), "name")
	params.Add(jsonutils.NewString(o.SIZE), "size")
	if o.StorageClass != "" {
		params.Add(jsonutils.NewString(o.StorageClass), "storageClass")
	}
	return params
}
