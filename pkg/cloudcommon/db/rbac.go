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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

func IsObjectRbacAllowed(ctx context.Context, model IModel, userCred mcclient.TokenCredential, action string, extra ...string) error {
	return isObjectRbacAllowed(ctx, model, userCred, action, extra...)
}

func isObjectRbacAllowed(ctx context.Context, model IModel, userCred mcclient.TokenCredential, action string, extra ...string) error {
	_, err := isObjectRbacAllowedResult(ctx, model, userCred, action, extra...)
	return err
}

func isObjectRbacAllowedResult(ctx context.Context, model IModel, userCred mcclient.TokenCredential, action string, extra ...string) (rbacutils.SPolicyResult, error) {
	manager := model.GetModelManager()
	objOwnerId := model.GetOwnerId()

	var ownerId mcclient.IIdentityProvider
	if userCred != nil {
		ownerId = userCred
	}

	var requireScope rbacscope.TRbacScope
	resScope := manager.ResourceScope()
	switch resScope {
	case rbacscope.ScopeSystem:
		requireScope = rbacscope.ScopeSystem
	case rbacscope.ScopeDomain:
		if ownerId != nil && objOwnerId != nil && (ownerId.GetUserId() == objOwnerId.GetUserId() && action == policy.PolicyActionGet) {
			requireScope = rbacscope.ScopeUser
		} else if ownerId != nil && objOwnerId != nil && (ownerId.GetProjectDomainId() == objOwnerId.GetProjectDomainId() || objOwnerId.GetProjectDomainId() == "" || (model.IsSharable(ownerId) && action == policy.PolicyActionGet)) {
			requireScope = rbacscope.ScopeDomain
		} else {
			requireScope = rbacscope.ScopeSystem
		}
	case rbacscope.ScopeUser:
		if ownerId != nil && objOwnerId != nil && (ownerId.GetUserId() == objOwnerId.GetUserId() || objOwnerId.GetUserId() == "" || (model.IsSharable(ownerId) && action == policy.PolicyActionGet)) {
			requireScope = rbacscope.ScopeUser
		} else if ownerId != nil && objOwnerId != nil && ownerId.GetProjectDomainId() == objOwnerId.GetProjectDomainId() {
			requireScope = rbacscope.ScopeDomain
		} else {
			requireScope = rbacscope.ScopeSystem
		}
	default:
		// objOwnerId should not be nil
		if ownerId != nil && objOwnerId != nil && (ownerId.GetProjectId() == objOwnerId.GetProjectId() || objOwnerId.GetProjectId() == "" || (model.IsSharable(ownerId) && action == policy.PolicyActionGet)) {
			requireScope = rbacscope.ScopeProject
		} else if ownerId != nil && objOwnerId != nil && ownerId.GetProjectDomainId() == objOwnerId.GetProjectDomainId() {
			requireScope = rbacscope.ScopeDomain
		} else {
			requireScope = rbacscope.ScopeSystem
		}
	}

	scope, result := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), action, extra...)

	if result.Result.IsAllow() && !requireScope.HigherThan(scope) {
		err := objectConfirmPolicyTags(ctx, model, result)
		if err != nil {
			return rbacutils.PolicyDeny, errors.Wrap(err, "objectConfirmPolicyTags")
		}
		return result, nil
	}
	return rbacutils.PolicyDeny, httperrors.NewForbiddenError("not enough privilege (require:%s,allow:%s:resource:%s) [tags:%s]", requireScope, scope, resScope, result.String())
}

func isJointObjectRbacAllowed(ctx context.Context, item IJointModel, userCred mcclient.TokenCredential, action string, extra ...string) error {
	err1 := isObjectRbacAllowed(ctx, JointMaster(item), userCred, action, extra...)
	err2 := isObjectRbacAllowed(ctx, JointSlave(item), userCred, action, extra...)
	if err1 == nil || err2 == nil {
		return nil
	}
	return err1
}

