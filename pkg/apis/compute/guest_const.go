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

import "yunion.io/x/cloudmux/pkg/apis/compute"

const (
	VM_INIT            = compute.VM_INIT
	VM_UNKNOWN         = compute.VM_UNKNOWN
	VM_SCHEDULE        = "schedule"
	VM_SCHEDULE_FAILED = "sched_fail"
	VM_CREATE_NETWORK  = "network"
	VM_NETWORK_FAILED  = "net_fail"
	VM_DEVICE_FAILED   = "dev_fail"
	VM_CREATE_FAILED   = compute.VM_CREATE_FAILED
	VM_CREATE_DISK     = "disk"
	VM_DISK_FAILED     = "disk_fail"
	VM_IMAGE_CACHING   = "image_caching" // 缓存镜像中
	VM_START_DEPLOY    = "start_deploy"
	VM_DEPLOYING       = compute.VM_DEPLOYING
	VM_DEPLOY_FAILED   = compute.VM_DEPLOY_FAILED
	VM_READY           = compute.VM_READY
	VM_START_START     = compute.VM_START_START
	VM_STARTING        = compute.VM_STARTING
	VM_START_FAILED    = "start_fail" // # = ready
	VM_RUNNING         = compute.VM_RUNNING
	VM_START_STOP      = compute.VM_START_STOP
	VM_STOPPING        = compute.VM_STOPPING
	VM_STOP_FAILED     = "stop_fail" // # = running
	VM_RENEWING        = "renewing"
	VM_RENEW_FAILED    = "renew_failed"
	VM_ATTACH_DISK     = "attach_disk"
	VM_DETACH_DISK     = "detach_disk"
	VM_UNSYNC          = "unsync"

	VM_BACKUP_STARTING         = "backup_starting"
	VM_BACKUP_STOPING          = "backup_stopping"
	VM_BACKUP_CREATING         = "backup_creating"
	VM_BACKUP_START_FAILED     = "backup_start_failed"
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
	VM_SUSPENDING     = compute.VM_SUSPENDING
	VM_SUSPEND        = compute.VM_SUSPEND
	VM_SUSPEND_FAILED = "suspend_failed"

	VM_RESUMING      = compute.VM_RESUMING
	VM_RESUME_FAILED = "resume_failed"

	VM_START_DELETE = "start_delete"
	VM_DELETE_FAIL  = "delete_fail"
	VM_DELETING     = compute.VM_DELETING

	VM_DEALLOCATED = compute.VM_DEALLOCATED

	VM_START_MIGRATE  = "start_migrate"
	VM_MIGRATING      = compute.VM_MIGRATING
	VM_LIVE_MIGRATING = "live_migrating"
	VM_MIGRATE_FAILED = "migrate_failed"

	VM_CHANGE_FLAVOR      = compute.VM_CHANGE_FLAVOR
	VM_CHANGE_FLAVOR_FAIL = "change_flavor_fail"
	VM_REBUILD_ROOT       = compute.VM_REBUILD_ROOT
	VM_REBUILD_ROOT_FAIL  = "rebuild_root_fail"

	VM_START_SNAPSHOT           = "snapshot_start"
	VM_SNAPSHOT                 = "snapshot"
	VM_SNAPSHOT_DELETE          = "snapshot_delete"
	VM_BLOCK_STREAM             = "block_stream"
	VM_BLOCK_STREAM_FAIL        = "block_stream_fail"
	VM_SNAPSHOT_SUCC            = "snapshot_succ"
	VM_SNAPSHOT_FAILED          = "snapshot_failed"
	VM_DISK_RESET               = "disk_reset"
	VM_DISK_RESET_FAIL          = "disk_reset_failed"
	VM_DISK_CHANGE_STORAGE      = "disk_change_storage"
	VM_DISK_CHANGE_STORAGE_FAIL = "disk_change_storage_fail"

	VM_START_INSTANCE_SNAPSHOT   = "start_instance_snapshot"
	VM_INSTANCE_SNAPSHOT_FAILED  = "instance_snapshot_failed"
	VM_START_SNAPSHOT_RESET      = "start_snapshot_reset"
	VM_SNAPSHOT_RESET_FAILED     = "snapshot_reset_failed"
	VM_SNAPSHOT_AND_CLONE_FAILED = "clone_from_snapshot_failed"

	VM_START_INSTANCE_BACKUP  = "start_instance_backup"
	VM_INSTANCE_BACKUP_FAILED = "instance_backup_failed"

	VM_SYNC_CONFIG = compute.VM_SYNC_CONFIG
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

	VM_ASSOCIATE_EIP         = INSTANCE_ASSOCIATE_EIP
	VM_ASSOCIATE_EIP_FAILED  = INSTANCE_ASSOCIATE_EIP_FAILED
	VM_DISSOCIATE_EIP        = INSTANCE_DISSOCIATE_EIP
	VM_DISSOCIATE_EIP_FAILED = INSTANCE_DISSOCIATE_EIP_FAILED

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

	VM_UPDATE_TAGS        = "update_tags"
	VM_UPDATE_TAGS_FAILED = "update_tags_fail"

	VM_RESTART_NETWORK        = "restart_network"
	VM_RESTART_NETWORK_FAILED = "restart_network_failed"

	VM_SYNC_ISOLATED_DEVICE_FAILED = "sync_isolated_device_failed"

	VM_QGA_SET_PASSWORD        = "qga_set_password"
	VM_QGA_COMMAND_EXECUTING   = "qga_command_executing"
	VM_QGA_EXEC_COMMAND_FAILED = "qga_exec_command_failed"

	SHUTDOWN_STOP      = "stop"
	SHUTDOWN_TERMINATE = "terminate"

	HYPERVISOR_KVM       = "kvm"
	HYPERVISOR_CONTAINER = "container"
	HYPERVISOR_BAREMETAL = "baremetal"
	HYPERVISOR_ESXI      = compute.HYPERVISOR_ESXI
	HYPERVISOR_HYPERV    = "hyperv"
	HYPERVISOR_XEN       = "xen"

	HYPERVISOR_ALIYUN         = compute.HYPERVISOR_ALIYUN
	HYPERVISOR_APSARA         = compute.HYPERVISOR_APSARA
	HYPERVISOR_QCLOUD         = compute.HYPERVISOR_QCLOUD
	HYPERVISOR_AZURE          = compute.HYPERVISOR_AZURE
	HYPERVISOR_AWS            = compute.HYPERVISOR_AWS
	HYPERVISOR_HUAWEI         = compute.HYPERVISOR_HUAWEI
	HYPERVISOR_HCS            = compute.HYPERVISOR_HCS
	HYPERVISOR_HCSO           = compute.HYPERVISOR_HCSO
	HYPERVISOR_HCSOP          = compute.HYPERVISOR_HCSOP
	HYPERVISOR_OPENSTACK      = compute.HYPERVISOR_OPENSTACK
	HYPERVISOR_UCLOUD         = compute.HYPERVISOR_UCLOUD
	HYPERVISOR_ZSTACK         = compute.HYPERVISOR_ZSTACK
	HYPERVISOR_GOOGLE         = compute.HYPERVISOR_GOOGLE
	HYPERVISOR_CTYUN          = compute.HYPERVISOR_CTYUN
	HYPERVISOR_ECLOUD         = compute.HYPERVISOR_ECLOUD
	HYPERVISOR_JDCLOUD        = compute.HYPERVISOR_JDCLOUD
	HYPERVISOR_CLOUDPODS      = compute.HYPERVISOR_CLOUDPODS
	HYPERVISOR_NUTANIX        = compute.HYPERVISOR_NUTANIX
	HYPERVISOR_BINGO_CLOUD    = compute.HYPERVISOR_BINGO_CLOUD
	HYPERVISOR_INCLOUD_SPHERE = compute.HYPERVISOR_INCLOUD_SPHERE
	HYPERVISOR_PROXMOX        = compute.HYPERVISOR_PROXMOX
	HYPERVISOR_REMOTEFILE     = compute.HYPERVISOR_REMOTEFILE
	HYPERVISOR_H3C            = compute.HYPERVISOR_H3C

	//	HYPERVISOR_DEFAULT = HYPERVISOR_KVM
	HYPERVISOR_DEFAULT = HYPERVISOR_KVM
)

