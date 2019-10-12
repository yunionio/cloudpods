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
	"net"
	"reflect"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SRoute struct {
	Type        string
	Cidr        string
	NextHopType string
	NextHopId   string
}

func (route *SRoute) Validate(data *jsonutils.JSONDict) error {
	if strings.Index(route.Cidr, "/") > 0 {
		_, ipNet, err := net.ParseCIDR(route.Cidr)
		if err != nil {
			return err
		}
		// normalize from 192.168.1.3/24 to 192.168.1.0/24
		route.Cidr = ipNet.String()
	} else {
		ip := net.ParseIP(route.Cidr).To4()
		if ip == nil {
			return httperrors.NewInputParameterError("invalid addr %s", route.Cidr)
		}
	}
	return nil
}

type SRoutes []*SRoute

func (routes SRoutes) String() string {
	return jsonutils.Marshal(routes).String()
}
func (routes SRoutes) IsZero() bool {
	if len(routes) == 0 {
		return true
	}
	return false
}

func (routes *SRoutes) Validate(data *jsonutils.JSONDict) error {
	if routes == nil {
		*routes = SRoutes{}
		return nil
	}

	found := map[string]struct{}{}
	for _, route := range *routes {
		if err := route.Validate(data); err != nil {
			return err
		}
		if _, ok := found[route.Cidr]; ok {
			// error so that the user has a chance to deal with comments
			return httperrors.NewInputParameterError("duplicate route cidr %s", route.Cidr)
		}
		// TODO aliyun: check overlap with System type route
		found[route.Cidr] = struct{}{}
	}
	return nil
}

type SRouteTableManager struct {
	db.SVirtualResourceBaseManager
}

var RouteTableManager *SRouteTableManager

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SRoutes{}), func() gotypes.ISerializable {
		return &SRoutes{}
	})
	RouteTableManager = &SRouteTableManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SRouteTable{},
			"route_tables_tbl",
			"route_table",
			"route_tables",
		),
	}
	RouteTableManager.SetVirtualObject(RouteTableManager)
}

type SRouteTable struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase

	VpcId         string   `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	CloudregionId string   `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	Type          string   `width:"16" charset:"ascii" nullable:"false" list:"user"`
	Routes        *SRoutes `list:"user" update:"user" create:"required"`
}

func (man *SRouteTableManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	var err error
	q, err = managedResourceFilterByAccount(q, query, "", nil)
	if err != nil {
		return nil, err
	}
	q = managedResourceFilterByCloudType(q, query, "", nil)

	q, err = man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	// userProjId := userCred.GetProjectId()
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "vpc", ModelKeyword: "vpc", OwnerId: userCred},
		{Key: "cloudregion", ModelKeyword: "cloudregion", OwnerId: userCred},
		{Key: "manager", ModelKeyword: "cloudprovider", OwnerId: userCred},
	})
	if err != nil {
		return nil, err
	}

	/*managerStr := jsonutils.GetAnyString(query, []string{"manager", "cloudprovider", "cloudprovider_id", "manager_id"})
	if len(managerStr) > 0 {
		provider, err := CloudproviderManager.FetchByIdOrName(nil, managerStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudproviderManager.Keyword(), managerStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		sq := VpcManager.Query("id").Equals("manager_id", provider.GetId())
		q = q.In("vpc_id", sq.SubQuery())
	}

	accountStr := jsonutils.GetAnyString(query, []string{"account", "account_id", "cloudaccount", "cloudaccount_id"})
	if len(accountStr) > 0 {
		account, err := CloudaccountManager.FetchByIdOrName(nil, accountStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudaccountManager.Keyword(), accountStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		vpcs := VpcManager.Query().SubQuery()
		cloudproviders := CloudproviderManager.Query().SubQuery()

		subq := vpcs.Query(vpcs.Field("id"))
		subq = subq.Join(cloudproviders, sqlchemy.Equals(cloudproviders.Field("id"), vpcs.Field("manager_id")))
		subq = subq.Filter(sqlchemy.Equals(cloudproviders.Field("cloudaccount_id"), account.GetId()))
		q = q.Filter(sqlchemy.In(q.Field("vpc_id"), subq.SubQuery()))
	}

	providerStr := jsonutils.GetAnyString(query, []string{"provider"})
	if len(providerStr) > 0 {
		vpcs := VpcManager.Query().SubQuery()
		cloudproviders := CloudproviderManager.Query().SubQuery()

		subq := vpcs.Query(vpcs.Field("id"))
		subq = subq.Join(cloudproviders, sqlchemy.Equals(cloudproviders.Field("id"), vpcs.Field("manager_id")))
		subq = subq.Filter(sqlchemy.Equals(cloudproviders.Field("provider"), providerStr))

		q = q.Filter(sqlchemy.In(q.Field("vpc_id"), subq.SubQuery()))
	}*/

	return q, nil
}

