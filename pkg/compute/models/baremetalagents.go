package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/regutils"
)

const (
	BAREMETAL_AGENT_ENABLED  = "enabled"
	BAREMETAL_AGENT_DISABLED = "disabled"
	BAREMETAL_AGENT_OFFLINE  = "offline"
)

type SBaremetalagentManager struct {
	db.SStandaloneResourceBaseManager
	SInfrastructureManager
}

type SBaremetalagent struct {
	db.SStandaloneResourceBase
	SInfrastructure

	Status     string `width:"36" charset:"ascii" nullable:"false" default:"disable" list:"user" create:"optional"`
	AccessIp   string `width:"16" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_required"`
	ManagerUri string `width:"256" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_required"`
	ZoneId     string `width:"128" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_required"`
}

var BaremetalagentManager *SBaremetalagentManager

func init() {
	BaremetalagentManager = &SBaremetalagentManager{SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(SBaremetalagent{}, "baremetalagents_tbl", "baremetalagent", "baremetalagents")}
}

func (self *SBaremetalagent) ValidateDeleteCondition(ctx context.Context) error {
	if self.Status == BAREMETAL_AGENT_ENABLED {
		return fmt.Errorf("Cannot delete in status %s", self.Status)
	}
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SBaremetalagent) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	mangerUri, err := data.GetString("manager_uri")
	if err == nil {
		count := BaremetalagentManager.Query().Equals("manager_uri", mangerUri).
			NotEquals("id", self.Id).Count()
		if count > 0 {
			return nil, httperrors.NewConflictError("Conflict manager_uri %s", mangerUri)
		}
	}
	accessIp, err := data.GetString("access_ip")
	if err == nil {
		count := BaremetalagentManager.Query().Equals("access_ip", accessIp).
			NotEquals("id", self.Id).Count()
		if count > 0 {
			return nil, httperrors.NewConflictError("Conflict access_ip %s", accessIp)
		}
	}
	return self.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (manager *SBaremetalagentManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	mangerUri, _ := data.GetString("manager_uri")
	count := manager.TableSpec().Query().Equals("manager_uri", mangerUri).Count()
	if count > 0 {
		return nil, httperrors.NewDuplicateResourceError("Duplicate manager_uri %s", mangerUri)
	}
	accessIp, _ := data.GetString("access_ip")
	count = manager.TableSpec().Query().Equals("access_ip", accessIp).Count()
	if count > 0 {
		return nil, httperrors.NewDuplicateResourceError("Duplicate access_ip %s", accessIp)
	}
	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (self *SBaremetalagent) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), self.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "enable")
}

func (self *SBaremetalagent) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != BAREMETAL_AGENT_ENABLED {
		self.GetModelManager().TableSpec().Update(self, func() error {
			self.Status = BAREMETAL_AGENT_ENABLED
			return nil
		})
		db.OpsLog.LogEvent(self, db.ACT_ENABLE, "", userCred)
	}
	return nil, nil
}

func (self *SBaremetalagent) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), self.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "disable")
}

func (self *SBaremetalagent) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != BAREMETAL_AGENT_DISABLED {
		self.GetModelManager().TableSpec().Update(self, func() error {
			self.Status = BAREMETAL_AGENT_DISABLED
			return nil
		})
		db.OpsLog.LogEvent(self, db.ACT_DISABLE, "", userCred)
	}
	return nil, nil
}

func (self *SBaremetalagent) AllowPerformOnline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), self.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "online")
}

func (self *SBaremetalagent) PerformOnline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != BAREMETAL_AGENT_OFFLINE {
		self.GetModelManager().TableSpec().Update(self, func() error {
			self.Status = BAREMETAL_AGENT_ENABLED
			return nil
		})
		db.OpsLog.LogEvent(self, db.ACT_ONLINE, "", userCred)
	}
	return nil, nil
}

func (self *SBaremetalagent) AllowPerformOffline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), self.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "offline")
}

func (self *SBaremetalagent) PerformOffline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != BAREMETAL_AGENT_ENABLED {
		self.GetModelManager().TableSpec().Update(self, func() error {
			self.Status = BAREMETAL_AGENT_OFFLINE
			return nil
		})
		db.OpsLog.LogEvent(self, db.ACT_OFFLINE, "", userCred)
	}
	return nil, nil
}

func (self *SBaremetalagent) GetZone() *SZone {
	if len(self.ZoneId) > 0 && regutils.MatchUUIDExact(self.ZoneId) {
		return ZoneManager.FetchZoneById(self.ZoneId)
	}
	return nil
}

func (self *SBaremetalagent) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	zone := self.GetZone()
	if zone != nil {
		extra.Set("zone", jsonutils.NewString(zone.GetName()))
	}
	return extra
}
