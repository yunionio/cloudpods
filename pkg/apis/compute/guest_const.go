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
	VM_INIT            = "init"
	VM_UNKNOWN         = "unknown"
	VM_SCHEDULE        = "schedule"
	VM_SCHEDULE_FAILED = "sched_fail"
	VM_CREATE_NETWORK  = "network"
	VM_NETWORK_FAILED  = "net_fail"
	VM_DEVICE_FAILED   = "dev_fail"
	VM_CREATE_FAILED   = "create_fail"
	VM_CREATE_DISK     = "disk"
	VM_DISK_FAILED     = "disk_fail"
	VM_START_DEPLOY    = "start_deploy"
	VM_DEPLOYING       = "deploying"
	VM_DEPLOY_FAILED   = "deploy_fail"
	VM_READY           = "ready"
	VM_START_START     = "start_start"
	VM_STARTING        = "starting"
	VM_START_FAILED    = "start_fail" // # = ready
	VM_RUNNING         = "running"
	VM_START_STOP      = "start_stop"
	VM_STOPPING        = "stopping"
	VM_STOP_FAILED     = "stop_fail" // # = running
	VM_RENEWING        = "renewing"
	VM_RENEW_FAILED    = "renew_failed"
	VM_ATTACH_DISK     = "attach_disk"
	VM_DETACH_DISK     = "detach_disk"

	VM_BACKUP_STARTING         = "backup_starting"
	VM_BACKUP_CREATING         = "backup_creating"
	VM_BACKUP_CREATE_FAILED    = "backup_create_fail"
	VM_DEPLOYING_BACKUP        = "deploying_backup"
	VM_DEPLOYING_BACKUP_FAILED = "deploging_backup_fail"
	VM_DELETING_BACKUP         = "deleting_backup"
	VM_BACKUP_DELETE_FAILED    = "backup_delete_fail"
	VM_SWITCH_TO_BACKUP        = "switch_to_backup"
	VM_SWITCH_TO_BACKUP_FAILED = "switch_to_backup_fail"

	VM_ATTACH_DISK_FAILED = "attach_disk_fail"
	VM_DETACH_DISK_FAILED = "detach_disk_fail"

	VM_START_SUSPEND  = "start_suspend"
	VM_SUSPENDING     = "suspending"
	VM_SUSPEND        = "suspend"
	VM_SUSPEND_FAILED = "suspend_failed"

	VM_RESUMING      = "resuming"
	VM_RESUME_FAILED = "resume_failed"

	VM_START_DELETE = "start_delete"
	VM_DELETE_FAIL  = "delete_fail"
	VM_DELETING     = "deleting"

	VM_DEALLOCATED = "deallocated"

	VM_START_MIGRATE  = "start_migrate"
	VM_MIGRATING      = "migrating"
	VM_MIGRATE_FAILED = "migrate_failed"

	VM_CHANGE_FLAVOR      = "change_flavor"
	VM_CHANGE_FLAVOR_FAIL = "change_flavor_fail"
	VM_REBUILD_ROOT       = "rebuild_root"
	VM_REBUILD_ROOT_FAIL  = "rebuild_root_fail"

	VM_START_SNAPSHOT    = "snapshot_start"
	VM_SNAPSHOT          = "snapshot"
	VM_SNAPSHOT_DELETE   = "snapshot_delete"
	VM_BLOCK_STREAM      = "block_stream"
	VM_BLOCK_STREAM_FAIL = "block_stream_fail"
	VM_SNAPSHOT_SUCC     = "snapshot_succ"
	VM_SNAPSHOT_FAILED   = "snapshot_failed"
	VM_DISK_RESET        = "disk_reset"
	VM_DISK_RESET_FAIL   = "disk_reset_failed"

	VM_START_INSTANCE_SNAPSHOT   = "start_instance_snapshot"
	VM_INSTANCE_SNAPSHOT_FAILED  = "instance_snapshot_failed"
	VM_START_SNAPSHOT_RESET      = "start_snapshot_reset"
	VM_SNAPSHOT_RESET_FAILED     = "snapshot_reset_failed"
	VM_SNAPSHOT_AND_CLONE_FAILED = "clone_from_snapshot_failed"

	VM_SYNC_CONFIG = "sync_config"
	VM_SYNC_FAIL   = "sync_fail"

	VM_START_RESIZE_DISK  = "start_resize_disk"
	VM_RESIZE_DISK        = "resize_disk"
	VM_RESIZE_DISK_FAILED = "resize_disk_fail"

	VM_START_SAVE_DISK  = "start_save_disk"
	VM_SAVE_DISK        = "save_disk"
	VM_SAVE_DISK_FAILED = "save_disk_failed"

	VM_RESTORING_SNAPSHOT = "restoring_snapshot"
	VM_RESTORE_DISK       = "restore_disk"
	VM_RESTORE_STATE      = "restore_state"
	VM_RESTORE_FAILED     = "restore_failed"

	VM_ASSOCIATE_EIP         = "associate_eip"
	VM_ASSOCIATE_EIP_FAILED  = "associate_eip_failed"
	VM_DISSOCIATE_EIP        = "dissociate_eip"
	VM_DISSOCIATE_EIP_FAILED = "dissociate_eip_failed"

	// 公网IP转换Eip中(EIP转换中)
	VM_START_EIP_CONVERT  = "start_eip_convert"
	VM_EIP_CONVERT_FAILED = "eip_convert_failed"

	// 设置自动续费
	VM_SET_AUTO_RENEW        = "set_auto_renew"
	VM_SET_AUTO_RENEW_FAILED = "set_auto_renew_failed"

	VM_REMOVE_STATEFILE = "remove_state"

	VM_IO_THROTTLE      = "io_throttle"
	VM_IO_THROTTLE_FAIL = "io_throttle_fail"

	VM_ADMIN = "admin"

	VM_IMPORT        = "import"
	VM_IMPORT_FAILED = "import_fail"

	VM_CONVERT        = "convert"
	VM_CONVERTING     = "converting"
	VM_CONVERT_FAILED = "convert_failed"
	VM_CONVERTED      = "converted"

	VM_TEMPLATE_SAVING      = "tempalte_saving"
	VM_TEMPLATE_SAVE_FAILED = "template_save_failed"

	SHUTDOWN_STOP      = "stop"
	SHUTDOWN_TERMINATE = "terminate"

	HYPERVISOR_KVM       = "kvm"
	HYPERVISOR_CONTAINER = "container"
	HYPERVISOR_BAREMETAL = "baremetal"
	HYPERVISOR_ESXI      = "esxi"
	HYPERVISOR_HYPERV    = "hyperv"
	HYPERVISOR_XEN       = "xen"

	HYPERVISOR_ALIYUN    = "aliyun"
	HYPERVISOR_APSARA    = "apsara"
	HYPERVISOR_QCLOUD    = "qcloud"
	HYPERVISOR_AZURE     = "azure"
	HYPERVISOR_AWS       = "aws"
	HYPERVISOR_HUAWEI    = "huawei"
	HYPERVISOR_OPENSTACK = "openstack"
	HYPERVISOR_UCLOUD    = "ucloud"
	HYPERVISOR_ZSTACK    = "zstack"
	HYPERVISOR_GOOGLE    = "google"
	HYPERVISOR_CTYUN     = "ctyun"

	//	HYPERVISOR_DEFAULT = HYPERVISOR_KVM
	HYPERVISOR_DEFAULT = HYPERVISOR_KVM
)

