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

var hysmi = SMeasurement{
	Context: []SMonitorContext{
		{
			"hysmi", "Hygon DCU metrics",
			monitor.METRIC_RES_TYPE_HOST, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"temperature_gpu", "DCU temperature", "",
		},
		{
			"power_draw", "DCU power draw", "",
		},
		{
			"power_cap", "DCU power cap", "",
		},
		{
			"utilization_gpu", "DCU utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_memory", "DCU memory utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_encoder", "DCU encoder utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_decoder", "DCU decoder utilization", monitor.METRIC_UNIT_PERCENT,
		},
	},
}
