// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

func (m PodManager) Get_IP(obj jsonutils.JSONObject) interface{} {
	ip, _ := obj.GetString("podIP")
	return ip
}

func (m PodManager) Get_Restarts(obj jsonutils.JSONObject) interface{} {
	count, _ := obj.Int("restartCount")
	return count
}

func (m PodManager) Get_Node(obj jsonutils.JSONObject) interface{} {
	node, _ := obj.GetString("nodeName")
	return node
}
