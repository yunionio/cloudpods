// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package db

const (
	ACT_CREATE      = "create"
	ACT_DELETE      = "delete"
	ACT_UPDATE      = "update"
	ACT_FETCH       = "fetch"
	ACT_ENABLE      = "enable"
	ACT_DISABLE     = "disable"
	ACT_OFFLINE     = "offline"
	ACT_ONLINE      = "online"
	ACT_ATTACH      = "attach"
	ACT_DETACH      = "detach"
	ACT_ATTACH_FAIL = "attach_fail"
	ACT_DETACH_FAIL = "detach_fail"
	ACT_DELETE_FAIL = "delete_fail"

	ACT_CANCEL = "cancel"
	ACT_DONE   = "done"

	ACT_PUBLIC  = "public"
	ACT_PRIVATE = "private"

	ACT_SYNC_UPDATE = "sync_update"
	ACT_SYNC_CREATE = "sync_create"

	ACT_START_CREATE_BACKUP        = "start_create_backup"
	ACT_CREATE_BACKUP              = "create_backup"
	ACT_CREATE_BACKUP_FAILED       = "create_backup_failed"
	ACT_DELETE_BACKUP              = "delete_backup"
	ACT_DELETE_BACKUP_FAILED       = "delete_backup_failed"
	ACT_UPDATE_BACKUP_GUEST_STATUS = "update_backup_guest_status"

	ACT_UPDATE_STATUS       = "updatestatus"
	ACT_STARTING            = "starting"
	ACT_START               = "start"
	ACT_START_FAIL          = "start_fail"
	ACT_BACKUP_START        = "backup_start"
	ACT_BACKUP_START_FAILED = "backup_start_fail"

	ACT_FREEZE      = "freeze"
	ACT_FREEZE_FAIL = "freeze_fail"
	ACT_UNFREEZE    = "unfreeze"

	ACT_RESTARING    = "restarting"
	ACT_RESTART_FAIL = "restart_fail"

	ACT_STOPPING  = "stopping"
	ACT_STOP      = "stop"
	ACT_STOP_FAIL = "stop_fail"

	ACT_RESUMING    = "resuming"
	ACT_RESUME      = "resume"
	ACT_RESUME_FAIL = "resume_fail"

	ACT_RESIZING    = "resizing"
	ACT_RESIZE      = "resize"
	ACT_RESIZE_FAIL = "resize_fail"

	ACT_MIGRATING    = "migrating"
	ACT_MIGRATE      = "migrate"
	ACT_MIGRATE_FAIL = "migrate_fail"

	ACT_VM_CONVERT      = "vm_convert"
	ACT_VM_CONVERTING   = "vm_converting"
	ACT_VM_CONVERT_FAIL = "vm_convert_fail"

	ACT_SPLIT       = "net_split"
	ACT_MERGE       = "net_merge"
	ACT_IP_MAC_BIND = "ip_mac_bind"

	ACT_SAVING            = "saving"
	ACT_SAVE              = "save"
	ACT_SAVE_FAIL         = "save_fail"
	ACT_PROBE             = "probe"
	ACT_PROBE_FAIL        = "probe_fail"
	ACT_IMAGE_DELETE_FAIL = "delete_fail"

	ACT_SWITCHED      = "switched"
	ACT_SWITCH_FAILED = "switch_failed"

	ACT_SNAPSHOTING                   = "snapshoting"
	ACT_SNAPSHOT_STREAM               = "snapshot_stream"
	ACT_SNAPSHOT_DONE                 = "snapshot"
	ACT_SNAPSHOT_READY                = "snapshot_ready"
	ACT_SNAPSHOT_SYNC                 = "snapshot_sync"
	ACT_SNAPSHOT_FAIL                 = "snapshot_fail"
	ACT_SNAPSHOT_DELETING             = "snapshot_deling"
	ACT_SNAPSHOT_DELETE               = "snapshot_del"
	ACT_SNAPSHOT_DELETE_FAIL          = "snapshot_del_fail"
	ACT_SNAPSHOT_FAKE_DELETE          = "snapshot_fake_del"
	ACT_SNAPSHOT_UNLINK               = "snapshot_unlink"
	ACT_APPLY_SNAPSHOT_POLICY         = "apply_snapshot_policy"
	ACT_APPLY_SNAPSHOT_POLICY_FAILED  = "apply_snapshot_policy_failed"
	ACT_CANCEL_SNAPSHOT_POLICY        = "cancel_snapshot_policy"
	ACT_CANCEL_SNAPSHOT_POLICY_FAILED = "cancel_snapshot_policy_failed"
	ACT_VM_SNAPSHOT_AND_CLONE         = "vm_snapshot_and_clone"
	ACT_VM_SNAPSHOT_AND_CLONE_FAILED  = "vm_snapshot_and_clone_failed"

	ACT_VM_RESET_SNAPSHOT        = "instance_reset_snapshot"
	ACT_VM_RESET_SNAPSHOT_FAILED = "instance_reset_snapshot_failed"

	ACT_SNAPSHOT_POLICY_BIND_DISK        = "snapshot_policy_bind_disk"
	ACT_SNAPSHOT_POLICY_BIND_DISK_FAIL   = "snapshot_policy_bind_disk_fail"
	ACT_SNAPSHOT_POLICY_UNBIND_DISK      = "snapshot_policy_unbind_disk"
	ACT_SNAPSHOT_POLICY_UNBIND_DISK_FAIL = "snapshot_policy_unbind_disk_fail"

	ACT_DISK_CLEAN_UP_SNAPSHOTS      = "disk_clean_up_snapshots"
	ACT_DISK_CLEAN_UP_SNAPSHOTS_FAIL = "disk_clean_up_snapshots_fail"
	ACT_DISK_AUTO_SNAPSHOT           = "disk_auto_snapshot"
	ACT_DISK_AUTO_SNAPSHOT_FAIL      = "disk_auto_snapshot_fail"

	ACT_DISK_AUTO_SYNC_SNAPSHOT      = "disk_auto_sync_snapshot"
	ACT_DISK_AUTO_SYNC_SNAPSHOT_FAIL = "disk_auto_sync_snapshot_fail"

	ACT_ALLOCATING           = "allocating"
	ACT_BACKUP_ALLOCATING    = "backup_allocating"
	ACT_ALLOCATE             = "allocate"
	ACT_BACKUP_ALLOCATE      = "backup_allocate"
	ACT_ALLOCATE_FAIL        = "alloc_fail"
	ACT_BACKUP_ALLOCATE_FAIL = "backup_alloc_fail"
	ACT_REW_FAIL             = "renew_fail"

	ACT_SET_AUTO_RENEW      = "set_auto_renew"
	ACT_SET_AUTO_RENEW_FAIL = "set_auto_renew_fail"

	ACT_DELOCATING    = "delocating"
	ACT_DELOCATE      = "delocate"
	ACT_DELOCATE_FAIL = "delocate_fail"

	ACT_ISO_PREPARING    = "iso_preparing"
	ACT_ISO_PREPARE_FAIL = "iso_prepare_fail"
	ACT_ISO_ATTACH       = "iso_attach"
	ACT_ISO_DETACH       = "iso_detach"

	ACT_VFD_PREPARING    = "vfd_preparing"
	ACT_VFD_PREPARE_FAIL = "vfd_prepare_fail"
	ACT_VFD_ATTACH       = "vfd_attach"
	ACT_VFD_DETACH       = "vfd_detach"

	ACT_EIP_ATTACH = "eip_attach"
	ACT_EIP_DETACH = "eip_detach"

	ACT_SET_METADATA = "set_meta"
	ACT_DEL_METADATA = "del_meta"

	ACT_VM_DEPLOY      = "deploy"
	ACT_VM_DEPLOY_FAIL = "deploy_fail"

	ACT_SET_USER_PASSWORD      = "set_user_password"
	ACT_SET_USER_PASSWORD_FAIL = "set_user_password_fail"

	ACT_VM_IO_THROTTLE      = "io_throttle"
	ACT_VM_IO_THROTTLE_FAIL = "io_throttle_fail"

	ACT_REBUILDING_ROOT   = "rebuilding_root"
	ACT_REBUILD_ROOT      = "rebuild_root"
	ACT_REBUILD_ROOT_FAIL = "rebuild_root_fail"

	ACT_CHANGING_FLAVOR    = "changing_flavor"
	ACT_CHANGE_FLAVOR      = "change_flavor"
	ACT_CHANGE_FLAVOR_FAIL = "change_flavor_fail"

	ACT_SYNCING_CONF   = "syncing_conf"
	ACT_SYNC_CONF      = "sync_conf"
	ACT_SYNC_CONF_FAIL = "sync_conf_fail"
	ACT_SYNC_STATUS    = "sync_status"

	ACT_CHANGE_OWNER = "change_owner"
	ACT_SYNC_OWNER   = "sync_owner"
	ACT_SYNC_SHARE   = "sync_share"

	ACT_RESERVE_IP = "reserve_ip"
	ACT_RELEASE_IP = "release_ip"

	ACT_CONVERT_START      = "converting"
	ACT_CONVERT_COMPLETE   = "converted"
	ACT_CONVERT_FAIL       = "convert_fail"
	ACT_UNCONVERT_START    = "unconverting"
	ACT_UNCONVERT_COMPLETE = "unconverted"
	ACT_UNCONVERT_FAIL     = "unconvert_fail"

	ACT_SYNC_HOST_START     = "sync_host_start"
	ACT_SYNCING_HOST        = "syncing_host"
	ACT_SYNC_HOST_COMPLETE  = "sync_host_end"
	ACT_SYNC_HOST_FAILED    = "sync_host_fail"
	ACT_SYNC_NETWORK        = "sync_network"
	ACT_SYNC_NETWORK_FAILED = "sync_network_failed"

	ACT_SYNC_PROJECT_COMPLETE = "sync_project_end"

	ACT_SYNC_LB_START    = "sync_lb_start"
	ACT_SYNCING_LB       = "syncing_lb"
	ACT_SYNC_LB_COMPLETE = "sync_lb_end"

	ACT_CACHING_IMAGE      = "caching_image"
	ACT_CACHE_IMAGE_FAIL   = "cache_image_fail"
	ACT_CACHED_IMAGE       = "cached_image"
	ACT_UNCACHING_IMAGE    = "uncaching_image"
	ACT_UNCACHE_IMAGE_FAIL = "uncache_image_fail"
	ACT_UNCACHED_IMAGE     = "uncached_image"

	ACT_SYNC_CLOUD_DISK          = "sync_cloud_disk"
	ACT_SYNC_CLOUD_SERVER        = "sync_cloud_server"
	ACT_SYNC_CLOUD_SKUS          = "sync_cloud_skus"
	ACT_SYNC_CLOUD_IMAGES        = "sync_cloud_images"
	ACT_SYNC_CLOUD_EIP           = "sync_cloud_eip"
	ACT_SYNC_CLOUD_PROJECT       = "sync_cloud_project"
	ACT_SYNC_CLOUD_ELASTIC_CACHE = "sync_cloud_elastic_cache"

	ACT_PENDING_DELETE = "pending_delete"
	ACT_CANCEL_DELETE  = "cancel_delete"

	// # isolated device (host)
	ACT_HOST_ATTACH_ISOLATED_DEVICE      = "host_attach_isolated_deivce"
	ACT_HOST_ATTACH_ISOLATED_DEVICE_FAIL = "host_attach_isolated_deivce_fail"
	ACT_HOST_DETACH_ISOLATED_DEVICE      = "host_detach_isolated_deivce"
	ACT_HOST_DETACH_ISOLATED_DEVICE_FAIL = "host_detach_isolated_deivce_fail"

	// # isolated device (guest)
	ACT_GUEST_ATTACH_ISOLATED_DEVICE      = "guest_attach_isolated_deivce"
	ACT_GUEST_ATTACH_ISOLATED_DEVICE_FAIL = "guest_attach_isolated_deivce_fail"
	ACT_GUEST_DETACH_ISOLATED_DEVICE      = "guest_detach_isolated_deivce"
	ACT_GUEST_DETACH_ISOLATED_DEVICE_FAIL = "guest_detach_isolated_deivce_fail"
	ACT_GUEST_SAVE_GUEST_IMAGE            = "guest_save_guest_image"
	ACT_GUEST_SAVE_GUEST_IMAGE_FAIL       = "guest_save_guest_image_fail"

	ACT_GUEST_SRC_CHECK = "guest_src_check"

	ACT_GUEST_CPUSET             = "guest_cpuset"
	ACT_GUEST_CPUSET_FAIL        = "guest_cpuset_fail"
	ACT_GUEST_CPUSET_REMOVE      = "guest_cpuset_remove"
	ACT_GUEST_CPUSET_REMOVE_FAIL = "guest_cpuset_remove_fail"

	ACT_CHANGE_BANDWIDTH = "eip_change_bandwidth"
	ACT_EIP_CONVERT_FAIL = "eip_convert_fail"

	ACT_RENEW = "renew"

	ACT_SCHEDULE = "schedule"

	ACT_RECYCLE_PREPAID      = "recycle_prepaid"
	ACT_UNDO_RECYCLE_PREPAID = "undo_recycle_prepaid"

	ACT_HOST_IMPORT_LIBVIRT_SERVERS      = "host_import_libvirt_servers"
	ACT_HOST_IMPORT_LIBVIRT_SERVERS_FAIL = "host_import_libvirt_servers_fail"
	ACT_GUEST_CREATE_FROM_IMPORT_SUCC    = "guest_create_from_import_succ"
	ACT_GUEST_CREATE_FROM_IMPORT_FAIL    = "guest_create_from_import_fail"
	ACT_GUEST_PANICKED                   = "guest_panicked"
	ACT_HOST_MAINTENANCE                 = "host_maintenance"
	ACT_HOST_DOWN                        = "host_down"

	ACT_UPLOAD_OBJECT  = "upload_obj"
	ACT_DELETE_OBJECT  = "delete_obj"
	ACT_MKDIR          = "mkdir"
	ACT_SET_WEBSITE    = "set_website"
	ACT_DELETE_WEBSITE = "delete_website"
	ACT_SET_CORS       = "set_cors"
	ACT_DELETE_CORS    = "delete_cors"
	ACT_SET_REFERER    = "set_referer"
	ACT_SET_POLICY     = "set_policy"
	ACT_DELETE_POLICY  = "delete_policy"

	ACT_GRANT_PRIVILEGE  = "grant_privilege"
	ACT_REVOKE_PRIVILEGE = "revoke_privilege"
	ACT_SET_PRIVILEGES   = "set_privileges"
	ACT_REBOOT           = "reboot"
	ACT_RESTORE          = "restore"
	ACT_CHANGE_CONFIG    = "change_config"
	ACT_RESET_PASSWORD   = "reset_password"

	ACT_SUBIMAGE_UPDATE_FAIL = "guest_image_subimages_update_fail"

	ACT_FLUSH_INSTANCE      = "flush_instance"
	ACT_FLUSH_INSTANCE_FAIL = "flush_instance_fail"

	ACT_SYNC_VPCS        = "sync_vpcs"
	ACT_SYNC_RECORD_SETS = "sync_record_sets"

	ACT_NETWORK_ADD_VPC             = "network_add_vpc"
	ACT_NETWORK_ADD_VPC_FAILED      = "network_add_vpc_failed"
	ACT_NETWORK_REMOVE_VPC          = "network_remove_vpc"
	ACT_NETWORK_REMOVE_VPC_FAILED   = "network_remove_vpc_failed"
	ACT_NETWORK_MODIFY_ROUTE        = "network_modify_route"
	ACT_NETWORK_MODIFY_ROUTE_FAILED = "network_modify_route_failed"

	ACT_UPDATE_RULE = "update_config"
	ACT_UPDATE_TAGS = "update_tags"

	ACT_UPDATE_MONITOR_RESOURCE_JOINT = "update_monitor_resource_joint"
	ACT_DETACH_MONITOR_RESOURCE_JOINT = "detach_monitor_resource_joint"

	ACT_MERGE_NETWORK        = "merge_network"
	ACT_MERGE_NETWORK_FAILED = "merge_network_failed"

	ACT_RECOVERY      = "recovery"
	ACT_RECOVERY_FAIL = "recovery_fail"
	ACT_PACK          = "pack"
	ACT_PACK_FAIL     = "pack_fail"
	ACT_UNPACK        = "unpack"
	ACT_UNPACK_FAIL   = "unpack_fail"

	ACT_ENCRYPT_START = "encrypt_start"
	ACT_ENCRYPT_FAIL  = "encrypt_fail"
	ACT_ENCRYPT_DONE  = "encrypted"

	ACT_BIND   = "bind"
	ACT_UNBIND = "unbind"
)
