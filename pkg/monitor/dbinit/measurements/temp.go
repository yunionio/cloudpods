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

var temp = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "temp",
			DisplayName:  "The temp input plugin gather metrics on system temperature",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
		{
			Name:         "agent_temp",
			DisplayName:  "The temp input plugin gather metrics on system temperature",
			ResourceType: monitor.METRIC_RES_TYPE_GUEST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "temp",
			DisplayName: "(float, celcius)",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
	},
}
