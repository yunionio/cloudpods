package service

import (
	"os"

	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/cloudcommon"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/compute"
	"github.com/yunionio/onecloud/pkg/compute/models"
	"github.com/yunionio/onecloud/pkg/compute/options"
	"github.com/yunionio/sqlchemy"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/yunionio/onecloud/pkg/compute/guestdrivers"
	_ "github.com/yunionio/onecloud/pkg/compute/hostdrivers"
	_ "github.com/yunionio/onecloud/pkg/compute/tasks"
	_ "github.com/yunionio/onecloud/pkg/util/aliyun/provider"
)

func StartService() {
	cloudcommon.ParseOptions(&options.Options, &options.Options.Options, os.Args, "region.conf")

	if options.Options.DebugSqlchemy {
		sqlchemy.DEBUG_SQLCHEMY = true
	}

	if options.Options.PortV2 > 0 {
		log.Infof("Port V2 %d is specified, use v2 port", options.Options.PortV2)
		options.Options.Port = options.Options.PortV2
	}

	cloudcommon.InitAuth(&options.Options.Options, func() {
		log.Infof("Auth complete!!")
	})

	if options.Options.GlobalVirtualResourceNamespace {
		db.EnableGlobalVirtualResourceNamespace()
	}

	cloudcommon.InitDB(&options.Options.DBOptions)
	defer cloudcommon.CloseDB()

	app := cloudcommon.InitApp(&options.Options.Options)

	compute.InitHandlers(app)

	if db.CheckSync(options.Options.AutoSyncTable) {
		err := models.InitDB()
		if err == nil {
			cloudcommon.ServeForever(app, &options.Options.Options)
		} else {
			log.Errorf("InitDB fail: %s", err)
		}
	}
}
