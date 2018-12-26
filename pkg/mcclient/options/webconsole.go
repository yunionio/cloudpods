package options

import (
	"time"

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
	Since string `help:"Only return logs newer than a relative duration like 5s, 2m or 3h"`
}

func (opt *PodLogOptoins) Params() (*jsonutils.JSONDict, error) {
	params, err := opt.PodBaseOptions.Params()
	if err != nil {
		return nil, err
	}
	if opt.Since != "" {
		_, err = time.ParseDuration(opt.Since)
		if err != nil {
			return nil, err
		}
		params.Add(jsonutils.NewString(opt.Since), "since")
	}
	return params, nil
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

type WebConsoleServerOptions struct {
	WebConsoleOptions
	ID string `help:"Server id or name"`
}
