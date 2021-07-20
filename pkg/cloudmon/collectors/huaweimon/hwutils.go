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

package huaweimon

const (
	PERIOD             = 60
	UNIT_AVERAGE       = "Average"
	DEFAULT_STATISTICS = "Average,Minimum,Maximum"
	UNIT_PERCENT       = "Percent"
	UNIT_BPS           = "bps"
	UNIT_MBPS          = "Mbps"
	UNIT_BYTEPS        = "Bps"
	UNIT_CPS           = "cps"
	UNIT_COUNT         = "count"
	UNIT_MEM           = "byte"
	UNIT_MSEC          = "ms"
	UNIT_COUNT_SEC     = "count/s"

	//ESC监控指标
	INFLUXDB_FIELD_CPU_USAGE           = "vm_cpu.usage_active"
	INFLUXDB_FIELD_MEM_USAGE           = "vm_mem.used_percent"
	INFLUXDB_FIELD_DISK_READ_BPS       = "vm_diskio.read_bps"
	INFLUXDB_FIELD_DISK_WRITE_BPS      = "vm_diskio.write_bps"
	INFLUXDB_FIELD_DISK_READ_IOPS      = "vm_diskio.read_iops"
	INFLUXDB_FIELD_DISK_WRITE_IOPS     = "vm_diskio.write_iops"
	INFLUXDB_FIELD_NET_BPS_RX          = "vm_netio.bps_recv"
	INFLUXDB_FIELD_NET_BPS_RX_INTERNET = INFLUXDB_FIELD_NET_BPS_RX + ",net_type=internet"
	INFLUXDB_FIELD_NET_BPS_RX_INTRANET = INFLUXDB_FIELD_NET_BPS_RX + ",net_type=intranet"
	INFLUXDB_FIELD_NET_BPS_TX          = "vm_netio.bps_sent"
	INFLUXDB_FIELD_NET_BPS_TX_INTERNET = INFLUXDB_FIELD_NET_BPS_TX + ",net_type=internet"
	INFLUXDB_FIELD_NET_BPS_TX_INTRANET = INFLUXDB_FIELD_NET_BPS_TX + ",net_type=intranet"
	INFLUXDB_FIELD_WANOUTTRAFFIC       = "vm_eipio.bps_out"
	INFLUXDB_FIELD_WANINTRAFFIC        = "vm_eipio.bps_in"
	INFLUXDB_FIELD_WANOUTPKG           = "vm_eipio.pps_out"
	INFLUXDB_FIELD_WANINPKG            = "vm_eipio.pps_in"

	//RDS监控指标
	INFLUXDB_FIELD_RDS_CPU_USAGE            = "rds_cpu.usage_active"
	INFLUXDB_FIELD_RDS_MEM_USAGE            = "rds_mem.used_percent"
	INFLUXDB_FIELD_RDS_NET_BPS_RX           = "rds_netio.bps_recv"
	INFLUXDB_FIELD_RDS_NET_BPS_RX_MYSQL     = INFLUXDB_FIELD_RDS_NET_BPS_RX + ",server_type=mysql"
	INFLUXDB_FIELD_RDS_NET_BPS_RX_SQLSERVER = INFLUXDB_FIELD_RDS_NET_BPS_RX + ",server_type=sqlserver"
	INFLUXDB_FIELD_RDS_NET_BPS_TX           = "rds_netio.bps_sent"
	INFLUXDB_FIELD_RDS_NET_BPS_TX_MYSQL     = INFLUXDB_FIELD_RDS_NET_BPS_TX + ",server_type=mysql"
	INFLUXDB_FIELD_RDS_NET_BPS_TX_SQLSERVER = INFLUXDB_FIELD_RDS_NET_BPS_TX + ",server_type=sqlserver"
	INFLUXDB_FIELD_RDS_DISK_USAGE           = "rds_disk.used_percent"
	INFLUXDB_FIELD_RDS_DISK_READ_BPS        = "rds_diskio.read_bps"
	INFLUXDB_FIELD_RDS_DISK_WRITE_BPS       = "rds_diskio.write_bps"
	INFLUXDB_FIELD_RDS_CONN_COUNT           = "rds_conn.used_count"
	INFLUXDB_FIELD_RDS_CONN_USAGE           = "rds_conn.used_percent"

	INFLUXDB_FIELD_RDS_QPS              = "rds_qps.query_qps"
	INFLUXDB_FIELD_RDS_TPS              = "rds_tps.trans_qps"
	INFLUXDB_FIELD_RDS_INNODB_REDA_BPS  = "rds_innodb.read_bps"
	INFLUXDB_FIELD_RDS_INNODB_WRITE_BPS = "rds_innodb.write_bps"

	//REDIS监控指标
	INFLUXDB_FIELD_REDIS_CPU_USAGE      = "dcs_cpu.usage_percent"
	INFLUXDB_FIELD_REDIS_MEM_USAGE      = "dcs_mem.used_percent"
	INFLUXDB_FIELD_REDIS_NET_BPS_RX     = "dcs_netio.bps_recv"
	INFLUXDB_FIELD_REDIS_NET_BPS_TX     = "dcs_netio.bps_sent"
	INFLUXDB_FIFLD_REDIS_CONN_USAGE     = "dcs_conn.used_conn"
	INFLUXDB_FIFLD_REDIS_OPT_SES        = "dcs_instantopt.opt_sec"
	INFLUXDB_FIFLD_REDIS_CACHE_KEYS     = "dcs_cachekeys.key_count"
	INFLUXDB_FIFLD_REDIS_CACHE_EXP_KEYS = INFLUXDB_FIFLD_REDIS_CACHE_KEYS + ",exp=expire"
	INFLUXDB_FIFLD_REDIS_DATA_MEM_USAGE = "dcs_datamem.used_byte"

	//对象存储OSS监控指标
	INFLUXDB_FIELD_OSS_NET_BPS_RX          = "oss_netio.bps_recv"
	INFLUXDB_FIELD_OSS_NET_BPS_RX_INTERNET = INFLUXDB_FIELD_OSS_NET_BPS_RX + ",net_type=internet"
	INFLUXDB_FIELD_OSS_NET_BPS_RX_INTRANET = INFLUXDB_FIELD_OSS_NET_BPS_RX + ",net_type=intranet"
	INFLUXDB_FIELD_OSS_NET_BPS_TX          = "oss_netio.bps_sent"
	INFLUXDB_FIELD_OSS_NET_BPS_TX_INTERNET = INFLUXDB_FIELD_OSS_NET_BPS_TX + ",net_type=internet"
	INFLUXDB_FIELD_OSS_NET_BPS_TX_INTRANET = INFLUXDB_FIELD_OSS_NET_BPS_TX + ",net_type=intranet"
	INFLUXDB_FIELD_OSS_LATECY              = "oss_latency.req_late"
	INFLUXDB_FIELD_OSS_LATECY_GET          = INFLUXDB_FIELD_OSS_LATECY + ",request=get"
	INFLUXDB_FIELD_OSS_LATECY_POST         = INFLUXDB_FIELD_OSS_LATECY + ",request=post"
	INFLUXDB_FIELD_OSS_REQ_COUNT           = "oss_req.req_count"
	INFLUXDB_FIELD_OSS_REQ_COUNT_GET       = INFLUXDB_FIELD_OSS_REQ_COUNT + ",request=get"
	INFLUXDB_FIELD_OSS_REQ_COUNT_POST      = INFLUXDB_FIELD_OSS_REQ_COUNT + ",request=post"
	INFLUXDB_FIELD_OSS_REQ_COUNT_5XX       = INFLUXDB_FIELD_OSS_REQ_COUNT + ",request=5xx"
	INFLUXDB_FIELD_OSS_REQ_COUNT_4XX       = INFLUXDB_FIELD_OSS_REQ_COUNT + ",request=4xx"

	//负载均衡监控指标
	INFLUXDB_FIELD_ELB_NET_BPS_RX     = "haproxy.bin"
	INFLUXDB_FIELD_ELB_NET_BPS_TX     = "haproxy.bout"
	INFLUXDB_FIELD_ELB_REQ_RATE       = "haproxy.req_rate,request=http"
	INFLUXDB_FIELD_ELB_CONN_RATE      = "haproxy.conn_rate,request=tcp"
	INFLUXDB_FIELD_ELB_DREQ_COUNT     = "haproxy.dreq,request=http"
	INFLUXDB_FIELD_ELB_DCONN_COUNT    = "haproxy.dcon,request=tcp"
	INFLUXDB_FIELD_ELB_HRSP_COUNT     = "haproxy.hrsp_Nxx"
	INFLUXDB_FIELD_ELB_HRSP_COUNT_2XX = INFLUXDB_FIELD_ELB_HRSP_COUNT + ",request=2xx"
	INFLUXDB_FIELD_ELB_HRSP_COUNT_3XX = INFLUXDB_FIELD_ELB_HRSP_COUNT + ",request=3xx"
	INFLUXDB_FIELD_ELB_HRSP_COUNT_4XX = INFLUXDB_FIELD_ELB_HRSP_COUNT + ",request=4xx"
	INFLUXDB_FIELD_ELB_HRSP_COUNT_5XX = INFLUXDB_FIELD_ELB_HRSP_COUNT + ",request=5xx"
	INFLUXDB_FIELD_ELB_CHC_STATUS     = "haproxy.check_status"
	INFLUXDB_FIELD_ELB_CHC_CODE       = "haproxy.check_code"
	INFLUXDB_FIELD_ELB_LAST_CHC       = "haproxy.last_chk"

	KEY_VMS   = "vms"
	KEY_CPUS  = "cpus"
	KEY_MEMS  = "mems"
	KEY_DISKS = "disks"
)

