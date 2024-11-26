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

var swap = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "swap",
			DisplayName:  "The swap plugin collects system swap metrics.",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "free",
			DisplayName: "(int, bytes): free swap memory",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "total",
			DisplayName: "(int, bytes): total swap memory",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "used",
			DisplayName: "(int, bytes): used swap memory",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "used_percent",
			DisplayName: "(float, percent): percentage of swap memory used",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "in",
			DisplayName: "(int, bytes): data swapped in since last boot calculated from page number",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "out",
			DisplayName: "(int, bytes): data swapped out since last boot calculated from page number",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
	},
}
