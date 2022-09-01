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

package cloudprovider

import (
	"strings"
	"time"
)

type TResourceType string
type TMetricType string

func (key TMetricType) Name() string {
	if !strings.Contains(string(key), ".") {
		return string(key)
	}
	return string(key)[0:strings.Index(string(key), ".")]
}

func (key TMetricType) Key() string {
	if len(key) == 0 {
		return ""
	}
	first, last := 0, len(key)
	if strings.Contains(string(key), ",") {
		last = strings.Index(string(key), ",")
	}
	if strings.Contains(string(key), ".") {
		first = strings.LastIndex(string(key), ".") + 1
	}
	return string(key)[first:last]
}

const (
	METRIC_RESOURCE_TYPE_RDS           TResourceType = "rds"
	METRIC_RESOURCE_TYPE_SERVER        TResourceType = "server"
	METRIC_RESOURCE_TYPE_HOST          TResourceType = "host"
	METRIC_RESOURCE_TYPE_REDIS         TResourceType = "redis"
	METRIC_RESOURCE_TYPE_LB            TResourceType = "lb"
	METRIC_RESOURCE_TYPE_BUCKET        TResourceType = "bucket"
	METRIC_RESOURCE_TYPE_K8S           TResourceType = "k8s"
	METRIC_RESOURCE_TYPE_STORAGE       TResourceType = "storage"
	METRIC_RESOURCE_TYPE_CLOUD_ACCOUNT TResourceType = "cloudaccount_balance"
)

