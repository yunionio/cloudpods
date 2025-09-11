# 云平台监控指标支持表

## RDS 监控指标

| 监控指标 | 华为云 | 阿里云 | 飞天 | Azure | 京东云 | 腾讯云 | AWS |
|---------|--------|--------|------|-------|--------|--------|-----|
| CPU利用率(rds_cpu.usage_active) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 内存利用率(rds_mem.used_percent) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| 网络入流量(rds_netio.bps_recv) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 网络出流量(rds_netio.bps_sent) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 磁盘使用率(rds_disk.used_percent) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| 磁盘读IO(rds_diskio.read_bps) | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| 磁盘写IO(rds_diskio.write_bps) | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| 磁盘IO使用率(rds_diskio.used_percent) | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ |
| 连接数(rds_conn.used_count) | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ |
| 活跃连接数(rds_conn.active_count) | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ |
| 连接数使用率(rds_conn.used_percent) | ❌ | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ |
| 失败连接数(rds_conn.failed_count) | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ |
| QPS(rds_qps.query_qps) | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ |
| TPS(rds_tps.trans_qps) | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ |
| InnoDB读IO(rds_innodb.read_bps) | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ |
| InnoDB写IO(rds_innodb.write_bps) | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ |

## 虚拟机监控指标

| 监控指标 | KVM | 华为云 | 阿里云 | 飞天 | Azure | VMware | Google | 品高云 | AWS | 京东云 | 移动云 | ZStack | 腾讯云 | 火山引擎 | 百度云 | 天翼云 | Oracle | 华为云Stack |
|---------|-----|--------|--------|------|-------|--------|--------|--------|-----|--------|--------|---------|--------|----------|--------|--------|--------|------------|
| CPU使用率(vm_cpu.usage_active) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 内存使用率(vm_mem.used_percent) | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| 磁盘使用率(vm_disk.used_percent) | ❌ | ❌ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ✅  | ❌ | ✅ | ✅ | ❌ | ✅ |
| 磁盘读速率(vm_diskio.read_bps) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ |
| 磁盘写速率(vm_diskio.write_bps) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ |
| 磁盘读IOPS(vm_diskio.read_iops) | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ |
| 磁盘写IOPS(vm_diskio.write_iops) | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ |
| 网络入速率(vm_netio.bps_recv) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 网络出速率(vm_netio.bps_sent) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| TCP连接数(vm_netio.tcp_connections) | ❌ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ |
| 进程数量(vm_process.number) | ❌ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |

## 宿主机监控指标

| 监控指标 | VMware | 品高云 | ZStack |
|---------|--------|--------|---------|
| CPU使用率(cpu.usage_active) | ✅ | ❌ | ✅ |
| 内存使用率(mem.used_percent) | ✅ | ❌ | ✅ |
| 磁盘读速率(diskio.read_bps) | ✅ | ❌ | ✅ |
| 磁盘写速率(diskio.write_bps) | ✅ | ❌ | ✅ |
| 网络入速率(net.bps_recv) | ✅ | ❌ | ✅ |
| 网络出速率(net.bps_sent) | ✅ | ❌ | ✅ |
| 磁盘读IOPS(diskio.read_iops) | ❌ | ✅ | ✅ |
| 磁盘写IOPS(diskio.write_iops) | ❌ | ✅ | ✅ |

## Redis监控指标

| 监控指标 | 华为云 | 阿里云 | Azure | 飞天 | AWS | 腾讯云 |
|---------|--------|--------|-------|--------|-----|--------|
| CPU使用率(dcs_cpu.usage_active) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 内存使用率(dcs_mem.used_percent) | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ |
| 网络入速率(dcs_netio.bps_recv) | ✅ | ✅ | ❌ | ✅ | ❌ | ✅ |
| 网络出速率(dcs_netio.bps_sent) | ✅ | ✅ | ❌ | ✅ | ❌ | ✅ |
| 网络连接数(dcs_conn.used_conn) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 每秒处理指令数(dcs_instantopt.opt_sec) | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ |
| 命中key数量(dcs_cachekeys.key_count) | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ |
| 过期key数量(dcs_cachekeys.expire_key_count) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 内存使用量(dcs_datamem.used_byte) | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ |

## 负载均衡监控指标

