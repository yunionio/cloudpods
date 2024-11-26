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

var diskio = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "diskio",
			DisplayName:  "Disk traffic and timing",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
		{
			Name:         "agent_diskio",
			DisplayName:  "Disk traffic and timing",
			ResourceType: monitor.METRIC_RES_TYPE_AGENT,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "bps",
			DisplayName: "Disk read and write rate",
			Unit:        monitor.METRIC_UNIT_BYTEPS,
		},
		{
			Name:        "read_bps",
			DisplayName: "Disk read rate",
			Unit:        monitor.METRIC_UNIT_BYTEPS,
		},
		{
			Name:        "write_bps",
			DisplayName: "Disk write rate",
			Unit:        monitor.METRIC_UNIT_BYTEPS,
		},
		{
			Name:        "read_iops",
			DisplayName: "Disk read operate rate",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "write_iops",
			DisplayName: "Disk write operate rate",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "iops",
			DisplayName: "Disk read/write operate rate",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "reads",
			DisplayName: "Number of reads",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "writes",
			DisplayName: "Number of writes",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "iocount",
			DisplayName: "Accumulated I/O request count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "read_bytes",
			DisplayName: "Bytes read",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "write_bytes",
			DisplayName: "Bytes write",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "iobytes",
			DisplayName: "Bytes of both read and write",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "read_time",
			DisplayName: "Time to wait for read",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "write_time",
			DisplayName: "Time to wait for write",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "io_time",
			DisplayName: "I / O request queuing time",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "read_await",
			DisplayName: "Time to wait for read",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "write_await",
			DisplayName: "Time to wait for write",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "await",
			DisplayName: "I / O request queuing time",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "merged_reads",
			DisplayName: "Merged disk read request count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "merged_writes",
			DisplayName: "Merged disk write request count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "merged_iocount",
			DisplayName: "Merged Accumulated I/O request count",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "weighted_io_time",
			DisplayName: "I / O request waiting time",
			Unit:        monitor.METRIC_UNIT_MS,
		},
		{
			Name:        "iops_in_progress",
			DisplayName: "Number of I / O requests issued but not yet completed",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "ioutil",
			DisplayName: "IO utilization in percentage",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "avgqu_sz",
			DisplayName: "Averate queing request count per seconds",
			Unit:        monitor.METRIC_UNIT_NULL,
		},
	},
}