const (
	VM_POWER_STATES_ON      = "on"
	VM_POWER_STATES_OFF     = "off"
	VM_POWER_STATES_UNKNOWN = "unknown"
)

const (
	VM_SHUTDOWN_MODE_KEEP_CHARGING = "keep_charging"
	VM_SHUTDOWN_MODE_STOP_CHARGING = "stop_charging"
)

const (
	QGA_STATUS_UNKNOWN        = "unknown"
	QGA_STATUS_EXCUTING       = "executing"
	QGA_STATUS_EXECUTE_FAILED = "execute_failed"
	QGA_STATUS_AVAILABLE      = "available"
)

const (
	CPU_MODE_QEMU = "qemu"
	CPU_MODE_HOST = "host"
)

const (
	VM_MACHINE_TYPE_PC  = "pc"
	VM_MACHINE_TYPE_Q35 = "q35"

	VM_MACHINE_TYPE_ARM_VIRT = "virt"

	VM_VDI_PROTOCOL_VNC   = "vnc"
	VM_VDI_PROTOCOL_SPICE = "spice"

	VM_VIDEO_STANDARD = "std"
	VM_VIDEO_QXL      = "qxl"
	VM_VIDEO_VIRTIO   = "virtio"
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
	HYPERVISOR_HCSO,
	HYPERVISOR_HCS,
	HYPERVISOR_HCSOP,
	HYPERVISOR_OPENSTACK,
	HYPERVISOR_UCLOUD,
	HYPERVISOR_ZSTACK,
	HYPERVISOR_GOOGLE,
	HYPERVISOR_CTYUN,
	HYPERVISOR_ECLOUD,
	HYPERVISOR_JDCLOUD,
	HYPERVISOR_CLOUDPODS,
	HYPERVISOR_NUTANIX,
	HYPERVISOR_BINGO_CLOUD,
	HYPERVISOR_INCLOUD_SPHERE,
	HYPERVISOR_PROXMOX,
	HYPERVISOR_REMOTEFILE,
	HYPERVISOR_H3C,
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
	HYPERVISOR_ECLOUD,
	HYPERVISOR_JDCLOUD,
}

