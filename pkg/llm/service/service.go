package service

import (
	"os"

	"yunion.io/x/log"
	_ "yunion.io/x/sqlchemy/backends"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	_ "yunion.io/x/onecloud/pkg/llm/drivers/llm_container"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/llm/options"
	_ "yunion.io/x/onecloud/pkg/llm/tasks"
	llmTask "yunion.io/x/onecloud/pkg/llm/tasks/llm"
)

// StartService the main service starts
func StartService() {
	opts := &options.Options
	commonOpts := &opts.CommonOptions
	dbOpts := &options.Options.DBOptions
	baseOpts := &opts.BaseOptions
	common_options.ParseOptions(opts, os.Args, "llm.conf", api.SERVICE_TYPE)

	llmTask.InitInstantModelSyncTaskManager()
	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	app := app_common.InitApp(&opts.BaseOptions, false)

	cloudcommon.InitDB(dbOpts)
	InitHandlers(app, opts.IsSlaveNode)

	db.EnsureAppSyncDB(app, dbOpts, models.InitDB)
	defer cloudcommon.CloseDB()

	// if !opts.IsSlaveNode {
	// 	models.InitializeCronjobs(app.GetContext())
	// }

	app_common.ServeForeverWithCleanup(app, baseOpts, func() {
		cloudcommon.CloseDB()
	})
}
