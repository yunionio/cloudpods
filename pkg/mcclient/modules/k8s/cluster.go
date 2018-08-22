package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Clusters *ResourceManager
)

func init() {
	Clusters = NewResourceManager("kube_cluster", "kube_clusters",
		NewResourceCols("mode", "k8s_version", "status", "api_endpoint"),
		NewColumns("is_public"))
	modules.Register(Clusters)
}
