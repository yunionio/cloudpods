package db

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
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
		isAllow = policy.PolicyManager.Allow(false, userCred, consts.GetServiceType(),
			manager.KeywordPlural(), action, extra...)
	}
	if !isAllow {
		isAllow = policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
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
		isAllow = policy.PolicyManager.Allow(false, userCred, consts.GetServiceType(),
			manager.KeywordPlural(), action, extra...)
	}
	if !isAllow {
		isAllow = policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
			manager.KeywordPlural(), action, extra...)
	}
	return isAllow
}
