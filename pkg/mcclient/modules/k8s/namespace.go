package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var Namespaces *NamespaceManager

type NamespaceManager struct {
	*MetaResourceManager
	statusGetter
}

func init() {
	Namespaces = &NamespaceManager{
		MetaResourceManager: NewMetaResourceManager("namespace", "namespaces", NewNameCols("Status"), NewClusterCols()),
		statusGetter:        getStatus,
	}

	modules.Register(Namespaces)
}
