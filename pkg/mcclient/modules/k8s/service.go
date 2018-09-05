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

func (s ServiceManager) GetType(obj jsonutils.JSONObject) interface{} {
	typ, _ := obj.GetString("type")
	return typ
}

func (s ServiceManager) GetClusterIP(obj jsonutils.JSONObject) interface{} {
	clusterIp, _ := obj.GetString("clusterIP")
	return clusterIp
}

func (s ServiceManager) GetSelector(obj jsonutils.JSONObject) interface{} {
	selectorObj, _ := obj.GetMap("selector")
	var selectors []string
	for k, obj := range selectorObj {
		val, _ := obj.GetString()
		selectors = append(selectors, fmt.Sprintf("%s=%s", k, val))
	}
	selectorStr := strings.Join(selectors, ",")
	return selectorStr
}

func (s ServiceManager) GetPorts(obj jsonutils.JSONObject) interface{} {
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
