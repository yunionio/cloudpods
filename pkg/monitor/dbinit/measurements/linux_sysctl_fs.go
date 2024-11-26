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

var linuxSysctlFs = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "linux_sysctl_fs",
			DisplayName:  "Provides Linux sysctl fs metrics",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "file_nr",
			DisplayName: "The number of allocated file handles",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "file_max",
			DisplayName: "The maximum number of file handles that the Linux kernel will allocate.",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "inode_nr",
			DisplayName: "The number of inodes the system has allocated",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "inode_free_nr",
			DisplayName: "The number of free inodes",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "aio_nr",
			DisplayName: "The running total of the number of events specified on the io_setup system call for all currently active aio contexts",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "aio_max_nr",
			DisplayName: "The running total of the number of events specified on the io_setup system call for all currently active aio contexts",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_open",
			DisplayName: "the maximum number of file-handles a process can allocate",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
	},
}
