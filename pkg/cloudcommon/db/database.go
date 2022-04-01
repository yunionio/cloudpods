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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/appsrv"
)

const (
	MIN_DB_CONN_MAX = 5

	ClickhouseDB = sqlchemy.DBName("clickhosue_db")
)

func AppDBInit(app *appsrv.Application) {
	dbConn := sqlchemy.GetDB()
	if dbConn != nil {
		connMax := appsrv.GetDBConnectionCount()
		if connMax < MIN_DB_CONN_MAX {
			connMax = MIN_DB_CONN_MAX
		}
		log.Infof("Total %d db workers, set db connection max", connMax)
		dbConn.SetMaxIdleConns(connMax)
		dbConn.SetMaxOpenConns(connMax*2 + 1)
	}

	app.AddDefaultHandler("GET", "/db_stats", DBStatsHandler, "db_stats")
}

func DBStatsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	result := jsonutils.NewDict()
	dbConn := sqlchemy.GetDB()
	if dbConn != nil {
		stats := dbConn.Stats()
		result.Add(jsonutils.Marshal(&stats), "db_stats")
	}
	fmt.Fprintf(w, result.String())
}
