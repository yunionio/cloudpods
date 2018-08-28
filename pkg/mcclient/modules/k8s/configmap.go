package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var ConfigMaps *ConfigMapManager

type ConfigMapManager struct {
	modules.ResourceManager
}

func init() {
	ConfigMaps = &ConfigMapManager{
		ResourceManager: *NewManager(
			"configmap", "configmaps",
			NewNamespaceCols(),
			NewClusterCols())}
	modules.Register(ConfigMaps)
}
