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

import (
	"yunion.io/x/onecloud/pkg/i18n"
)

var OpsActionI18nTable = i18n.Table{}

func init() {
	t := OpsActionI18nTable

	t.Set(ACT_ADDTAG, i18n.NewTableEntry().
		EN("Addtag").
		CN("添加标签"),
	)
	t.Set(ACT_ALLOCATE, i18n.NewTableEntry().
		EN("Allocate").
		CN("分配"),
	)
	t.Set(ACT_DELOCATE, i18n.NewTableEntry().
		EN("Delocate").
		CN("释放资源"),
	)
	t.Set(ACT_BM_CONVERT_HYPER, i18n.NewTableEntry().
		EN("Bm Convert Hyper").
		CN("转换为宿主机"),
	)
	t.Set(ACT_BM_MAINTENANCE, i18n.NewTableEntry().
		EN("Bm Maintenance").
		CN("进入离线状态"),
	)
	t.Set(ACT_BM_UNCONVERT_HYPER, i18n.NewTableEntry().
		EN("Bm Unconvert Hyper").
		CN("转换为受管物理机"),
	)
	t.Set(ACT_BM_UNMAINTENANCE, i18n.NewTableEntry().
		EN("Bm Unmaintenance").
		CN("退出离线状态"),
	)
	t.Set(ACT_CANCEL_DELETE, i18n.NewTableEntry().
		EN("Cancel Delete").
		CN("恢复"),
	)
	t.Set(ACT_CHANGE_OWNER, i18n.NewTableEntry().
		EN("Change Owner").
		CN("更改项目"),
	)
	t.Set(ACT_SYNC_CLOUD_OWNER, i18n.NewTableEntry().
		EN("Sync Cloud Owner").
		CN("同步云项目"),
	)
	t.Set(ACT_CLOUD_FULLSYNC, i18n.NewTableEntry().
		EN("Cloud Fullsync").
		CN("全量同步"),
	)
	t.Set(ACT_CLOUD_SYNC, i18n.NewTableEntry().
		EN("Cloud Sync").
		CN("同步"),
	)
	t.Set(ACT_CREATE, i18n.NewTableEntry().
		EN("Create").
		CN("创建"),
	)
	t.Set(ACT_DELETE, i18n.NewTableEntry().
		EN("Delete").
		CN("删除"),
	)
	t.Set(ACT_PENDING_DELETE, i18n.NewTableEntry().
		EN("Pending Delete").
		CN("预删除"),
	)
	t.Set(ACT_DISABLE, i18n.NewTableEntry().
		EN("Disable").
		CN("禁用"),
	)
	t.Set(ACT_ENABLE, i18n.NewTableEntry().
		EN("Enable").
		CN("启用"),
	)
	t.Set(ACT_GUEST_ATTACH_ISOLATED_DEVICE, i18n.NewTableEntry().
		EN("Guest Attach Isolated Device").
		CN("挂载透传设备"),
	)
	t.Set(ACT_GUEST_DETACH_ISOLATED_DEVICE, i18n.NewTableEntry().
		EN("Guest Detach Isolated Device").
		CN("卸载透传设备"),
	)
	t.Set(ACT_MERGE, i18n.NewTableEntry().
		EN("Merge").
		CN("合并"),
	)
	t.Set(ACT_OFFLINE, i18n.NewTableEntry().
		EN("Offline").
		CN("下线"),
	)
	t.Set(ACT_ONLINE, i18n.NewTableEntry().
		EN("Online").
		CN("上线"),
	)
	t.Set(ACT_PRIVATE, i18n.NewTableEntry().
		EN("Private").
		CN("设为私有"),
	)
	t.Set(ACT_PUBLIC, i18n.NewTableEntry().
		EN("Public").
		CN("设为共享"),
	)
	t.Set(ACT_RELEASE_IP, i18n.NewTableEntry().
		EN("Release Ip").
		CN("释放IP"),
	)
	t.Set(ACT_RESERVE_IP, i18n.NewTableEntry().
		EN("Reserve Ip").
		CN("预留IP"),
	)
	t.Set(ACT_RESIZE, i18n.NewTableEntry().
		EN("Resize").
		CN("扩容"),
	)
	t.Set(ACT_RMTAG, i18n.NewTableEntry().
		EN("Rmtag").
		CN("删除标签"),
	)
	t.Set(ACT_SPLIT, i18n.NewTableEntry().
		EN("Split").
		CN("分割"),
	)
	t.Set(ACT_UNCACHED_IMAGE, i18n.NewTableEntry().
		EN("Uncached Image").
		CN("清除缓存"),
	)
	t.Set(ACT_UPDATE, i18n.NewTableEntry().
		EN("Update").
		CN("更新"),
	)
	t.Set(ACT_VM_ATTACH_DISK, i18n.NewTableEntry().
		EN("Vm Attach Disk").
		CN("挂载磁盘"),
	)
	t.Set(ACT_VM_BIND_KEYPAIR, i18n.NewTableEntry().
		EN("Vm Bind Keypair").
		CN("绑定密钥"),
	)
	t.Set(ACT_VM_CHANGE_FLAVOR, i18n.NewTableEntry().
		EN("Vm Change Flavor").
		CN("调整配置"),
	)
	t.Set(ACT_VM_DEPLOY, i18n.NewTableEntry().
		EN("Vm Deploy").
		CN("部署"),
	)
	t.Set(ACT_VM_DETACH_DISK, i18n.NewTableEntry().
		EN("Vm Detach Disk").
		CN("卸载磁盘"),
	)
	t.Set(ACT_VM_PURGE, i18n.NewTableEntry().
		EN("Vm Purge").
		CN("清除"),
	)
	t.Set(ACT_VM_REBUILD, i18n.NewTableEntry().
		EN("Vm Rebuild").
		CN("重装系统"),
	)
	t.Set(ACT_VM_RESET_PSWD, i18n.NewTableEntry().
		EN("Vm Reset Pswd").
		CN("重置密码"),
	)
	t.Set(ACT_VM_CHANGE_BANDWIDTH, i18n.NewTableEntry().
		EN("Vm Change Bandwidth").
		CN("调整带宽"),
	)
	t.Set(ACT_VM_SRC_CHECK, i18n.NewTableEntry().
		EN("Vm Src Check").
		CN("调整源IP、MAC地址检查"),
	)
	t.Set(ACT_VM_START, i18n.NewTableEntry().
		EN("Vm Start").
		CN("开机"),
	)
	t.Set(ACT_VM_STOP, i18n.NewTableEntry().
		EN("Vm Stop").
		CN("关机"),
	)
	t.Set(ACT_VM_RESTART, i18n.NewTableEntry().
		EN("Vm Restart").
		CN("重启"),
	)
	t.Set(ACT_VM_SYNC_CONF, i18n.NewTableEntry().
		EN("Vm Sync Conf").
		CN("同步配置"),
	)
	t.Set(ACT_VM_SYNC_STATUS, i18n.NewTableEntry().
		EN("Vm Sync Status").
		CN("同步状态"),
	)
	t.Set(ACT_VM_UNBIND_KEYPAIR, i18n.NewTableEntry().
		EN("Vm Unbind Keypair").
		CN("解绑密钥"),
	)
	t.Set(ACT_VM_ASSIGNSECGROUP, i18n.NewTableEntry().
		EN("Vm Assignsecgroup").
		CN("关联安全组"),
	)
	t.Set(ACT_VM_REVOKESECGROUP, i18n.NewTableEntry().
		EN("Vm Revokesecgroup").
		CN("取消关联安全组"),
	)
	t.Set(ACT_VM_SETSECGROUP, i18n.NewTableEntry().
		EN("Vm Setsecgroup").
		CN("设置安全组"),
	)
	t.Set(ACT_RESET_DISK, i18n.NewTableEntry().
		EN("Reset Disk").
		CN("回滚磁盘"),
	)
	t.Set(ACT_SYNC_STATUS, i18n.NewTableEntry().
		EN("Sync Status").
		CN("同步状态"),
	)
	t.Set(ACT_SYNC_CONF, i18n.NewTableEntry().
		EN("Sync Conf").
		CN("同步配置"),
	)
	t.Set(ACT_CREATE_BACKUP, i18n.NewTableEntry().
		EN("Create Backup").
		CN("创建备份机"),
	)
	t.Set(ACT_SWITCH_TO_BACKUP, i18n.NewTableEntry().
		EN("Switch To Backup").
		CN("主备切换"),
	)
	t.Set(ACT_RENEW, i18n.NewTableEntry().
		EN("Renew").
		CN("续费"),
	)
	t.Set(ACT_SET_AUTO_RENEW, i18n.NewTableEntry().
		EN("Set Auto Renew").
		CN("设置自动续费"),
	)
	t.Set(ACT_MIGRATE, i18n.NewTableEntry().
		EN("Migrate").
		CN("迁移"),
	)
	t.Set(ACT_EIP_ASSOCIATE, i18n.NewTableEntry().
		EN("Eip Associate").
		CN("绑定弹性IP"),
	)
	t.Set(ACT_EIP_DISSOCIATE, i18n.NewTableEntry().
		EN("Eip Dissociate").
		CN("解绑弹性IP"),
	)
	t.Set(ACT_EIP_CONVERT, i18n.NewTableEntry().
		EN("Eip Convert").
		CN("弹性IP转换"),
	)
	t.Set(ACT_CHANGE_BANDWIDTH, i18n.NewTableEntry().
		EN("Change Bandwidth").
		CN("调整带宽"),
	)
	t.Set(ACT_DISK_CREATE_SNAPSHOT, i18n.NewTableEntry().
		EN("Disk Create Snapshot").
		CN("磁盘创建快照"),
	)
	t.Set(ACT_LB_ADD_BACKEND, i18n.NewTableEntry().
		EN("Lb Add Backend").
		CN("添加后端服务器"),
	)
	t.Set(ACT_LB_REMOVE_BACKEND, i18n.NewTableEntry().
		EN("Lb Remove Backend").
		CN("移除后端服务器"),
	)
	t.Set(ACL_LB_SYNC_BACKEND_CONF, i18n.NewTableEntry().
		EN("Lb Sycn Backend Conf").
		CN("同步后端服务器配置"),
	)
	t.Set(ACT_LB_ADD_LISTENER_RULE, i18n.NewTableEntry().
		EN("Lb Add Listener Rule").
		CN("添加负载均衡转发规则"),
	)
	t.Set(ACT_LB_REMOVE_LISTENER_RULE, i18n.NewTableEntry().
		EN("Lb Remove Listener Rule").
		CN("移除负载均衡转发规则"),
	)
	t.Set(ACT_DELETE_BACKUP, i18n.NewTableEntry().
		EN("Delete Backup").
		CN("删除备份机"),
	)
	t.Set(ACT_APPLY_SNAPSHOT_POLICY, i18n.NewTableEntry().
		EN("Apply Snapshot Policy").
		CN("绑定快照策略"),
	)
	t.Set(ACT_CANCEL_SNAPSHOT_POLICY, i18n.NewTableEntry().
		EN("Cancel Snapshot Policy").
		CN("取消快照策略"),
	)
	t.Set(ACT_BIND_DISK, i18n.NewTableEntry().
		EN("Bind Disk").
		CN("绑定磁盘"),
	)
	t.Set(ACT_UNBIND_DISK, i18n.NewTableEntry().
		EN("Unbind Disk").
		CN("解绑磁盘"),
	)
	t.Set(ACT_ATTACH_HOST, i18n.NewTableEntry().
		EN("Attach Host").
		CN("关联宿主机"),
	)
	t.Set(ACT_DETACH_HOST, i18n.NewTableEntry().
		EN("Detach Host").
		CN("取消关联宿主机"),
	)
	t.Set(ACT_VM_IO_THROTTLE, i18n.NewTableEntry().
		EN("Vm Io Throttle").
		CN("虚拟机磁盘限速"),
	)
	t.Set(ACT_VM_RESET, i18n.NewTableEntry().
		EN("Vm Reset").
		CN("虚拟机回滚快照"),
	)
	t.Set(ACT_VM_SNAPSHOT_AND_CLONE, i18n.NewTableEntry().
		EN("Vm Snapshot And Clone").
		CN("虚拟机快照并克隆"),
	)
	t.Set(ACT_VM_BLOCK_STREAM, i18n.NewTableEntry().
		EN("Vm Block Stream").
		CN("同步数据"),
	)
	t.Set(ACT_ATTACH_NETWORK, i18n.NewTableEntry().
		EN("Attach Network").
		CN("绑定网卡"),
	)
	t.Set(ACT_VM_CONVERT, i18n.NewTableEntry().
		EN("Vm Convert").
		CN("虚拟机转换Hypervisor"),
	)

	t.Set(ACT_CACHED_IMAGE, i18n.NewTableEntry().
		EN("Cached Image").
		CN("缓存镜像"),
	)

	t.Set(ACT_REBOOT, i18n.NewTableEntry().
		EN("Reboot").
		CN("重启"),
	)
	t.Set(ACT_CHANGE_CONFIG, i18n.NewTableEntry().
		EN("Change Config").
		CN("调整配置"),
	)

	t.Set(ACT_OPEN_PUBLIC_CONNECTION, i18n.NewTableEntry().
		EN("Open Public Connection").
		CN("打开外网地址"),
	)
	t.Set(ACT_CLOSE_PUBLIC_CONNECTION, i18n.NewTableEntry().
		EN("Close Public Connection").
		CN("关闭外网地址"),
	)

	t.Set(ACT_IMAGE_SAVE, i18n.NewTableEntry().
		EN("Image Save").
		CN("上传镜像"),
	)
	t.Set(ACT_IMAGE_PROBE, i18n.NewTableEntry().
		EN("Image Probe").
		CN("镜像检测"),
	)

	t.Set(ACT_AUTHENTICATE, i18n.NewTableEntry().
		EN("Authenticate").
		CN("认证登录"),
	)

	t.Set(ACT_HEALTH_CHECK, i18n.NewTableEntry().
		EN("Health Check").
		CN("健康检查"),
	)

	t.Set(ACT_RECYCLE_PREPAID, i18n.NewTableEntry().
		EN("Recycle Prepaid").
		CN("池化预付费主机"),
	)
	t.Set(ACT_UNDO_RECYCLE_PREPAID, i18n.NewTableEntry().
		EN("Undo Recycle Prepaid").
		CN("取消池化预付费主机"),
	)

	t.Set(ACT_FETCH, i18n.NewTableEntry().
		EN("Fetch").
		CN("下载密钥"),
	)

	t.Set(ACT_VM_CHANGE_NIC, i18n.NewTableEntry().
		EN("Vm Change Nic").
		CN("更改网卡配置"),
	)

	t.Set(ACT_HOST_IMPORT_LIBVIRT_SERVERS, i18n.NewTableEntry().
		EN("Host Import Libvirt Servers").
		CN("libvirt托管虚拟机导入"),
	)
	t.Set(ACT_GUEST_CREATE_FROM_IMPORT, i18n.NewTableEntry().
		EN("Guest Create From Import").
		CN("导入虚拟机创建"),
	)
	t.Set(ACT_GUEST_PANICKED, i18n.NewTableEntry().
		EN("Guest Panicked").
		CN("GuestPanicked"),
	)
	t.Set(ACT_HOST_MAINTAINING, i18n.NewTableEntry().
		EN("Host Maintaining").
		CN("宿主机进入维护模式"),
	)

	t.Set(ACT_MKDIR, i18n.NewTableEntry().
		EN("Mkdir").
		CN("创建目录"),
	)
	t.Set(ACT_DELETE_OBJECT, i18n.NewTableEntry().
		EN("Delete Object").
		CN("删除对象"),
	)
	t.Set(ACT_UPLOAD_OBJECT, i18n.NewTableEntry().
		EN("Upload Object").
		CN("上传对象"),
	)
	t.Set(ACT_SET_WEBSITE, i18n.NewTableEntry().
		EN("Set Static Website").
		CN("设置静态网站"),
	)
	t.Set(ACT_DELETE_WEBSITE, i18n.NewTableEntry().
		EN("Delete Static Website").
		CN("删除静态网站"),
	)
	t.Set(ACT_SET_CORS, i18n.NewTableEntry().
		EN("Set CORS").
		CN("设置CORS"),
	)
	t.Set(ACT_DELETE_CORS, i18n.NewTableEntry().
		EN("Delete CORS").
		CN("删除CORS"),
	)
	t.Set(ACT_SET_REFERER, i18n.NewTableEntry().
		EN("Set Referer").
		CN("设置Referer"),
	)
	t.Set(ACT_SET_POLICY, i18n.NewTableEntry().
		EN("Set Policy").
		CN("设置Policy"),
	)
	t.Set(ACT_DELETE_POLICY, i18n.NewTableEntry().
		EN("Delete Policy").
		CN("删除Policy"),
	)

	t.Set(ACT_NAT_CREATE_SNAT, i18n.NewTableEntry().
		EN("Nat Create Snat").
		CN("创建SNAT规则"),
	)
	t.Set(ACT_NAT_CREATE_DNAT, i18n.NewTableEntry().
		EN("Nat Create Dnat").
		CN("创建DNAT规则"),
	)
	t.Set(ACT_NAT_DELETE_SNAT, i18n.NewTableEntry().
		EN("Nat Delete Snat").
		CN("删除SNAT规则"),
	)
	t.Set(ACT_NAT_DELETE_DNAT, i18n.NewTableEntry().
		EN("Nat Delete Dnat").
		CN("删除DNAT规则"),
	)

	t.Set(ACT_GRANT_PRIVILEGE, i18n.NewTableEntry().
		EN("Grant Privilege").
		CN("赋予权限"),
	)
	t.Set(ACT_REVOKE_PRIVILEGE, i18n.NewTableEntry().
		EN("Revoke Privilege").
		CN("解除权限"),
	)
	t.Set(ACT_SET_PRIVILEGES, i18n.NewTableEntry().
		EN("Set Privileges").
		CN("设置权限"),
	)
	t.Set(ACT_RESTORE, i18n.NewTableEntry().
		EN("Restore").
		CN("备份恢复"),
	)
	t.Set(ACT_RESET_PASSWORD, i18n.NewTableEntry().
		EN("Reset Password").
		CN("重置密码"),
	)

	t.Set(ACT_VM_ASSOCIATE, i18n.NewTableEntry().
		EN("Vm Associate").
		CN("绑定虚拟机"),
	)
	t.Set(ACT_VM_DISSOCIATE, i18n.NewTableEntry().
		EN("Vm Dissociate").
		CN("解绑虚拟机"),
	)
	t.Set(ACT_NATGATEWAY_DISSOCIATE, i18n.NewTableEntry().
		EN("Natgateway Dissociate").
		CN("解绑NAT网关"),
	)
	t.Set(ACT_LOADBALANCER_DISSOCIATE, i18n.NewTableEntry().
		EN("Loadbalancer Dissociate").
		CN("解绑负载均衡"),
	)

	t.Set(ACT_PREPARE, i18n.NewTableEntry().
		EN("Prepare").
		CN("同步硬件配置"),
	)
	t.Set(ACT_PROBE, i18n.NewTableEntry().
		EN("Probe").
		CN("检测配置"),
	)

	t.Set(ACT_INSTANCE_GROUP_BIND, i18n.NewTableEntry().
		EN("Instance Group Bind").
		CN("绑定主机组"),
	)
	t.Set(ACT_INSTANCE_GROUP_UNBIND, i18n.NewTableEntry().
		EN("Instance Group Unbind").
		CN("解绑主机组"),
	)

	t.Set(ACT_FLUSH_INSTANCE, i18n.NewTableEntry().
		EN("Flush Instance").
		CN("清空数据"),
	)

	t.Set(ACT_UPDATE_STATUS, i18n.NewTableEntry().
		EN("Update Status").
		CN("更新状态"),
	)

	t.Set(ACT_UPDATE_PASSWORD, i18n.NewTableEntry().
		EN("Update Password").
		CN("更新密码"),
	)

	t.Set(ACT_REMOVE_GUEST, i18n.NewTableEntry().
		EN("Remove Guest").
		CN("移除实例"),
	)
	t.Set(ACT_CREATE_SCALING_POLICY, i18n.NewTableEntry().
		EN("Create Scaling Policy").
		CN("创建伸缩策略"),
	)
	t.Set(ACT_DELETE_SCALING_POLICY, i18n.NewTableEntry().
		EN("Delete Scaling Policy").
		CN("删除伸缩策略"),
	)

	t.Set(ACT_SAVE_TO_TEMPLATE, i18n.NewTableEntry().
		EN("Save To Template").
		CN("保存为模版"),
	)

	t.Set(ACT_SYNC_POLICIES, i18n.NewTableEntry().
		EN("Sync Policies").
		CN("同步权限"),
	)
	t.Set(ACT_SYNC_USERS, i18n.NewTableEntry().
		EN("Sync Users").
		CN("同步用户"),
	)
	t.Set(ACT_ADD_USER, i18n.NewTableEntry().
		EN("Add User").
		CN("添加用户"),
	)
	t.Set(ACT_REMOVE_USER, i18n.NewTableEntry().
		EN("Remove User").
		CN("移除用户"),
	)
	t.Set(ACT_ATTACH_POLICY, i18n.NewTableEntry().
		EN("Attach Policy").
		CN("绑定权限"),
	)
	t.Set(ACT_DETACH_POLICY, i18n.NewTableEntry().
		EN("Detach Policy").
		CN("移除权限"),
	)

	t.Set(ACT_UPDATE_BILLING_OPTIONS, i18n.NewTableEntry().
		EN("Update Billing Options").
		CN("更新账单文件"),
	)
	t.Set(ACT_UPDATE_CREDENTIAL, i18n.NewTableEntry().
		EN("Update Credential").
		CN("更新账号密码"),
	)

	t.Set(ACT_PULL_SUBCONTACT, i18n.NewTableEntry().
		EN("Pull Subcontact").
		CN("拉取联系方式"),
	)
	t.Set(ACT_SEND_NOTIFICATION, i18n.NewTableEntry().
		EN("Send Notification").
		CN("发送通知消息"),
	)
	t.Set(ACT_SEND_VERIFICATION, i18n.NewTableEntry().
		EN("Send Verification").
		CN("发送验证消息"),
	)

	t.Set(ACT_SYNC_VPCS, i18n.NewTableEntry().
		EN("Sync Vpcs").
		CN("同步VPC"),
	)
	t.Set(ACT_SYNC_RECORD_SETS, i18n.NewTableEntry().
		EN("Sync Record Sets").
		CN("同步解析列表"),
	)

	t.Set(ACT_FREEZE, i18n.NewTableEntry().
		EN("Freeze").
		CN("冻结资源"),
	)
	t.Set(ACT_UNFREEZE, i18n.NewTableEntry().
		EN("Freeze").
		CN("解冻资源"),
	)
	t.Set(ACT_DETACH_ALERTRESOURCE, i18n.NewTableEntry().
		EN("Detach AlertResource").
		CN("取消关联报警资源"),
	)
	t.Set(ACT_NETWORK_ADD_VPC, i18n.NewTableEntry().
		EN("Network Add Vpc").
		CN("网络加入vpc实例"),
	)
	t.Set(ACT_NETWORK_REMOVE_VPC, i18n.NewTableEntry().
		EN("Network Remove Vpc").
		CN("网络移除vpc实例"),
	)
	t.Set(ACT_NETWORK_MODIFY_ROUTE, i18n.NewTableEntry().
		EN("Modify Network Route").
		CN("修改网络路由策略"),
	)
	t.Set(ACT_UPDATE_RULE, i18n.NewTableEntry().
		EN("Update RuleConfig").
		CN("调整规则配置"),
	)
	t.Set(ACT_UPDATE_TAGS, i18n.NewTableEntry().
		EN("Update Tags").
		CN("修改标签"),
	)
}
