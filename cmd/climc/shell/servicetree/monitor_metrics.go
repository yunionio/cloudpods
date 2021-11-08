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

package servicetree

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/servicetree"
)

func init() {

	/**
	 * 添加/修改树节点的统计指标（配置服务树某个节点都有什么监控指标）
	 */
	type MonitorMetricsUpdateOptions struct {
		LABEL      []string `help:"Labels to this tree node"`
		MetricName []string `help:"Metric names to add to" nargs:"+"`
	}
	R(&MonitorMetricsUpdateOptions{}, "monitor-metrics-set", "Set monitor metric for the tree-node", func(s *mcclient.ClientSession, args *MonitorMetricsUpdateOptions) error {
		names := []string{"corp", "owt", "pdl", "srv", "env"}
		segs := make([]string, len(args.LABEL))

		for i := 0; i < len(args.LABEL); i += 1 {
			sublabel := args.LABEL[:i+1]
			pid, _ := modules.TreeNodes.GetNodeIDByLabels(s, sublabel)
			segs[i] = fmt.Sprintf("%s=%d", names[i], pid)
		}

		node_labels := strings.Join(segs, ",")

		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(node_labels), "node_labels")
		arr := jsonutils.NewArray()
		if len(args.MetricName) > 0 {
			for _, f := range args.MetricName {
				arr.Add(jsonutils.NewString(f))
			}
		}
		params.Add(arr, "metrics")

		rst, err := modules.MonitorMetrics.Create(s, params)

		if err != nil {
			return err
		}

		printObject(rst)
		return nil
	})

}
