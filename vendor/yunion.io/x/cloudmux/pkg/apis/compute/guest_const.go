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
	VM_INIT          = "init"
	VM_UNKNOWN       = "unknown"
	VM_CREATE_FAILED = "create_fail"
	VM_DEPLOYING     = "deploying"
	VM_DEPLOY_FAILED = "deploy_fail"
	VM_READY         = "ready"
	VM_START_START   = "start_start"
	VM_STARTING      = "starting"
	VM_RUNNING       = "running"
	VM_START_STOP    = "start_stop"
	VM_STOPPING      = "stopping"

	VM_BACKUP_CREATING = "backup_creating"

	VM_SUSPENDING = "suspending"
	VM_SUSPEND    = "suspend"

	VM_RESUMING = "resuming"

	VM_DELETING = "deleting"

	VM_DEALLOCATED = "deallocated"

	VM_MIGRATING = "migrating"

	VM_CHANGE_FLAVOR = "change_flavor"
	VM_REBUILD_ROOT  = "rebuild_root"

	VM_SYNC_CONFIG = "sync_config"

	VM_POWER_STATES_OFF = "off"
	VM_POWER_STATES_ON  = "on"

	HYPERVISOR_ESXI = "esxi"

	HYPERVISOR_ALIYUN         = "aliyun"
	HYPERVISOR_APSARA         = "apsara"
	HYPERVISOR_QCLOUD         = "qcloud"
	HYPERVISOR_AZURE          = "azure"
	HYPERVISOR_AWS            = "aws"
	HYPERVISOR_HUAWEI         = "huawei"
	HYPERVISOR_HCS            = "hcs"
	HYPERVISOR_HCSOP          = "hcsop"
	HYPERVISOR_HCSO           = "hcso"
	HYPERVISOR_OPENSTACK      = "openstack"
	HYPERVISOR_UCLOUD         = "ucloud"
	HYPERVISOR_ZSTACK         = "zstack"
	HYPERVISOR_GOOGLE         = "google"
	HYPERVISOR_CTYUN          = "ctyun"
	HYPERVISOR_ECLOUD         = "ecloud"
	HYPERVISOR_JDCLOUD        = "jdcloud"
	HYPERVISOR_CLOUDPODS      = "cloudpods"
	HYPERVISOR_NUTANIX        = "nutanix"
	HYPERVISOR_BINGO_CLOUD    = "bingocloud"
	HYPERVISOR_INCLOUD_SPHERE = "incloudsphere"
	HYPERVISOR_PROXMOX        = "proxmox"
	HYPERVISOR_REMOTEFILE     = "remotefile"
	HYPERVISOR_H3C            = "h3c"
	HYPERVISOR_KSYUN          = "ksyun"
	HYPERVISOR_BAIDU          = "baidu"
	HYPERVISOR_CUCLOUD        = "cucloud"
	HYPERVISOR_QINGCLOUD      = "qingcloud"
	HYPERVISOR_VOLCENGINE     = "volcengine"
	HYPERVISOR_ORACLE         = "oracle"
)

const (
	VM_DEFAULT_WINDOWS_LOGIN_USER         = "Administrator"
	VM_DEFAULT_LINUX_LOGIN_USER           = "root"
	VM_AWS_DEFAULT_LOGIN_USER             = "ec2user"
	VM_AWS_DEFAULT_WINDOWS_LOGIN_USER     = "Administrator"
	VM_JDCLOUD_DEFAULT_WINDOWS_LOGIN_USER = "administrator"
	VM_AZURE_DEFAULT_LOGIN_USER           = "azureuser"
	VM_ZSTACK_DEFAULT_LOGIN_USER          = "root"
)
