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
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SLoadbalancerBackendgroupResourceBase struct {
	// 负载均衡后端组ID
	BackendGroupId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"backend_group_id"`
}

type SLoadbalancerBackendgroupResourceBaseManager struct {
	SLoadbalancerResourceBaseManager
}

func (self *SLoadbalancerBackendgroupResourceBase) GetLoadbalancerBackendGroup() (*SLoadbalancerBackendGroup, error) {
	w, err := LoadbalancerBackendGroupManager.FetchById(self.BackendGroupId)
	if err != nil {
		return nil, err
	}
	return w.(*SLoadbalancerBackendGroup), nil
}

func (self *SLoadbalancerBackendgroupResourceBase) GetLoadbalancer() (*SLoadbalancer, error) {
	lbbg, err := self.GetLoadbalancerBackendGroup()
	if err != nil {
		return nil, err
	}
	return lbbg.GetLoadbalancer()
}

func (self *SLoadbalancerBackendgroupResourceBase) GetVpc() (*SVpc, error) {
	lb, err := self.GetLoadbalancer()
	if err != nil {
		return nil, errors.Wrapf(err, "GetLoadbalancer")
	}
	return lb.GetVpc()
}

func (self *SLoadbalancerBackendgroupResourceBase) GetCloudprovider() (*SCloudprovider, error) {
	lb, err := self.GetLoadbalancer()
	if err != nil {
		return nil, err
	}
	return lb.GetCloudprovider(), nil
}

func (self *SLoadbalancerBackendgroupResourceBase) GetProviderName() string {
	lb, err := self.GetLoadbalancer()
	if err != nil {
		return ""
	}
	return lb.SManagedResourceBase.GetProviderName()
}

func (self *SLoadbalancerBackendgroupResourceBase) GetRegion() (*SCloudregion, error) {
	vpc, err := self.GetVpc()
	if err != nil {
		return nil, err
	}
	return vpc.GetRegion()
}

func (self *SLoadbalancerBackendgroupResourceBase) GetZone() (*SZone, error) {
	lb, err := self.GetLoadbalancer()
	if err != nil {
		return nil, err
	}
	return lb.GetZone()
}

func (manager *SLoadbalancerBackendgroupResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerBackendGroupResourceInfo {
	rows := make([]api.LoadbalancerBackendGroupResourceInfo, len(objs))

	lbbgIds := make([]string, len(objs))
	for i := range objs {
		var base *SLoadbalancerBackendgroupResourceBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil {
			lbbgIds[i] = base.BackendGroupId
		}
	}

	lbbgs := make(map[string]SLoadbalancerBackendGroup)
	err := db.FetchStandaloneObjectsByIds(LoadbalancerBackendGroupManager, lbbgIds, &lbbgs)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	lbList := make([]interface{}, len(rows))
	for i := range rows {
		rows[i] = api.LoadbalancerBackendGroupResourceInfo{}
		if lbbg, ok := lbbgs[lbbgIds[i]]; ok {
			rows[i].BackendGroup = lbbg.Name
			rows[i].LoadbalancerId = lbbg.LoadbalancerId
		}
		lbList[i] = &SLoadbalancerResourceBase{rows[i].LoadbalancerId}
	}

	lbRows := manager.SLoadbalancerResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, lbList, fields, isList)

	for i := range rows {
		rows[i].LoadbalancerResourceInfo = lbRows[i]
	}
	return rows
}

func (manager *SLoadbalancerBackendgroupResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerBackendGroupFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.BackendGroupId) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, LoadbalancerBackendGroupManager, &query.BackendGroupId)
		if err != nil {
			return nil, err
		}
		q = q.Equals("backend_group_id", query.BackendGroupId)
	}

	lbbgQ := LoadbalancerBackendGroupManager.Query("id").Snapshot()

	lbbgQ, err := manager.SLoadbalancerResourceBaseManager.ListItemFilter(ctx, lbbgQ, userCred, query.LoadbalancerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerResourceBaseManager.ListItemFilter")
	}

	if lbbgQ.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("backend_group_id"), lbbgQ.SubQuery()))
	}
	return q, nil
}

func (manager *SLoadbalancerBackendgroupResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	if field == "backend_group" {
		lbbgQuery := LoadbalancerBackendGroupManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(lbbgQuery.Field("name", field))
		q = q.Join(lbbgQuery, sqlchemy.Equals(q.Field("backend_group_id"), lbbgQuery.Field("id")))
		q.GroupBy(lbbgQuery.Field("name"))
		return q, nil
	} else {
		lbbgs := LoadbalancerBackendGroupManager.Query("id", "loadbalancer_id").SubQuery()
		q = q.LeftJoin(lbbgs, sqlchemy.Equals(q.Field("backend_id"), lbbgs.Field("id")))
		q, err := manager.SLoadbalancerResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}
	}
	return q, httperrors.ErrNotFound
}

func (manager *SLoadbalancerBackendgroupResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerBackendGroupFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := LoadbalancerBackendGroupManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("backend_group_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SLoadbalancerBackendgroupResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerBackendGroupFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	lbbgQ := LoadbalancerBackendGroupManager.Query().SubQuery()
	q = q.LeftJoin(lbbgQ, sqlchemy.Equals(joinField, lbbgQ.Field("id")))
	q = q.AppendField(lbbgQ.Field("name").Label("backend_group"))
	orders = append(orders, query.OrderByBackendGroup)
	fields = append(fields, subq.Field("backend_group"))
	q, orders, fields = manager.SLoadbalancerResourceBaseManager.GetOrderBySubQuery(q, subq, lbbgQ.Field("loadbalancer_id"), userCred, query.LoadbalancerFilterListInput, orders, fields)
	return q, orders, fields
}

func (manager *SLoadbalancerBackendgroupResourceBaseManager) GetOrderByFields(query api.LoadbalancerBackendGroupFilterListInput) []string {
	fields := make([]string, 0)
	lbFields := manager.SLoadbalancerResourceBaseManager.GetOrderByFields(query.LoadbalancerFilterListInput)
	fields = append(fields, lbFields...)
	fields = append(fields, query.OrderByBackendGroup)
	return fields
}

func (manager *SLoadbalancerBackendgroupResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		var err error
		subq := LoadbalancerBackendGroupManager.Query("id", "name", "loadbalancer_id").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("backend_group_id"), subq.Field("id")))
		if keys.Contains("backend_group") {
			q = q.AppendField(subq.Field("name", "backend_group"))
		}
		if keys.ContainsAny(manager.SLoadbalancerResourceBaseManager.GetExportKeys()...) {
			q, err = manager.SLoadbalancerResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SLoadbalancerResourceBaseManager.ListItemExportKeys")
			}
		}
	}
	return q, nil
}

func (manager *SLoadbalancerBackendgroupResourceBaseManager) GetExportKeys() []string {
	keys := []string{"backend_group"}
	keys = append(keys, manager.SLoadbalancerResourceBaseManager.GetExportKeys()...)
	return keys
}
