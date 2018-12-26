package service

import (
	"os"
	"time"

	"yunion.io/x/onecloud/pkg/compute/skus"

	"yunion.io/x/log"

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

	opts := &options.Options
	commonOpts := &options.Options.CommonOptions
	dbOpts := &options.Options.DBOptions
	cloudcommon.ParseOptions(&options.Options, commonOpts, os.Args, "region.conf")

	if opts.PortV2 > 0 {
		log.Infof("Port V2 %d is specified, use v2 port", opts.PortV2)
		commonOpts.Port = opts.PortV2
	}

	cloudcommon.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	cloudcommon.InitDB(dbOpts)
	defer cloudcommon.CloseDB()

	app := cloudcommon.InitApp(commonOpts, true)

	compute.InitHandlers(app)

	if db.CheckSync(opts.AutoSyncTable) {
		err := models.InitDB()
		if err == nil {
			cron := cronman.GetCronJobManager()
			cron.AddJob1("CleanPendingDeleteServers", time.Duration(opts.PendingDeleteCheckSeconds)*time.Second, models.GuestManager.CleanPendingDeleteServers)
			cron.AddJob1("CleanPendingDeleteDisks", time.Duration(opts.PendingDeleteCheckSeconds)*time.Second, models.DiskManager.CleanPendingDeleteDisks)
			cron.AddJob1("CleanPendingDeleteLoadbalancers", time.Duration(opts.LoadbalancerPendingDeleteCheckInterval)*time.Second, models.LoadbalancerAgentManager.CleanPendingDeleteLoadbalancers)
			cron.AddJob1("CleanExpiredPrepaidServers", time.Duration(opts.PrepaidExpireCheckSeconds)*time.Second, models.GuestManager.DeleteExpiredPrepaidServers)
			cron.AddJob1("StartHostPingDetectionTask", time.Duration(opts.HostOfflineDetectionInterval)*time.Second, models.HostManager.PingDetectionTask)

			cron.AddJob2("AutoDiskSnapshot", opts.AutoSnapshotDay, opts.AutoSnapshotHour, 0, 0, models.DiskManager.AutoDiskSnapshot, false)
			cron.AddJob2("SyncSkus", opts.SyncSkusDay, opts.SyncSkusHour, 0, 0, skus.SyncSkus, true)

			cron.Start()
			defer cron.Stop()

			cloudcommon.ServeForever(app, commonOpts)
		} else {
			log.Errorf("InitDB fail: %s", err)
		}
	}
}
