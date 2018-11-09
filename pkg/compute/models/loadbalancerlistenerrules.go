package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
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

	ListenerId     string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	BackendGroupId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`

	Domain string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	Path   string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"optional"`
}

func loadbalancerListenerRuleCheckUniqueness(ctx context.Context, lbls *SLoadbalancerListener, domain, path string) error {
	q := LoadbalancerListenerRuleManager.Query().
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
		sub.PreDelete(ctx, userCred)
	}
}

func (man *SLoadbalancerListenerRuleManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	userProjId := userCred.GetProjectId()
	data := query.(*jsonutils.JSONDict)
	{
		listenerV := validators.NewModelIdOrNameValidator("listener", "loadbalancerlistener", userProjId)
		listenerV.Optional(true)
		q, err = listenerV.QueryFilter(q, data)
		if err != nil {
			return nil, err
		}
	}
	{
		backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", userProjId)
		backendGroupV.Optional(true)
		q, err = backendGroupV.QueryFilter(q, data)
		if err != nil {
			return nil, err
		}
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
	}
	for _, v := range keyV {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	listener := listenerV.Model.(*SLoadbalancerListener)
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
	return man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (lbr *SLoadbalancerListenerRule) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return lbr.IsOwner(userCred) || userCred.IsSystemAdmin()
}

func (lbr *SLoadbalancerListenerRule) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", lbr.GetOwnerProjectId())
	backendGroupV.Optional(true)
	err := backendGroupV.Validate(data)
	if err != nil {
		return nil, err
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

func (lbr *SLoadbalancerListenerRule) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lbr.GetCustomizeColumns(ctx, userCred, query)
	return extra
}

func (lbr *SLoadbalancerListenerRule) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	lbr.SetStatus(userCred, LB_STATUS_DISABLED, "preDelete")
	lbr.DoPendingDelete(ctx, userCred)
}

func (lbr *SLoadbalancerListenerRule) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

// Delete, Update
