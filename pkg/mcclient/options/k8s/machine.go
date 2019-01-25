package k8s

import (
	"yunion.io/x/jsonutils"
)

type MachineCreateOptions struct {
	CLUSTER  string `help:"Cluster id"`
	ROLE     string `help:"Machine role" choices:"node|controlplane"`
	Type     string `help:"Resource type" choices:"vm|baremetal" json:"resource_type"`
	Instance string `help:"VM or host instance id" json:"resource_id"`
	Name     string `help:"Name of node"`
}

func (o MachineCreateOptions) Params() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	if o.Name != "" {
		params.Add(jsonutils.NewString(o.Name), "name")
	}
	params.Add(jsonutils.NewString(o.CLUSTER), "cluster")
	if o.ROLE != "" {
		params.Add(jsonutils.NewString(o.ROLE), "role")
	}
	if o.Instance != "" {
		params.Add(jsonutils.NewString(o.Instance), "resource_id")
	}
	if o.Type != "" {
		params.Add(jsonutils.NewString(o.Type), "resource_type")
	}
	return params
}
