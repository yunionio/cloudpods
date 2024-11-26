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

var haproxy = SMeasurement{
	Context: []SMonitorContext{
		{
			"haproxy", "load balance", monitor.METRIC_RES_TYPE_ELB,
			monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"used_snat_port", "used SNAT port count", monitor.METRIC_UNIT_COUNT,
		},
		{
			"snat_conn_count", "SNAT connection count", monitor.METRIC_UNIT_COUNT,
		},
	},
}
