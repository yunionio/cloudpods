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
)

const (
	MIN_DB_CONN_MAX = 5
)

func InitDB(options *DBOptions) {
	if options.GlobalVirtualResourceNamespace {
		consts.EnableGlobalVirtualResourceNamespace()
	}

	if options.DebugSqlchemy {
		sqlchemy.DEBUG_SQLCHEMY = true
	}

	dialect, sqlStr, err := options.GetDBConnection()
	if err != nil {
		log.Fatalf("Invalid SqlConnection string: %s", options.SqlConnection)
	}
	dbConn, err := sql.Open(dialect, sqlStr)
	if err != nil {
		panic(err)
	}
	sqlchemy.SetDB(dbConn)

	lm := lockman.NewInMemoryLockManager()
	// lm := lockman.NewNoopLockManager()
	lockman.Init(lm)
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