var huaweiMetricSpecs = map[string][]string{
	"cpu_util":                              {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_CPU_USAGE},          //CPU使用率，该指标用于统计测量对象的CPU使用率，以百分为单位。
	"network_incoming_bytes_aggregate_rate": {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_NET_BPS_RX_INTERNET}, //带外网络流入速率，该指标用于在虚拟化层统计每秒流入测量对象的网络流量，以字节/秒为单位。
	"network_outgoing_bytes_aggregate_rate": {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_NET_BPS_TX_INTERNET}, //带外网络流出速率，该指标用于在虚拟化层统计每秒流出测量对象的网络流量，以字节/秒为单位。
	"disk_read_bytes_rate":                  {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_DISK_READ_BPS},       //磁盘读速率，该指标用于统计每秒从测量对象读出数据量，以字节/秒为单位。
	"disk_write_bytes_rate":                 {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_DISK_WRITE_BPS},      //磁盘写速率，该指标用于统计每秒写到测量对象的数据量，以字节/秒为单位。
	"disk_read_requests_rate":               {DEFAULT_STATISTICS, UNIT_CPS, INFLUXDB_FIELD_DISK_READ_IOPS},         //该指标用于统计每秒从测量对象读取数据的请求次数，以请求/秒为单位。
	"disk_write_requests_rate":              {DEFAULT_STATISTICS, UNIT_CPS, INFLUXDB_FIELD_DISK_WRITE_IOPS},        //该指标用于统计每秒从测量对象写数据的请求次数，以请求/秒为单位
}

