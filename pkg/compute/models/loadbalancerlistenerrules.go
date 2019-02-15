package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLoadbalancerListenerRuleManager struct {
	db.SVirtualResourceBaseManager
}

var LoadbalancerListenerRuleManager *SLoadbalancerListenerRuleManager

func init() {
	LoadbalancerListenerRuleManager = &SLoadbalancerListenerRuleManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SLoadbalancerListenerRule{},
			"loadbalancerlistenerrules_tbl",
			"loadbalancerlistenerrule",
			"loadbalancerlistenerrules",
		),
	}
}

type SLoadbalancerListenerRule struct {
	db.SVirtualResourceBase
	SManagedResourceBase

	CloudregionId  string `width:"36" charset:"ascii" nullable:"false" list:"admin" default:"default" create:"optional"`
	ListenerId     string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	BackendGroupId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`

	Domain string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	Path   string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"optional"`

	SLoadbalancerHTTPRateLimiter
}

func loadbalancerListenerRuleCheckUniqueness(ctx context.Context, lbls *SLoadbalancerListener, domain, path string) error {
	q := LoadbalancerListenerRuleManager.Query().
		IsFalse("pending_deleted").
		Equals("listener_id", lbls.Id).
		Equals("domain", domain).
		Equals("path", path)
	var lblsr SLoadbalancerListenerRule
	q.First(&lblsr)
	if len(lblsr.Id) > 0 {
		return httperrors.NewConflictError("rule %s/%s already occupied by rule %s(%s)", domain, path, lblsr.Name, lblsr.Id)
	}
	return nil
}

func (man *SLoadbalancerListenerRuleManager) PreDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery) {
	subs := []SLoadbalancerListenerRule{}
	db.FetchModelObjects(man, q, &subs)
	for _, sub := range subs {
		sub.DoPendingDelete(ctx, userCred)
	}
}

func (man *SLoadbalancerListenerRuleManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	userProjId := userCred.GetProjectId()
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "listener", ModelKeyword: "loadbalancerlistener", ProjectId: userProjId},
		{Key: "backend_group", ModelKeyword: "loadbalancerbackendgroup", ProjectId: userProjId},
	})
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (man *SLoadbalancerListenerRuleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	listenerV := validators.NewModelIdOrNameValidator("listener", "loadbalancerlistener", ownerProjId)
	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", ownerProjId)
	domainV := validators.NewDomainNameValidator("domain")
	pathV := validators.NewURLPathValidator("path")
	keyV := map[string]validators.IValidator{
		"status": validators.NewStringChoicesValidator("status", LB_STATUS_SPEC).Default(LB_STATUS_ENABLED),

		"listener":      listenerV,
		"backend_group": backendGroupV,
		"domain":        domainV.AllowEmpty(true).Default(""),
		"path":          pathV.Default(""),

		"http_request_rate":         validators.NewNonNegativeValidator("http_request_rate").Default(0),
		"http_request_rate_per_src": validators.NewNonNegativeValidator("http_request_rate_per_src").Default(0),
	}
	for _, v := range keyV {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	listener := listenerV.Model.(*SLoadbalancerListener)
	data.Set("cloudregion_id", jsonutils.NewString(listener.CloudregionId))
	data.Set("manager_id", jsonutils.NewString(listener.ManagerId))
	listenerType := listener.ListenerType
	if listenerType != LB_LISTENER_TYPE_HTTP && listenerType != LB_LISTENER_TYPE_HTTPS {
		return nil, fmt.Errorf("listener type must be http/https, got %s", listenerType)
	}
	{
		if backendGroup, ok := backendGroupV.Model.(*SLoadbalancerBackendGroup); ok && backendGroup.LoadbalancerId != listener.LoadbalancerId {
			return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s instead of %s",
				backendGroup.Name, backendGroup.Id, backendGroup.LoadbalancerId, listener.LoadbalancerId)
		}
	}
	err := loadbalancerListenerRuleCheckUniqueness(ctx, listener, domainV.Value, pathV.Value)
	if err != nil {
		return nil, err
	}
	if _, err := man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data); err != nil {
		return nil, err
	}
	region := listener.GetRegion()
	if region == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to find region for loadbalancer listener %s", listener.Name)
	}

	return region.GetDriver().ValidateCreateLoadbalancerListenerRuleData(ctx, userCred, data, backendGroupV.Model)
}

