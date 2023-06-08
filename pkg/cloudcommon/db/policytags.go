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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/tagutils"
)

func objectConfirmPolicyTags(ctx context.Context, model IModel, result rbacutils.SPolicyResult) error {
	model.GetModelManager().ResourceScope()
	if _, ok := model.(IStandaloneModel); !ok {
		// a plain resource
		return nil
	}
	// now, its is a system resource

	resTagsMap, err := Metadata.rawGetAll(model.Keyword(), model.GetId(), nil, "")
	if err != nil {
		return errors.Wrap(err, "Standalone model GetAllMetadata")
	}
	resTags := tagutils.Map2Tagset(resTagsMap)
	if model.Keyword() == "domain" && !result.DomainTags.Contains(resTags) {
		return httperrors.NewNotSufficientPrivilegeError("resource (domain) tags not match (tags:%s,require:%s)", jsonutils.Marshal(resTags), jsonutils.Marshal(result.DomainTags))
	}
	if model.Keyword() == "project" && !result.ProjectTags.Contains(resTags) {
		return httperrors.NewNotSufficientPrivilegeError("resource (project) tags not match (tags:%s,require:%s)", jsonutils.Marshal(resTags), jsonutils.Marshal(result.ProjectTags))
	}
	if !result.ObjectTags.Contains(resTags) {
		return httperrors.NewNotSufficientPrivilegeError("resource tags not match (tags:%s,require:%s)", jsonutils.Marshal(resTags), jsonutils.Marshal(result.ObjectTags))
	}
	if _, ok := model.(IDomainLevelModel); !ok {
		// a system level resource
		return nil
	}
	// now, it is a domain level resource
	ownerId := model.(IDomainLevelModel).GetOwnerId()
	// domain, err := TenantCacheManager.FetchDomainById(ctx, ownerId.GetProjectDomainId())
	domain, err := DefaultDomainFetcher(ctx, ownerId.GetProjectDomainId())
	if err != nil {
		return errors.Wrap(err, "DefaultDomainFetcher")
	}
	if !result.DomainTags.Contains(domain.GetTags()) {
		return httperrors.NewNotSufficientPrivilegeError("domain tags not match (%s,require:%s)", jsonutils.Marshal(domain.GetTags()), result.DomainTags)
	}

	if _, ok := model.(IVirtualModel); !ok {
		// a domain level resource
		return nil
	}
	// now it is a virtual resource/project level resource
	ownerId = model.(IVirtualModel).GetOwnerId()
	// project, err := TenantCacheManager.FetchTenantById(ctx, ownerId.GetProjectId())
	project, err := DefaultProjectFetcher(ctx, ownerId.GetProjectId(), ownerId.GetProjectDomainId())
	if err != nil {
		return errors.Wrap(err, "DefaultProjectFetcher")
	}
	if !result.ProjectTags.Contains(project.GetTags()) {
		return httperrors.NewNotSufficientPrivilegeError("project tags not match (%s,require:%s)", jsonutils.Marshal(project.GetTags()), result.ProjectTags)
	}
	// pass all checks, all
	return nil
}

func classConfirmPolicyTags(ctx context.Context, manager IModelManager, objectOwnerId mcclient.IIdentityProvider, result rbacutils.SPolicyResult) error {
	if _, ok := manager.(IStandaloneModelManager); !ok {
		return nil
	}
	// now, the manager is a standalone model manager
	if _, ok := manager.(IDomainLevelModelManager); !ok {
		// a system level resource manager
		return nil
	}
	// now the manager is a domain level manager, we should check domain tags
	if objectOwnerId != nil && objectOwnerId.GetProjectDomainId() != "" {
		// domain, err := TenantCacheManager.FetchDomainById(ctx, objectOwnerId.GetProjectDomainId())
		domain, err := DefaultDomainFetcher(ctx, objectOwnerId.GetProjectDomainId())
		if err != nil {
			return errors.Wrap(err, "DefaultDomainFetcher")
		}
		if !result.DomainTags.Contains(domain.GetTags()) {
			return httperrors.NewNotSufficientPrivilegeError("domain tags not match (%s,require:%s)", jsonutils.Marshal(domain.GetTags()), result.DomainTags)
		}
	}
	if _, ok := manager.(IVirtualModelManager); !ok {
		// a domain level resource manager
		return nil
	}
	// now the manager is project level manager, we should check project tags
	if objectOwnerId != nil && objectOwnerId.GetProjectId() != "" {
		// project, err := TenantCacheManager.FetchTenantById(ctx, objectOwnerId.GetProjectId())
		project, err := DefaultProjectFetcher(ctx, objectOwnerId.GetProjectId(), objectOwnerId.GetProjectDomainId())
		if err != nil {
			return errors.Wrap(err, "DefaultProjectFetcher")
		}
		if !result.ProjectTags.Contains(project.GetTags()) {
			return httperrors.NewNotSufficientPrivilegeError("project tags not match (%s,require:%s)", jsonutils.Marshal(project.GetTags()), result.ProjectTags)
		}
	}
	return nil
}
