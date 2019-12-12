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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SEndpointManager struct {
	db.SStandaloneResourceBaseManager
}

var EndpointManager *SEndpointManager

func init() {
	EndpointManager = &SEndpointManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SEndpoint{},
			"endpoint",
			"endpoint",
			"endpoints",
		),
	}
	EndpointManager.SetVirtualObject(EndpointManager)
}

/*
+--------------------+--------------+------+-----+---------+-------+
| Field              | Type         | Null | Key | Default | Extra |
+--------------------+--------------+------+-----+---------+-------+
| id                 | varchar(64)  | NO   | PRI | NULL    |       |
| legacy_endpoint_id | varchar(64)  | YES  |     | NULL    |       |
| interface          | varchar(8)   | NO   |     | NULL    |       |
| service_id         | varchar(64)  | NO   | MUL | NULL    |       |
| url                | text         | NO   |     | NULL    |       |
| extra              | text         | YES  |     | NULL    |       |
| enabled            | tinyint(1)   | NO   |     | 1       |       |
| region_id          | varchar(255) | YES  | MUL | NULL    |       |
| created_at         | datetime     | YES  |     | NULL    |       |
+--------------------+--------------+------+-----+---------+-------+
*/

type SEndpoint struct {
	db.SStandaloneResourceBase

	LegacyEndpointId string              `width:"64" charset:"ascii" nullable:"true"`
	Interface        string              `width:"8" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`
	ServiceId        string              `width:"64" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`
	Url              string              `charset:"utf8" nullable:"false" list:"admin" update:"admin" create:"admin_required"`
	Extra            *jsonutils.JSONDict `nullable:"true"`
	Enabled          tristate.TriState   `nullable:"false" default:"true" list:"admin" update:"admin" create:"admin_optional"`
	RegionId         string              `width:"255" charset:"utf8" nullable:"true" list:"admin" create:"admin_required"`
}

func (manager *SEndpointManager) InitializeData() error {
	q := manager.Query()
	q = q.IsNullOrEmpty("name")
	eps := make([]SEndpoint, 0)
	err := db.FetchModelObjects(manager, q, &eps)
	if err != nil {
		return err
	}
	for i := range eps {
		name, _ := eps[i].Extra.GetString("name")
		desc, _ := eps[i].Extra.GetString("description")
		if len(name) == 0 {
			serv := eps[i].getService()
			if serv != nil {
				name = fmt.Sprintf("%s-%s", serv.Type, eps[i].Interface)
			}
		}
		if len(name) > 0 {
			db.Update(&eps[i], func() error {
				eps[i].Name = name
				eps[i].Description = desc
				return nil
			})
		}
	}
	return nil
}

func (ep *SEndpoint) getService() *SService {
	srvObj, err := ServiceManager.FetchById(ep.ServiceId)
	if err != nil {
		return nil
	}
	return srvObj.(*SService)
}

type SEndpointExtended struct {
	Id          string
	Name        string
	Interface   string
	Url         string
	Region      string
	RegionId    string
	ServiceId   string
	ServiceType string
	ServiceName string
}

type SServiceCatalog []SEndpointExtended

func (manager *SEndpointManager) FetchAll() (SServiceCatalog, error) {
	endpoints := manager.Query().SubQuery()
	services := ServiceManager.Query().SubQuery()
	regions := RegionManager.Query().SubQuery()

	q := endpoints.Query(
		endpoints.Field("id"),
		endpoints.Field("name"),
		endpoints.Field("interface"),
		endpoints.Field("url"),
		endpoints.Field("region_id"),
		regions.Field("name", "region"),
		endpoints.Field("service_id"),
		services.Field("type", "service_type"),
		services.Field("name", "service_name"),
	)
	q = q.Join(regions, sqlchemy.Equals(endpoints.Field("region_id"), regions.Field("id")))
	q = q.Join(services, sqlchemy.Equals(endpoints.Field("service_id"), services.Field("id")))
	q = q.Filter(sqlchemy.IsTrue(endpoints.Field("enabled")))
	q = q.Filter(sqlchemy.IsTrue(services.Field("enabled")))

	eps := make([]SEndpointExtended, 0)
	err := q.All(&eps)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return eps, nil
}

func (cata SServiceCatalog) GetKeystoneCatalogV3() mcclient.KeystoneServiceCatalogV3 {
	ksCata := make(map[string]mcclient.KeystoneServiceV3)

	for i := range cata {
		srvId := cata[i].ServiceId
		srv, ok := ksCata[srvId]
		if !ok {
			srv = mcclient.KeystoneServiceV3{
				Id:        srvId,
				Name:      cata[i].ServiceName,
				Type:      cata[i].ServiceType,
				Endpoints: make([]mcclient.KeystoneEndpointV3, 0),
			}
		}
		ep := mcclient.KeystoneEndpointV3{
			Id:        cata[i].Id,
			Name:      cata[i].Name,
			Interface: cata[i].Interface,
			Region:    cata[i].Region,
			RegionId:  cata[i].RegionId,
			Url:       cata[i].Url,
		}
		srv.Endpoints = append(srv.Endpoints, ep)
		ksCata[cata[i].ServiceId] = srv
	}

	results := make([]mcclient.KeystoneServiceV3, 0)
	for k := range ksCata {
		results = append(results, ksCata[k])
	}
	return results
}