const (
	CPU_MODE_QEMU = "qemu"
	CPU_MODE_HOST = "host"
)

const (
	OS_ARCH_X86 = "x86"
	OS_ARCH_ARM = "arm"
)

var VM_RUNNING_STATUS = []string{VM_START_START, VM_STARTING, VM_RUNNING, VM_BLOCK_STREAM, VM_BLOCK_STREAM_FAIL}
var VM_CREATING_STATUS = []string{VM_CREATE_NETWORK, VM_CREATE_DISK, VM_START_DEPLOY, VM_DEPLOYING}

var HYPERVISORS = []string{
	HYPERVISOR_KVM,
	HYPERVISOR_BAREMETAL,
	HYPERVISOR_ESXI,
	HYPERVISOR_CONTAINER,
	HYPERVISOR_ALIYUN,
	HYPERVISOR_APSARA,
	HYPERVISOR_AZURE,
	HYPERVISOR_AWS,
	HYPERVISOR_QCLOUD,
	HYPERVISOR_HUAWEI,
	HYPERVISOR_OPENSTACK,
	HYPERVISOR_UCLOUD,
	HYPERVISOR_ZSTACK,
	HYPERVISOR_GOOGLE,
	HYPERVISOR_CTYUN,
}

var ONECLOUD_HYPERVISORS = []string{
	HYPERVISOR_BAREMETAL,
	HYPERVISOR_KVM,
	HYPERVISOR_CONTAINER,
}

var PUBLIC_CLOUD_HYPERVISORS = []string{
	HYPERVISOR_ALIYUN,
	HYPERVISOR_AWS,
	HYPERVISOR_AZURE,
	HYPERVISOR_QCLOUD,
	HYPERVISOR_HUAWEI,
	HYPERVISOR_UCLOUD,
	HYPERVISOR_GOOGLE,
	HYPERVISOR_CTYUN,
}

