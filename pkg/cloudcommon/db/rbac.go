package db

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

func isListRbacAllowed(manager IModelManager, userCred mcclient.TokenCredential, isAdminMode bool) bool {
	return isListRbacAllowedInternal(manager, manager.KeywordPlural(), userCred, isAdminMode)
}

func isListRbacAllowedInternal(manager IModelManager, resource string, userCred mcclient.TokenCredential, isAdminMode bool) bool {
	log.Debugf("%s %s", manager.KeywordPlural(), resource)
	var requireAdmin bool
	ownerId := manager.GetOwnerId(userCred)
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
	log.Debugf("allow list for non-admin %s %v ownerId: %s", result, requireAdmin, ownerId)
	if (result == rbacutils.OwnerAllow && !requireAdmin) || (result == rbacutils.Allow && requireAdmin) {
		return true
	}
	result = policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
		resource, policy.PolicyActionList)
	log.Debugf("allow list for admin %s %v ownerId: %s", result, requireAdmin, ownerId)
	return result == rbacutils.Allow
}

func isJointListRbacAllowed(manager IJointModelManager, userCred mcclient.TokenCredential, isAdminMode bool) bool {
	return isListRbacAllowedInternal(manager.GetMasterManager(), manager.KeywordPlural(), userCred, isAdminMode)
}

func isClassActionRbacAllowed(manager IModelManager, userCred mcclient.TokenCredential, ownerProjId string, action string, extra ...string) bool {
	var requireAdmin bool
	ownerId := manager.GetOwnerId(userCred)
	if len(ownerId) > 0 {
		if ownerProjId == ownerId {
			requireAdmin = false
		} else {
			requireAdmin = true
		}
	} else {
		requireAdmin = true
	}
	// if !requireAdmin {
	result := policy.PolicyManager.Allow(false, userCred, consts.GetServiceType(),
		manager.KeywordPlural(), action, extra...)
	if result == rbacutils.Allow || (!requireAdmin && result == rbacutils.OwnerAllow) {
		return true
	}
	// }
	result = policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
		manager.KeywordPlural(), action, extra...)
	return result == rbacutils.Allow
}

func isObjectRbacAllowed(manager IModelManager, model IModel, userCred mcclient.TokenCredential, action string, extra ...string) bool {
	var requireAdmin bool
	var isOwner bool

	ownerId := model.GetModelManager().GetOwnerId(userCred)

	if len(ownerId) > 0 {
		objOwnerId := model.GetOwnerProjectId()
		if ownerId == objOwnerId {
			isOwner = true
			requireAdmin = false
		} else {
			isOwner = false
			requireAdmin = true
		}
	} else {
		requireAdmin = true
	}

	//if !requireAdmin {
	result := policy.PolicyManager.Allow(false, userCred, consts.GetServiceType(),
		manager.KeywordPlural(), action, extra...)
	if result == rbacutils.Allow || (!requireAdmin && result == rbacutils.OwnerAllow && isOwner) {
		return true
	}
	//}
	result = policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
		manager.KeywordPlural(), action, extra...)
	return result == rbacutils.Allow
}

func isJointObjectRbacAllowed(manager IJointModelManager, item IJointModel, userCred mcclient.TokenCredential, action string, extra ...string) bool {
	return isObjectRbacAllowed(manager, item.Master(), userCred, action, extra...)
}
