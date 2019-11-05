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
	CLOUD_PROVIDER_INIT          = "init"
	CLOUD_PROVIDER_CONNECTED     = "connected"
	CLOUD_PROVIDER_DISCONNECTED  = "disconnected"
	CLOUD_PROVIDER_START_DELETE  = "start_delete"
	CLOUD_PROVIDER_DELETING      = "deleting"
	CLOUD_PROVIDER_DELETED       = "deleted"
	CLOUD_PROVIDER_DELETE_FAILED = "delete_failed"

	CLOUD_PROVIDER_SYNC_STATUS_QUEUING = "queuing"
	CLOUD_PROVIDER_SYNC_STATUS_QUEUED  = "queued"
	CLOUD_PROVIDER_SYNC_STATUS_SYNCING = "syncing"
	CLOUD_PROVIDER_SYNC_STATUS_IDLE    = "idle"
	CLOUD_PROVIDER_SYNC_STATUS_ERROR   = "error"

	CLOUD_PROVIDER_ONECLOUD  = "OneCloud"
	CLOUD_PROVIDER_VMWARE    = "VMware"
	CLOUD_PROVIDER_ALIYUN    = "Aliyun"
	CLOUD_PROVIDER_QCLOUD    = "Qcloud"
	CLOUD_PROVIDER_AZURE     = "Azure"
	CLOUD_PROVIDER_AWS       = "Aws"
	CLOUD_PROVIDER_HUAWEI    = "Huawei"
	CLOUD_PROVIDER_OPENSTACK = "OpenStack"
	CLOUD_PROVIDER_UCLOUD    = "Ucloud"
	CLOUD_PROVIDER_ZSTACK    = "ZStack"

	CLOUD_PROVIDER_GENERICS3 = "S3"
	CLOUD_PROVIDER_CEPH      = "Ceph"
	CLOUD_PROVIDER_XSKY      = "Xsky"

	CLOUD_PROVIDER_HEALTH_NORMAL        = "normal"        // 远端处于健康状态
	CLOUD_PROVIDER_HEALTH_INSUFFICIENT  = "insufficient"  // 不足按需资源余额
	CLOUD_PROVIDER_HEALTH_SUSPENDED     = "suspended"     // 远端处于冻结状态
	CLOUD_PROVIDER_HEALTH_ARREARS       = "arrears"       // 远端处于欠费状态
	CLOUD_PROVIDER_HEALTH_UNKNOWN       = "unknown"       // 未知状态，查询失败
	CLOUD_PROVIDER_HEALTH_NO_PERMISSION = "no permission" // 没有权限获取账单信息

	ZSTACK_BRAND_DSTACK     = "DStack"
	ONECLOUD_BRAND_ONECLOUD = "OneCloud"
)

const (
	CLOUD_ACCESS_ENV_AWS_GLOBAL          = "Aws-int"
	CLOUD_ACCESS_ENV_AWS_CHINA           = CLOUD_PROVIDER_AWS
	CLOUD_ACCESS_ENV_AZURE_GLOBAL        = "Azure-int"
	CLOUD_ACCESS_ENV_AZURE_GERMAN        = "Azure-de"
	CLOUD_ACCESS_ENV_AZURE_US_GOVERNMENT = "Azure-us-gov"
	CLOUD_ACCESS_ENV_AZURE_CHINA         = CLOUD_PROVIDER_AZURE
	CLOUD_ACCESS_ENV_HUAWEI_GLOBAL       = "Huawei-int"
	CLOUD_ACCESS_ENV_HUAWEI_CHINA        = CLOUD_PROVIDER_HUAWEI
)

var (
	CLOUD_PROVIDER_VALID_STATUS        = []string{CLOUD_PROVIDER_CONNECTED}
	CLOUD_PROVIDER_VALID_HEALTH_STATUS = []string{CLOUD_PROVIDER_HEALTH_NORMAL, CLOUD_PROVIDER_HEALTH_NO_PERMISSION}
	PRIVATE_CLOUD_PROVIDERS            = []string{CLOUD_PROVIDER_ZSTACK, CLOUD_PROVIDER_OPENSTACK}

	CLOUD_PROVIDERS = []string{
		CLOUD_PROVIDER_ONECLOUD,
		CLOUD_PROVIDER_VMWARE,
		CLOUD_PROVIDER_ALIYUN,
		CLOUD_PROVIDER_QCLOUD,
		CLOUD_PROVIDER_AZURE,
		CLOUD_PROVIDER_AWS,
		CLOUD_PROVIDER_HUAWEI,
		CLOUD_PROVIDER_OPENSTACK,
		CLOUD_PROVIDER_UCLOUD,
		CLOUD_PROVIDER_ZSTACK,
	}
)

const (
	CLOUD_ENV_PUBLIC_CLOUD  = "public"
	CLOUD_ENV_PRIVATE_CLOUD = "private"
	CLOUD_ENV_ON_PREMISE    = "onpremise"

	CLOUD_ENV_PRIVATE_ON_PREMISE = "private_or_onpremise"
)
