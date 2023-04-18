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
	METRIC_RESOURCE_TYPE_RDS            TResourceType = "rds"
	METRIC_RESOURCE_TYPE_SERVER         TResourceType = "server"
	METRIC_RESOURCE_TYPE_HOST           TResourceType = "host"
	METRIC_RESOURCE_TYPE_REDIS          TResourceType = "redis"
	METRIC_RESOURCE_TYPE_LB             TResourceType = "lb"
	METRIC_RESOURCE_TYPE_BUCKET         TResourceType = "bucket"
	METRIC_RESOURCE_TYPE_K8S            TResourceType = "k8s"
	METRIC_RESOURCE_TYPE_STORAGE        TResourceType = "storage"
	METRIC_RESOURCE_TYPE_WIRE           TResourceType = "wire"
	METRIC_RESOURCE_TYPE_CLOUD_ACCOUNT  TResourceType = "cloudaccount_balance"
	METRIC_RESOURCE_TYPE_MODELARTS_POOL TResourceType = "modelarts"
	METRIC_RESOURCE_TYPE_EIP            TResourceType = "eip"
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

	// RDS QPS(每秒查询数)
	// 支持平台: huawei, qcloud, aliyun, apsara
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

	// 虚拟机TCP连接数
	// 支持平台: aliyun, apsara
	VM_METRIC_TYPE_NET_TCP_CONNECTION TMetricType = "vm_netio.tcp_connections"
	// 虚拟机进程监控
	// 支持平台: aliyun, apsara
	VM_METRIC_TYPE_PROCESS_NUMBER = "vm_process.number"

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
	//宿主机磁盘读IOPS
	// 支持平台: bingocloud
	HOST_METRIC_TYPE_DISK_IO_READ_IOPS TMetricType = "diskio.read_iops"
	//宿主机磁盘写IOPS
	// 支持平台: bingocloud
	HOST_METRIC_TYPE_DISK_IO_WRITE_IOPS TMetricType = "diskio.write_iops"

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
	// 入带宽速率
	// 支持平台: huawei, aliyun, apsara
	LB_METRIC_TYPE_NET_BPS_RX TMetricType = "haproxy.bin"
	// 出带宽速率
	// 支持平台: huawei, aliyun, apsara
	LB_METRIC_TYPE_NET_BPS_TX TMetricType = "haproxy.bout"
	// 入包速率
	// 支持平台: aliyun, apsara
	LB_METRIC_TYPE_NET_PACKET_RX TMetricType = "haproxy.packet_rx"
	// 出包速率
	// 支持平台: aliyun, apsara
	LB_METRIC_TYPE_NET_PACKET_TX TMetricType = "haproxy.packet_tx"
	// 非活跃连接数
	// 支持平台: apsara, aliyun, huawei
	LB_METRIC_TYPE_NET_INACTIVE_CONNECTION = "haproxy.inactive_connection"
	// 活跃连接数
	// 支持平台: apsara, aliyun, huawei
	LB_METRIC_TYPE_NET_ACTIVE_CONNECTION = "haproxy.active_connection"
	// 最大并发数
	// 支持平台: apsara, aliyun, huawei
	LB_METRIC_TYPE_MAX_CONNECTION = "haproxy.max_connection"
	// 后端异常ECS实例个数
	// 支持平台: apsara, aliyun
	LB_METRIC_TYPE_UNHEALTHY_SERVER_COUNT = "haproxy.unhealthy_server_count"
	// 状态码统计
	// 支持平台: huawei, aliyun, apsara
	LB_METRIC_TYPE_HRSP_COUNT TMetricType = "haproxy.hrsp_Nxx"
	// 入方向丢弃流量
	// 支持平台: aliyun
	LB_METRIC_TYPE_DROP_TRAFFIC_TX = "haproxy.drop_traffic_tx"
	// 出方向丢弃流量
	// 支持平台: aliyun, apsara
	LB_METRIC_TYPE_DROP_TRAFFIC_RX = "haproxy.drop_traffic_rx"
	// 入方向丢弃包数
	// 支持平台: aliyun
	LB_METRIC_TYPE_DROP_PACKET_TX = "haproxy.drop_packet_tx"
	// 出方向丢弃包数
	// 支持平台: aliyun, apsara
	LB_METRIC_TYPE_DROP_PACKET_RX = "haproxy.drop_packet_rx"

	// 对象存储出速率
	// 支持平台: huawei, aliyun, apsara
	BUCKET_METRIC_TYPE_NET_BPS_TX TMetricType = "oss_netio.bps_sent"
	// 对象存储入速率
	// 支持平台: huawei, aliyun, apsara
	BUCKET_METRIC_TYPE_NET_BPS_RX TMetricType = "oss_netio.bps_recv"
	// 请求延时
	// 支持平台: huawei, aliyun, apsara
	BUCKET_METRIC_TYPE_LATECY TMetricType = "oss_latency.req_late"
	// 总请求数量
	// 支持平台: huawei, aliyun, apsara
	BUCKET_METRYC_TYPE_REQ_COUNT TMetricType = "oss_req.req_count"
	// 服务端请求错误数量
	// 支持平台: aliyun, apsara
	BUCKET_METRIC_TYPE_REQ_5XX_COUNT TMetricType = "oss_req.5xx_count"
	// 服务端请求错误数量
	// 支持平台: aliyun, apsara
	BUCKET_METRIC_TYPE_REQ_4XX_COUNT TMetricType = "oss_req.4xx_count"
	// 重定向数量
	// 支持平台: aliyun, apsara
	BUCKET_METRIC_TYPE_REQ_3XX_COUNT TMetricType = "oss_req.3xx_count"
	// 正常请求数量
	// 支持平台: aliyun, apsara
	BUCKET_METRIC_TYPE_REQ_2XX_COUNT TMetricType = "oss_req.2xx_count"
	// 存储总容量(bit)
	// 支持平台: aliyun, apsara
	BUCKET_METRIC_TYPE_STORAGE_SIZE = "oss_storage.size"

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

	// 进程名称
	METRIC_TAG_PROCESS_NAME = "process_name"

	// 状态
	METRIC_TAG_STATE = "state"

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

	// modelarts专属资源池监控数据
	MODELARTS_POOL_METRIC_TYPE_CPU_USAGE     TMetricType = "modelarts_pool_cpu.usage_percent"
	MODELARTS_POOL_METRIC_TYPE_MEM_USAGE     TMetricType = "modelarts_pool_mem.usage_percent"
	MODELARTS_POOL_METRIC_TYPE_GPU_MEM_USAGE TMetricType = "modelarts_pool_gpu_mem.usage_percent"
	MODELARTS_POOL_METRIC_TYPE_GPU_UTIL      TMetricType = "modelarts_pool_gpu_util.percent"
	MODELARTS_POOL_METRIC_TYPE_NPU_UTIL      TMetricType = "modelarts_pool_npu_util.percent"
	MODELARTS_POOL_METRIC_TYPE_NPU_MEM_USAGE TMetricType = "modelarts_pool_npu_mem.usage_percent"

	//磁盘可用容量
	MODELARTS_POOL_METRIC_TYPE_DISK_AVAILABLE_CAPACITY TMetricType = "modelarts_pool_disk.available_capacity"
	MODELARTS_POOL_METRIC_TYPE_DISK_CAPACITY           TMetricType = "modelarts_pool_disk.capacity"
	MODELARTS_POOL_METRIC_TYPE_DISK_USAGE              TMetricType = "modelarts_pool_disk.usage_percent"

	WIRE_METRIC_TYPE_CPU_USAGE            TMetricType = "wire_cpu.usage_percent"
	WIRE_METRIC_TYPE_MEM_USAGE            TMetricType = "wire_mem.usage_percent"
	WIRE_METRIC_TYPE_NET_RT               TMetricType = "wire_net.rt"               // 响应时间ms
	WIRE_METRIC_TYPE_NET_UNREACHABLE_RATE TMetricType = "wire_net.unreachable_rate" // 不可达率

	// EIP入带宽
	EIP_METRIC_TYPE_NET_BPS_RX TMetricType = "eip_net.bps_recv"
	// EIP出带宽
	EIP_METRIC_TYPE_NET_BPS_TX TMetricType = "eip_net.bps_sent"

	// EIP 出方向限速丢包率
	EIP_METRIC_TYPE_NET_DROP_SPEED_TX TMetricType = "eip_net.drop_speed_rx"
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
		VM_METRIC_TYPE_NET_TCP_CONNECTION,

		VM_METRIC_TYPE_PROCESS_NUMBER,
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

		LB_METRIC_TYPE_NET_PACKET_RX,
		LB_METRIC_TYPE_NET_PACKET_TX,

		LB_METRIC_TYPE_UNHEALTHY_SERVER_COUNT,
		LB_METRIC_TYPE_NET_INACTIVE_CONNECTION,
		LB_METRIC_TYPE_MAX_CONNECTION,

		LB_METRIC_TYPE_DROP_PACKET_RX,
		LB_METRIC_TYPE_DROP_PACKET_TX,

		LB_METRIC_TYPE_DROP_TRAFFIC_RX,
		LB_METRIC_TYPE_DROP_TRAFFIC_TX,
	}

	ALL_BUCKET_TYPES = []TMetricType{
		BUCKET_METRIC_TYPE_NET_BPS_TX,
		BUCKET_METRIC_TYPE_NET_BPS_RX,
		BUCKET_METRIC_TYPE_LATECY,
		BUCKET_METRYC_TYPE_REQ_COUNT,

		BUCKET_METRIC_TYPE_REQ_5XX_COUNT,
		BUCKET_METRIC_TYPE_REQ_4XX_COUNT,
		BUCKET_METRIC_TYPE_REQ_3XX_COUNT,
		BUCKET_METRIC_TYPE_REQ_2XX_COUNT,

		BUCKET_METRIC_TYPE_STORAGE_SIZE,
	}

	ALL_K8S_NODE_TYPES = []TMetricType{
		K8S_NODE_METRIC_TYPE_CPU_USAGE,
		K8S_NODE_METRIC_TYPE_MEM_USAGE,
	}

	ALL_EIP_TYPES = []TMetricType{
		EIP_METRIC_TYPE_NET_BPS_RX,
		EIP_METRIC_TYPE_NET_BPS_TX,

		EIP_METRIC_TYPE_NET_DROP_SPEED_TX,
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

	OsType   string
	Interval int
	// rds
	Engine string

	// azure 内存,磁盘使用率监控需要查询table storage,会产生额外的存储费用，默认关闭
	IsSupportAzureTableStorageMetric bool
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
