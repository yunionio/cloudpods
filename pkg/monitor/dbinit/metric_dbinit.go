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

package dbinit

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/monitor"
)

var MetricNeedDeleteDescriptions = []string{}
var metricInitInputMap map[string]monitor.MetricCreateInput

func RegistryMetricCreateInput(name, displayName, resType, database string, score int,
	fields []monitor.MetricFieldCreateInput) {
	if metricInitInputMap == nil {
		metricInitInputMap = make(map[string]monitor.MetricCreateInput)
	}
	if _, ok := metricInitInputMap[name]; ok {
		log.Errorf("inputMeasurementName:%s has exist", name)
		return
	}
	metricInitInputMap[name] = monitor.MetricCreateInput{
		Measurement: monitor.MetricMeasurementCreateInput{
			StandaloneResourceCreateInput: apis.StandaloneResourceCreateInput{Name: name},
			ResType:                       resType,
			DisplayName:                   displayName,
			Database:                      database,
			Score:                         score,
		},
		MetricFields: fields,
	}
}

func GetRegistryMetricInput() (metricInitInputs []monitor.MetricCreateInput) {
	if metricInitInputMap == nil {
		metricInitInputMap = make(map[string]monitor.MetricCreateInput)
	}
	for name, _ := range metricInitInputMap {
		metricInitInputs = append(metricInitInputs, metricInitInputMap[name])
	}
	return
}

func newMetricFieldCreateInput(name, displayName, unit string, score int) monitor.MetricFieldCreateInput {
	return monitor.MetricFieldCreateInput{
		StandaloneResourceCreateInput: apis.StandaloneResourceCreateInput{Name: name},
		DisplayName:                   displayName,
		Unit:                          unit,
		ValueType:                     "",
		Score:                         score,
	}
}

