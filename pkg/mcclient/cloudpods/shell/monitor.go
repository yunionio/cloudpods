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

package shell

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/shellutils"

	"yunion.io/x/onecloud/pkg/mcclient/cloudpods"
)

func init() {
	shellutils.R(&cloudprovider.MetricListOptions{}, "metric-list", "List metrics in a namespace", func(cli *cloudpods.SRegion, args *cloudprovider.MetricListOptions) error {
		metrics, err := cli.GetClient().GetMetrics(args)
		if err != nil {
			return err
		}
		for i := range metrics {
			log.Infof("metric: %s %s %s", metrics[i].Id, metrics[i].MetricType, metrics[i].Unit)
			printList(metrics[i].Values, len(metrics[i].Values), 0, 0, []string{})
		}
		return nil
	})
}
