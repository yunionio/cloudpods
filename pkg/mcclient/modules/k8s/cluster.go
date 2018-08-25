package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Clusters *modules.ResourceManager
)

func init() {
	Clusters = NewManager("kube_cluster", "kube_clusters",
		NewResourceCols("mode", "k8s_version", "status", "api_endpoint"),
		NewColumns())
	modules.Register(Clusters)
}
