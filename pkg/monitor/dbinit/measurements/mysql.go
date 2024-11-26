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

var mysql = SMeasurement{
	Context: []SMonitorContext{
		{
			"mysql", "mysql",
			monitor.METRIC_RES_TYPE_EXT_MYSQL, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"binary_size_bytes", "binary_size_bytes", monitor.METRIC_UNIT_BYTE,
		},
		{
			"binary_files_count", "binary_files_count", monitor.METRIC_UNIT_COUNT,
		},
		{
			"connections", "connections", monitor.METRIC_UNIT_COUNT,
		},
		{
			"table_io_waits_total_fetch", "table_io_waits_total_fetch", monitor.METRIC_UNIT_COUNT,
		},
		{
			"table_io_waits_seconds_total_fetch", "table_io_waits_seconds_total_fetch", monitor.METRIC_UNIT_MS,
		},
		{
			"index_io_waits_total_fetch", "index_io_waits_total_fetch", monitor.METRIC_UNIT_COUNT,
		},
		{
			"index_io_waits_seconds_total_fetch", "index_io_waits_seconds_total_fetch", monitor.METRIC_UNIT_MS,
		},
		{
			"info_schema_table_rows", "info_schema_table_rows", monitor.METRIC_UNIT_COUNT,
		},
		{
			"info_schema_table_size_data_length", "info_schema_table_size_data_length", monitor.METRIC_UNIT_COUNT,
		},
		{
			"info_schema_table_size_index_length", "info_schema_table_size_index_length", monitor.METRIC_UNIT_COUNT,
		},
	},
}
