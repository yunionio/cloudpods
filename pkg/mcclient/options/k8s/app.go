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
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sets"
)

type K8sAppBaseCreateOptions struct {
	NamespaceWithClusterOptions
	ServiceSpecOptions
	NAME            string   `help:"Name of deployment"`
	Image           string   `help:"The image for the container to run" required:"true"`
	Replicas        int64    `help:"Number of replicas for pods in this deployment"`
	RunAsPrivileged bool     `help:"Whether to run the container as privileged user"`
	RegistrySecret  string   `help:"Docker registry secret"`
	Label           []string `help:"Labels to apply to the pod(s), e.g. 'env=prod'"`
	Env             []string `help:"Environment variables to set in container"`
	Net             string   `help:"Network config, e.g. net1, net1:10.168.222.171"`
	Mem             int      `help:"Memory request MB size"`
	Cpu             float64  `help:"Cpu request cores"`
	Command         string   `help:"Container start command"`
	CommandArgs     string   `help:"Container start command args"`
	Pvc             []string `help:"PVC volume desc, format is <pvc_name>:<mount_point>"`
}

func (o K8sAppBaseCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params := o.NamespaceWithClusterOptions.Params()
	params.Add(jsonutils.NewString(o.NAME), "name")
	if len(o.Image) == 0 {
		return nil, fmt.Errorf("Image must provided")
	}
	params.Add(jsonutils.NewString(o.Image), "containerImage")
	if o.Replicas > 1 {
		params.Add(jsonutils.NewInt(o.Replicas), "replicas")
	}
	if o.RunAsPrivileged {
		params.Add(jsonutils.JSONTrue, "runAsPrivileged")
	}

	svcSpec, err := o.ServiceSpecOptions.Params()
	if err != nil {
		return nil, err
	}
	params.Update(svcSpec)

	envList := jsonutils.NewArray()
	for _, env := range o.Env {
		parts := strings.Split(env, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("Bad env value: %v", env)
		}
		envObj := jsonutils.NewDict()
		envObj.Add(jsonutils.NewString(parts[0]), "name")
		envObj.Add(jsonutils.NewString(parts[1]), "value")
		envList.Add(envObj)
	}
	params.Add(envList, "variables")
	if o.Net != "" {
		net, err := parseNetConfig(o.Net)
		if err != nil {
			return nil, err
		}
		params.Add(net, "networkConfig")
	}
	labels := jsonutils.NewArray()
	for _, label := range o.Label {
		label, err := parseLabel(label)
		if err != nil {
			return nil, err
		}
		labels.Add(label)
	}
	params.Add(labels, "labels")

	if o.Cpu > 0 {
		params.Add(jsonutils.NewString(fmt.Sprintf("%dm", int64(o.Cpu*1000))), "cpuRequirement")
	}
	if o.Mem > 0 {
		params.Add(jsonutils.NewString(fmt.Sprintf("%dMi", o.Mem)), "memoryRequirement")
	}
	if o.RegistrySecret != "" {
		params.Add(jsonutils.NewString(o.RegistrySecret), "imagePullSecret")
	}
	if o.Command != "" {
		params.Add(jsonutils.NewString(o.Command), "containerCommand")
	}
	if o.CommandArgs != "" {
		params.Add(jsonutils.NewString(o.CommandArgs), "containerCommandArgs")
	}
	vols := jsonutils.NewArray()
	volMounts := jsonutils.NewArray()
	for _, pvc := range o.Pvc {
		vol, volMount, err := parsePvc(pvc)
		if err != nil {
			return nil, err
		}
		vols.Add(vol)
		volMounts.Add(volMount)
	}
	params.Add(vols, "volumes")
	params.Add(volMounts, "volumeMounts")
	return params, nil
}

type portMapping struct {
	Port       int32  `json:"port"`
	TargetPort int32  `json:"targetPort"`
	Protocol   string `json:"protocol"`
}

