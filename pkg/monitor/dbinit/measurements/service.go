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

var worker = SMeasurement{
	Context: []SMonitorContext{
		{
			"worker", "Worker queue",
			monitor.METRIC_RES_TYPE_SYSTEM, monitor.METRIC_DATABASE_SYSTEM,
		},
	},
	Metrics: []SMetric{
		{
			"active_worker_cnt", "Active Worker Count", monitor.METRIC_UNIT_NULL,
		},
		{
			"max_worker_count", "Max Worker Count", monitor.METRIC_UNIT_NULL,
		},
		{
			"detach_worker_cnt", "Detach worker Count", monitor.METRIC_UNIT_NULL,
		},
		{
			"queue_cnt", "Worker Queue Count", monitor.METRIC_UNIT_NULL,
		},
	},
}

var statusProbe = SMeasurement{
	Context: []SMonitorContext{
		{
			"status_probe", "Resource status probe results", monitor.METRIC_RES_TYPE_SYSTEM, monitor.METRIC_DATABASE_SYSTEM,
		},
	},
	Metrics: []SMetric{
		{
			"count", "Resouce count for each status", monitor.METRIC_UNIT_NULL,
		},
		{
			"pending_deleted", "Pending deleted resource count for each status", monitor.METRIC_UNIT_NULL,
		},
	},
}
