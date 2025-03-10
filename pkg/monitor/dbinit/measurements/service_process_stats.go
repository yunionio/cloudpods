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

var serviceProcessStats = SMeasurement{
	Context: []SMonitorContext{
		{
			"process", "Service process stats",
			monitor.METRIC_RES_TYPE_PROCESS, monitor.METRIC_DATABASE_SYSTEM,
		},
	},
	Metrics: []SMetric{
		{
			"cpu_percent", "CPU percent", monitor.METRIC_UNIT_NULL,
		},
		{
			"mem_percent", "Memory percent", monitor.METRIC_UNIT_NULL,
		},
		{
			"mem_size", "Memory size", monitor.METRIC_UNIT_NULL,
		},
		{
			"goroutine_num", "Goroutine num", monitor.METRIC_UNIT_NULL,
		},
	},
}
