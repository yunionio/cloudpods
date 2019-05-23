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

package models

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SProjectManager struct {
	SEnabledIdentityBaseResourceManager
}

var ProjectManager *SProjectManager

func init() {
	ProjectManager = &SProjectManager{
		SEnabledIdentityBaseResourceManager: NewEnabledIdentityBaseResourceManager(
			SProject{},
			"project",
			"project",
			"projects",
		),
	}
	ProjectManager.SetVirtualObject(ProjectManager)
}

/*
+-------------+-------------+------+-----+---------+-------+
| Field       | Type        | Null | Key | Default | Extra |
+-------------+-------------+------+-----+---------+-------+
| id          | varchar(64) | NO   | PRI | NULL    |       |
| name        | varchar(64) | NO   |     | NULL    |       |
| extra       | text        | YES  |     | NULL    |       |
| description | text        | YES  |     | NULL    |       |
| enabled     | tinyint(1)  | YES  |     | NULL    |       |
| domain_id   | varchar(64) | NO   | MUL | NULL    |       |
| parent_id   | varchar(64) | YES  | MUL | NULL    |       |
| is_domain   | tinyint(1)  | NO   |     | 0       |       |
| created_at  | datetime    | YES  |     | NULL    |       |
+-------------+-------------+------+-----+---------+-------+
*/

type SBaseProject struct {
	SEnabledIdentityBaseResource

	ParentId string            `width:"64" charset:"ascii" index:"true" list:"admin" create:"admin_optional"`
	IsDomain tristate.TriState `default:"false" nullable:"false" create:"admin_optional"`
}

type SProject struct {
	SBaseProject
}

func (manager *SProjectManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{UserManager},
		{GroupManager},
	}
}

func (manager *SProjectManager) InitializeData() error {
	return manager.initSysProject()
}

func (manager *SProjectManager) initSysProject() error {
	q := manager.Query().Equals("name", api.SystemAdminProject)
	q = q.Equals("domain_id", api.DEFAULT_DOMAIN_ID)
	cnt, err := q.CountWithError()
	if err != nil {
		return errors.Wrap(err, "query")
	}
	if cnt == 1 {
		return nil
	}
	if cnt > 2 {
		// ???
		log.Fatalf("duplicate system project???")
	}
	// insert
	project := SProject{}
	project.Name = api.SystemAdminProject
	project.DomainId = api.DEFAULT_DOMAIN_ID
	project.Enabled = tristate.True
	project.Description = "Boostrap system default admin project"
	project.IsDomain = tristate.False
	project.ParentId = api.DEFAULT_DOMAIN_ID
	project.SetModelManager(manager, &project)

	err = manager.TableSpec().Insert(&project)
	if err != nil {
		return errors.Wrap(err, "insert")
	}
	return nil
}

func (manager *SProjectManager) Query(fields ...string) *sqlchemy.SQuery {
	return manager.SEnabledIdentityBaseResourceManager.Query(fields...).IsFalse("is_domain")
}

func (manager *SProjectManager) FetchProjectByName(projectName string, domainId, domainName string) (*SProject, error) {
	obj, err := db.NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	domain, err := DomainManager.FetchDomain(domainId, domainName)
	if err != nil {
		return nil, err
	}
	q := manager.Query().Equals("name", projectName).Equals("domain_id", domain.Id)
	err = q.First(obj)
	if err != nil {
		return nil, err
	}
	return obj.(*SProject), err
}

func (manager *SProjectManager) FetchProjectById(projectId string) (*SProject, error) {
	obj, err := db.NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	q := manager.Query().Equals("id", projectId)
	err = q.First(obj)
	if err != nil {
		return nil, err
	}
	return obj.(*SProject), err
}

func (manager *SProjectManager) FetchProject(projectId, projectName string, domainId, domainName string) (*SProject, error) {
	if len(projectId) > 0 {
		return manager.FetchProjectById(projectId)
	}
	if len(projectName) > 0 {
		return manager.FetchProjectByName(projectName, domainId, domainName)
	}
	return nil, fmt.Errorf("no project Id or name provided")
}

type SProjectExtended struct {
	SProject

	DomainName string
}

func (proj *SProject) getDomain() (*SDomain, error) {
	return DomainManager.FetchDomainById(proj.DomainId)
}

func (proj *SProject) FetchExtend() (*SProjectExtended, error) {
	domain, err := proj.getDomain()
	if err != nil {
		return nil, err
	}
	ext := SProjectExtended{
		SProject:   *proj,
		DomainName: domain.Name,
	}
	return &ext, nil
}

func (manager *SProjectManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledIdentityBaseResourceManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	userStr := jsonutils.GetAnyString(query, []string{"user_id"})
	if len(userStr) > 0 {
		userObj, err := UserManager.FetchById(userStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(UserManager.Keyword(), userStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		subq := AssignmentManager.fetchUserProjectIdsQuery(userObj.GetId())
		q = q.In("id", subq.SubQuery())
	}

	groupStr := jsonutils.GetAnyString(query, []string{"group_id"})
	if len(groupStr) > 0 {
		groupObj, err := GroupManager.FetchById(groupStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(GroupManager.Keyword(), groupStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		subq := AssignmentManager.fetchGroupProjectIdsQuery(groupObj.GetId())
		q = q.In("id", subq.SubQuery())
	}

	return q, nil
}

func (model *SProject) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	model.ParentId = ownerId.GetProjectDomainId()
	model.IsDomain = tristate.False
	return model.SBaseProject.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (proj *SProject) GetUserCount() (int, error) {
	q := AssignmentManager.fetchProjectUserIdsQuery(proj.Id)
	return q.CountWithError()
}

func (proj *SProject) GetGroupCount() (int, error) {
	q := AssignmentManager.fetchProjectGroupIdsQuery(proj.Id)
	return q.CountWithError()
}

func (proj *SProject) ValidateDeleteCondition(ctx context.Context) error {
	usrCnt, _ := proj.GetUserCount()
	if usrCnt > 0 {
		return httperrors.NewNotEmptyError("project contains user")
	}
	grpCnt, _ := proj.GetGroupCount()
	if grpCnt > 0 {
		return httperrors.NewNotEmptyError("project contains group")
	}
	if proj.IsAdminProject() {
		return httperrors.NewForbiddenError("cannot delete system project")
	}
	return proj.SEnabledIdentityBaseResource.ValidateDeleteCondition(ctx)
}

func (proj *SProject) IsAdminProject() bool {
	return proj.Name == api.SystemAdminProject && proj.DomainId == api.DEFAULT_DOMAIN_ID
}

func (proj *SProject) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("name") {
		if proj.IsAdminProject() {
			return nil, httperrors.NewForbiddenError("cannot alter system project name")
		}
	}
	return proj.SEnabledIdentityBaseResource.ValidateUpdateData(ctx, userCred, query, data)
}

func (proj *SProject) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := proj.SEnabledIdentityBaseResource.GetCustomizeColumns(ctx, userCred, query)
	return projectExtra(proj, extra)
}

func (proj *SProject) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := proj.SEnabledIdentityBaseResource.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return projectExtra(proj, extra), nil
}

func projectExtra(proj *SProject, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	grpCnt, _ := proj.GetGroupCount()
	extra.Add(jsonutils.NewInt(int64(grpCnt)), "group_count")
	usrCnt, _ := proj.GetUserCount()
	extra.Add(jsonutils.NewInt(int64(usrCnt)), "user_count")
	return extra
}
