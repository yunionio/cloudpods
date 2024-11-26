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

var rabbitmqOverview = SMeasurement{
	Context: []SMonitorContext{
		{
			"rabbitmq_overview", "rabbitmq_overview",
			monitor.METRIC_RES_TYPE_EXT_RABBITMQ, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"channels", "channels", monitor.METRIC_UNIT_COUNT,
		},
		{
			"consumers", "consumers", monitor.METRIC_UNIT_COUNT,
		},
		{
			"messages", "messages", monitor.METRIC_UNIT_COUNT,
		},
		{
			"queues", "queues", monitor.METRIC_UNIT_COUNT,
		},
	},
}

var rabbitmqNode = SMeasurement{
	Context: []SMonitorContext{
		{
			"rabbitmq_node", "rabbitmq_node",
			monitor.METRIC_RES_TYPE_EXT_RABBITMQ, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"disk_free", "disk_free", monitor.METRIC_UNIT_BYTE,
		},
		{
			"mem_used", "mem_used", monitor.METRIC_UNIT_BYTE,
		},
	},
}

var rabbitmqQueue = SMeasurement{
	Context: []SMonitorContext{
		{
			"rabbitmq_queue", "rabbitmq_queue",
			monitor.METRIC_RES_TYPE_EXT_RABBITMQ, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			"consumer_utilisation", "consumer_utilisation", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"message_bytes", "message_bytes", monitor.METRIC_UNIT_BYTE,
		},
		{
			"message_bytes_ram", "message_bytes_ram", monitor.METRIC_UNIT_BYTE,
		},
		{
			"messages", "messages", monitor.METRIC_UNIT_COUNT,
		},
	},
}