var PRIVATE_CLOUD_HYPERVISORS = []string{
	HYPERVISOR_ZSTACK,
	HYPERVISOR_OPENSTACK,
	HYPERVISOR_APSARA,
	HYPERVISOR_CLOUDPODS,
	HYPERVISOR_HCSO,
	HYPERVISOR_HCS,
	HYPERVISOR_HCSOP,
	HYPERVISOR_NUTANIX,
	HYPERVISOR_BINGO_CLOUD,
	HYPERVISOR_INCLOUD_SPHERE,
	HYPERVISOR_PROXMOX,
	HYPERVISOR_REMOTEFILE,
	HYPERVISOR_H3C,
}

// var HYPERVISORS = []string{HYPERVISOR_ALIYUN}

var HYPERVISOR_HOSTTYPE = map[string]string{
	HYPERVISOR_KVM:            HOST_TYPE_HYPERVISOR,
	HYPERVISOR_BAREMETAL:      HOST_TYPE_BAREMETAL,
	HYPERVISOR_ESXI:           HOST_TYPE_ESXI,
	HYPERVISOR_CONTAINER:      HOST_TYPE_KUBELET,
	HYPERVISOR_ALIYUN:         HOST_TYPE_ALIYUN,
	HYPERVISOR_APSARA:         HOST_TYPE_APSARA,
	HYPERVISOR_AZURE:          HOST_TYPE_AZURE,
	HYPERVISOR_AWS:            HOST_TYPE_AWS,
	HYPERVISOR_QCLOUD:         HOST_TYPE_QCLOUD,
	HYPERVISOR_HUAWEI:         HOST_TYPE_HUAWEI,
	HYPERVISOR_HCSO:           HOST_TYPE_HCSO,
	HYPERVISOR_HCSOP:          HOST_TYPE_HCSOP,
	HYPERVISOR_HCS:            HOST_TYPE_HCS,
	HYPERVISOR_OPENSTACK:      HOST_TYPE_OPENSTACK,
	HYPERVISOR_UCLOUD:         HOST_TYPE_UCLOUD,
	HYPERVISOR_ZSTACK:         HOST_TYPE_ZSTACK,
	HYPERVISOR_GOOGLE:         HOST_TYPE_GOOGLE,
	HYPERVISOR_CTYUN:          HOST_TYPE_CTYUN,
	HYPERVISOR_ECLOUD:         HOST_TYPE_ECLOUD,
	HYPERVISOR_JDCLOUD:        HOST_TYPE_JDCLOUD,
	HYPERVISOR_CLOUDPODS:      HOST_TYPE_CLOUDPODS,
	HYPERVISOR_NUTANIX:        HOST_TYPE_NUTANIX,
	HYPERVISOR_BINGO_CLOUD:    HOST_TYPE_BINGO_CLOUD,
	HYPERVISOR_INCLOUD_SPHERE: HOST_TYPE_INCLOUD_SPHERE,
	HYPERVISOR_PROXMOX:        HOST_TYPE_PROXMOX,
	HYPERVISOR_REMOTEFILE:     HOST_TYPE_REMOTEFILE,
	HYPERVISOR_H3C:            HOST_TYPE_H3C,
}