func isClassRbacAllowed(ctx context.Context, manager IModelManager, userCred mcclient.TokenCredential, objOwnerId mcclient.IIdentityProvider, action string, extra ...string) (rbacutils.SPolicyResult, error) {
	var ownerId mcclient.IIdentityProvider
	if userCred != nil {
		ownerId = userCred
	}

	var requireScope rbacscope.TRbacScope
	resScope := manager.ResourceScope()
	switch resScope {
	case rbacscope.ScopeSystem:
		requireScope = rbacscope.ScopeSystem
	case rbacscope.ScopeDomain:
		// objOwnerId should not be nil
		if ownerId != nil && ownerId.GetProjectDomainId() == objOwnerId.GetProjectDomainId() {
			requireScope = rbacscope.ScopeDomain
		} else {
			requireScope = rbacscope.ScopeSystem
		}
	case rbacscope.ScopeUser:
		if ownerId != nil && ownerId.GetUserId() == objOwnerId.GetUserId() {
			requireScope = rbacscope.ScopeUser
		} else if ownerId != nil && ownerId.GetProjectDomainId() == objOwnerId.GetProjectDomainId() {
			requireScope = rbacscope.ScopeDomain
		} else {
			requireScope = rbacscope.ScopeSystem
		}
	default:
		// objOwnerId should not be nil
		if ownerId != nil && ownerId.GetProjectId() == objOwnerId.GetProjectId() {
			requireScope = rbacscope.ScopeProject
		} else if ownerId != nil && ownerId.GetProjectDomainId() == objOwnerId.GetProjectDomainId() {
			requireScope = rbacscope.ScopeDomain
		} else {
			requireScope = rbacscope.ScopeSystem
		}
	}

	allowScope, result := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), action, extra...)

	if result.Result.IsAllow() && !requireScope.HigherThan(allowScope) {
		err := classConfirmPolicyTags(ctx, manager, objOwnerId, result)
		if err != nil {
			return rbacutils.PolicyDeny, errors.Wrap(err, "classConfirmPolicyTags")
		}
		return result, nil
	}
	return rbacutils.PolicyDeny, httperrors.NewForbiddenError("not enough privilege (require:%s,allow:%s)", requireScope, allowScope)
}

type IResource interface {
	KeywordPlural() string
}

func IsAllowList(scope rbacscope.TRbacScope, userCred mcclient.TokenCredential, manager IResource) rbacutils.SPolicyResult {
	if userCred == nil {
		return rbacutils.PolicyDeny
	}
	return userCred.IsAllow(scope, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
}

func IsAdminAllowList(userCred mcclient.TokenCredential, manager IResource) rbacutils.SPolicyResult {
	return IsAllowList(rbacscope.ScopeSystem, userCred, manager)
}

func IsDomainAllowList(userCred mcclient.TokenCredential, manager IResource) rbacutils.SPolicyResult {
	return IsAllowList(rbacscope.ScopeDomain, userCred, manager)
}

func IsProjectAllowList(userCred mcclient.TokenCredential, manager IResource) rbacutils.SPolicyResult {
	return IsAllowList(rbacscope.ScopeProject, userCred, manager)
}

func IsAllowCreate(scope rbacscope.TRbacScope, userCred mcclient.TokenCredential, manager IResource) rbacutils.SPolicyResult {
	if userCred == nil {
		return rbacutils.PolicyDeny
	}
	return userCred.IsAllow(scope, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionCreate)
}

func IsAdminAllowCreate(userCred mcclient.TokenCredential, manager IResource) rbacutils.SPolicyResult {
	return IsAllowCreate(rbacscope.ScopeSystem, userCred, manager)
}

func IsDomainAllowCreate(userCred mcclient.TokenCredential, manager IResource) rbacutils.SPolicyResult {
	return IsAllowCreate(rbacscope.ScopeDomain, userCred, manager)
}

func IsProjectAllowCreate(userCred mcclient.TokenCredential, manager IResource) rbacutils.SPolicyResult {
	return IsAllowCreate(rbacscope.ScopeProject, userCred, manager)
}

func IsAllowClassPerform(scope rbacscope.TRbacScope, userCred mcclient.TokenCredential, manager IResource, action string) rbacutils.SPolicyResult {
	if userCred == nil {
		return rbacutils.PolicyDeny
	}
	return userCred.IsAllow(scope, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionPerform, action)
}

func IsAdminAllowClassPerform(userCred mcclient.TokenCredential, manager IResource, action string) rbacutils.SPolicyResult {
	return IsAllowClassPerform(rbacscope.ScopeSystem, userCred, manager, action)
}

func IsDomainAllowClassPerform(userCred mcclient.TokenCredential, manager IResource, action string) rbacutils.SPolicyResult {
	return IsAllowClassPerform(rbacscope.ScopeDomain, userCred, manager, action)
}

func IsProjectAllowClassPerform(userCred mcclient.TokenCredential, manager IResource, action string) rbacutils.SPolicyResult {
	return IsAllowClassPerform(rbacscope.ScopeProject, userCred, manager, action)
}

func IsAllowGet(ctx context.Context, scope rbacscope.TRbacScope, userCred mcclient.TokenCredential, obj IModel) bool {
	if userCred == nil {
		return false
	}
	result := userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionGet)
	err := objectConfirmPolicyTags(ctx, obj, result)
	if err != nil {
		log.Errorf("IsAllowGet %s", err)
		return false
	} else {
		return true
	}
}

