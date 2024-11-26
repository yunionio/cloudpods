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

var radeontop = SMeasurement{
	Context: []SMonitorContext{
		{
			"radeontop", "AMD GPU metrics", monitor.METRIC_RES_TYPE_HOST, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"clocks_current_memory", "GPU current memory clocks, MHz", "",
		},
		{
			"clocks_current_shader", "GPU current shader clocks, MHz", "",
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
			"gtt_total", "GPU gtt total size", "",
		},
		{
			"gtt_free", "GPU gtt free size", "",
		},
		{
			"gtt_used", "GPU gtt used size", "",
		},
		{
			"utilization_clock_memory", "GPU block memory utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_clock_shader", "GPU block shader utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_gpu", "GPU utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_memory", "GPU memory utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_event_engine", "GPU event engine utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_vertex_grouper_tesselator", "GPU vertex grouper tesselator utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_texture_addresser", "GPU texture addresser utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_shader_exporter", "GPU shader export utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_sequencer_instruction_cache", "GPU sequencer instruction cache utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_shader_interpolator", "GPU shader interpolator utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_scan_converter", "GPU scan converter utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_primitive_assembly", "GPU primitive assembly utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_depth_block", "GPU depth block utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_color_block", "GPU color block utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_gtt", "GPU gtt utilization", monitor.METRIC_UNIT_PERCENT,
		},
	},
}
