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

package monitor

const (
	SUGGEST_ALERT_READY        = "ready"
	SUGGEST_ALERT_START_DELETE = "start_delete"
	SUGGEST_ALERT_DELETE_FAIL  = "delete_fail"
	SUGGEST_ALERT_DELETING     = "deleting"
)

type SuggestDriverType string

type SuggestDriverAction string

const (
	EIP_UNUSED      SuggestDriverType = "EIP_UNUSED"
	DISK_UNUSED     SuggestDriverType = "DISK_UNUSED"
	LB_UNUSED       SuggestDriverType = "LB_UNUSED"
	SNAPSHOT_UNUSED SuggestDriverType = "SNAPSHOT_UNUSED"
	SCALE_DOWN      SuggestDriverType = "SCALE_DOWN"
	SCALE_UP        SuggestDriverType = "SCALE_UP"

	DELETE_DRIVER_ACTION     SuggestDriverAction = "DELETE"
	SCALE_DOWN_DRIVER_ACTION SuggestDriverAction = "SCALE_DOWN"
)

type MonitorSuggest string

type MonitorResourceType string

const (
	EIP_MONITOR_RES_TYPE    = MonitorResourceType("弹性EIP")
	DISK_MONITOR_RES_TYPE   = MonitorResourceType("云硬盘")
	LB_MONITOR_RES_TYPE     = MonitorResourceType("负载均衡实例")
	SCALE_MONTITOR_RES_TYPE = MonitorResourceType("虚拟机")
)

const (
	EIP_MONITOR_SUGGEST        = MonitorSuggest("释放未使用的EIP")
	DISK_MONITOR_SUGGEST       = MonitorSuggest("释放未使用的Disk")
	LB_MONITOR_SUGGEST         = MonitorSuggest("释放未使用的LB")
	SCALE_DOWN_MONITOR_SUGGEST = MonitorSuggest("缩减机器配置")
)

const (
	LB_UNUSED_NLISTENER = "没有监听"
	LB_UNUSED_NBCGROUP  = "没有后端服务器组"
	LB_UNUSED_NBC       = "没有后端服务器"
)
