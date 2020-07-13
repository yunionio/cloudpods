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
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var Services *ServiceManager

type ServiceManager struct {
	*NamespaceResourceManager
}

func init() {
	Services = &ServiceManager{NewNamespaceResourceManager(
		"k8s_service", "k8s_services",
		NewNamespaceCols("Type", "ClusterIP", "Ports", "Selector"),
		NewColumns()),
	}
	modules.Register(Services)
}

func (s ServiceManager) Get_Type(obj jsonutils.JSONObject) interface{} {
	typ, _ := obj.GetString("type")
	return typ
}

func (s ServiceManager) Get_ClusterIP(obj jsonutils.JSONObject) interface{} {
	clusterIp, _ := obj.GetString("clusterIP")
	return clusterIp
}

func (s ServiceManager) Get_Selector(obj jsonutils.JSONObject) interface{} {
	selectorObj, _ := obj.GetMap("selector")
	var selectors []string
	for k, obj := range selectorObj {
		val, _ := obj.GetString()
		selectors = append(selectors, fmt.Sprintf("%s=%s", k, val))
	}
	selectorStr := strings.Join(selectors, ",")
	return selectorStr
}

func (s ServiceManager) Get_Ports(obj jsonutils.JSONObject) interface{} {
	var ports []string
	var portsStr string
	portObjs, _ := obj.GetArray("internalEndpoint", "ports")
	if len(portObjs) != 0 {
		for _, obj := range portObjs {
			port, _ := obj.Int("port")
			proto, _ := obj.GetString("protocol")
			ports = append(ports, fmt.Sprintf("%d/%s", port, proto))
		}
		portsStr = strings.Join(ports, ",")
	}
	return portsStr
}
