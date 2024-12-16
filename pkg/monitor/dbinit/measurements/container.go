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

var containerCpu = SMeasurement{
	Context: []SMonitorContext{
		{
			"container_cpu", "Container cpu", monitor.METRIC_RES_TYPE_CONTAINER,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"usage_rate", "Container cpu usage rate", monitor.METRIC_UNIT_PERCENT,
		},
	},
}

var containerMem = SMeasurement{
	Context: []SMonitorContext{
		{
			"container_mem", "Container memory", monitor.METRIC_RES_TYPE_CONTAINER, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"usage_rate", "Container memory usage rate", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"working_set_bytes", "Container memory working set bytes", monitor.METRIC_UNIT_BYTE,
		},
	},
}

var containerProcess = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "container_process",
			DisplayName:  "Container process",
			ResourceType: monitor.METRIC_RES_TYPE_CONTAINER,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: newCadvisorProcessMetrics("Container"),
}

var containerDiskIo = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "container_diskio",
			DisplayName:  "Container diskio",
			ResourceType: monitor.METRIC_RES_TYPE_CONTAINER,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: newCadvisorDiskIoMetrics("Container"),
}
