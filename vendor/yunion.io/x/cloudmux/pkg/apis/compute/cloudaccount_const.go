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
	CLOUD_PROVIDER_ONECLOUD       = "OneCloud"
	CLOUD_PROVIDER_VMWARE         = "VMware"
	CLOUD_PROVIDER_NUTANIX        = "Nutanix"
	CLOUD_PROVIDER_ALIYUN         = "Aliyun"
	CLOUD_PROVIDER_APSARA         = "Apsara"
	CLOUD_PROVIDER_QCLOUD         = "Qcloud"
	CLOUD_PROVIDER_AZURE          = "Azure"
	CLOUD_PROVIDER_AWS            = "Aws"
	CLOUD_PROVIDER_HUAWEI         = "Huawei"
	CLOUD_PROVIDER_HCSO           = "HCSO"
	CLOUD_PROVIDER_HCS            = "HCS"
	CLOUD_PROVIDER_HCSOP          = "HCSOP"
	CLOUD_PROVIDER_OPENSTACK      = "OpenStack"
	CLOUD_PROVIDER_UCLOUD         = "Ucloud"
	CLOUD_PROVIDER_ZSTACK         = "ZStack"
	CLOUD_PROVIDER_GOOGLE         = "Google"
	CLOUD_PROVIDER_CTYUN          = "Ctyun"
	CLOUD_PROVIDER_ECLOUD         = "Ecloud"
	CLOUD_PROVIDER_JDCLOUD        = "JDcloud"
	CLOUD_PROVIDER_CLOUDPODS      = "Cloudpods"
	CLOUD_PROVIDER_BINGO_CLOUD    = "BingoCloud"
	CLOUD_PROVIDER_INCLOUD_SPHERE = "InCloudSphere"
	CLOUD_PROVIDER_PROXMOX        = "Proxmox"
	CLOUD_PROVIDER_REMOTEFILE     = "RemoteFile"
	CLOUD_PROVIDER_H3C            = "H3C"
	CLOUD_PROVIDER_KSYUN          = "Ksyun"
	CLOUD_PROVIDER_BAIDU          = "Baidu"
	CLOUD_PROVIDER_CUCLOUD        = "ChinaUnion"
	CLOUD_PROVIDER_QINGCLOUD      = "QingCloud"
	CLOUD_PROVIDER_VOLCENGINE     = "VolcEngine"
	CLOUD_PROVIDER_ORACLE         = "OracleCloud"

	CLOUD_PROVIDER_GENERICS3 = "S3"
	CLOUD_PROVIDER_CEPH      = "Ceph"
	CLOUD_PROVIDER_XSKY      = "Xsky"

	CLOUD_PROVIDER_HEALTH_NORMAL        = "normal"        // 远端处于健康状态
	CLOUD_PROVIDER_HEALTH_INSUFFICIENT  = "insufficient"  // 不足按需资源余额
	CLOUD_PROVIDER_HEALTH_SUSPENDED     = "suspended"     // 远端处于冻结状态
	CLOUD_PROVIDER_HEALTH_ARREARS       = "arrears"       // 远端处于欠费状态
	CLOUD_PROVIDER_HEALTH_UNKNOWN       = "unknown"       // 未知状态，查询失败
	CLOUD_PROVIDER_HEALTH_NO_PERMISSION = "no permission" // 没有权限获取账单信息
)

const (
	CLOUD_ACCESS_ENV_AWS_GLOBAL          = CLOUD_PROVIDER_AWS + "-int"
	CLOUD_ACCESS_ENV_AWS_CHINA           = CLOUD_PROVIDER_AWS
	CLOUD_ACCESS_ENV_AZURE_GLOBAL        = CLOUD_PROVIDER_AZURE + "-int"
	CLOUD_ACCESS_ENV_AZURE_GERMAN        = CLOUD_PROVIDER_AZURE + "-de"
	CLOUD_ACCESS_ENV_AZURE_US_GOVERNMENT = CLOUD_PROVIDER_AZURE + "-us-gov"
	CLOUD_ACCESS_ENV_AZURE_CHINA         = CLOUD_PROVIDER_AZURE
	CLOUD_ACCESS_ENV_ALIYUN_GLOBAL       = CLOUD_PROVIDER_ALIYUN
	CLOUD_ACCESS_ENV_ALIYUN_FINANCE      = CLOUD_PROVIDER_ALIYUN + "-fin"
	CLOUD_ACCESS_ENV_CTYUN_CHINA         = CLOUD_PROVIDER_CTYUN
	CLOUD_ACCESS_ENV_ECLOUD_CHINA        = CLOUD_PROVIDER_ECLOUD
	CLOUD_ACCESS_ENV_JDCLOUD_CHINA       = CLOUD_PROVIDER_JDCLOUD
	CLOUD_ACCESS_ENV_VOLCENGINE_CHINA    = CLOUD_PROVIDER_VOLCENGINE
)
