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

var redis = SMeasurement{
	Context: []SMonitorContext{
		{
			"redis", "redis",
			monitor.METRIC_RES_TYPE_EXT_REDIS, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"used_memory", "used_memory", monitor.METRIC_UNIT_BYTE,
		},
		{
			"used_memory_peak", "used_memory_peak", monitor.METRIC_UNIT_BYTE,
		},
		{
			"used_cpu_sys", "used_cpu_sys", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"used_cpu_user", "used_cpu_user", monitor.METRIC_UNIT_PERCENT,
		},
	},
}

var redisKeyspace = SMeasurement{
	Context: []SMonitorContext{
		{
			"redis_keyspace", "redis_keyspace",
			monitor.METRIC_RES_TYPE_EXT_REDIS, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"keys", "keys", monitor.METRIC_UNIT_COUNT,
		},
		{
			"expires", "expires", monitor.METRIC_UNIT_COUNT,
		},
	},
}
