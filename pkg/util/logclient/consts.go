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

// 这些状态不做 websocket 通知
var BlackList = []string{
	ACT_CLOUD_FULLSYNC,
	ACT_CREATE,
	ACT_DELETE,
	ACT_PENDING_DELETE,
	ACT_PRIVATE,
	ACT_PUBLIC,
	ACT_UPDATE,
	ACT_VM_SYNC_STATUS,
	ACT_VM_SYNC_CONF,
}

// 这些状态需要做 websocket 通知
var WhiteList = []string{
	ACT_ADDTAG, ACT_ALLOCATE, ACT_DELOCATE, ACT_BM_CONVERT_HYPER,
	ACT_BM_MAINTENANCE, ACT_BM_UNCONVERT_HYPER, ACT_BM_UNMAINTENANCE,
	ACT_CANCEL_DELETE, ACT_CHANGE_OWNER, ACT_SYNC_CLOUD_OWNER,
	ACT_CLOUD_SYNC, ACT_DISABLE, ACT_ENABLE,
	ACT_GUEST_ATTACH_ISOLATED_DEVICE, ACT_GUEST_DETACH_ISOLATED_DEVICE,
	ACT_MERGE, ACT_OFFLINE, ACT_ONLINE, ACT_RELEASE_IP,
	ACT_RESERVE_IP, ACT_RESIZE, ACT_RMTAG, ACT_SPLIT,
	ACT_UNCACHED_IMAGE, ACT_VM_ATTACH_DISK, ACT_VM_BIND_KEYPAIR,
	ACT_VM_CHANGE_FLAVOR, ACT_VM_DEPLOY, ACT_VM_DETACH_DISK,
	ACT_VM_PURGE, ACT_VM_REBUILD, ACT_VM_RESET_PSWD,
	ACT_VM_CHANGE_BANDWIDTH, ACT_VM_START, ACT_VM_STOP,
	ACT_VM_RESTART, ACT_VM_UNBIND_KEYPAIR, ACT_VM_ASSIGNSECGROUP,
	ACT_VM_REVOKESECGROUP, ACT_VM_SETSECGROUP, ACT_RESET_DISK,
	ACT_SYNC_STATUS, ACT_SYNC_CONF, ACT_CREATE_BACKUP,
	ACT_SWITCH_TO_BACKUP, ACT_RENEW, ACT_MIGRATE,
	ACT_IMAGE_SAVE, ACT_RECYCLE_PREPAID, ACT_UNDO_RECYCLE_PREPAID,
	ACT_FETCH, ACT_VM_CHANGE_NIC, ACT_HOST_IMPORT_LIBVIRT_SERVERS,
	ACT_GUEST_CREATE_FROM_IMPORT, ACT_DISK_CREATE_SNAPSHOT,
	ACT_IMAGE_PROBE,
}

