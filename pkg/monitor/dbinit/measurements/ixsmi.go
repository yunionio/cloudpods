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

var ixsmi = SMeasurement{
	Context: []SMonitorContext{
		{
			"ixsmi", "Iluvatar GPU metrics",
			monitor.METRIC_RES_TYPE_HOST, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"temperature_gpu", "GPU temperature", "",
		},
		{
			"temperature_memory", "GPU memory temperature", "",
		},
		{
			"memory_total", "GPU memory total size", "",
		},
		{
			"memory_free", "GPU memory free size", "",
		},
		{
			"memory_used", "GPU memory used size", "",
		},
		{
			"utilization_gpu", "GPU utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_memory", "GPU memory utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"power_draw", "GPU power draw", "",
		},
		{
			"clocks_current_sm", "GPU current SM clocks, MHz", "",
		},
		{
			"clocks_current_memory", "GPU current memory clocks, MHz", "",
		},
	},
}
