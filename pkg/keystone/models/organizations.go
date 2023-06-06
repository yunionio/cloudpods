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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SOrganizationManager struct {
	db.SStandaloneResourceBaseManager

	cache *db.SCacheManager[SOrganization]
}

var OrganizationManager *SOrganizationManager

func init() {
	OrganizationManager = &SOrganizationManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SOrganization{},
			"organizations_tbl",
			"organization",
			"organizations",
		),
	}
	OrganizationManager.SetVirtualObject(OrganizationManager)
	OrganizationManager.cache = db.NewCacheManager[SOrganization](OrganizationManager)
}

type SOrganization struct {
	db.SStandaloneResourceBase

	ParentId string `width:"128" charset:"ascii" list:"user" create:"admin_required"`

	Parent string `ignore:"true"`

	RootId string `width:"128" charset:"ascii" list:"user" create:"admin_required"`

	Root string `ignore:"true"`

	Type api.TOrgType `width:"32" charset:"ascii" list:"user" create:"admin_required"`

	Keys []string `width:"256" charset:"utf8" list:"user" create:"admin_optional"`
}

func (manager *SOrganizationManager) fetchOrganizationById(orgId string) (*SOrganization, error) {
	org, err := manager.cache.FetchById(orgId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			obj, err := manager.FetchById(orgId)
			if err != nil {
				return nil, errors.Wrap(err, "manager.FetchById")
			}
			manager.cache.Invalidate()
			org = obj.(*SOrganization)
		} else {
			return nil, errors.Wrapf(err, "cache.FetchById %s", orgId)
		}
	}
	return org, nil
}

// 项目列表
func (manager *SOrganizationManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.OrganizationListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}

	if len(query.Type) > 0 {
		q = q.In("type", query.Type)
	}

	if len(query.RootId) > 0 {
		orgObj, err := manager.FetchByIdOrName(userCred, query.RootId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(manager.Keyword(), query.RootId)
			} else {
				return nil, errors.Wrap(err, "FetchByIdOrName")
			}
		}
		query.RootId = orgObj.GetId()
		q = q.Filter(sqlchemy.OR(
			sqlchemy.Equals(q.Field("id"), query.RootId),
			sqlchemy.Equals(q.Field("root_id"), query.RootId),
		))
	}

	if query.RootOnly != nil && *query.RootOnly {
		q = q.Equals("parent_id", api.OrganizationRootParent)
	}

	return q, nil
}

func (manager *SOrganizationManager) ExtendListQuery(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.OrganizationListInput,
) (*sqlchemy.SQuery, error) {
	{
		// parent
		subqParent := manager.Query("id", "name").SubQuery()
		q = q.LeftJoin(subqParent, sqlchemy.Equals(q.Field("parent_id"), subqParent.Field("id")))
		q = q.AppendField(subqParent.Field("name").Label("parent"))
	}
	{
		// root
		subqRoot := manager.Query("id", "name").SubQuery()
		q = q.LeftJoin(subqRoot, sqlchemy.Equals(q.Field("root_id"), subqRoot.Field("id")))
		q = q.AppendField(subqRoot.Field("name").Label("root"))
	}
	return q, nil
}

