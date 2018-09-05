package options

import (
	"yunion.io/x/jsonutils"
)

type WebConsoleFrontendOptions struct {
	WebUrl string `help:"Frontend terminal HTML web url" short-token:"w"`
}

type PodBaseOptions struct {
	WebConsoleFrontendOptions
	NAME      string `help:"Name of k8s pod to connect"`
	Namespace string `help:"Namespace of this pod"`
	Container string `help:"Container in this pod"`
	Cluster   string `default:"$K8S_CLUSTER|default" help:"Kubernetes cluster name"`
}

func (opt *PodBaseOptions) Params() (*jsonutils.JSONDict, error) {
	return StructToParams(opt)
}

type PodShellOptions struct {
	PodBaseOptions
}

type PodLogOptoins struct {
	PodBaseOptions
}
