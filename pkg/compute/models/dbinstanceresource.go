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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDBInstanceResourceBase struct {
	DBInstanceId string `width:"36" charset:"ascii" name:"dbinstance_id" nullable:"false" list:"user" create:"required" index:"true" json:"dbinstance_id"`
}

type SDBInstanceResourceBaseManager struct {
	SVpcResourceBaseManager
}

func ValidateDBInstanceResourceInput(ctx context.Context, userCred mcclient.TokenCredential, input api.DBInstanceResourceInput) (*SDBInstance, api.DBInstanceResourceInput, error) {
	rdsObj, err := DBInstanceManager.FetchByIdOrName(ctx, userCred, input.DBInstanceId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", DBInstanceManager.Keyword(), input.DBInstanceId)
		} else {
			return nil, input, errors.Wrap(err, "DBInstanceManager.FetchByIdOrName")
		}
	}
	input.DBInstanceId = rdsObj.GetId()
	return rdsObj.(*SDBInstance), input, nil
}

func (self *SDBInstanceResourceBase) GetDBInstance() (*SDBInstance, error) {
	instance, err := DBInstanceManager.FetchById(self.DBInstanceId)
	if err != nil {
		return nil, errors.Wrap(err, "DBInstanceManager.FetchById")
	}
	return instance.(*SDBInstance), nil
}

func (self *SDBInstanceResourceBase) GetVpc() (*SVpc, error) {
	nat, err := self.GetDBInstance()
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstance")
	}
	return nat.GetVpc()
}

func (self *SDBInstanceResourceBase) GetCloudprovider() *SCloudprovider {
	vpc, err := self.GetVpc()
	if err != nil {
		return nil
	}
	return vpc.GetCloudprovider()
}

func (manager *SDBInstanceResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DBInstanceResourceInfo {
	rows := make([]api.DBInstanceResourceInfo, len(objs))
	dbinstanceIds := make([]string, len(objs))
	for i := range objs {
		var base *SDBInstanceResourceBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil {
			dbinstanceIds[i] = base.DBInstanceId
		}
	}
	dbInstances := make(map[string]SDBInstance)
	err := db.FetchStandaloneObjectsByIds(DBInstanceManager, dbinstanceIds, &dbInstances)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	vpcList := make([]interface{}, len(rows))
	for i := range rows {
		rows[i] = api.DBInstanceResourceInfo{}
		if dbInstance, ok := dbInstances[dbinstanceIds[i]]; ok {
			rows[i].DBInstance = dbInstance.Name
			rows[i].VpcId = dbInstance.VpcId
		}
		vpcList[i] = &SVpcResourceBase{rows[i].VpcId}
	}

	vpcRows := manager.SVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, vpcList, fields, isList)
	for i := range rows {
		rows[i].VpcResourceInfo = vpcRows[i]
	}

	return rows
}

func (manager *SDBInstanceResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DBInstanceFilterListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	if len(query.DBInstanceId) > 0 {
		var dbObj *SDBInstance
		dbObj, _, err = ValidateDBInstanceResourceInput(ctx, userCred, query.DBInstanceResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "DBInstanceManager.FetchByIdOrName")
		}
		q = q.Equals("dbinstance_id", dbObj.GetId())
	}

	subq := DBInstanceManager.Query("id").Snapshot()

	subq, err = manager.SVpcResourceBaseManager.ListItemFilter(ctx, subq, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemFilter")
	}

	if subq.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("dbinstance_id"), subq.SubQuery()))
	}
	return q, nil
}

func (manager *SDBInstanceResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "dbinstance":
		dbQuery := DBInstanceManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(dbQuery.Field("name", field))
		q = q.Join(dbQuery, sqlchemy.Equals(q.Field("dbinstance_id"), dbQuery.Field("id")))
		q.GroupBy(dbQuery.Field("name"))
		return q, nil
	}
	dbInstances := DBInstanceManager.Query("id", "vpc_id").SubQuery()
	q = q.LeftJoin(dbInstances, sqlchemy.Equals(q.Field("dbinstance_id"), dbInstances.Field("id")))
	q, err := manager.SVpcResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SDBInstanceResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DBInstanceFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := DBInstanceManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("dbinstance_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SDBInstanceResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.DBInstanceFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	dbQ := DBInstanceManager.Query().SubQuery()
	q = q.LeftJoin(dbQ, sqlchemy.Equals(joinField, dbQ.Field("id")))
	q = q.AppendField(dbQ.Field("name").Label("dbinstance"))
	orders = append(orders, query.OrderByDBInstance)
	fields = append(fields, subq.Field("dbinstance"))
	q, orders, fields = manager.SVpcResourceBaseManager.GetOrderBySubQuery(q, subq, dbQ.Field("vpc_id"), userCred, query.VpcFilterListInput, orders, fields)
	return q, orders, fields
}

func (manager *SDBInstanceResourceBaseManager) GetOrderByFields(query api.DBInstanceFilterListInput) []string {
	fields := make([]string, 0)
	vpcFields := manager.SVpcResourceBaseManager.GetOrderByFields(query.VpcFilterListInput)
	fields = append(fields, vpcFields...)
	fields = append(fields, query.OrderByDBInstance)
	return fields
}

func (manager *SDBInstanceResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		var err error
		subq := DBInstanceManager.Query("id", "name", "vpc_id", "manager_id", "cloudregion_id").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("dbinstance_id"), subq.Field("id")))
		if keys.Contains("dbinstance") {
			q = q.AppendField(subq.Field("name", "dbinstance"))
		}
		if keys.ContainsAny(manager.SVpcResourceBaseManager.GetExportKeys()...) {
			q, err = manager.SVpcResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemExportKeys")
			}
		}
	}
	return q, nil
}

func (manager *SDBInstanceResourceBaseManager) GetExportKeys() []string {
	keys := []string{"dbinstance"}
	keys = append(keys, manager.SVpcResourceBaseManager.GetExportKeys()...)
	return keys
}

func (self *SDBInstanceResourceBase) GetChangeOwnerCandidateDomainIds() []string {
	dbinst, _ := self.GetDBInstance()
	if dbinst != nil {
		return dbinst.GetChangeOwnerCandidateDomainIds()
	}
	return nil
}
