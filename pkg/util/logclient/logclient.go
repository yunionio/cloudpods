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
	"context"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

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
	ACT_ATTACH_HOST                  = "关联宿主机"
	ACT_DETACH_HOST                  = "取消关联宿主机"

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
	ACT_HOST_MIGRATE                = "宿主机迁移"
)

type SessionGenerator func(ctx context.Context, token mcclient.TokenCredential, region, apiVersion string) *mcclient.ClientSession

var (
	DefaultSessionGenerator = auth.GetSession
)

// golang 不支持 const 的string array, http://t.cn/EzAvbw8
var BLACK_LIST_OBJ_TYPE = []string{"parameter"}

var logclientWorkerMan *appsrv.SWorkerManager

func init() {
	logclientWorkerMan = appsrv.NewWorkerManager("LogClientWorkerManager", 1, 50, false)
}

type IObject interface {
	GetId() string
	GetName() string
	Keyword() string
}

type IVirtualObject interface {
	IObject
	GetOwnerId() mcclient.IIdentityProvider
}

type IModule interface {
	Create(session *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
}

// save log to db.
func AddSimpleActionLog(model IObject, action string, iNotes interface{}, userCred mcclient.TokenCredential, success bool) {
	addLog(model, action, iNotes, userCred, success, time.Time{}, &modules.Actions)
}

func AddActionLogWithContext(ctx context.Context, model IObject, action string, iNotes interface{}, userCred mcclient.TokenCredential, success bool) {
	addLog(model, action, iNotes, userCred, success, appctx.AppContextStartTime(ctx), &modules.Actions)
}

func AddActionLogWithStartable(task cloudcommon.IStartable, model IObject, action string, iNotes interface{}, userCred mcclient.TokenCredential, success bool) {
	addLog(model, action, iNotes, userCred, success, task.GetStartTime(), &modules.Actions)
}

// add websocket log to notify active browser users
func PostWebsocketNotify(model IObject, action string, iNotes interface{}, userCred mcclient.TokenCredential, success bool) {
	addLog(model, action, iNotes, userCred, success, time.Time{}, &modules.Websockets)
}

func addLog(model IObject, action string, iNotes interface{}, userCred mcclient.TokenCredential, success bool, startTime time.Time, api IModule) {
	if !consts.OpsLogEnabled() {
		return
	}
	if ok, _ := utils.InStringArray(model.Keyword(), BLACK_LIST_OBJ_TYPE); ok {
		log.Errorf("不支持的 actionlog 类型")
		return
	}
	if action == ACT_UPDATE {
		if iNotes == nil {
			return
		}
		if uds, ok := iNotes.(sqlchemy.UpdateDiffs); ok && len(uds) == 0 {
			return
		}
	}

	notes := stringutils.Interface2String(iNotes)

	objId := model.GetId()
	if len(objId) == 0 {
		objId = "-"
	}
	objName := model.GetName()
	if len(objName) == 0 {
		objName = "-"
	}

	logentry := jsonutils.NewDict()

	logentry.Add(jsonutils.NewString(objName), "obj_name")
	logentry.Add(jsonutils.NewString(model.Keyword()), "obj_type")
	logentry.Add(jsonutils.NewString(objId), "obj_id")
	logentry.Add(jsonutils.NewString(action), "action")
	logentry.Add(jsonutils.NewString(userCred.GetUserId()), "user_id")
	logentry.Add(jsonutils.NewString(userCred.GetUserName()), "user")
	logentry.Add(jsonutils.NewString(userCred.GetTenantId()), "tenant_id")
	logentry.Add(jsonutils.NewString(userCred.GetTenantName()), "tenant")
	logentry.Add(jsonutils.NewString(userCred.GetDomainId()), "domain_id")
	logentry.Add(jsonutils.NewString(userCred.GetDomainName()), "domain")
	logentry.Add(jsonutils.NewString(userCred.GetProjectDomainId()), "project_domain_id")
	logentry.Add(jsonutils.NewString(userCred.GetProjectDomain()), "project_domain")
	logentry.Add(jsonutils.NewString(strings.Join(userCred.GetRoles(), ",")), "roles")

	service := consts.GetServiceType()
	if len(service) > 0 {
		logentry.Add(jsonutils.NewString(service), "service")
	}

	if !startTime.IsZero() {
		logentry.Add(jsonutils.NewString(timeutils.FullIsoTime(startTime)), "start_time")
	}

	if virtualModel, ok := model.(IVirtualObject); ok {
		ownerId := virtualModel.GetOwnerId()
		if ownerId != nil {
			projectId := ownerId.GetProjectId()
			if len(projectId) > 0 {
				logentry.Add(jsonutils.NewString(projectId), "owner_tenant_id")
			}
			domainId := ownerId.GetProjectDomainId()
			if len(domainId) > 0 {
				logentry.Add(jsonutils.NewString(domainId), "owner_domain_id")
			}
		}
	}

	if !success {
		// 失败日志
		logentry.Add(jsonutils.JSONFalse, "success")
	} else {
		// 成功日志
		logentry.Add(jsonutils.JSONTrue, "success")
	}

	logentry.Add(jsonutils.NewString(notes), "notes")
	logclientWorkerMan.Run(func() {
		s := DefaultSessionGenerator(context.Background(), userCred, "", "")
		_, err := api.Create(s, logentry)
		if err != nil {
			log.Errorf("create action log failed %s", err)
		}
	}, nil, nil)
}
