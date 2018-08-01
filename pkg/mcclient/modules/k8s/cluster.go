package k8s

import (
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

var (
	Clusters modules.ResourceManager
)

func init() {
	Clusters = NewManager("cluster", "clusters",
		[]string{"id", "name", "mode", "k8s_version", "status", "api_endpoint"},
		[]string{})
	modules.Register(&Clusters)
}
