package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
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
}

type SBaremetalagent struct {
	db.SStandaloneResourceBase

	Status     string `width:"36" charset:"ascii" nullable:"false" default:"disable" list:"user" create:"optional"`
	AccessIp   string `width:"16" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_required"`
	ManagerUri string `width:"256" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_required"`
	ZoneId     string `width:"128" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_required"`
}

var BaremetalagentManager *SBaremetalagentManager

func init() {
	BaremetalagentManager = &SBaremetalagentManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(SBaremetalagent{},
			"baremetalagents_tbl",
			"baremetalagent",
			"baremetalagents",
		)}
}

func (self *SBaremetalagentManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SBaremetalagentManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SBaremetalagent) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SBaremetalagent) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SBaremetalagent) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
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
	count := manager.Query().Equals("manager_uri", mangerUri).Count()
	if count > 0 {
		return nil, httperrors.NewDuplicateResourceError("Duplicate manager_uri %s", mangerUri)
	}
	accessIp, _ := data.GetString("access_ip")
	count = manager.Query().Equals("access_ip", accessIp).Count()
	if count > 0 {
		return nil, httperrors.NewDuplicateResourceError("Duplicate access_ip %s", accessIp)
	}
	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (self *SBaremetalagent) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "enable")
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
	return db.IsAdminAllowPerform(userCred, self, "disable")
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
	return db.IsAdminAllowPerform(userCred, self, "online")
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
	return db.IsAdminAllowPerform(userCred, self, "offline")
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

func (self *SBaremetalagent) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	zone := self.GetZone()
	if zone != nil {
		extra.Set("zone", jsonutils.NewString(zone.GetName()))
	}
	return extra, nil
}
