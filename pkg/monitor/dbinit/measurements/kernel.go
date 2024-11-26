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

var kernel = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "kernel",
			DisplayName:  "kernel metrics",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "boot_time",
			DisplayName: "seconds since epoch, btime",
			Unit:        monitor.METRIC_UNIT_SEC,
		},
		{
			Name:        "context_switches",
			DisplayName: "context switch count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "disk_pages_in",
			DisplayName: "page in count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "disk_pages_out",
			DisplayName: "page out count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "interrupts",
			DisplayName: "interrupts count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "processes_forked",
			DisplayName: "processes forked count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "entropy_avail",
			DisplayName: "entropy available",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
	},
}
