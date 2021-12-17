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

package qcmon

import "yunion.io/x/onecloud/pkg/cloudmon/collectors/common"

const (
	SERVER_METRIC_NAMESPACE = "QCE/CVM"
	REDIS_METRIC_NAMESPACE  = "QCE/REDIS"
	RDS_METRIC_NAMESPACE    = "QCE/CDB"
	K8S_METRIC_NAMESPACE    = "QCE/TKE"

	KEY_VMS   = "vms"
	KEY_CPUS  = "cpus"
	KEY_MEMS  = "mems"
	KEY_DISKS = "disks"

	KEY_LIMIT  = "limit"
	KEY_ADMIN  = "admin"
	KEY_USABLE = "usable"
)

var tecentMetricSpecs = map[string][]string{
	"CPUUsage":      {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_CPU_USAGE},
	"MemUsage":      {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_MEM_USAGE},
	"lanOuttraffic": {common.DEFAULT_STATISTICS, common.UNIT_MBPS, common.INFLUXDB_FIELD_NET_BPS_TX_INTRANET},
	"WanOuttraffic": {common.DEFAULT_STATISTICS, common.UNIT_MBPS, common.INFLUXDB_FIELD_NET_BPS_TX_INTERNET},
	"lanIntraffic":  {common.DEFAULT_STATISTICS, common.UNIT_MBPS, common.INFLUXDB_FIELD_NET_BPS_RX_INTRANET},
	"WanIntraffic":  {common.DEFAULT_STATISTICS, common.UNIT_MBPS, common.INFLUXDB_FIELD_NET_BPS_RX_INTERNET},
}

var tecentRedisMetricSpecs = map[string][]string{
	"CpuUsMin":         {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_REDIS_CPU_USAGE},
	"StorageUsMin":     {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_REDIS_MEM_USAGE},
	"InFlowMin":        {common.DEFAULT_STATISTICS, common.UNIT_BPS, common.INFLUXDB_FIELD_REDIS_NET_BPS_RX},
	"OutFlowMin":       {common.DEFAULT_STATISTICS, common.UNIT_BPS, common.INFLUXDB_FIELD_REDIS_NET_BPS_TX},
	"ConnectionsUsMin": {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIFLD_REDIS_CONN_USAGE},
	"QpsMin":           {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIFLD_REDIS_OPT_SES},
	"KeysMin":          {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIFLD_REDIS_CACHE_KEYS},
	"ExpiredKeysMin":   {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIFLD_REDIS_CACHE_EXP_KEYS},
	"StorageMin":       {common.DEFAULT_STATISTICS, common.UNIT_MEM, common.INFLUXDB_FIFLD_REDIS_DATA_MEM_USAGE},
}

var tecentRdsMetricSpecs = map[string][]string{
	"CPUUseRate":        {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_RDS_CPU_USAGE},
	"MemoryUseRate":     {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_RDS_MEM_USAGE},
	"BytesSent":         {common.DEFAULT_STATISTICS, common.UNIT_BYTEPS, common.INFLUXDB_FIELD_RDS_NET_BPS_TX_INTRANET},
	"BytesReceived":     {common.DEFAULT_STATISTICS, common.UNIT_BYTEPS, common.INFLUXDB_FIELD_RDS_NET_BPS_RX_INTRANET},
	"VolumeRate":        {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_RDS_DISK_USAGE},
	"ThreadsConnected":  {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIELD_RDS_CONN_COUNT},
	"ConnectionUseRate": {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_RDS_CONN_USAGE},
	"QPS":               {common.DEFAULT_STATISTICS, common.UNIT_COUNT_SEC, common.INFLUXDB_FIELD_RDS_QPS},
	"TPS":               {common.DEFAULT_STATISTICS, common.UNIT_COUNT_SEC, common.INFLUXDB_FIELD_RDS_TPS},
	"InnodbDataRead":    {common.DEFAULT_STATISTICS, common.UNIT_BYTEPS, common.INFLUXDB_FIELD_RDS_INNODB_REDA_BPS},
	"InnodbDataWritten": {common.DEFAULT_STATISTICS, common.UNIT_BYTEPS, common.INFLUXDB_FIELD_RDS_INNODB_WRITE_BPS},
}

