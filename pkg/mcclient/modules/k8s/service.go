package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var Services *ServiceManager

type ServiceManager struct {
	modules.ResourceManager
}

func init() {
	Services = &ServiceManager{
		ResourceManager: *NewManager(
			"service", "services",
			NewNamespaceCols("clusterIP", "selector", "internalEndpoint", "externalEndpoints"),
			NewClusterCols())}
	modules.Register(Services)
}
