package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initIngress() {
	initK8sNamespaceResource("ingress", k8s.Ingresses)
}
