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
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
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
	Enabled tristate.TriState   `nullable:"false" default:"true" list:"admin" update:"admin" create:"admin_optional"`
	Extra   *jsonutils.JSONDict `nullable:"true" list:"admin"`
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

func (service *SService) ValidateDeleteCondition(ctx context.Context) error {
	epCnt, _ := service.GetEndpointCount()
	if epCnt > 0 {
		return httperrors.NewNotEmptyError("service contains endpoints")
	}
	if service.Enabled.IsTrue() {
		return httperrors.NewInvalidStatusError("service is enabled")
	}
	return service.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (service *SService) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := service.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return serviceExtra(service, extra)
}

func (service *SService) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := service.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return serviceExtra(service, extra), nil
}

func serviceExtra(service *SService, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	epCnt, _ := service.GetEndpointCount()
	extra.Add(jsonutils.NewInt(int64(epCnt)), "endpoint_count")
	return extra
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

func (service *SService) AllowGetDetailsConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, service, "config")
}

func (service *SService) GetDetailsConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	conf, err := GetConfigs(service, false)
	if err != nil {
		return nil, err
	}
	result := jsonutils.NewDict()
	result.Add(jsonutils.Marshal(conf), "config")
	return result, nil
}

func (service *SService) AllowPerformConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) bool {
	return db.IsAdminAllowUpdateSpec(userCred, service, "config")
}

func (service *SService) isCommonService() bool {
	if service.Type == consts.COMMON_SERVICE {
		return true
	} else {
		return false
	}
}

func (service *SService) PerformConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	action, _ := data.GetString("action")
	opts := api.TConfigs{}
	err := data.Unmarshal(&opts, "config")
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid input data")
	}
	if service.isCommonService() {
		err = saveConfigs(action, service, opts, api.CommonWhitelistOptionMap, nil, nil)
	} else {
		err = saveConfigs(action, service, opts, nil, api.ServiceBlacklistOptionMap, nil)
	}
	if err != nil {
		return nil, httperrors.NewInternalServerError("saveConfig fail %s", err)
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
