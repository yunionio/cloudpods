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

var netstat = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "netstat",
			DisplayName:  "netstat",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "tcp_established",
			DisplayName: "",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "tcp_syn_sent",
			DisplayName: "",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "tcp_syn_recv",
			DisplayName: "",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "tcp_fin_wait1",
			DisplayName: "",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "tcp_fin_wait2",
			DisplayName: "",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "tcp_time_wait",
			DisplayName: "",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "tcp_close",
			DisplayName: "",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "tcp_close_wait",
			DisplayName: "",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "tcp_last_ack",
			DisplayName: "",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "tcp_listen",
			DisplayName: "",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "tcp_closing",
			DisplayName: "",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "tcp_none",
			DisplayName: "",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "udp_socket",
			DisplayName: "",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
	},
}
