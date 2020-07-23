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
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

func IsObjectRbacAllowed(model IModel, userCred mcclient.TokenCredential, action string, extra ...string) error {
	return isObjectRbacAllowed(model, userCred, action, extra...)
}

func isObjectRbacAllowed(model IModel, userCred mcclient.TokenCredential, action string, extra ...string) error {
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

	scope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), action, extra...)

	if !requireScope.HigherThan(scope) {
		return nil
	}
	return httperrors.NewForbiddenError("not enough privilege (require:%s,allow:%s:resource:%s)", requireScope, scope, resScope)
}

func isJointObjectRbacAllowed(item IJointModel, userCred mcclient.TokenCredential, action string, extra ...string) error {
	err1 := isObjectRbacAllowed(JointMaster(item), userCred, action, extra...)
	err2 := isObjectRbacAllowed(JointSlave(item), userCred, action, extra...)
	if err1 == nil || err2 == nil {
		return nil
	}
	return err1
}

func isClassRbacAllowed(manager IModelManager, userCred mcclient.TokenCredential, objOwnerId mcclient.IIdentityProvider, action string, extra ...string) error {
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

	allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), action, extra...)

	if !requireScope.HigherThan(allowScope) {
		return nil
	}
	return httperrors.NewForbiddenError("not enough privilege (require:%s,allow:%s)", requireScope, allowScope)
}

type IResource interface {
	KeywordPlural() string
}

func IsAllowList(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, manager IResource) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAllow(scope, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
}

func IsAdminAllowList(userCred mcclient.TokenCredential, manager IResource) bool {
	return IsAllowList(rbacutils.ScopeSystem, userCred, manager)
}

func IsDomainAllowList(userCred mcclient.TokenCredential, manager IResource) bool {
	return IsAllowList(rbacutils.ScopeDomain, userCred, manager)
}

func IsProjectAllowList(userCred mcclient.TokenCredential, manager IResource) bool {
	return IsAllowList(rbacutils.ScopeProject, userCred, manager)
}

func IsAllowCreate(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, manager IResource) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAllow(scope, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionCreate)
}

func IsAdminAllowCreate(userCred mcclient.TokenCredential, manager IResource) bool {
	return IsAllowCreate(rbacutils.ScopeSystem, userCred, manager)
}

func IsDomainAllowCreate(userCred mcclient.TokenCredential, manager IResource) bool {
	return IsAllowCreate(rbacutils.ScopeDomain, userCred, manager)
}

func IsProjectAllowCreate(userCred mcclient.TokenCredential, manager IResource) bool {
	return IsAllowCreate(rbacutils.ScopeProject, userCred, manager)
}

func IsAllowClassPerform(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, manager IResource, action string) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAllow(scope, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionPerform, action)
}

func IsAdminAllowClassPerform(userCred mcclient.TokenCredential, manager IResource, action string) bool {
	return IsAllowClassPerform(rbacutils.ScopeSystem, userCred, manager, action)
}

func IsDomainAllowClassPerform(userCred mcclient.TokenCredential, manager IResource, action string) bool {
	return IsAllowClassPerform(rbacutils.ScopeDomain, userCred, manager, action)
}

func IsProjectAllowClassPerform(userCred mcclient.TokenCredential, manager IResource, action string) bool {
	return IsAllowClassPerform(rbacutils.ScopeProject, userCred, manager, action)
}

func IsAllowGet(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, obj IResource) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionGet)
}

func IsAdminAllowGet(userCred mcclient.TokenCredential, obj IResource) bool {
	return IsAllowGet(rbacutils.ScopeSystem, userCred, obj)
}

func IsDomainAllowGet(userCred mcclient.TokenCredential, obj IResource) bool {
	return IsAllowGet(rbacutils.ScopeDomain, userCred, obj)
}

func IsProjectAllowGet(userCred mcclient.TokenCredential, obj IResource) bool {
	return IsAllowGet(rbacutils.ScopeProject, userCred, obj)
}

func IsAllowGetSpec(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, obj IResource, spec string) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionGet, spec)
}

