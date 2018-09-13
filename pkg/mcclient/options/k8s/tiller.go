package k8s

import (
	"yunion.io/x/jsonutils"
)

type TillerCreateOptions struct {
	ClusterBaseOptions
	KubeContext string `json:"kube_context"`
	Namespace   string `json:"namespace" default:"kube-system"`
	// Upgrade if Tiller is already installed
	Upgrade bool `json:"upgrade"`
	// Name of service account
	ServiceAccount string `json:"service_account" default:"tiller"`
	// Use the canary Tiller image
	Canary bool `json:"canary_image"`

	// Override Tiller image
	Image string `json:"tiller_image" default:"yunion/tiller:v2.9.0"`
	// Limit the maximum number of revisions saved per release. Use 0 for no limit.
	MaxHistory int `json:"history_max"`
}

func (o TillerCreateOptions) Params() *jsonutils.JSONDict {
	params := o.ClusterBaseOptions.Params()
	if len(o.KubeContext) > 0 {
		params.Add(jsonutils.NewString(o.KubeContext), "kube_context")
	}
	params.Add(jsonutils.NewString(o.Namespace), "namespace")
	params.Add(jsonutils.NewString(o.ServiceAccount), "service_account")
	if o.Canary {
		params.Add(jsonutils.JSONTrue, "canary_image")
	}
	if o.Upgrade {
		params.Add(jsonutils.JSONTrue, "upgrade")
	}
	if len(o.Image) > 0 {
		params.Add(jsonutils.NewString(o.Image), "tiller_image")
	}
	if o.MaxHistory > 0 {
		params.Add(jsonutils.NewInt(int64(o.MaxHistory)), "history_max")
	}
	return params
}
