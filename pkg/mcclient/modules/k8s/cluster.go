package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Clusters     *ResourceManager
	KubeClusters *ResourceManager
)

func init() {
	Clusters = NewResourceManager("kube_cluster", "kube_clusters",
		NewResourceCols("mode", "k8s_version", "status", "api_endpoint"),
		NewColumns("is_public"))
	KubeClusters = NewResourceManager("kubecluster", "kubeclusters",
		NewResourceCols("cluster_type", "cloud_type", "version", "status", "mode", "provider", "machines"),
		NewColumns())
	modules.Register(Clusters)
	modules.Register(KubeClusters)
}
