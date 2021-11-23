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

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/tagutils"
)

func IsObjectRbacAllowed(ctx context.Context, model IModel, userCred mcclient.TokenCredential, action string, extra ...string) error {
	return isObjectRbacAllowed(ctx, model, userCred, action, extra...)
}

func isObjectRbacAllowed(ctx context.Context, model IModel, userCred mcclient.TokenCredential, action string, extra ...string) error {
	manager := model.GetModelManager()
	objOwnerId := model.GetOwnerId()

	var ownerId mcclient.IIdentityProvider
	if userCred != nil {
		ownerId = userCred
	}

	var requireScope rbacutils.TRbacScope
	resScope := manager.ResourceScope()
	switch resScope {
	case rbacutils.ScopeSystem:
		requireScope = rbacutils.ScopeSystem
	case rbacutils.ScopeDomain:
		if ownerId != nil && objOwnerId != nil && (ownerId.GetUserId() == objOwnerId.GetUserId() && action == policy.PolicyActionGet) {
			requireScope = rbacutils.ScopeUser
		} else if ownerId != nil && objOwnerId != nil && (ownerId.GetProjectDomainId() == objOwnerId.GetProjectDomainId() || objOwnerId.GetProjectDomainId() == "" || (model.IsSharable(ownerId) && action == policy.PolicyActionGet)) {
			requireScope = rbacutils.ScopeDomain
		} else {
			requireScope = rbacutils.ScopeSystem
		}
	case rbacutils.ScopeUser:
		if ownerId != nil && objOwnerId != nil && (ownerId.GetUserId() == objOwnerId.GetUserId() || objOwnerId.GetUserId() == "" || (model.IsSharable(ownerId) && action == policy.PolicyActionGet)) {
			requireScope = rbacutils.ScopeUser
		} else if ownerId != nil && objOwnerId != nil && ownerId.GetProjectDomainId() == objOwnerId.GetProjectDomainId() {
			requireScope = rbacutils.ScopeDomain
		} else {
			requireScope = rbacutils.ScopeSystem
		}
	default:
		// objOwnerId should not be nil
		if ownerId != nil && objOwnerId != nil && (ownerId.GetProjectId() == objOwnerId.GetProjectId() || objOwnerId.GetProjectId() == "" || (model.IsSharable(ownerId) && action == policy.PolicyActionGet)) {
			requireScope = rbacutils.ScopeProject
		} else if ownerId != nil && objOwnerId != nil && ownerId.GetProjectDomainId() == objOwnerId.GetProjectDomainId() {
			requireScope = rbacutils.ScopeDomain
		} else {
			requireScope = rbacutils.ScopeSystem
		}
	}

	scope, result := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), action, extra...)

	if !requireScope.HigherThan(scope) {
		return objectConfirmPolicyTags(ctx, userCred, model, result)
	}
	return httperrors.NewForbiddenError("not enough privilege (require:%s,allow:%s:resource:%s) [tags:%s]", requireScope, scope, resScope, result.String())
}

func isJointObjectRbacAllowed(ctx context.Context, item IJointModel, userCred mcclient.TokenCredential, action string, extra ...string) error {
	err1 := isObjectRbacAllowed(ctx, JointMaster(item), userCred, action, extra...)
	err2 := isObjectRbacAllowed(ctx, JointSlave(item), userCred, action, extra...)
	if err1 == nil || err2 == nil {
		return nil
	}
	return err1
}

