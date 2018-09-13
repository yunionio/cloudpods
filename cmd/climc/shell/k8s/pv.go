package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initPV() {
	initK8sClusterResource("persistentvolume", k8s.PersistentVolumes)
}