func (lbr *SLoadbalancerListenerRule) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbr.SVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)

	lbr.SetStatus(userCred, LB_CREATING, "")
	if err := lbr.StartLoadBalancerListenerRuleCreateTask(ctx, userCred, ""); err != nil {
		log.Errorf("Failed to create loadbalancer listener rule error: %v", err)
	}
}

func (lbr *SLoadbalancerListenerRule) StartLoadBalancerListenerRuleCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerListenerRuleCreateTask", lbr, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbr *SLoadbalancerListenerRule) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lbr, "purge")
}

func (lbr *SLoadbalancerListenerRule) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	parasm := jsonutils.NewDict()
	parasm.Add(jsonutils.JSONTrue, "purge")
	return nil, lbr.StartLoadBalancerListenerRuleDeleteTask(ctx, userCred, parasm, "")
}

func (lbr *SLoadbalancerListenerRule) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lbr.SetStatus(userCred, LB_STATUS_DELETING, "")
	return lbr.StartLoadBalancerListenerRuleDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (lbr *SLoadbalancerListenerRule) StartLoadBalancerListenerRuleDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerListenerRuleDeleteTask", lbr, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbr *SLoadbalancerListenerRule) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return lbr.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, lbr, "status")
}

func (lbr *SLoadbalancerListenerRule) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", lbr.GetOwnerProjectId())
	keyV := map[string]validators.IValidator{
		"backend_group":             backendGroupV,
		"http_request_rate":         validators.NewNonNegativeValidator("http_request_rate"),
		"http_request_rate_per_src": validators.NewNonNegativeValidator("http_request_rate_per_src"),
	}
	for _, v := range keyV {
		v.Optional(true)
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	if backendGroup, ok := backendGroupV.Model.(*SLoadbalancerBackendGroup); ok && backendGroup.Id != lbr.BackendGroupId {
		listenerM, err := LoadbalancerListenerManager.FetchById(lbr.ListenerId)
		if err != nil {
			return nil, httperrors.NewInputParameterError("loadbalancerlistenerrule %s(%s): fetching listener %s failed",
				lbr.Name, lbr.Id, lbr.ListenerId)
		}
		listener := listenerM.(*SLoadbalancerListener)
		if backendGroup.LoadbalancerId != listener.LoadbalancerId {
			return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s instead of %s",
				backendGroup.Name, backendGroup.Id, backendGroup.LoadbalancerId, listener.LoadbalancerId)
		}
	}
	return lbr.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (lbr *SLoadbalancerListenerRule) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lbr.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if lbr.BackendGroupId == "" {
		log.Errorf("loadbalancer listener rule %s(%s): empty backend group field", lbr.Name, lbr.Id)
		return extra
	}
	lbbg, err := LoadbalancerBackendGroupManager.FetchById(lbr.BackendGroupId)
	if err != nil {
		log.Errorf("loadbalancer listener rule %s(%s): fetch backend group (%s) error: %s",
			lbr.Name, lbr.Id, lbr.BackendGroupId, err)
		return extra
	}
	extra.Set("backend_group", jsonutils.NewString(lbbg.GetName()))
	return extra
}

func (lbr *SLoadbalancerListenerRule) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra := lbr.GetCustomizeColumns(ctx, userCred, query)
	return extra, nil
}

func (lbr *SLoadbalancerListenerRule) GetLoadbalancerListener() *SLoadbalancerListener {
	listener, err := LoadbalancerListenerManager.FetchById(lbr.ListenerId)
	if err != nil {
		log.Errorf("failed to find listener for loadbalancer listener rule %s", lbr.Name)
		return nil
	}
	return listener.(*SLoadbalancerListener)
}

func (lbr *SLoadbalancerListenerRule) GetRegion() *SCloudregion {
	if listener := lbr.GetLoadbalancerListener(); listener != nil {
		return listener.GetRegion()
	}
	return nil
}

func (lbr *SLoadbalancerListenerRule) GetLoadbalancerBackendGroup() *SLoadbalancerBackendGroup {
	group, err := LoadbalancerBackendGroupManager.FetchById(lbr.BackendGroupId)
	if err != nil {
		return nil
	}
	return group.(*SLoadbalancerBackendGroup)
}

func (lbr *SLoadbalancerListenerRule) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

// Delete, Update

