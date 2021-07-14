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

package jdmon

const (
	PERIOD             = 60
	UNIT_AVERAGE       = "Average"
	DEFAULT_STATISTICS = "Average,Minimum,Maximum"
	UNIT_PERCENT       = "Percent"
	UNIT_BPS           = "bps"
	UNIT_KBPS          = "Kbps"
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
	INFLUXDB_FIELD_RDS_CPU_USAGE             = "rds_cpu.usage_active"
	INFLUXDB_FIELD_RDS_MEM_USAGE             = "rds_mem.used_percent"
	INFLUXDB_FIELD_RDS_NET_BPS_RX            = "rds_netio.bps_recv"
	INFLUXDB_FIELD_RDS_NET_BPS_RX_PERCONA    = INFLUXDB_FIELD_RDS_NET_BPS_RX + ",server_type=percona"
	INFLUXDB_FIELD_RDS_NET_BPS_RX_MARIADB    = INFLUXDB_FIELD_RDS_NET_BPS_RX + ",server_type=mariadb"
	INFLUXDB_FIELD_RDS_NET_BPS_RX_POSTGRESQL = INFLUXDB_FIELD_RDS_NET_BPS_RX + ",server_type=postgresql"
	INFLUXDB_FIELD_RDS_NET_BPS_RX_MYSQL      = INFLUXDB_FIELD_RDS_NET_BPS_RX + ",server_type=mysql"
	INFLUXDB_FIELD_RDS_NET_BPS_RX_SQLSERVER  = INFLUXDB_FIELD_RDS_NET_BPS_RX + ",server_type=sqlserver"
	INFLUXDB_FIELD_RDS_NET_BPS_TX            = "rds_netio.bps_sent"
	INFLUXDB_FIELD_RDS_NET_BPS_TX_PERCONA    = INFLUXDB_FIELD_RDS_NET_BPS_TX + ",server_type=percona"
	INFLUXDB_FIELD_RDS_NET_BPS_TX_MARIADB    = INFLUXDB_FIELD_RDS_NET_BPS_TX + ",server_type=mariadb"
	INFLUXDB_FIELD_RDS_NET_BPS_TX_POSTGRESQL = INFLUXDB_FIELD_RDS_NET_BPS_TX + ",server_type=postgresql"
	INFLUXDB_FIELD_RDS_NET_BPS_TX_MYSQL      = INFLUXDB_FIELD_RDS_NET_BPS_TX + ",server_type=mysql"
	INFLUXDB_FIELD_RDS_NET_BPS_TX_SQLSERVER  = INFLUXDB_FIELD_RDS_NET_BPS_TX + ",server_type=sqlserver"
	INFLUXDB_FIELD_RDS_DISK_USAGE            = "rds_disk.used_percent"
	INFLUXDB_FIELD_RDS_DISK_READ_BPS         = "rds_diskio.read_bps"
	INFLUXDB_FIELD_RDS_DISK_WRITE_BPS        = "rds_diskio.write_bps"
	INFLUXDB_FIELD_RDS_CONN_COUNT            = "rds_conn.used_count"
	INFLUXDB_FIELD_RDS_CONN_USAGE            = "rds_conn.used_percent"

	INFLUXDB_FIELD_RDS_QPS              = "rds_qps.query_qps"
	INFLUXDB_FIELD_RDS_TPS              = "rds_tps.trans_qps"
	INFLUXDB_FIELD_RDS_INNODB_REDA_BPS  = "rds_innodb.read_bps"
	INFLUXDB_FIELD_RDS_INNODB_WRITE_BPS = "rds_innodb.write_bps"

	KEY_VMS   = "vms"
	KEY_CPUS  = "cpus"
	KEY_MEMS  = "mems"
	KEY_DISKS = "disks"
)

