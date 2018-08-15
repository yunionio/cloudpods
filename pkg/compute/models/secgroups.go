package models

import (
	"context"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/secrules"
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
	extra.Add(jsonutils.NewString(self.getSecurityRuleString()), "rules")
	return extra
}

func (self *SSecurityGroup) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SSharableVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	extra.Add(jsonutils.NewInt(int64(len(self.GetGuests()))), "guest_cnt")
	extra.Add(jsonutils.NewTimeString(self.CreatedAt), "created_at")
	extra.Add(jsonutils.NewString(self.Description), "description")
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

func (self *SSecurityGroup) getSecurityRules() (rules []SSecurityGroupRule) {
	secgrouprules := SecurityGroupRuleManager.Query().SubQuery()
	sql := secgrouprules.Query().Filter(sqlchemy.Equals(secgrouprules.Field("secgroup_id"), self.Id))
	if err := db.FetchModelObjects(SecurityGroupRuleManager, sql, &rules); err != nil {
		log.Errorf("GetGuests fail %s", err)
		return
	}
	return
}

func (self *SSecurityGroup) getSecRules() []secrules.SecurityRule {
	rules := make([]secrules.SecurityRule, 0)
	for _, _rule := range self.getSecurityRules() {
		singleRules, err := _rule.SingleRules()
		if err != nil {
			log.Errorf(err.Error())
		}
		rules = append(rules, singleRules...)
	}
	return rules
}

func (self *SSecurityGroup) getSecurityRuleString() string {
	secgrouprules := self.getSecurityRules()
	var rules []string
	for _, rule := range secgrouprules {
		rules = append(rules, rule.String())
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
	secgrouprules := self.getSecurityRules()
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

func (manager *SSecurityGroupManager) getSecurityGroups() ([]SSecurityGroup, error) {
	secgroups := make([]SSecurityGroup, 0)
	q := manager.Query()
	if err := db.FetchModelObjects(manager, q, &secgroups); err != nil {
		return nil, err
	} else {
		return secgroups, nil
	}
}

func (manager *SSecurityGroupManager) SyncSecgroups(ctx context.Context, userCred mcclient.TokenCredential, secgroups []cloudprovider.ICloudSecurityGroup) ([]SSecurityGroup, []cloudprovider.ICloudSecurityGroup, compare.SyncResult) {
	localSecgroups := make([]SSecurityGroup, 0)
	remoteSecgroups := make([]cloudprovider.ICloudSecurityGroup, 0)
	syncResult := compare.SyncResult{}

	if dbSecgroups, err := manager.getSecurityGroups(); err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	} else {
		removed := make([]SSecurityGroup, 0)
		commondb := make([]SSecurityGroup, 0)
		commonext := make([]cloudprovider.ICloudSecurityGroup, 0)
		added := make([]cloudprovider.ICloudSecurityGroup, 0)
		if err := compare.CompareSets(dbSecgroups, secgroups, &removed, &commondb, &commonext, &added); err != nil {
			syncResult.Error(err)
			return nil, nil, syncResult
		}

		for i := 0; i < len(commondb); i += 1 {
			if rules, err := commonext[i].GetRules(); err != nil {
				syncResult.Error(err)
			} else if len(rules) > 0 {
				if err = commondb[i].SyncWithCloudSecurityGroup(userCred, commonext[i]); err != nil {
					syncResult.UpdateError(err)
				} else {
					localSecgroups = append(localSecgroups, commondb[i])
					remoteSecgroups = append(remoteSecgroups, commonext[i])
					SecurityGroupRuleManager.SyncRules(ctx, userCred, &commondb[i], rules)
					syncResult.Update()
				}
			}
		}

		for i := 0; i < len(added); i += 1 {
			if rules, err := added[i].GetRules(); err != nil {
				syncResult.AddError(err)
			} else if len(rules) > 0 {
				if new, err := manager.newFromCloudVpc(added[i]); err != nil {
					syncResult.AddError(err)
				} else {
					localSecgroups = append(localSecgroups, *new)
					remoteSecgroups = append(remoteSecgroups, added[i])
					SecurityGroupRuleManager.SyncRules(ctx, userCred, new, rules)
					syncResult.Add()
				}
			}
		}
	}
	return localSecgroups, remoteSecgroups, syncResult
}

func (self *SSecurityGroup) SyncWithCloudSecurityGroup(userCred mcclient.TokenCredential, extSec cloudprovider.ICloudSecurityGroup) error {
	if _, err := self.GetModelManager().TableSpec().Update(self, func() error {
		extSec.Refresh()
		self.Name = extSec.GetName()
		self.Description = extSec.GetDescription()
		return nil
	}); err != nil {
		log.Errorf("syncWithCloudSecurityGroup error %s", err)
		return err
	}
	return nil
}

func (manager *SSecurityGroupManager) newFromCloudVpc(extSec cloudprovider.ICloudSecurityGroup) (*SSecurityGroup, error) {
	secgroup := SSecurityGroup{}
	secgroup.SetModelManager(manager)
	secgroup.Name = extSec.GetName()
	secgroup.ExternalId = extSec.GetGlobalId()
	secgroup.Description = extSec.GetDescription()

	if err := manager.TableSpec().Insert(&secgroup); err != nil {
		return nil, err
	}
	return &secgroup, nil
}

func (manager *SSecurityGroupManager) DelaySync(ctx context.Context, userCred mcclient.TokenCredential, idStr string) {
	if secgrp := manager.FetchSecgroupById(idStr); secgrp == nil {
		log.Errorf("DelaySync secgroup failed")
	} else {
		needSync := false
		lockman.LockObject(ctx, secgrp)
		defer lockman.ReleaseObject(ctx, secgrp)
		if secgrp.IsDirty {
			if _, err := secgrp.GetModelManager().TableSpec().Update(secgrp, func() error {
				secgrp.IsDirty = false
				return nil
			}); err != nil {
				log.Errorf("Update Security Group error: %s", err.Error())
			}
			needSync = true
		}
		if needSync {
			for _, guest := range secgrp.GetGuests() {
				guest.StartSyncTask(ctx, userCred, true, "")
			}
		}
	}
}

func (self *SSecurityGroup) DoSync(ctx context.Context, userCred mcclient.TokenCredential) {
	if _, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.IsDirty = true
		return nil
	}); err != nil {
		log.Errorf("Update Security Group error: %s", err.Error())
	}
	time.AfterFunc(10*time.Second, func() {
		SecurityGroupManager.DelaySync(ctx, userCred, self.Id)
	})
}
