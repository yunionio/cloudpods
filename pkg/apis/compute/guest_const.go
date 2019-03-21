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

	VM_START_SNAPSHOT  = "snapshot_start"
	VM_SNAPSHOT        = "snapshot"
	VM_SNAPSHOT_DELETE = "snapshot_delete"
	VM_BLOCK_STREAM    = "block_stream"
	VM_MIRROR_FAIL     = "mirror_failed"
	VM_SNAPSHOT_SUCC   = "snapshot_succ"
	VM_SNAPSHOT_FAILED = "snapshot_failed"

	VM_SYNCING_STATUS = "syncing"
	VM_SYNC_CONFIG    = "sync_config"
	VM_SYNC_FAIL      = "sync_fail"

	VM_RESIZE_DISK        = "resize_disk"
	VM_RESIZE_DISK_FAILED = "resize_disk_fail"
	VM_START_SAVE_DISK    = "start_save_disk"
	VM_SAVE_DISK          = "save_disk"
	VM_SAVE_DISK_FAILED   = "save_disk_failed"

	VM_RESTORING_SNAPSHOT = "restoring_snapshot"
	VM_RESTORE_DISK       = "restore_disk"
	VM_RESTORE_STATE      = "restore_state"
	VM_RESTORE_FAILED     = "restore_failed"

	VM_ASSOCIATE_EIP         = "associate_eip"
	VM_ASSOCIATE_EIP_FAILED  = "associate_eip_failed"
	VM_DISSOCIATE_EIP        = "dissociate_eip"
	VM_DISSOCIATE_EIP_FAILED = "dissociate_eip_failed"

	VM_REMOVE_STATEFILE = "remove_state"

	VM_ADMIN = "admin"

	VM_IMPORT        = "import"
	VM_IMPORT_FAILED = "import_fail"

	SHUTDOWN_STOP      = "stop"
	SHUTDOWN_TERMINATE = "terminate"

	HYPERVISOR_KVM       = "kvm"
	HYPERVISOR_CONTAINER = "container"
	HYPERVISOR_BAREMETAL = "baremetal"
	HYPERVISOR_ESXI      = "esxi"
	HYPERVISOR_HYPERV    = "hyperv"
	HYPERVISOR_XEN       = "xen"

	HYPERVISOR_ALIYUN    = "aliyun"
	HYPERVISOR_QCLOUD    = "qcloud"
	HYPERVISOR_AZURE     = "azure"
	HYPERVISOR_AWS       = "aws"
	HYPERVISOR_HUAWEI    = "huawei"
	HYPERVISOR_OPENSTACK = "openstack"
	HYPERVISOR_UCLOUD    = "ucloud"

	//	HYPERVISOR_DEFAULT = HYPERVISOR_KVM
	HYPERVISOR_DEFAULT = HYPERVISOR_KVM
)

var VM_RUNNING_STATUS = []string{VM_START_START, VM_STARTING, VM_RUNNING, VM_BLOCK_STREAM}
var VM_CREATING_STATUS = []string{VM_CREATE_NETWORK, VM_CREATE_DISK, VM_START_DEPLOY, VM_DEPLOYING}

var HYPERVISORS = []string{HYPERVISOR_KVM,
	HYPERVISOR_BAREMETAL,
	HYPERVISOR_ESXI,
	HYPERVISOR_CONTAINER,
	HYPERVISOR_ALIYUN,
	HYPERVISOR_AZURE,
	HYPERVISOR_AWS,
	HYPERVISOR_QCLOUD,
	HYPERVISOR_HUAWEI,
	HYPERVISOR_OPENSTACK,
	HYPERVISOR_UCLOUD,
}

var PUBLIC_CLOUD_HYPERVISORS = []string{
	HYPERVISOR_ALIYUN,
	HYPERVISOR_AWS,
	HYPERVISOR_AZURE,
	HYPERVISOR_QCLOUD,
	HYPERVISOR_HUAWEI,
	HYPERVISOR_OPENSTACK,
	HYPERVISOR_UCLOUD,
}

// var HYPERVISORS = []string{HYPERVISOR_ALIYUN}

var HYPERVISOR_HOSTTYPE = map[string]string{
	HYPERVISOR_KVM:       HOST_TYPE_HYPERVISOR,
	HYPERVISOR_BAREMETAL: HOST_TYPE_BAREMETAL,
	HYPERVISOR_ESXI:      HOST_TYPE_ESXI,
	HYPERVISOR_CONTAINER: HOST_TYPE_KUBELET,
	HYPERVISOR_ALIYUN:    HOST_TYPE_ALIYUN,
	HYPERVISOR_AZURE:     HOST_TYPE_AZURE,
	HYPERVISOR_AWS:       HOST_TYPE_AWS,
	HYPERVISOR_QCLOUD:    HOST_TYPE_QCLOUD,
	HYPERVISOR_HUAWEI:    HOST_TYPE_HUAWEI,
	HYPERVISOR_OPENSTACK: HOST_TYPE_OPENSTACK,
	HYPERVISOR_UCLOUD:    HOST_TYPE_UCLOUD,
}

var HOSTTYPE_HYPERVISOR = map[string]string{
	HOST_TYPE_HYPERVISOR: HYPERVISOR_KVM,
	HOST_TYPE_BAREMETAL:  HYPERVISOR_BAREMETAL,
	HOST_TYPE_ESXI:       HYPERVISOR_ESXI,
	HOST_TYPE_KUBELET:    HYPERVISOR_CONTAINER,
	HOST_TYPE_ALIYUN:     HYPERVISOR_ALIYUN,
	HOST_TYPE_AZURE:      HYPERVISOR_AZURE,
	HOST_TYPE_AWS:        HYPERVISOR_AWS,
	HOST_TYPE_QCLOUD:     HYPERVISOR_QCLOUD,
	HOST_TYPE_HUAWEI:     HYPERVISOR_HUAWEI,
	HOST_TYPE_OPENSTACK:  HYPERVISOR_OPENSTACK,
	HOST_TYPE_UCLOUD:     HYPERVISOR_UCLOUD,
}
