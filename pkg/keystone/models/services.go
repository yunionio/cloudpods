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
	"yunion.io/x/jsonutils"

	"context"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/tristate"
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
