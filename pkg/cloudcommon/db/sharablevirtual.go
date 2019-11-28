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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SSharableVirtualResourceBase struct {
	SVirtualResourceBase

	IsPublic    bool   `default:"false" nullable:"false" create:"domain_optional" list:"user"`
	PublicScope string `width:"16" charset:"ascii" nullable:"false" default:"system" create:"domain_optional" list:"user"`
}

type SSharableVirtualResourceBaseManager struct {
	SVirtualResourceBaseManager
}

func NewSharableVirtualResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SSharableVirtualResourceBaseManager {
	return SSharableVirtualResourceBaseManager{SVirtualResourceBaseManager: NewVirtualResourceBaseManager(dt, tableName, keyword, keywordPlural)}
}

func (manager *SSharableVirtualResourceBaseManager) GetISharableVirtualModelManager() ISharableVirtualModelManager {
	return manager.GetVirtualObject().(ISharableVirtualModelManager)
}

func (manager *SSharableVirtualResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacutils.ScopeProject:
			ownerProjectid := owner.GetProjectId()
			if len(ownerProjectid) > 0 {
				subq := SharedResourceManager.Query("resource_id")
				subq = subq.Equals("resource_type", manager.Keyword())
				subq = subq.Equals("target_project_id", ownerProjectid)
				subq = subq.Equals("owner_project_id", q.Field("tenant_id"))
				q = q.Filter(sqlchemy.OR(
					sqlchemy.Equals(q.Field("tenant_id"), ownerProjectid),
					sqlchemy.AND(
						sqlchemy.IsTrue(q.Field("is_public")),
						sqlchemy.Equals(q.Field("public_scope"), rbacutils.ScopeSystem),
					),
					sqlchemy.AND(
						sqlchemy.IsTrue(q.Field("is_public")),
						sqlchemy.Equals(q.Field("public_scope"), rbacutils.ScopeDomain),
						sqlchemy.Equals(q.Field("domain_id"), owner.GetProjectDomainId()),
					),
					sqlchemy.AND(
						sqlchemy.IsFalse(q.Field("is_public")),
						sqlchemy.In(q.Field("id"), subq.SubQuery()),
					),
				))
			}
		case rbacutils.ScopeDomain:
			if len(owner.GetProjectDomainId()) > 0 {
				q = q.Filter(sqlchemy.OR(
					sqlchemy.Equals(q.Field("domain_id"), owner.GetProjectDomainId()),
					sqlchemy.AND(
						sqlchemy.IsTrue(q.Field("is_public")),
						sqlchemy.Equals(q.Field("public_scope"), rbacutils.ScopeSystem),
					),
				))
			}
		}
	}
	return q
}

func (model *SSharableVirtualResourceBase) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return model.IsOwner(userCred) || model.IsPublic || IsAllowGet(rbacutils.ScopeSystem, userCred, model)
}

func (model *SSharableVirtualResourceBase) IsSharable(reqUsrId mcclient.IIdentityProvider) bool {
	if model.IsPublic {
		switch rbacutils.String2Scope(model.PublicScope) {
		case rbacutils.ScopeSystem:
			return true
		case rbacutils.ScopeDomain:
			ownerId := model.GetOwnerId()
			if ownerId != nil && ownerId.GetProjectDomainId() == reqUsrId.GetProjectDomainId() {
				return true
			}
		}
	}
	return false
}

func (model *SSharableVirtualResourceBase) AllowPerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAllowPerform(rbacutils.ScopeSystem, userCred, model, "public")
}

func (model *SSharableVirtualResourceBase) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	targetScopeStr, _ := data.GetString("scope")
	targetScope := rbacutils.String2ScopeDefault(targetScopeStr, rbacutils.ScopeSystem)
	if targetScope == rbacutils.ScopeProject {
		if sharedProjects, err := data.GetArray("shared_projects"); err == nil {
			delProjects := make([]*SSharedResource, 0)
			addProjects := make([]string, 0)
			ops := make(map[string]*SSharedResource, 0)
			nps := make([]string, 0)
			srs := make([]SSharedResource, 0)
			q := SharedResourceManager.Query()
			err := q.Filter(sqlchemy.AND(
				sqlchemy.Equals(q.Field("owner_project_id"), model.ProjectId),
				sqlchemy.Equals(q.Field("resource_id"), model.GetId()),
				sqlchemy.Equals(q.Field("resource_type"), model.GetModelManager().Keyword()),
			)).All(&srs)
			if err != nil {
				return nil, httperrors.NewInternalServerError("Fetch project error %s", err)
			}
			for i := 0; i < len(srs); i++ {
				ops[srs[i].TargetProjectId] = &srs[i]
			}
			for i := 0; i < len(sharedProjects); i++ {
				sharedProject, _ := sharedProjects[i].GetString()
				tenant, err := TenantCacheManager.FetchTenantByIdOrName(ctx, sharedProject)
				if err != nil {
					return nil, httperrors.NewBadRequestError("fetch tenant %s error %s", sharedProject, err)
				}
				if tenant.DomainId != model.DomainId {
					return nil, httperrors.NewBadRequestError("can't shared project to other domain")
				}
				if tenant.GetId() == model.ProjectId {
					return nil, httperrors.NewBadRequestError("Can't share project to yourself")
				}
				nps = append(nps, tenant.GetId())
				if _, ok := ops[tenant.GetId()]; !ok {
					addProjects = append(addProjects, tenant.GetId())
				}
			}
			for k, v := range ops {
				if !utils.IsInStringArray(k, nps) {
					delProjects = append(delProjects, v)
				}
			}

			for i := 0; i < len(addProjects); i++ {
				sharedResource := new(SSharedResource)
				sharedResource.ResourceType = model.GetModelManager().Keyword()
				sharedResource.ResourceId = model.Id
				sharedResource.OwnerProjectId = model.ProjectId
				sharedResource.TargetProjectId = addProjects[i]
				if insetErr := SharedResourceManager.TableSpec().Insert(sharedResource); insetErr != nil {
					return nil, httperrors.NewInternalServerError("Insert shared resource failed %s", insetErr)
				}
			}
			for i := 0; i < len(delProjects); i++ {
				delProjects[i].SetModelManager(SharedResourceManager, delProjects[i])
				if err := delProjects[i].Delete(ctx, userCred); err != nil {
					return nil, httperrors.NewInternalServerError("Unshare project failed %s", err)
				}
			}

			if (len(addProjects) + len(srs) - len(delProjects)) > 0 {
				diff, err := Update(model, func() error {
					model.IsPublic = false
					model.PublicScope = string(targetScope)
					return nil
				})
				if err == nil {
					OpsLog.LogEvent(model, ACT_UPDATE, diff, userCred)
				} else {
					return nil, err
				}
			}
		} else {
			return nil, httperrors.NewMissingParameterError("shared_projects")
		}
	} else {
		allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), model.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "public")
		if targetScope.HigherThan(allowScope) {
			return nil, httperrors.NewForbiddenError("not enough privilege")
		}
		if targetScope != rbacutils.ScopeSystem && targetScope != rbacutils.ScopeDomain {
			return nil, httperrors.NewInputParameterError("invalid scope %s", targetScope)
		}
		diff, err := Update(model, func() error {
			model.IsPublic = true
			model.PublicScope = string(targetScope)
			return nil
		})
		if err == nil {
			OpsLog.LogEvent(model, ACT_UPDATE, diff, userCred)
		} else {
			return nil, err
		}
	}
	model.GetIStandaloneModel().ClearSchedDescCache()
	return nil, nil
}

