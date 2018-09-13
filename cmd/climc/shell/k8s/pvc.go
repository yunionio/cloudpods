package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initPVC() {
	initK8sNamespaceResource("persistentvolumeclaim", k8s.PersistentVolumeClaims)
}
