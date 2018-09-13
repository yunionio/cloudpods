package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initStatefulset() {
	initK8sNamespaceResource("statefulset", k8s.StatefulSets)
}
