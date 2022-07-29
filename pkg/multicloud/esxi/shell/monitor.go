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
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/esxi"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	shellutils.R(&cloudprovider.MetricListOptions{}, "metric-list", "List metrics in a namespace", func(cli *esxi.SESXiClient, args *cloudprovider.MetricListOptions) error {
		metrics, err := cli.GetMetrics(args)
		if err != nil {
			return err
		}
		for i := range metrics {
			log.Infof("metric %s %s", metrics[i].Id, metrics[i].MetricType)
			printList(metrics[i].Values, nil)
		}
		return nil
	})

	type SMetricTypeShowOptions struct {
	}

	shellutils.R(&SMetricTypeShowOptions{}, "metric-type-show", "List metrics", func(cli *esxi.SESXiClient, args *SMetricTypeShowOptions) error {
		metrics, err := cli.GetMetricTypes()
		if err != nil {
			return err
		}
		printObject(metrics)
		return nil
	})

}
