package db

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	ACT_CREATE  = "create"
	ACT_DELETE  = "delete"
	ACT_UPDATE  = "update"
	ACT_FETCH   = "fetch"
	ACT_ENABLE  = "enable"
	ACT_DISABLE = "disable"
	ACT_OFFLINE = "offline"
	ACT_ONLINE  = "online"
	ACT_ATTACH  = "attach"
	ACT_DETACH  = "detach"

	ACT_SYNC_UPDATE = "sync_update"
	ACT_SYNC_CREATE = "sync_create"

	ACT_START_CREATE_BACKUP  = "start_create_backup"
	ACT_CREATE_BACKUP        = "create_backup"
	ACT_CREATE_BACKUP_FAILED = "create_backup_failed"

	ACT_UPDATE_STATUS       = "updatestatus"
	ACT_STARTING            = "starting"
	ACT_START               = "start"
	ACT_START_FAIL          = "start_fail"
	ACT_BACKUP_START        = "backup_start"
	ACT_BACKUP_START_FAILED = "backup_start_fail"

	ACT_STOPPING  = "stopping"
	ACT_STOP      = "stop"
	ACT_STOP_FAIL = "stop_fail"

	ACT_RESIZING    = "resizing"
	ACT_RESIZE      = "resize"
	ACT_RESIZE_FAIL = "resize_fail"

	ACT_MIGRATING    = "migrating"
	ACT_MIGRATE      = "migrate"
	ACT_MIGRATE_FAIL = "migrate_fail"

	ACT_SPLIT = "net_split"
	ACT_MERGE = "net_merge"

	ACT_SAVING    = "saving"
	ACT_SAVE      = "save"
	ACT_SAVE_FAIL = "save_fail"

	ACT_SWITCHED      = "switched"
	ACT_SWITCH_FAILED = "switch_failed"

	ACT_SNAPSHOTING          = "snapshoting"
	ACT_SNAPSHOT_STREAM      = "snapshot_stream"
	ACT_SNAPSHOT_DONE        = "snapshot"
	ACT_SNAPSHOT_READY       = "snapshot_ready"
	ACT_SNAPSHOT_SYNC        = "snapshot_sync"
	ACT_SNAPSHOT_FAIL        = "snapshot_fail"
	ACT_SNAPSHOT_DELETING    = "snapshot_deling"
	ACT_SNAPSHOT_DELETE      = "snapshot_del"
	ACT_SNAPSHOT_DELETE_FAIL = "snapshot_del_fail"
	ACT_SNAPSHOT_UNLINK      = "snapshot_unlink"

	ACT_DISK_CLEAN_UP_SNAPSHOTS      = "disk_clean_up_snapshots"
	ACT_DISK_CLEAN_UP_SNAPSHOTS_FAIL = "disk_clean_up_snapshots_fail"

	ACT_ALLOCATING           = "allocating"
	ACT_BACKUP_ALLOCATING    = "backup_allocating"
	ACT_ALLOCATE             = "allocate"
	ACT_BACKUP_ALLOCATE      = "backup_allocate"
	ACT_ALLOCATE_FAIL        = "alloc_fail"
	ACT_BACKUP_ALLOCATE_FAIL = "backup_alloc_fail"

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

	ACT_SYNC_LB_START    = "sync_lb_start"
	ACT_SYNCING_LB       = "syncing_lb"
	ACT_SYNC_LB_COMPLETE = "sync_lb_end"

	ACT_CACHING_IMAGE      = "caching_image"
	ACT_CACHE_IMAGE_FAIL   = "cache_image_fail"
	ACT_CACHED_IMAGE       = "cached_image"
	ACT_UNCACHING_IMAGE    = "uncaching_image"
	ACT_UNCACHE_IMAGE_FAIL = "uncache_image_fail"
	ACT_UNCACHED_IMAGE     = "uncached_image"

	ACT_SYNC_CLOUD_DISK   = "sync_cloud_disk"
	ACT_SYNC_CLOUD_SERVER = "sync_cloud_server"
	ACT_SYNC_CLOUD_EIP    = "sync_cloud_eip"

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

	ACT_CHANGE_BANDWIDTH = "eip_change_bandwidth"

	ACT_RENEW = "renew"

	ACT_SCHEDULE = "schedule"

	ACT_RECYCLE_PREPAID      = "recycle_prepaid"
	ACT_UNDO_RECYCLE_PREPAID = "undo_recycle_prepaid"
)

type SOpsLogManager struct {
	SModelBaseManager
}

