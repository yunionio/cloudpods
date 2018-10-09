package k8s

import (
	"fmt"
	"io/ioutil"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ClusterListOptions struct {
	options.BaseListOptions
}

func (o ClusterListOptions) Params() *jsonutils.JSONDict {
	o.Details = options.Bool(true)
	params, err := o.BaseListOptions.Params()
	if err != nil {
		panic(err)
	}
	return params
}

type ClusterCreateOptions struct {
	NAME       string `help:"Name of cluster"`
	Mode       string `help:"Cluster mode" choices:"internal"`
	K8sVersion string `help:"Cluster kubernetes components version" choices:"v1.8.10|v1.9.5|v1.10.0"`
	InfraImage string `help:"Cluster kubelet infra container image"`
	Cidr       string `help:"Cluster service CIDR, e.g. 10.43.0.0/16"`
	Domain     string `help:"Cluster pod domain, e.g. cluster.local"`
}

func (o ClusterCreateOptions) Params() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.NAME), "name")
	if o.Mode != "" {
		params.Add(jsonutils.NewString(o.Mode), "mode")
	}
	if o.K8sVersion != "" {
		params.Add(jsonutils.NewString(o.K8sVersion), "k8s_version")
	}
	if o.InfraImage != "" {
		params.Add(jsonutils.NewString(o.InfraImage), "infra_container_image")
	}
	if o.Cidr != "" {
		params.Add(jsonutils.NewString(o.Cidr), "cluster_cidr")
	}
	if o.Domain != "" {
		params.Add(jsonutils.NewString(o.Domain), "cluster_domain")
	}
	return params
}

type ClusterImportOptions struct {
	NAME       string `help:"Name of cluster to import"`
	KUBECONFIG string `help:"Kubernetes auth config"`
}

func (o ClusterImportOptions) Params() (*jsonutils.JSONDict, error) {
	kubeconfig, err := ioutil.ReadFile(o.KUBECONFIG)
	if err != nil {
		return nil, fmt.Errorf("Read kube config %q error: %v", o.KUBECONFIG, err)
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(string(kubeconfig)), "kube_config")
	return params, nil
}

type ClusterUpdateOptions struct {
	NAME       string `help:"Name of cluster"`
	K8sVersion string `help:"Cluster kubernetes components version" choices:"v1.8.10|v1.9.5|v1.10.0"`
}

func (o ClusterUpdateOptions) Params() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	if o.K8sVersion != "" {
		params.Add(jsonutils.NewString(o.K8sVersion), "k8s_version")
	}
	return params
}

type IdentOptions struct {
	ID string `help:"ID or name of the model"`
}

type IdentsOptions struct {
	ID []string `help:"ID of models to operate"`
}

type ClusterDeployOptions struct {
	IdentOptions
	Force bool `help:"Force deploy"`
}

func (o ClusterDeployOptions) Params() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	if o.Force {
		params.Add(jsonutils.JSONTrue, "force")
	}
	return params
}

type ClusterDeleteOptions struct {
	IdentsOptions
}

type ClusterKubeconfigOptions struct {
	IdentOptions
	Directly bool `help:"Get directly connect kubeconfig"`
}

func (o ClusterKubeconfigOptions) Params() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	if o.Directly {
		params.Add(jsonutils.JSONTrue, "directly")
	}
	return params
}

type ClusterAddNodesOptions struct {
	IdentOptions
	NodeConfig []string `help:"Node spec, 'host:[roles]' e.g: --node-config host01:controlplane,etcd,worker --node-config host02:worker"`
	AutoDeploy bool     `help:"Auto deploy"`
}

func (o ClusterAddNodesOptions) Params() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	if o.AutoDeploy {
		params.Add(jsonutils.JSONTrue, "auto_deploy")
	}
	nodesArray := jsonutils.NewArray()
	for _, config := range o.NodeConfig {
		opt, err := parseNodeAddConfigStr(config)
		if err != nil {
			return nil, err
		}
		nodesArray.Add(jsonutils.Marshal(opt))
	}
	params.Add(nodesArray, "nodes")
	return params, nil
}

type dockerConfig struct {
	RegistryMirrors    []string `json:"registry-mirrors"`
	InsecureRegistries []string `json:"insecure-registries"`
}

type nodeAddConfig struct {
	Host             string       `json:"host"`
	Roles            []string     `json:"roles"`
	Name             string       `json:"name"`
	HostnameOverride string       `json:"hostname_override"`
	DockerdConfig    dockerConfig `json:"dockerd_config"`
}

func parseNodeAddConfigStr(config string) (nodeAddConfig, error) {
	ret := nodeAddConfig{}
	parts := strings.Split(config, ":")
	if len(parts) != 2 {
		return ret, fmt.Errorf("Invalid config: %q", config)
	}
	host := parts[0]
	roleStr := parts[1]
	ret.Host = host
	roles := []string{}
	for _, role := range strings.Split(roleStr, ",") {
		if !sets.NewString("etcd", "controlplane", "worker").Has(role) {
			return ret, fmt.Errorf("Invalid role: %q", role)
		}
		roles = append(roles, role)
	}
	ret.Roles = roles
	return ret, nil
}

type ClusterDeleteNodesOptions struct {
	IdentOptions
	Node []string `help:"Node id or name"`
}

func (o ClusterDeleteNodesOptions) Params() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	nodesArray := jsonutils.NewArray()
	for _, node := range o.Node {
		nodesArray.Add(jsonutils.NewString(node))
	}
	params.Add(nodesArray, "nodes")
	return params, nil
}
