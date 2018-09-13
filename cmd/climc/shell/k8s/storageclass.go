package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initStorageClass() {
	initK8sClusterResource("storageclass", k8s.Storageclass)
}
