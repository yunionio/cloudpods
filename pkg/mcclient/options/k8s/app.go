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
}

func (o K8sAppBaseCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params := o.NamespaceWithClusterOptions.Params()

	svcSpec, err := o.ServiceSpecOptions.Params()
	if err != nil {
		return nil, err
	}
	params.Update(svcSpec)

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

func parseImage(str string) (jsonutils.JSONObject, error) {
	parts := strings.Split(str, "=")
	if len(parts) != 2 {
		return nil, fmt.Errorf("Invalid image string: %s", str)
	}
	ci := jsonutils.NewDict()
	ci.Add(jsonutils.NewString(parts[0]), "name")
	ci.Add(jsonutils.NewString(parts[1]), "image")
	return ci, nil
}
