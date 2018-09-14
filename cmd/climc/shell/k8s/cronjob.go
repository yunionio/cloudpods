package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initCronJob() {
	initK8sNamespaceResource("cronjob", k8s.CronJobs)
}
