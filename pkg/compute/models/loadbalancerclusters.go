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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SLoadbalancerClusterManager struct {
	db.SStandaloneResourceBaseManager
	SZoneResourceBaseManager
	SWireResourceBaseManager
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
	SWireResourceBase `width:"36" charset:"ascii" nullable:"true" list:"admin" create:"optional" update:"admin"`
}

// 负载均衡集群列表
func (man *SLoadbalancerClusterManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerClusterListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = man.SZoneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ZonalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemFilter")
	}
	wireQuery := api.WireFilterListInput{
		WireFilterListBase: query.WireFilterListBase,
	}
	q, err = man.SWireResourceBaseManager.ListItemFilter(ctx, q, userCred, wireQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SWireResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (man *SLoadbalancerClusterManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerClusterListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SZoneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ZonalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.OrderByExtraFields")
	}
	wireQuery := api.WireFilterListInput{
		WireFilterListBase: query.WireFilterListBase,
	}
	q, err = man.SWireResourceBaseManager.OrderByExtraFields(ctx, q, userCred, wireQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SWireResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (man *SLoadbalancerClusterManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SZoneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SWireResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
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

	input := apis.StandaloneResourceCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal StandaloneResourceCreateInput fail %s", err)
	}
	input, err = man.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))
	return data, nil
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

	input := apis.StandaloneResourceBaseUpdateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	input, err = lbc.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBase.ValidateUpdateData")
	}
	data.Update(jsonutils.Marshal(input))

	return data, nil
}

func (lbc *SLoadbalancerCluster) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	men := []db.IModelManager{
		LoadbalancerManager,
	}
	lbcId := lbc.Id
	for _, man := range men {
		t := man.TableSpec().Instance()
		n, err := t.Query().
			Equals("cluster_id", lbcId).
			CountWithError()
		if err != nil {
			return httperrors.NewInternalServerError("get lbcluster refcount fail %v", err)
		}
		if n > 0 {
			return httperrors.NewResourceBusyError("lbcluster %s(%s) is still referred to by %d %s",
				lbcId, lbc.Name, n, man.KeywordPlural())
		}
	}
	return lbc.SStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (man *SLoadbalancerClusterManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerClusterDetails {
	rows := make([]api.LoadbalancerClusterDetails, len(objs))

	stdRows := man.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	zoneRows := man.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	wireRows := man.SWireResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.LoadbalancerClusterDetails{
			StandaloneResourceDetails: stdRows[i],
			ZoneResourceInfo:          zoneRows[i],
			WireResourceInfoBase:      wireRows[i].WireResourceInfoBase,
		}
	}

	return rows
}

func (lbc *SLoadbalancerCluster) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lbagents := []SLoadbalancerAgent{}
	q := LoadbalancerAgentManager.Query().Equals("cluster_id", lbc.Id)
	if err := db.FetchModelObjects(LoadbalancerAgentManager, q, &lbagents); err != nil {
		return errors.Wrapf(err, "lbcluster %s(%s): find lbagents", lbc.Name, lbc.Id)
	}
	for i := range lbagents {
		lbagent := &lbagents[i]
		if err := lbagent.ValidateDeleteCondition(ctx, nil); err != nil {
			return errors.Wrapf(err, "lbagent %s(%s): validate delete", lbagent.Name, lbagent.Id)
		}
		if err := lbagent.CustomizeDelete(ctx, userCred, query, data); err != nil {
			return errors.Wrapf(err, "lbagent %s(%s): customize delete", lbagent.Name, lbagent.Id)
		}
		lbagent.PreDelete(ctx, userCred)
		if err := lbagent.Delete(ctx, userCred); err != nil {
			return errors.Wrapf(err, "lbagent %s(%s): delete", lbagent.Name, lbagent.Id)
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
	lbQ := LoadbalancerManager.Query()
	vpcs := VpcManager.Query().SubQuery()
	lbQ = lbQ.Join(vpcs, sqlchemy.Equals(lbQ.Field("vpc_id"), vpcs.Field("id")))
	lbQ = lbQ.Filter(sqlchemy.IsFalse(lbQ.Field("pending_deleted")))
	lbQ = lbQ.Filter(sqlchemy.IsNullOrEmpty(vpcs.Field("manager_id")))
	lbQ = lbQ.Filter(sqlchemy.IsNullOrEmpty(lbQ.Field("cluster_id")))
	if err := db.FetchModelObjects(LoadbalancerManager, lbQ, &lbs); err != nil {
		return errors.Wrap(err, "find lb with empty cluster_id")
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
					return errors.Wrap(err, "new model object")
				}
				lbc = m.(*SLoadbalancerCluster)
				lbc.Name = "auto-lbc-" + zoneId
				lbc.ZoneId = zoneId
				if err := man.TableSpec().Insert(context.TODO(), lbc); err != nil {
					return errors.Wrap(err, "insert lbcluster model")
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
			return errors.Wrapf(err, "lb %s(%s): assign cluster: %s(%s)", lb.Name, lb.Name, lbc.Name, lbc.Id)
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
			return errors.Wrap(err, "find lbagents with empty cluster_id")
		}
		for i := range lbagents {
			lbagent := &lbagents[i]
			if _, err := db.UpdateWithLock(context.Background(), lbagent, func() error {
				lbagent.ClusterId = lbc.Id
				return nil
			}); err != nil {
				return errors.Wrapf(err, "lbagent %s(%s): assign cluster: %s(%s)",
					lbagent.Name, lbagent.Id, lbc.Name, lbc.Id)
			}
		}
	}

	return nil
}

func (manager *SLoadbalancerClusterManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SZoneResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SZoneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.Contains("wire") {
		q, err = manager.SWireResourceBaseManager.ListItemExportKeys(ctx, q, userCred, stringutils2.NewSortedStrings([]string{"wire"}))
		if err != nil {
			return nil, errors.Wrap(err, "SWireResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}
