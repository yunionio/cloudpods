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
	"yunion.io/x/onecloud/pkg/yunionconf"
	"yunion.io/x/onecloud/pkg/yunionconf/models"
	"yunion.io/x/onecloud/pkg/yunionconf/options"
)

func StartService() {

	opts := &options.Options
	commonOpts := &options.Options.CommonOptions
	dbOpts := &options.Options.DBOptions
	common_options.ParseOptions(opts, os.Args, "yunionconf.conf", "yunionconf")
	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	if opts.GlobalVirtualResourceNamespace {
		consts.EnableGlobalVirtualResourceNamespace()
	}

	cloudcommon.InitDB(dbOpts)

	app := app_common.InitApp(commonOpts, true)
	yunionconf.InitHandlers(app)
	cloudcommon.AppDBInit(app)

	if db.CheckSync(opts.AutoSyncTable) {
		err := models.InitDB()
		if err == nil {
			app_common.ServeForeverWithCleanup(app, commonOpts, func() {
				cloudcommon.CloseDB()
			})
		} else {
			log.Errorf("InitDB fail: %s", err)
		}
	}
}
