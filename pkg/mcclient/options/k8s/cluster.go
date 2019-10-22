// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8s

import (
	"fmt"
	"io/ioutil"
	"strings"

	"yunion.io/x/jsonutils"

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

type AddMachineOptions struct {
	Machine       []string `help:"Machine create desc, e.g. host01:baremetal:controlplane"`
	MachineNet    string   `help:"Machine net config"`
	MachineDisk   string   `help:"Machine root disk size, e.g. 100G"`
	MachineCpu    int      `help:"Machine cpu count"`
	MachineMemory string   `help:"Machine memory size, e.g. 1G"`
}

type KubeClusterCreateOptions struct {
	NAME          string `help:"Name of cluster"`
	ClusterType   string `help:"Cluster cluster type" choices:"default|serverless"`
	ResourceType  string `help:"Cluster cluster type" choices:"host|guest"`
	CloudType     string `help:"Cluster cloud type" choices:"private|public|hybrid"`
	Mode          string `help:"Cluster mode type" choices:"customize|managed|import"`
	Provider      string `help:"Cluster provider" choices:"onecloud|aws|aliyun|azure|qcloud|system"`
	ServiceCidr   string `help:"Cluster service CIDR, e.g. 10.43.0.0/16"`
	ServiceDomain string `help:"Cluster service domain, e.g. cluster.local"`
	Vip           string `help:"Cluster api server static loadbalancer vip"`
	Version       string `help:"Cluster kubernetes version"`

	AddMachineOptions
}

type KubeClusterImportOptions struct {
	NAME       string `help:"Name of cluster"`
	APISERVER  string `help:"API server of this cluster"`
	KUBECONFIG string `help:"Cluster kubeconfig file path"`
}

func parseMachineDesc(desc string, disk string, netConf string, ncpu int, memorySize string) (*MachineCreateOptions, error) {
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
	mo.Disk = disk
	mo.Cpu = ncpu
	mo.Memory = memorySize
	mo.Net = netConf
	return mo, nil
}

func (o KubeClusterCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.NAME), "name")
	if o.ClusterType != "" {
		params.Add(jsonutils.NewString(o.ClusterType), "cluster_type")
	}
	if o.ResourceType != "" {
		params.Add(jsonutils.NewString(o.ResourceType), "resource_type")
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
	machineObjs, err := o.AddMachineOptions.Params()
	if err != nil {
		return nil, err
	}
	params.Add(machineObjs, "machines")
	return params, nil
}

func (o KubeClusterImportOptions) Params() (*jsonutils.JSONDict, error) {
	kubeconfig, err := ioutil.ReadFile(o.KUBECONFIG)
	if err != nil {
		return nil, fmt.Errorf("Read kube config %q error: %v", o.KUBECONFIG, err)
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.NAME), "name")
	params.Add(jsonutils.NewString("import"), "mode")
	params.Add(jsonutils.NewString(o.APISERVER), "api_server")
	params.Add(jsonutils.NewString(string(kubeconfig)), "kubeconfig")
	params.Add(jsonutils.NewString("external"), "provider")
	params.Add(jsonutils.NewString("unknown"), "resource_type")
	return params, nil
}

type IdentOptions struct {
	ID string `help:"ID or name of the model"`
}

type ClusterK8sVersions struct {
	PROVIDER string `help:"cluster provider" choices:"system|onecloud"`
}

type ClusterCheckOptions struct{}

type IdentsOptions struct {
	ID []string `help:"ID of models to operate"`
}

type ClusterDeleteOptions struct {
	IdentsOptions
}

type KubeClusterAddMachinesOptions struct {
	IdentOptions
	AddMachineOptions
}

func (o AddMachineOptions) Params() (*jsonutils.JSONArray, error) {
	machineObjs := jsonutils.NewArray()
	if len(o.Machine) == 0 {
		return machineObjs, nil
	}
	for _, m := range o.Machine {
		machine, err := parseMachineDesc(m, o.MachineDisk, o.MachineNet, o.MachineCpu, o.MachineMemory)
		if err != nil {
			return nil, err
		}
		params, err := machine.Params()
		if err != nil {
			return nil, err
		}
		machineObjs.Add(params)
	}
	return machineObjs, nil
}

func (o KubeClusterAddMachinesOptions) Params() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	machinesArray, err := o.AddMachineOptions.Params()
	if err != nil {
		return nil, err
	}
	params.Add(machinesArray, "machines")
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
