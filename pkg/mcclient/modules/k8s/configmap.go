package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var ConfigMaps *ConfigMapManager

type ConfigMapManager struct {
	*NamespaceResourceManager
}

func init() {
	ConfigMaps = &ConfigMapManager{
		NamespaceResourceManager: NewNamespaceResourceManager(
			"configmap", "configmaps",
			NewColumns(), NewColumns())}
	modules.Register(ConfigMaps)
}
