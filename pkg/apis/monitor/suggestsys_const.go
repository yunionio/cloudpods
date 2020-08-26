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
	EIP_UNUSED                 SuggestDriverType = "EIP_UNUSED"
	DISK_UNUSED                SuggestDriverType = "DISK_UNUSED"
	LB_UNUSED                  SuggestDriverType = "LB_UNUSED"
	SNAPSHOT_UNUSED            SuggestDriverType = "SNAPSHOT_UNUSED"
	INSTANCE_SNAPSHOT_UNUSED   SuggestDriverType = "INSTANCE_SNAPSHOT_UNUSED"
	SCALE_DOWN                 SuggestDriverType = "SCALE_DOWN"
	SCALE_UP                   SuggestDriverType = "SCALE_UP"
	SECGROUPRULEINSERVER_ALLIN SuggestDriverType = "SECGROUPRULEINSERVER_ALLIN"
	REDIS_UNREASONABLE         SuggestDriverType = "REDIS_UNREASONABLE"
	RDS_UNREASONABLE           SuggestDriverType = "RDS_UNREASONABLE"
	OSS_UNREASONABLE           SuggestDriverType = "OSS_UNREASONABLE"
	OSS_SEC_ACL                SuggestDriverType = "OSS_SEC_ACL"

	DELETE_DRIVER_ACTION               SuggestDriverAction = "DELETE"
	SCALE_DOWN_DRIVER_ACTION           SuggestDriverAction = "SCALE_DOWN"
	SECGROUPRULEINSERVER_DRIVER_ACTION SuggestDriverAction = "MODIFY_RULE"
	REDIS_UNREASONABLE_DRIVER_ACTION   SuggestDriverAction = "REASONABLE"
	OSS_SEC_ACL_DRIVER_ACTION          SuggestDriverAction = "MODIFY_ACL"
)

type MonitorSuggest string

type MonitorResourceType string

const (
	EIP_MONITOR_RES_TYPE                  = MonitorResourceType("eip")
	DISK_MONITOR_RES_TYPE                 = MonitorResourceType("disk")
	LB_MONITOR_RES_TYPE                   = MonitorResourceType("loadbalancer")
	SCALE_MONTITOR_RES_TYPE               = MonitorResourceType("server")
	SNAPSHOT_MONITOR_RES_TYPE             = MonitorResourceType("snapshot")
	INSTANCE_SNAPSHOT_MONITOR_RES_TYPE    = MonitorResourceType("instance_snapshot")
	SECGROUPRULEINSERVER_MONITOR_RES_TYPE = MonitorResourceType("server")
	REDIS_UNREASONABLE_MONITOR_RES_TYPE   = MonitorResourceType("redis")
	RDS_UNREASONABLE_MONITOR_RES_TYPE     = MonitorResourceType("rds")
	OSS_UNREASONABLE_MONITOR_RES_TYPE     = MonitorResourceType("oss")
	OSS_SEC_ACL_MONITOR_RES_TYPE          = MonitorResourceType("oss")
)

const (
	EIP_MONITOR_SUGGEST                  = MonitorSuggest("release unused EIP")
	DISK_MONITOR_SUGGEST                 = MonitorSuggest("release unused Disk")
	LB_MONITOR_SUGGEST                   = MonitorSuggest("release unused LB")
	SCALE_DOWN_MONITOR_SUGGEST           = MonitorSuggest("adjust machine configuration")
	SECGROUPRULEINSERVER_MONITOR_SUGGEST = MonitorSuggest("adjust secgroup rule")
	OSS_SEC_ACL_MONITOR_SUGGEST          = MonitorSuggest("adjust oss acl")
)

const (
	LB_UNUSED_NLISTENER = "no listener"
	LB_UNUSED_NBCGROUP  = "no backend servergroup"
	LB_UNUSED_NBC       = "no backend server"
)

const (
	SECGROUPRULEINSERVER_CIDR            = "0.0.0.0/0"
	SECGROUPRULEINSERVER_FILTER_PROTOCOL = "icmp"
)
