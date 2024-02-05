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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
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

type IVpcResource interface {
	GetVpc() (*SVpc, error)
	GetRegion() (*SCloudregion, error)
}

type SVpcResourceBase struct {
	VpcId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"vpc_id"`
}

type SVpcResourceBaseManager struct {
	SCloudregionResourceBaseManager
	SManagedResourceBaseManager
}

func (self *SVpcResourceBase) GetVpc() (*SVpc, error) {
	obj, err := VpcManager.FetchById(self.VpcId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpc(%s)", self.VpcId)
	}
	return obj.(*SVpc), nil
}

func (self *SVpcResourceBase) GetRegion() (*SCloudregion, error) {
	vpc, err := self.GetVpc()
	if err != nil {
		return nil, err
	}
	return vpc.GetRegion()
}

func (self *SVpcResourceBase) GetRegionId() string {
	region, err := self.GetRegion()
	if err != nil {
		return ""
	}
	return region.Id
}

func (self *SVpcResourceBase) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	vpc, err := self.GetVpc()
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpc")
	}
	return vpc.GetIRegion(ctx)
}

func (self *SVpcResourceBase) GetCloudprovider() *SCloudprovider {
	vpc, err := self.GetVpc()
	if err != nil {
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
	vpc, _ := self.GetVpc()
	if vpc == nil {
		return ""
	}
	return vpc.GetProviderName()
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
			rows[i].IsDefaultVpc = vpc.IsDefault
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
	conditions := []sqlchemy.ICondition{}
	for _, vpcId := range query.VpcId {
		if len(vpcId) == 0 {
			continue
		}
		switch vpcId {
		case api.CLASSIC_VPC_NAME:
			conditions = append(conditions, sqlchemy.Equals(q.Field("name"), api.CLASSIC_VPC_NAME))
		default:
			_, err := validators.ValidateModel(ctx, userCred, VpcManager, &vpcId)
			if err != nil {
				return nil, err
			}
			conditions = append(conditions, sqlchemy.Equals(q.Field("vpc_id"), vpcId))
		}
	}
	if len(conditions) > 0 {
		q = q.Filter(sqlchemy.OR(conditions...))
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
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := VpcManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("vpc_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SVpcResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	subqField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.VpcFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	vpcQ := VpcManager.Query().SubQuery()
	q = q.LeftJoin(vpcQ, sqlchemy.Equals(subqField, vpcQ.Field("id")))
	q = q.AppendField(vpcQ.Field("name").Label("vpc"))
	orders = append(orders, query.OrderByVpc)
	fields = append(fields, subq.Field("vpc"))
	q, orders, fields = manager.SCloudregionResourceBaseManager.GetOrderBySubQuery(q, subq, vpcQ.Field("cloudregion_id"), userCred, query.RegionalFilterListInput, orders, fields)
	q, orders, fields = manager.SManagedResourceBaseManager.GetOrderBySubQuery(q, subq, vpcQ.Field("manager_id"), userCred, query.ManagedResourceListInput, orders, fields)
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
	vpc, _ := self.GetVpc()
	if vpc != nil {
		return vpc.GetChangeOwnerCandidateDomainIds()
	}
	return nil
}

func IsOneCloudVpcResource(res IVpcResource) bool {
	vpc, _ := res.GetVpc()
	if vpc == nil {
		return false
	}
	region, _ := res.GetRegion()
	if region == nil {
		return false
	}
	if region.Provider == api.CLOUD_PROVIDER_ONECLOUD && vpc.Id != api.DEFAULT_VPC_ID {
		return true
	}
	return false
}
