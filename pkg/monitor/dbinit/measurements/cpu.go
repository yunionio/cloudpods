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

var cpu = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "cpu",
			DisplayName:  "CPU usage",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
		{
			Name:         "agent_cpu",
			DisplayName:  "CPU usage",
			ResourceType: monitor.METRIC_RES_TYPE_AGENT,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "usage_active",
			DisplayName: "CPU active state utilization rate",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "usage_idle",
			DisplayName: "CPU idle state utilization rate",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "usage_system",
			DisplayName: "CPU system state utilization rate",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "usage_user",
			DisplayName: "CPU user mode utilization rate",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "usage_iowait",
			DisplayName: "CPU IO usage",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "usage_irq",
			DisplayName: "CPU IRQ usage",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "usage_guest",
			DisplayName: "CPU guest usage",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "usage_nice",
			DisplayName: "CPU priority switch utilization",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "usage_softirq",
			DisplayName: "CPU softirq usage",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "usage_steal",
			DisplayName: "CPU steal usage",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "usage_guest_nice",
			DisplayName: "CPU guest nice usage",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
	},
}
