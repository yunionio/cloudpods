package service

import (
	"os"

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/logger/models"
	"yunion.io/x/onecloud/pkg/logger/options"
)

const (
	SERVICE_TYPE = "log"
)

func StartService() {

	consts.DisableOpsLog()

	opts := &options.Options
	commonOpts := &opts.CommonOptions
	dbOpts := &opts.DBOptions
	common_options.ParseOptions(opts, os.Args, "log.conf", SERVICE_TYPE)

	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	cloudcommon.InitDB(dbOpts)
	defer cloudcommon.CloseDB()

	models.StartNotifyToWebsocketWorker()

	app := app_common.InitApp(commonOpts, true)
	cloudcommon.AppDBInit(app)
	initHandlers(app)

	if !db.CheckSync(opts.AutoSyncTable) {
		log.Fatalf("database schema not in sync!")
	}

	app_common.ServeForever(app, commonOpts)
}
