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

func AddProjectResourceCountHandler(prefix string, app *appsrv.Application) {
	prefix = fmt.Sprintf("%s/project-resources", prefix)
	app.AddHandler2("GET", prefix, auth.Authenticate(getAllProjectResourceCountsHandler), nil, "get_project_resources", nil)
}

func getAllProjectResourceCountsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	cnt, err := getAllProjectResourceCounts()
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, jsonutils.Marshal(cnt))
}

func getAllProjectResourceCounts() (map[string][]SProjectResourceCount, error) {
	ret := make(map[string][]SProjectResourceCount)
	for _, manager := range globalTables {
		virtman, ok := manager.(IVirtualModelManager)
		if ok {
			resCnt, err := virtman.GetResourceCount()
			if err != nil {
				return nil, errors.Wrap(err, "getProjectResourceCount")
			}
			ret[virtman.KeywordPlural()] = resCnt
		}
	}
	return ret, nil
}

type SProjectResourceCount struct {
	TenantId string
	ResCount int
}

func (virtman *SVirtualResourceBaseManager) GetResourceCount() ([]SProjectResourceCount, error) {
	virts := virtman.GetIVirtualModelManager().Query()
	// log.Debugf("GetResourceCount: %s", virtman.keywordPlural)
	return CalculateProjectResourceCount(virts)
}

func CalculateProjectResourceCount(query *sqlchemy.SQuery) ([]SProjectResourceCount, error) {
	virts := query.SubQuery()
	q := virts.Query(virts.Field("tenant_id"), sqlchemy.COUNT("res_count"))
	q = q.IsNotEmpty("tenant_id")
	q = q.GroupBy(virts.Field("tenant_id"))
	cnts := make([]SProjectResourceCount, 0)
	err := q.All(&cnts)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "q.All")
	}
	return cnts, nil
}
