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

	"yunion.io/x/onecloud/pkg/util/tagutils"

	"yunion.io/x/sqlchemy"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

func objectConfirmPolicyTags(ctx context.Context, userCred mcclient.TokenCredential, model IModel, result rbacutils.SPolicyResult) error {
	if _, ok := model.(IStandaloneModel); !ok {
		// a plain resource
		return nil
	}
	// now, its is a system resource
	resTags, err := model.(IStandaloneModel).GetAllMetadata(userCred)
	if err != nil {
		return errors.Wrap(err, "Standalone model GetAllMetadata")
	}
	if !result.ObjectTags.Contains(tagutils.Map2Tagset(resTags)) {
		return httperrors.NewNotSufficientPrivilegeError("resource tags not match (%s,require:%s)", jsonutils.Marshal(resTags), result.ObjectTags)
	}
	if _, ok := model.(IDomainLevelModel); !ok {
		// a system level resource
		return nil
	}
	// now, it is a domain level resource
	ownerId := model.(IDomainLevelModel).GetOwnerId()
	domain, err := TenantCacheManager.FetchDomainById(ctx, ownerId.GetProjectDomainId())
	if err != nil {
		return errors.Wrap(err, "TenantCacheManager.FetchDomainById")
	}
	if !result.DomainTags.Contains(tagutils.Map2Tagset(domain.GetTags())) {
		return httperrors.NewNotSufficientPrivilegeError("domain tags not match (%s,require:%s)", jsonutils.Marshal(domain.GetTags()), result.DomainTags)
	}

	if _, ok := model.(IVirtualModel); !ok {
		// a domain level resource
		return nil
	}
	// now it is a virtual resource/project level resource
	ownerId = model.(IVirtualModel).GetOwnerId()
	project, err := TenantCacheManager.FetchTenantById(ctx, ownerId.GetProjectId())
	if err != nil {
		return errors.Wrap(err, "TenantCacheManager.FetchTenantById")
	}
	if !result.ProjectTags.Contains(tagutils.Map2Tagset(project.GetTags())) {
		return httperrors.NewNotSufficientPrivilegeError("project tags not match (%s,require:%s)", jsonutils.Marshal(project.GetTags()), result.ProjectTags)
	}
	// pass all checks, all
	return nil
}

func classConfirmPolicyTags(ctx context.Context, userCred mcclient.TokenCredential, manager IModelManager, objectOwnerId mcclient.IIdentityProvider, result rbacutils.SPolicyResult) (tagutils.TTagSet, error) {
	if _, ok := manager.(IStandaloneModelManager); !ok {
		return nil, nil
	}
	// now, the manager is a standalone model manager
	requireResourceTags := result.ObjectTags.Flattern()
	if _, ok := manager.(IDomainLevelModelManager); !ok {
		// a system level resource manager
		return requireResourceTags, nil
	}
	// now the manager is a domain level manager, we should check domain tags
	if objectOwnerId != nil && objectOwnerId.GetProjectDomainId() != "" {
		domain, err := TenantCacheManager.FetchDomainById(ctx, objectOwnerId.GetProjectDomainId())
		if err != nil {
			return nil, errors.Wrap(err, "TenantCacheManager.FetchDomainById")
		}
		if !result.DomainTags.Contains(tagutils.Map2Tagset(domain.GetTags())) {
			return nil, httperrors.NewNotSufficientPrivilegeError("domain tags not match (%s,require:%s)", jsonutils.Marshal(domain.GetTags()), result.DomainTags)
		}
	}
	if _, ok := manager.(IVirtualModelManager); !ok {
		// a domain level resource manager
		return requireResourceTags, nil
	}
	// now the manager is project level manager, we should check project tags
	if objectOwnerId != nil && objectOwnerId.GetProjectId() != "" {
		project, err := TenantCacheManager.FetchTenantById(ctx, objectOwnerId.GetProjectId())
		if err != nil {
			return nil, errors.Wrap(err, "TenantCacheManager.FetchTenantById")
		}
		if !result.ProjectTags.Contains(tagutils.Map2Tagset(project.GetTags())) {
			return nil, httperrors.NewNotSufficientPrivilegeError("project tags not match (%s,require:%s)", jsonutils.Marshal(project.GetTags()), result.ProjectTags)
		}
	}
	return requireResourceTags, nil
}

func filterByTagFilters(q *sqlchemy.SQuery, result rbacutils.SPolicyResult) *sqlchemy.SQuery {
	// to do
	return q
}