func (manager *SOrganizationManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.OrganizationListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SOrganizationManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SOrganizationManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []SOrganization {
	rows := make([]SOrganization, len(objs))
	for i := range rows {
		org := objs[i].(*SOrganization)
		rows[i] = *org
	}
	return rows
}

func (manager *SOrganizationManager) FetchOrgnaizations(filter func(q *sqlchemy.SQuery) *sqlchemy.SQuery) ([]SOrganization, error) {
	q := manager.Query()
	q = filter(q)
	ret := make([]SOrganization, 0)
	err := db.FetchModelObjects(manager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return ret, nil
}

func (org *SOrganization) GetOrgChildCount() (int, error) {
	q := OrganizationManager.Query().Equals("root_id", org.Id)
	cnt, err := q.CountWithError()
	if err != nil {
		return 0, errors.Wrap(err, "CountWithError")
	}
	return cnt, nil
}

func (org *SOrganization) GetDirectChildCount() (int, error) {
	q := OrganizationManager.Query().Equals("parent_id", org.Id)
	cnt, err := q.CountWithError()
	if err != nil {
		return 0, errors.Wrap(err, "CountWithError")
	}
	return cnt, nil
}

func (org *SOrganization) IsRoot() bool {
	return org.ParentId == api.OrganizationRootParent
}

func (org *SOrganization) ValidateDeleteCondition(ctx context.Context, info *api.ProjectDetails) error {
	var childCnt int
	var err error
	if org.IsRoot() {
		childCnt, err = org.GetOrgChildCount()
		if err != nil {
			return errors.Wrap(err, "GetOrgChildCount")
		}
	} else {
		childCnt, err = org.GetDirectChildCount()
		if err != nil {
			return errors.Wrap(err, "GetChildCount")
		}
	}
	if childCnt > 0 {
		return errors.Wrap(httperrors.ErrNotEmpty, "not an empty organization")
	}
	return org.SStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (manager *SOrganizationManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.OrganizationCreateInput,
) (api.OrganizationCreateInput, error) {
	var err error

	input.StandaloneResourceCreateInput, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBaseManager.ValidateCreateData")
	}

	if !api.IsValidOrgType(input.Type) {
		return input, errors.Wrapf(httperrors.ErrInputParameter, "invalid organization type %s", input.Type)
	}

	if len(input.ParentId) > 0 {
		parentObj, err := manager.FetchByIdOrName(userCred, input.ParentId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, httperrors.NewResourceNotFoundError2(manager.Keyword(), input.ParentId)
			} else {
				return input, errors.Wrap(err, "FetchByIdOrName")
			}
		}
		parent := parentObj.(*SOrganization)
		input.Level = parent.Level + 1
		input.ParentId = parent.Id

		root, err := parent.GetRoot()
		if err != nil {
			return input, errors.Wrap(err, "GetRoot")
		}
		input.RootId = root.Id

		if root.Info == nil {
			return input, errors.Wrapf(errors.ErrInvalidStatus, "empty root info")
		}

		if len(root.Info.Keys) < int(input.Level) {
			return input, errors.Wrapf(httperrors.ErrTooLarge, "input level more than allowd %d", len(root.Info.Keys))
		}

		input.Info = &api.SOrganizationInfo{
			Tags: make(map[string]string),
		}
		for k, v := range parent.Info.Tags {
			input.Info.Tags[k] = v
		}

		input.Info.Tags[root.Info.Keys[input.Level-1]] = input.Name

	} else {
		input.ParentId = api.OrganizationRootParent
		input.RootId = api.OrganizationRootParent

		input.Level = 0

		if input.Info == nil || len(input.Info.Keys) == 0 {
			return input, errors.Wrap(httperrors.ErrInputParameter, "empy keys")
		}
		input.Info.Tags = nil
	}

	return input, nil
}

func (org *SOrganization) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	org.SStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	OrganizationManager.cache.Update(org)
}

func (org *SOrganization) GetParent() (*SOrganization, error) {
	if org.ParentId == api.OrganizationRootParent {
		return nil, nil
	}
	parent, err := OrganizationManager.fetchOrganizationById(org.ParentId)
	if err != nil {
		return nil, errors.Wrap(err, "fetchOrganizationById")
	}
	return parent, nil
}

func (org *SOrganization) GetRoot() (*SOrganization, error) {
	if org.ParentId == api.OrganizationRootParent {
		return org, nil
	}
	parent, err := org.GetParent()
	if err != nil {
		return nil, errors.Wrap(err, "GetParent")
	}
	root, err := parent.GetRoot()
	if err != nil {
		return nil, errors.Wrap(err, "GetRoot")
	}
	return root, nil
}

func (org *SOrganization) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.OrganizationUpdateInput,
) (api.OrganizationUpdateInput, error) {
	if !org.IsRoot() {
		// allow update description only
		if len(input.Name) > 0 && input.Name != org.Name {
			return input, errors.Wrap(httperrors.ErrForbidden, "not allow to update name of non-root node")
		}
	}
	return input, nil
}

