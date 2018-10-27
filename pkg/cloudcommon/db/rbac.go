package db

import (
	"yunion.io/x/onecloud/pkg/mcclient"
)

func isRbacAllowed(manager IModelManager, model IModel, userCred mcclient.TokenCredential, action string, extra ...string) bool {
	var isAllow bool
	var isAdmin bool
	if model == nil {
		_, ok := manager.(IVirtualModelManager)
		if ok {
			isAdmin = false
		}
	} else {
		virtModel, ok := model.(IVirtualModel)
		if ok {
			if virtModel.IsOwner(userCred) {
				isAdmin = false
			} else {
				isAdmin = true
			}
		} else {
			isAdmin = true
		}
	}
	if !isAdmin {
		isAllow = PolicyManager.Allow(false, userCred, GetGlobalServiceType(),
			manager.KeywordPlural(), action, extra...)
	}
	if !isAllow {
		isAllow = PolicyManager.Allow(true, userCred, GetGlobalServiceType(),
			manager.KeywordPlural(), action, extra...)
	}
	return isAllow
}

func isJointRbacAllowed(manager IJointModelManager, item IJointModel, userCred mcclient.TokenCredential, action string, extra ...string) bool {
	isAllow := false
	isAdmin := true
	master := item.Master()
	virtualMaster, ok := master.(IVirtualModel)
	if ok && virtualMaster.IsOwner(userCred) {
		isAdmin = false
	}
	if !isAdmin {
		isAllow = PolicyManager.Allow(false, userCred, GetGlobalServiceType(),
			manager.KeywordPlural(), action, extra...)
	}
	if !isAllow {
		isAllow = PolicyManager.Allow(true, userCred, GetGlobalServiceType(),
			manager.KeywordPlural(), action, extra...)
	}
	return isAllow
}
