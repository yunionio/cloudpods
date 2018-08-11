package service

import (
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"

	_ "yunion.io/x/onecloud/pkg/compute/tasks"
	_ "yunion.io/x/onecloud/pkg/compute/guestdrivers"
	_ "yunion.io/x/onecloud/pkg/util/aliyun/provider"
	_ "yunion.io/x/onecloud/pkg/util/esxi/provider"
)

func StartService() {
	cloudcommon.ParseOptions(&options.Options, &options.Options.Options, os.Args, "region.conf")

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

			cron := cronman.NewCronJobManager(0)
			cron.AddJob("CleanPendingDeleteServers", time.Duration(options.Options.PendingDeleteExpireSeconds)*time.Second, models.GuestManager.CleanPendingDeleteServers)
			cron.AddJob("CleanPendingDeleteDisks", time.Duration(options.Options.PendingDeleteExpireSeconds)*time.Second, models.DiskManager.CleanPendingDeleteDisks)
			cron.Start()
			defer cron.Stop()

			cloudcommon.ServeForever(app, &options.Options.Options)
		} else {
			log.Errorf("InitDB fail: %s", err)
		}
	}
}
