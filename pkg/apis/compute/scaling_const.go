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

package compute

const (
	EXPANSION_BALANCED = "balanced" //均衡分布

	SHRINK_EARLIEST_CREATION_FIRST        = "earliest"        //最早创建优先
	SHRINK_LATEST_CREATION_FIRST          = "latest"          //最晚创建优先
	SHRINK_CONFIG_EARLIEST_CREATION_FIRST = "config_earliest" //最早配置最早创建优先
	SHRINK_CONFIG_LATEST_CREATION_FIRST   = "config_latest"   //最早配置最晚创建优先

	HEALTH_CHECK_MODE_NORMAL       = "normal"
	HEALTH_CHECK_MODE_LOADBALANCER = "loadbalancer"

	TRIGGER_ALARM  = "alarm"  // 告警
	TRIGGER_TIMING = "timing" // 定时
	TRIGGER_CYCLE  = "cycle"  // 周期定时

	ACTION_ADD    = "add"    // 增加
	ACTION_REMOVE = "remove" // 减少
	ACTION_SET    = "set"    // 设置

	UNIT_ONE     = "s" // 个
	UNIT_PERCENT = "%" // 百分之

	INDICATOR_CPU        = "cpu"        // CPU利用率
	INDICATOR_MEM        = "mem"        // 内存利用率
	INDICATOR_DISK_READ  = "disk_read"  // 磁盘读速率
	INDICATOR_DISK_WRITE = "disk_write" // 磁盘写速率
	INDICATOR_FLOW_INTO  = "flow_into"  // 网络入流量
	INDICATOR_FLOW_OUT   = "flow_out"   // 网络出流量

	WRAPPER_MAX  = "max"     // 最大值
	WRAPPER_MIN  = "min"     //最小值
	WRAPPER_AVER = "average" // 平均值

	OPERATOR_GT = "gt" // 大于
	OPERATOR_LT = "lt" // 小于

	TIMER_TYPE_ONCE  = "once"
	TIMER_TYPE_DAY   = "day"
	TIMER_TYPE_WEEK  = "week"
	TIMER_TYPE_MONTH = "month"

	// 加入中 和 加入失败 的不算是 ScalingGroup 的机器
	SG_GUEST_STATUS_JOINING        = "joining"        // 加入中
	SG_GUEST_STATUS_READY          = "ready"          // 正常
	SG_GUEST_STATUS_REMOVING       = "removing"       // 移除中
	SG_GUEST_STATUS_REMOVE_FAILED  = "remove_failed"  // 移除失败
	SG_GUEST_STATUS_PENDING_REMOVE = "pending_remove" // 机器进入回收站

	// 只有ready状态是正常的
	SG_STATUS_READY              = "ready"              // 正常
	SG_STATUS_DELETING           = "deleting"           // 删除中
	SG_STATUS_WAIT_ACTIVITY_OVER = "wait_activity_over" // 正在等待伸缩活动完毕
	SG_STATUS_DESTROY_INSTANCE   = "destroy_instance"   // 正在销毁伸缩组内实例
	SG_STATUS_DELETE_FAILED      = "delete_failed"      // 删除失败
	SG_STATUS_CREATE_FAILED      = "create_failed"
	SG_STATUS_DELETED            = "deleted" // 删除

	SP_STATUS_READY         = "ready" // 正常
	SP_STATUS_CREATING      = "creating"
	SP_STATUS_CREATE_FAILED = "create_failed" // 创建失败
	SP_STATUS_DELETING      = "deleting"      // 删除中
	SP_STATUS_DELETE_FAILED = "delete_failed" // 删除失败

	SA_STATUS_WAIT         = "wait"         // 等待中
	SA_STATUS_EXEC         = "execution"    // 执行中
	SA_STATUS_SUCCEED      = "succeed"      // 成功
	SA_STATUS_PART_SUCCEED = "part_succeed" // 部分成功
	SA_STATUS_FAILED       = "failed"       // 失败
	SA_STATUS_REJECT       = "reject"       // 拒绝
)
