package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var Ingresses *IngressManager

type IngressManager struct {
	*NamespaceResourceManager
}

func init() {
	Ingresses = &IngressManager{
		NewNamespaceResourceManager("ingress", "ingresses",
			NewNamespaceCols(), NewColumns())}
	modules.Register(Ingresses)
}