var HOSTTYPE_HYPERVISOR = map[string]string{
	HOST_TYPE_HYPERVISOR:     HYPERVISOR_KVM,
	HOST_TYPE_BAREMETAL:      HYPERVISOR_BAREMETAL,
	HOST_TYPE_ESXI:           HYPERVISOR_ESXI,
	HOST_TYPE_KUBELET:        HYPERVISOR_CONTAINER,
	HOST_TYPE_ALIYUN:         HYPERVISOR_ALIYUN,
	HOST_TYPE_APSARA:         HYPERVISOR_APSARA,
	HOST_TYPE_AZURE:          HYPERVISOR_AZURE,
	HOST_TYPE_AWS:            HYPERVISOR_AWS,
	HOST_TYPE_QCLOUD:         HYPERVISOR_QCLOUD,
	HOST_TYPE_HUAWEI:         HYPERVISOR_HUAWEI,
	HOST_TYPE_HCSO:           HYPERVISOR_HCSO,
	HOST_TYPE_HCSOP:          HYPERVISOR_HCSOP,
	HOST_TYPE_HCS:            HYPERVISOR_HCS,
	HOST_TYPE_OPENSTACK:      HYPERVISOR_OPENSTACK,
	HOST_TYPE_UCLOUD:         HYPERVISOR_UCLOUD,
	HOST_TYPE_ZSTACK:         HYPERVISOR_ZSTACK,
	HOST_TYPE_GOOGLE:         HYPERVISOR_GOOGLE,
	HOST_TYPE_CTYUN:          HYPERVISOR_CTYUN,
	HOST_TYPE_ECLOUD:         HYPERVISOR_ECLOUD,
	HOST_TYPE_JDCLOUD:        HYPERVISOR_JDCLOUD,
	HOST_TYPE_CLOUDPODS:      HYPERVISOR_CLOUDPODS,
	HOST_TYPE_NUTANIX:        HYPERVISOR_NUTANIX,
	HOST_TYPE_BINGO_CLOUD:    HYPERVISOR_BINGO_CLOUD,
	HOST_TYPE_INCLOUD_SPHERE: HYPERVISOR_INCLOUD_SPHERE,
	HOST_TYPE_PROXMOX:        HYPERVISOR_PROXMOX,
	HOST_TYPE_REMOTEFILE:     HYPERVISOR_REMOTEFILE,
	HOST_TYPE_H3C:            HYPERVISOR_H3C,
}

const (
	VM_DEFAULT_WINDOWS_LOGIN_USER         = compute.VM_DEFAULT_WINDOWS_LOGIN_USER
	VM_DEFAULT_LINUX_LOGIN_USER           = compute.VM_DEFAULT_LINUX_LOGIN_USER
	VM_AWS_DEFAULT_LOGIN_USER             = compute.VM_AWS_DEFAULT_LOGIN_USER
	VM_AWS_DEFAULT_WINDOWS_LOGIN_USER     = compute.VM_AWS_DEFAULT_WINDOWS_LOGIN_USER
	VM_JDCLOUD_DEFAULT_WINDOWS_LOGIN_USER = compute.VM_JDCLOUD_DEFAULT_WINDOWS_LOGIN_USER
	VM_AZURE_DEFAULT_LOGIN_USER           = compute.VM_AZURE_DEFAULT_LOGIN_USER
	VM_ZSTACK_DEFAULT_LOGIN_USER          = compute.VM_ZSTACK_DEFAULT_LOGIN_USER

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
	VM_METADATA_CGROUP_CPUSET       = "cgroup_cpuset"
	VM_METADATA_ENABLE_MEMCLEAN     = "enable_memclean"
)

func Hypervisors2HostTypes(hypervisors []string) []string {
	hostTypes := make([]string, len(hypervisors))
	for i := range hypervisors {
		hostTypes[i] = HYPERVISOR_HOSTTYPE[hypervisors[i]]
	}
	return hostTypes
}

// windows allow a maximal length of 15
// http://support.microsoft.com/kb/909264
const MAX_WINDOWS_COMPUTER_NAME_LENGTH = 15
