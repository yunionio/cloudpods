package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initConfigMap() {
	initK8sNamespaceResource("configmap", k8s.ConfigMaps)
}
