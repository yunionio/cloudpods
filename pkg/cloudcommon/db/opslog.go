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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SOpsLogManager struct {
	SModelBaseManager
}

type SOpsLog struct {
	SModelBase

	Id      int64  `primary:"true" auto_increment:"true" list:"user"`
	ObjType string `width:"40" charset:"ascii" nullable:"false" list:"user" create:"required"`
	ObjId   string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	ObjName string `width:"128" charset:"utf8" nullable:"false" list:"user" create:"required"`
	Action  string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"required"`
	Notes   string `charset:"utf8" list:"user" create:"required"`

	ProjectId string `name:"tenant_id" width:"128" charset:"ascii" list:"user" create:"optional" index:"true"`
	Project   string `name:"tenant" width:"128" charset:"utf8" list:"user" create:"optional"`

	ProjectDomainId string `name:"project_domain_id" default:"default" width:"128" charset:"ascii" list:"user" create:"optional"`
	ProjectDomain   string `name:"project_domain" default:"Default" width:"128" charset:"utf8" list:"user" create:"optional"`

	UserId   string `width:"128" charset:"ascii" list:"user" create:"required"`
	User     string `width:"128" charset:"utf8" list:"user" create:"required"`
	DomainId string `width:"128" charset:"ascii" list:"user" create:"optional"`
	Domain   string `width:"128" charset:"utf8" list:"user" create:"optional"`
	Roles    string `width:"64" charset:"ascii" list:"user" create:"optional"`

	OpsTime time.Time `nullable:"false" list:"user"`

	OwnerDomainId  string `name:"owner_domain_id" default:"default" width:"128" charset:"ascii" list:"user" create:"optional"`
	OwnerProjectId string `name:"owner_tenant_id" width:"128" charset:"ascii" list:"user" create:"optional"`
}

var OpsLog *SOpsLogManager

var _ IModelManager = (*SOpsLogManager)(nil)
var _ IModel = (*SOpsLog)(nil)

var opslogQueryWorkerMan *appsrv.SWorkerManager

func init() {
	OpsLog = &SOpsLogManager{NewModelBaseManagerWithSplitable(
		SOpsLog{},
		"opslog_tbl",
		"event",
		"events",
		"id",
		"ops_time",
		consts.SplitableMaxDuration(),
		consts.SplitableMaxKeepSegments(),
	)}
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
	opslog := &SOpsLog{
		OpsTime: time.Now().UTC(),
		ObjType: model.Keyword(),
		ObjId:   model.GetId(),
		ObjName: model.GetName(),
		Action:  action,
		Notes:   stringutils.Interface2String(notes),

		ProjectId:       userCred.GetProjectId(),
		Project:         userCred.GetProjectName(),
		ProjectDomainId: userCred.GetProjectDomainId(),
		ProjectDomain:   userCred.GetProjectDomain(),

		UserId:   userCred.GetUserId(),
		User:     userCred.GetUserName(),
		DomainId: userCred.GetDomainId(),
		Domain:   userCred.GetDomainName(),
		Roles:    strings.Join(userCred.GetRoles(), ","),
	}
	opslog.SetModelManager(OpsLog, opslog)

	if virtualModel, ok := model.(IVirtualModel); ok {
		ownerId := virtualModel.GetOwnerId()
		if ownerId != nil {
			opslog.OwnerProjectId = ownerId.GetProjectId()
			opslog.OwnerDomainId = ownerId.GetProjectDomainId()
		}
	}

	err := manager.TableSpec().Insert(context.Background(), opslog)
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

// 操作日志列表
func (manager *SOpsLogManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (*sqlchemy.SQuery, error) {
	projStrs := jsonutils.GetQueryStringArray(query, "owner_project_ids")
	if len(projStrs) > 0 {
		for i := range projStrs {
			projObj, err := DefaultProjectFetcher(ctx, projStrs[i])
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError2("project", projStrs[i])
				} else {
					return nil, httperrors.NewGeneralError(err)
				}
			}
			projStrs[i] = projObj.GetId()
		}
		q = q.Filter(sqlchemy.In(q.Field("owner_tenant_id"), projStrs))
	}
	domainStrs := jsonutils.GetQueryStringArray(query, "owner_domain_ids")
	if len(domainStrs) > 0 {
		for i := range domainStrs {
			domainObj, err := DefaultDomainFetcher(ctx, domainStrs[i])
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError2("domain", domainStrs[i])
				} else {
					return nil, httperrors.NewGeneralError(err)
				}
			}
			domainStrs[i] = domainObj.GetId()
		}
		q = q.Filter(sqlchemy.In(q.Field("owner_domain_id"), domainStrs))
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
		case rbacutils.ScopeUser:
			if len(ownerId.GetUserId()) > 0 {
				q = q.Filter(sqlchemy.Equals(q.Field("user_id"), ownerId.GetUserId()))
			}
		case rbacutils.ScopeProject:
			if len(ownerId.GetProjectId()) > 0 {
				q = q.Filter(sqlchemy.Equals(q.Field("tenant_id"), ownerId.GetProjectId()))
			}
		case rbacutils.ScopeDomain:
			if len(ownerId.GetProjectDomainId()) > 0 {
				q = q.Filter(sqlchemy.Equals(q.Field("domain_id"), ownerId.GetProjectDomainId()))
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

func (manager *SOpsLogManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeUser
}

func (manager *SOpsLogManager) GetPagingConfig() *SPagingConfig {
	return &SPagingConfig{
		Order:        sqlchemy.SQL_ORDER_DESC,
		MarkerFields: []string{"id"},
		DefaultLimit: 20,
	}
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
