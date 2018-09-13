package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initJob() {
	initK8sNamespaceResource("job", k8s.Jobs)
}
