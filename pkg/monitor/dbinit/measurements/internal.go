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

var internalMemstats = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "internal_memstats",
			DisplayName:  "Memory usage statistics of telegraf agent",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "alloc_bytes",
			DisplayName: "alloc_bytes",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name: "frees",
		},
		{
			Name: "heap_alloc_bytes",
		},
		{
			Name: "heap_idle_bytes",
		},
		{
			Name: "heap_in_use_bytes",
		},
		{
			Name: "heap_objects_bytes",
		},
		{
			Name: "heap_released_bytes",
		},
		{
			Name: "heap_sys_bytes",
		},
		{
			Name: "mallocs",
		},
		{
			Name: "num_gc",
		},
		{
			Name: "pointer_lookups",
		},
		{
			Name: "sys_bytes",
		},
		{
			Name: "total_alloc_bytes",
		},
	},
}

var internalAgent = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "internal_agent",
			DisplayName:  "Agent wide statistics of telegraf agent",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name: "gather_errors",
		},
		{
			Name: "metrics_dropped",
		},
		{
			Name: "metrics_gathered",
		},
		{
			Name: "metrics_written",
		},
	},
}

var internalGather = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "internal_gather",
			DisplayName:  "Gather statistics of telegraf agent",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name: "gather_time_ns",
		},
		{
			Name: "metrics_gathered",
		},
	},
}

var internalWrite = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "internal_write",
			DisplayName:  "Write statistics of telegraf agent",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name: "buffer_limit",
		},
		{
			Name: "buffer_size",
		},
		{
			Name: "metrics_added",
		},
		{
			Name: "metrics_written",
		},
		{
			Name: "metrics_dropped",
		},
		{
			Name: "metrics_filtered",
		},
		{
			Name: "write_time_ns",
		},
	},
}
