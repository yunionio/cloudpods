package models

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type SSecurityGroupManager struct {
	db.SSharableVirtualResourceBaseManager
}

var SecurityGroupManager *SSecurityGroupManager

func init() {
	SecurityGroupManager = &SSecurityGroupManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SSecurityGroup{},
			"secgroups_tbl",
			"secgroup",
			"secgroups",
		),
	}
	SecurityGroupManager.NameLength = 128
	SecurityGroupManager.NameRequireAscii = true
}

const (
	SECURITY_GROUP_SEPARATOR = ";"
)

type SSecurityGroup struct {
	db.SSharableVirtualResourceBase
	IsDirty bool `nullable:"false" default:"false"` // Column(Boolean, nullable=False, default=False)
}

func (manager *SSecurityGroupManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	equalSecgroup, _ := query.GetString("equals")
	if len(equalSecgroup) > 0 {
		_secgroup, err := manager.FetchByIdOrName(userCred, equalSecgroup)
		if err != nil {
			return nil, httperrors.NewInputParameterError("Failed fetching secgroup %s", equalSecgroup)
		}
		secgroup := _secgroup.(*SSecurityGroup)
		sq := manager.Query().NotEquals("id", secgroup.Id)
		secgroups := []SSecurityGroup{}
		err = db.FetchModelObjects(manager, sq, &secgroups)
		if err != nil {
			return nil, err
		}
		srs := secgroup.getSecurityGroupRuleSet()
		secgroupIds := []string{}
		for i := 0; i < len(secgroups); i++ {
			if srs.IsEqual(secgroups[i].getSecurityGroupRuleSet()) {
				secgroupIds = append(secgroupIds, secgroups[i].Id)
			}
		}
		q = q.In("id", secgroupIds)
	}
	return q, nil
}

func (self *SSecurityGroup) GetGuestsQuery() *sqlchemy.SQuery {
	guests := GuestManager.Query().SubQuery()
	return guests.Query().Filter(
		sqlchemy.OR(
			sqlchemy.Equals(guests.Field("secgrp_id"), self.Id),
			sqlchemy.Equals(guests.Field("admin_secgrp_id"), self.Id),
			sqlchemy.In(guests.Field("id"), GuestsecgroupManager.Query("guest_id").Equals("secgroup_id", self.Id).SubQuery()),
		),
	).Filter(sqlchemy.NotIn(guests.Field("hypervisor"), []string{HYPERVISOR_CONTAINER, HYPERVISOR_BAREMETAL, HYPERVISOR_ESXI}))
}

func (self *SSecurityGroup) GetGuestsCount() int {
	return self.GetGuestsQuery().Count()
}

func (self *SSecurityGroup) GetGuests() []SGuest {
	guests := []SGuest{}
	q := self.GetGuestsQuery()
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("GetGuests fail %s", err)
		return nil
	}
	return guests
}

func (self *SSecurityGroup) GetSecgroupCacheQuery() *sqlchemy.SQuery {
	return SecurityGroupCacheManager.Query().Equals("secgroup_id", self.Id)
}

func (self *SSecurityGroup) GetSecgroupCacheCount() int {
	return self.GetSecgroupCacheQuery().Count()
}

func (self *SSecurityGroup) getDesc() jsonutils.JSONObject {
	desc := jsonutils.NewDict()
	desc.Add(jsonutils.NewString(self.Name), "name")
	desc.Add(jsonutils.NewString(self.Id), "id")
	//desc.Add(jsonutils.NewString(self.getSecurityRuleString("")), "security_rules")
	return desc
}

func (self *SSecurityGroup) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SSharableVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	extra.Add(jsonutils.NewInt(int64(len(self.GetGuests()))), "guest_cnt")
	extra.Add(jsonutils.NewInt(int64(self.GetSecgroupCacheCount())), "cache_cnt")
	extra.Add(jsonutils.NewString(self.getSecurityRuleString("")), "rules")
	extra.Add(jsonutils.NewString(self.getSecurityRuleString("in")), "in_rules")
	extra.Add(jsonutils.NewString(self.getSecurityRuleString("out")), "out_rules")
	return extra, nil
}

func (self *SSecurityGroup) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SSharableVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra.Add(jsonutils.NewInt(int64(len(self.GetGuests()))), "guest_cnt")
	extra.Add(jsonutils.NewInt(int64(self.GetSecgroupCacheCount())), "cache_cnt")
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
		return
	}
	return
}

func (self *SSecurityGroup) GetSecRules(direction string) []secrules.SecurityRule {
	rules := make([]secrules.SecurityRule, 0)
	for _, _rule := range self.getSecurityRules(direction) {
		//这里没必要拆分为单个单个的端口,到公有云那边适配
		rule, err := _rule.toRule()
		if err != nil {
			log.Errorf(err.Error())
			continue
		}
		rules = append(rules, *rule)
	}
	return rules
}

