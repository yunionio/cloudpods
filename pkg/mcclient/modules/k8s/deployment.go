package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var Deployments *DeploymentManager

type DeploymentManager struct {
	modules.ResourceManager
}

func init() {
	Deployments = &DeploymentManager{
		ResourceManager: *NewManager(
			"deployment", "deployments",
			NewNamespaceCols("labels"),
			NewClusterCols())}
	modules.Register(Deployments)
}