const (
	// RDS监控指标

	// RDS CPU利用率
	// 支持的平台: huawei, aliyun, apsara, azure, jdcloud, qcloud, aws
	// 仅azure的sqlserver支持group_by = database
	RDS_METRIC_TYPE_CPU_USAGE TMetricType = "rds_cpu.usage_active"
	// RDS 内存利用率
	// 支持平台: huawei, aliyun, apsara, azure, jdcloud, qcloud
	// 仅azure的sqlserver支持group_by = database
	RDS_METRIC_TYPE_MEM_USAGE TMetricType = "rds_mem.used_percent"
	// RDS 网络入流量
	// 支持平台: huawei, aliyun, apsara, azure, aws, jdcloud, qcloud
	// 仅azure的sqlserver支持group_by = database
	RDS_METRIC_TYPE_NET_BPS_RX TMetricType = "rds_netio.bps_recv"
	// RDS 网络出流量
	// 支持平台: huawei, aliyun, apsara, azure, aws, jdcloud, qcloud
	// 仅azure的sqlserver支持group_by = database
	RDS_METRIC_TYPE_NET_BPS_TX TMetricType = "rds_netio.bps_sent"

	// RDS磁盘使用率
	// 支持平台: huawei, aliyun, apsara, azure, jdcloud, qcloud
	// 仅azure的sqlserver支持group_by = database
	RDS_METRIC_TYPE_DISK_USAGE TMetricType = "rds_disk.used_percent"
	// RDS磁盘读取IO
	// 支持平台: huawei
	RDS_METRIC_TYPE_DISK_READ_BPS TMetricType = "rds_diskio.read_bps"
	// RDS磁盘写IO
	// 支持平台: huawei
	RDS_METRIC_TYPE_DISK_WRITE_BPS TMetricType = "rds_diskio.write_bps"
	// ---
	// 支持平台: azure
	RDS_METRIC_TYPE_DISK_IO_PERCENT TMetricType = "rds_diskio.used_percent"

	// RDS 连接数
	// 支持平台: huawei, aws, qcloud
	RDS_METRIC_TYPE_CONN_COUNT TMetricType = "rds_conn.used_count"
	// RDS 活跃连接数
	// 支持平台: azure
	RDS_METRIC_TYPE_CONN_ACTIVE TMetricType = "rds_conn.active_count"
	// 连接数使用率
	// 支持平台: aliyun, apsara, qcloud
	RDS_METRIC_TYPE_CONN_USAGE TMetricType = "rds_conn.used_percent"
	// 支持平台: azure
	// 失败连接数
	RDS_METRIC_TYPE_CONN_FAILED TMetricType = "rds_conn.failed_count"

	METRIC_TAG_DATABASE = "database"

	// RDS QPS
	// 支持平台: huawei, qcloud
	RDS_METRIC_TYPE_QPS TMetricType = "rds_qps.query_qps"
	// RDS TPS
	// 支持平台: huawei, qcloud
	RDS_METRIC_TYPE_TPS TMetricType = "rds_tps.trans_qps"
	// RDS innodb读IO
	// 支持平台: huawei, qcloud
	RDS_METRIC_TYPE_INNODB_READ_BPS TMetricType = "rds_innodb.read_bps"
	// RDS innodb写IO
	// 支持平台 huawei, qcloud
	RDS_METRIC_TYPE_INNODB_WRITE_BPS TMetricType = "rds_innodb.write_bps"

	// 虚拟机CPU使用率
	// 支持平台: kvm, huawei, aliyun, apsara, azure, esxi, google, bingocloud, aws, jdcloud, ecloud, zstack, qcloud
	VM_METRIC_TYPE_CPU_USAGE TMetricType = "vm_cpu.usage_active"
	// 虚拟机内存使用率
	// 支持平台: kvm, aliyun, apsara, azure, esxi, bingocloud, jdcloud, ecloud, qcloud
	VM_METRIC_TYPE_MEM_USAGE TMetricType = "vm_mem.used_percent"
	// 虚拟机磁盘使用率
	// 支持平台: aliyun, apsara, jdcloud, azure
	// 支持按盘符(group_by=device)平台: aliyun, apsara
	VM_METRIC_TYPE_DISK_USAGE TMetricType = "vm_disk.used_percent"

	// 虚拟机磁盘读速率
	// 支持平台: huawei, aliyun, apsara, azure, esxi, google, bingocloud, aws, jdcloud, ecloud, zstack
	VM_METRIC_TYPE_DISK_IO_READ_BPS TMetricType = "vm_diskio.read_bps"
	// 虚拟机磁盘写速率
	// 支持平台: huawei, aliyun, apsara, azure, esxi, google, bingocloud, aws, jdcloud, ecloud, zstack
	VM_METRIC_TYPE_DISK_IO_WRITE_BPS TMetricType = "vm_diskio.write_bps"
	// 虚拟机磁盘读IOPS
	// 支持平台: huawei, aliyun, apsara, azure, google, bingocloud, aws, jdcloud, ecloud, zstack
	VM_METRIC_TYPE_DISK_IO_READ_IOPS TMetricType = "vm_diskio.read_iops"
	// 虚拟机磁盘写IOPS
	// 支持平台: huawei, aliyun, apsara, azure, google, bingocloud, aws, jdcloud, ecloud, zstack
	VM_METRIC_TYPE_DISK_IO_WRITE_IOPS TMetricType = "vm_diskio.write_iops"

	// 虚拟机网络入速率
	// 支持平台: huawei, aliyun, apsara, azure, esxi, google, bingocloud, aws, jdcloud, ecloud, zstack, qcloud
	VM_METRIC_TYPE_NET_BPS_RX TMetricType = "vm_netio.bps_recv"
	// 虚拟机网络出速率
	// 支持平台: huawei, aliyun, apsara, azure, esxi, google, bingocloud, aws, jdcloud, ecloud, zstack, qcloud
	VM_METRIC_TYPE_NET_BPS_TX TMetricType = "vm_netio.bps_sent"

	// 宿主机CPU使用率
	// 支持平台: esxi
	HOST_METRIC_TYPE_CPU_USAGE TMetricType = "cpu.usage_active"
	// 宿主机内存使用率
	// 支持平台: esxi
	HOST_METRIC_TYPE_MEM_USAGE TMetricType = "mem.used_percent"
	// 宿主机磁盘读速率
	// 支持平台: esxi
	HOST_METRIC_TYPE_DISK_IO_READ_BPS TMetricType = "diskio.read_bps"
	// 宿主机磁盘写速率
	// 支持平台: esxi
	HOST_METRIC_TYPE_DISK_IO_WRITE_BPS TMetricType = "diskio.write_bps"
	// 宿主机网络入速率
	// 支持平台: esxi
	HOST_METRIC_TYPE_NET_BPS_RX TMetricType = "net.bps_recv"
	// 宿主机网络出速率
	// 支持平台: esxi
	HOST_METRIC_TYPE_NET_BPS_TX TMetricType = "net.bps_sent"

	// Redis CPU使用率
	// 支持平台: huawei, aliyun, azure, apsara, aws, qcloud
	REDIS_METRIC_TYPE_CPU_USAGE TMetricType = "dcs_cpu.usage_active"
	// Redis 内存使用率
	// 支持平台: huawei, aliyun, azure, apsara, qcloud
	REDIS_METRIC_TYPE_MEM_USAGE TMetricType = "dcs_mem.used_percent"
	// Redis 网络入速率
	// 支持平台: huawei, aliyun, apsara, qcloud
	REDIS_METRIC_TYPE_NET_BPS_RX TMetricType = "dcs_netio.bps_recv"
	// Redis 网络出速率
	// 支持平台: huawei, aliyun, apsara, qcloud
	REDIS_METRIC_TYPE_NET_BPS_TX TMetricType = "dcs_netio.bps_sent"
	// Redis 网络连接数
	// 支持平台: huawei, aliyun, apsara, azure, aws, qcloud
	REDIS_METRIC_TYPE_USED_CONN TMetricType = "dcs_conn.used_conn"
	// 每秒处理指令数
	// 支持平台: huawei, aliyun, apsara, azure, qcloud
	REDIS_METRIC_TYPE_OPT_SES TMetricType = "dcs_instantopt.opt_sec"
	// 命中key数量
	// 支持平台: huawei, aliyun, apsara, azure, qcloud
	REDIS_METRIC_TYPE_CACHE_KEYS TMetricType = "dcs_cachekeys.key_count"
	// Expired Key数量
	// 支持平台: huawei, aliyun, apsara, azure, aws, qcloud
	REDIS_METRIC_TYPE_CACHE_EXP_KEYS TMetricType = "dcs_cachekeys.expire_key_count"
	// 内存使用量
	// 支持平台: huawei, aliyun, azure, apsara, qcloud
	REDIS_METRIC_TYPE_DATA_MEM_USAGE TMetricType = "dcs_datamem.used_byte"

	// 支持平台: azure
	LB_METRIC_TYPE_SNAT_PORT TMetricType = "haproxy.used_snat_port"
	// 支持平台: azure
	LB_METRIC_TYPE_SNAT_CONN_COUNT TMetricType = "haproxy.snat_conn_count"
	// 入速率
	// 支持平台: huawei, aliyun, apsara
	LB_METRIC_TYPE_NET_BPS_RX TMetricType = "haproxy.bin"
	// 出速率
	// 支持平台: huawei, aliyun, apsara
	LB_METRIC_TYPE_NET_BPS_TX TMetricType = "haproxy.bout"
	// 状态码统计
	// 支持平台: huawei, aliyun, apsara
	LB_METRIC_TYPE_HRSP_COUNT TMetricType = "haproxy.hrsp_Nxx"

	// 对象存储出速率
	// 支持平台: huawei, aliyun, apsara
	BUCKET_METRIC_TYPE_NET_BPS_TX TMetricType = "oss_netio.bps_sent"
	// 对象存储入速率
	// 支持平台: huawei, aliyun, apsara
	BUCKET_METRIC_TYPE_NET_BPS_RX TMetricType = "oss_netio.bps_recv"
	// 请求延时
	// 支持平台: huawei, aliyun, apsara
	BUCKET_METRIC_TYPE_LATECY TMetricType = "oss_latency.req_late"
	// 请求数量
	// 支持平台: huawei, aliyun, apsara
	BUCKET_METRYC_TYPE_REQ_COUNT TMetricType = "oss_req.req_count"

	METRIC_TAG_REQUST      = "request"
	METRIC_TAG_REQUST_GET  = "get"
	METRIC_TAG_REQUST_POST = "post"
	METRIC_TAG_REQUST_2XX  = "2xx"
	METRIC_TAG_REQUST_3XX  = "3xx"
	METRIC_TAG_REQUST_4XX  = "4xx"
	METRIC_TAG_REQUST_5XX  = "5xx"

	METRIC_TAG_NET_TYPE          = "net_type"
	METRIC_TAG_NET_TYPE_INTERNET = "internet"
	METRIC_TAG_NET_TYPE_INTRANET = "intranet"

	METRIC_TAG_TYPE_DISK_TYPE     = "disk_type"
	METRIC_TAG_TYPE_DISK_TYPE_EBS = "ebs"

	// 磁盘利用率
	METRIC_TAG_DEVICE = "device"

	METRIC_TAG_NODE = "node"

	// k8s节点CPU使用率
	// 支持平台: aliyun, azure, qcloud
	K8S_NODE_METRIC_TYPE_CPU_USAGE TMetricType = "k8s_node_cpu.usage_active"
	// k8s节点内存使用率
	// 支持平台: aliyun, azure, qcloud
	K8S_NODE_METRIC_TYPE_MEM_USAGE TMetricType = "k8s_node_mem.used_percent"

	K8S_NODE_METRIC_TYPE_DISK_USAGE TMetricType = "k8s_node_disk.used_percent"
	K8S_NODE_METRIC_TYPE_NET_BPS_RX TMetricType = "k8s_node_netio.bps_recv"
	K8S_NODE_METRIC_TYPE_NET_BPS_TX TMetricType = "k8s_node_netio.bps_sent"
)