// order by score asc
// score default:99
func init() {

	// cpu
	RegistryMetricCreateInput("cpu", "CPU usage", monitor.METRIC_RES_TYPE_HOST, monitor.METRIC_DATABASE_TELE, 1,
		[]monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("usage_active", "CPU active state utilization rate", monitor.METRIC_UNIT_PERCENT, 1),
			newMetricFieldCreateInput("usage_idle", "CPU idle state utilization rate", monitor.METRIC_UNIT_PERCENT, 2),
			newMetricFieldCreateInput("usage_system", "CPU system state utilization rate", monitor.METRIC_UNIT_PERCENT, 3),
			newMetricFieldCreateInput("usage_user", "CPU user mode utilization rate", monitor.METRIC_UNIT_PERCENT, 4),
			newMetricFieldCreateInput("usage_iowait", "CPU IO usage", monitor.METRIC_UNIT_PERCENT, 5),
			newMetricFieldCreateInput("usage_irq", "CPU IRQ usage", monitor.METRIC_UNIT_PERCENT, 6),
			newMetricFieldCreateInput("usage_guest", "CPU guest usage", monitor.METRIC_UNIT_PERCENT, 7),
			newMetricFieldCreateInput("usage_nice", "CPU priority switch utilization", monitor.METRIC_UNIT_PERCENT, 8),
			newMetricFieldCreateInput("usage_softirq", "CPU softirq usage", monitor.METRIC_UNIT_PERCENT, 9),
		})

	// disk
	RegistryMetricCreateInput("disk", "Disk usage", monitor.METRIC_RES_TYPE_HOST,
		monitor.METRIC_DATABASE_TELE, 3,
		[]monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("used_percent", "Percentage of used disks", monitor.METRIC_UNIT_PERCENT, 1),
			newMetricFieldCreateInput("free", "Free space size", monitor.METRIC_UNIT_BYTE, 2),
			newMetricFieldCreateInput("used", "Used disk size", monitor.METRIC_UNIT_BYTE, 3),
			newMetricFieldCreateInput("total", "Total disk size", monitor.METRIC_UNIT_BYTE, 4),
			newMetricFieldCreateInput("inodes_free", "Available inode", monitor.METRIC_UNIT_COUNT, 5),
			newMetricFieldCreateInput("inodes_used", "Number of inodes used", monitor.METRIC_UNIT_COUNT, 6),
			newMetricFieldCreateInput("inodes_total", "Total inodes", monitor.METRIC_UNIT_COUNT, 7),
		})

	// diskio
	RegistryMetricCreateInput("diskio", "Disk traffic and timing",
		monitor.METRIC_RES_TYPE_HOST, monitor.METRIC_DATABASE_TELE, 4, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("read_bps", "Disk read rate", monitor.METRIC_UNIT_BPS, 1),
			newMetricFieldCreateInput("write_bps", "Disk write rate", monitor.METRIC_UNIT_BPS, 2),
			newMetricFieldCreateInput("read_iops", "Disk read operate rate", monitor.METRIC_UNIT_COUNT, 3),
			newMetricFieldCreateInput("write_iops", "Disk write operate rate", monitor.METRIC_UNIT_COUNT, 4),
			newMetricFieldCreateInput("reads", "Number of reads", monitor.METRIC_UNIT_COUNT, 5),
			newMetricFieldCreateInput("writes", "Number of writes", monitor.METRIC_UNIT_COUNT, 6),
			newMetricFieldCreateInput("read_bytes", "Bytes read", monitor.METRIC_UNIT_BYTE, 7),
			newMetricFieldCreateInput("write_bytes", "Bytes write", monitor.METRIC_UNIT_BYTE, 8),
			newMetricFieldCreateInput("write_time", "Time to wait for write", monitor.METRIC_UNIT_MS, 9),
			newMetricFieldCreateInput("io_time", "I / O request queuing time", monitor.METRIC_UNIT_MS, 10),
			newMetricFieldCreateInput("weighted_io_time", "I / O request waiting time", monitor.METRIC_UNIT_MS, 11),
			newMetricFieldCreateInput("iops_in_progress", "Number of I / O requests issued but not yet completed", monitor.METRIC_UNIT_COUNT, 12),
		})

	// mem
	RegistryMetricCreateInput("mem", "Memory", monitor.METRIC_RES_TYPE_HOST,
		monitor.METRIC_DATABASE_TELE, 2, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("used_percent", "Used memory rate", monitor.METRIC_UNIT_PERCENT, 1),
			newMetricFieldCreateInput("available_percent", "Available memory rate", monitor.METRIC_UNIT_PERCENT, 2),
			newMetricFieldCreateInput("used", "Used memory", monitor.METRIC_UNIT_BYTE, 3),
			newMetricFieldCreateInput("free", "Free memory", monitor.METRIC_UNIT_BYTE, 4),
			newMetricFieldCreateInput("active", "The amount of active memory", monitor.METRIC_UNIT_BYTE, 5),
			newMetricFieldCreateInput("inactive", "The amount of inactive memory", monitor.METRIC_UNIT_BYTE, 6),
			newMetricFieldCreateInput("cached", "Cache memory", monitor.METRIC_UNIT_BYTE, 7),
			newMetricFieldCreateInput("buffered", "Buffer memory", monitor.METRIC_UNIT_BYTE, 7),
			newMetricFieldCreateInput("slab", "Number of kernel caches", monitor.METRIC_UNIT_BYTE, 8),
			newMetricFieldCreateInput("available", "Available memory", monitor.METRIC_UNIT_BYTE, 9),
			newMetricFieldCreateInput("total", "Total memory", monitor.METRIC_UNIT_BYTE, 10),
		})

	// net
	RegistryMetricCreateInput("net", "Network interface and protocol usage",
		monitor.METRIC_RES_TYPE_HOST, monitor.METRIC_DATABASE_TELE, 5, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("bytes_sent", "The total number of bytes sent by the network interface", monitor.METRIC_UNIT_BYTE, 1),
			newMetricFieldCreateInput("bytes_recv", "The total number of bytes received by the network interface", monitor.METRIC_UNIT_BYTE, 2),
			newMetricFieldCreateInput("packets_sent", "The total number of packets sent by the network interface", monitor.METRIC_UNIT_COUNT, 3),
			newMetricFieldCreateInput("packets_recv", "The total number of packets received by the network interface", monitor.METRIC_UNIT_COUNT, 4),
			newMetricFieldCreateInput("err_in", "The total number of receive errors detected by the network interface", monitor.METRIC_UNIT_COUNT, 5),
			newMetricFieldCreateInput("err_out", "The total number of transmission errors detected by the network interface", monitor.METRIC_UNIT_COUNT, 6),
			newMetricFieldCreateInput("drop_in", "The total number of received packets dropped by the network interface", monitor.METRIC_UNIT_COUNT, 7),
			newMetricFieldCreateInput("drop_out", "The total number of transmission packets dropped by the network interface", monitor.METRIC_UNIT_COUNT, 8),
		})

	// vm_cpu
	RegistryMetricCreateInput("vm_cpu", "Guest CPU usage", monitor.METRIC_RES_TYPE_GUEST,
		monitor.METRIC_DATABASE_TELE, 1, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("usage_active", "CPU active state utilization rate", monitor.METRIC_UNIT_PERCENT, 1),
			newMetricFieldCreateInput("cpu_usage_pcore", "CPU utilization rate per core", monitor.METRIC_UNIT_PERCENT, 2),
			newMetricFieldCreateInput("cpu_usage_idle_pcore", "CPU idle rate per core", monitor.METRIC_UNIT_PERCENT, 3),
			newMetricFieldCreateInput("cpu_time_system", "CPU system state time", monitor.METRIC_UNIT_MS, 4),
			newMetricFieldCreateInput("cpu_time_user", "CPU user state time", monitor.METRIC_UNIT_MS, 5),
			newMetricFieldCreateInput("thread_count", "The number of threads used by the process", monitor.METRIC_UNIT_COUNT, 6),
		})

	// vm_diskio
	RegistryMetricCreateInput("vm_diskio", "Guest disk traffic", monitor.METRIC_RES_TYPE_GUEST,
		monitor.METRIC_DATABASE_TELE, 3, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("read_bps", "Disk read rate", monitor.METRIC_UNIT_BYTEPS, 1),
			newMetricFieldCreateInput("write_bps", "Disk write rate", monitor.METRIC_UNIT_BYTEPS, 2),
			newMetricFieldCreateInput("read_iops", "Disk read operate rate", monitor.METRIC_UNIT_COUNT, 3),
			newMetricFieldCreateInput("write_iops", "Disk write operate rate", monitor.METRIC_UNIT_COUNT, 4),
			newMetricFieldCreateInput("read_bytes", "Bytes read", monitor.METRIC_UNIT_BYTE, 5),
			newMetricFieldCreateInput("write_bytes", "Bytes write", monitor.METRIC_UNIT_BYTE, 6),
		})

	// vm_mem
	RegistryMetricCreateInput("vm_mem", "Guest memory", monitor.METRIC_RES_TYPE_GUEST,
		monitor.METRIC_DATABASE_TELE, 2, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("used_percent", "Used memory rate", monitor.METRIC_UNIT_PERCENT, 1),
			newMetricFieldCreateInput("vms", "Virtual memory consumption", monitor.METRIC_UNIT_BYTE, 2),
			newMetricFieldCreateInput("rss", "Actual use of physical memory", monitor.METRIC_UNIT_BYTE, 3),
		})

	// vm_netio
	RegistryMetricCreateInput("vm_netio", "Guest network traffic", monitor.METRIC_RES_TYPE_GUEST,
		monitor.METRIC_DATABASE_TELE, 4, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("bps_recv", "Received traffic per second", monitor.METRIC_UNIT_BPS, 1),
			newMetricFieldCreateInput("bps_sent", "Send traffic per second", monitor.METRIC_UNIT_BPS, 2),
		})

	// oss_latency
	RegistryMetricCreateInput("oss_latency", "Object storage latency",
		monitor.METRIC_RES_TYPE_OSS, monitor.METRIC_DATABASE_TELE, 1, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("req_late", "Request average E2E delay", monitor.METRIC_UNIT_MS, 1),
		})

	// oss_netio
	RegistryMetricCreateInput("oss_netio", "Object storage network traffic",
		monitor.METRIC_RES_TYPE_OSS, monitor.METRIC_DATABASE_TELE, 2, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("bps_recv", "Receive byte", monitor.METRIC_UNIT_BYTE, 1),
			newMetricFieldCreateInput("bps_sent", "Send byte", monitor.METRIC_UNIT_BYTE, 2),
		})

	// oss_req
	RegistryMetricCreateInput("oss_req", "Object store request", monitor.METRIC_RES_TYPE_OSS,
		monitor.METRIC_DATABASE_TELE, 3, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("req_count", "request count", monitor.METRIC_UNIT_COUNT, 1),
		})

	// rds_conn
	RegistryMetricCreateInput("rds_conn", "Rds connect", monitor.METRIC_RES_TYPE_RDS,
		monitor.METRIC_DATABASE_TELE, 5, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("used_percent", "Connection usage", monitor.METRIC_UNIT_PERCENT, 1),
		})

	// rds_cpu
	RegistryMetricCreateInput("rds_cpu", "Rds CPU usage", monitor.METRIC_RES_TYPE_RDS,
		monitor.METRIC_DATABASE_TELE, 1, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("usage_active", "CPU active state utilization rate", monitor.METRIC_UNIT_PERCENT, 2),
		})

	// rds_mem
	RegistryMetricCreateInput("rds_mem", "Rds memory", monitor.METRIC_RES_TYPE_RDS,
		monitor.METRIC_DATABASE_TELE, 2, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("used_percent", "Used memory rate", monitor.METRIC_UNIT_PERCENT, 1),
		})

	// rds_netio
	RegistryMetricCreateInput("rds_netio", "Rds network traffic", monitor.METRIC_RES_TYPE_RDS,
		monitor.METRIC_DATABASE_TELE, 4, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("bps_recv", "Received traffic per second", monitor.METRIC_UNIT_BPS, 1),
			newMetricFieldCreateInput("bps_sent", "Send traffic per second", monitor.METRIC_UNIT_BPS, 2),
		})

	// rds_disk
	RegistryMetricCreateInput("rds_disk", "Rds disk usage", monitor.METRIC_RES_TYPE_RDS,
		monitor.METRIC_DATABASE_TELE, 3, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("used_percent", "Percentage of used disks", monitor.METRIC_UNIT_PERCENT, 1),
		})

	// dcs_cpu
	RegistryMetricCreateInput("dcs_cpu", "Redis CPU usage", monitor.METRIC_RES_TYPE_REDIS,
		monitor.METRIC_DATABASE_TELE, 1, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("usage_percent", "CPU active state utilization rate", monitor.METRIC_UNIT_PERCENT, 1),
		})

	// dcs_mem
	RegistryMetricCreateInput("dcs_mem", "Redis memory", monitor.METRIC_RES_TYPE_REDIS,
		monitor.METRIC_DATABASE_TELE, 2, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("used_percent", "Used memory rate", monitor.METRIC_UNIT_PERCENT, 1),
		})

	// dcs_netio
	RegistryMetricCreateInput("dcs_netio", "Redis network traffic",
		monitor.METRIC_RES_TYPE_REDIS,
		monitor.METRIC_DATABASE_TELE, 4, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("bps_recv", "Received traffic per second", monitor.METRIC_UNIT_BPS, 1),
			newMetricFieldCreateInput("bps_sent", "Send traffic per second", monitor.METRIC_UNIT_BPS, 2),
		})

	// dcs_conn
	RegistryMetricCreateInput("dcs_conn", "Redis connect", monitor.METRIC_RES_TYPE_REDIS,
		monitor.METRIC_DATABASE_TELE, 5, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("used_percent", "Connection usage", monitor.METRIC_UNIT_PERCENT, 1),
		})

	// dcs_instantopt
	RegistryMetricCreateInput("dcs_instantopt", "Redis operator",
		monitor.METRIC_RES_TYPE_REDIS, monitor.METRIC_DATABASE_TELE, 5, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("opt_sec", "Number of commands processed per second", monitor.METRIC_UNIT_COUNT, 1),
		})

	// dcs_cachekeys
	RegistryMetricCreateInput("dcs_cachekeys", "Redis keys", monitor.METRIC_RES_TYPE_REDIS,
		monitor.METRIC_DATABASE_TELE, 6, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("key_count", "Number of cache keys", monitor.METRIC_UNIT_COUNT, 1),
		})

	// dcs_datamem
	RegistryMetricCreateInput("dcs_datamem", "Redis data memory", monitor.METRIC_RES_TYPE_REDIS,
		monitor.METRIC_DATABASE_TELE, 3, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("used_byte", "Data node memory usage", monitor.METRIC_UNIT_BYTE, 1),
		})

	// cloudaccount_balance
	RegistryMetricCreateInput("cloudaccount_balance", "Cloud account balance",
		monitor.METRIC_RES_TYPE_CLOUDACCOUNT,
		monitor.METRIC_DATABASE_METER, 1, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("balance", "balance", monitor.METRIC_UNIT_RMB, 1),
		})

	// cpu
	RegistryMetricCreateInput("agent_cpu", "CPU usage", monitor.METRIC_RES_TYPE_AGENT, monitor.METRIC_DATABASE_TELE, 1,
		[]monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("usage_active", "CPU active state utilization rate", monitor.METRIC_UNIT_PERCENT, 1),
			newMetricFieldCreateInput("usage_idle", "CPU idle state utilization rate", monitor.METRIC_UNIT_PERCENT, 2),
			newMetricFieldCreateInput("usage_system", "CPU system state utilization rate", monitor.METRIC_UNIT_PERCENT, 3),
			newMetricFieldCreateInput("usage_user", "CPU user mode utilization rate", monitor.METRIC_UNIT_PERCENT, 4),
			newMetricFieldCreateInput("usage_iowait", "CPU IO usage", monitor.METRIC_UNIT_PERCENT, 5),
			newMetricFieldCreateInput("usage_irq", "CPU IRQ usage", monitor.METRIC_UNIT_PERCENT, 6),
			newMetricFieldCreateInput("usage_guest", "CPU guest usage", monitor.METRIC_UNIT_PERCENT, 7),
			newMetricFieldCreateInput("usage_nice", "CPU priority switch utilization", monitor.METRIC_UNIT_PERCENT, 8),
			newMetricFieldCreateInput("usage_softirq", "CPU softirq usage", monitor.METRIC_UNIT_PERCENT, 9),
		})

	// disk
	RegistryMetricCreateInput("agent_disk", "Disk usage", monitor.METRIC_RES_TYPE_AGENT,
		monitor.METRIC_DATABASE_TELE, 3,
		[]monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("used_percent", "Percentage of used disks", monitor.METRIC_UNIT_PERCENT, 1),
			newMetricFieldCreateInput("free", "Free space size", monitor.METRIC_UNIT_BYTE, 2),
			newMetricFieldCreateInput("used", "Used disk size", monitor.METRIC_UNIT_BYTE, 3),
			newMetricFieldCreateInput("total", "Total disk size", monitor.METRIC_UNIT_BYTE, 4),
			newMetricFieldCreateInput("inodes_free", "Available inode", monitor.METRIC_UNIT_COUNT, 5),
			newMetricFieldCreateInput("inodes_used", "Number of inodes used", monitor.METRIC_UNIT_COUNT, 6),
			newMetricFieldCreateInput("inodes_total", "Total inodes", monitor.METRIC_UNIT_COUNT, 7),
		})

	// diskio
	RegistryMetricCreateInput("agent_diskio", "Disk traffic and timing",
		monitor.METRIC_RES_TYPE_AGENT, monitor.METRIC_DATABASE_TELE, 4, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("read_bps", "Disk read rate", monitor.METRIC_UNIT_BPS, 1),
			newMetricFieldCreateInput("write_bps", "Disk write rate", monitor.METRIC_UNIT_BPS, 2),
			newMetricFieldCreateInput("read_iops", "Disk read operate rate", monitor.METRIC_UNIT_COUNT, 3),
			newMetricFieldCreateInput("write_iops", "Disk write operate rate", monitor.METRIC_UNIT_COUNT, 4),
			newMetricFieldCreateInput("reads", "Number of reads", monitor.METRIC_UNIT_COUNT, 5),
			newMetricFieldCreateInput("writes", "Number of writes", monitor.METRIC_UNIT_COUNT, 6),
			newMetricFieldCreateInput("read_bytes", "Bytes read", monitor.METRIC_UNIT_BYTE, 7),
			newMetricFieldCreateInput("write_bytes", "Bytes write", monitor.METRIC_UNIT_BYTE, 8),
			newMetricFieldCreateInput("write_time", "Time to wait for write", monitor.METRIC_UNIT_MS, 9),
			newMetricFieldCreateInput("io_time", "I / O request queuing time", monitor.METRIC_UNIT_MS, 10),
			newMetricFieldCreateInput("weighted_io_time", "I / O request waiting time", monitor.METRIC_UNIT_MS, 11),
			newMetricFieldCreateInput("iops_in_progress", "Number of I / O requests issued but not yet completed", monitor.METRIC_UNIT_COUNT, 12),
		})

	// mem
	RegistryMetricCreateInput("agent_mem", "Memory", monitor.METRIC_RES_TYPE_AGENT,
		monitor.METRIC_DATABASE_TELE, 2, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("used_percent", "Used memory rate", monitor.METRIC_UNIT_PERCENT, 1),
			newMetricFieldCreateInput("available_percent", "Available memory rate", monitor.METRIC_UNIT_PERCENT, 2),
			newMetricFieldCreateInput("used", "Used memory", monitor.METRIC_UNIT_BYTE, 3),
			newMetricFieldCreateInput("free", "Free memory", monitor.METRIC_UNIT_BYTE, 4),
			newMetricFieldCreateInput("active", "The amount of active memory", monitor.METRIC_UNIT_BYTE, 5),
			newMetricFieldCreateInput("inactive", "The amount of inactive memory", monitor.METRIC_UNIT_BYTE, 6),
			newMetricFieldCreateInput("cached", "Cache memory", monitor.METRIC_UNIT_BYTE, 7),
			newMetricFieldCreateInput("buffered", "Buffer memory", monitor.METRIC_UNIT_BYTE, 7),
			newMetricFieldCreateInput("slab", "Number of kernel caches", monitor.METRIC_UNIT_BYTE, 8),
			newMetricFieldCreateInput("available", "Available memory", monitor.METRIC_UNIT_BYTE, 9),
			newMetricFieldCreateInput("total", "Total memory", monitor.METRIC_UNIT_BYTE, 10),
		})

	// net
	RegistryMetricCreateInput("agent_net", "Network interface and protocol usage",
		monitor.METRIC_RES_TYPE_AGENT, monitor.METRIC_DATABASE_TELE, 5, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("bytes_sent", "The total number of bytes sent by the network interface", monitor.METRIC_UNIT_BYTE, 1),
			newMetricFieldCreateInput("bytes_recv", "The total number of bytes received by the network interface", monitor.METRIC_UNIT_BYTE, 2),
			newMetricFieldCreateInput("packets_sent", "The total number of packets sent by the network interface", monitor.METRIC_UNIT_COUNT, 3),
			newMetricFieldCreateInput("packets_recv", "The total number of packets received by the network interface", monitor.METRIC_UNIT_COUNT, 4),
			newMetricFieldCreateInput("err_in", "The total number of receive errors detected by the network interface", monitor.METRIC_UNIT_COUNT, 5),
			newMetricFieldCreateInput("err_out", "The total number of transmission errors detected by the network interface", monitor.METRIC_UNIT_COUNT, 6),
			newMetricFieldCreateInput("drop_in", "The total number of received packets dropped by the network interface", monitor.METRIC_UNIT_COUNT, 7),
			newMetricFieldCreateInput("drop_out", "The total number of transmission packets dropped by the network interface", monitor.METRIC_UNIT_COUNT, 8),
		})

	RegistryMetricCreateInput("storage", "Storage usage",
		monitor.METRIC_RES_TYPE_STORAGE, monitor.METRIC_DATABASE_TELE, 1, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("usage_active", "Storage utilization rate", monitor.METRIC_UNIT_PERCENT, 1),
			newMetricFieldCreateInput("free", "Free storage", monitor.METRIC_UNIT_MB, 2),
		})

	//jenkins
	RegistryMetricCreateInput("jenkins_node", "jenkins node",
		monitor.METRIC_RES_TYPE_JENKINS, monitor.METRIC_DATABASE_TELE, 1, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("disk_available", "disk_available", monitor.METRIC_UNIT_BYTE, 1),
			newMetricFieldCreateInput("temp_available", "temp_available", monitor.METRIC_UNIT_BYTE, 2),
			newMetricFieldCreateInput("memory_available", "memory_available", monitor.METRIC_UNIT_BYTE, 3),
			newMetricFieldCreateInput("memory_total", "memory_total", monitor.METRIC_UNIT_BYTE, 4),
			newMetricFieldCreateInput("swap_available", "swap_available", monitor.METRIC_UNIT_BYTE, 5),
			newMetricFieldCreateInput("swap_total", "swap_total", monitor.METRIC_UNIT_BYTE, 6),
		})
	RegistryMetricCreateInput("jenkins_job", "jenkins job",
		monitor.METRIC_RES_TYPE_JENKINS, monitor.METRIC_DATABASE_TELE, 2, []monitor.MetricFieldCreateInput{
			newMetricFieldCreateInput("duration", "duration", monitor.METRIC_UNIT_MS, 1),
			newMetricFieldCreateInput("number", "number", monitor.METRIC_UNIT_COUNT, 2),
		})

}
