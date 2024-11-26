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

var ping = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "ping",
			DisplayName:  "Ping monitor",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"packets_transmitted", "used SNAT port count", monitor.METRIC_UNIT_COUNT,
		},
		{
			"packets_received", "SNAT connection count", monitor.METRIC_UNIT_COUNT,
		},
		{
			"percent_packet_loss", "Packet loss rate in percetile", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"ttl", "TTL count", monitor.METRIC_UNIT_COUNT,
		},
		{
			"average_response_ms", "average_response_ms", monitor.METRIC_UNIT_MS,
		},
		{
			"minimum_response_ms", "minimum response ms", monitor.METRIC_UNIT_MS,
		},
		{
			"maximum_response_ms", "maximum_response_ms", monitor.METRIC_UNIT_MS,
		},
		{
			"standard_deviation_ms", "standard_deviation_ms", monitor.METRIC_UNIT_MS,
		},
		{
			"percentile50_ms", "SNAT connection count", monitor.METRIC_UNIT_MS,
		},
		{
			"errors", "SNAT connection count", monitor.METRIC_UNIT_COUNT,
		},
		/*{
			"reply_received", "SNAT connection count", monitor.METRIC_UNIT_COUNT,
		},
		{
			"percent_reply_loss", "SNAT connection count", monitor.METRIC_UNIT_COUNT,
		},*/
		{
			"result_code", "Ping result code, success = 0, no such host = 1, ping error = 2", monitor.METRIC_UNIT_NULL,
		},
	},
}
