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

var dcsRedisCpu = SMeasurement{
	Context: []SMonitorContext{
		{
			"dcs_cpu", "Redis CPU usage", monitor.METRIC_RES_TYPE_REDIS,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"usage_percent", "CPU active state utilization rate", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"server_load", "server load", monitor.METRIC_UNIT_PERCENT,
		},
	},
}

var dcsRedisMem = SMeasurement{
	Context: []SMonitorContext{
		{
			"dcs_mem", "Redis memory", monitor.METRIC_RES_TYPE_REDIS,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"used_percent", "Used memory rate", monitor.METRIC_UNIT_PERCENT,
		},
	},
}

var dcsRedisNetio = SMeasurement{
	Context: []SMonitorContext{
		{
			"dcs_netio", "Redis network traffic",
			monitor.METRIC_RES_TYPE_REDIS,
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

var dcsRedisConn = SMeasurement{
	Context: []SMonitorContext{
		{
			"dcs_conn", "Redis connect", monitor.METRIC_RES_TYPE_REDIS,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"used_percent", "Connection usage", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"errors", "Connection errors", monitor.METRIC_UNIT_COUNT,
		},
	},
}

var dcsRedisInstanceOpt = SMeasurement{
	Context: []SMonitorContext{
		{
			"dcs_instantopt", "Redis operator",
			monitor.METRIC_RES_TYPE_REDIS, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"opt_sec", "Number of commands processed per second", monitor.METRIC_UNIT_COUNT,
		},
	},
}

var dcsRedisCachekeys = SMeasurement{
	Context: []SMonitorContext{
		{
			"dcs_cachekeys", "Redis keys", monitor.METRIC_RES_TYPE_REDIS,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"key_count", "Number of cache keys", monitor.METRIC_UNIT_COUNT,
		},
	},
}

var dcsRedisDatamem = SMeasurement{
	Context: []SMonitorContext{
		{
			"dcs_datamem", "Redis data memory", monitor.METRIC_RES_TYPE_REDIS,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"used_byte", "Data node memory usage", monitor.METRIC_UNIT_BYTE,
		},
	},
}
