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

var processes = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "processes",
			DisplayName:  "Processes status",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
		{
			Name:         "agent_processes",
			DisplayName:  "Processes status in guest",
			ResourceType: monitor.METRIC_RES_TYPE_AGENT,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "total",
			DisplayName: "Total processes count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "total_threads",
			DisplayName: "Total threads count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "zombies",
			DisplayName: "Z state, zombie processes count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "running",
			DisplayName: "R state, running processes count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "sleeping",
			DisplayName: "S state, sleeping processes count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "blocked",
			DisplayName: "D, waiting in uninterruptible sleep, or locked, aka disk sleep",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "stopped",
			DisplayName: "T state, stopped process count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "dead",
			DisplayName: "X state, dead",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "wait",
			DisplayName: "W state, (freebsd only)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "idle",
			DisplayName: "I state, (sleeping for longer than about 20 seconds, bsd and Linux 4+ only)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "paging",
			DisplayName: "W state (linux kernel < 2.6 only)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "parked",
			DisplayName: "(linux only)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
	},
}

/*
Linux  FreeBSD  Darwin  meaning
R       R       R     running
S       S       S     sleeping
Z       Z       Z     zombie
X      none    none   dead
T       T       T     stopped
I       I       I     idle (sleeping for longer than about 20 seconds)
D      D,L      U     blocked (waiting in uninterruptible sleep, or locked)
W       W      none   paging (linux kernel < 2.6 only), wait (freebsd)
*/
