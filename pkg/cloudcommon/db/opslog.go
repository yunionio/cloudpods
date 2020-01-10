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

package db

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

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

	ACT_SYNC_UPDATE = "sync_update"
	ACT_SYNC_CREATE = "sync_create"

	ACT_START_CREATE_BACKUP  = "start_create_backup"
	ACT_CREATE_BACKUP        = "create_backup"
	ACT_CREATE_BACKUP_FAILED = "create_backup_failed"
	ACT_DELETE_BACKUP        = "delete_backup"
	ACT_DELETE_BACKUP_FAILED = "delete_backup_failed"

	ACT_UPDATE_STATUS       = "updatestatus"
	ACT_STARTING            = "starting"
	ACT_START               = "start"
	ACT_START_FAIL          = "start_fail"
	ACT_BACKUP_START        = "backup_start"
	ACT_BACKUP_START_FAILED = "backup_start_fail"

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

	ACT_SPLIT = "net_split"
	ACT_MERGE = "net_merge"

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

	ACT_DELOCATING    = "delocating"
	ACT_DELOCATE      = "delocate"
	ACT_DELOCATE_FAIL = "delocate_fail"

	ACT_ISO_PREPARING    = "iso_preparing"
	ACT_ISO_PREPARE_FAIL = "iso_prepare_fail"
	ACT_ISO_ATTACH       = "iso_attach"
	ACT_ISO_DETACH       = "iso_detach"

	ACT_EIP_ATTACH = "eip_attach"
	ACT_EIP_DETACH = "eip_detach"

	ACT_SET_METADATA = "set_meta"
	ACT_DEL_METADATA = "del_meta"

	ACT_VM_DEPLOY      = "deploy"
	ACT_VM_DEPLOY_FAIL = "deploy_fail"

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

	ACT_RESERVE_IP = "reserve_ip"
	ACT_RELEASE_IP = "release_ip"

	ACT_CONVERT_START      = "converting"
	ACT_CONVERT_COMPLETE   = "converted"
	ACT_CONVERT_FAIL       = "convert_fail"
	ACT_UNCONVERT_START    = "unconverting"
	ACT_UNCONVERT_COMPLETE = "unconverted"
	ACT_UNCONVERT_FAIL     = "unconvert_fail"

	ACT_SYNC_HOST_START    = "sync_host_start"
	ACT_SYNCING_HOST       = "syncing_host"
	ACT_SYNC_HOST_COMPLETE = "sync_host_end"
	ACT_SYNC_HOST_FAILED   = "sync_host_fail"

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

	ACT_CHANGE_BANDWIDTH = "eip_change_bandwidth"

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

	ACT_UPLOAD_OBJECT = "upload_obj"
	ACT_DELETE_OBJECT = "delete_obj"
	ACT_MKDIR         = "mkdir"

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
)

type SOpsLogManager struct {
	SModelBaseManager
}