func (self *SSecurityGroup) getSecurityRuleString(direction string) string {
	secgrouprules := self.getSecurityRules(direction)
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

func (self *SSecurityGroup) AllowPerformAddRule(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "add-rule")
}

func (self *SSecurityGroup) PerformAddRule(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	secgrouprule := &SSecurityGroupRule{SecgroupID: self.Id}
	secgrouprule.SetModelManager(SecurityGroupRuleManager)
	if err := data.Unmarshal(secgrouprule); err != nil {
		return nil, err
	}
	if len(secgrouprule.CIDR) > 0 {
		if !regutils.MatchCIDR(secgrouprule.CIDR) && !regutils.MatchIPAddr(secgrouprule.CIDR) {
			return nil, httperrors.NewInputParameterError("invalid ip address: %s", secgrouprule.CIDR)
		}
	} else {
		secgrouprule.CIDR = "0.0.0.0/0"
	}
	rule := secrules.SecurityRule{
		Priority:  int(secgrouprule.Priority),
		Direction: secrules.TSecurityRuleDirection(secgrouprule.Direction),
		Action:    secrules.TSecurityRuleAction(secgrouprule.Action),
		Protocol:  secgrouprule.Protocol,
		Ports:     []int{},
		PortStart: -1,
		PortEnd:   -1,
	}
	if err := rule.ParsePorts(secgrouprule.Ports); err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}
	if err := rule.ValidateRule(); err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}
	if err := SecurityGroupRuleManager.TableSpec().Insert(secgrouprule); err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}
	self.DoSync(ctx, userCred)
	return nil, nil
}

