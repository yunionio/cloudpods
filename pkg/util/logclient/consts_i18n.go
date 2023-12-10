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
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/i18n"
)

var OpsActionI18nTable = i18n.Table{}
var OpsServiceI18nTable = i18n.Table{}
var OpsObjTypeI18nTable = i18n.Table{}

func init() {
	t := OpsActionI18nTable
	s := OpsServiceI18nTable
	o := OpsObjTypeI18nTable

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
	t.Set(ACT_SET_EXPIRED_TIME, i18n.NewTableEntry().
		EN("Set Resource Expire Time").
		CN("到期释放"),
	)
	t.Set(ACT_VM_SYNC_ISOLATED_DEVICE, i18n.NewTableEntry().
		EN("Guest Sync Isolated Device").
		CN("同步透传设备"),
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
	t.Set(ACT_MIGRATING, i18n.NewTableEntry().
		EN("Migrating").
		CN("迁移中"),
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
	t.Set(ACT_DETACH_NETWORK, i18n.NewTableEntry().
		EN("Detach Network").
		CN("解绑网卡"),
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

	t.Set(ACT_LOGOUT, i18n.NewTableEntry().
		EN("Logout").
		CN("退出登录"),
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

	t.Set(ACT_ADD_VPCS, i18n.NewTableEntry().
		EN("Add Vpcs").
		CN("添加VPC"),
	)
	t.Set(ACT_REMOVE_VPCS, i18n.NewTableEntry().
		EN("Remove Vpcs").
		CN("移除VPC"),
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
	t.Set(ACT_SET_ALERT, i18n.NewTableEntry().
		EN("Set Alert").
		CN("配置报警"),
	)

	s.Set(apis.SERVICE_TYPE_MONITOR, i18n.NewTableEntry().
		EN("Monitor").
		CN("监控"),
	)
	s.Set(apis.SERVICE_TYPE_REGION, i18n.NewTableEntry().
		EN("Compute").
		CN("计算"),
	)
	s.Set(apis.SERVICE_TYPE_IMAGE, i18n.NewTableEntry().
		EN("Image").
		CN("镜像"),
	)
	s.Set(apis.SERVICE_TYPE_CLOUDID, i18n.NewTableEntry().
		EN("Cloud SSO").
		CN("多云统一认证"),
	)
	s.Set(apis.SERVICE_TYPE_DEVTOOL, i18n.NewTableEntry().
		EN("Dev Tools").
		CN("运维工具"),
	)
	s.Set(apis.SERVICE_TYPE_ANSIBLE, i18n.NewTableEntry().
		EN("Ansible").
		CN("Ansible"),
	)
	s.Set(apis.SERVICE_TYPE_KEYSTONE, i18n.NewTableEntry().
		EN("Keystone").
		CN("认证服务"),
	)
	s.Set(apis.SERVICE_TYPE_NOTIFY, i18n.NewTableEntry().
		EN("Notify").
		CN("通知服务"),
	)
	/*s.Set(apis.SERVICE_TYPE_SUGGESTION, i18n.NewTableEntry().
		EN("Suggestion").
		CN("优化建议"),
	)
	*/
	s.Set(apis.SERVICE_TYPE_METER, i18n.NewTableEntry().
		EN("Suggestion").
		CN("计费服务"),
	)
	s.Set("k8s", i18n.NewTableEntry().
		EN("Kubernetes").
		CN("容器服务"),
	)

	o.Set("domain", i18n.NewTableEntry().
		EN("Domain").
		CN("域"),
	)
	o.Set("kubemachine", i18n.NewTableEntry().
		EN("Kube Machine").
		CN("Kube Machine"),
	)
	o.Set("clouduser", i18n.NewTableEntry().
		EN("Cloud user").
		CN("云上用户"),
	)
	o.Set("x509keypair", i18n.NewTableEntry().
		EN("x509 Keypair").
		CN("x509 Keypair"),
	)
	o.Set("kubecluster", i18n.NewTableEntry().
		EN("Kube Cluster").
		CN("Kube Cluster"),
	)
	o.Set("role", i18n.NewTableEntry().
		EN("Role").
		CN("角色"),
	)
	o.Set("notifyconfig", i18n.NewTableEntry().
		EN("Notify Config").
		CN("通知配置"),
	)
	o.Set("wire", i18n.NewTableEntry().
		EN("Wire").
		CN("二层网络"),
	)
	o.Set("loadbalancerlistenerrule", i18n.NewTableEntry().
		EN("Loadbalancer Listener Rule").
		CN("负载均衡监听规则"),
	)
	o.Set("loadbalancerlistener", i18n.NewTableEntry().
		EN("Loadbalancer Listener").
		CN("负载均衡监听器"),
	)
	o.Set("elasticcache", i18n.NewTableEntry().
		EN("Elastic Cache").
		CN("弹性缓存"),
	)
	o.Set("notifytemplate", i18n.NewTableEntry().
		EN("Notify Templete").
		CN("通知模板"),
	)
	o.Set("policy", i18n.NewTableEntry().
		EN("Policy").
		CN("Policy"),
	)
	o.Set("scheduledtask", i18n.NewTableEntry().
		EN("Scheduled Task").
		CN("Scheduled Task"),
	)
	o.Set("saml_provider", i18n.NewTableEntry().
		EN("SAML Provider").
		CN("SAML 身份提供商"),
	)
	o.Set("daemonset", i18n.NewTableEntry().
		EN("Daemonset").
		CN("Daemonset"),
	)
	o.Set("network", i18n.NewTableEntry().
		EN("Network").
		CN("IP子网"),
	)
	o.Set("vpc", i18n.NewTableEntry().
		EN("VPC").
		CN("VPC"),
	)
	o.Set("dbinstancebackup", i18n.NewTableEntry().
		EN("RDS Backup").
		CN("关系型数据库备份"),
	)
	o.Set("host", i18n.NewTableEntry().
		EN("Host").
		CN("宿主机"),
	)
	o.Set("identity_provider", i18n.NewTableEntry().
		EN("Identity Provider").
		CN("身份提供商"),
	)
	o.Set("commonalert", i18n.NewTableEntry().
		EN("Common Alert").
		CN("Common Alert"),
	)
	o.Set("loadbalanceragent", i18n.NewTableEntry().
		EN("Loadbalancer Agent").
		CN("负载均衡Agent"),
	)
	o.Set("loadbalancer", i18n.NewTableEntry().
		EN("Loadbalancer").
		CN("负载均衡"),
	)
	o.Set("cloudgroup", i18n.NewTableEntry().
		EN("Cloud Group").
		CN("权限组"),
	)
	o.Set("cloudgroupcache", i18n.NewTableEntry().
		EN("Cloud group cache").
		CN("权限组缓存"),
	)
	o.Set("samluser", i18n.NewTableEntry().
		EN("SAML User").
		CN("免密用户"),
	)
	o.Set("project", i18n.NewTableEntry().
		EN("Project").
		CN("项目"),
	)
	o.Set("keypair", i18n.NewTableEntry().
		EN("Keypair").
		CN("秘钥对"),
	)
	o.Set("loadbalancerbackendgroup", i18n.NewTableEntry().
		EN("Loadbalancer Backendgroup").
		CN("后端服务器组"),
	)
	o.Set("statefulset", i18n.NewTableEntry().
		EN("State Fulset").
		CN("State Fulset"),
	)
	o.Set("bucket", i18n.NewTableEntry().
		EN("Bucket").
		CN("存储桶"),
	)
	o.Set("receiver", i18n.NewTableEntry().
		EN("Receiver").
		CN("接收者"),
	)
	o.Set("suggestsysrule", i18n.NewTableEntry().
		EN("Suggest sysrule").
		CN("建议规则"),
	)
	o.Set("dbinstance", i18n.NewTableEntry().
		EN("RDS").
		CN("关系型数据库"),
	)
	o.Set("storagecachedimage", i18n.NewTableEntry().
		EN("Storage Cached Image").
		CN("存储镜像缓存"),
	)
	o.Set("image", i18n.NewTableEntry().
		EN("Image").
		CN("镜像"),
	)
	o.Set("itsm", i18n.NewTableEntry().
		EN("ITSM").
		CN("ITSM"),
	)
	o.Set("disk", i18n.NewTableEntry().
		EN("Disk").
		CN("磁盘"),
	)
	o.Set("eip", i18n.NewTableEntry().
		EN("Elastic Ip").
		CN("弹性公网IP"),
	)
	o.Set("alert", i18n.NewTableEntry().
		EN("Alert").
		CN("报警"),
	)
	o.Set("budget", i18n.NewTableEntry().
		EN("Budget").
		CN("预算"),
	)
	o.Set("costalert", i18n.NewTableEntry().
		EN("Cost Alert").
		CN("Cost Alert"),
	)
	o.Set("alert_notification", i18n.NewTableEntry().
		EN("Alert Notification").
		CN("Alert Notification"),
	)
	o.Set("servertemplate", i18n.NewTableEntry().
		EN("Instance Templete").
		CN("主机模板"),
	)
	o.Set("cachedimage", i18n.NewTableEntry().
		EN("Cached Image").
		CN("镜像缓存"),
	)
	o.Set("suggestsysalert", i18n.NewTableEntry().
		EN("Suggest Sys Alert").
		CN("建议预警"),
	)
	o.Set("cloudaccount", i18n.NewTableEntry().
		EN("Cloud Account").
		CN("云账号"),
	)
	o.Set("cloudprovider", i18n.NewTableEntry().
		EN("Subscription").
		CN("订阅"),
	)
	o.Set("snapshot", i18n.NewTableEntry().
		EN("Snapshot").
		CN("快照"),
	)
	o.Set("costreport", i18n.NewTableEntry().
		EN("Cost Report").
		CN("消费报告"),
	)
	o.Set("deployment", i18n.NewTableEntry().
		EN("Deployment").
		CN("Deployment"),
	)
	o.Set("storage", i18n.NewTableEntry().
		EN("Storage").
		CN("块存储"),
	)
	o.Set("server", i18n.NewTableEntry().
		EN("Server").
		CN("虚拟机"),
	)
	o.Set("notification", i18n.NewTableEntry().
		EN("Notification").
		CN("通知"),
	)
	o.Set("secgroup", i18n.NewTableEntry().
		EN("Security Group").
		CN("安全组"),
	)
	o.Set("endpoint", i18n.NewTableEntry().
		EN("Endpoint").
		CN("端点"),
	)
	o.Set("service", i18n.NewTableEntry().
		EN("Service").
		CN("服务"),
	)
	o.Set("user", i18n.NewTableEntry().
		EN("User").
		CN("用户"),
	)
	o.Set("instance_snapshot", i18n.NewTableEntry().
		EN("Instance Snapshot").
		CN("主机快照"),
	)
	o.Set("guestimage", i18n.NewTableEntry().
		EN("Instance Image").
		CN("主机镜像"),
	)
	o.Set("instancegroup", i18n.NewTableEntry().
		EN("Instance Group").
		CN("反亲和组"),
	)
	o.Set("dbinstanceaccount", i18n.NewTableEntry().
		EN("RDS Account").
		CN("关系型数据库账号"),
	)
	o.Set("elasticcacheaccount", i18n.NewTableEntry().
		EN("Elastic Cache Account").
		CN("弹性缓存账号"),
	)
	o.Set("dbinstancedatabase", i18n.NewTableEntry().
		EN("RDS Database").
		CN("数据库"),
	)
	o.Set("devtool_template", i18n.NewTableEntry().
		EN("DevTool Template").
		CN("DevTool Templete"),
	)
	o.Set("devtool_cronjob", i18n.NewTableEntry().
		EN("DevTool Cronjob").
		CN("DevTool Cronjob"),
	)
	o.Set("elasticcachebackup", i18n.NewTableEntry().
		EN("Elastic Cache Backup").
		CN("弹性缓存备份"),
	)
	o.Set("kubecomponent", i18n.NewTableEntry().
		EN("Kube Component").
		CN("Kube Component"),
	)
	o.Set("scalinggroup", i18n.NewTableEntry().
		EN("Scaling Group").
		CN("弹性伸缩组"),
	)
	o.Set("scalingpolicy", i18n.NewTableEntry().
		EN("Scaling Policy").
		CN("弹性伸缩策略"),
	)
	o.Set("proxysetting", i18n.NewTableEntry().
		EN("Proxy Setting").
		CN("代理"),
	)
	o.Set("credential", i18n.NewTableEntry().
		EN("Credential").
		CN("Credential"),
	)
	o.Set("cloudproviderquota", i18n.NewTableEntry().
		EN("Subscription Quota").
		CN("订阅配额"),
	)
	o.Set("nodealert", i18n.NewTableEntry().
		EN("Node Alert").
		CN("Node Alert"),
	)
	o.Set("globalvpc", i18n.NewTableEntry().
		EN("Gloal VPC").
		CN("全局VPC"),
	)
	o.Set("contact", i18n.NewTableEntry().
		EN("Contact").
		CN("Contact"),
	)
	o.Set("schedpolicy", i18n.NewTableEntry().
		EN("Scheduler Policy").
		CN("调度策略"),
	)
	o.Set("config", i18n.NewTableEntry().
		EN("Config").
		CN("配置"),
	)
	o.Set("namespace", i18n.NewTableEntry().
		EN("Namespace").
		CN("Namespace"),
	)
	o.Set("repo", i18n.NewTableEntry().
		EN("Repo").
		CN("Repo"),
	)
	o.Set("release", i18n.NewTableEntry().
		EN("Release").
		CN("Release"),
	)
	o.Set("servicecertificate", i18n.NewTableEntry().
		EN("Service Certificate").
		CN("Service Certificate"),
	)
	o.Set("pod", i18n.NewTableEntry().
		EN("Pod").
		CN("Pod"),
	)
	o.Set("elasticcacheacl", i18n.NewTableEntry().
		EN("Elastic Cache Acl").
		CN("弹性缓存ACL"),
	)
	o.Set("metricmeasurement", i18n.NewTableEntry().
		EN("Metric Measurement").
		CN("Metric Measurement"),
	)
	o.Set("alertdashboard", i18n.NewTableEntry().
		EN("Alert Dashboard").
		CN("Alert Dashboard"),
	)
	o.Set("dns_zone", i18n.NewTableEntry().
		EN("DNS Zone").
		CN("DNS Zone"),
	)
	o.Set("dns_recordset", i18n.NewTableEntry().
		EN("DNS Records").
		CN("DNS Records"),
	)
	o.Set("dns_zonecache", i18n.NewTableEntry().
		EN("DNS Zone Cache").
		CN("DNS Zone Cache"),
	)
	o.Set("federatednamespace", i18n.NewTableEntry().
		EN("Federated Namespace").
		CN("Federated Namespace"),
	)
	o.Set("ingress", i18n.NewTableEntry().
		EN("Ingress").
		CN("Ingress"),
	)
	o.Set("cronjob", i18n.NewTableEntry().
		EN("Cronjob").
		CN("定时任务"),
	)
	o.Set("federatedrole", i18n.NewTableEntry().
		EN("Federated Role").
		CN("Federated Role"),
	)
	o.Set("federatedclusterrole", i18n.NewTableEntry().
		EN("Federated Cluster Role").
		CN("Federated Cluster Role"),
	)
	o.Set("rbacclusterrole", i18n.NewTableEntry().
		EN("RBAC Cluster Role").
		CN("RBAC Cluster Role"),
	)
	o.Set("job", i18n.NewTableEntry().
		EN("Job").
		CN("Job"),
	)
	o.Set("rbacrole", i18n.NewTableEntry().
		EN("RBAC Role").
		CN("RBAC Role"),
	)
	o.Set("federatedrolebinding", i18n.NewTableEntry().
		EN("Federated Role Binding").
		CN("Federated Role Binding"),
	)
	o.Set("scopedpolicy", i18n.NewTableEntry().
		EN("Scoped Policy").
		CN("Scoped Policy"),
	)
	o.Set("alertpanel", i18n.NewTableEntry().
		EN("Alert Panel").
		CN("监控面板"),
	)
	o.Set("networkaddress", i18n.NewTableEntry().
		EN("Network Address").
		CN("网卡地址"),
	)
	o.Set("networkaddress", i18n.NewTableEntry().
		EN("Network Address").
		CN("网卡地址"),
	)
	o.Set("baremetalagent", i18n.NewTableEntry().
		EN("Baremetal Agent").
		CN("Baremetal Agent"),
	)
	o.Set("notice", i18n.NewTableEntry().
		EN("Notice").
		CN("Notice"),
	)
	o.Set("hostwire", i18n.NewTableEntry().
		EN("Host Wire").
		CN("宿主机网络"),
	)
	o.Set("storagecache", i18n.NewTableEntry().
		EN("Storage Cache").
		CN("存储缓存"),
	)
	o.Set("isolated_device", i18n.NewTableEntry().
		EN("Isolated Device").
		CN("透传设备"),
	)
	o.Set("guestdisk", i18n.NewTableEntry().
		EN("Server Disk").
		CN("虚拟机磁盘"),
	)
	o.Set("guestnetwork", i18n.NewTableEntry().
		EN("Server Network").
		CN("虚拟机网络"),
	)
	o.Set("kube_node", i18n.NewTableEntry().
		EN("Kube Node").
		CN("Kube Node"),
	)
	o.Set("secgrouprule", i18n.NewTableEntry().
		EN("Security Group Rule").
		CN("安全组规则"),
	)
	o.Set("kube_node", i18n.NewTableEntry().
		EN("Kube Node").
		CN("Kube Node"),
	)
	o.Set("kube_cluster", i18n.NewTableEntry().
		EN("Kube Cluster").
		CN("Kube Cluster"),
	)
	o.Set("hoststorage", i18n.NewTableEntry().
		EN("Host Storage").
		CN("宿主机存储"),
	)
	o.Set("servicetree", i18n.NewTableEntry().
		EN("Service Tree").
		CN("服务目录"),
	)
	o.Set("baremetalnetwork", i18n.NewTableEntry().
		EN("Baremetal Network").
		CN("裸金属服务器网络"),
	)
	o.Set("guestsecgroup", i18n.NewTableEntry().
		EN("Server Security Group").
		CN("虚拟机安全组"),
	)
	o.Set("schedtaghost", i18n.NewTableEntry().
		EN("Schedtag Host").
		CN("Schedtag Host"),
	)
	o.Set("loadbalancernetwork", i18n.NewTableEntry().
		EN("Loadbalancer Network").
		CN("负载均衡网络"),
	)
	o.Set("infos", i18n.NewTableEntry().
		EN("Infos").
		CN("通知"),
	)
	o.Set("finance-alert", i18n.NewTableEntry().
		EN("Finance Alert").
		CN(""),
	)
	o.Set("group", i18n.NewTableEntry().
		EN("Group").
		CN("Group"),
	)
	o.Set("readmark", i18n.NewTableEntry().
		EN("Readmark").
		CN("Readmark"),
	)
	o.Set("meteralert", i18n.NewTableEntry().
		EN("Meter Alert").
		CN("Meter Alert"),
	)
	o.Set("reservedip", i18n.NewTableEntry().
		EN("Reserve IP").
		CN("预留IP"),
	)
	o.Set("cloudregion", i18n.NewTableEntry().
		EN("Cloudregion").
		CN("区域"),
	)
	o.Set("meter_vm", i18n.NewTableEntry().
		EN("Meter VM").
		CN("Meter VM"),
	)
	o.Set("route_table", i18n.NewTableEntry().
		EN("Route Table").
		CN("路由表"),
	)
	o.Set("externalproject", i18n.NewTableEntry().
		EN("Cloud Project").
		CN("云上项目"),
	)
	o.Set("cloudproviderregion", i18n.NewTableEntry().
		EN("Cloudprovider Region").
		CN("订阅区域"),
	)
	o.Set("metadata", i18n.NewTableEntry().
		EN("Tag").
		CN("标签"),
	)
	o.Set("ansibleplaybook", i18n.NewTableEntry().
		EN("Ansible Playbook").
		CN("Ansible Playbook"),
	)
	o.Set("natdentry", i18n.NewTableEntry().
		EN("NAT D Entry").
		CN("NAT D Entry"),
	)
	o.Set("natsentry", i18n.NewTableEntry().
		EN("NAT S Entry").
		CN("NAT S Entry"),
	)
	o.Set("snapshotpolicy", i18n.NewTableEntry().
		EN("Snapshot Policy").
		CN("快照策略"),
	)
	o.Set("meshnetwork_member", i18n.NewTableEntry().
		EN("Meshnetwork Member").
		CN("Meshnetwork Member"),
	)
	o.Set("ifacepeer", i18n.NewTableEntry().
		EN("Iface Peer").
		CN("Iface Peer"),
	)
	o.Set("iface", i18n.NewTableEntry().
		EN("Iface").
		CN("Iface"),
	)
	o.Set("router", i18n.NewTableEntry().
		EN("Router").
		CN("路由"),
	)
	o.Set("zone", i18n.NewTableEntry().
		EN("Zone").
		CN("可用区"),
	)
	o.Set("schedtag", i18n.NewTableEntry().
		EN("Sched Tag").
		CN("调度标签"),
	)
	o.Set("dynamicschedtag", i18n.NewTableEntry().
		EN("Dynamic Sched Tag").
		CN("动态调度标签"),
	)
	o.Set("serversku", i18n.NewTableEntry().
		EN("Server Sku").
		CN("虚拟机套餐"),
	)

	o.Set(ACT_UPDATE_MONITOR_RESOURCE_JOINT, i18n.NewTableEntry().
		EN("Update Monitor Resource joint").
		CN("更新监控资源关联报警策略"),
	)

	o.Set(ACT_DETACH_MONITOR_RESOURCE_JOINT, i18n.NewTableEntry().
		EN("Update Monitor Resource joint").
		CN("解绑监控资源关联报警策略"),
	)

	o.Set(ACT_RECOVERY, i18n.NewTableEntry().
		EN("Recover backup").
		CN("备份恢复"),
	)

	o.Set(ACT_PACK, i18n.NewTableEntry().
		EN("Package VM").
		CN("导出主机"),
	)

	o.Set(ACT_UNPACK, i18n.NewTableEntry().
		EN("Unpackage VM").
		CN("导入主机"),
	)

	o.Set(ACT_ENCRYPTION, i18n.NewTableEntry().
		EN("Encryption").
		CN("加密"),
	)

	o.Set(ACT_CONSOLE, i18n.NewTableEntry().
		EN("Console").
		CN("控制台"),
	)

	o.Set(ACT_WEBSSH, i18n.NewTableEntry().
		EN("WebSSH").
		CN("WebSSH"),
	)

	o.Set(ACT_CLOUDACCOUNT_SYNC_NETWORK, i18n.NewTableEntry().
		EN("Probe Network").
		CN("探测网络配置"),
	)

	o.Set(ACT_EXPORT, i18n.NewTableEntry().
		EN("Export").
		CN("导出"),
	)

	o.Set(ACT_CANCEL, i18n.NewTableEntry().
		EN("Cancel").
		CN("取消"),
	)

	o.Set(ACT_START, i18n.NewTableEntry().
		EN("Start").
		CN("开始"),
	)

	o.Set(ACT_DONE, i18n.NewTableEntry().
		EN("Done").
		CN("完成"),
	)

	o.Set(ACT_ASSOCIATE, i18n.NewTableEntry().
		EN("Associate").
		CN("关联"),
	)

	o.Set(ACT_DISSOCIATE, i18n.NewTableEntry().
		EN("Dissociate").
		CN("解除关联"),
	)

	o.Set(ACT_BIND, i18n.NewTableEntry().
		EN("Bind").
		CN("关联"),
	)

	o.Set(ACT_PROGRESS, i18n.NewTableEntry().
		EN("Progress").
		CN("进展"),
	)

	o.Set(ACT_ADD_BASTION_SERVER, i18n.NewTableEntry().
		EN("Add Bastionhost Server").
		CN("添加实例到堡垒机"),
	)

	o.Set(ACT_SET_USER_PASSWORD, i18n.NewTableEntry().
		EN("Set Password For User").
		CN("设置用户密码"),
	)

	o.Set(ACT_DISK_CHANGE_STORAGE, i18n.NewTableEntry().
		EN("Disk Change Storage").
		CN("磁盘更换存储"),
	)

	o.Set(ACT_SYNC_TRAFFIC_LIMIT, i18n.NewTableEntry().
		EN("Sync Nic Traffic Limit").
		CN("同步网卡流量限制"),
	)

	o.Set(ACT_GENERATE_REPORT, i18n.NewTableEntry().
		EN("Generate Report").
		CN("生成报表"),
	)

	o.Set(ACT_REPORT_COLLECT_DATA, i18n.NewTableEntry().
		EN("Collect Report Data").
		CN("采集报表数据"),
	)

	o.Set(ACT_REPORT_SEND, i18n.NewTableEntry().
		EN("Send Report").
		CN("发送报表"),
	)

	o.Set(ACT_REPORT_TEMPLATE, i18n.NewTableEntry().
		EN("Report Template").
		CN("报表模板"),
	)

	o.Set(ACT_SAVE_IMAGE, i18n.NewTableEntry().
		EN("Save Image").
		CN("保存镜像"),
	)

	o.Set(ACT_CLOUD_SYNC, i18n.NewTableEntry().
		EN("Sync Cloud Resource").
		CN("同步云资源"),
	)

	o.Set(ACT_COLLECT_METRICS, i18n.NewTableEntry().
		EN("Collect monitoring metrics").
		CN("采集监控指标"),
	)
}
