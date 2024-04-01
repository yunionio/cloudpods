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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SServiceManager struct {
	db.SStandaloneResourceBaseManager
}

var ServiceManager *SServiceManager

func init() {
	ServiceManager = &SServiceManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SService{},
			"service",
			"service",
			"services",
		),
	}
	ServiceManager.SetVirtualObject(ServiceManager)
}

/*
+------------+--------------+------+-----+---------+-------+
| Field      | Type         | Null | Key | Default | Extra |
+------------+--------------+------+-----+---------+-------+
| id         | varchar(64)  | NO   | PRI | NULL    |       |
| type       | varchar(255) | YES  |     | NULL    |       |
| enabled    | tinyint(1)   | NO   |     | 1       |       |
| extra      | text         | YES  |     | NULL    |       |
| created_at | datetime     | YES  |     | NULL    |       |
+------------+--------------+------+-----+---------+-------+
*/

type SService struct {
	db.SStandaloneResourceBase

	Type    string              `width:"255" charset:"utf8" list:"admin" create:"admin_required"`
	Enabled tristate.TriState   `default:"true" list:"admin" update:"admin" create:"admin_optional"`
	Extra   *jsonutils.JSONDict `nullable:"true" list:"admin"`

	ConfigVersion int `list:"admin" nullable:"false" default:"0"`
}

func (manager *SServiceManager) InitializeData() error {
	q := manager.Query()
	q = q.IsNullOrEmpty("name")
	srvs := make([]SService, 0)
	err := db.FetchModelObjects(manager, q, &srvs)
	if err != nil {
		return err
	}
	for i := range srvs {
		if gotypes.IsNil(srvs[i].Extra) {
			continue
		}
		name, _ := srvs[i].Extra.GetString("name")
		desc, _ := srvs[i].Extra.GetString("description")
		if len(name) == 0 {
			name = srvs[i].Type
		}
		db.Update(&srvs[i], func() error {
			srvs[i].Name = name
			srvs[i].Description = desc
			return nil
		})
	}
	return nil
}

func (service *SService) GetEndpointCount() (int, error) {
	q := EndpointManager.Query().Equals("service_id", service.Id)
	return q.CountWithError()
}

func (service *SService) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	epCnt, _ := service.GetEndpointCount()
	if epCnt > 0 {
		return httperrors.NewNotEmptyError("service contains endpoints")
	}
	if service.Enabled.IsTrue() {
		return httperrors.NewInvalidStatusError("service is enabled")
	}
	return service.SStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (manager *SServiceManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ServiceDetails {
	rows := make([]api.ServiceDetails, len(objs))

	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.ServiceDetails{
			StandaloneResourceDetails: stdRows[i],
		}
		rows[i].EndpointCount, _ = objs[i].(*SService).GetEndpointCount()
	}

	return rows
}

func (service *SService) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	service.SStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	logclient.AddActionLogWithContext(ctx, service, logclient.ACT_CREATE, data, userCred, true)
}

func (service *SService) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	service.SStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
	logclient.AddActionLogWithContext(ctx, service, logclient.ACT_UPDATE, data, userCred, true)
}

func (service *SService) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	service.SStandaloneResourceBase.PostDelete(ctx, userCred)
	logclient.AddActionLogWithContext(ctx, service, logclient.ACT_DELETE, nil, userCred, true)
}

func (service *SService) GetDetailsConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var whiteList, blackList map[string][]string
	if service.isCommonService() {
		// whitelist common options
		whiteList = api.CommonWhitelistOptionMap
	} else {
		// blacklist common options
		blackList = api.CommonWhitelistOptionMap
	}
	conf, err := GetConfigs(service, false, whiteList, blackList)
	if err != nil {
		return nil, err
	}
	result := jsonutils.NewDict()
	result.Add(jsonutils.Marshal(conf), "config")
	return result, nil
}

func (service *SService) isCommonService() bool {
	if service.Type == consts.COMMON_SERVICE {
		return true
	} else {
		return false
	}
}

func (service *SService) PerformConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.PerformConfigInput) (jsonutils.JSONObject, error) {
	var err error
	var changed bool
	action := input.Action
	opts := input.Config
	if service.isCommonService() {
		changed, err = saveConfigs(userCred, action, service, opts, api.CommonWhitelistOptionMap, nil, nil)
	} else {
		changed, err = saveConfigs(userCred, action, service, opts, nil, api.MergeServiceConfigOptions(api.CommonWhitelistOptionMap, api.ServiceBlacklistOptionMap), nil)
	}
	if err != nil {
		return nil, err
	}
	if changed {
		diff := SService{ConfigVersion: 1}
		err = ServiceManager.TableSpec().Increment(ctx, diff, service)
		if err != nil {
			return nil, httperrors.NewInternalServerError("update config version fail %s", err)
		}
		if service.Type == api.SERVICE_TYPE || service.Type == consts.COMMON_SERVICE {
			options.OptionManager.SyncOnce(false, false)
		}
	}
	return service.GetDetailsConfig(ctx, userCred, query)
}

func (manager *SServiceManager) fetchServiceByType(typeStr string) (*SService, error) {
	q := manager.Query().Equals("type", typeStr)
	cnt, err := q.CountWithError()
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrap(err, "CountWithError")
	}
	if cnt == 0 {
		return nil, sql.ErrNoRows
	} else if cnt > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	}
	srvObj, err := db.NewModelObject(manager)
	if err != nil {
		return nil, errors.Wrap(err, "db.NewModelObject")
	}
	err = q.First(srvObj)
	if err != nil {
		return nil, errors.Wrap(err, "q.First")
	}
	return srvObj.(*SService), nil
}

// 服务列表
func (manager *SServiceManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ServiceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	if len(query.Type) > 0 {
		q = q.In("type", query.Type)
	}
	if query.Enabled != nil {
		if *query.Enabled {
			q = q.IsTrue("enabled")
		} else {
			q = q.IsFalse("enabled")
		}
	}
	return q, nil
}

func (manager *SServiceManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.RegionListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SServiceManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}
