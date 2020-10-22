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

package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func AddScopeResourceCountHandler(prefix string, app *appsrv.Application) {
	prefix = fmt.Sprintf("%s/scope-resources", prefix)
	app.AddHandler2("GET", prefix, auth.Authenticate(getAllScopeResourceCountsHandler), nil, "get_scope_resources", nil)
}

func getAllScopeResourceCountsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	cnt, err := getAllScopeResourceCounts()
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, jsonutils.Marshal(cnt))
}

func getAllScopeResourceCounts() (map[string][]SScopeResourceCount, error) {
	ret := make(map[string][]SScopeResourceCount)
	for _, manager := range globalTables {
		if cntMan, ok := manager.(IResourceCountManager); ok {
			resCnt, err := cntMan.GetResourceCount()
			if err != nil {
				return nil, errors.Wrap(err, "cntMan.GetResourceCount")
			}
			ret[cntMan.KeywordPlural()] = resCnt
		}
	}
	return ret, nil
}

type SScopeResourceCount struct {
	TenantId string `json:"tenant_id"`
	DomainId string `json:"domain_id"`
	OwnerId  string `json:"owner_id"`
	ResCount int    `json:"res_count"`
}

type IResourceCountManager interface {
	GetResourceCount() ([]SScopeResourceCount, error)
	KeywordPlural() string
}

func (virtman *SVirtualResourceBaseManager) GetResourceCount() ([]SScopeResourceCount, error) {
	virts := virtman.GetIVirtualModelManager().Query()
	return CalculateResourceCount(virts, "tenant_id")
}

func (domainman *SDomainLevelResourceBaseManager) GetResourceCount() ([]SScopeResourceCount, error) {
	virts := domainman.GetIDomainLevelModelManager().Query()
	return CalculateResourceCount(virts, "domain_id")
}

func (userman *SUserResourceBaseManager) GetResourceCount() ([]SScopeResourceCount, error) {
	virts := userman.GetIUserModelManager().Query()
	return CalculateResourceCount(virts, "owner_id")
}

func CalculateResourceCount(query *sqlchemy.SQuery, field string) ([]SScopeResourceCount, error) {
	virts := query.SubQuery()
	q := virts.Query(virts.Field(field), sqlchemy.COUNT("res_count"))
	q = q.IsNotEmpty(field)
	q = q.GroupBy(virts.Field(field))
	cnts := make([]SScopeResourceCount, 0)
	err := q.All(&cnts)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "q.All")
	}
	return cnts, nil
}
