package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLoadbalancerBackendGroupManager struct {
	db.SVirtualResourceBaseManager
}

var LoadbalancerBackendGroupManager *SLoadbalancerBackendGroupManager

func init() {
	LoadbalancerBackendGroupManager = &SLoadbalancerBackendGroupManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SLoadbalancerBackendGroup{},
			"loadbalancerbackendgroups_tbl",
			"loadbalancerbackendgroup",
			"loadbalancerbackendgroups",
		),
	}
}

type SLoadbalancerBackendGroup struct {
	db.SVirtualResourceBase

	LoadbalancerId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
}

func (man *SLoadbalancerBackendGroupManager) PreDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery) {
	subs := []SLoadbalancerBackendGroup{}
	db.FetchModelObjects(man, q, &subs)
	for _, sub := range subs {
		sub.PreDelete(ctx, userCred)
	}
}

func (man *SLoadbalancerBackendGroupManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	userProjId := userCred.GetProjectId()
	data := query.(*jsonutils.JSONDict)
	{
		lbV := validators.NewModelIdOrNameValidator("loadbalancer", "loadbalancer", userProjId)
		lbV.Optional(true)
		q, err = lbV.QueryFilter(q, data)
		if err != nil {
			return nil, err
		}
	}
	return q, nil
}

func (man *SLoadbalancerBackendGroupManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	lbV := validators.NewModelIdOrNameValidator("loadbalancer", "loadbalancer", ownerProjId)
	err := lbV.Validate(data)
	if err != nil {
		return nil, err
	}
	return man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (lbbg *SLoadbalancerBackendGroup) GetLoadbalancer() *SLoadbalancer {
	lb, _ := LoadbalancerManager.FetchById(lbbg.LoadbalancerId)
	return lb.(*SLoadbalancer)
}

func (lbbg *SLoadbalancerBackendGroup) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (lbbg *SLoadbalancerBackendGroup) ValidateDeleteCondition(ctx context.Context) error {
	men := []db.IModelManager{
		LoadbalancerManager,
		LoadbalancerListenerManager,
		LoadbalancerListenerRuleManager,
	}
	lbbgId := lbbg.Id
	for _, man := range men {
		t := man.TableSpec().Instance()
		pdF := t.Field("pending_deleted")
		n := t.Query().
			Equals("backend_group_id", lbbgId).
			Filter(sqlchemy.OR(sqlchemy.IsNull(pdF), sqlchemy.IsFalse(pdF))).
			Count()
		if n > 0 {
			return fmt.Errorf("backend group %s is still referred to by %d %s",
				lbbgId, n, man.KeywordPlural())
		}
	}
	return nil
}

func (lbbg *SLoadbalancerBackendGroup) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lbbg.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	{
		lb, err := LoadbalancerManager.FetchById(lbbg.LoadbalancerId)
		if err != nil {
			log.Errorf("loadbalancer backend group %s(%s): fetch loadbalancer (%s) error: %s",
				lbbg.Name, lbbg.Id, lbbg.LoadbalancerId, err)
			return extra
		}
		extra.Set("loadbalancer", jsonutils.NewString(lb.GetName()))
	}
	return extra
}

func (lbbg *SLoadbalancerBackendGroup) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lbbg.GetCustomizeColumns(ctx, userCred, query)
	return extra
}

func (lbbg *SLoadbalancerBackendGroup) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	lbbg.DoPendingDelete(ctx, userCred)
	lbbg.PreDeleteSubs(ctx, userCred)
}

func (lbbg *SLoadbalancerBackendGroup) PreDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential) {
	subMan := LoadbalancerBackendManager
	ownerProjId := lbbg.GetOwnerProjectId()

	lockman.LockClass(ctx, subMan, ownerProjId)
	defer lockman.ReleaseClass(ctx, subMan, ownerProjId)
	q := subMan.Query().Equals("backend_group_id", lbbg.Id)
	subMan.PreDeleteSubs(ctx, userCred, q)
}

func (lbbg *SLoadbalancerBackendGroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}