func (model *SSharableVirtualResourceBase) AllowPerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAllowPerform(rbacutils.ScopeSystem, userCred, model, "private")
}

func (model *SSharableVirtualResourceBase) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if model.IsPublic {
		allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), model.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "private")
		requireScope := rbacutils.String2ScopeDefault(model.PublicScope, rbacutils.ScopeSystem)
		if requireScope.HigherThan(allowScope) {
			return nil, httperrors.NewForbiddenError("not enough privileges: allow %s require %s", allowScope, requireScope)
		}
		diff, err := Update(model, func() error {
			model.PublicScope = string(rbacutils.ScopeNone)
			model.IsPublic = false
			return nil
		})
		if err == nil {
			OpsLog.LogEvent(model, ACT_UPDATE, diff, userCred)
		} else {
			return nil, httperrors.NewInternalServerError("Update shared resource error: %s", err)
		}
	}

	if err := SharedResourceManager.CleanModelSharedProjects(ctx, userCred, &model.SVirtualResourceBase); err != nil {
		return nil, err
	}
	model.GetIStandaloneModel().ClearSchedDescCache()
	return nil, nil
}

func (model *SSharableVirtualResourceBase) GetISharableVirtualModel() ISharableVirtualModel {
	return model.GetVirtualObject().(ISharableVirtualModel)
}

func (model *SSharableVirtualResourceBase) GetSharedProjects() []string {
	sharedResources := make([]SSharedResource, 0)
	res := make([]string, 0)
	SharedResourceManager.Query().Equals("resource_type", model.GetModelManager().Keyword()).Equals("resource_id", model.GetId()).All(&sharedResources)
	for i := 0; i < len(sharedResources); i++ {
		res = append(res, sharedResources[i].TargetProjectId)
	}
	return res
}

func (model *SSharableVirtualResourceBase) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	sharedProjects := jsonutils.NewArray()
	projects := model.GetSharedProjects()
	for i := 0; i < len(projects); i++ {
		tenant, err := TenantCacheManager.FetchTenantByIdOrName(ctx, projects[i])
		if err != nil {
			log.Errorf("failed fetch tenant by id %s", projects[i])
			continue
		}
		p := jsonutils.NewDict()
		p.Set("id", jsonutils.NewString(tenant.GetId()))
		p.Set("name", jsonutils.NewString(tenant.GetName()))
		sharedProjects.Add(p)
	}
	if sharedProjects.Length() > 0 {
		extra.Set("shared_projects", sharedProjects)
	}
	return extra
}

func (model *SSharableVirtualResourceBase) getMoreDetailsV2(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) []apis.SharedProject {
	out := []apis.SharedProject{}
	for _, project := range model.GetSharedProjects() {
		tenant, err := TenantCacheManager.FetchTenantByIdOrName(ctx, project)
		if err != nil {
			log.Errorf("failed fetch tenant by id %s", project)
			continue
		}
		out = append(out, apis.SharedProject{Id: tenant.GetId(), Name: tenant.GetName()})
	}
	return out
}

func (model *SSharableVirtualResourceBase) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := model.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return model.getMoreDetails(ctx, userCred, query, extra)
}

func (model *SSharableVirtualResourceBase) GetExtraDetailsV2(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, out *apis.SharableVirtualResourceDetails) error {
	err := model.SVirtualResourceBase.GetExtraDetailsV2(ctx, userCred, query, &out.VirtualResourceDetails)
	if err != nil {
		return err
	}
	out.SharedProjects = model.getMoreDetailsV2(ctx, userCred, query)
	return nil
}

func (model *SSharableVirtualResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := model.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return model.getMoreDetails(ctx, userCred, query, extra), nil
}
