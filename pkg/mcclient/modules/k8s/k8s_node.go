package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	K8sNodes *K8sNodeManager
)

type K8sNodeManager struct {
	*MetaResourceManager
}

func init() {
	K8sNodes = &K8sNodeManager{
		MetaResourceManager: NewMetaResourceManager("k8s_node", "k8s_nodes", NewColumns(), NewColumns()),
	}

	modules.Register(K8sNodes)
}
