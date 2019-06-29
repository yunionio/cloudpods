package service

import (
	"os"

	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/s3gateway"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/s3gateway/models"
	"yunion.io/x/onecloud/pkg/s3gateway/options"
)

func StartService() {
	opts := &options.Options
	commonOpts := &opts.CommonOptions
	baseOpts := &opts.BaseOptions
	dbOpts := &opts.DBOptions
	common_options.ParseOptions(opts, os.Args, "s3gateway.conf", api.SERVICE_TYPE)

	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	cloudcommon.InitDB(dbOpts)

	app := app_common.InitApp(&opts.BaseOptions, true)
	initHandlers(app)

	cloudcommon.InitDB(&opts.DBOptions)

	if !db.CheckSync(opts.AutoSyncTable) {
		log.Fatalf("database schema not in sync!")
	}

	models.InitDB()

	if opts.ExitAfterDBInit {
		log.Infof("Exiting after db initialization ...")
		os.Exit(0)
	}

	/*if !opts.IsSlaveNode {
		cron := cronman.GetCronJobManager(true)
		cron.AddJob1("CleanPendingDeleteImages", time.Duration(options.Options.PendingDeleteCheckSeconds)*time.Second, models.ImageManager.CleanPendingDeleteImages)
		cron.AddJob1("CalculateQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.QuotaManager.CalculateQuotaUsages)

		cron.Start()
	}*/

	cloudcommon.AppDBInit(app)
	app_common.ServeForeverWithCleanup(app, baseOpts, func() {
		cloudcommon.CloseDB()

	})
}
