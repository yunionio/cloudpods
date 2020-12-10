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

package logclient

const (
	ACT_ADDTAG                       = "addtag"
	ACT_ALLOCATE                     = "allocate"
	ACT_DELOCATE                     = "delocate"
	ACT_BM_CONVERT_HYPER             = "bm_convert_hyper"
	ACT_BM_MAINTENANCE               = "bm_maintenance"
	ACT_BM_UNCONVERT_HYPER           = "bm_unconvert_hyper"
	ACT_BM_UNMAINTENANCE             = "bm_unmaintenance"
	ACT_CANCEL_DELETE                = "cancel_delete"
	ACT_CHANGE_OWNER                 = "change_owner"
	ACT_SYNC_CLOUD_OWNER             = "sync_cloud_owner"
	ACT_CLOUD_FULLSYNC               = "cloud_fullsync"
	ACT_CLOUD_SYNC                   = "cloud_sync"
	ACT_CREATE                       = "create"
	ACT_DELETE                       = "delete"
	ACT_PENDING_DELETE               = "pending_delete"
	ACT_DISABLE                      = "disable"
	ACT_ENABLE                       = "enable"
	ACT_GUEST_ATTACH_ISOLATED_DEVICE = "guest_attach_isolated_device"
	ACT_GUEST_DETACH_ISOLATED_DEVICE = "guest_detach_isolated_device"
	ACT_MERGE                        = "merge"
	ACT_OFFLINE                      = "offline"
	ACT_ONLINE                       = "online"
	ACT_PRIVATE                      = "private"
	ACT_PUBLIC                       = "public"
	ACT_RELEASE_IP                   = "release_ip"
	ACT_RESERVE_IP                   = "reserve_ip"
	ACT_RESIZE                       = "resize"
	ACT_RMTAG                        = "rmtag"
	ACT_SPLIT                        = "split"
	ACT_UNCACHED_IMAGE               = "uncached_image"
	ACT_UPDATE                       = "update"
	ACT_VM_ATTACH_DISK               = "vm_attach_disk"
	ACT_VM_BIND_KEYPAIR              = "vm_bind_keypair"
	ACT_VM_CHANGE_FLAVOR             = "vm_change_flavor"
	ACT_VM_DEPLOY                    = "vm_deploy"
	ACT_VM_DETACH_DISK               = "vm_detach_disk"
	ACT_VM_PURGE                     = "vm_purge"
	ACT_VM_REBUILD                   = "vm_rebuild"
	ACT_VM_RESET_PSWD                = "vm_reset_pswd"
	ACT_VM_CHANGE_BANDWIDTH          = "vm_change_bandwidth"
	ACT_VM_SRC_CHECK                 = "vm_src_check"
	ACT_VM_START                     = "vm_start"
	ACT_VM_STOP                      = "vm_stop"
	ACT_VM_RESTART                   = "vm_restart"
	ACT_VM_SYNC_CONF                 = "vm_sync_conf"
	ACT_VM_SYNC_STATUS               = "vm_sync_status"
	ACT_VM_UNBIND_KEYPAIR            = "vm_unbind_keypair"
	ACT_VM_ASSIGNSECGROUP            = "vm_assignsecgroup"
	ACT_VM_REVOKESECGROUP            = "vm_revokesecgroup"
	ACT_VM_SETSECGROUP               = "vm_setsecgroup"
	ACT_RESET_DISK                   = "reset_disk"
	ACT_SYNC_STATUS                  = "sync_status"
	ACT_SYNC_CONF                    = "sync_conf"
	ACT_CREATE_BACKUP                = "create_backup"
	ACT_SWITCH_TO_BACKUP             = "switch_to_backup"
	ACT_RENEW                        = "renew"
	ACT_SET_AUTO_RENEW               = "set_auto_renew"
	ACT_MIGRATE                      = "migrate"
	ACT_EIP_ASSOCIATE                = "eip_associate"
	ACT_EIP_DISSOCIATE               = "eip_dissociate"
	ACT_EIP_CONVERT                  = "eip_convert"
	ACT_CHANGE_BANDWIDTH             = "change_bandwidth"
	ACT_DISK_CREATE_SNAPSHOT         = "disk_create_snapshot"
	ACT_LB_ADD_BACKEND               = "lb_add_backend"
	ACT_LB_REMOVE_BACKEND            = "lb_remove_backend"
	ACL_LB_SYNC_BACKEND_CONF         = "lb_sync_backend_conf"
	ACT_LB_ADD_LISTENER_RULE         = "lb_add_listener_rule"
	ACT_LB_REMOVE_LISTENER_RULE      = "lb_remove_listener_rule"
	ACT_DELETE_BACKUP                = "delete_backup"
	ACT_APPLY_SNAPSHOT_POLICY        = "apply_snapshot_policy"
	ACT_CANCEL_SNAPSHOT_POLICY       = "cancel_snapshot_policy"
	ACT_BIND_DISK                    = "bind_disk"
	ACT_UNBIND_DISK                  = "unbind_disk"
	ACT_ATTACH_HOST                  = "attach_host"
	ACT_DETACH_HOST                  = "detach_host"
	ACT_VM_IO_THROTTLE               = "vm_io_throttle"
	ACT_VM_RESET                     = "vm_reset"
	ACT_VM_SNAPSHOT_AND_CLONE        = "vm_snapshot_and_clone"
	ACT_VM_BLOCK_STREAM              = "vm_block_stream"
	ACT_ATTACH_NETWORK               = "attach_network"
	ACT_VM_CONVERT                   = "vm_convert"
	ACT_FREEZE                       = "freeze"
	ACT_UNFREEZE                     = "unfreeze"

	ACT_CACHED_IMAGE = "cached_image"

	ACT_REBOOT        = "reboot"
	ACT_CHANGE_CONFIG = "change_config"

	ACT_OPEN_PUBLIC_CONNECTION  = "open_public_connection"
	ACT_CLOSE_PUBLIC_CONNECTION = "close_public_connection"

	ACT_IMAGE_SAVE  = "image_save"
	ACT_IMAGE_PROBE = "image_probe"

	ACT_AUTHENTICATE = "authenticate"

	ACT_HEALTH_CHECK = "health_check"

	ACT_RECYCLE_PREPAID      = "recycle_prepaid"
	ACT_UNDO_RECYCLE_PREPAID = "undo_recycle_prepaid"

	ACT_FETCH = "fetch"

	ACT_VM_CHANGE_NIC = "vm_change_nic"

	ACT_HOST_IMPORT_LIBVIRT_SERVERS = "host_import_libvirt_servers"
	ACT_GUEST_CREATE_FROM_IMPORT    = "guest_create_from_import"
	ACT_GUEST_PANICKED              = "guest_panicked"
	ACT_HOST_MAINTAINING            = "host_maintaining"

	ACT_MKDIR          = "mkdir"
	ACT_DELETE_OBJECT  = "delete_object"
	ACT_UPLOAD_OBJECT  = "upload_object"
	ACT_SET_WEBSITE    = "set_website"
	ACT_DELETE_WEBSITE = "delete_website"
	ACT_SET_CORS       = "set_cors"
	ACT_DELETE_CORS    = "delete_cors"
	ACT_SET_REFERER    = "set_referer"
	ACT_SET_POLICY     = "set_policy"
	ACT_DELETE_POLICY  = "delete_policy"

	ACT_NAT_CREATE_SNAT = "nat_create_snat"
	ACT_NAT_CREATE_DNAT = "nat_create_dnat"
	ACT_NAT_DELETE_SNAT = "nat_delete_snat"
	ACT_NAT_DELETE_DNAT = "nat_delete_dnat"

	ACT_GRANT_PRIVILEGE  = "grant_privilege"
	ACT_REVOKE_PRIVILEGE = "revoke_privilege"
	ACT_SET_PRIVILEGES   = "set_privileges"
	ACT_RESTORE          = "restore"
	ACT_RESET_PASSWORD   = "reset_password"

	ACT_VM_ASSOCIATE            = "vm_associate"
	ACT_VM_DISSOCIATE           = "vm_dissociate"
	ACT_NATGATEWAY_DISSOCIATE   = "natgateway_dissociate"
	ACT_LOADBALANCER_DISSOCIATE = "loadbalancer_dissociate"

	ACT_PREPARE = "prepare"
	ACT_PROBE   = "probe"

	ACT_INSTANCE_GROUP_BIND   = "instance_group_bind"
	ACT_INSTANCE_GROUP_UNBIND = "instance_group_unbind"

	ACT_FLUSH_INSTANCE = "flush_instance"

	ACT_UPDATE_STATUS = "update_status"

	ACT_UPDATE_PASSWORD = "update_password"

	ACT_REMOVE_GUEST          = "remove_guest"
	ACT_CREATE_SCALING_POLICY = "create_scaling_policy"
	ACT_DELETE_SCALING_POLICY = "delete_scaling_policy"

	ACT_SAVE_TO_TEMPLATE = "save_to_template"

	ACT_SYNC_POLICIES = "sync_policies"
	ACT_SYNC_USERS    = "sync_users"
	ACT_ADD_USER      = "add_user"
	ACT_REMOVE_USER   = "remove_user"
	ACT_ATTACH_POLICY = "attach_policy"
	ACT_DETACH_POLICY = "detach_policy"

	ACT_UPDATE_BILLING_OPTIONS = "update_billing_options"
	ACT_UPDATE_CREDENTIAL      = "update_credential"

	ACT_PULL_SUBCONTACT   = "pull_subcontact"
	ACT_SEND_NOTIFICATION = "send_notification"
	ACT_SEND_VERIFICATION = "send_verification"
	ACT_REPULL_SUBCONTACT = "repull_subcontact"

	ACT_SYNC_VPCS        = "sync_vpcs"
	ACT_SYNC_RECORD_SETS = "sync_record_sets"

	ACT_DETACH_ALERTRESOURCE = "detach_alertresoruce"
	ACT_NETWORK_ADD_VPC      = "network_add_vpc"
	ACT_NETWORK_REMOVE_VPC   = "network_remove_vpc"
	ACT_NETWORK_MODIFY_ROUTE = "network_modify_route"

	ACT_UPDATE_RULE = "update_config"
	ACT_UPDATE_TAGS = "update_tags"
)
