package service

import (
	"os"

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/logger/options"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

const (
	SERVICE_TYPE = "log"
)

func StartService() {
	opts := &options.Options
	commonOpts := &opts.CommonOptions
	dbOpts := &opts.DBOptions
	cloudcommon.ParseOptions(opts, os.Args, "log.conf", SERVICE_TYPE)

	cloudcommon.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	cloudcommon.InitDB(dbOpts)
	defer cloudcommon.CloseDB()

	app := cloudcommon.InitApp(commonOpts, true)
	initHandlers(app)

	if !db.CheckSync(opts.AutoSyncTable) {
		log.Fatalf("database schema not in sync!")
	}

	cloudcommon.ServeForever(app, commonOpts)
}