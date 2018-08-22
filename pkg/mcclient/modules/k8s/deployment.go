package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var Deployments *DeploymentManager

type DeploymentManager struct {
	*NamespaceResourceManager
}

func init() {
	Deployments = &DeploymentManager{
		NewNamespaceResourceManager("deployment", "deployments",
			NewColumns("labels"), NewColumns())}
	modules.Register(Deployments)
}
