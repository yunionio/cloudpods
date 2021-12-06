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

package alimon

import "yunion.io/x/onecloud/pkg/cloudmon/collectors/common"

const (
	SERVER_METRIC_NAMESPACE = "acs_ecs_dashboard"

	REDIS_METRIC_NAMESPACE = "acs_kvstore"
	RDS_METRIC_NAMESPACE   = "acs_rds_dashboard"
	OSS_METRIC_NAMESPACE   = "acs_oss"
	ELB_METRIC_NAMESPACE   = "acs_slb_dashboard"

	K8S_METRIC_NAMESPACE = "acs_k8s"
)

//multiCloud查询指标列表组装
var aliMetricSpecs = map[string][]string{
	"CPUUtilization":  {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_CPU_USAGE},
	"InternetInRate":  {common.DEFAULT_STATISTICS, common.UNIT_BPS, common.INFLUXDB_FIELD_NET_BPS_RX_INTERNET},
	"IntranetInRate":  {common.DEFAULT_STATISTICS, common.UNIT_BPS, common.INFLUXDB_FIELD_NET_BPS_RX_INTRANET},
	"InternetOutRate": {common.DEFAULT_STATISTICS, common.UNIT_BPS, common.INFLUXDB_FIELD_NET_BPS_TX_INTERNET},
	"IntranetOutRate": {common.DEFAULT_STATISTICS, common.UNIT_BPS, common.INFLUXDB_FIELD_NET_BPS_TX_INTRANET},
	"DiskReadBPS":     {common.DEFAULT_STATISTICS, common.UNIT_BYTEPS, common.INFLUXDB_FIELD_DISK_READ_BPS},
	"DiskWriteBPS":    {common.DEFAULT_STATISTICS, common.UNIT_BYTEPS, common.INFLUXDB_FIELD_DISK_WRITE_BPS},
	"DiskReadIOPS":    {common.DEFAULT_STATISTICS, common.UNIT_CPS, common.INFLUXDB_FIELD_DISK_READ_IOPS},
	"DiskWriteIOPS":   {common.DEFAULT_STATISTICS, common.UNIT_CPS, common.INFLUXDB_FIELD_DISK_WRITE_IOPS},
}
var aliRdsMetricSpecs = map[string][]string{
	"CpuUsage":                {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_RDS_CPU_USAGE},
	"MemoryUsage":             {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_RDS_MEM_USAGE},
	"MySQL_NetworkInNew":      {common.DEFAULT_STATISTICS, common.UNIT_BPS, common.INFLUXDB_FIELD_RDS_NET_BPS_RX_MYSQL},
	"MySQL_NetworkOutNew":     {common.DEFAULT_STATISTICS, common.UNIT_BPS, common.INFLUXDB_FIELD_RDS_NET_BPS_TX_MYSQL},
	"DiskUsage":               {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_RDS_DISK_USAGE},
	"SQLServer_NetworkInNew":  {common.DEFAULT_STATISTICS, common.UNIT_BPS, common.INFLUXDB_FIELD_RDS_NET_BPS_RX_SQLSERVER},
	"SQLServer_NetworkOutNew": {common.DEFAULT_STATISTICS, common.UNIT_BPS, common.INFLUXDB_FIELD_RDS_NET_BPS_TX_SQLSERVER},
	"ConnectionUsage":         {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_RDS_CONN_USAGE},
}
var aliRedisMetricSpecs = map[string][]string{
	"CpuUsage":       {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_REDIS_CPU_USAGE},
	"MemoryUsage":    {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_REDIS_MEM_USAGE},
	"IntranetIn":     {common.DEFAULT_STATISTICS, common.UNIT_BPS, common.INFLUXDB_FIELD_REDIS_NET_BPS_RX},
	"IntranetOut":    {common.DEFAULT_STATISTICS, common.UNIT_BPS, common.INFLUXDB_FIELD_REDIS_NET_BPS_TX},
	"UsedConnection": {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIFLD_REDIS_CONN_USAGE},
	"UsedQPS":        {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIFLD_REDIS_OPT_SES},
	"Keys":           {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIFLD_REDIS_CACHE_KEYS},
	"ExpiredKeys":    {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIFLD_REDIS_CACHE_EXP_KEYS},
	"UsedMemory":     {common.DEFAULT_STATISTICS, common.UNIT_MEM, common.INFLUXDB_FIFLD_REDIS_DATA_MEM_USAGE},
}
var aliOSSMetricSpecs = map[string][]string{
	"InternetSend":         {common.DEFAULT_STATISTICS, common.UNIT_MEM, common.INFLUXDB_FIELD_OSS_NET_BPS_TX_INTERNET},
	"InternetRecv":         {common.DEFAULT_STATISTICS, common.UNIT_MEM, common.INFLUXDB_FIELD_OSS_NET_BPS_RX_INTERNET},
	"IntranetSend":         {common.DEFAULT_STATISTICS, common.UNIT_MEM, common.INFLUXDB_FIELD_OSS_NET_BPS_TX_INTRANET},
	"IntranetRecv":         {common.DEFAULT_STATISTICS, common.UNIT_MEM, common.INFLUXDB_FIELD_OSS_NET_BPS_RX_INTRANET},
	"GetObjectE2eLatency":  {common.DEFAULT_STATISTICS, common.UNIT_MSEC, common.INFLUXDB_FIELD_OSS_LATECY_GET},
	"PostObjectE2eLatency": {common.DEFAULT_STATISTICS, common.UNIT_MSEC, common.INFLUXDB_FIELD_OSS_LATECY_POST},
	"GetObjectCount":       {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIELD_OSS_REQ_COUNT_GET},
	"PostObjectCount":      {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIELD_OSS_REQ_COUNT_POST},
	"ServerErrorCount":     {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIELD_OSS_REQ_COUNT_5XX},
}
var aliElbMetricSpecs = map[string][]string{
	"InstanceTrafficRX":     {common.DEFAULT_STATISTICS, common.UNIT_BPS, common.INFLUXDB_FIELD_ELB_NET_BPS_RX},
	"InstanceTrafficTX":     {common.DEFAULT_STATISTICS, common.UNIT_BPS, common.INFLUXDB_FIELD_ELB_NET_BPS_TX},
	"InstanceStatusCode2xx": {common.DEFAULT_STATISTICS, common.UNIT_COUNT_SEC, common.INFLUXDB_FIELD_ELB_HRSP_COUNT_2XX},
	"InstanceStatusCode3xx": {common.DEFAULT_STATISTICS, common.UNIT_COUNT_SEC, common.INFLUXDB_FIELD_ELB_HRSP_COUNT_3XX},
	"InstanceStatusCode4xx": {common.DEFAULT_STATISTICS, common.UNIT_COUNT_SEC, common.INFLUXDB_FIELD_ELB_HRSP_COUNT_4XX},
	"InstanceStatusCode5xx": {common.DEFAULT_STATISTICS, common.UNIT_COUNT_SEC, common.INFLUXDB_FIELD_ELB_HRSP_COUNT_5XX},
}

var aliK8SClusterMetricSpecs = map[string][]string{
	"cluster.cpu.utilization":    {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_CLUSTER_CPU_USAGE},
	"cluster.memory.utilization": {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_CLUSTER_MEM_USAGE},
}

var aliK8SPodMetricSpecs = map[string][]string{
	"pod.cpu.utilization":    {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_POD_CPU_USAGE},
	"pod.memory.utilization": {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_POD_MEM_USAGE},
}

var aliK8SNodeMetricSpecs = map[string][]string{
	"node.cpu.utilization":    {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_NODE_CPU_USAGE},
	"node.memory.utilization": {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_NODE_MEM_USAGE},
}
