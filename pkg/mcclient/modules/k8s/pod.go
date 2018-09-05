package k8s

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var Pods *PodManager

type PodManager struct {
	*NamespaceResourceManager
	statusGetter
}

func init() {
	Pods = &PodManager{
		NamespaceResourceManager: NewNamespaceResourceManager("pod", "pods",
			NewNamespaceCols("IP", "Status", "Restarts"),
			NewClusterCols("Node")),
		statusGetter: getStatus,
	}

	modules.Register(Pods)
}

func (m PodManager) GetIP(obj jsonutils.JSONObject) interface{} {
	ip, _ := obj.GetString("podIP")
	return ip
}

func (m PodManager) GetRestarts(obj jsonutils.JSONObject) interface{} {
	count, _ := obj.Int("restartCount")
	return count
}

func (m PodManager) GetNode(obj jsonutils.JSONObject) interface{} {
	node, _ := obj.GetString("nodeName")
	return node
}
