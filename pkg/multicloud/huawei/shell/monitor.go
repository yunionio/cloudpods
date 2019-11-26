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
	"yunion.io/x/pkg/util/timeutils"

	"yunion.io/x/onecloud/pkg/multicloud/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type MetricListOptions struct {
	}
	shellutils.R(&MetricListOptions{}, "metrics-list", "List metrics", func(cli *huawei.SRegion, args *MetricListOptions) error {
		metrics, err := cli.GetMetrics()
		if err != nil {
			return err
		}
		printList(metrics, 0, 0, 0, nil)
		return nil
	})

	type MetricDataOptions struct {
		START int    `help:"Start metrics"`
		Count int    `help:"Metric count" default:"1"`
		SINCE string `help:"since"`
		UNTIL string `help:"until"`
	}
	shellutils.R(&MetricDataOptions{}, "metrics-data-list", "List metrics", func(cli *huawei.SRegion, args *MetricDataOptions) error {
		metrics, err := cli.GetMetrics()
		if err != nil {
			return err
		}
		since, err := timeutils.ParseTimeStr(args.SINCE)
		if err != nil {
			return err
		}
		until, err := timeutils.ParseTimeStr(args.UNTIL)
		if err != nil {
			return err
		}
		data, err := cli.GetMetricsData(metrics[args.START:args.START+args.Count], since, until)
		if err != nil {
			return err
		}
		printList(data, 0, 0, 0, nil)
		return nil
	})
}
