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

var dbStats = SMeasurement{
	Context: []SMonitorContext{
		{
			"db_stats", "Database Stats",
			monitor.METRIC_RES_TYPE_SYSTEM, monitor.METRIC_DATABASE_SYSTEM,
		},
	},
	Metrics: []SMetric{
		{
			"idle", "Database Idle", monitor.METRIC_UNIT_NULL,
		},
		{
			"in_use", "Database InUse", monitor.METRIC_UNIT_NULL,
		},
		{
			"max_idle_closed", "Database max idle closed", monitor.METRIC_UNIT_NULL,
		},
		{
			"max_idle_time_closed", "Database max idle time closed", monitor.METRIC_UNIT_NULL,
		},
		{
			"max_lifetime_closed", "Database max lifetime closed", monitor.METRIC_UNIT_NULL,
		},
		{
			"max_open_connections", "Database max open connections", monitor.METRIC_UNIT_NULL,
		},
		{
			"open_connections", "Database open connections", monitor.METRIC_UNIT_NULL,
		},
		{
			"wait_count", "Database wait count", monitor.METRIC_UNIT_NULL,
		},
		{
			"wait_duration", "Database wait duration", monitor.METRIC_UNIT_NULL,
		},
	},
}
