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

package measurements

import "yunion.io/x/onecloud/pkg/apis/monitor"

var npuSmi = SMeasurement{
	Context: []SMonitorContext{
		{
			"npu_smi", "NPU metrics",
			monitor.METRIC_RES_TYPE_HOST, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"temperature", "NPU temperature", "",
		},
		{
			"npu_real_time_power", "NPU real time power", "",
		},
		{
			"npu_utilization", "NPU utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"hbm_usage_rate", "NPU hbm usage rate", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"hbm_capacity", "NPU hbm capacity", "",
		},
		{
			"aicore_usage_rate", "NPU aicore usage rate", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"aivector_usage_rate", "NPU aivector usage rate", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"aicube_usage_rate", "NPU aicube usage rate", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"aicpu_usage_rate", "NPU aicpu usage rate", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"hbm_bandwidth_usage_rate", "NPU hbm bandwidth usage rate", "",
		},
	},
}
