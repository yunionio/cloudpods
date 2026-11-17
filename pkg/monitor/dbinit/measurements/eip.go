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

var eipNet = SMeasurement{
	Context: []SMonitorContext{
		{
			"eip_net", "EIP network traffic", monitor.METRIC_RES_TYPE_EIP,
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
		{
			"drop_speed_rx", "Outbound rate-limit drop speed", monitor.METRIC_UNIT_BPS,
		},
	},
}
