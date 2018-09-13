package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initNamespace() {
	initK8sClusterResource("namespace", k8s.Namespaces)
}
