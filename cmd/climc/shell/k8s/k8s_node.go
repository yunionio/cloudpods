package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initK8sNode() {
	initK8sClusterResource("node", k8s.K8sNodes)
}
