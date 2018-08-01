package modules

import (
	"fmt"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
)

type TreenodeManager struct {
	ResourceManager
}

var (
	TreeNodes TreenodeManager
)

func (this *TreenodeManager) GetMap(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/getMap", this.KeywordPlural)
	return this._get(s, path, this.KeywordPlural)
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
	TreeNodes = TreenodeManager{NewMonitorManager("tree_node", "tree_nodes",
		[]string{"id", "name", "pid", "order_no", "level", "group", "status", "project_id", "remark"},
		[]string{})}

	register(&TreeNodes)
}
