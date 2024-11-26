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

var disk = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "disk",
			DisplayName:  "Disk usage",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
		{
			Name:         "agent_disk",
			DisplayName:  "Disk usage",
			ResourceType: monitor.METRIC_RES_TYPE_AGENT,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "used_percent",
			DisplayName: "Percentage of used disks",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "free",
			DisplayName: "Free space size",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "used",
			DisplayName: "Used disk size",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "total",
			DisplayName: "Total disk size",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "inodes_used_percent",
			DisplayName: "Percentage of used inodes",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "inodes_free",
			DisplayName: "Available inode",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "inodes_used",
			DisplayName: "Number of inodes used",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "inodes_total",
			DisplayName: "Total inodes",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "read_only",
			DisplayName: "Test if disk is read only",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
	},
}