func isClassRbacAllowed(ctx context.Context, manager IModelManager, userCred mcclient.TokenCredential, objOwnerId mcclient.IIdentityProvider, action string, extra ...string) (tagutils.TTagSet, error) {
	var ownerId mcclient.IIdentityProvider
	if userCred != nil {
		ownerId = userCred
	}

	var requireScope rbacutils.TRbacScope
	resScope := manager.ResourceScope()
	switch resScope {
	case rbacutils.ScopeSystem:
		requireScope = rbacutils.ScopeSystem
	case rbacutils.ScopeDomain:
		// objOwnerId should not be nil
		if ownerId != nil && ownerId.GetProjectDomainId() == objOwnerId.GetProjectDomainId() {
			requireScope = rbacutils.ScopeDomain
		} else {
			requireScope = rbacutils.ScopeSystem
		}
	case rbacutils.ScopeUser:
		if ownerId != nil && ownerId.GetUserId() == objOwnerId.GetUserId() {
			requireScope = rbacutils.ScopeUser
		} else if ownerId != nil && ownerId.GetProjectDomainId() == objOwnerId.GetProjectDomainId() {
			requireScope = rbacutils.ScopeDomain
		} else {
			requireScope = rbacutils.ScopeSystem
		}
	default:
		// objOwnerId should not be nil
		if ownerId != nil && ownerId.GetProjectId() == objOwnerId.GetProjectId() {
			requireScope = rbacutils.ScopeProject
		} else if ownerId != nil && ownerId.GetProjectDomainId() == objOwnerId.GetProjectDomainId() {
			requireScope = rbacutils.ScopeDomain
		} else {
			requireScope = rbacutils.ScopeSystem
		}
	}

	allowScope, result := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), action, extra...)

	if !requireScope.HigherThan(allowScope) {
		return classConfirmPolicyTags(ctx, userCred, manager, objOwnerId, result)
	}
	return nil, httperrors.NewForbiddenError("not enough privilege (require:%s,allow:%s)", requireScope, allowScope)
}

type IResource interface {
	KeywordPlural() string
}

func IsAllowList(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, manager IResource) rbacutils.SPolicyResult {
	if userCred == nil {
		return rbacutils.PolicyDeny
	}
	return userCred.IsAllow(scope, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
}

func IsAdminAllowList(userCred mcclient.TokenCredential, manager IResource) rbacutils.SPolicyResult {
	return IsAllowList(rbacutils.ScopeSystem, userCred, manager)
}

func IsDomainAllowList(userCred mcclient.TokenCredential, manager IResource) rbacutils.SPolicyResult {
	return IsAllowList(rbacutils.ScopeDomain, userCred, manager)
}

func IsProjectAllowList(userCred mcclient.TokenCredential, manager IResource) rbacutils.SPolicyResult {
	return IsAllowList(rbacutils.ScopeProject, userCred, manager)
}

func IsAllowCreate(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, manager IResource) rbacutils.SPolicyResult {
	if userCred == nil {
		return rbacutils.PolicyDeny
	}
	return userCred.IsAllow(scope, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionCreate)
}

func IsAdminAllowCreate(userCred mcclient.TokenCredential, manager IResource) rbacutils.SPolicyResult {
	return IsAllowCreate(rbacutils.ScopeSystem, userCred, manager)
}

func IsDomainAllowCreate(userCred mcclient.TokenCredential, manager IResource) rbacutils.SPolicyResult {
	return IsAllowCreate(rbacutils.ScopeDomain, userCred, manager)
}

func IsProjectAllowCreate(userCred mcclient.TokenCredential, manager IResource) rbacutils.SPolicyResult {
	return IsAllowCreate(rbacutils.ScopeProject, userCred, manager)
}

func IsAllowClassPerform(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, manager IResource, action string) rbacutils.SPolicyResult {
	if userCred == nil {
		return rbacutils.PolicyDeny
	}
	return userCred.IsAllow(scope, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionPerform, action)
}

func IsAdminAllowClassPerform(userCred mcclient.TokenCredential, manager IResource, action string) rbacutils.SPolicyResult {
	return IsAllowClassPerform(rbacutils.ScopeSystem, userCred, manager, action)
}

func IsDomainAllowClassPerform(userCred mcclient.TokenCredential, manager IResource, action string) rbacutils.SPolicyResult {
	return IsAllowClassPerform(rbacutils.ScopeDomain, userCred, manager, action)
}

func IsProjectAllowClassPerform(userCred mcclient.TokenCredential, manager IResource, action string) rbacutils.SPolicyResult {
	return IsAllowClassPerform(rbacutils.ScopeProject, userCred, manager, action)
}

func IsAllowGet(ctx context.Context, scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, obj IModel) bool {
	if userCred == nil {
		return false
	}
	result := userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionGet)
	err := objectConfirmPolicyTags(ctx, userCred, obj, result)
	if err != nil {
		log.Errorf("IsAllowGet %s", err)
		return false
	} else {
		return true
	}
}

