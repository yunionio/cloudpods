package cloudcommon

import (
	"database/sql"

	"github.com/yunionio/log"
	"github.com/yunionio/sqlchemy"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db/lockman"
)

func InitDB(options *DBOptions) {
	dialect, sqlStr, err := options.GetDBConnection()
	if err != nil {
		log.Fatalf("Invalid SqlConnection string: %s", options.SqlConnection)
	}
	dbConn, err := sql.Open(dialect, sqlStr)
	if err != nil {
		panic(err)
	}
	sqlchemy.SetDB(dbConn)

	// lm := lockman.NewInMemoryLockManager()
	lm := lockman.NewNoopLockManager()
	lockman.Init(lm)
}

func CloseDB() {
	sqlchemy.CloseDB()
}
