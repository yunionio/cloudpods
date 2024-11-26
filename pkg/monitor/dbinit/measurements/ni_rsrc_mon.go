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

var netint = SMeasurement{
	Context: []SMonitorContext{
		{
			"ni_rsrc_mon", "NETINT device metrics",
			monitor.METRIC_RES_TYPE_HOST, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"load", "Load utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"model_load", "Model load utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"fw_load", "FW load utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"inst", "INST utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"max_inst", "MAX INST utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"mem", "Memory utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"critical_mem", "Critical memory utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"share_mem", "Share memory utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"p2p_mem", "P2P memory utilization", monitor.METRIC_UNIT_PERCENT,
		},
	},
}
