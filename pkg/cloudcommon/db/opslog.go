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
	"runtime/debug"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SOpsLogManager struct {
	SLogBaseManager
}

type SOpsLog struct {
	SLogBase

	ObjType string `width:"40" charset:"ascii" nullable:"false" list:"user" create:"required"`
	ObjId   string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	ObjName string `width:"128" charset:"utf8" nullable:"false" list:"user" create:"required"`
	Action  string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"required"`
	Notes   string `charset:"utf8" list:"user" create:"optional"`

	ProjectId string `name:"tenant_id" width:"128" charset:"ascii" list:"user" create:"optional" index:"true"`
	Project   string `name:"tenant" width:"128" charset:"utf8" list:"user" create:"optional"`

	ProjectDomainId string `name:"project_domain_id" default:"default" width:"128" charset:"ascii" list:"user" create:"optional"`
	ProjectDomain   string `name:"project_domain" default:"Default" width:"128" charset:"utf8" list:"user" create:"optional"`

	UserId   string `width:"128" charset:"ascii" list:"user" create:"required"`
	User     string `width:"128" charset:"utf8" list:"user" create:"required"`
	DomainId string `width:"128" charset:"ascii" list:"user" create:"optional"`
	Domain   string `width:"128" charset:"utf8" list:"user" create:"optional"`
	Roles    string `width:"64" charset:"utf8" list:"user" create:"optional"`

	OpsTime time.Time `nullable:"false" list:"user" clickhouse_ttl:"6m"`

	OwnerDomainId  string `name:"owner_domain_id" default:"default" width:"128" charset:"ascii" list:"user" create:"optional"`
	OwnerProjectId string `name:"owner_tenant_id" width:"128" charset:"ascii" list:"user" create:"optional"`
}

var OpsLog *SOpsLogManager

var _ IModelManager = (*SOpsLogManager)(nil)
var _ IModel = (*SOpsLog)(nil)

var opslogQueryWorkerMan *appsrv.SWorkerManager
var opslogWriteWorkerMan *appsrv.SWorkerManager

func NewOpsLogManager(opslog interface{}, tblName string, keyword, keywordPlural string, timeField string, clickhouse bool) SOpsLogManager {
	return SOpsLogManager{
		SLogBaseManager: NewLogBaseManager(opslog, tblName, keyword, keywordPlural, timeField, clickhouse),
	}
}

func InitOpsLog() {
	tmp := NewOpsLogManager(SOpsLog{}, "opslog_tbl", "event", "events", "ops_time", consts.OpsLogWithClickhouse)
	OpsLog = &tmp
	OpsLog.SetVirtualObject(OpsLog)

	opslogQueryWorkerMan = appsrv.NewWorkerManager("opslog_query_worker", 2, 512, true)
	opslogWriteWorkerMan = appsrv.NewWorkerManager("opslog_write_worker", 1, 2048, true)
}

func (manager *SOpsLogManager) CustomizeHandlerInfo(info *appsrv.SHandlerInfo) {
	manager.SModelBaseManager.CustomizeHandlerInfo(info)

	switch info.GetName(nil) {
	case "list":
		info.SetProcessTimeout(time.Minute * 15).SetWorkerManager(opslogQueryWorkerMan)
	}
}

func (opslog *SOpsLog) GetName() string {
	return fmt.Sprintf("%s-%s", opslog.ObjType, opslog.Action)
}

func (opslog *SOpsLog) GetUpdatedAt() time.Time {
	return opslog.OpsTime
}

func (opslog *SOpsLog) GetUpdateVersion() int {
	return 1
}

func (opslog *SOpsLog) GetModelManager() IModelManager {
	return OpsLog
}

func (manager *SOpsLogManager) LogEvent(model IModel, action string, notes interface{}, userCred mcclient.TokenCredential) {
	if !consts.OpsLogEnabled() {
		return
	}

	var (
		objId   = model.GetId()
		objName = model.GetName()
	)
	if objId == "" {
		if joint, ok := model.(IJointModel); ok {
			var (
				mm = JointMaster(joint)
				ms = JointSlave(joint)
			)
			if mm == nil || ms == nil {
				log.Errorf("logevent for jointmodel with nil sides %v/%v\n%s", mm, ms, debug.Stack())
				return
			}
			objId = mm.GetId() + "/" + ms.GetId()
			objName = mm.GetName() + "/" + ms.GetName()
		} else {
			log.Errorf("logevent for an object without ID: %T\n%s", model, debug.Stack())
			return
		}
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
	opslog := &SOpsLog{
		OpsTime: time.Now().UTC(),
		ObjType: model.Keyword(),
		ObjId:   objId,
		ObjName: objName,
		Action:  action,
		Notes:   stringutils.Interface2String(notes),
	}
	if userCred == nil {
		log.Warningf("Log event with empty userCred: objType=%s objId=%s objName=%s action=%s", model.Keyword(), objId, objName, action)
		const unknown = "unknown"
		opslog.ProjectId = unknown
		opslog.Project = unknown
		opslog.ProjectDomainId = unknown
		opslog.ProjectDomain = unknown
		opslog.UserId = unknown
		opslog.User = unknown
		opslog.DomainId = unknown
		opslog.Domain = unknown
		opslog.Roles = unknown
	} else {
		opslog.ProjectId = userCred.GetProjectId()
		opslog.Project = userCred.GetProjectName()
		opslog.ProjectDomainId = userCred.GetProjectDomainId()
		opslog.ProjectDomain = userCred.GetProjectDomain()
		opslog.UserId = userCred.GetUserId()
		opslog.User = userCred.GetUserName()
		opslog.DomainId = userCred.GetDomainId()
		opslog.Domain = userCred.GetDomainName()
		opslog.Roles = strings.Join(userCred.GetRoles(), ",")
	}
	opslog.SetModelManager(OpsLog, opslog)

	if virtualModel, ok := model.(IVirtualModel); ok {
		ownerId := virtualModel.GetOwnerId()
		if ownerId != nil {
			opslog.OwnerProjectId = ownerId.GetProjectId()
			opslog.OwnerDomainId = ownerId.GetProjectDomainId()
		}
	}

	opslogWriteWorkerMan.Run(opslog, nil, nil)
}

func (opslog *SOpsLog) Run() {
	err := OpsLog.TableSpec().Insert(context.Background(), opslog)
	if err != nil {
		log.Errorf("fail to insert opslog: %s", err)
	}
}

func (opslog *SOpsLog) Dump() string {
	return fmt.Sprintf("[%s] %s %s", timeutils.CompactTime(opslog.OpsTime), opslog.Action, opslog.Notes)
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

// 操作日志列表
func (manager *SOpsLogManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.OpsLogListInput,
) (*sqlchemy.SQuery, error) {
	for idx, domainId := range input.OwnerDomainIds {
		domainObj, err := DefaultDomainFetcher(ctx, domainId)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("domain", domainId)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		input.OwnerDomainIds[idx] = domainObj.GetId()
	}
	if len(input.OwnerDomainIds) > 0 {
		q = q.Filter(sqlchemy.In(q.Field("owner_domain_id"), input.OwnerDomainIds))
	}
	for idx, projectId := range input.OwnerProjectIds {
		domainId := ""
		if len(input.OwnerDomainIds) == 1 {
			domainId = input.OwnerDomainIds[0]
		}
		projObj, err := DefaultProjectFetcher(ctx, projectId, domainId)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("project", projectId)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		input.OwnerProjectIds[idx] = projObj.GetId()
	}
	if len(input.OwnerProjectIds) > 0 {
		q = q.Filter(sqlchemy.In(q.Field("owner_tenant_id"), input.OwnerProjectIds))
	}
	if len(input.ObjTypes) > 0 {
		if len(input.ObjTypes) == 1 {
			q = q.Filter(sqlchemy.Equals(q.Field("obj_type"), input.ObjTypes[0]))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("obj_type"), input.ObjTypes))
		}
	}
	if len(input.Objs) > 0 {
		if len(input.Objs) == 1 {
			q = q.Filter(sqlchemy.OR(sqlchemy.Equals(q.Field("obj_id"), input.Objs[0]), sqlchemy.Equals(q.Field("obj_name"), input.Objs[0])))
		} else {
			q = q.Filter(sqlchemy.OR(sqlchemy.In(q.Field("obj_id"), input.Objs), sqlchemy.In(q.Field("obj_name"), input.Objs)))
		}
	}
	if len(input.ObjIds) > 0 {
		if len(input.ObjIds) == 1 {
			q = q.Filter(sqlchemy.Equals(q.Field("obj_id"), input.ObjIds[0]))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("obj_id"), input.ObjIds))
		}
	}
	if len(input.ObjNames) > 0 {
		if len(input.ObjNames) == 1 {
			q = q.Filter(sqlchemy.Equals(q.Field("obj_name"), input.ObjNames[0]))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("obj_name"), input.ObjNames))
		}
	}
	if len(input.Actions) > 0 {
		if len(input.Actions) == 1 {
			q = q.Filter(sqlchemy.Equals(q.Field("action"), input.Actions[0]))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("action"), input.Actions))
		}
	}
	//if !IsAdminAllowList(userCred, manager) {
	// 	q = q.Filter(sqlchemy.OR(
	//		sqlchemy.Equals(q.Field("owner_tenant_id"), manager.GetOwnerId(userCred)),
	//		sqlchemy.Equals(q.Field("tenant_id"), manager.GetOwnerId(userCred)),
	//	))
	//}
	if !input.Since.IsZero() {
		q = q.GT("ops_time", input.Since)
	}
	if !input.Until.IsZero() {
		q = q.LE("ops_time", input.Until)
	}
	return q, nil
}

func (manager *SOpsLogManager) SyncOwner(m IModel, former *STenant, userCred mcclient.TokenCredential) {
	notes := jsonutils.NewDict()
	notes.Add(jsonutils.NewString(former.GetProjectDomainId()), "former_domain_id")
	notes.Add(jsonutils.NewString(former.GetProjectDomain()), "former_domain")
	notes.Add(jsonutils.NewString(former.GetProjectId()), "former_project_id")
	notes.Add(jsonutils.NewString(former.GetProjectName()), "former_project")
	manager.LogEvent(m, ACT_CHANGE_OWNER, notes, userCred)
}

func (manager *SOpsLogManager) LogSyncUpdate(m IModel, uds sqlchemy.UpdateDiffs, userCred mcclient.TokenCredential) {
	if len(uds) > 0 {
		manager.LogEvent(m, ACT_SYNC_UPDATE, uds, userCred)
	}
}

func (self *SOpsLogManager) FilterByOwner(q *sqlchemy.SQuery, man FilterByOwnerProvider, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if ownerId != nil {
		switch scope {
		case rbacscope.ScopeUser:
			if len(ownerId.GetUserId()) > 0 {
				/*
				 * 默认只能查看本人发起的操作
				 */
				q = q.Filter(sqlchemy.Equals(q.Field("user_id"), ownerId.GetUserId()))
			}
		case rbacscope.ScopeProject:
			if len(ownerId.GetProjectId()) > 0 {
				/*
				 * 项目视图可以查看本项目人员发起的操作，或者对本项目资源实施的操作, QIU Jian
				 */
				q = q.Filter(sqlchemy.OR(
					sqlchemy.Equals(q.Field("tenant_id"), ownerId.GetProjectId()),
					sqlchemy.Equals(q.Field("owner_tenant_id"), ownerId.GetProjectId()),
				))
			}
		case rbacscope.ScopeDomain:
			if len(ownerId.GetProjectDomainId()) > 0 {
				/*
				 * 域视图可以查看本域人员发起的操作，或者对本域资源实施的操作, QIU Jian
				 */
				q = q.Filter(sqlchemy.OR(
					sqlchemy.Equals(q.Field("project_domain_id"), ownerId.GetProjectDomainId()),
					sqlchemy.Equals(q.Field("owner_domain_id"), ownerId.GetProjectDomainId()),
				))
			}
		default:
			// systemScope, no filter
		}
	}
	return q
}

func (self *SOpsLog) GetOwnerId() mcclient.IIdentityProvider {
	owner := SOwnerId{
		Domain:       self.ProjectDomain,
		DomainId:     self.ProjectDomainId,
		Project:      self.Project,
		ProjectId:    self.ProjectId,
		User:         self.User,
		UserId:       self.UserId,
		UserDomain:   self.Domain,
		UserDomainId: self.DomainId,
	}
	return &owner
}

func (self *SOpsLog) IsSharable(reqCred mcclient.IIdentityProvider) bool {
	return false
}

func (manager *SOpsLogManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeUser
}

func (manager *SOpsLogManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	ownerId := SOwnerId{}
	err := data.Unmarshal(&ownerId)
	if err != nil {
		return nil, errors.Wrap(err, "data.Unmarshal")
	}
	if ownerId.IsValid() {
		return &ownerId, nil
	}
	return FetchUserInfo(ctx, data)
}

func (manager *SOpsLogManager) ValidateCreateData(ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data apis.OpsLogCreateInput,
) (apis.OpsLogCreateInput, error) {
	data.User = ownerId.GetUserName()
	return data, nil
}

func (log *SOpsLog) CustomizeCreate(ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) error {
	log.User = ownerId.GetUserName()
	log.UserId = ownerId.GetUserId()
	log.Domain = ownerId.GetDomainName()
	log.DomainId = ownerId.GetDomainId()
	log.Project = ownerId.GetProjectName()
	log.ProjectId = ownerId.GetProjectId()
	log.ProjectDomain = ownerId.GetProjectDomain()
	log.ProjectDomainId = ownerId.GetProjectDomainId()
	return log.SModelBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

// override
func (log *SOpsLog) GetRecordTime() time.Time {
	return log.OpsTime
}

func (manager *SOpsLogManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.OpsLogDetails {
	rows := make([]apis.OpsLogDetails, len(objs))

	projectIds := make([]string, len(rows))
	domainIds := make([]string, len(rows))
	for i := range rows {
		var base *SOpsLog
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find OpsLog in %#v: %s", objs[i], err)
		} else {
			if len(base.OwnerProjectId) > 0 {
				projectIds[i] = base.OwnerProjectId
			} else if len(base.OwnerDomainId) > 0 {
				domainIds[i] = base.OwnerDomainId
			}
		}
	}

	projects := DefaultProjectsFetcher(ctx, projectIds, false)
	domains := DefaultProjectsFetcher(ctx, domainIds, true)

	for i := range rows {
		if project, ok := projects[projectIds[i]]; ok {
			rows[i].OwnerProject = project.Name
			rows[i].OwnerDomain = project.Domain
		} else if domain, ok := domains[domainIds[i]]; ok {
			rows[i].OwnerDomain = domain.Name
		}
	}

	return rows
}