func (man *SRouteTableManager) validateRoutes(data *jsonutils.JSONDict, update bool) (*jsonutils.JSONDict, error) {
	routes := SRoutes{}
	routesV := validators.NewStructValidator("routes", &routes)
	if update {
		routesV.Optional(true)
	}
	err := routesV.Validate(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (man *SRouteTableManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := man.validateRoutes(data, false)
	if err != nil {
		return nil, err
	}
	vpcV := validators.NewModelIdOrNameValidator("vpc", "vpc", ownerId)
	if err := vpcV.Validate(data); err != nil {
		return nil, err
	}
	vpc := vpcV.Model.(*SVpc)
	cloudregion, err := vpc.GetRegion()
	if err != nil {
		return nil, httperrors.NewConflictError("failed getting region of vpc %s(%s)", vpc.Name, vpc.Id)
	}
	data.Set("cloudregion_id", jsonutils.NewString(cloudregion.Id))
	return man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (manager *SRouteTableManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	switch field {
	case "account":
		cloudproviders := CloudproviderManager.Query().SubQuery()
		cloudaccounts := CloudaccountManager.Query("name", "id").Distinct().SubQuery()
		q = q.Join(cloudproviders, sqlchemy.Equals(q.Field("manager_id"), cloudproviders.Field("id")))
		q = q.Join(cloudaccounts, sqlchemy.Equals(cloudproviders.Field("cloudaccount_id"), cloudaccounts.Field("id")))
		q.GroupBy(cloudaccounts.Field("name"))
		q.AppendField(cloudaccounts.Field("name", "account"))
	default:
		return q, httperrors.NewBadRequestError("unsupport field %s", field)
	}
	return q, nil
}

func (rt *SRouteTable) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, rt, "purge")
}

func (rt *SRouteTable) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := rt.ValidateDeleteCondition(ctx)
	if err != nil {
		return nil, err
	}
	provider := rt.GetCloudprovider()
	if provider != nil {
		if provider.Enabled {
			return nil, httperrors.NewInvalidStatusError("Cannot purge route_table on enabled cloud provider")
		}
	}
	err = rt.RealDelete(ctx, userCred)
	return nil, err
}

func (rt *SRouteTable) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return rt.SVirtualResourceBase.Delete(ctx, userCred)
}

func (rt *SRouteTable) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := RouteTableManager.validateRoutes(data, true)
	if err != nil {
		return nil, err
	}
	return rt.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (rt *SRouteTable) AllowPerformAddRoutes(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) bool {
	return rt.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, rt, "add-routes")
}

func (rt *SRouteTable) AllowPerformDelRoutes(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) bool {
	return rt.AllowPerformAddRoutes(ctx, userCred, query, data)
}

