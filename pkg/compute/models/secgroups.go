package models

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"
)

type SSecurityGroupManager struct {
	db.SSharableVirtualResourceBaseManager
}

var SecurityGroupManager *SSecurityGroupManager

func init() {
	SecurityGroupManager = &SSecurityGroupManager{SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(SSecurityGroup{}, "secgroups_tbl", "secgroup", "secgroups")}
	SecurityGroupManager.NameRequireAscii = false
}

const (
	SECURITY_GROUP_SEPARATOR = ";"
)

type SSecurityGroup struct {
	db.SSharableVirtualResourceBase
	IsDirty bool `nullable:"false" default:"false"` // Column(Boolean, nullable=False, default=False)
}

func (self *SSecurityGroup) GetGuestsQuery() *sqlchemy.SQuery {
	guests := GuestManager.Query().SubQuery()
	return guests.Query().Filter(sqlchemy.OR(sqlchemy.Equals(guests.Field("secgrp_id"), self.Id),
		sqlchemy.Equals(guests.Field("admin_secgrp_id"), self.Id)))
}

func (self *SSecurityGroup) GetGuestsCount() int {
	return self.GetGuestsQuery().Count()
}

func (self *SSecurityGroup) GetGuests() []SGuest {
	guests := make([]SGuest, 0)
	q := self.GetGuestsQuery()
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("GetGuests fail %s", err)
		return nil
	}
	return guests
}

func (self *SSecurityGroup) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SSharableVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	extra.Add(jsonutils.NewInt(int64(len(self.GetGuests()))), "guest_cnt")
	extra.Add(jsonutils.NewString(self.getSecurityRuleString("")), "rules")
	return extra
}

func (self *SSecurityGroup) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SSharableVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	extra.Add(jsonutils.NewInt(int64(len(self.GetGuests()))), "guest_cnt")
	extra.Add(jsonutils.NewTimeString(self.CreatedAt), "created_at")
	extra.Add(jsonutils.NewString(self.Description), "description")
	extra.Add(jsonutils.NewString(self.getSecurityRuleString("in")), "in_rules")
	extra.Add(jsonutils.NewString(self.getSecurityRuleString("out")), "out_rules")
	return extra
}

func (manager *SSecurityGroupManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerProjId string,
	query jsonutils.JSONObject,
	data *jsonutils.JSONDict,
) (*jsonutils.JSONDict, error) {
	// TODO: check set pending quota
	return manager.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (manager *SSecurityGroupManager) FetchSecgroupById(secId string) *SSecurityGroup {
	if len(secId) > 0 {
		secgrp, _ := manager.FetchById(secId)
		if secgrp != nil {
			return secgrp.(*SSecurityGroup)
		}
	}
	return nil
}

func (self *SSecurityGroup) getSecurityRules(direction string) (rules []SSecurityGroupRule) {
	secgrouprules := SecurityGroupRuleManager.Query().SubQuery()
	sql := secgrouprules.Query().Filter(sqlchemy.Equals(secgrouprules.Field("secgroup_id"), self.Id)).Desc("priority")
	if len(direction) > 0 && utils.IsInStringArray(direction, []string{"in", "out"}) {
		sql = sql.Equals("direction", direction)
	}
	if err := db.FetchModelObjects(SecurityGroupRuleManager, sql, &rules); err != nil {
		log.Errorf("GetGuests fail %s", err)
		return nil
	}
	return
}

func (self *SSecurityGroup) getSecurityRuleString(direction string) string {
	secgrouprules := self.getSecurityRules(direction)
	var rules []string
	for _, rule := range secgrouprules {
		rules = append(rules, rule.GetRule())
	}
	return strings.Join(rules, SECURITY_GROUP_SEPARATOR)
}

func totalSecurityGroupCount(projectId string) int {
	q := SecurityGroupManager.Query().Equals("tenant_id", projectId)
	return q.Count()
}

func (self *SSecurityGroup) AllowPerformClone(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (self *SSecurityGroup) PerformClone(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if name, _ := data.GetString("name"); len(name) == 0 {
		return nil, httperrors.NewInputParameterError("Missing name params")
	} else {
		sql := SecurityGroupManager.Query()
		sql = SecurityGroupManager.FilterByName(sql, name)
		if sql.Count() != 0 {
			return nil, httperrors.NewDuplicateNameError("Dumplicate name %s", name)
		}
	}

	secgroup := &SSecurityGroup{}
	secgroup.SetModelManager(SecurityGroupManager)

	secgroup.Name, _ = data.GetString("name")
	secgroup.Description, _ = data.GetString("description")
	secgroup.ProjectId = userCred.GetTenantId()
	if err := SecurityGroupManager.TableSpec().Insert(secgroup); err != nil {
		return nil, err
		//db.OpsLog.LogCloneEvent(self, secgroup, userCred, nil)
	}
	secgrouprules := self.getSecurityRules("")
	for _, rule := range secgrouprules {
		secgrouprule := &SSecurityGroupRule{}
		secgrouprule.SetModelManager(SecurityGroupRuleManager)

		secgrouprule.Priority = rule.Priority
		secgrouprule.Protocol = rule.Protocol
		secgrouprule.Ports = rule.Ports
		secgrouprule.Direction = rule.Direction
		secgrouprule.CIDR = rule.CIDR
		secgrouprule.Action = rule.Action
		secgrouprule.Description = rule.Description
		secgrouprule.SecgroupID = secgroup.Id
		if err := SecurityGroupRuleManager.TableSpec().Insert(secgrouprule); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (self *SSecurityGroup) DoSync() {
	if _, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.IsDirty = true
		return nil
	}); err != nil {
		log.Errorf("Update Security Group error: %s", err.Error())
	}
}
