package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Deployments    *DeploymentManager
	DeployFromFile *DeployFromFileManager
)

type DeploymentManager struct {
	*NamespaceResourceManager
}

type DeployFromFileManager struct {
	*NamespaceResourceManager
}

func init() {
	Deployments = &DeploymentManager{
		NewNamespaceResourceManager("deployment", "deployments",
			NewNamespaceCols(), NewColumns())}

	DeployFromFile = &DeployFromFileManager{
		NewNamespaceResourceManager("deployfromfile", "deployfromfiles",
			NewNamespaceCols(), NewColumns())}

	modules.Register(Deployments)
	modules.Register(DeployFromFile)
}
