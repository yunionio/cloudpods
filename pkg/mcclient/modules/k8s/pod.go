package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var Pods *PodManager

type PodManager struct {
	modules.ResourceManager
}

func init() {
	Pods = &PodManager{
		ResourceManager: *NewManager(
			"pod", "pods",
			[]string{"id", "name", "cluster", "roles", "address", "status"},
			[]string{})}
	modules.Register(Pods)
}