type SOpsLog struct {
	SModelBase

	Id      int64  `primary:"true" auto_increment:"true" list:"user"`                                        // = Column(BigInteger, primary_key=True)
	ObjType string `width:"40" charset:"ascii" nullable:"false" list:"user" create:"required"`               // = Column(VARCHAR(40, charset='ascii'), nullable=False)
	ObjId   string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"` //  = Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=False)
	ObjName string `width:"128" charset:"utf8" nullable:"false" list:"user" create:"required"`               //= Column(VARCHAR(128, charset='utf8'), nullable=False)
	Action  string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"required"`                //= Column(VARCHAR(32, charset='ascii'), nullable=False)
	Notes   string `width:"2048" charset:"utf8" list:"user" create:"required"`                               // = Column(VARCHAR(2048, charset='utf8'))

	ProjectId string `name:"tenant_id" width:"128" charset:"ascii" list:"user" create:"required" index:"true"` // = Column(VARCHAR(ID_LENGTH, charset='ascii'))
	Project   string `name:"tenant" width:"128" charset:"utf8" list:"user" create:"required"`                  // tenant    = Column(VARCHAR(128, charset='utf8'))

	ProjectDomainId string `name:"project_domain_id" default:"default" width:"128" charset:"ascii" list:"user" create:"required"`
	ProjectDomain   string `name:"project_domain" default:"Default" width:"128" charset:"utf8" list:"user" create:"required"`

	UserId   string `width:"128" charset:"ascii" list:"user" create:"required"` // = Column(VARCHAR(ID_LENGTH, charset='ascii'))
	User     string `width:"128" charset:"utf8" list:"user" create:"required"`  // = Column(VARCHAR(128, charset='utf8'))
	DomainId string `width:"128" charset:"ascii" list:"user" create:"optional"`
	Domain   string `width:"128" charset:"utf8" list:"user" create:"optional"`
	Roles    string `width:"64" charset:"ascii" list:"user" create:"optional"` // = Column(VARCHAR(64, charset='ascii'))

	// BillingType    string    `width:"64" charset:"ascii" default:"postpaid" list:"user" create:"user"`      // billing_type = Column(VARCHAR(64, charset='ascii'), nullable=True)
	OpsTime time.Time `nullable:"false" list:"user"` // = Column(DateTime, nullable=False)

	OwnerDomainId  string `name:"owner_domain_id" default:"default" width:"128" charset:"ascii" list:"user" create:"optional"`
	OwnerProjectId string `name:"owner_tenant_id" width:"128" charset:"ascii" list:"user" create:"optional"` // = Column(VARCHAR(ID_LENGTH, charset='ascii'))
	// owner_user_id   = Column(VARCHAR(ID_LENGTH, charset='ascii'))
}

var OpsLog *SOpsLogManager

var _ IModelManager = (*SOpsLogManager)(nil)
var _ IModel = (*SOpsLog)(nil)

var opslogQueryWorkerMan *appsrv.SWorkerManager

func init() {
	OpsLog = &SOpsLogManager{NewModelBaseManager(SOpsLog{}, "opslog_tbl", "event", "events")}
	OpsLog.SetVirtualObject(OpsLog)

	opslogQueryWorkerMan = appsrv.NewWorkerManager("opslog_query_worker", 2, 1024, true)
}

func (manager *SOpsLogManager) CustomizeHandlerInfo(info *appsrv.SHandlerInfo) {
	manager.SModelBaseManager.CustomizeHandlerInfo(info)

	switch info.GetName(nil) {
	case "list":
		info.SetProcessTimeout(time.Minute * 15).SetWorkerManager(opslogQueryWorkerMan)
	}
}

func (opslog *SOpsLog) GetId() string {
	return fmt.Sprintf("%d", opslog.Id)
}

func (opslog *SOpsLog) GetName() string {
	return fmt.Sprintf("%s-%s", opslog.ObjType, opslog.Action)
}

func (opslog *SOpsLog) GetModelManager() IModelManager {
	return OpsLog
}

func (manager *SOpsLogManager) LogEvent(model IModel, action string, notes interface{}, userCred mcclient.TokenCredential) {
	if !consts.OpsLogEnabled() {
		return
	}
	if len(model.GetId()) == 0 || len(model.GetName()) == 0 {
		return
	}
	if action == ACT_UPDATE {
		// skip empty diff
		if notes == nil {
			return
		}
		if uds, ok := notes.(sqlchemy.UpdateDiffs); ok && len(uds) == 0 {
			return
		}
	}
	opslog := SOpsLog{}
	opslog.ObjType = model.Keyword()
	opslog.ObjId = model.GetId()
	opslog.ObjName = model.GetName()
	opslog.Action = action
	opslog.Notes = stringutils.Interface2String(notes)
	opslog.ProjectId = userCred.GetProjectId()
	opslog.Project = userCred.GetProjectName()
	opslog.ProjectDomainId = userCred.GetProjectDomainId()
	opslog.ProjectDomain = userCred.GetProjectDomain()
	opslog.UserId = userCred.GetUserId()
	opslog.User = userCred.GetUserName()
	opslog.DomainId = userCred.GetDomainId()
	opslog.Domain = userCred.GetDomainName()
	opslog.Roles = strings.Join(userCred.GetRoles(), ",")
	opslog.OpsTime = time.Now().UTC()

	if virtualModel, ok := model.(IVirtualModel); ok && virtualModel != nil {
		ownerId := virtualModel.GetOwnerId()
		if ownerId != nil {
			opslog.OwnerProjectId = ownerId.GetProjectId()
			opslog.OwnerDomainId = ownerId.GetProjectDomainId()
		}
	}

	err := manager.TableSpec().Insert(&opslog)
	if err != nil {
		log.Errorf("fail to insert opslog: %s", err)
	}
}

func combineNotes(ctx context.Context, m2 IModel, notes jsonutils.JSONObject) *jsonutils.JSONDict {
	desc := m2.GetShortDesc(ctx)
	if notes != nil {
		if notesDict, ok := notes.(*jsonutils.JSONDict); ok {
			notesMap, _ := notesDict.GetMap()
			if notesMap != nil {
				for k, v := range notesMap {
					desc.Add(v, k)
				}
			}
		} else if notesArray, ok := notes.(*jsonutils.JSONArray); ok {
			noteList, _ := notesArray.GetArray()
			if noteList != nil {
				for i, v := range noteList {
					desc.Add(v, fmt.Sprintf("notes.%d", i))
				}
			}
		} else {
			desc.Add(jsonutils.NewString(notes.String()), "notes")
		}
	}
	return desc
}

func (manager *SOpsLogManager) logOneJointEvent(ctx context.Context, m1, m2 IModel, event string, userCred mcclient.TokenCredential, notes jsonutils.JSONObject) {
	nn := notes
	if m2 != nil {
		nn = combineNotes(ctx, m2, notes)
	}
	manager.LogEvent(m1, event, nn, userCred)
}

func (manager *SOpsLogManager) logJoinEvent(ctx context.Context, m1, m2 IModel, event string, userCred mcclient.TokenCredential, notes jsonutils.JSONObject) {
	if m1 != nil {
		manager.logOneJointEvent(ctx, m1, m2, event, userCred, notes)
	}
	if m2 != nil {
		manager.logOneJointEvent(ctx, m2, m1, event, userCred, notes)
	}
}

func (manager *SOpsLogManager) LogAttachEvent(ctx context.Context, m1, m2 IModel, userCred mcclient.TokenCredential, notes jsonutils.JSONObject) {
	manager.logJoinEvent(ctx, m1, m2, ACT_ATTACH, userCred, notes)
}

func (manager *SOpsLogManager) LogDetachEvent(ctx context.Context, m1, m2 IModel, userCred mcclient.TokenCredential, notes jsonutils.JSONObject) {
	manager.logJoinEvent(ctx, m1, m2, ACT_DETACH, userCred, notes)
}

func (manager *SOpsLogManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	userStrs := jsonutils.GetQueryStringArray(query, "user")
	if len(userStrs) > 0 {
		for i := range userStrs {
			usrObj, err := UserCacheManager.FetchUserByIdOrName(ctx, userStrs[i])
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError2("user", userStrs[i])
				} else if err == sqlchemy.ErrDuplicateEntry {
					return nil, httperrors.NewDuplicateNameError("user", userStrs[i])
				} else {
					return nil, httperrors.NewGeneralError(err)
				}
			}
			userStrs[i] = usrObj.GetId()
		}
		if len(userStrs) == 1 {
			q = q.Filter(sqlchemy.Equals(q.Field("user_id"), userStrs[0]))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("user_id"), userStrs))
		}
	}
	projStrs := jsonutils.GetQueryStringArray(query, "project")
	if len(projStrs) > 0 {
		for i := range projStrs {
			projObj, err := TenantCacheManager.FetchTenantByIdOrName(ctx, projStrs[i])
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError2("project", projStrs[i])
				} else {
					return nil, httperrors.NewGeneralError(err)
				}
			}
			projStrs[i] = projObj.GetId()
		}
		if len(projStrs) == 1 {
			q = q.Filter(sqlchemy.Equals(q.Field("owner_tenant_id"), projStrs[0]))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("owner_tenant_id"), projStrs))
		}
	}
	objTypes := jsonutils.GetQueryStringArray(query, "obj_type")
	if len(objTypes) > 0 {
		if len(objTypes) == 1 {
			q = q.Filter(sqlchemy.Equals(q.Field("obj_type"), objTypes[0]))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("obj_type"), objTypes))
		}
	}
	objs := jsonutils.GetQueryStringArray(query, "obj")
	if len(objs) > 0 {
		if len(objs) == 1 {
			q = q.Filter(sqlchemy.OR(sqlchemy.Equals(q.Field("obj_id"), objs[0]), sqlchemy.Equals(q.Field("obj_name"), objs[0])))
		} else {
			q = q.Filter(sqlchemy.OR(sqlchemy.In(q.Field("obj_id"), objs), sqlchemy.In(q.Field("obj_name"), objs)))
		}
	}
	objIds := jsonutils.GetQueryStringArray(query, "obj_id")
	if len(objIds) > 0 {
		if len(objIds) == 1 {
			q = q.Filter(sqlchemy.Equals(q.Field("obj_id"), objIds[0]))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("obj_id"), objIds))
		}
	}
	objNames := jsonutils.GetQueryStringArray(query, "obj_name")
	if len(objNames) > 0 {
		if len(objNames) == 1 {
			q = q.Filter(sqlchemy.Equals(q.Field("obj_name"), objNames[0]))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("obj_name"), objNames))
		}
	}
	queryDict := query.(*jsonutils.JSONDict)
	queryDict.Remove("obj_id")
	action := jsonutils.GetQueryStringArray(query, "action")
	if action != nil && len(action) > 0 {
		if len(action) == 1 {
			q = q.Filter(sqlchemy.Equals(q.Field("action"), action[0]))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("action"), action))
		}
	}
	//if !IsAdminAllowList(userCred, manager) {
	// 	q = q.Filter(sqlchemy.OR(
	//		sqlchemy.Equals(q.Field("owner_tenant_id"), manager.GetOwnerId(userCred)),
	//		sqlchemy.Equals(q.Field("tenant_id"), manager.GetOwnerId(userCred)),
	//	))
	//}
	since, _ := query.GetTime("since")
	if !since.IsZero() {
		q = q.GT("ops_time", since)
	}
	until, _ := query.GetTime("until")
	if !until.IsZero() {
		q = q.LE("ops_time", until)
	}
	return q, nil
}

func (manager *SOpsLogManager) SyncOwner(m IModel, former *STenant, userCred mcclient.TokenCredential) {
	notes := jsonutils.NewDict()
	notes.Add(jsonutils.NewString(former.GetDomain()), "former_domain_id")
	notes.Add(jsonutils.NewString(former.GetId()), "former_project_id")
	notes.Add(jsonutils.NewString(former.GetName()), "former_project")
	manager.LogEvent(m, ACT_CHANGE_OWNER, notes, userCred)
}

func (manager *SOpsLogManager) LogSyncUpdate(m IModel, uds sqlchemy.UpdateDiffs, userCred mcclient.TokenCredential) {
	if len(uds) > 0 {
		manager.LogEvent(m, ACT_SYNC_UPDATE, uds, userCred)
	}
}

func (manager *SOpsLogManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (manager *SOpsLogManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (self *SOpsLog) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return IsAllowGet(rbacutils.ScopeSystem, userCred, self) || ((userCred.GetProjectDomainId() == self.OwnerDomainId || userCred.GetProjectDomainId() == self.ProjectDomainId) && IsAllowGet(rbacutils.ScopeDomain, userCred, self)) || userCred.GetProjectId() == self.ProjectId || userCred.GetProjectId() == self.OwnerProjectId
}

func (self *SOpsLog) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (self *SOpsLog) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (self *SOpsLog) ValidateDeleteCondition(ctx context.Context) error {
	return httperrors.NewForbiddenError("not allow to delete log")
}

func (self *SOpsLogManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	id, _ := strconv.Atoi(idStr)
	return q.Equals("id", id)
}

func (self *SOpsLogManager) FilterByNotId(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	id, _ := strconv.Atoi(idStr)
	return q.NotEquals("id", id)
}

func (self *SOpsLogManager) FilterByName(q *sqlchemy.SQuery, name string) *sqlchemy.SQuery {
	return q
}

func (self *SOpsLogManager) FilterByOwner(q *sqlchemy.SQuery, ownerId mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if ownerId != nil {
		switch scope {
		case rbacutils.ScopeProject:
			if len(ownerId.GetProjectId()) > 0 {
				q = q.Filter(sqlchemy.OR(
					sqlchemy.Equals(q.Field("owner_tenant_id"), ownerId.GetProjectId()),
					sqlchemy.Equals(q.Field("tenant_id"), ownerId.GetProjectId()),
				))
			}
		case rbacutils.ScopeDomain:
			if len(ownerId.GetProjectDomainId()) > 0 {
				q = q.Filter(sqlchemy.OR(
					sqlchemy.Equals(q.Field("owner_domain_id"), ownerId.GetProjectDomainId()),
					sqlchemy.Equals(q.Field("domain_id"), ownerId.GetProjectDomainId()),
				))
			}
		}
		/* if len(ownerId.GetProjectId()) > 0 {
			q = q.Filter(sqlchemy.OR(
				sqlchemy.Equals(q.Field("owner_tenant_id"), ownerId.GetProjectId()),
				sqlchemy.Equals(q.Field("tenant_id"), ownerId.GetProjectId()),
			))
		} else if len(ownerId.GetProjectDomainId()) > 0 {
			q = q.Filter(sqlchemy.OR(
				sqlchemy.Equals(q.Field("owner_domain_id"), ownerId.GetProjectDomainId()),
				sqlchemy.Equals(q.Field("domain_id"), ownerId.GetProjectDomainId()),
			))
		}
		*/
	}
	return q
}

func (self *SOpsLog) GetOwnerId() mcclient.IIdentityProvider {
	owner := SOwnerId{DomainId: self.OwnerDomainId, ProjectId: self.OwnerProjectId}
	return &owner
}

func (self *SOpsLog) IsSharable(reqCred mcclient.IIdentityProvider) bool {
	return false
}

func (manager *SOpsLogManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeProject
}

func (manager *SOpsLogManager) GetPagingConfig() *SPagingConfig {
	return &SPagingConfig{
		Order:        sqlchemy.SQL_ORDER_DESC,
		MarkerField:  "id",
		DefaultLimit: 20,
	}
}
