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
	"fmt"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func AddHistoryDataCleanHandler(prefix string, app *appsrv.Application) {
	prefix = fmt.Sprintf("%s/history-data-clean", prefix)
	app.AddHandler2("POST", prefix, auth.Authenticate(historyDataCleanHandler), nil, "history_data_clean", nil)
}

func historyDataCleanHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := fetchUserCredential(ctx)
	if !userCred.HasSystemAdminPrivilege() {
		httperrors.ForbiddenError(ctx, w, "only sysadmin can clean history data")
		return
	}
	input, err := appsrv.FetchJSON(r)
	if err != nil {
		httperrors.InputParameterError(ctx, w, "invalid input json")
		return
	}
	date := time.Now().AddDate(0, -1, 0)
	if !gotypes.IsNil(input) && input.Contains("day") {
		day, _ := input.Int("day")
		date = time.Now().AddDate(0, 0, int(day)*-1)
	}
	go func() {
		for _, manager := range globalTables {
			if hM, ok := manager.(IHistoryDataManager); ok {
				start := time.Now()
				cnt, err := hM.HistoryDataClean(ctx, date)
				if err != nil {
					log.Errorf("clean %s data error: %v", manager.Keyword(), err)
					continue
				}
				log.Debugf("clean %d %s history data cost %s", cnt, manager.Keyword(), time.Now().Sub(start).Round(time.Second))
			}
		}
	}()
	appsrv.SendJSON(w, jsonutils.Marshal(map[string]string{"status": "ok"}))
}

type IHistoryDataManager interface {
	HistoryDataClean(ctx context.Context, timeBefor time.Time) (int, error)
}