var huaweiRdsMetricSpecs = map[string][]string{
	"rds001_cpu_util":             {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_CPU_USAGE},
	"rds002_mem_util":             {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_MEM_USAGE},
	"rds004_bytes_in":             {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_RDS_NET_BPS_RX},
	"rds005_bytes_out":            {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_RDS_NET_BPS_TX},
	"rds039_disk_util":            {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_DISK_USAGE},
	"rds049_disk_read_throughput": {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_RDS_DISK_READ_BPS},

	"rds050_disk_write_throughput": {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_RDS_DISK_WRITE_BPS},

	"rds006_conn_count": {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_CONN_COUNT},

	"rds008_qps": {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_QPS},
	"rds009_tps": {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_TPS},

	"rds013_innodb_reads":  {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_RDS_INNODB_REDA_BPS},
	"rds014_innodb_writes": {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_RDS_INNODB_WRITE_BPS},
}

var huaweiRedisMetricSpecs = map[string][]string{
	"cpu_usage":                 {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_REDIS_CPU_USAGE},
	"memory_usage":              {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_REDIS_MEM_USAGE},
	"instantaneous_input_kbps":  {DEFAULT_STATISTICS, UNIT_BPS, INFLUXDB_FIELD_REDIS_NET_BPS_RX},
	"instantaneous_output_kbps": {DEFAULT_STATISTICS, UNIT_BPS, INFLUXDB_FIELD_REDIS_NET_BPS_TX},
	"connected_clients":         {DEFAULT_STATISTICS, UNIT_COUNT, INFLUXDB_FIFLD_REDIS_CONN_USAGE},
	"instantaneous_ops":         {DEFAULT_STATISTICS, UNIT_COUNT, INFLUXDB_FIFLD_REDIS_OPT_SES},
	"keys":                      {DEFAULT_STATISTICS, UNIT_COUNT, INFLUXDB_FIFLD_REDIS_CACHE_KEYS},
	"expires":                   {DEFAULT_STATISTICS, UNIT_COUNT, INFLUXDB_FIFLD_REDIS_CACHE_EXP_KEYS},
	"used_memory_dataset":       {DEFAULT_STATISTICS, UNIT_MEM, INFLUXDB_FIFLD_REDIS_DATA_MEM_USAGE},
}

