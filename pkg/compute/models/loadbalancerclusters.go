// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLoadbalancerClusterManager struct {
	db.SStandaloneResourceBaseManager
}

var LoadbalancerClusterManager *SLoadbalancerClusterManager

func init() {
	LoadbalancerClusterManager = &SLoadbalancerClusterManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SLoadbalancerCluster{},
			"loadbalancerclusters_tbl",
			"loadbalancercluster",
			"loadbalancerclusters",
		),
	}
	LoadbalancerClusterManager.SetVirtualObject(LoadbalancerClusterManager)
}

type SLoadbalancerCluster struct {
	db.SStandaloneResourceBase
	SZoneResourceBase
	WireId string `width:"36" charset:"ascii" nullable:"true" list:"admin" create:"optional" update:"admin"`
}

func (man *SLoadbalancerClusterManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, man)
}

func (man *SLoadbalancerClusterManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, man)
}

func (lbc *SLoadbalancerCluster) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, lbc)
}

func (lbc *SLoadbalancerCluster) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, lbc)
}

func (lbc *SLoadbalancerCluster) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, lbc)
}

func (man *SLoadbalancerClusterManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "zone", ModelKeyword: "zone", OwnerId: userCred},
		{Key: "wire", ModelKeyword: "wire", OwnerId: userCred},
	})
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (man *SLoadbalancerClusterManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	zoneV := validators.NewModelIdOrNameValidator("zone", "zone", ownerId)
	wireV := validators.NewModelIdOrNameValidator("wire", "wire", ownerId)
	vs := []validators.IValidator{
		zoneV,
		wireV.Optional(true),
	}
	for _, v := range vs {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	zone := zoneV.Model.(*SZone)
	if zone.ExternalId != "" {
		return nil, httperrors.NewInputParameterError("allow only internal zone, got %s(%s)", zone.Name, zone.Id)
	}
	if wireV.Model != nil {
		wire := wireV.Model.(*SWire)
		if wire.ZoneId != zone.Id {
			return nil, httperrors.NewInputParameterError("wire zone must match zone parameter, got %s, want %s(%s)",
				wire.ZoneId, zone.Name, zone.Id)
		}
	}
	return man.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (lbc *SLoadbalancerCluster) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	wireV := validators.NewModelIdOrNameValidator("wire", "wire", lbc.GetOwnerId())
	wireV.Optional(true)
	if err := wireV.Validate(data); err != nil {
		return nil, err
	}
	if wireV.Model != nil {
		wire := wireV.Model.(*SWire)
		if wire.ZoneId != lbc.ZoneId {
			return nil, httperrors.NewInputParameterError("zone of wire must be %s, got %s", lbc.ZoneId, wire.ZoneId)
		}
		var from string
		if lbc.WireId != "" {
			from = "from " + lbc.WireId + " "
		}
		log.Infof("changing wire attribute of lbcluster %s(%s) %sto %s(%s)",
			lbc.Name, lbc.Id, from, wire.Name, wire.Id)
	}
	return lbc.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (lbc *SLoadbalancerCluster) ValidateDeleteCondition(ctx context.Context) error {
	men := []db.IModelManager{
		LoadbalancerManager,
	}
	lbcId := lbc.Id
	for _, man := range men {
		t := man.TableSpec().Instance()
		pdF := t.Field("pending_deleted")
		n, err := t.Query().
			Equals("cluster_id", lbcId).
			Filter(sqlchemy.OR(sqlchemy.IsNull(pdF), sqlchemy.IsFalse(pdF))).
			CountWithError()
		if err != nil {
			return httperrors.NewInternalServerError("get lbcluster refcount fail %v", err)
		}
		if n > 0 {
			return httperrors.NewResourceBusyError("lbcluster %s(%s) is still referred to by %d %s",
				lbcId, lbc.Name, n, man.KeywordPlural())
		}
	}
	return lbc.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (lbc *SLoadbalancerCluster) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lbc.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	zoneInfo := lbc.SZoneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if zoneInfo != nil {
		extra.Update(zoneInfo)
	}
	return extra
}

func (lbc *SLoadbalancerCluster) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra := lbc.GetCustomizeColumns(ctx, userCred, query)
	return extra, nil
}

func (lbc *SLoadbalancerCluster) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lbagents := []SLoadbalancerAgent{}
	q := LoadbalancerAgentManager.Query().Equals("cluster_id", lbc.Id)
	if err := db.FetchModelObjects(LoadbalancerAgentManager, q, &lbagents); err != nil {
		return errors.WithMessagef(err, "lbcluster %s(%s): find lbagents", lbc.Name, lbc.Id)
	}
	for i := range lbagents {
		lbagent := &lbagents[i]
		if err := lbagent.ValidateDeleteCondition(ctx); err != nil {
			return errors.WithMessagef(err, "lbagent %s(%s): validate delete", lbagent.Name, lbagent.Id)
		}
		if err := lbagent.CustomizeDelete(ctx, userCred, query, data); err != nil {
			return errors.WithMessagef(err, "lbagent %s(%s): customize delete", lbagent.Name, lbagent.Id)
		}
		lbagent.PreDelete(ctx, userCred)
		if err := lbagent.Delete(ctx, userCred); err != nil {
			return errors.WithMessagef(err, "lbagent %s(%s): delete", lbagent.Name, lbagent.Id)
		}
		lbagent.PostDelete(ctx, userCred)
	}
	return lbc.SStandaloneResourceBase.CustomizeDelete(ctx, userCred, query, data)
}

