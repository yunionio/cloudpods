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

var vmCpu = SMeasurement{
	Context: []SMonitorContext{
		{
			"vm_cpu", "Guest CPU usage", monitor.METRIC_RES_TYPE_GUEST,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"usage_active", "CPU active state utilization rate", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"cpu_usage_pcore", "CPU utilization rate per core", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"cpu_usage_idle_pcore", "CPU idle rate per core", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"cpu_time_system", "CPU system state time", monitor.METRIC_UNIT_MS,
		},
		{
			"cpu_time_user", "CPU user state time", monitor.METRIC_UNIT_MS,
		},
		{
			"thread_count", "The number of threads used by the process", monitor.METRIC_UNIT_COUNT,
		},
	},
}

var vmMem = SMeasurement{
	Context: []SMonitorContext{
		{
			"vm_mem", "Guest memory", monitor.METRIC_RES_TYPE_GUEST,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"used_percent", "Used memory rate", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"vms", "Virtual memory consumption", monitor.METRIC_UNIT_BYTE,
		},
		{
			"rss", "Actual use of physical memory", monitor.METRIC_UNIT_BYTE,
		},
	},
}

var vmDiskio = SMeasurement{
	Context: []SMonitorContext{
		{
			"vm_diskio", "Guest disk traffic", monitor.METRIC_RES_TYPE_GUEST,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"read_bps", "Disk read rate", monitor.METRIC_UNIT_BYTEPS,
		},
		{
			"write_bps", "Disk write rate", monitor.METRIC_UNIT_BYTEPS,
		},
		{
			"read_iops", "Disk read operate rate", monitor.METRIC_UNIT_COUNT,
		},
		{
			"write_iops", "Disk write operate rate", monitor.METRIC_UNIT_COUNT,
		},
		{
			"read_bytes", "Bytes read", monitor.METRIC_UNIT_BYTE,
		},
		{
			"write_bytes", "Bytes write", monitor.METRIC_UNIT_BYTE,
		},
	},
}

var vmDisk = SMeasurement{
	Context: []SMonitorContext{
		{
			"vm_disk", "Guest disk", monitor.METRIC_RES_TYPE_GUEST,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"used_percent", "Used vm disk rate", monitor.METRIC_UNIT_PERCENT,
		},
	},
}

var vmNetio = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "vm_netio",
			DisplayName:  "Guest network traffic",
			ResourceType: monitor.METRIC_RES_TYPE_GUEST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
		{
			Name:         "pod_netio",
			DisplayName:  "Pod network traffic",
			ResourceType: monitor.METRIC_RES_TYPE_CONTAINER,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "bps_recv",
			DisplayName: "Received traffic per second",
			Unit:        monitor.METRIC_UNIT_BPS,
		},
		{
			Name:        "bps_sent",
			DisplayName: "Send traffic per second",
			Unit:        monitor.METRIC_UNIT_BPS,
		},
		{
			Name:        "pps_recv",
			DisplayName: "Received packets per second",
			Unit:        monitor.METRIC_UNIT_PPS,
		},
		{
			Name:        "pps_sent",
			DisplayName: "Send packets per second",
			Unit:        monitor.METRIC_UNIT_PPS,
		},
	},
}
