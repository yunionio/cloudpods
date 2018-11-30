package models

import (
	"context"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SInfrastructureManager struct {
}

type SInfrastructure struct {
}

func (self *SInfrastructureManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), "*", policy.PolicyActionList)
}

func (self *SInfrastructureManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), "*", policy.PolicyActionCreate)
}

func (self *SInfrastructure) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), "*", policy.PolicyActionGet)
}

func (self *SInfrastructure) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), "*", policy.PolicyActionUpdate)
}

func (self *SInfrastructure) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), "*", policy.PolicyActionDelete)
}
