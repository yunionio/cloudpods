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
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

func isListRbacAllowed(manager IModelManager, userCred mcclient.TokenCredential, isAdminMode bool) bool {
	return isListRbacAllowedInternal(manager, manager.KeywordPlural(), userCred, isAdminMode)
}

func isListRbacAllowedInternal(manager IModelManager, resource string, userCred mcclient.TokenCredential, isAdminMode bool) bool {
	var requireAdmin bool
	var ownerId string
	if userCred != nil {
		ownerId = manager.GetOwnerId(userCred)
	}
	if len(ownerId) > 0 {
		if isAdminMode {
			requireAdmin = true
		} else {
			requireAdmin = false
		}
	} else {
		requireAdmin = true
	}
	result := policy.PolicyManager.Allow(false, userCred, consts.GetServiceType(),
		resource, policy.PolicyActionList)

	switch {
	case result == rbacutils.GuestAllow:
		return true
	case result == rbacutils.UserAllow && userCred != nil && userCred.IsValid():
		return true
	case result == rbacutils.OwnerAllow && !requireAdmin:
		return true
	}

	result = policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
		resource, policy.PolicyActionList)

	return result == rbacutils.AdminAllow
}

func isJointListRbacAllowed(manager IJointModelManager, userCred mcclient.TokenCredential, isAdminMode bool) bool {
	return isListRbacAllowedInternal(manager.GetMasterManager(), manager.KeywordPlural(), userCred, isAdminMode)
}

func isClassActionRbacAllowed(manager IModelManager, userCred mcclient.TokenCredential, ownerProjId string, action string, extra ...string) bool {
	var requireAdmin bool
	var ownerId string
	if userCred != nil {
		ownerId = manager.GetOwnerId(userCred)
	}
	if len(ownerId) > 0 {
		if ownerProjId == ownerId {
			requireAdmin = false
		} else {
			requireAdmin = true
		}
	} else {
		requireAdmin = true
	}

	result := policy.PolicyManager.Allow(false, userCred, consts.GetServiceType(),
		manager.KeywordPlural(), action, extra...)
	switch {
	case result == rbacutils.GuestAllow:
		return true
	case result == rbacutils.UserAllow && userCred != nil && userCred.IsValid():
		return true
	case result == rbacutils.OwnerAllow && !requireAdmin:
		return true
	}

	result = policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
		manager.KeywordPlural(), action, extra...)
	return result == rbacutils.AdminAllow
}

func isObjectRbacAllowed(manager IModelManager, model IModel, userCred mcclient.TokenCredential, action string, extra ...string) bool {
	var requireAdmin bool
	var isOwner bool

	var ownerId string

	if userCred != nil {
		ownerId = model.GetModelManager().GetOwnerId(userCred)
	}

	if len(ownerId) > 0 {
		objOwnerId := model.GetOwnerProjectId()
		if ownerId == objOwnerId || (model.IsSharable() && action == policy.PolicyActionGet) {
			isOwner = true
			requireAdmin = false
		} else {
			isOwner = false
			requireAdmin = true
		}
	} else {
		requireAdmin = true
	}

	result := policy.PolicyManager.Allow(false, userCred, consts.GetServiceType(),
		manager.KeywordPlural(), action, extra...)
	switch {
	case result == rbacutils.GuestAllow:
		return true
	case result == rbacutils.UserAllow && userCred != nil && userCred.IsValid():
		return true
	case result == rbacutils.OwnerAllow && isOwner && !requireAdmin:
		return true
	}

	result = policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
		manager.KeywordPlural(), action, extra...)
	return result == rbacutils.AdminAllow
}

func isJointObjectRbacAllowed(manager IJointModelManager, item IJointModel, userCred mcclient.TokenCredential, action string, extra ...string) bool {
	return isObjectRbacAllowed(manager, item.Master(), userCred, action, extra...)
}

func IsAdminAllowList(userCred mcclient.TokenCredential, manager IModelManager) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAdminAllow(consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
}

func IsAdminAllowCreate(userCred mcclient.TokenCredential, manager IModelManager) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAdminAllow(consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionCreate)
}

func IsAdminAllowClassPerform(userCred mcclient.TokenCredential, manager IModelManager, action string) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAdminAllow(consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionPerform, action)
}

func IsAdminAllowGet(userCred mcclient.TokenCredential, obj IModel) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAdminAllow(consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionGet)
}

func IsAdminAllowGetSpec(userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAdminAllow(consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionGet, spec)
}

func IsAdminAllowPerform(userCred mcclient.TokenCredential, obj IModel, action string) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAdminAllow(consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionPerform, action)
}

func IsAdminAllowUpdate(userCred mcclient.TokenCredential, obj IModel) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAdminAllow(consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionUpdate)
}

func IsAdminAllowUpdateSpec(userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAdminAllow(consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionUpdate, spec)
}

func IsAdminAllowDelete(userCred mcclient.TokenCredential, obj IModel) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAdminAllow(consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionDelete)
}

func IsAdminAllowDeleteSpec(userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	if userCred == nil {
		return false
	}
	return userCred.IsAdminAllow(consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionDelete, spec)
}
