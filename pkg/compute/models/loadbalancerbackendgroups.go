package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
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
	SManagedResourceBase

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
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "loadbalancer", ModelKeyword: "loadbalancer", ProjectId: userProjId},
	})
	if err != nil {
		return nil, err
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

func (man *SLoadbalancerBackendGroupManager) getLoadbalancerBackendgroupsByLoadbalancer(lb *SLoadbalancer) ([]SLoadbalancerBackendGroup, error) {
	lbbgs := []SLoadbalancerBackendGroup{}
	q := man.Query().Equals("loadbalancer_id", lb.Id)
	if err := db.FetchModelObjects(man, q, &lbbgs); err != nil {
		log.Errorf("failed to get lbbgs for lb: %v error: %v", lb, err)
		return nil, err
	}
	return lbbgs, nil
}

func (man *SLoadbalancerBackendGroupManager) SyncLoadbalancerBackendgroups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, lb *SLoadbalancer, lbbgs []cloudprovider.ICloudLoadbalancerBackendgroup, syncRange *SSyncRange) ([]SLoadbalancerBackendGroup, []cloudprovider.ICloudLoadbalancerBackendgroup, compare.SyncResult) {
	localLbgs := []SLoadbalancerBackendGroup{}
	remoteLbbgs := []cloudprovider.ICloudLoadbalancerBackendgroup{}
	syncResult := compare.SyncResult{}

	dbLbbgs, err := man.getLoadbalancerBackendgroupsByLoadbalancer(lb)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := []SLoadbalancerBackendGroup{}
	commondb := []SLoadbalancerBackendGroup{}
	commonext := []cloudprovider.ICloudLoadbalancerBackendgroup{}
	added := []cloudprovider.ICloudLoadbalancerBackendgroup{}

	err = compare.CompareSets(dbLbbgs, lbbgs, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
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
		err = commondb[i].SyncWithCloudLoadbalancerBackendgroup(ctx, userCred, commonext[i], provider.ProjectId, syncRange.ProjectSync)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localLbgs = append(localLbgs, commondb[i])
			remoteLbbgs = append(remoteLbbgs, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		new, err := man.newFromCloudLoadbalancerBackendgroup(ctx, userCred, lb, added[i], provider.ProjectId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			localLbgs = append(localLbgs, *new)
			remoteLbbgs = append(remoteLbbgs, added[i])
			syncResult.Add()
		}
	}
	return localLbgs, remoteLbbgs, syncResult
}

func (lbbg *SLoadbalancerBackendGroup) SyncWithCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential, extLbbg cloudprovider.ICloudLoadbalancerBackendgroup, projectId string, projectSync bool) error {
	_, err := LoadbalancerBackendGroupManager.TableSpec().Update(lbbg, func() error {
		lbbg.Name = extLbbg.GetName()

		if projectSync && len(projectId) > 0 {
			lbbg.ProjectId = projectId
		}

		return nil
	})
	return err
}

func (man *SLoadbalancerBackendGroupManager) newFromCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extLbbg cloudprovider.ICloudLoadbalancerBackendgroup, projectId string) (*SLoadbalancerBackendGroup, error) {
	lbbg := SLoadbalancerBackendGroup{}
	lbbg.SetModelManager(man)

	lbbg.LoadbalancerId = lb.Id
	lbbg.Name = extLbbg.GetName()
	lbbg.ExternalId = extLbbg.GetGlobalId()
	lbbg.ProjectId = userCred.GetProjectId()
	if len(projectId) > 0 {
		lbbg.ProjectId = projectId
	}
	return &lbbg, man.TableSpec().Insert(&lbbg)
}
