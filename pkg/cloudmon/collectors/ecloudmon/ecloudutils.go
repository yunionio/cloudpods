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

package ecloudmon

const (
	PERIOD             = 60
	UNIT_AVERAGE       = "Average"
	DEFAULT_STATISTICS = "Average,Minimum,Maximum"
	UNIT_PERCENT       = "Percent"
	UNIT_BPS           = "bps"
	UNIT_CPS           = "cps"
	UNIT_MBPS          = "Mbps"
	UNIT_BYTEPS        = "Bps"
	UNIT_KBYTEPS       = "KBps"
	UNIT_COUNT         = "count"
	UNIT_MEM           = "byte"

	//ESC监控指标
	INFLUXDB_FIELD_CPU_USAGE       = "vm_cpu.usage_active"
	INFLUXDB_FIELD_MEM_USAGE       = "vm_mem.used_percent"
	INFLUXDB_FIELD_DISK_READ_BPS   = "vm_diskio.read_bps"
	INFLUXDB_FIELD_DISK_WRITE_BPS  = "vm_diskio.write_bps"
	INFLUXDB_FIELD_DISK_READ_IOPS  = "vm_diskio.read_iops"
	INFLUXDB_FIELD_DISK_WRITE_IOPS = "vm_diskio.write_iops"
	INFLUXDB_FIELD_NET_BPS_RX      = "vm_netio.bps_recv"
	INFLUXDB_FIELD_NET_BPS_TX      = "vm_netio.bps_sent"

	KEY_LIMIT = "limit"
	KEY_ADMIN = "admin"
)

var ecloudProTypeMetric = map[string]string{}

//multiCloud查询指标列表组装
var ecloudMetricSpecs = map[string][]string{
	"cpu_util":                        {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_CPU_USAGE},
	"memory.util":                     {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_MEM_USAGE},
	"disk.device.read.bytes.rate":     {DEFAULT_STATISTICS, UNIT_KBYTEPS, INFLUXDB_FIELD_DISK_READ_BPS},
	"disk.device.write.bytes.rate":    {DEFAULT_STATISTICS, UNIT_KBYTEPS, INFLUXDB_FIELD_DISK_WRITE_BPS},
	"disk.device.read.requests.rate":  {DEFAULT_STATISTICS, UNIT_CPS, INFLUXDB_FIELD_DISK_READ_IOPS},
	"disk.device.write.requests.rate": {DEFAULT_STATISTICS, UNIT_CPS, INFLUXDB_FIELD_DISK_WRITE_IOPS},
	"network.incoming.bytes":          {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_NET_BPS_RX},
	"network.outgoing.bytes":          {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_NET_BPS_TX},
}
