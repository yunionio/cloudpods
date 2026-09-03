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

var ppusmi = SMeasurement{
	Context: []SMonitorContext{
		{
			"ppusmi", "T-Head PPU metrics",
			monitor.METRIC_RES_TYPE_HOST, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"temperature_gpu", "PPU temperature", "",
		},
		{
			"temperature_memory", "PPU memory temperature", "",
		},
		{
			"memory_total", "PPU memory total size", "",
		},
		{
			"memory_free", "PPU memory free size", "",
		},
		{
			"memory_used", "PPU memory used size", "",
		},
		{
			"utilization_gpu", "PPU utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_memory", "PPU memory utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"power_draw", "PPU power draw", "",
		},
		{
			"clocks_current_sm", "PPU current CU clocks, MHz", "",
		},
		{
			"clocks_current_memory", "PPU current memory clocks, MHz", "",
		},
	},
}
