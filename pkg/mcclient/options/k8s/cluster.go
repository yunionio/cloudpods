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

type K8sSupportVersion struct {
	K8sVersion string `help:"Cluster kubernetes components version" choices:"v1.10.5|v1.11.3|v1.12.3"`
}

type ClusterCreateOptions struct {
	K8sSupportVersion
	NAME       string `help:"Name of cluster"`
	Mode       string `help:"Cluster mode" choices:"internal"`
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

type KubeClusterCreateOptions struct {
	NAME          string   `help:"Name of cluster"`
	ClusterType   string   `help:"Cluster cluster type" choices:"default|serverless"`
	CloudType     string   `help:"Cluster cloud type" choices:"private|public|hybrid"`
	Mode          string   `help:"Cluster mode type" choices:"customize|managed"`
	Provider      string   `help:"Cluster provider" choices:"onecloud|aws|aliyun|azure|qcloud"`
	ServiceCidr   string   `help:"Cluster service CIDR, e.g. 10.43.0.0/16"`
	ServiceDomain string   `help:"Cluster service domain, e.g. cluster.local"`
	Vip           string   `help:"Cluster api server static loadbalancer vip"`
	Version       string   `help:"Cluster kubernetes version"`
	Machine       []string `help:"Machine create desc, e.g. host01:baremetal:controlplane"`
}

func parseMachineDesc(desc string) (*MachineCreateOptions, error) {
	matchType := func(p string) bool {
		switch p {
		case "baremetal", "vm":
			return true
		default:
			return false
		}
	}
	matchRole := func(p string) bool {
		switch p {
		case "controlplane", "node":
			return true
		default:
			return false
		}
	}
	mo := new(MachineCreateOptions)
	for _, part := range strings.Split(desc, ":") {
		switch {
		case matchType(part):
			mo.Type = part
		case matchRole(part):
			mo.ROLE = part
		default:
			mo.Instance = part
		}
	}
	if mo.ROLE == "" {
		return nil, fmt.Errorf("Machine role is empty")
	}
	if mo.Type == "" {
		return nil, fmt.Errorf("Machine type is empty")
	}
	return mo, nil
}

func (o KubeClusterCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.NAME), "name")
	if o.ClusterType != "" {
		params.Add(jsonutils.NewString(o.ClusterType), "cluster_type")
	}
	if o.CloudType != "" {
		params.Add(jsonutils.NewString(o.CloudType), "cloud_type")
	}
	if o.Mode != "" {
		params.Add(jsonutils.NewString(o.Mode), "mode")
	}
	if o.Provider != "" {
		params.Add(jsonutils.NewString(o.Provider), "provider")
	}
	if o.ServiceCidr != "" {
		params.Add(jsonutils.NewString(o.ServiceCidr), "service_cidr")
	}
	if o.ServiceDomain != "" {
		params.Add(jsonutils.NewString(o.ServiceDomain), "service_domain")
	}
	if o.Vip != "" {
		params.Add(jsonutils.NewString(o.Vip), "vip")
	}
	if len(o.Machine) != 0 {
		machineObjs := jsonutils.NewArray()
		for _, m := range o.Machine {
			machine, err := parseMachineDesc(m)
			if err != nil {
				return nil, err
			}
			machineObjs.Add(machine.Params())
		}
		params.Add(machineObjs, "machines")
	}
	return params, nil
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
	NAME string `help:"Name of cluster"`
	K8sSupportVersion
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

type ClusterK8sVersions struct {
	PROVIDER string `help:"cluster provider" choices:"system|onecloud"`
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

type KubeClusterAddMachinesOptions struct {
	IdentOptions
	Machine []string `help:"Node spec, 'host:[role]' e.g: --machine host01:controlplane:baremetal --node-config host02:node:baremetal"`
}

func (o KubeClusterAddMachinesOptions) Params() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	machinesArray := jsonutils.NewArray()
	for _, config := range o.Machine {
		opt, err := parseMachineDesc(config)
		if err != nil {
			return nil, err
		}
		machinesArray.Add(jsonutils.Marshal(opt))
	}
	params.Add(machinesArray, "machines")
	return params, nil
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

type KubeClusterDeleteMachinesOptions struct {
	IdentOptions
	Machines []string `help:"Machine id or name"`
}

func (o KubeClusterDeleteMachinesOptions) Params() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	machinesArray := jsonutils.NewArray()
	for _, m := range o.Machines {
		machinesArray.Add(jsonutils.NewString(m))
	}
	params.Add(machinesArray, "machines")
	return params, nil
}

type ClusterRestartAgentsOptions struct {
	ClusterDeleteNodesOptions
	All bool `help:"Restart all nodes agent"`
}

func (o ClusterRestartAgentsOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := o.ClusterDeleteNodesOptions.Params()
	if err != nil {
		return nil, err
	}
	all := jsonutils.JSONFalse
	if o.All {
		all = jsonutils.JSONTrue
	}
	params.Add(all, "all")
	return params, nil
}
