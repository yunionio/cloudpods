package logclient

import (
	"context"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/util/timeutils"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

const (
	ACT_ADDTAG                       = "添加标签"
	ACT_ALLOCATE                     = "分配"
	ACT_BM_CONVERT_HYPER             = "转换为宿主机"
	ACT_BM_MAINTENANCE               = "进入离线状态"
	ACT_BM_UNCONVERT_HYPER           = "转换为受管物理机"
	ACT_BM_UNMAINTENANCE             = "退出离线状态"
	ACT_CANCEL_DELETE                = "恢复"
	ACT_CHANGE_OWNER                 = "更改项目"
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

	ACT_IMAGE_SAVE = "上传镜像"

	ACT_RECYCLE_PREPAID      = "池化预付费主机"
	ACT_UNDO_RECYCLE_PREPAID = "取消池化预付费主机"

	ACT_FETCH = "下载密钥"
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
	GetOwnerProjectId() string
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

	token := userCred
	notes := stringutils.Interface2String(iNotes)

	// 忽略不黑名单里的资源类型
	for _, v := range BLACK_LIST_OBJ_TYPE {
		if v == model.Keyword() {
			log.Errorf("不支持的 actionlog 类型")
			return
		}
	}

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
	logentry.Add(jsonutils.NewString(token.GetUserId()), "user_id")
	logentry.Add(jsonutils.NewString(token.GetUserName()), "user")
	logentry.Add(jsonutils.NewString(token.GetTenantId()), "tenant_id")
	logentry.Add(jsonutils.NewString(token.GetTenantName()), "tenant")
	logentry.Add(jsonutils.NewString(token.GetDomainId()), "domain_id")
	logentry.Add(jsonutils.NewString(token.GetDomainName()), "domain")
	logentry.Add(jsonutils.NewString(strings.Join(token.GetRoles(), ",")), "roles")

	service := consts.GetServiceType()
	if len(service) > 0 {
		logentry.Add(jsonutils.NewString(service), "service")
	}

	if !startTime.IsZero() {
		logentry.Add(jsonutils.NewString(timeutils.FullIsoTime(startTime)), "start_time")
	}

	if virtualModel, ok := model.(IVirtualObject); ok {
		ownerProjId := virtualModel.GetOwnerProjectId()
		if len(ownerProjId) > 0 {
			logentry.Add(jsonutils.NewString(ownerProjId), "owner_tenant_id")
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
		s := auth.GetSession(context.Background(), userCred, "", "")
		_, err := api.Create(s, logentry)
		if err != nil {
			log.Errorf("create action log failed %s", err)
		}
	}, nil, nil)
}
