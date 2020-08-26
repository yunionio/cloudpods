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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	SharedTargetProject = "project"
	SharedTargetDomain  = "domain"
)

// sharing resource between projects/domains
type SSharedResource struct {
	SResourceBase

	Id int64 `primary:"true" auto_increment:"true"`

	ResourceType string `width:"32" charset:"ascii" nullable:"false" json:"resource_type"`
	ResourceId   string `width:"128" charset:"ascii" nullable:"false" index:"true" json:"resource_id"`
	// OwnerProjectId  string `width:"128" charset:"ascii" nullable:"false" index:"true" json:"owner_project_id"`
	TargetProjectId string `width:"128" charset:"ascii" nullable:"false" index:"true" json:"target_project_id"`

	TargetType string `width:"8" charset:"ascii" default:"project" nullable:"false" json:"target_type"`
}

type SSharedResourceManager struct {
	SResourceBaseManager
}

var SharedResourceManager *SSharedResourceManager

func init() {
	SharedResourceManager = &SSharedResourceManager{
		SResourceBaseManager: NewResourceBaseManager(
			SSharedResource{},
			"shared_resources_tbl",
			"shared_resource",
			"shared_resources",
		),
	}
}

func (manager *SSharedResourceManager) CleanModelShares(ctx context.Context, userCred mcclient.TokenCredential, model ISharableBaseModel) error {
	var err error
	resScope := model.GetModelManager().ResourceScope()
	switch resScope {
	case rbacutils.ScopeProject:
		_, err = manager.shareToTarget(ctx, userCred, model, SharedTargetProject, nil, nil, nil)
		if err != nil {
			return errors.Wrap(err, "remove shared project")
		}
		_, err = manager.shareToTarget(ctx, userCred, model, SharedTargetDomain, nil, nil, nil)
		if err != nil {
			return errors.Wrap(err, "remove shared domain")
		}
	case rbacutils.ScopeDomain:
		_, err = manager.shareToTarget(ctx, userCred, model, SharedTargetDomain, nil, nil, nil)
		if err != nil {
			return errors.Wrap(err, "remove shared domain")
		}
	}
	return nil
}

func (manager *SSharedResourceManager) shareToTarget(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	model ISharableBaseModel,
	targetType string,
	targetIds []string,
	candidateIds []string,
	requireDomainIds []string,
) ([]string, error) {
	var requireScope rbacutils.TRbacScope
	resScope := model.GetModelManager().ResourceScope()
	switch resScope {
	case rbacutils.ScopeProject:
		switch targetType {
		case SharedTargetProject:
			// should have domain-level privileges
			// cannot share to a project across domain
			requireScope = rbacutils.ScopeDomain
		case SharedTargetDomain:
			// should have system-level privileges
			requireScope = rbacutils.ScopeSystem
		}
	case rbacutils.ScopeDomain:
		switch targetType {
		case SharedTargetDomain:
			// should have system-level privileges
			requireScope = rbacutils.ScopeSystem
		case SharedTargetProject:
			if len(targetIds) > 0 {
				return nil, errors.Wrap(httperrors.ErrNotSupported, "cannot share a domain resource to specific project")
			}
		}
	default:
		return nil, errors.Wrap(httperrors.ErrNotSupported, "cannot share a non-project/domain resource")
	}

	srs := make([]SSharedResource, 0)
	q := SharedResourceManager.Query()
	q = q.Equals("resource_type", model.Keyword())
	q = q.Equals("resource_id", model.GetId())
	q = q.Equals("target_type", targetType)
	err := FetchModelObjects(SharedResourceManager, q, &srs)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrap(err, "Fetch shared project")
	}
	srsMap := make(map[string]*SSharedResource)
	_existIds := make([]string, len(srs))
	for i := 0; i < len(srs); i++ {
		_existIds[i] = srs[i].TargetProjectId
		srsMap[srs[i].TargetProjectId] = &srs[i]
	}
	existIds := stringutils2.NewSortedStrings(_existIds)

	newIds := stringutils2.NewSortedStrings([]string{})
	modelOwnerId := model.GetOwnerId()
	for i := 0; i < len(targetIds); i++ {
		switch targetType {
		case SharedTargetProject:
			tenant, err := DefaultProjectFetcher(ctx, targetIds[i])
			if err != nil {
				return nil, errors.Wrapf(err, "fetch tenant %s error", targetIds[i])
			}
			if tenant.DomainId != modelOwnerId.GetProjectDomainId() {
				return nil, errors.Wrap(httperrors.ErrBadRequest, "can't shared project to other domain")
			}
			if tenant.GetId() == modelOwnerId.GetProjectId() {
				// ignore self project
				continue
				// return nil, errors.Wrap(httperrors.ErrBadRequest, "can't share to self project")
			}
			newIds = stringutils2.Append(newIds, tenant.GetId())
		case SharedTargetDomain:
			domain, err := DefaultDomainFetcher(ctx, targetIds[i])
			if err != nil {
				return nil, errors.Wrapf(err, "fetch domain %s error", targetIds[i])
			}
			if domain.GetId() == modelOwnerId.GetProjectDomainId() {
				// ignore self domain
				continue
				// return nil, errors.Wrapf(httperrors.ErrBadRequest, "can't share to self domain %s", modelOwnerId.GetProjectDomainId())
			}
			if len(candidateIds) > 0 && !utils.IsInStringArray(domain.GetId(), candidateIds) {
				return nil, errors.Wrapf(httperrors.ErrForbidden, "share target domain %s not in candidate list %s", domain.GetId(), candidateIds)
			}
			newIds = stringutils2.Append(newIds, domain.GetId())
		}
	}
	delIds, keepIds, addIds := stringutils2.Split(existIds, newIds)

	if len(delIds) == 0 && len(addIds) == 0 {
		return keepIds, nil
	}

	if targetType == SharedTargetDomain && len(requireDomainIds) > 0 && len(delIds) > 0 {
		for _, delId := range delIds {
			if utils.IsInStringArray(delId, requireDomainIds) {
				return nil, errors.Wrapf(httperrors.ErrForbidden, "domain %s is required to share", delId)
			}
		}
	}

	allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), model.KeywordPlural(), policy.PolicyActionPerform, "public")
	if requireScope.HigherThan(allowScope) {
		return nil, errors.Wrapf(httperrors.ErrNotSufficientPrivilege, "require %s allow %s", requireScope, allowScope)
	}

	for _, targetId := range delIds {
		sr := srsMap[targetId]
		if err := sr.Delete(ctx, userCred); err != nil {
			return nil, errors.Wrap(err, "delete")
		}
	}
	for _, targetId := range addIds {
		sharedResource := new(SSharedResource)
		sharedResource.ResourceType = model.Keyword()
		sharedResource.ResourceId = model.GetId()
		sharedResource.TargetProjectId = targetId
		sharedResource.TargetType = targetType
		if insetErr := SharedResourceManager.TableSpec().Insert(ctx, sharedResource); insetErr != nil {
			return nil, httperrors.NewInternalServerError("Insert shared resource failed %s", insetErr)
		}
	}

	keepIds = append(keepIds, addIds...)
	return keepIds, nil
}