func IsAdminAllowGet(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowGet(ctx, rbacutils.ScopeSystem, userCred, obj)
}

func IsDomainAllowGet(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowGet(ctx, rbacutils.ScopeDomain, userCred, obj)
}

func IsProjectAllowGet(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowGet(ctx, rbacutils.ScopeProject, userCred, obj)
}

func IsAllowGetSpec(ctx context.Context, scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	if userCred == nil {
		return false
	}
	result := userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionGet, spec)
	err := objectConfirmPolicyTags(ctx, userCred, obj, result)
	if err != nil {
		log.Errorf("IsAllowGetSpec %s", err)
		return false
	} else {
		return true
	}
}

func IsAdminAllowGetSpec(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	return IsAllowGetSpec(ctx, rbacutils.ScopeSystem, userCred, obj, spec)
}

func IsDomainAllowGetSpec(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	return IsAllowGetSpec(ctx, rbacutils.ScopeDomain, userCred, obj, spec)
}

func IsProjectAllowGetSpec(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	return IsAllowGetSpec(ctx, rbacutils.ScopeProject, userCred, obj, spec)
}

func IsAllowPerform(ctx context.Context, scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, obj IModel, action string) bool {
	if userCred == nil {
		return false
	}
	result := userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionPerform, action)
	err := objectConfirmPolicyTags(ctx, userCred, obj, result)
	if err != nil {
		log.Errorf("IsAllowPerform %s", err)
		return false
	} else {
		return true
	}
}

func IsAdminAllowPerform(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, action string) bool {
	return IsAllowPerform(ctx, rbacutils.ScopeSystem, userCred, obj, action)
}

func IsDomainAllowPerform(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, action string) bool {
	return IsAllowPerform(ctx, rbacutils.ScopeDomain, userCred, obj, action)
}

func IsProjectAllowPerform(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, action string) bool {
	return IsAllowPerform(ctx, rbacutils.ScopeProject, userCred, obj, action)
}

func IsAllowUpdate(ctx context.Context, scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, obj IModel) bool {
	if userCred == nil {
		return false
	}
	result := userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionUpdate)
	err := objectConfirmPolicyTags(ctx, userCred, obj, result)
	if err != nil {
		log.Errorf("IsAllowUpdate %s", err)
		return false
	} else {
		return true
	}
}

func IsAdminAllowUpdate(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowUpdate(ctx, rbacutils.ScopeSystem, userCred, obj)
}

func IsDomainAllowUpdate(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowUpdate(ctx, rbacutils.ScopeDomain, userCred, obj)
}

func IsProjectAllowUpdate(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowUpdate(ctx, rbacutils.ScopeProject, userCred, obj)
}

func IsAllowUpdateSpec(ctx context.Context, scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	if userCred == nil {
		return false
	}
	result := userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionUpdate, spec)
	err := objectConfirmPolicyTags(ctx, userCred, obj, result)
	if err != nil {
		log.Errorf("IsAllowUpdateSpec %s", err)
		return false
	} else {
		return true
	}
}

func IsAdminAllowUpdateSpec(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	return IsAllowUpdateSpec(ctx, rbacutils.ScopeSystem, userCred, obj, spec)
}

func IsDomainAllowUpdateSpec(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	return IsAllowUpdateSpec(ctx, rbacutils.ScopeDomain, userCred, obj, spec)
}

func IsProjectAllowUpdateSpec(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	return IsAllowUpdateSpec(ctx, rbacutils.ScopeProject, userCred, obj, spec)
}

func IsAllowDelete(ctx context.Context, scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, obj IModel) bool {
	if userCred == nil {
		return false
	}
	result := userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionDelete)
	err := objectConfirmPolicyTags(ctx, userCred, obj, result)
	if err != nil {
		log.Errorf("IsAllowDelete %s", err)
		return false
	} else {
		return true
	}
}

func IsAdminAllowDelete(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowDelete(ctx, rbacutils.ScopeSystem, userCred, obj)
}

func IsDomainAllowDelete(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowDelete(ctx, rbacutils.ScopeDomain, userCred, obj)
}

func IsProjectAllowDelete(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowDelete(ctx, rbacutils.ScopeProject, userCred, obj)
}
