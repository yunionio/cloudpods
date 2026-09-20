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

var xpusmi = SMeasurement{
	Context: []SMonitorContext{
		{
			"xpusmi", "Kunlunxin XPU metrics",
			monitor.METRIC_RES_TYPE_HOST, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"temperature_gpu", "XPU temperature", "",
		},
		{
			"memory_total", "XPU memory total size", "",
		},
		{
			"memory_free", "XPU memory free size", "",
		},
		{
			"memory_used", "XPU memory used size", "",
		},
		{
			"l3_memory_total", "XPU L3 memory total size", "",
		},
		{
			"l3_memory_free", "XPU L3 memory free size", "",
		},
		{
			"l3_memory_used", "XPU L3 memory used size", "",
		},
		{
			"utilization_gpu", "XPU utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"power_draw", "XPU power draw", "",
		},
		{
			"power_limit", "XPU power limit", "",
		},
		{
			"clocks_current_cluster", "XPU current cluster clocks, MHz", "",
		},
		{
			"clocks_current_cdnn", "XPU current CDNN clocks, MHz", "",
		},
		{
			"pcie_link_gen_current", "XPU current PCIe link generation", "",
		},
		{
			"pcie_link_width_current", "XPU current PCIe link width", "",
		},
		{
			"ecc_errors_dram_correctable", "XPU correctable DRAM ECC errors", "",
		},
		{
			"ecc_errors_dram_uncorrectable", "XPU uncorrectable DRAM ECC errors", "",
		},
		{
			"ecc_errors_dram_correctable_aggregate", "XPU correctable DRAM ECC errors in total", "",
		},
		{
			"ecc_errors_dram_uncorrectable_aggregate", "XPU uncorrectable DRAM ECC errors in total", "",
		},
	},
}
