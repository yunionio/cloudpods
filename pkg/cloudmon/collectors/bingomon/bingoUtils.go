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

package bingomon

import "yunion.io/x/onecloud/pkg/cloudmon/collectors/common"

const (
	SERVER_METRIC_NAMESPACE = "AWS/EC2"
	HOST_METRIC_NAMESPACE   = "AWS/HOST"
)

//multiCloud查询指标列表组装
var bingoMetricSpecs = map[string][]string{
	"CPUUtilization": {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_CPU_USAGE},
	"MemeryUsage":    {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_MEM_USAGE},
	"NetworkIn":      {common.DEFAULT_STATISTICS, common.UNIT_BYTEPS, common.INFLUXDB_FIELD_NET_BPS_RX},
	"NetworkOut":     {common.DEFAULT_STATISTICS, common.UNIT_BYTEPS, common.INFLUXDB_FIELD_NET_BPS_TX},
	"DiskReadBytes":  {common.DEFAULT_STATISTICS, common.UNIT_BYTEPS, common.INFLUXDB_FIELD_DISK_READ_BPS},
	"DiskWriteBytes": {common.DEFAULT_STATISTICS, common.UNIT_BYTEPS, common.INFLUXDB_FIELD_DISK_WRITE_BPS},
	"DiskReadOps":    {common.DEFAULT_STATISTICS, common.UNIT_CPS, common.INFLUXDB_FIELD_DISK_READ_IOPS},
	"DiskWriteOps":   {common.DEFAULT_STATISTICS, common.UNIT_CPS, common.INFLUXDB_FIELD_DISK_WRITE_IOPS},
}
