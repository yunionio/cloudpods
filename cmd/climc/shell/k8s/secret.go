package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initSecret() {
	initK8sNamespaceResource("secret", k8s.Secrets)
}
