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

package cloudcommon

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

const (
	MIN_DB_CONN_MAX = 5
)

func InitDB(options *common_options.DBOptions) {
	if options.DebugSqlchemy {
		log.Warningf("debug Sqlchemy is turned on")
		sqlchemy.DEBUG_SQLCHEMY = true
	}

	consts.QueryOffsetOptimization = options.QueryOffsetOptimization

	dialect, sqlStr, err := options.GetDBConnection()
	if err != nil {
		log.Fatalf("Invalid SqlConnection string: %s", options.SqlConnection)
	}
	dbConn, err := sql.Open(dialect, sqlStr)
	if err != nil {
		panic(err)
	}
	sqlchemy.SetDB(dbConn)

	switch options.LockmanMethod {
	case "inmemory", "":
		log.Infof("using inmemory lockman")
		lm := lockman.NewInMemoryLockManager()
		lockman.Init(lm)
	case "etcd":
		log.Infof("using etcd lockman")
		tlsCfg, err := options.GetEtcdTLSConfig()
		if err != nil {
			log.Fatalln(err.Error())
		}
		lm, err := lockman.NewEtcdLockManager(&lockman.SEtcdLockManagerConfig{
			Endpoints:  options.EtcdEndpoints,
			Username:   options.EtcdUsername,
			Password:   options.EtcdPassword,
			LockTTL:    options.EtcdLockTTL,
			LockPrefix: options.EtcdLockPrefix,
			TLS:        tlsCfg,
		})
		if err != nil {
			log.Fatalf("etcd lockman: %v", err)
		}
		lockman.Init(lm)
	}
	// lm := lockman.NewNoopLockManager()
}

func CloseDB() {
	sqlchemy.CloseDB()
}

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