var huaweiOSSMetricSpecs = map[string][]string{
	"download_bytes":     {DEFAULT_STATISTICS, UNIT_MEM, INFLUXDB_FIELD_OSS_NET_BPS_TX},
	"upload_bytes":       {DEFAULT_STATISTICS, UNIT_MEM, INFLUXDB_FIELD_OSS_NET_BPS_RX},
	"first_byte_latency": {DEFAULT_STATISTICS, UNIT_MSEC, INFLUXDB_FIELD_OSS_LATECY_GET},
	"get_request_count":  {DEFAULT_STATISTICS, UNIT_COUNT, INFLUXDB_FIELD_OSS_REQ_COUNT_GET},
	"request_count_4xx":  {DEFAULT_STATISTICS, UNIT_COUNT, INFLUXDB_FIELD_OSS_REQ_COUNT_4XX},
	"request_count_5xx":  {DEFAULT_STATISTICS, UNIT_COUNT, INFLUXDB_FIELD_OSS_REQ_COUNT_5XX},
}

var huaweiElbMetricSpecs = map[string][]string{
	"m7_in_Bps":      {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_ELB_NET_BPS_RX},
	"m8_out_Bps":     {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_ELB_NET_BPS_TX},
	"mb_l7_qps":      {DEFAULT_STATISTICS, UNIT_COUNT_SEC, INFLUXDB_FIELD_ELB_REQ_RATE},
	"mc_l7_http_2xx": {DEFAULT_STATISTICS, UNIT_COUNT_SEC, INFLUXDB_FIELD_ELB_HRSP_COUNT_2XX},
	"md_l7_http_3xx": {DEFAULT_STATISTICS, UNIT_COUNT_SEC, INFLUXDB_FIELD_ELB_HRSP_COUNT_3XX},
	"me_l7_http_4xx": {DEFAULT_STATISTICS, UNIT_COUNT_SEC, INFLUXDB_FIELD_ELB_HRSP_COUNT_4XX},
	"mf_l7_http_5xx": {DEFAULT_STATISTICS, UNIT_COUNT_SEC, INFLUXDB_FIELD_ELB_HRSP_COUNT_5XX},
}