var tecentK8SClusterMetricSpecs = map[string][]string{
	"K8sClusterRateCpuCoreUsedCluster":     {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_CLUSTER_CPU_USAGE},
	"K8sClusterRateMemRequestBytesCluster": {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_CLUSTER_MEM_USAGE},
	"K8sClusterAllocatablePodsTotal":       {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIELD_K8S_CLUSTER_ALLOCATABLE_POD},
	"K8sClusterCpuCoreTotal":               {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIELD_K8S_CLUSTER_TOTAL_CPUCORE},
	"K8sClusterRateCpuCoreRequestCluster":  {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_CLUSTER_CPU_ALLOCATED},
}

var tecentK8SDeployMetricSpecs = map[string][]string{
	"K8sWorkloadRateCpuCoreUsedCluster":   {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_DEPLOY_CPU_USAGE},
	"K8sWorkloadRateMemUsageBytesCluster": {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_DEPLOY_MEM_USAGE},
	"K8sWorkloadPodRestartTotal":          {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIELD_K8S_DEPLOY_RESTART_TOTAL},
}

var tecentK8SPodMetricSpecs = map[string][]string{
	"K8sPodRateCpuCoreUsedLimit": {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_POD_CPU_USAGE},
	"K8sPodRateMemUsageLimit":    {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_POD_MEM_USAGE},
	"K8sPodRestartTotal":         {common.DEFAULT_STATISTICS, common.UNIT_COUNT, common.INFLUXDB_FIELD_K8S_POD_RESTART_TOTAL},
}

var tecentK8SContainerMetricSpecs = map[string][]string{
	"K8sContainerRateCpuCoreUsedLimit":  {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_CONTAINER_CPU_USAGE},
	"K8sContainerRateMemUsageLimit":     {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_CONTAINER_MEM_USAGE},
	"K8sWorkloadNetworkReceiveBytesBw":  {common.DEFAULT_STATISTICS, common.UNIT_MBPS, common.INFLUXDB_FIELD_K8S_DEPLOY_NET_BPS_RX},
	"K8sWorkloadNetworkTransmitBytesBw": {common.DEFAULT_STATISTICS, common.UNIT_MBPS, common.INFLUXDB_FIELD_K8S_DEPLOY_NET_BPS_TX},
}

var tecentK8SNodeMetricSpecs = map[string][]string{
	"K8sNodeCpuUsage":        {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_NODE_CPU_USAGE},
	"K8sNodeMemUsage":        {common.DEFAULT_STATISTICS, common.UNIT_PERCENT, common.INFLUXDB_FIELD_K8S_NODE_MEM_USAGE},
	"K8sNodeLanIntraffic":    {common.DEFAULT_STATISTICS, common.UNIT_MBPS, common.INFLUXDB_FIELD_K8S_NODE_NET_BPS_RX_INTRANET},
	"K8sNodeWanIntraffic":    {common.DEFAULT_STATISTICS, common.UNIT_MBPS, common.INFLUXDB_FIELD_K8S_NODE_NET_BPS_RX_INTERNET},
	"K8sNodeLanOuttraffic":   {common.DEFAULT_STATISTICS, common.UNIT_MBPS, common.INFLUXDB_FIELD_K8S_NODE_NET_BPS_TX_INTRANET},
	"K8sNodeWanOuttraffic":   {common.DEFAULT_STATISTICS, common.UNIT_MBPS, common.INFLUXDB_FIELD_K8S_NODE_NET_BPS_TX_INTERNET},
	"K8sNodePodRestartTotal": {common.DEFAULT_STATISTICS, common.UNIT_MBPS, common.INFLUXDB_FIELD_K8S_NODE_POD_RESTART_TOTAL},
}
