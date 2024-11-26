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

var rdsConn = SMeasurement{
	Context: []SMonitorContext{
		{
			"rds_conn", "Rds connection", monitor.METRIC_RES_TYPE_RDS,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"used_percent", "Connection usage", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"active_count", "active connection count", monitor.METRIC_UNIT_COUNT,
		},
		{
			"failed_count", "failed connection count", monitor.METRIC_UNIT_COUNT,
		},
	},
}

var rdsCpu = SMeasurement{
	Context: []SMonitorContext{
		{
			"rds_cpu", "Rds CPU usage", monitor.METRIC_RES_TYPE_RDS,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"usage_active", "CPU active state utilization rate", monitor.METRIC_UNIT_PERCENT,
		},
	},
}

var rdsMem = SMeasurement{
	Context: []SMonitorContext{
		{
			"rds_mem", "Rds memory", monitor.METRIC_RES_TYPE_RDS,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"used_percent", "Used memory rate", monitor.METRIC_UNIT_PERCENT,
		},
	},
}

var rdsNetio = SMeasurement{
	Context: []SMonitorContext{
		{
			"rds_netio", "Rds network traffic", monitor.METRIC_RES_TYPE_RDS,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"bps_recv", "Received traffic per second", monitor.METRIC_UNIT_BPS,
		},
		{
			"bps_sent", "Send traffic per second", monitor.METRIC_UNIT_BPS,
		},
	},
}

var rdsDisk = SMeasurement{
	Context: []SMonitorContext{
		{
			"rds_disk", "Rds disk usage", monitor.METRIC_RES_TYPE_RDS,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"used_percent", "Percentage of used disks", monitor.METRIC_UNIT_PERCENT,
		},
	},
}