func (org *SOrganization) PostUpdate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	org.SStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
	OrganizationManager.cache.Update(org)
}

func (org *SOrganization) PerformAddLevel(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.OrganizationPerformAddLevelsInput,
) (jsonutils.JSONObject, error) {
	var err error
	if !org.IsRoot() {
		org, err = org.GetRoot()
		if err != nil {
			return nil, errors.Wrap(err, "GetRoot")
		}
	}
	info := *org.Info
	info.Keys = append(info.Keys, input.Keys...)
	_, err = db.Update(org, func() error {
		org.Info = &info
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "Update")
	}
	db.OpsLog.LogEvent(org, db.ACT_UPDATE, org.GetShortDesc(ctx), userCred)
	logclient.AddSimpleActionLog(org, logclient.ACT_UPDATE, org.GetShortDesc(ctx), userCred, true)
	return nil, nil
}

func (org *SOrganization) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := org.SStandaloneResourceBase.GetShortDesc(ctx)
	desc.Set("keys", jsonutils.Marshal(org.Info.Keys))
	return desc
}

func (org *SOrganization) PerformBind(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.OrganizationPerformBindInput,
) (jsonutils.JSONObject, error) {
	if org.IsRoot() {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "cannot bind to root organization")
	}
	org.startOrganizationBindTask(ctx, userCred, input)
	return nil, nil
}

func (org *SOrganization) startOrganizationBindTask(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input api.OrganizationPerformBindInput,
) error {
	params := jsonutils.NewDict()
	params.Set("input", jsonutils.Marshal(input))
	task, err := taskman.TaskManager.NewTask(ctx, "OrganizationBindTask", org, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (org *SOrganization) BindTargetIds(ctx context.Context, userCred mcclient.TokenCredential, input api.OrganizationPerformBindInput) error {
	var errs []error
	for _, id := range input.TargetId {
		err := org.bindTargetId(ctx, userCred, input.ResourceType, id)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.NewAggregate(errs)
}

func (org *SOrganization) bindTargetId(ctx context.Context, userCred mcclient.TokenCredential, resType string, idstr string) error {
	switch org.Type {
	case api.OrgTypeProject:
		return org.bindProject(ctx, userCred, idstr)
	case api.OrgTypeDomain:
		return org.bindDomain(ctx, userCred, idstr)
	case api.OrgTypeObject:
		return org.bindObject(ctx, userCred, resType, idstr)
	}
	return errors.Wrapf(errors.ErrNotSupported, "cannot bind to %s %s", resType, idstr)
}

func (org *SOrganization) bindProject(ctx context.Context, userCred mcclient.TokenCredential, idstr string) error {
	projObj, err := ProjectManager.FetchById(idstr)
	if err != nil {
		return errors.Wrapf(err, "ProjectManager.FetchById %s", idstr)
	}
	return projObj.(*SProject).SetOrganizationMetadataAll(ctx, org.Info.Tags, userCred)
}

func (org *SOrganization) bindDomain(ctx context.Context, userCred mcclient.TokenCredential, idstr string) error {
	domainObj, err := DomainManager.FetchById(idstr)
	if err != nil {
		return errors.Wrapf(err, "DomainManager.FetchById %s", idstr)
	}
	return domainObj.(*SDomain).SetOrganizationMetadataAll(ctx, org.Info.Tags, userCred)
}

func (org *SOrganization) bindObject(ctx context.Context, userCred mcclient.TokenCredential, resType, idstr string) error {
	s := auth.GetSession(ctx, userCred, consts.GetRegion())
	module, err := modulebase.GetModule(s, resType)
	if err != nil {
		return errors.Wrap(err, "GetModule")
	}
	input := jsonutils.Marshal(org.Info.Tags)
	_, err = module.PerformAction(s, idstr, "set-org-metadata", input)
	if err != nil {
		return errors.Wrapf(err, "PerformAction set-org-metadata %s", idstr)
	}
	return nil
}
