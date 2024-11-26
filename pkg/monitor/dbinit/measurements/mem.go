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

var mem = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "mem",
			DisplayName:  "Memory",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
		{
			Name:         "agent_mem",
			DisplayName:  "Memory",
			ResourceType: monitor.METRIC_RES_TYPE_AGENT,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "used_percent",
			DisplayName: "Used memory rate",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "available_percent",
			DisplayName: "Available memory rate",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "free_percent",
			DisplayName: "Free memory rate",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "used",
			DisplayName: "Used memory",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "free",
			DisplayName: "Free memory",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "active",
			DisplayName: "The amount of active memory",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "inactive",
			DisplayName: "The amount of inactive memory",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "cached",
			DisplayName: "Cache memory",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "buffered",
			DisplayName: "Buffer memory",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "slab",
			DisplayName: "Number of kernel caches",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "available",
			DisplayName: "Available memory",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "total",
			DisplayName: "Total memory",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
	},
}
