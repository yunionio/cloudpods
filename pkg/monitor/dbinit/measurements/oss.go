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

var ossLatency = SMeasurement{
	Context: []SMonitorContext{
		{
			"oss_latency", "Object storage latency",
			monitor.METRIC_RES_TYPE_OSS, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"req_late", "Request average E2E delay", monitor.METRIC_UNIT_MS,
		},
	},
}

var ossNetio = SMeasurement{
	Context: []SMonitorContext{
		{
			"oss_netio", "Object storage network traffic",
			monitor.METRIC_RES_TYPE_OSS, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"bps_recv", "Receive byte", monitor.METRIC_UNIT_BYTE,
		},
		{
			"bps_sent", "Send byte", monitor.METRIC_UNIT_BYTE,
		},
	},
}

var ossReq = SMeasurement{
	Context: []SMonitorContext{
		{
			"oss_req", "Object store request", monitor.METRIC_RES_TYPE_OSS,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"req_count", "request count", monitor.METRIC_UNIT_COUNT,
		},
	},
}
