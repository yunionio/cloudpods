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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/splitable"
)

const (
	MIN_DB_CONN_MAX = 5

	ClickhouseDB = sqlchemy.DBName("clickhouse_db")
)

func AppDBInit(app *appsrv.Application) {
	dbConn := sqlchemy.GetDefaultDB()
	if dbConn != nil {
		setDbConnection(dbConn.DB())
	}
	dbConn = sqlchemy.GetDBWithName(ClickhouseDB)
	if dbConn != nil {
		setDbConnection(dbConn.DB())
	}

	app.AddDefaultHandler("GET", "/db_stats", DBStatsHandler, "db_stats")
	app.AddDefaultHandler("GET", "/db_stats/<db>", DBStatsHandler, "db_stats_with_name")
}

func setDbConnection(dbConn *sql.DB) {
	if dbConn != nil {
		connMax := appsrv.GetDBConnectionCount()
		if connMax < MIN_DB_CONN_MAX {
			connMax = MIN_DB_CONN_MAX
		}
		log.Infof("Total %d db workers, set db connection max", connMax)
		dbConn.SetMaxIdleConns(connMax)
		dbConn.SetMaxOpenConns(connMax*2 + 1)
	}
}

func DBStatsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var dbname sqlchemy.DBName
	params := appctx.AppContextParams(ctx)
	if dbn, ok := params["<db>"]; ok && strings.HasPrefix(dbn, "click") {
		dbname = ClickhouseDB
	} else {
		dbname = sqlchemy.DefaultDB
	}
	result := jsonutils.NewDict()
	dbConn := sqlchemy.GetDBWithName(dbname)
	if dbConn != nil {
		stats := dbConn.DB().Stats()
		result.Add(jsonutils.Marshal(&stats), "db_stats")
	}
	fmt.Fprintf(w, result.String())
}

func AutoPurgeSplitable(ctx context.Context, userCred mcclient.TokenCredential, startRun bool) {
	err := splitable.PurgeAll()
	if err != nil {
		log.Errorf("AutoPurgeSplitable fail %s", err)
	}
}