var PRIVATE_CLOUD_HYPERVISORS = []string{
	HYPERVISOR_ZSTACK,
	HYPERVISOR_OPENSTACK,
	HYPERVISOR_APSARA,
}

// var HYPERVISORS = []string{HYPERVISOR_ALIYUN}

var HYPERVISOR_HOSTTYPE = map[string]string{
	HYPERVISOR_KVM:       HOST_TYPE_HYPERVISOR,
	HYPERVISOR_BAREMETAL: HOST_TYPE_BAREMETAL,
	HYPERVISOR_ESXI:      HOST_TYPE_ESXI,
	HYPERVISOR_CONTAINER: HOST_TYPE_KUBELET,
	HYPERVISOR_ALIYUN:    HOST_TYPE_ALIYUN,
	HYPERVISOR_APSARA:    HOST_TYPE_APSARA,
	HYPERVISOR_AZURE:     HOST_TYPE_AZURE,
	HYPERVISOR_AWS:       HOST_TYPE_AWS,
	HYPERVISOR_QCLOUD:    HOST_TYPE_QCLOUD,
	HYPERVISOR_HUAWEI:    HOST_TYPE_HUAWEI,
	HYPERVISOR_OPENSTACK: HOST_TYPE_OPENSTACK,
	HYPERVISOR_UCLOUD:    HOST_TYPE_UCLOUD,
	HYPERVISOR_ZSTACK:    HOST_TYPE_ZSTACK,
	HYPERVISOR_GOOGLE:    HOST_TYPE_GOOGLE,
	HYPERVISOR_CTYUN:     HOST_TYPE_CTYUN,
}

var HOSTTYPE_HYPERVISOR = map[string]string{
	HOST_TYPE_HYPERVISOR: HYPERVISOR_KVM,
	HOST_TYPE_BAREMETAL:  HYPERVISOR_BAREMETAL,
	HOST_TYPE_ESXI:       HYPERVISOR_ESXI,
	HOST_TYPE_KUBELET:    HYPERVISOR_CONTAINER,
	HOST_TYPE_ALIYUN:     HYPERVISOR_ALIYUN,
	HOST_TYPE_APSARA:     HYPERVISOR_APSARA,
	HOST_TYPE_AZURE:      HYPERVISOR_AZURE,
	HOST_TYPE_AWS:        HYPERVISOR_AWS,
	HOST_TYPE_QCLOUD:     HYPERVISOR_QCLOUD,
	HOST_TYPE_HUAWEI:     HYPERVISOR_HUAWEI,
	HOST_TYPE_OPENSTACK:  HYPERVISOR_OPENSTACK,
	HOST_TYPE_UCLOUD:     HYPERVISOR_UCLOUD,
	HOST_TYPE_ZSTACK:     HYPERVISOR_ZSTACK,
	HOST_TYPE_GOOGLE:     HYPERVISOR_GOOGLE,
	HOST_TYPE_CTYUN:      HYPERVISOR_CTYUN,
}

const (
	VM_DEFAULT_WINDOWS_LOGIN_USER     = "Administrator"
	VM_DEFAULT_LINUX_LOGIN_USER       = "root"
	VM_AWS_DEFAULT_LOGIN_USER         = "ec2user"
	VM_AWS_DEFAULT_WINDOWS_LOGIN_USER = "Administrator"
	VM_AZURE_DEFAULT_LOGIN_USER       = "azureuser"
	VM_ZSTACK_DEFAULT_LOGIN_USER      = "root"

	VM_METADATA_APP_TAGS            = "app_tags"
	VM_METADATA_CREATE_PARAMS       = "create_params"
	VM_METADATA_LOGIN_ACCOUNT       = "login_account"
	VM_METADATA_LOGIN_KEY           = "login_key"
	VM_METADATA_LAST_LOGIN_KEY      = "last_login_key"
	VM_METADATA_LOGIN_KEY_TIMESTAMP = "login_key_timestamp"
	VM_METADATA_OS_ARCH             = "os_arch"
	VM_METADATA_OS_DISTRO           = "os_distribution"
	VM_METADATA_OS_NAME             = "os_name"
	VM_METADATA_OS_VERSION          = "os_version"
)

func Hypervisors2HostTypes(hypervisors []string) []string {
	hostTypes := make([]string, len(hypervisors))
	for i := range hypervisors {
		hostTypes[i] = HYPERVISOR_HOSTTYPE[hypervisors[i]]
	}
	return hostTypes
}
