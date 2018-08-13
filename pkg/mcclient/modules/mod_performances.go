package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type PerformanceManager struct {
	ResourceManager
}

var (
	Performances PerformanceManager
)

func (this *PerformanceManager) GetTop5(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	node_labels, err := params.GetString("node_labels")

	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}

	path := fmt.Sprintf("/%s/top5?node_labels=%s", this.KeywordPlural, node_labels)

	return this._get(s, path, this.Keyword)
}

func init() {
	Performances = PerformanceManager{NewMonitorManager("performance", "performances",
		[]string{"cpu_idle", "memory_usage", "disk_ioread", "disk_iowrite", "traffic_in", "traffic_out"},
		[]string{})}

	register(&Performances)
}