const (
	ACT_ADDTAG                       = "添加标签"
	ACT_ALLOCATE                     = "分配"
	ACT_DELOCATE                     = "释放资源"
	ACT_BM_CONVERT_HYPER             = "转换为宿主机"
	ACT_BM_MAINTENANCE               = "进入离线状态"
	ACT_BM_UNCONVERT_HYPER           = "转换为受管物理机"
	ACT_BM_UNMAINTENANCE             = "退出离线状态"
	ACT_CANCEL_DELETE                = "恢复"
	ACT_CHANGE_OWNER                 = "更改项目"
	ACT_SYNC_CLOUD_OWNER             = "同步云项目"
	ACT_CLOUD_FULLSYNC               = "全量同步"
	ACT_CLOUD_SYNC                   = "同步"
	ACT_CREATE                       = "创建"
	ACT_DELETE                       = "删除"
	ACT_PENDING_DELETE               = "预删除"
	ACT_DISABLE                      = "禁用"
	ACT_ENABLE                       = "启用"
	ACT_GUEST_ATTACH_ISOLATED_DEVICE = "挂载透传设备"
	ACT_GUEST_DETACH_ISOLATED_DEVICE = "卸载透传设备"
	ACT_MERGE                        = "合并"
	ACT_OFFLINE                      = "下线"
	ACT_ONLINE                       = "上线"
	ACT_PRIVATE                      = "设为私有"
	ACT_PUBLIC                       = "设为共享"
	ACT_RELEASE_IP                   = "释放IP"
	ACT_RESERVE_IP                   = "预留IP"
	ACT_RESIZE                       = "扩容"
	ACT_RMTAG                        = "删除标签"
	ACT_SPLIT                        = "分割"
	ACT_UNCACHED_IMAGE               = "清除缓存"
	ACT_UPDATE                       = "更新"
	ACT_VM_ATTACH_DISK               = "挂载磁盘"
	ACT_VM_BIND_KEYPAIR              = "绑定密钥"
	ACT_VM_CHANGE_FLAVOR             = "调整配置"
	ACT_VM_DEPLOY                    = "部署"
	ACT_VM_DETACH_DISK               = "卸载磁盘"
	ACT_VM_PURGE                     = "清除"
	ACT_VM_REBUILD                   = "重装系统"
	ACT_VM_RESET_PSWD                = "重置密码"
	ACT_VM_CHANGE_BANDWIDTH          = "调整带宽"
	ACT_VM_START                     = "开机"
	ACT_VM_STOP                      = "关机"
	ACT_VM_RESTART                   = "重启"
	ACT_VM_SYNC_CONF                 = "同步配置"
	ACT_VM_SYNC_STATUS               = "同步状态"
	ACT_VM_UNBIND_KEYPAIR            = "解绑密钥"
	ACT_VM_ASSIGNSECGROUP            = "关联安全组"
	ACT_VM_REVOKESECGROUP            = "取消关联安全组"
	ACT_VM_SETSECGROUP               = "设置安全组"
	ACT_RESET_DISK                   = "回滚磁盘"
	ACT_SYNC_STATUS                  = "同步状态"
	ACT_SYNC_CONF                    = "同步配置"
	ACT_CREATE_BACKUP                = "创建备份机"
	ACT_SWITCH_TO_BACKUP             = "主备切换"
	ACT_RENEW                        = "续费"
	ACT_MIGRATE                      = "迁移"
	ACT_EIP_ASSOCIATE                = "绑定弹性IP"
	ACT_EIP_DISSOCIATE               = "解绑弹性IP"
	ACT_CHANGE_BANDWIDTH             = "调整带宽"
	ACT_DISK_CREATE_SNAPSHOT         = "磁盘创建快照"
	ACT_LB_ADD_BACKEND               = "添加后端服务器"
	ACT_LB_REMOVE_BACKEND            = "移除后端服务器"
	ACL_LB_SYNC_BACKEND_CONF         = "同步后端服务器配置"
	ACT_LB_ADD_LISTENER_RULE         = "添加负载均衡转发规则"
	ACT_LB_REMOVE_LISTENER_RULE      = "移除负载均衡转发规则"
	ACT_DELETE_BACKUP                = "删除备份机"
	ACT_APPLY_SNAPSHOT_POLICY        = "绑定快照策略"
	ACT_CANCEL_SNAPSHOT_POLICY       = "取消快照策略"
	ACT_BIND_DISK                    = "绑定磁盘"
	ACT_UNBIND_DISK                  = "解绑磁盘"
	ACT_ATTACH_HOST                  = "关联宿主机"
	ACT_DETACH_HOST                  = "取消关联宿主机"
	ACT_VM_IO_THROTTLE               = "虚拟机磁盘限速"
	ACT_VM_RESET                     = "虚拟机回滚快照"
	ACT_VM_SNAPSHOT_AND_CLONE        = "虚拟机快照并克隆"

	ACT_REBOOT        = "重启"
	ACT_CHANGE_CONFIG = "调整配置"

	ACT_OPEN_PUBLIC_CONNECTION  = "打开外网地址"
	ACT_CLOSE_PUBLIC_CONNECTION = "关闭外网地址"

	ACT_IMAGE_SAVE  = "上传镜像"
	ACT_IMAGE_PROBE = "镜像检测"

	ACT_AUTHENTICATE = "认证登录"

	ACT_RECYCLE_PREPAID      = "池化预付费主机"
	ACT_UNDO_RECYCLE_PREPAID = "取消池化预付费主机"

	ACT_FETCH = "下载密钥"

	ACT_VM_CHANGE_NIC = "更改网卡配置"

	ACT_HOST_IMPORT_LIBVIRT_SERVERS = "libvirt托管虚拟机导入"
	ACT_GUEST_CREATE_FROM_IMPORT    = "导入虚拟机创建"
	ACT_GUEST_PANICKED              = "GuestPanicked"
	ACT_HOST_MAINTAINING            = "宿主机进入维护模式"

	ACT_MKDIR         = "创建目录"
	ACT_DELETE_OBJECT = "删除对象"
	ACT_UPLOAD_OBJECT = "上传对象"

	ACT_NAT_CREATE_SNAT = "创建SNAT规则"
	ACT_NAT_CREATE_DNAT = "创建DNAT规则"
	ACT_NAT_DELETE_SNAT = "删除SNAT规则"
	ACT_NAT_DELETE_DNAT = "删除DNAT规则"

	ACT_GRANT_PRIVILEGE  = "赋予权限"
	ACT_REVOKE_PRIVILEGE = "解除权限"
	ACT_SET_PRIVILEGES   = "设置权限"
	ACT_RESTORE          = "备份恢复"
	ACT_RESET_PASSWORD   = "重置密码"

	ACT_VM_ASSOCIATE            = "绑定虚拟机"
	ACT_VM_DISSOCIATE           = "解绑虚拟机"
	ACT_NATGATEWAY_DISSOCIATE   = "解绑NAT网关"
	ACT_LOADBALANCER_DISSOCIATE = "解绑负载均衡"

	ACT_SUBIMAGE_UPDATE = "更新子镜像"

	ACT_PREPARE = "同步硬件配置"

	ACT_INSTANCE_GROUP_BIND   = "绑定主机组"
	ACT_INSTANCE_GROUP_UNBIND = "解绑主机组"
)