func IsAdminAllowGetSpec(userCred mcclient.TokenCredential, obj IResource, spec string) bool {
	return IsAllowGetSpec(rbacutils.ScopeSystem, userCred, obj, spec)
}

func IsDomainAllowGetSpec(userCred mcclient.TokenCredential, obj IResource, spec string) bool {
	return IsAllowGetSpec(rbacutils.ScopeDomain, userCred, obj, spec)
}

func IsProjectAllowGetSpec(userCred mcclient.TokenCredential, obj IResource, spec string) bool {
	return IsAllowGetSpec(rbacutils.ScopeProject, userCred, obj, spec)
}

func IsAllowPerform(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, obj IResource, action string) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionPerform, action)
}

func IsAdminAllowPerform(userCred mcclient.TokenCredential, obj IResource, action string) bool {
	return IsAllowPerform(rbacutils.ScopeSystem, userCred, obj, action)
}

func IsDomainAllowPerform(userCred mcclient.TokenCredential, obj IResource, action string) bool {
	return IsAllowPerform(rbacutils.ScopeDomain, userCred, obj, action)
}

func IsProjectAllowPerform(userCred mcclient.TokenCredential, obj IResource, action string) bool {
	return IsAllowPerform(rbacutils.ScopeProject, userCred, obj, action)
}

func IsAllowUpdate(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, obj IResource) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionUpdate)
}

func IsAdminAllowUpdate(userCred mcclient.TokenCredential, obj IResource) bool {
	return IsAllowUpdate(rbacutils.ScopeSystem, userCred, obj)
}

func IsDomainAllowUpdate(userCred mcclient.TokenCredential, obj IResource) bool {
	return IsAllowUpdate(rbacutils.ScopeDomain, userCred, obj)
}

func IsProjectAllowUpdate(userCred mcclient.TokenCredential, obj IResource) bool {
	return IsAllowUpdate(rbacutils.ScopeProject, userCred, obj)
}

func IsAllowUpdateSpec(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, obj IResource, spec string) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionUpdate, spec)
}

func IsAdminAllowUpdateSpec(userCred mcclient.TokenCredential, obj IResource, spec string) bool {
	return IsAllowUpdateSpec(rbacutils.ScopeSystem, userCred, obj, spec)
}

func IsDomainAllowUpdateSpec(userCred mcclient.TokenCredential, obj IResource, spec string) bool {
	return IsAllowUpdateSpec(rbacutils.ScopeDomain, userCred, obj, spec)
}

func IsProjectAllowUpdateSpec(userCred mcclient.TokenCredential, obj IResource, spec string) bool {
	return IsAllowUpdateSpec(rbacutils.ScopeProject, userCred, obj, spec)
}

func IsAllowDelete(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, obj IResource) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionDelete)
}

func IsAdminAllowDelete(userCred mcclient.TokenCredential, obj IResource) bool {
	return IsAllowDelete(rbacutils.ScopeSystem, userCred, obj)
}

func IsDomainAllowDelete(userCred mcclient.TokenCredential, obj IResource) bool {
	return IsAllowDelete(rbacutils.ScopeDomain, userCred, obj)
}

func IsProjectAllowDelete(userCred mcclient.TokenCredential, obj IResource) bool {
	return IsAllowDelete(rbacutils.ScopeProject, userCred, obj)
}

func IsAllowDeleteSpec(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, obj IResource, spec string) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAllow(scope, consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionDelete, spec)
}

func IsAdminAllowDeleteSpec(userCred mcclient.TokenCredential, obj IResource, spec string) bool {
	return IsAllowDeleteSpec(rbacutils.ScopeSystem, userCred, obj, spec)
}

func IsDomainAllowDeleteSpec(userCred mcclient.TokenCredential, obj IResource, spec string) bool {
	return IsAllowDeleteSpec(rbacutils.ScopeDomain, userCred, obj, spec)
}

func IsProjectAllowDeleteSpec(userCred mcclient.TokenCredential, obj IResource, spec string) bool {
	return IsAllowDeleteSpec(rbacutils.ScopeProject, userCred, obj, spec)
}
