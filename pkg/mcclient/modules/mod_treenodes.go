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

package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type TreenodeManager struct {
	modulebase.ResourceManager
}

var (
	TreeNodes TreenodeManager
)

func (this *TreenodeManager) GetMap(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/getMap", this.KeywordPlural)
	return modulebase.Get(this.ResourceManager, s, path, this.KeywordPlural)
}

func (this *TreenodeManager) GetNodeIDByLabels(s *mcclient.ClientSession, labels []string) (int64, error) {
	var pid int64 = 0
	for _, label := range labels {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(label), "name")
		params.Add(jsonutils.NewInt(pid), "pid")
		nodeList, err := this.List(s, params)
		if err != nil {
			return -1, err
		}
		if len(nodeList.Data) != 1 {
			return -1, fmt.Errorf("Invalid label %s", label)
		}
		pid, err = nodeList.Data[0].Int("id")
		if err != nil {
			return -1, fmt.Errorf("Invalid node data")
		}
	}
	return pid, nil
}

func init() {
	TreeNodes = TreenodeManager{NewServiceTreeManager("tree_node", "tree_nodes",
		[]string{"id", "name", "pid", "order_no", "level", "group", "status", "project_id", "project_type", "create_way", "remark"},
		[]string{})}

	register(&TreeNodes)
}
