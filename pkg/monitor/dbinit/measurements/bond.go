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

var bond = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "bond",
			DisplayName:  "Bond interface",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "active_slave",
			DisplayName: "used SNAT port count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "status",
			DisplayName: "Status of bond interface (down = 0, up = 1)",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
	},
}

var bondSlave = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "bond_slave",
			DisplayName:  "Slave interfaces of bond interface",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "status",
			DisplayName: "Status of bonds's slave interface (down = 0, up = 1)",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
		{
			Name:        "failures",
			DisplayName: "Amount of failures for bond's slave interface",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
	},
}
