package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initPod() {
	initK8sNamespaceResource("pod", k8s.Pods)
}
