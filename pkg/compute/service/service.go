package service

import (
	"os"
	"time"
	"yunion.io/x/onecloud/pkg/compute/skus"

	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	_ "github.com/go-sql-driver/mysql"
	_ "yunion.io/x/onecloud/pkg/compute/guestdrivers"
	_ "yunion.io/x/onecloud/pkg/compute/hostdrivers"
	_ "yunion.io/x/onecloud/pkg/compute/tasks"
	_ "yunion.io/x/onecloud/pkg/util/aliyun/provider"
	_ "yunion.io/x/onecloud/pkg/util/aws/provider"
	_ "yunion.io/x/onecloud/pkg/util/azure/provider"
	_ "yunion.io/x/onecloud/pkg/util/esxi/provider"
	_ "yunion.io/x/onecloud/pkg/util/qcloud/provider"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
)

func StartService() {
	consts.SetServiceType("compute")

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

	cloudcommon.InitDB(&options.Options.DBOptions)
	defer cloudcommon.CloseDB()

	app := cloudcommon.InitApp(&options.Options.Options)

	compute.InitHandlers(app)

	if db.CheckSync(options.Options.AutoSyncTable) {
		err := models.InitDB()
		if err == nil {
			cron := cronman.GetCronJobManager()
			cron.AddJob1("CleanPendingDeleteServers", time.Duration(options.Options.PendingDeleteCheckSeconds)*time.Second, models.GuestManager.CleanPendingDeleteServers)
			cron.AddJob1("CleanPendingDeleteDisks", time.Duration(options.Options.PendingDeleteCheckSeconds)*time.Second, models.DiskManager.CleanPendingDeleteDisks)
			cron.AddJob1("CleanPendingDeleteLoadbalancers", time.Duration(options.Options.LoadbalancerPendingDeleteCheckInterval)*time.Second, models.LoadbalancerAgentManager.CleanPendingDeleteLoadbalancers)
			cron.AddJob2("AutoDiskSnapshot", options.Options.AutoSnapshotDay, options.Options.AutoSnapshotHour, 0, 0, models.DiskManager.AutoDiskSnapshot)
			cron.AddJob2("SyncSkus", options.Options.SyncSkusDay, options.Options.SyncSkusHour, 0, 0, skus.SyncSkus)

			cron.Start()
			defer cron.Stop()

			cloudcommon.ServeForever(app, &options.Options.Options)
		} else {
			log.Errorf("InitDB fail: %s", err)
		}
	}
}
