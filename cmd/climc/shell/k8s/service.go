package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initService() {
	initK8sNamespaceResource("service", k8s.Services)
}
