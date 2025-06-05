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

var serviceHttpCode = SMeasurement{
	Context: []SMonitorContext{
		{
			"http_request", "HTTP Request hit",
			monitor.METRIC_RES_TYPE_SYSTEM, monitor.METRIC_DATABASE_SYSTEM,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "duration_ms_any",
			DisplayName: "Accumulated request duration in milliseconds",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "dura_ms_delta_any",
			DisplayName: "Accumulated request duration in milliseconds in last interval",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "hit_any",
			DisplayName: "Accumulated request count",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
		{
			Name:        "hit_delta_any",
			DisplayName: "Accumulated request count in last interval",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
		{
			Name:        "delay_ms_any",
			DisplayName: "Average request delay in miilliseconds",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "qps_any",
			DisplayName: "Averatge request per second",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
		{
			Name:        "duration_ms_2xx",
			DisplayName: "Accumulated request duration in milliseconds for 2xx http code",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "dura_ms_delta_2xx",
			DisplayName: "Accumulated request duration in milliseconds in last interval for 2xx http code",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "hit_2xx",
			DisplayName: "Accumulated request count for 2xx http code",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
		{
			Name:        "hit_delta_2xx",
			DisplayName: "Accumulated request count in last interval for 2xx http code",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
		{
			Name:        "delay_ms_2xx",
			DisplayName: "Average request delay in miilliseconds for 2xx http code",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "percent_hit_2xx",
			DisplayName: "Request hit weight in percentage for 2xx http code",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "percent_duration_2xx",
			DisplayName: "Request duration weight in percentage for 2xx http code",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "qps_2xx",
			DisplayName: "Averatge request per second for 2xx http code",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
		{
			Name:        "duration_ms_4xx",
			DisplayName: "Accumulated request duration in milliseconds for 4xx http code",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "dura_ms_delta_4xx",
			DisplayName: "Accumulated request duration in milliseconds in last interval for 4xx http code",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "hit_4xx",
			DisplayName: "Accumulated request count for 4xx http code",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
		{
			Name:        "hit_delta_4xx",
			DisplayName: "Accumulated request count in last interval for 4xx http code",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
		{
			Name:        "delay_ms_4xx",
			DisplayName: "Average request delay in miilliseconds for 4xx http code",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "percent_hit_4xx",
			DisplayName: "Request hit weight in percentage for 4xx http code",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "percent_duration_4xx",
			DisplayName: "Request duration weight in percentage for 4xx http code",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "qps_4xx",
			DisplayName: "Averatge request per second for 4xx http code",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
		{
			Name:        "duration_ms_5xx",
			DisplayName: "Accumulated request duration in milliseconds for 5xx http code",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "dura_ms_delta_5xx",
			DisplayName: "Accumulated request duration in milliseconds in last interval for 5xx http code",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "hit_5xx",
			DisplayName: "Accumulated request count for 5xx http code",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
		{
			Name:        "hit_delta_5xx",
			DisplayName: "Accumulated request count in last interval for 5xx http code",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
		{
			Name:        "delay_ms_5xx",
			DisplayName: "Average request delay in miilliseconds for 5xx http code",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "percent_hit_5xx",
			DisplayName: "Request hit weight in percentage for 5xx http code",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "percent_duration_5xx",
			DisplayName: "Request duration weight in percentage for 5xx http code",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "qps_5xx",
			DisplayName: "Averatge request per second for 5xx http code",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
	},
}