func IsAdminAllowGet(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowGet(ctx, rbacscope.ScopeSystem, userCred, obj)
}

func IsDomainAllowGet(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowGet(ctx, rbacscope.ScopeDomain, userCred, obj)
}

func IsProjectAllowGet(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowGet(ctx, rbacscope.ScopeProject, userCred, obj)
}

func IsAllowGetSpec(ctx context.Context, scope rbacscope.TRbacScope, userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	if userCred == nil {
		return false
	}
	result := userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionGet, spec)
	err := objectConfirmPolicyTags(ctx, obj, result)
	if err != nil {
		log.Errorf("IsAllowGetSpec %s", err)
		return false
	} else {
		return true
	}
}

func IsAdminAllowGetSpec(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	return IsAllowGetSpec(ctx, rbacscope.ScopeSystem, userCred, obj, spec)
}

func IsDomainAllowGetSpec(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	return IsAllowGetSpec(ctx, rbacscope.ScopeDomain, userCred, obj, spec)
}

func IsProjectAllowGetSpec(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	return IsAllowGetSpec(ctx, rbacscope.ScopeProject, userCred, obj, spec)
}

func IsAllowPerform(ctx context.Context, scope rbacscope.TRbacScope, userCred mcclient.TokenCredential, obj IModel, action string) bool {
	if userCred == nil {
		return false
	}
	result := userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionPerform, action)
	err := objectConfirmPolicyTags(ctx, obj, result)
	if err != nil {
		log.Errorf("IsAllowPerform %s", err)
		return false
	} else {
		return true
	}
}

func IsAdminAllowPerform(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, action string) bool {
	return IsAllowPerform(ctx, rbacscope.ScopeSystem, userCred, obj, action)
}

func IsDomainAllowPerform(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, action string) bool {
	return IsAllowPerform(ctx, rbacscope.ScopeDomain, userCred, obj, action)
}

func IsProjectAllowPerform(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, action string) bool {
	return IsAllowPerform(ctx, rbacscope.ScopeProject, userCred, obj, action)
}

func IsAllowUpdate(ctx context.Context, scope rbacscope.TRbacScope, userCred mcclient.TokenCredential, obj IModel) bool {
	if userCred == nil {
		return false
	}
	result := userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionUpdate)
	err := objectConfirmPolicyTags(ctx, obj, result)
	if err != nil {
		log.Errorf("IsAllowUpdate %s", err)
		return false
	} else {
		return true
	}
}

func IsAdminAllowUpdate(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowUpdate(ctx, rbacscope.ScopeSystem, userCred, obj)
}

func IsDomainAllowUpdate(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowUpdate(ctx, rbacscope.ScopeDomain, userCred, obj)
}

func IsProjectAllowUpdate(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowUpdate(ctx, rbacscope.ScopeProject, userCred, obj)
}

func IsAllowUpdateSpec(ctx context.Context, scope rbacscope.TRbacScope, userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	if userCred == nil {
		return false
	}
	result := userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionUpdate, spec)
	err := objectConfirmPolicyTags(ctx, obj, result)
	if err != nil {
		log.Errorf("IsAllowUpdateSpec %s", err)
		return false
	} else {
		return true
	}
}

func IsAdminAllowUpdateSpec(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	return IsAllowUpdateSpec(ctx, rbacscope.ScopeSystem, userCred, obj, spec)
}

func IsDomainAllowUpdateSpec(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	return IsAllowUpdateSpec(ctx, rbacscope.ScopeDomain, userCred, obj, spec)
}

func IsProjectAllowUpdateSpec(ctx context.Context, userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	return IsAllowUpdateSpec(ctx, rbacscope.ScopeProject, userCred, obj, spec)
}

func IsAllowDelete(ctx context.Context, scope rbacscope.TRbacScope, userCred mcclient.TokenCredential, obj IModel) bool {
	if userCred == nil {
		return false
	}
	result := userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionDelete)
	err := objectConfirmPolicyTags(ctx, obj, result)
	if err != nil {
		log.Errorf("IsAllowDelete %s", err)
		return false
	} else {
		return true
	}
}

func IsAdminAllowDelete(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowDelete(ctx, rbacscope.ScopeSystem, userCred, obj)
}

func IsDomainAllowDelete(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowDelete(ctx, rbacscope.ScopeDomain, userCred, obj)
}

func IsProjectAllowDelete(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) bool {
	return IsAllowDelete(ctx, rbacscope.ScopeProject, userCred, obj)
}
