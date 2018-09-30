package options

import (
	"yunion.io/x/jsonutils"
)

type WebConsoleOptions struct {
	WebconsoleUrl string `help:"Frontend webconsole url" short-token:"w" default:"$WEBCONSOLE_URL"`
}

type PodBaseOptions struct {
	WebConsoleOptions
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

type WebConsoleBaremetalOptions struct {
	WebConsoleOptions
	ID string `help:"Baremetal host id or name"`
}

func (opt *WebConsoleBaremetalOptions) Params() (*jsonutils.JSONDict, error) {
	return StructToParams(opt)
}

type WebConsoleSshOptions struct {
	WebConsoleOptions
	IP string `help:"IP to connect"`
}