// PerformAddRoutes patches acl entries by adding then deleting the specified acls.
// This is intended mainly for command line operations.
func (rt *SRouteTable) PerformAddRoutes(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var routes SRoutes
	if rt.Routes != nil {
		routes_ := gotypes.DeepCopy(rt.Routes).(*SRoutes)
		routes = *routes_
	}
	{
		adds := SRoutes{}
		addsV := validators.NewStructValidator("routes", &adds)
		addsV.Optional(true)
		err := addsV.Validate(data)
		if err != nil {
			return nil, err
		}
		for _, add := range adds {
			found := false
			for _, route := range routes {
				if route.Cidr == add.Cidr {
					found = true
					break
				}
			}
			if !found {
				routes = append(routes, add)
			}
		}
	}
	_, err := db.Update(rt, func() error {
		rt.Routes = &routes
		return nil
	})
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (rt *SRouteTable) PerformDelRoutes(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var routes SRoutes
	if rt.Routes != nil {
		routes_ := gotypes.DeepCopy(rt.Routes).(*SRoutes)
		routes = *routes_
	}
	{
		cidrs := []string{}
		err := data.Unmarshal(&cidrs, "cidrs")
		if err != nil {
			return nil, httperrors.NewInputParameterError("unmarshaling cidrs failed: %s", err)
		}
		for _, cidr := range cidrs {
			for i := len(routes) - 1; i >= 0; i-- {
				route := routes[i]
				if route.Type == "system" {
					continue
				}
				if route.Cidr == cidr {
					routes = append(routes[:i], routes[i+1:]...)
					break
				}
			}
		}
	}
	_, err := db.Update(rt, func() error {
		rt.Routes = &routes
		return nil
	})
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (rt *SRouteTable) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	info := rt.getCloudProviderInfo()
	extra.Update(jsonutils.Marshal(&info))
	return extra
}

func (rt *SRouteTable) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := rt.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	vpcM, err := VpcManager.FetchById(rt.VpcId)
	if err != nil {
		log.Errorf("route table %s(%s): fetch vpc (%s) error: %s",
			rt.Name, rt.Id, rt.VpcId, err)
		return extra
	}
	cloudregionM, err := CloudregionManager.FetchById(rt.CloudregionId)
	if err != nil {
		log.Errorf("route table %s(%s): fetch cloud region (%s) error: %s",
			rt.Name, rt.Id, rt.CloudregionId, err)
		return extra
	}
	extra.Set("vpc", jsonutils.NewString(vpcM.GetName()))
	extra.Set("cloudregion", jsonutils.NewString(cloudregionM.GetName()))

	extra = rt.getMoreDetails(extra)
	return extra
}

func (rt *SRouteTable) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra := rt.GetCustomizeColumns(ctx, userCred, query)
	extra = rt.getMoreDetails(extra)
	return extra, nil
}

