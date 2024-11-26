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

var net = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "net",
			DisplayName:  "Network interface and protocol usage",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
		{
			Name:         "agent_net",
			DisplayName:  "Network interface and protocol usage",
			ResourceType: monitor.METRIC_RES_TYPE_AGENT,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "bps_sent",
			DisplayName: "Send traffic per second",
			Unit:        monitor.METRIC_UNIT_BPS,
		},
		{
			Name:        "bps_recv",
			DisplayName: "Received traffic per second",
			Unit:        monitor.METRIC_UNIT_BPS,
		},
		{
			Name:        "pps_recv",
			DisplayName: "Received packets per second",
			Unit:        monitor.METRIC_UNIT_PPS,
		},
		{
			Name:        "pps_sent",
			DisplayName: "Send packets per second",
			Unit:        monitor.METRIC_UNIT_PPS,
		},
		{
			Name:        "bytes_sent",
			DisplayName: "The total number of bytes sent by the network interface",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "bytes_recv",
			DisplayName: "The total number of bytes received by the network interface",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "packets_sent",
			DisplayName: "The total number of packets sent by the network interface",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "packets_recv",
			DisplayName: "The total number of packets received by the network interface",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "err_in",
			DisplayName: "The total number of receive errors detected by the network interface",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "err_out",
			DisplayName: "The total number of transmission errors detected by the network interface",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "drop_in",
			DisplayName: "The total number of received packets dropped by the network interface",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "drop_out",
			DisplayName: "The total number of transmission packets dropped by the network interface",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
	},
}
