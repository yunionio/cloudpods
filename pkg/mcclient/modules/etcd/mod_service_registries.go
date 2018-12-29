package etcd

import "yunion.io/x/onecloud/pkg/mcclient/modules"

type SServiceRegistryManager struct {
	modules.ResourceManager
}

var ServiceRegistryManager SServiceRegistryManager

func init() {
	ServiceRegistryManager = SServiceRegistryManager{
		ResourceManager: NewCloudirManager(
			"service-registry",
			"service-registries",
			[]string{},
			[]string{},
		),
	}

	modules.Register(&ServiceRegistryManager)
}