var (
	jdMetricSpecs = map[string][]string{
		"cpu_util":                 {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_CPU_USAGE},     //CPU使用率，该指标用于统计测量对象的CPU使用率，以百分为单位。
		"memory.usage":             {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_MEM_USAGE},     //CPU使用率，该指标用于统计测量对象的CPU使用率，以百分为单位。
		"vm.disk.dev.used":         {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_MEM_USAGE},     //CPU使用率，该指标用于统计测量对象的CPU使用率，以百分为单位。
		"vm.network.dev.bytes.in":  {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_NET_BPS_RX},     //带外网络流入速率，该指标用于在虚拟化层统计每秒流入测量对象的网络流量，以字节/秒为单位。
		"vm.network.dev.bytes.out": {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_NET_BPS_TX},     //带外网络流出速率，该指标用于在虚拟化层统计每秒流出测量对象的网络流量，以字节/秒为单位。
		"vm.disk.dev.bytes.read":   {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_DISK_READ_BPS},  //磁盘读速率，该指标用于统计每秒从测量对象读出数据量，以字节/秒为单位。
		"vm.disk.dev.bytes.write":  {DEFAULT_STATISTICS, UNIT_BYTEPS, INFLUXDB_FIELD_DISK_WRITE_BPS}, //磁盘写速率，该指标用于统计每秒写到测量对象的数据量，以字节/秒为单位。
		"vm.disk.dev.io.read":      {DEFAULT_STATISTICS, UNIT_CPS, INFLUXDB_FIELD_DISK_READ_IOPS},    //该指标用于统计每秒从测量对象读取数据的请求次数，以请求/秒为单位。
		"vm.disk.dev.io.write":     {DEFAULT_STATISTICS, UNIT_CPS, INFLUXDB_FIELD_DISK_WRITE_IOPS},   //该指标用于统计每秒从测量对象写数据的请求次数，以请求/秒为单位
	}

	jdRdsSqlserverMetricSpecs = map[string][]string{
		"database.sqlserver.kvm.cpu.util":               {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_CPU_USAGE},
		"database.sqlserver.kvm.memory.usage":           {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_MEM_USAGE},
		"database.sqlserver.kvm.disk1.usedpercent":      {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_DISK_USAGE},
		"database.sqlserver.kvm.network.bytes.incoming": {DEFAULT_STATISTICS, UNIT_KBPS, INFLUXDB_FIELD_RDS_NET_BPS_RX_SQLSERVER},
		"database.sqlserver.kvm.network.bytes.outgoing": {DEFAULT_STATISTICS, UNIT_KBPS, INFLUXDB_FIELD_RDS_NET_BPS_TX_SQLSERVER},
	}
	jdRdsMysqlMetricSpecs = map[string][]string{
		"database.docker.cpu.util":         {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_CPU_USAGE},
		"database.docker.memory.pused":     {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_MEM_USAGE},
		"database.docker.disk1.used":       {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_DISK_USAGE},
		"database.docker.network.incoming": {DEFAULT_STATISTICS, UNIT_KBPS, INFLUXDB_FIELD_RDS_NET_BPS_RX_MYSQL},
		"database.docker.network.outgoing": {DEFAULT_STATISTICS, UNIT_KBPS, INFLUXDB_FIELD_RDS_NET_BPS_TX_MYSQL},
	}
	jdRdsPerconaMetricSpecs = map[string][]string{
		"database.docker.cpu.util":         {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_CPU_USAGE},
		"database.docker.memory.pused":     {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_MEM_USAGE},
		"database.docker.disk1.used":       {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_DISK_USAGE},
		"database.docker.network.incoming": {DEFAULT_STATISTICS, UNIT_KBPS, INFLUXDB_FIELD_RDS_NET_BPS_RX_PERCONA},
		"database.docker.network.outgoing": {DEFAULT_STATISTICS, UNIT_KBPS, INFLUXDB_FIELD_RDS_NET_BPS_TX_PERCONA},
	}
	jdRdsMariadbMetricSpecs = map[string][]string{
		"database.docker.cpu.util":         {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_CPU_USAGE},
		"database.docker.memory.pused":     {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_MEM_USAGE},
		"database.docker.disk1.used":       {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_DISK_USAGE},
		"database.docker.network.incoming": {DEFAULT_STATISTICS, UNIT_KBPS, INFLUXDB_FIELD_RDS_NET_BPS_RX_MARIADB},
		"database.docker.network.outgoing": {DEFAULT_STATISTICS, UNIT_KBPS, INFLUXDB_FIELD_RDS_NET_BPS_TX_MARIADB},
	}
	jdRdsPostgresqlMetricSpecs = map[string][]string{
		"database.docker.cpu.util":               {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_CPU_USAGE},
		"database.docker.memory.pused":           {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_MEM_USAGE},
		"database.docker.disk1.used":             {DEFAULT_STATISTICS, UNIT_PERCENT, INFLUXDB_FIELD_RDS_DISK_USAGE},
		"database.docker.network.bytes.incoming": {DEFAULT_STATISTICS, UNIT_BPS, INFLUXDB_FIELD_RDS_NET_BPS_RX_POSTGRESQL},
		"database.docker.network.bytes.outgoing": {DEFAULT_STATISTICS, UNIT_BPS, INFLUXDB_FIELD_RDS_NET_BPS_TX_POSTGRESQL},
	}
)