func parsePortMapping(port string) (*portMapping, error) {
	if len(port) == 0 {
		return nil, fmt.Errorf("empty port mapping desc string")
	}
	parts := strings.Split(port, ":")
	mapping := &portMapping{}
	for _, part := range parts {
		if sets.NewString("tcp", "udp").Has(strings.ToLower(part)) {
			mapping.Protocol = strings.ToUpper(part)
		}
		if port, err := strconv.Atoi(part); err != nil {
			continue
		} else {
			if mapping.Port == 0 {
				mapping.Port = int32(port)
			} else {
				mapping.TargetPort = int32(port)
			}
		}
	}
	if mapping.Protocol == "" {
		mapping.Protocol = "TCP"
	}
	if mapping.Port <= 0 {
		return nil, fmt.Errorf("Service port not provided")
	}
	if mapping.TargetPort < 0 {
		return nil, fmt.Errorf("Container invalid targetPort %d", mapping.TargetPort)
	}
	if mapping.TargetPort == 0 {
		mapping.TargetPort = mapping.Port
	}
	return mapping, nil
}

func parsePortMappings(ports []string) (*jsonutils.JSONArray, error) {
	ret := jsonutils.NewArray()
	for _, port := range ports {
		mapping, err := parsePortMapping(port)
		if err != nil {
			return nil, fmt.Errorf("Port %q error: %v", port, err)
		}
		ret.Add(jsonutils.Marshal(mapping))
	}
	return ret, nil
}

func parseNetConfig(net string) (*jsonutils.JSONDict, error) {
	ret := jsonutils.NewDict()
	for _, p := range strings.Split(net, ":") {
		if regutils.MatchIP4Addr(p) {
			ret.Add(jsonutils.NewString(p), "address")
		} else {
			ret.Add(jsonutils.NewString(p), "network")
		}
	}
	return ret, nil
}

type K8sAppCreateFromFileOptions struct {
	NamespaceResourceGetOptions
	FILE string `help:"K8s resource YAML or JSON file"`
}

func (o K8sAppCreateFromFileOptions) Params() (*jsonutils.JSONDict, error) {
	params := o.NamespaceResourceGetOptions.Params()
	params.Add(jsonutils.NewString(o.NAME), "name")
	content, err := ioutil.ReadFile(o.FILE)
	if err != nil {
		return nil, err
	}
	params.Add(jsonutils.NewString(string(content)), "content")
	return params, nil
}

func parseLabel(str string) (jsonutils.JSONObject, error) {
	parts := strings.Split(str, "=")
	if len(parts) != 2 {
		return nil, fmt.Errorf("Invalid label string: %s", str)
	}
	label := jsonutils.NewDict()
	label.Add(jsonutils.NewString(parts[0]), "key")
	label.Add(jsonutils.NewString(parts[1]), "value")
	return label, nil
}

func parsePvc(pvcDesc string) (jsonutils.JSONObject, jsonutils.JSONObject, error) {
	parts := strings.Split(pvcDesc, ":")
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("Invalid PVC desc string: %s", pvcDesc)
	}
	pvcName := parts[0]
	pvcMntPath := parts[1]

	pvcVol := jsonutils.NewDict()
	pvcVol.Add(jsonutils.NewString(pvcName), "claimName")
	vol := jsonutils.NewDict()
	vol.Add(jsonutils.NewString(pvcName), "name")
	vol.Add(pvcVol, "persistentVolumeClaim")

	volMnt := jsonutils.NewDict()
	volMnt.Add(jsonutils.NewString(pvcName), "name")
	volMnt.Add(jsonutils.NewString(pvcMntPath), "mountPath")

	return vol, volMnt, nil
}

func parsePvcTemplate(pvcDesc string) (jsonutils.JSONObject, jsonutils.JSONObject, error) {
	parts := strings.Split(pvcDesc, ":")
	if len(parts) != 3 {
		return nil, nil, fmt.Errorf("Invalid PVC desc string: %s", pvcDesc)
	}
	pvcName := parts[0]
	pvcSize := parts[1]
	pvcMntPath := parts[2]

	vol := jsonutils.NewDict()
	vol.Add(jsonutils.NewString(pvcName), "name")
	vol.Add(jsonutils.NewString(pvcSize), "size")

	volMnt := jsonutils.NewDict()
	volMnt.Add(jsonutils.NewString(pvcName), "name")
	volMnt.Add(jsonutils.NewString(pvcMntPath), "mountPath")

	return vol, volMnt, nil
}