func (self *SSecurityGroup) AllowPerformClone(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (self *SSecurityGroup) PerformClone(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if name, _ := data.GetString("name"); len(name) == 0 {
		return nil, httperrors.NewMissingParameterError("name")
	} else {
		sql := SecurityGroupManager.Query()
		sql = SecurityGroupManager.FilterByName(sql, name)
		if sql.Count() != 0 {
			return nil, httperrors.NewDuplicateNameError("name", name)
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

func (self *SSecurityGroup) AllowPerformUnion(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "union")
}

func (self *SSecurityGroup) PerformUnion(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	secgroupIds := jsonutils.GetQueryStringArray(data, "secgroups")
	if len(secgroupIds) == 0 {
		return nil, httperrors.NewMissingParameterError("secgroups")
	}
	srs := self.getSecurityGroupRuleSet()
	secgroups := []*SSecurityGroup{}
	for _, secgroupId := range secgroupIds {
		_secgroup, err := SecurityGroupManager.FetchByIdOrName(userCred, secgroupId)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("failed to find secgroup %s error: %v", secgroupId, err)
		}
		secgroup := _secgroup.(*SSecurityGroup)
		secgroup.SetModelManager(SecurityGroupManager)
		_srs := secgroup.getSecurityGroupRuleSet()
		if !srs.IsEqual(_srs) {
			return nil, httperrors.NewUnsupportOperationError("secgroup %s rules not equals %s rules", secgroup.Name, self.Name)
		}
		secgroups = append(secgroups, secgroup)
	}

	for i := 0; i < len(secgroups); i++ {
		secgroup := secgroups[i]
		if err := self.migrateSecurityGroupCache(secgroup); err != nil {
			return nil, err
		}

		if err := self.migrateGuestSecurityGroup(secgroup); err != nil {
			return nil, err
		}
		secgroup.RealDelete(ctx, userCred)
	}
	self.DoSync(ctx, userCred)
	return nil, nil
}

func (self *SSecurityGroup) getSecurityGroupRuleSet() secrules.SecurityGroupRuleSet {
	rules := self.GetSecRules("")
	srs := secrules.SecurityGroupRuleSet{}
	for i := 0; i < len(rules); i++ {
		srs.AddRule(rules[i])
	}
	return srs
}

func (self *SSecurityGroup) migrateSecurityGroupCache(secgroup *SSecurityGroup) error {
	caches := []SSecurityGroupCache{}
	q := SecurityGroupCacheManager.Query().Equals("secgroup_id", secgroup.Id)
	err := db.FetchModelObjects(SecurityGroupCacheManager, q, &caches)
	if err != nil {
		return err
	}
	for i := 0; i < len(caches); i++ {
		cache := caches[i]
		_, err := db.Update(&cache, func() error {
			cache.SecgroupId = self.Id
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SSecurityGroup) migrateGuestSecurityGroup(secgroup *SSecurityGroup) error {
	guests := secgroup.GetGuests()
	for i := 0; i < len(guests); i++ {
		guest := guests[i]
		_, err := db.Update(&guest, func() error {
			if guest.SecgrpId == secgroup.Id {
				guest.SecgrpId = self.Id
			}
			if guest.AdminSecgrpId == secgroup.Id {
				guest.AdminSecgrpId = self.Id
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	guestsecgroups, err := GuestsecgroupManager.GetGuestSecgroups(nil, secgroup)
	if err != nil {
		return err
	}

	for i := 0; i < len(guestsecgroups); i++ {
		guestsecgroup := guestsecgroups[i]
		_, err := db.Update(&guestsecgroup, func() error {
			guestsecgroup.SecgroupId = self.Id
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
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

func (manager *SSecurityGroupManager) newFromCloudSecgroup(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extSec cloudprovider.ICloudSecurityGroup) (*SSecurityGroup, error) {
	lockman.LockClass(ctx, manager, manager.GetOwnerId(userCred))
	defer lockman.ReleaseClass(ctx, manager, manager.GetOwnerId(userCred))

	secgroup := SSecurityGroup{}
	secgroup.SetModelManager(manager)
	secgroup.Name = db.GenerateName(manager, "", extSec.GetName())
	secgroup.Description = extSec.GetDescription()
	secgroup.ProjectId = userCred.GetProjectId()

	if err := manager.TableSpec().Insert(&secgroup); err != nil {
		return nil, err
	}

	db.OpsLog.LogEvent(&secgroup, db.ACT_CREATE, secgroup.GetShortDesc(ctx), userCred)

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
			if _, err := db.Update(secgrp, func() error {
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
	if _, err := db.Update(self, func() error {
		self.IsDirty = true
		return nil
	}); err != nil {
		log.Errorf("Update Security Group error: %s", err.Error())
	}
	time.AfterFunc(10*time.Second, func() {
		SecurityGroupManager.DelaySync(ctx, userCred, self.Id)
	})
}

func (manager *SSecurityGroupManager) InitializeData() error {
	_, err := manager.FetchById("default")
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("find default secgroup fail %s", err)
		return err
	}
	if err == sql.ErrNoRows {
		var secGrp *SSecurityGroup
		secGrp = &SSecurityGroup{}
		secGrp.SetModelManager(manager)
		secGrp.Id = "default"
		secGrp.Name = "Default"
		secGrp.ProjectId = auth.AdminCredential().GetProjectId()
		secGrp.IsEmulated = false
		secGrp.IsPublic = true
		err = manager.TableSpec().Insert(secGrp)
		if err != nil {
			log.Errorf("Insert default secgroup failed!!! %s", err)
			return err
		}

		defRule := SSecurityGroupRule{}
		defRule.SetModelManager(SecurityGroupRuleManager)
		defRule.Direction = secrules.DIR_IN
		defRule.Protocol = secrules.PROTO_ANY
		defRule.Priority = 1
		defRule.CIDR = "0.0.0.0/0"
		defRule.Action = string(secrules.SecurityRuleAllow)
		defRule.SecgroupID = "default"
		err = SecurityGroupRuleManager.TableSpec().Insert(&defRule)
		if err != nil {
			log.Errorf("Insert default secgroup rule fail %s", err)
			return err
		}
	}
	guests := make([]SGuest, 0)
	q := GuestManager.Query()
	q = q.Filter(sqlchemy.OR(sqlchemy.IsEmpty(q.Field("secgrp_id")), sqlchemy.IsNull(q.Field("secgrp_id"))))

	err = db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("fetch guests without secgroup fail %s", err)
		return err
	}
	for i := 0; i < len(guests); i += 1 {
		db.Update(&guests[i], func() error {
			guests[i].SecgrpId = "default"
			return nil
		})
	}
	return nil
}

func (self *SSecurityGroup) ValidateDeleteCondition(ctx context.Context) error {
	cnt := self.GetGuestsCount()
	if cnt > 0 {
		return httperrors.NewNotEmptyError("the security group is in use")
	}
	if self.Id == "default" {
		return httperrors.NewProtectedResourceError("not allow to delete default security group")
	}
	return self.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SSecurityGroup) GetSecurityGroupCaches() []SSecurityGroupCache {
	caches := []SSecurityGroupCache{}
	q := SecurityGroupCacheManager.Query()
	q = q.Filter(sqlchemy.Equals(q.Field("secgroup_id"), self.Id))
	if err := db.FetchModelObjects(SecurityGroupCacheManager, q, &caches); err != nil {
		log.Errorf("get secgroupcache for secgroup %s error: %v", self.Name, err)
	}
	return caches
}

func (self *SSecurityGroup) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteSecurityGroupTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (self *SSecurityGroup) StartDeleteSecurityGroupTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "SecurityGroupDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SSecurityGroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SSharableVirtualResourceBase.DoPendingDelete(ctx, userCred)
}

func (self *SSecurityGroup) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	rules := []SSecurityGroupRule{}
	q := SecurityGroupRuleManager.Query().Equals("secgroup_id", self.Id)
	if err := db.FetchModelObjects(SecurityGroupRuleManager, q, &rules); err != nil {
		log.Errorf("failed to fetch secgroup %s rules error: %v", self.Name, err)
		return err
	}
	for i := 0; i < len(rules); i++ {
		if err := rules[i].Delete(ctx, userCred); err != nil {
			return err
		}
	}
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}
