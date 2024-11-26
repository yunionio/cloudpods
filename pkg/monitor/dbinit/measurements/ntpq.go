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

var ntpq = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "ntpq",
			DisplayName:  "standard NTP query metrics",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "delay",
			DisplayName: "(float, milliseconds)",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "jitter",
			DisplayName: "(float, milliseconds)",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "offset",
			DisplayName: "(float, milliseconds)",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "poll",
			DisplayName: "(int, seconds)",
			Unit:        monitor.METRIC_UNIT_SEC,
		},
		{
			Name:        "reach",
			DisplayName: "(int)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "when",
			DisplayName: "(int, seconds)",
			Unit:        monitor.METRIC_UNIT_SEC,
		},
	},
}
