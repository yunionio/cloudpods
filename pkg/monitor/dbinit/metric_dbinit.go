package dbinit

var MetricDescriptions = `
[
	{
		"measurement": {
			"name": "cpu",
			"display_name": "CPU usage",
			"database":"telegraf",
			"res_type":"host"
		},
		"metric_fields": [
			{
				"name":"usage_active",
				"display_name":"CPU active state utilization rate",
				"unit":"%"
			},
			{
				"name":"usage_guest",
				"display_name":"CPU guest usage",
				"unit":"%"
			},
			{
				"name":"usage_idle",
				"display_name":"CPU idle state utilization rate",
				"unit":"%"
			},
			{
				"name":"usage_iowait",
				"display_name":"CPU IO usage",
				"unit":"%"
			},
			{
				"name":"usage_irq",
				"display_name":"CPU IRQ usage",
				"unit":"%"
			},
			{
				"name":"usage_nice",
				"display_name":"CPU priority switch utilization",
				"unit":"%"
			},
			{
				"name":"usage_softirq",
				"display_name":"CPU softirq usage",
				"unit":"%"
			},
			{
				"name":"usage_system",
				"display_name":"CPU system state utilization rate",
				"unit":"%"
			},
			{
				"name":"usage_user",
				"display_name":"CPU user mode utilization rate",
				"unit":"%"
			}
		]
	},
	{
		"measurement": {
			"name": "disk",
			"display_name": "Disk usage",
			"database":"telegraf",
			"res_type":"host"
		},
		"metric_fields": [
			{
				"name":"free",
				"display_name":"Free space size",
				"unit":"byte"
			},
			{
				"name":"inodes_free",
				"display_name":"Available inode",
				"unit":"count"
			},
			{
				"name":"inodes_total",
				"display_name":"Total inodes",
				"unit":"count"
			},
			{
				"name":"inodes_used",
				"display_name":"Number of inodes used",
				"unit":"count"
			},
			{
				"name":"total",
				"display_name":"Total disk size",
				"unit":"byte"
			},
			{
				"name":"used",
				"display_name":"Used disk size",
				"unit":"byte"
			},
			{
				"name":"used_percent",
				"display_name":"Percentage of used disks",
				"unit":"%"
			}
		]
	},
	{
		"measurement": {
			"name": "diskio",
			"display_name": "Disk traffic and timing",
			"database":"telegraf",
			"res_type":"host"
		},
		"metric_fields": [
			{
				"name":"reads",
				"display_name":"Number of reads",
				"unit":"count"
			},
			{
				"name":"writes",
				"display_name":"Number of writes",
				"unit":"count"
			},
			{
				"name":"read_bytes",
				"display_name":"Bytes read",
				"unit":"byte"
			},
			{
				"name":"write_bytes",
				"display_name":"Bytes write",
				"unit":"byte"
			},
			{
				"name":"write_time",
				"display_name":"Time to wait for write",
				"unit":"ms"
			},
			{
				"name":"io_time",
				"display_name":"I / O request queuing time",
				"unit":"ms"
			},
			{
				"name":"weighted_io_time",
				"display_name":"I / O request waiting time",
				"unit":"ms"
			},
			{
				"name":"iops_in_progress",
				"display_name":"Number of I / O requests issued but not yet completed",
				"unit":"count"
			},
			{
				"name":"read_bps",
				"display_name":"Disk read rate",
				"unit":"bps"
			},
			{
				"name":"write_bps",
				"display_name":"Disk write rate",
				"unit":"bps"
			},
			{
				"name":"read_iops",
				"display_name":"Disk read operate rate",
				"unit":"count"
			},
			{
				"name":"write_iops",
				"display_name":"Disk write operate rate",
				"unit":"count"
			},
		]
	},
	{
		"measurement": {
			"name": "mem",
			"display_name": "Memory",
			"database":"telegraf",
			"res_type":"host"
		},
		"metric_fields": [
			{
				"name":"active",
				"display_name":"The amount of active memory",
				"unit":"byte"
			},
			{
				"name":"available",
				"display_name":"Available memory",
				"unit":"byte"
			},
			{
				"name":"available_percent",
				"display_name":"Available memory rate",
				"unit":"%"
			},
			{
				"name":"buffered",
				"display_name":"Buffer memory",
				"unit":"byte"
			},
			{
				"name":"cached",
				"display_name":"Cache memory",
				"unit":"byte"
			},
			{
				"name":"free",
				"display_name":"Free memory",
				"unit":"byte"
			},
			{
				"name":"inactive",
				"display_name":"The amount of inactive memory",
				"unit":"byte"
			},
			{
				"name":"slab",
				"display_name":"Number of kernel caches",
				"unit":"byte"
			},
			{
				"name":"total",
				"display_name":"Total memory",
				"unit":"byte"
			},
			{
				"name":"used",
				"display_name":"Used memory",
				"unit":"byte"
			},
			{
				"name":"used_percent",
				"display_name":"Used memory rate",
				"unit":"%"
			}
		]
	},
	{
		"measurement": {
			"name": "net",
			"display_name": "Network interface and protocol usage",
			"database":"telegraf",
			"res_type":"host"
		},
		"metric_fields": [
			{
				"name":"bytes_sent",
				"display_name":"The total number of bytes sent by the network interface",
				"unit":"byte"
			},
			{
				"name":"bytes_recv",
				"display_name":"The total number of bytes received by the network interface",
				"unit":"byte"
			},
			{
				"name":"packets_sent",
				"display_name":"The total number of packets sent by the network interface",
				"unit":"count"
			},
			{
				"name":"packets_recv",
				"display_name":"The total number of packets received by the network interface",
				"unit":"count"
			},
			{
				"name":"err_in",
				"display_name":"The total number of receive errors detected by the network interface",
				"unit":"count"
			},
			{
				"name":"err_out",
				"display_name":"The total number of transmission errors detected by the network interface",
				"unit":"count"
			},
			{
				"name":"drop_in",
				"display_name":"The total number of received packets dropped by the network interface",
				"unit":"byte"
			},
			{
				"name":"drop_out",
				"display_name":"The total number of transmission packets dropped by the network interface",
				"unit":"byte"
			}
		]
	},
	{
		"measurement": {
			"name": "vm_cpu",
			"display_name": "Guest CPU usage",
			"res_type": "guest",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"cpu_time_system",
				"display_name":"CPU system state time",
				"unit":"ms"
			},
			{
				"name":"cpu_time_user",
				"display_name":"CPU user state time",
				"unit":"ms"
			},
			{
				"name":"cpu_usage_idle_pcore",
				"display_name":"CPU idle rate per core",
				"unit":"%"
			},
			{
				"name":"cpu_usage_pcore",
				"display_name":"CPU utilization rate per core",
				"unit":"%"
			},
			{
				"name":"thread_count",
				"display_name":"The number of threads used by the process",
				"unit":"count"
			},
			{
				"name":"usage_active",
				"display_name":"CPU active state utilization rate",
				"unit":"%"
			}
		]
	},
	{
		"measurement": {
			"name": "vm_diskio",
			"display_name": "Guest disk traffic",
			"res_type": "guest",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"read_bps",
				"display_name":"Disk read rate",
				"unit":"Bps"
			},
			{
				"name":"read_bytes",
				"display_name":"Bytes read",
				"unit":"byte"
			},
			{
				"name":"read_iops",
				"display_name":"Disk read operate rate",
				"unit":"count"
			},
			{
				"name":"write_bps",
				"display_name":"Disk write rate",
				"unit":"Bps"
			},
			{
				"name":"write_bytes",
				"display_name":"Bytes write",
				"unit":"byte"
			},
			{
				"name":"write_iops",
				"display_name":"Disk write operate rate",
				"unit":"count"
			}
		]
	},
	{
		"measurement": {
			"name": "vm_mem",
			"display_name": "Guest memory",
			"res_type": "guest",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"used_percent",
				"display_name":"Used memory rate",
				"unit":"%"
			},
			{
				"name":"vms",
				"display_name":"Virtual memory consumption",
				"unit":"byte"
			},
			{
				"name":"rss",
				"display_name":"Actual use of physical memory",
				"unit":"byte"
			}
		]
	},
	{
		"measurement": {
			"name": "oss_latency",
			"display_name": "Object storage latency",
			"res_type": "oss",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"req_late",
				"display_name":"Request average E2E delay",
				"unit":"ms"
			}
		]
	},
	{
		"measurement": {
			"name": "oss_netio",
			"display_name": "Object storage network traffic",
			"res_type": "oss",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"bps_recv",
				"display_name":"Receive byte",
				"unit":"byte"
			},
			{
				"name":"bps_sent",
				"display_name":"Send byte",
				"unit":"byte"
			}
		]
	},
	{
		"measurement": {
			"name": "oss_req",
			"display_name": "Object store request",
			"res_type": "oss",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"req_count",
				"display_name":"request count",
				"unit":"count"
			}
		]
	},
	{
		"measurement": {
			"name": "rds_conn",
			"display_name": "Rds connect",
			"res_type": "rds",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"used_percent",
				"display_name":"Connection usage",
				"unit":"%"
			}
		]
	},
	{
		"measurement": {
			"name": "rds_cpu",
			"display_name": "Rds CPU usage",
			"res_type": "rds",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"usage_active",
				"display_name":"CPU active state utilization rate",
				"unit":"%"
			},
			{
				"name":"used_percent",
				"display_name":"Connection usage",
				"unit":"%"
			}
		]
	},
	{
		"measurement": {
			"name": "rds_mem",
			"display_name": "Rds memory",
			"res_type": "rds",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"used_percent",
				"display_name":"memory usage",
				"unit":"%"
			}
		]
	},
	{
		"measurement": {
			"name": "rds_netio",
			"display_name": "Rds network traffic",
			"res_type": "rds",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"bps_recv",
				"display_name":"Received traffic per second",
				"unit":"bps"
			},
			{
				"name":"bps_sent",
				"display_name":"Send traffic per second",
				"unit":"bps"
			}
		]
	},
	{
		"measurement": {
			"name": "rds_disk",
			"display_name": "Rds disk usage",
			"res_type": "rds",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"used_percent",
				"display_name":"disk usage",
				"unit":"%"
			}
		]
	},
	{
		"measurement": {
			"name": "dcs_cpu",
			"display_name": "Redis CPU usage",
			"res_type": "redis",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"usage_percent",
				"display_name":"CPU active state utilization rate",
				"unit":"%"
			}
		]	
	},
	{
		"measurement": {
			"name": "dcs_mem",
			"display_name": "Redis memory",
			"res_type": "redis",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"used_percent",
				"display_name":"memory usage",
				"unit":"%"
			}
		]
	},
	{
		"measurement": {
			"name": "dcs_netio",
			"display_name": "Redis network traffic",
			"res_type": "redis",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"bps_recv",
				"display_name":"Received traffic per second",
				"unit":"bps"
			},
			{
				"name":"bps_sent",
				"display_name":"Send traffic per second",
				"unit":"bps"
			}
		]
	},
	{
		"measurement": {
			"name": "dcs_conn",
			"display_name": "Redis connect",
			"res_type": "redis",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"used_conn",
				"display_name":"Connection usage",
				"unit":"%"
			}
		]
	},
	{
		"measurement": {
			"name": "dcs_instantopt",
			"display_name": "Redis operator",
			"res_type": "redis",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"opt_sec",
				"display_name":"Number of commands processed per second",
				"unit":"count"
			}
		]
	},
	{
		"measurement": {
			"name": "dcs_cachekeys",
			"display_name": "Redis keys",
			"res_type": "redis",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"key_count",
				"display_name":"Number of cache keys",
				"unit":"count"
			}
		]
	},
	{
		"measurement": {
			"name": "dcs_datamem",
			"display_name": "Redis data memory",
			"res_type": "redis",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"used_byte",
				"display_name":"Data node memory usage",
				"unit":"byte"
			}
		]
	},
	{
		"measurement": {
			"name": "cloudaccount_balance",
			"display_name": "Cloud account balance",
			"res_type": "cloudaccount",
			"database":"meter_db"
		},
		"metric_fields": [
			{
				"name":"balance",
				"display_name":"balance",
				"unit":"RMB"
			}
		]
	},
	{
		"measurement": {
			"name": "storage",
			"display_name": "Storage usage",
			"res_type": "storage",
			"database":"telegraf"
		},
		"metric_fields": [
			{
				"name":"usage_active",
				"display_name":"Storage utilization rate",
				"unit":"%"
			},
            {
				"name":"free",
				"display_name":"Free storage",
				"unit":"Mb"
			},
		]
	}
]
`
var MetricNeedDeleteDescriptions = []string{"rds_conn", "rds_cpu", "rds_mem", "rds_netio", "rds_disk", "dcs_cpu",
	"dcs_mem", "dcs_netio", "dcs_conn", "dcs_instantopt", "dcs_cachekeys", "dcs_datamem", "oss_latency",
	"oss_netio", "oss_req"}
