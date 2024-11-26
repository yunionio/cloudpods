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

var system = SMeasurement{
	Context: []SMonitorContext{
		{
			"system", "System load",
			monitor.METRIC_RES_TYPE_HOST, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"load1", "Loadavg load1", monitor.METRIC_UNIT_NULL,
		},
		{
			"load5", "Loadavg load5", monitor.METRIC_UNIT_NULL,
		},
		{
			"load15", "Loadavg load15", monitor.METRIC_UNIT_NULL,
		},
		{
			"load1_pcore", "Loadavg load1 per cpu core", monitor.METRIC_UNIT_NULL,
		},
		{
			"load5_pcore", "Loadavg load5 per cpu core", monitor.METRIC_UNIT_NULL,
		},
		{
			"load15_pcore", "Loadavg load15 per cpu core", monitor.METRIC_UNIT_NULL,
		},
	},
}
