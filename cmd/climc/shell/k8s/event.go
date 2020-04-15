package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initEvent() {
	initK8sNamespaceResource("event", k8s.Events)
}
