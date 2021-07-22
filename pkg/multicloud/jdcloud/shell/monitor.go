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
	"yunion.io/x/onecloud/pkg/multicloud/jdcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DescribeMetricDataOptions struct {
		/* 监控项英文标识(id)  */
		Metric       string `help:"metric name"json:"metric"`
		TimeInterval string `help:"time interval" choices:"1h|6h|12h|1d|3d|7d|14d" json:"timeInterval"`
		ServiceCode  string `help:"resource code" choices:"vm" json:"serviceCode"`
		ResourceId   string `help:"resource id" json:"resourceId"`
	}
	shellutils.R(&DescribeMetricDataOptions{}, "metric-list", "list metric", func(cli *jdcloud.SRegion,
		args *DescribeMetricDataOptions) error {
		request := jdcloud.NewDescribeMetricDataRequestWithAllParams(cli.GetId(),
			args.Metric, nil, nil, &args.TimeInterval, &args.ServiceCode, args.ResourceId)
		response, err := cli.GetMetricsData(request)
		if err != nil {
			return err
		}
		printList(response.Result.MetricDatas, 0, 0, 0, nil)
		return nil
	})
}