type SOpsLog struct {
	SModelBase

	Id        int64  `primary:"true" auto_increment:"true" list:"user"`                           // = Column(BigInteger, primary_key=True)
	ObjType   string `width:"40" charset:"ascii" nullable:"false" list:"user" create:"required"`  // = Column(VARCHAR(40, charset='ascii'), nullable=False)
	ObjId     string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"` //  = Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=False)
	ObjName   string `width:"128" charset:"utf8" nullable:"false" list:"user" create:"required"`  //= Column(VARCHAR(128, charset='utf8'), nullable=False)
	Action    string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"required"`   //= Column(VARCHAR(32, charset='ascii'), nullable=False)
	Notes     string `width:"2048" charset:"utf8" list:"user" create:"required"`                  // = Column(VARCHAR(2048, charset='utf8'))
	ProjectId string `name:"tenant_id" width:"128" charset:"ascii" list:"user" create:"required"` // = Column(VARCHAR(ID_LENGTH, charset='ascii'))
	Project   string `name:"tenant" width:"128" charset:"utf8" list:"user" create:"required"`     // tenant    = Column(VARCHAR(128, charset='utf8'))
	UserId    string `width:"128" charset:"ascii" list:"user" create:"required"`                  // = Column(VARCHAR(ID_LENGTH, charset='ascii'))
	User      string `width:"128" charset:"utf8" list:"user" create:"required"`                   // = Column(VARCHAR(128, charset='utf8'))
	DomainId  string `width:"128" charset:"ascii" list:"user" create:"optional"`
	Domain    string `width:"128" charset:"utf8" list:"user" create:"optional"`
	Roles     string `width:"64" charset:"ascii" list:"user" create:"optional"` // = Column(VARCHAR(64, charset='ascii'))

	// BillingType    string    `width:"64" charset:"ascii" default:"postpaid" list:"user" create:"user"`      // billing_type = Column(VARCHAR(64, charset='ascii'), nullable=True)
	OpsTime        time.Time `nullable:"false" list:"user"`                                                     // = Column(DateTime, nullable=False)
	OwnerProjectId string    `name:"owner_tenant_id" width:"128" charset:"ascii" list:"user" create:"optional"` // = Column(VARCHAR(ID_LENGTH, charset='ascii'))
	// owner_user_id   = Column(VARCHAR(ID_LENGTH, charset='ascii'))
}

var OpsLog *SOpsLogManager

var _ IModelManager = (*SOpsLogManager)(nil)
var _ IModel = (*SOpsLog)(nil)

func init() {
	OpsLog = &SOpsLogManager{NewModelBaseManager(SOpsLog{}, "opslog_tbl", "event", "events")}
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

/* @classmethod
   def list_fields(cls, user_cred):
       return ['id', 'obj_type', 'obj_id', 'obj_name', 'action', 'notes',
                   'tenant_id', 'tenant', 'user_id', 'user', 'roles',
                   'billing_type', 'ops_time', 'owner_tenant_id',
                   'owner_user_id', ]
*/

func (manager *SOpsLogManager) LogEvent(model IModel, action string, notes interface{}, userCred mcclient.TokenCredential) {
	if !consts.OpsLogEnabled() {
		return
	}
	if len(model.GetId()) == 0 || len(model.GetName()) == 0 {
		return
	}
	opslog := SOpsLog{}
	opslog.ObjType = model.Keyword()
	opslog.ObjId = model.GetId()
	opslog.ObjName = model.GetName()
	opslog.Action = action
	opslog.Notes = stringutils.Interface2String(notes)
	opslog.ProjectId = userCred.GetProjectId()
	opslog.Project = userCred.GetProjectName()
	opslog.UserId = userCred.GetUserId()
	opslog.User = userCred.GetUserName()
	opslog.DomainId = userCred.GetDomainId()
	opslog.Domain = userCred.GetDomainName()
	opslog.Roles = strings.Join(userCred.GetRoles(), ",")
	opslog.OpsTime = time.Now().UTC()

	if virtualModel, ok := model.(IVirtualModel); ok && virtualModel != nil {
		opslog.OwnerProjectId = virtualModel.GetOwnerProjectId()
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
	objTypes := jsonutils.GetQueryStringArray(query, "obj_type")
	if objTypes != nil && len(objTypes) > 0 {
		q = q.Filter(sqlchemy.In(q.Field("obj_type"), objTypes))
	}
	objIds := jsonutils.GetQueryStringArray(query, "obj_id")
	if objIds != nil && len(objIds) > 0 {
		q = q.Filter(sqlchemy.OR(sqlchemy.In(q.Field("obj_id"), objIds), sqlchemy.In(q.Field("obj_name"), objIds)))
	}
	action := jsonutils.GetQueryStringArray(query, "action")
	if action != nil && len(action) > 0 {
		q = q.Filter(sqlchemy.In(q.Field("action"), action))
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
	notes.Add(jsonutils.NewString(former.GetId()), "former_project_id")
	notes.Add(jsonutils.NewString(former.GetName()), "former_project")
	manager.LogEvent(m, ACT_CHANGE_OWNER, notes, userCred)
}

func (manager *SOpsLogManager) LogSyncUpdate(m IModel, diff map[string]sqlchemy.SUpdateDiff, userCred mcclient.TokenCredential) {
	diffStr := sqlchemy.UpdateDiffString(diff)
	if len(diffStr) > 0 {
		manager.LogEvent(m, ACT_SYNC_UPDATE, diffStr, userCred)
	}
}

func (manager *SOpsLogManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (manager *SOpsLogManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (self *SOpsLog) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return IsAdminAllowGet(userCred, self) || userCred.GetProjectId() == self.ProjectId || userCred.GetProjectId() == self.OwnerProjectId
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

func (self *SOpsLogManager) FilterByOwner(q *sqlchemy.SQuery, owner string) *sqlchemy.SQuery {
	if len(owner) > 0 {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.Equals(q.Field("owner_tenant_id"), owner),
			sqlchemy.Equals(q.Field("tenant_id"), owner),
		))
	}
	return q
}

func (manager *SOpsLogManager) GetOwnerId(userCred mcclient.IIdentityProvider) string {
	return userCred.GetProjectId()
}

func (self *SOpsLog) GetOwnerProjectId() string {
	return self.OwnerProjectId
}

func (self *SOpsLog) IsSharable() bool {
	return false
}
