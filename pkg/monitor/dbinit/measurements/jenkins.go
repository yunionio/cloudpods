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

var jenkinsNode = SMeasurement{
	Context: []SMonitorContext{
		{
			"jenkins_node", "jenkins node",
			monitor.METRIC_RES_TYPE_JENKINS, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"disk_available", "disk_available", monitor.METRIC_UNIT_BYTE,
		},
		{
			"temp_available", "temp_available", monitor.METRIC_UNIT_BYTE,
		},
		{
			"memory_available", "memory_available", monitor.METRIC_UNIT_BYTE,
		},
		{
			"memory_total", "memory_total", monitor.METRIC_UNIT_BYTE,
		},
		{
			"swap_available", "swap_available", monitor.METRIC_UNIT_BYTE,
		},
		{
			"swap_total", "swap_total", monitor.METRIC_UNIT_BYTE,
		},
	},
}

var jenkinsJob = SMeasurement{
	Context: []SMonitorContext{
		{
			"jenkins_job", "jenkins job",
			monitor.METRIC_RES_TYPE_JENKINS, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"duration", "duration", monitor.METRIC_UNIT_MS,
		},
		{
			"number", "number", monitor.METRIC_UNIT_COUNT,
		},
	},
}
