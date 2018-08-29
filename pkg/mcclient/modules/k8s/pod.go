package k8s

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var Pods *PodManager

type PodManager struct {
	*NamespaceResourceManager
}

func init() {
	Pods = &PodManager{
		NewNamespaceResourceManager("pod", "pods",
			NewNamespaceCols("IP", "Status", "Restarts", "Labels"),
			NewClusterCols("Node"))}

	modules.Register(Pods)
}

func (m PodManager) GetIP(obj jsonutils.JSONObject) interface{} {
	ip, _ := obj.GetString("podIP")
	return ip
}

func (m PodManager) GetStatus(obj jsonutils.JSONObject) interface{} {
	status, _ := obj.GetString("status")
	return status
}

func (m PodManager) GetRestarts(obj jsonutils.JSONObject) interface{} {
	count, _ := obj.Int("restartCount")
	return count
}

func (m PodManager) GetLabels(obj jsonutils.JSONObject) interface{} {
	labels, _ := obj.GetMap("labels")
	str := ""
	ls := []string{}
	for k, v := range labels {
		vs, _ := v.GetString()
		ls = append(ls, fmt.Sprintf("%s=%s", k, vs))
	}
	if len(ls) != 0 {
		str = strings.Join(ls, ",")
	}
	return str
}

func (m PodManager) GetNode(obj jsonutils.JSONObject) interface{} {
	node, _ := obj.GetString("nodeName")
	return node
}