func (man *SLoadbalancerClusterManager) FindByZoneId(zoneId string) []SLoadbalancerCluster {
	r := []SLoadbalancerCluster{}
	q := man.Query().Equals("zone_id", zoneId)
	if err := db.FetchModelObjects(man, q, &r); err != nil {
		log.Errorf("find lbclusters by zone_id %s: %v", zoneId, err)
		return nil
	}
	return r
}

func (man *SLoadbalancerClusterManager) findByVrrpRouterIdInZone(zoneId string, routerId int) (*SLoadbalancerCluster, error) {
	var r *SLoadbalancerCluster

	peerClusters := man.FindByZoneId(zoneId)
	for i := range peerClusters {
		peerCluster := &peerClusters[i]
		peerClusterLbagents, err := man.getLoadbalancerAgents(peerCluster.Id)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		for j := range peerClusterLbagents {
			peerClusterLbagent := &peerClusterLbagents[j]
			if peerClusterLbagent.Params.Vrrp.VirtualRouterId == routerId {
				if r != nil {
					return nil, httperrors.NewInternalServerError("lbclusters %s(%s) and %s(%s) has conflict virtual_router_id: %d ", r.Name, r.Id, peerCluster.Name, peerCluster.Id, routerId)
				}
				r = peerCluster
				break
			}
		}
	}
	return r, nil
}

func (man *SLoadbalancerClusterManager) getLoadbalancerAgents(clusterId string) ([]SLoadbalancerAgent, error) {
	r := []SLoadbalancerAgent{}
	q := LoadbalancerAgentManager.Query().Equals("cluster_id", clusterId)
	err := db.FetchModelObjects(LoadbalancerAgentManager, q, &r)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (man *SLoadbalancerClusterManager) InitializeData() error {
	// find existing lb with empty clusterid
	lbs := []SLoadbalancer{}
	lbQ := LoadbalancerManager.Query().
		IsFalse("pending_deleted").
		IsNullOrEmpty("manager_id").
		IsNullOrEmpty("cluster_id")
	if err := db.FetchModelObjects(LoadbalancerManager, lbQ, &lbs); err != nil {
		return errors.WithMessage(err, "find lb with empty cluster_id")
	}

	// create 1 cluster for each zone
	zoneCluster := map[string]*SLoadbalancerCluster{}
	for i := range lbs {
		lb := &lbs[i]
		zoneId := lb.ZoneId
		if zoneId == "" {
			// just in case
			log.Warningf("found lb with empty zone_id: %s(%s)", lb.Name, lb.Id)
			continue
		}
		lbc, ok := zoneCluster[zoneId]
		if !ok {
			lbcs := man.FindByZoneId(zoneId)
			if len(lbcs) == 0 {
				m, err := db.NewModelObject(man)
				if err != nil {
					return errors.WithMessage(err, "new model object")
				}
				lbc = m.(*SLoadbalancerCluster)
				lbc.Name = "auto-lbc-" + zoneId
				lbc.ZoneId = zoneId
				if err := man.TableSpec().Insert(lbc); err != nil {
					return errors.WithMessage(err, "insert lbcluster model")
				}
			} else {
				if len(lbcs) > 1 {
					log.Infof("zone %s has %d lbclusters, select one", zoneId, len(lbcs))
				}
				lbc = &lbcs[0]
			}
			zoneCluster[zoneId] = lbc
		}
		if _, err := db.UpdateWithLock(context.Background(), lb, func() error {
			lb.ClusterId = lbc.Id
			return nil
		}); err != nil {
			return errors.WithMessagef(err, "lb %s(%s): assign cluster: %s(%s)", lb.Name, lb.Name, lbc.Name, lbc.Id)
		}
	}

	// associate existing lbagents with the cluster
	if len(zoneCluster) > 1 {
		log.Warningf("found %d zones with lb not assigned to any lbcluster, skip assigning lbagent to lbcluster", len(zoneCluster))
		return nil
	}
	for _, lbc := range zoneCluster {
		lbagents := []SLoadbalancerAgent{}
		q := LoadbalancerAgentManager.Query().
			IsNullOrEmpty("cluster_id")
		if err := db.FetchModelObjects(LoadbalancerAgentManager, q, &lbagents); err != nil {
			return errors.WithMessage(err, "find lbagents with empty cluster_id")
		}
		for i := range lbagents {
			lbagent := &lbagents[i]
			if _, err := db.UpdateWithLock(context.Background(), lbagent, func() error {
				lbagent.ClusterId = lbc.Id
				return nil
			}); err != nil {
				return errors.WithMessagef(err, "lbagent %s(%s): assign cluster: %s(%s)",
					lbagent.Name, lbagent.Id, lbc.Name, lbc.Id)
			}
		}
	}

	return nil
}
