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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SVpcResourceBase struct {
	VpcId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"vpc_id"`
}

type SVpcResourceBaseManager struct {
	SCloudregionResourceBaseManager
	SManagedResourceBaseManager
}

func ValidateVpcResourceInput(userCred mcclient.TokenCredential, input api.VpcResourceInput) (*SVpc, api.VpcResourceInput, error) {
	vpcObj, err := VpcManager.FetchByIdOrName(userCred, input.VpcId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, httperrors.NewResourceNotFoundError2(VpcManager.Keyword(), input.VpcId)
		} else {
			return nil, input, errors.Wrap(err, "VpcManager.FetchByIdOrName")
		}
	}
	input.VpcId = vpcObj.GetId()
	return vpcObj.(*SVpc), input, nil
}

func (self *SVpcResourceBase) GetVpc() *SVpc {
	obj, _ := VpcManager.FetchById(self.VpcId)
	if obj == nil {
		return nil
	}
	return obj.(*SVpc)
}

func (self *SVpcResourceBase) GetRegion() *SCloudregion {
	vpc := self.GetVpc()
	if vpc == nil {
		return nil
	}
	region, _ := vpc.GetRegion()
	return region
}

func (self *SVpcResourceBase) GetRegionId() string {
	region := self.GetRegion()
	if region != nil {
		return region.Id
	}
	return ""
}

func (self *SVpcResourceBase) GetIRegion() (cloudprovider.ICloudRegion, error) {
	vpc := self.GetVpc()
	if vpc != nil {
		return vpc.GetIRegion()
	}
	return nil, errors.Wrap(httperrors.ErrBadRequest, "not a valid vpc")
}

func (self *SVpcResourceBase) GetCloudprovider() *SCloudprovider {
	vpc := self.GetVpc()
	if vpc == nil {
		return nil
	}
	return vpc.GetCloudprovider()
}

func (self *SVpcResourceBase) GetCloudproviderId() string {
	cloudprovider := self.GetCloudprovider()
	if cloudprovider != nil {
		return cloudprovider.Id
	}
	return ""
}

func (self *SVpcResourceBase) GetProviderName() string {
	vpc := self.GetVpc()
	if vpc == nil {
		return ""
	}
	return vpc.GetProviderName()
}

func (self *SVpcResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) api.VpcResourceInfo {
	return api.VpcResourceInfo{}
}

func (manager *SVpcResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.VpcResourceInfo {
	rows := make([]api.VpcResourceInfo, len(objs))
	vpcIds := make([]string, len(objs))
	for i := range objs {
		var base *SVpcResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SCloudregionResourceBase in object %s", objs[i])
			continue
		}
		vpcIds[i] = base.VpcId
	}

	vpcs := make(map[string]SVpc)
	err := db.FetchStandaloneObjectsByIds(VpcManager, vpcIds, vpcs)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return nil
	}

	regionList := make([]interface{}, len(rows))
	managerList := make([]interface{}, len(rows))
	for i := range rows {
		rows[i] = api.VpcResourceInfo{}
		if _, ok := vpcs[vpcIds[i]]; ok {
			vpc := vpcs[vpcIds[i]]
			rows[i].Vpc = vpc.Name
			rows[i].VpcExtId = vpc.ExternalId
			rows[i].CloudregionId = vpc.CloudregionId
			rows[i].ManagerId = vpc.ManagerId
		}
		regionList[i] = &SCloudregionResourceBase{rows[i].CloudregionId}
		managerList[i] = &SManagedResourceBase{rows[i].ManagerId}
	}

	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, regionList, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, managerList, fields, isList)
	for i := range rows {
		rows[i].CloudregionResourceInfo = regionRows[i]
		rows[i].ManagedResourceInfo = managerRows[i]
	}

	return rows
}

func (manager *SVpcResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.VpcFilterListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	if len(query.VpcId) > 0 {
		vpcObj, _, err := ValidateVpcResourceInput(userCred, query.VpcResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateVpcResourceInput")
		}
		q = q.Equals("vpc_id", vpcObj.GetId())
	}
	subq := VpcManager.Query("id").Snapshot()
	subq, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, subq, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	subq, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, subq, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	if subq.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("vpc_id"), subq.SubQuery()))
	}
	return q, nil
}

func (manager *SVpcResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "vpc":
		vpcQuery := VpcManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(vpcQuery.Field("name", field))
		q = q.Join(vpcQuery, sqlchemy.Equals(q.Field("vpc_id"), vpcQuery.Field("id")))
		q.GroupBy(vpcQuery.Field("name"))
		return q, nil
	case "region":
		vpcs := VpcManager.Query("id", "cloudregion_id").SubQuery()
		q = q.LeftJoin(vpcs, sqlchemy.Equals(q.Field("vpc_id"), vpcs.Field("id")))
		return manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	case "manager", "account", "provider", "brand":
		vpcs := VpcManager.Query("id", "manager_id").SubQuery()
		q = q.LeftJoin(vpcs, sqlchemy.Equals(q.Field("vpc_id"), vpcs.Field("id")))
		return manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	}
	return q, httperrors.ErrNotFound
}

func (manager *SVpcResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.VpcFilterListInput,
) (*sqlchemy.SQuery, error) {
	q, orders, fields := manager.GetOrderBySubQuery(q, userCred, query)
	if len(orders) > 0 {
		q = db.OrderByFields(q, orders, fields)
	}
	return q, nil
}

func (manager *SVpcResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.VpcFilterListInput,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	vpcQ := VpcManager.Query("id", "name")
	var orders []string
	var fields []sqlchemy.IQueryField
	if db.NeedOrderQuery(manager.SCloudregionResourceBaseManager.GetOrderByFields(query.RegionalFilterListInput)) {
		var regionOrders []string
		var regionFields []sqlchemy.IQueryField
		vpcQ, regionOrders, regionFields = manager.SCloudregionResourceBaseManager.GetOrderBySubQuery(vpcQ, userCred, query.RegionalFilterListInput)
		if len(regionOrders) > 0 {
			orders = append(orders, regionOrders...)
			fields = append(fields, regionFields...)
		}
	}
	if db.NeedOrderQuery(manager.SManagedResourceBaseManager.GetOrderByFields(query.ManagedResourceListInput)) {
		var managerOrders []string
		var managerFields []sqlchemy.IQueryField
		vpcQ, managerOrders, managerFields = manager.SManagedResourceBaseManager.GetOrderBySubQuery(vpcQ, userCred, query.ManagedResourceListInput)
		if len(managerOrders) > 0 {
			orders = append(orders, managerOrders...)
			fields = append(fields, managerFields...)
		}
	}
	if db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		subq := vpcQ.SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("vpc_id"), subq.Field("id")))
		if db.NeedOrderQuery([]string{query.OrderByVpc}) {
			orders = append(orders, query.OrderByVpc)
			fields = append(fields, subq.Field("name"))
		}
	}
	return q, orders, fields
}

func (manager *SVpcResourceBaseManager) GetOrderByFields(query api.VpcFilterListInput) []string {
	fields := make([]string, 0)
	regionFields := manager.SCloudregionResourceBaseManager.GetOrderByFields(query.RegionalFilterListInput)
	fields = append(fields, regionFields...)
	managerFields := manager.SManagedResourceBaseManager.GetOrderByFields(query.ManagedResourceListInput)
	fields = append(fields, managerFields...)
	fields = append(fields, query.OrderByVpc)
	return fields
}

func (manager *SVpcResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		var err error
		subq := VpcManager.Query("id", "name", "manager_id", "cloudregion_id").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("vpc_id"), subq.Field("id")))
		if keys.Contains("vpc") {
			q = q.AppendField(subq.Field("name", "vpc"))
		}
		if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
			q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
			}
		}
		if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
			q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
			}
		}
	}
	return q, nil
}

func (manager *SVpcResourceBaseManager) GetExportKeys() []string {
	keys := []string{"vpc"}
	keys = append(keys, manager.SManagedResourceBaseManager.GetExportKeys()...)
	keys = append(keys, manager.SCloudregionResourceBaseManager.GetExportKeys()...)
	return keys
}

func (self *SVpcResourceBase) GetChangeOwnerCandidateDomainIds() []string {
	vpc := self.GetVpc()
	if vpc != nil {
		return vpc.GetChangeOwnerCandidateDomainIds()
	}
	return nil
}