var (
	ALL_RDS_METRIC_TYPES = []TMetricType{
		RDS_METRIC_TYPE_CPU_USAGE,
		RDS_METRIC_TYPE_MEM_USAGE,
		RDS_METRIC_TYPE_NET_BPS_RX,
		RDS_METRIC_TYPE_NET_BPS_TX,

		RDS_METRIC_TYPE_DISK_USAGE,
		RDS_METRIC_TYPE_DISK_READ_BPS,
		RDS_METRIC_TYPE_DISK_WRITE_BPS,
		RDS_METRIC_TYPE_DISK_IO_PERCENT,

		RDS_METRIC_TYPE_CONN_COUNT,
		RDS_METRIC_TYPE_CONN_ACTIVE,
		RDS_METRIC_TYPE_CONN_USAGE,
		RDS_METRIC_TYPE_CONN_FAILED,

		RDS_METRIC_TYPE_QPS,
		RDS_METRIC_TYPE_TPS,
		RDS_METRIC_TYPE_INNODB_READ_BPS,
		RDS_METRIC_TYPE_INNODB_WRITE_BPS,
	}

	ALL_HOST_METRIC_TYPES = []TMetricType{
		HOST_METRIC_TYPE_CPU_USAGE,
		HOST_METRIC_TYPE_MEM_USAGE,
		HOST_METRIC_TYPE_DISK_IO_READ_BPS,
		HOST_METRIC_TYPE_DISK_IO_WRITE_BPS,
		HOST_METRIC_TYPE_NET_BPS_RX,
		HOST_METRIC_TYPE_NET_BPS_TX,
	}

	ALL_VM_METRIC_TYPES = []TMetricType{
		VM_METRIC_TYPE_CPU_USAGE,
		VM_METRIC_TYPE_MEM_USAGE,
		VM_METRIC_TYPE_DISK_USAGE,

		VM_METRIC_TYPE_DISK_IO_READ_BPS,
		VM_METRIC_TYPE_DISK_IO_WRITE_BPS,
		VM_METRIC_TYPE_DISK_IO_READ_IOPS,
		VM_METRIC_TYPE_DISK_IO_WRITE_IOPS,

		VM_METRIC_TYPE_NET_BPS_RX,
		VM_METRIC_TYPE_NET_BPS_TX,
	}

	ALL_REDIS_METRIC_TYPES = []TMetricType{
		REDIS_METRIC_TYPE_CPU_USAGE,
		REDIS_METRIC_TYPE_MEM_USAGE,
		REDIS_METRIC_TYPE_NET_BPS_RX,
		REDIS_METRIC_TYPE_NET_BPS_TX,
		REDIS_METRIC_TYPE_USED_CONN,
		REDIS_METRIC_TYPE_OPT_SES,
		REDIS_METRIC_TYPE_CACHE_KEYS,
		REDIS_METRIC_TYPE_CACHE_EXP_KEYS,
		REDIS_METRIC_TYPE_DATA_MEM_USAGE,
	}

	ALL_LB_METRIC_TYPES = []TMetricType{
		LB_METRIC_TYPE_SNAT_PORT,
		LB_METRIC_TYPE_SNAT_CONN_COUNT,
		LB_METRIC_TYPE_NET_BPS_RX,
		LB_METRIC_TYPE_NET_BPS_TX,
		LB_METRIC_TYPE_HRSP_COUNT,
	}

	ALL_BUCKET_TYPES = []TMetricType{
		BUCKET_METRIC_TYPE_NET_BPS_TX,
		BUCKET_METRIC_TYPE_NET_BPS_RX,
		BUCKET_METRIC_TYPE_LATECY,
		BUCKET_METRYC_TYPE_REQ_COUNT,
	}

	ALL_K8S_NODE_TYPES = []TMetricType{
		K8S_NODE_METRIC_TYPE_CPU_USAGE,
		K8S_NODE_METRIC_TYPE_MEM_USAGE,
	}
)

type MetricListOptions struct {
	ResourceType TResourceType
	MetricType   TMetricType

	ResourceId string
	// batch metric pull for tencentcloud
	ResourceIds []string
	RegionExtId string
	StartTime   time.Time
	EndTime     time.Time

	Interval int
	// rds
	Engine string
}

type MetricValue struct {
	Timestamp time.Time
	Value     float64
	Tags      map[string]string
}

type MetricValues struct {
	Id         string
	Unit       string
	MetricType TMetricType
	Values     []MetricValue
}