func (cata SServiceCatalog) GetKeystoneCatalogV2() mcclient.KeystoneServiceCatalogV2 {
	ksCata := make(map[string]mcclient.KeystoneServiceV2)

	for i := range cata {
		srvId := cata[i].ServiceId
		srv, ok := ksCata[srvId]
		if !ok {
			srv = mcclient.KeystoneServiceV2{
				Name:      cata[i].ServiceName,
				Type:      cata[i].ServiceType,
				Endpoints: make([]mcclient.KeystoneEndpointV2, 0),
			}
		}
		findIdx := -1
		for j := range srv.Endpoints {
			if srv.Endpoints[j].Region == cata[i].RegionId {
				findIdx = j
			}
		}
		if findIdx < 0 {
			ep := mcclient.KeystoneEndpointV2{
				Id:     cata[i].Id,
				Region: cata[i].RegionId,
			}
			srv.Endpoints = append(srv.Endpoints, ep)
			findIdx = len(srv.Endpoints) - 1
		}
		switch cata[i].Interface {
		case api.EndpointInterfacePublic:
			srv.Endpoints[findIdx].PublicURL = cata[i].Url
		case api.EndpointInterfaceInternal:
			srv.Endpoints[findIdx].InternalURL = cata[i].Url
		case api.EndpointInterfaceAdmin:
			srv.Endpoints[findIdx].AdminURL = cata[i].Url
		}
		ksCata[cata[i].ServiceId] = srv
	}

	results := make([]mcclient.KeystoneServiceV2, 0)
	for k := range ksCata {
		results = append(results, ksCata[k])
	}
	return results
}

func (manager *SEndpointManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []db.IModel, fields stringutils2.SSortedStrings) []*jsonutils.JSONDict {
	rows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields)
	serviceIds := stringutils2.SSortedStrings{}
	for i := range objs {
		ep := objs[i].(*SEndpoint)
		serviceIds = stringutils2.Append(serviceIds, ep.ServiceId)
	}
	if len(fields) == 0 || fields.Contains("service_name") || fields.Contains("service_type") {
		svs := fetchServices(serviceIds)
		if svs != nil {
			for i := range rows {
				ep := objs[i].(*SEndpoint)
				if srv, ok := svs[ep.ServiceId]; ok {
					if len(fields) == 0 || fields.Contains("service_name") {
						rows[i].Add(jsonutils.NewString(srv.Name), "service_name")
					}
					if len(fields) == 0 || fields.Contains("service_type") {
						rows[i].Add(jsonutils.NewString(srv.Type), "service_type")
					}
				}
			}
		}
	}
	return rows
}

func fetchServices(srvIds []string) map[string]SService {
	q := ServiceManager.Query().In("id", srvIds)
	srvs := make([]SService, 0)
	err := db.FetchModelObjects(ServiceManager, q, &srvs)
	if err != nil {
		return nil
	}
	ret := make(map[string]SService)
	for i := range srvs {
		ret[srvs[i].Id] = srvs[i]
	}
	return ret
}

func (endpoint *SEndpoint) ValidateDeleteCondition(ctx context.Context) error {
	if endpoint.Enabled.IsTrue() {
		return httperrors.NewInvalidStatusError("endpoint is enabled")
	}
	return endpoint.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (endpoint *SEndpoint) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := endpoint.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return endpointExtra(endpoint, extra)
}

func (endpoint *SEndpoint) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := endpoint.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return endpointExtra(endpoint, extra), nil
}

func endpointExtra(endpoint *SEndpoint, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	return extra
}

func (manager *SEndpointManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	infname, _ := data.GetString("interface")
	if len(infname) == 0 {
		return nil, httperrors.NewInputParameterError("missing input field interface")
	}
	serviceStr := jsonutils.GetAnyString(data, []string{"service_id", "service"})
	if len(serviceStr) > 0 {
		servObj, err := ServiceManager.FetchByIdOrName(userCred, serviceStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ServiceManager.Keyword(), serviceStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		service := servObj.(*SService)
		if !data.Contains("name") {
			data.Set("name", jsonutils.NewString(fmt.Sprintf("%s-%s", service.Type, infname)))
		}
		data.Set("service_id", jsonutils.NewString(service.Id))
	} else {
		return nil, httperrors.NewInputParameterError("missing input field service/service_id")
	}
	input := apis.StandaloneResourceCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal StandaloneResourceCreateInput fail %s", err)
	}
	input, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))
	return data, nil
}

func (manager *SEndpointManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	svcStr := jsonutils.GetAnyString(query, []string{"service", "service_id"})
	if len(svcStr) > 0 {
		svcObj, err := ServiceManager.FetchByIdOrName(userCred, svcStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ServiceManager.Keyword(), svcStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		subq := ServiceManager.Query("id").Equals("id", svcObj.GetId())
		q = q.Equals("service_id", subq.SubQuery())
	}
	return q, nil
}

func (endpoint *SEndpoint) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	endpoint.SStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	logclient.AddActionLogWithContext(ctx, endpoint, logclient.ACT_CREATE, data, userCred, true)
	refreshDefaultClientServiceCatalog()
}

func (endpoint *SEndpoint) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	endpoint.SStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
	logclient.AddActionLogWithContext(ctx, endpoint, logclient.ACT_UPDATE, data, userCred, true)
	refreshDefaultClientServiceCatalog()
}

func (endpoint *SEndpoint) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	endpoint.SStandaloneResourceBase.PostDelete(ctx, userCred)
	logclient.AddActionLogWithContext(ctx, endpoint, logclient.ACT_DELETE, nil, userCred, true)
	refreshDefaultClientServiceCatalog()
}
