package k8s

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type NodeListOptions struct {
	options.BaseListOptions
	Cluster string `help:"Filter by cluster"`
}

func (o NodeListOptions) Params() (*jsonutils.JSONDict, error) {
	return options.ListStructToParams(&o)
}

type NodeCreateOptions struct {
	CLUSTER          string   `help:"Cluster id"`
	Etcd             bool     `help:"Etcd role"`
	Controlplane     bool     `help:"Controlplane role"`
	Worker           bool     `help:"Worker role"`
	AllRole          bool     `help:"All roles"`
	HostnameOverride string   `help:"Worker node overrided hostname"`
	Host             string   `help:"Yunion host server name or id"`
	Name             string   `help:"Name of node"`
	RegistryMirror   []string `help:"Docker registry mirrors, e.g. 'https://registry.docker-cn.com'"`
	InsecureRegistry []string `help:"Docker insecure registry"`
}

func (o NodeCreateOptions) Params() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	if o.Name != "" {
		params.Add(jsonutils.NewString(o.Name), "name")
	}
	params.Add(jsonutils.NewString(o.CLUSTER), "cluster")
	dockerConf := dockerConfig{}
	for _, rm := range o.RegistryMirror {
		dockerConf.RegistryMirrors = append(dockerConf.RegistryMirrors, rm)
	}
	for _, im := range o.InsecureRegistry {
		dockerConf.InsecureRegistries = append(dockerConf.InsecureRegistries, im)
	}
	confObj := jsonutils.Marshal(dockerConf)
	params.Add(confObj, "dockerd_config")

	roles := jsonutils.NewArray()
	if o.AllRole {
		roles.Add(jsonutils.NewString("etcd"), jsonutils.NewString("controlplane"), jsonutils.NewString("worker"))
	} else {
		if o.Etcd {
			roles.Add(jsonutils.NewString("etcd"))
		}
		if o.Controlplane {
			roles.Add(jsonutils.NewString("controlplane"))
		}
		if o.Worker {
			roles.Add(jsonutils.NewString("worker"))
		}
	}
	params.Add(roles, "roles")
	if o.HostnameOverride != "" {
		params.Add(jsonutils.NewString(o.HostnameOverride), "hostname_override")
	}
	if o.Host != "" {
		params.Add(jsonutils.NewString(o.Host), "host")
	}
	return params
}

type NodeConfigDockerRegistryOptions struct {
	IdentsOptions
	RegistryMirror   []string `help:"Docker registry mirrors, e.g. 'https://registry.docker-cn.com'"`
	InsecureRegistry []string `help:"Docker insecure registry"`
}

func (o NodeConfigDockerRegistryOptions) Params() jsonutils.JSONObject {
	dockerConf := dockerConfig{}
	for _, rm := range o.RegistryMirror {
		dockerConf.RegistryMirrors = append(dockerConf.RegistryMirrors, rm)
	}
	for _, im := range o.InsecureRegistry {
		dockerConf.InsecureRegistries = append(dockerConf.InsecureRegistries, im)
	}
	return jsonutils.Marshal(dockerConf)
}