func (man *SLoadbalancerListenerRuleManager) getLoadbalancerListenerRulesByListener(listener *SLoadbalancerListener) ([]SLoadbalancerListenerRule, error) {
	rules := []SLoadbalancerListenerRule{}
	q := man.Query().Equals("listener_id", listener.Id)
	if err := db.FetchModelObjects(man, q, &rules); err != nil {
		log.Errorf("failed to get lb listener rules for listener %s error: %v", listener.Name, err)
		return nil, err
	}
	return rules, nil
}

func (man *SLoadbalancerListenerRuleManager) SyncLoadbalancerListenerRules(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, listener *SLoadbalancerListener, rules []cloudprovider.ICloudLoadbalancerListenerRule, syncRange *SSyncRange) compare.SyncResult {
	syncResult := compare.SyncResult{}

	dbRules, err := man.getLoadbalancerListenerRulesByListener(listener)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := []SLoadbalancerListenerRule{}
	commondb := []SLoadbalancerListenerRule{}
	commonext := []cloudprovider.ICloudLoadbalancerListenerRule{}
	added := []cloudprovider.ICloudLoadbalancerListenerRule{}

	err = compare.CompareSets(dbRules, rules, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].ValidateDeleteCondition(ctx)
		if err != nil { // cannot delete
			err = removed[i].SetStatus(userCred, LB_STATUS_UNKNOWN, "sync to delete")
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		} else {
			err = removed[i].Delete(ctx, userCred)
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		}
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerListenerRule(ctx, userCred, commonext[i], provider.ProjectId, syncRange.ProjectSync)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		_, err := man.newFromCloudLoadbalancerListenerRule(ctx, userCred, listener, added[i], provider.ProjectId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}

	return syncResult
}

func (lbr *SLoadbalancerListenerRule) constructFieldsFromCloudListenerRule(extRule cloudprovider.ICloudLoadbalancerListenerRule) {
	lbr.Name = extRule.GetName()
	lbr.Domain = extRule.GetDomain()
	lbr.Path = extRule.GetPath()
	if groupId := extRule.GetBackendGroupId(); len(groupId) > 0 {
		if backendgroup, err := LoadbalancerBackendGroupManager.FetchByExternalId(groupId); err == nil {
			lbr.BackendGroupId = backendgroup.GetId()
		}
	}
}

func (man *SLoadbalancerListenerRuleManager) newFromCloudLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, listener *SLoadbalancerListener, extRule cloudprovider.ICloudLoadbalancerListenerRule, projectId string) (*SLoadbalancerListenerRule, error) {
	lbr := &SLoadbalancerListenerRule{}
	lbr.SetModelManager(man)

	lbr.ExternalId = extRule.GetGlobalId()
	lbr.ListenerId = listener.Id

	lbr.constructFieldsFromCloudListenerRule(extRule)
	lbr.ProjectId = userCred.GetProjectId()
	if len(projectId) > 0 {
		lbr.ProjectId = projectId
	}

	return lbr, man.TableSpec().Insert(lbr)
}

func (lbr *SLoadbalancerListenerRule) SyncWithCloudLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, extRule cloudprovider.ICloudLoadbalancerListenerRule, projectId string, projectSync bool) error {
	_, err := lbr.GetModelManager().TableSpec().Update(lbr, func() error {
		lbr.constructFieldsFromCloudListenerRule(extRule)

		if projectSync && len(projectId) > 0 {
			lbr.ProjectId = projectId
		}
		return nil
	})
	return err
}

func (manager *SLoadbalancerListenerRuleManager) InitializeData() error {
	rules := []SLoadbalancerListenerRule{}
	q := manager.Query()
	q = q.Filter(sqlchemy.IsNullOrEmpty(q.Field("cloudregion_id")))
	if err := db.FetchModelObjects(manager, q, &rules); err != nil {
		return err
	}
	for i := 0; i < len(rules); i++ {
		rule := &rules[i]
		if listener := rule.GetLoadbalancerListener(); listener != nil && len(listener.CloudregionId) > 0 {
			_, err := listener.GetModelManager().TableSpec().Update(rule, func() error {
				rule.CloudregionId = listener.CloudregionId
				rule.ManagerId = listener.ManagerId
				return nil
			})
			if err != nil {
				log.Errorf("failed to update loadbalancer listener rule %s cloudregion_id", rule.Name)
			}
		}
	}
	return nil
}
