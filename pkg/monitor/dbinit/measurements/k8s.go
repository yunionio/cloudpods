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

var k8sPod = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "k8s_pod",
			DisplayName:  "k8s pod",
			ResourceType: monitor.METRIC_RES_TYPE_K8S,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "cpu_used_percent",
			DisplayName: "CPU active state utilization rate",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "mem_used_percent",
			DisplayName: "Used memory rate",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "restart_total",
			DisplayName: "pod restart count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
	},
}

var k8sDeploy = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "k8s_deploy",
			DisplayName:  "k8s deploy",
			ResourceType: monitor.METRIC_RES_TYPE_K8S,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "pod_oom_total",
			DisplayName: "oom pod count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pod_restarting_total",
			DisplayName: "restarting pod count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
	},
}

var k8sNode = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "k8s_node",
			DisplayName:  "k8s node",
			ResourceType: monitor.METRIC_RES_TYPE_K8S,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"cpu_used_percent", "CPU active state utilization rate", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"mem_used_percent", "Used memory rate", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"disk_used_percent", "Percentage of used disks", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"bps_sent", "Send traffic per second", monitor.METRIC_UNIT_BPS,
		},
		{
			"bps_recv", "Received traffic per second", monitor.METRIC_UNIT_BPS,
		},
		{
			"pod_restart_total", "pod restart count in node", monitor.METRIC_UNIT_COUNT,
		},
	},
}
