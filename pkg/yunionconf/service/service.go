package service

import (
	"os"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/yunionconf"
	"yunion.io/x/onecloud/pkg/yunionconf/models"
	"yunion.io/x/onecloud/pkg/yunionconf/options"
)

func StartService() {
	consts.SetServiceType("yunionconf")

	cloudcommon.ParseOptions(&options.Options, &options.Options.Options, os.Args, "yunionconf.conf")
	cloudcommon.InitAuth(&options.Options.Options, func() {
		log.Infof("Auth complete!!")
	})

	if options.Options.GlobalVirtualResourceNamespace {
		consts.EnableGlobalVirtualResourceNamespace()
	}

	cloudcommon.InitDB(&options.Options.DBOptions)
	defer cloudcommon.CloseDB()

	app := cloudcommon.InitApp(&options.Options.Options, true)
	yunionconf.InitHandlers(app)

	if db.CheckSync(options.Options.AutoSyncTable) {
		err := models.InitDB()
		if err == nil {
			cloudcommon.ServeForever(app, &options.Options.Options)
		} else {
			log.Errorf("InitDB fail: %s", err)
		}
	}
}