func (man *SRouteTableManager) SyncRouteTables(ctx context.Context, userCred mcclient.TokenCredential, vpc *SVpc, cloudRouteTables []cloudprovider.ICloudRouteTable) ([]SRouteTable, []cloudprovider.ICloudRouteTable, compare.SyncResult) {
	lockman.LockClass(ctx, man, db.GetLockClassKey(man, userCred))
	defer lockman.ReleaseClass(ctx, man, db.GetLockClassKey(man, userCred))

	localRouteTables := make([]SRouteTable, 0)
	remoteRouteTables := make([]cloudprovider.ICloudRouteTable, 0)
	syncResult := compare.SyncResult{}

	dbRouteTables := []SRouteTable{}
	if err := db.FetchModelObjects(man, man.Query().Equals("vpc_id", vpc.Id), &dbRouteTables); err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}
	removed := make([]SRouteTable, 0)
	commondb := make([]SRouteTable, 0)
	commonext := make([]cloudprovider.ICloudRouteTable, 0)
	added := make([]cloudprovider.ICloudRouteTable, 0)
	if err := compare.CompareSets(dbRouteTables, cloudRouteTables, &removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err := removed[i].syncRemoveCloudRouteTable(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		err := commondb[i].SyncWithCloudRouteTable(ctx, userCred, vpc, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}
		syncMetadata(ctx, userCred, &commondb[i], commonext[i])
		localRouteTables = append(localRouteTables, commondb[i])
		remoteRouteTables = append(remoteRouteTables, commonext[i])
		syncResult.Update()
	}

	for i := 0; i < len(added); i += 1 {
		routeTableNew, err := man.insertFromCloud(ctx, userCred, vpc, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		syncMetadata(ctx, userCred, routeTableNew, added[i])
		localRouteTables = append(localRouteTables, *routeTableNew)
		remoteRouteTables = append(remoteRouteTables, added[i])
		syncResult.Add()
	}
	return localRouteTables, remoteRouteTables, syncResult
}

func (man *SRouteTableManager) newRouteTableFromCloud(userCred mcclient.TokenCredential, vpc *SVpc, cloudRouteTable cloudprovider.ICloudRouteTable) (*SRouteTable, error) {
	routes := SRoutes{}
	{
		cloudRoutes, err := cloudRouteTable.GetIRoutes()
		if err != nil {
			return nil, err
		}
		for _, cloudRoute := range cloudRoutes {
			route := &SRoute{
				Type:        cloudRoute.GetType(),
				Cidr:        cloudRoute.GetCidr(),
				NextHopType: cloudRoute.GetNextHopType(),
				NextHopId:   cloudRoute.GetNextHop(),
			}
			routes = append(routes, route)
		}
	}
	routeTable := &SRouteTable{
		CloudregionId: vpc.CloudregionId,
		VpcId:         vpc.Id,
		Type:          cloudRouteTable.GetType(),
		Routes:        &routes,
	}
	{
		basename := routeTableBasename(cloudRouteTable.GetName(), vpc.Name)
		newName, err := db.GenerateName(man, userCred, basename)
		if err != nil {
			return nil, err
		}
		routeTable.Name = newName
	}
	routeTable.ManagerId = vpc.ManagerId
	routeTable.ExternalId = cloudRouteTable.GetGlobalId()
	routeTable.Description = cloudRouteTable.GetDescription()
	routeTable.ProjectId = userCred.GetProjectId()
	routeTable.DomainId = userCred.GetProjectDomainId()
	routeTable.SetModelManager(man, routeTable)
	return routeTable, nil
}

func routeTableBasename(name, vpcName string) string {
	if name != "" {
		return name
	} else if vpcName != "" {
		return "rtbl-" + vpcName
	} else {
		return "rtbl"
	}
}

func (man *SRouteTableManager) insertFromCloud(ctx context.Context, userCred mcclient.TokenCredential, vpc *SVpc, cloudRouteTable cloudprovider.ICloudRouteTable) (*SRouteTable, error) {
	routeTable, err := man.newRouteTableFromCloud(userCred, vpc, cloudRouteTable)
	if err != nil {
		return nil, err
	}
	if err := man.TableSpec().Insert(routeTable); err != nil {
		return nil, err
	}
	db.OpsLog.LogEvent(routeTable, db.ACT_CREATE, routeTable.GetShortDesc(ctx), userCred)
	return routeTable, nil
}

func (self *SRouteTable) syncRemoveCloudRouteTable(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	err = self.RealDelete(ctx, userCred)
	return err
}

func (self *SRouteTable) SyncWithCloudRouteTable(ctx context.Context, userCred mcclient.TokenCredential, vpc *SVpc, cloudRouteTable cloudprovider.ICloudRouteTable) error {
	man := self.GetModelManager().(*SRouteTableManager)
	routeTable, err := man.newRouteTableFromCloud(userCred, vpc, cloudRouteTable)
	if err != nil {
		return err
	}
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.CloudregionId = routeTable.CloudregionId
		self.VpcId = vpc.Id
		self.Type = routeTable.Type
		self.Routes = routeTable.Routes
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (self *SRouteTable) getVpc() (*SVpc, error) {
	val, err := VpcManager.FetchById(self.VpcId)
	if err != nil {
		log.Errorf("VpcManager.FetchById fail %s", err)
		return nil, err
	}
	return val.(*SVpc), nil
}

func (self *SRouteTable) getRegion() (*SCloudregion, error) {
	vpc, err := self.getVpc()
	if err != nil {
		return nil, err
	}
	return vpc.GetRegion()
}

func (self *SRouteTable) getCloudProviderInfo() SCloudProviderInfo {
	region, _ := self.getRegion()
	provider := self.GetCloudprovider()
	return MakeCloudProviderInfo(region, nil, provider)
}