| 监控指标 | 华为云 | 阿里云 | 飞天 | Azure | 华为云Stack |
|---------|--------|--------|------|-------|------------|
| SNAT端口使用率(haproxy.used_snat_port) | ❌ | ❌ | ❌ | ✅ | ❌ |
| SNAT连接数(haproxy.snat_conn_count) | ❌ | ❌ | ❌ | ✅ | ❌ |
| 入带宽速率(haproxy.bin) | ✅ | ✅ | ✅ | ❌ | ✅ |
| 出带宽速率(haproxy.bout) | ✅ | ✅ | ✅ | ❌ | ✅ |
| 入包速率(haproxy.packet_rx) | ❌ | ✅ | ✅ | ❌ | ❌ |
| 出包速率(haproxy.packet_tx) | ❌ | ✅ | ✅ | ❌ | ❌ |
| 非活跃连接数(haproxy.inactive_connection) | ✅ | ✅ | ✅ | ❌ | ❌ |
| 活跃连接数(haproxy.active_connection) | ✅ | ❌ | ❌ | ❌ | ✅ |
| 最大并发数(haproxy.max_connection) | ✅ | ❌ | ❌ | ❌ | ✅ |
| 后端异常实例数(haproxy.unhealthy_server_count) | ❌ | ✅ | ✅ | ❌ | ❌ |
| 状态码统计(haproxy.hrsp_Nxx) | ✅ | ✅ | ✅ | ❌ | ❌ |
| 入方向丢弃流量(haproxy.drop_traffic_tx) | ❌ | ✅ | ❌ | ❌ | ❌ |
| 出方向丢弃流量(haproxy.drop_traffic_rx) | ❌ | ✅ | ✅ | ❌ | ❌ |
| 入方向丢弃包数(haproxy.drop_packet_tx) | ❌ | ✅ | ❌ | ❌ | ❌ |
| 出方向丢弃包数(haproxy.drop_packet_rx) | ❌ | ✅ | ✅ | ❌ | ❌ |

## 对象存储监控指标

| 监控指标 | 华为云 | 阿里云 | 飞天 | 火山引擎 | 华为云Stack |
|---------|--------|--------|------|----------|------------|
| 出速率(oss_netio.bps_sent) | ✅ | ✅ | ✅ | ❌ | ❌ |
| 入速率(oss_netio.bps_recv) | ✅ | ✅ | ✅ | ❌ | ❌ |
| 请求延时(oss_latency.req_late) | ✅ | ✅ | ✅ | ❌ | ❌ |
| 总请求数量(oss_req.req_count) | ✅ | ✅ | ✅ | ❌ | ✅ |
| 5XX错误数量(oss_req.5xx_count) | ❌ | ✅ | ✅ | ✅ | ❌ |
| 4XX错误数量(oss_req.4xx_count) | ❌ | ✅ | ✅ | ✅ | ❌ |
| 3XX重定向数量(oss_req.3xx_count) | ❌ | ✅ | ✅ | ✅ | ❌ |
| 2XX成功数量(oss_req.2xx_count) | ❌ | ✅ | ✅ | ✅ | ❌ |
| 存储总容量(oss_storage.size) | ❌ | ✅ | ✅ | ❌ | ❌ |

## K8S节点监控指标

| 监控指标 | 阿里云 | Azure | 腾讯云 |
|---------|--------|-------|--------|
| CPU使用率(k8s_node_cpu.usage_active) | ✅ | ✅ | ✅ |
| 内存使用率(k8s_node_mem.used_percent) | ✅ | ✅ | ✅ |
| 磁盘使用率(k8s_node_disk.used_percent) | ✅ | ✅ | ✅ |
| 网络入速率(k8s_node_netio.bps_recv) | ✅ | ✅ | ✅ |
| 网络出速率(k8s_node_netio.bps_sent) | ✅ | ✅ | ✅ |

## ModelArts资源池监控指标

| 监控指标 | 华为云 |
|---------|--------|
| CPU使用率(modelarts_pool_cpu.usage_percent) | ✅ |
| 内存使用率(modelarts_pool_mem.usage_percent) | ✅ |
| GPU内存使用率(modelarts_pool_gpu_mem.usage_percent) | ✅ |
| GPU利用率(modelarts_pool_gpu_util.percent) | ✅ |
| NPU利用率(modelarts_pool_npu_util.percent) | ✅ |
| NPU内存使用率(modelarts_pool_npu_mem.usage_percent) | ✅ |
| 磁盘可用容量(modelarts_pool_disk.available_capacity) | ✅ |
| 磁盘总容量(modelarts_pool_disk.capacity) | ✅ |
| 磁盘使用率(modelarts_pool_disk.usage_percent) | ✅ |

## 网络监控指标

| 监控指标 | 华为云 |
|---------|--------|
| CPU使用率(wire_cpu.usage_percent) | ✅ |
| 内存使用率(wire_mem.usage_percent) | ✅ |
| 网络响应时间(wire_net.rt) | ✅ |
| 网络不可达率(wire_net.unreachable_rate) | ✅ |

## EIP监控指标

| 监控指标 | 华为云 | 阿里云 | 飞天 |
|---------|--------|--------|------|
| 入带宽(eip_net.bps_recv) | ❌ | ✅ | ✅ |
| 出带宽(eip_net.bps_sent) | ❌ | ✅ | ✅ |
| 出方向限速丢包率(eip_net.drop_speed_rx) | ❌ | ✅ | ✅ |

## 说明

- ✅ 表示支持该监控指标
- ❌ 表示不支持该监控指标
- 部分指标在不同平台上的实现方式可能略有差异
- 某些指标需要特定的配置或权限才能获取
- 监控数据的粒度和历史保留时间可能因平台而异
