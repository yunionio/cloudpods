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

package azuremon

import "yunion.io/x/onecloud/pkg/cloudmon/collectors/common"

// multiCloud查询指标列表组装
const (
	MetricKeyPercentageCPU   = "Percentage CPU"
	MetricKeyNetworkInTotal  = "Network In Total"
	MetricKeyNetworkOutTotal = "Network Out Total"
	MetricKeyDiskReadBytes   = "Disk Read Bytes"
	MetricKeyDiskWriteBytes  = "Disk Write Bytes"
	MetricKeyDiskReadOPS     = "Disk Read Operations/Sec"
	MetricKeyDiskWriteOPS    = "Disk Write Operations/Sec"

	MetricKeyClassicNetworkIn    = "Network In"
	MetricKeyClassicNetworkOut   = "Network Out"
	MetricKeyClassicDiskReadBPS  = "Disk Read Bytes/Sec"
	MetricKeyClassicDiskWriteBPS = "Disk Write Bytes/Sec"

	SERVER_METRIC_NAMESPACE   = "Microsoft.Compute/virtualMachines"
	REDIS_METRIC_NAMESPACE    = "Microsoft.Cache/redis"
	ELB_METRIC_NAMESPACE      = "Microsoft.Network/loadBalancers"
	K8S_NODE_METRIC_NAMESPACE = "Microsoft.ContainerService/managedClusters"

	// pricing metric
	K8S_POD_METRIC_NAMESPACE = "insights.container/pods"
)

var azureMetricSpecs = map[string][]string{
	MetricKeyPercentageCPU:   {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_CPU_USAGE},
	MetricKeyNetworkInTotal:  {common.DEFAULT_STATISTICS, common.UNIT_MEM, common.INFLUXDB_FIELD_NET_BPS_RX},
	MetricKeyNetworkOutTotal: {common.DEFAULT_STATISTICS, common.UNIT_MEM, common.INFLUXDB_FIELD_NET_BPS_TX},
	MetricKeyDiskReadBytes:   {common.DEFAULT_STATISTICS, common.UNIT_MEM, common.INFLUXDB_FIELD_DISK_READ_BPS},
	MetricKeyDiskWriteBytes:  {common.DEFAULT_STATISTICS, common.UNIT_MEM, common.INFLUXDB_FIELD_DISK_WRITE_BPS},
	MetricKeyDiskReadOPS:     {common.DEFAULT_STATISTICS, common.UNIT_COUNT_SEC, common.INFLUXDB_FIELD_DISK_READ_IOPS},
	MetricKeyDiskWriteOPS:    {common.DEFAULT_STATISTICS, common.UNIT_COUNT_SEC, common.INFLUXDB_FIELD_DISK_WRITE_IOPS},
}

var azureClassicMetricsSpec = map[string][]string{
	MetricKeyPercentageCPU:       {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_CPU_USAGE},
	MetricKeyClassicNetworkIn:    {common.DEFAULT_STATISTICS, common.UNIT_MEM, common.INFLUXDB_FIELD_NET_BPS_RX},
	MetricKeyClassicNetworkOut:   {common.DEFAULT_STATISTICS, common.UNIT_MEM, common.INFLUXDB_FIELD_NET_BPS_TX},
	MetricKeyClassicDiskReadBPS:  {common.DEFAULT_STATISTICS, common.UNIT_MEM, common.INFLUXDB_FIELD_DISK_READ_BPS},
	MetricKeyClassicDiskWriteBPS: {common.DEFAULT_STATISTICS, common.UNIT_MEM, common.INFLUXDB_FIELD_DISK_WRITE_BPS},
	MetricKeyDiskReadOPS:         {common.DEFAULT_STATISTICS, common.UNIT_COUNT_SEC, common.INFLUXDB_FIELD_DISK_READ_IOPS},
	MetricKeyDiskWriteOPS:        {common.DEFAULT_STATISTICS, common.UNIT_COUNT_SEC, common.INFLUXDB_FIELD_DISK_WRITE_IOPS},
}

var azureRedisMetricsSpec = map[string][]string{
	"percentProcessorTime": {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_REDIS_CPU_USAGE},
	"usedmemorypercentage": {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_REDIS_MEM_USAGE},
	"connectedclients":     {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIFLD_REDIS_CONN_USAGE},
	"operationsPerSecond":  {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIFLD_REDIS_OPT_SES},
	"alltotalkeys":         {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIFLD_REDIS_CACHE_KEYS},
	"expiredkeys":          {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIFLD_REDIS_CACHE_EXP_KEYS},
	"usedmemory":           {common.DEFAULT_STATISTICS, common.UNIT_BYTES, common.INFLUXDB_FIFLD_REDIS_DATA_MEM_USAGE},
	"serverLoad":           {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIFLD_REDIS_SERVER_LOAD},
	"errors":               {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIFLD_REDIS_CONN_ERRORS},
}

//mariadb,mysql,postgresql
var azureRdsMetricsSpec = map[string][]string{
	"cpu_percent":            {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_RDS_CPU_USAGE},
	"memory_percent":         {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_RDS_MEM_USAGE},
	"storage_percent":        {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_RDS_DISK_USAGE},
	"network_bytes_ingress":  {common.DEFAULT_STATISTICS, common.UNIT_BYTES, common.INFLUXDB_FIELD_RDS_NET_BPS_RX},
	"network_bytes_egress":   {common.DEFAULT_STATISTICS, common.UNIT_BYTES, common.INFLUXDB_FIELD_RDS_NET_BPS_TX},
	"io_consumption_percent": {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_RDS_DISK_IO_PERSENT},
	"connections_failed":     {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIELD_RDS_CONN_FAILED},
	"active_connections":     {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIELD_RDS_CONN_ACTIVE},
}

var azureRdsMetricsSpecSqlserver = map[string][]string{
	"cpu_percent":                      {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_RDS_CPU_USAGE},
	"sqlserver_process_memory_percent": {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_RDS_MEM_USAGE},
	"storage_percent":                  {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_RDS_DISK_USAGE},
	"connection_failed":                {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIELD_RDS_CONN_FAILED},
}

var azureElbMetricSpecs = map[string][]string{
	"SnatConnectionCount": {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIELD_ELB_SNAT_PORT},
	"UsedSnatPorts":       {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIELD_ELB_SNAT_CONN_COUNT},
}

// insights.container/pods
var azureK8SPodMetricSpecs = map[string][]string{
	"oomKilledContainerCount":  {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIELD_K8S_POD_OOM_CONTAINER_COUNT},
	"restartingContainerCount": {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIELD_K8S_POD_RESTARTING_CONTAINER_COUNT},
}

// Microsoft.ContainerService/managedClusters
var azureK8SNodeMetricSpecs = map[string][]string{
	"node_cpu_usage_percentage":  {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_NODE_CPU_USAGE},
	"node_memory_rss_percentage": {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_NODE_MEM_USAGE},
	//磁盘已用百分比
	"node_disk_usage_percentage": {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_NODE_DISK_USAGE},
	"node_network_in_bytes":      {common.DEFAULT_STATISTICS, common.UNIT_BYTEPS, common.INFLUXDB_FIELD_K8S_NODE_NET_BPS_RX},
	"node_network_out_bytes":     {common.DEFAULT_STATISTICS, common.UNIT_BYTEPS, common.INFLUXDB_FIELD_K8S_NODE_NET_BPS_TX},
}

// insights.container/persistentvolumes
