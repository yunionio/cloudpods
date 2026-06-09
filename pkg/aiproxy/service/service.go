// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"os"
	"time"

	"yunion.io/x/log"
	_ "yunion.io/x/sqlchemy/backends"

	"yunion.io/x/onecloud/pkg/aiproxy/handlers"
	"yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/aiproxy/options"
	apolicy "yunion.io/x/onecloud/pkg/aiproxy/policy"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/cachesync"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

func StartService() {
	opts := &options.Options
	commonOpts := &opts.CommonOptions
	dbOpts := &opts.DBOptions
	baseOpts := &opts.BaseOptions
	common_options.ParseOptions(opts, os.Args, "aiproxy.conf", api.SERVICE_TYPE)
	apolicy.Init()
	if err := models.InitLocalProxyNodeId(opts, opts.IsSlaveNode); err != nil {
		log.Fatalf("init local proxy node id: %v", err)
	}

	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})
	common_options.StartOptionManager(opts, opts.ConfigSyncPeriodSeconds, api.SERVICE_TYPE, api.SERVICE_VERSION, options.OnOptionsChange)

	app := app_common.InitApp(&opts.BaseOptions, false)

	cloudcommon.InitDB(dbOpts)
	handlers.InitHandlers(app, opts.IsSlaveNode)

	if opts.IsSlaveNode {
		if !db.CheckSync(false, dbOpts.EnableDBChecksumTables, dbOpts.DBChecksumSkipInit) {
			log.Fatalf("database schema not in sync!")
		}
		if dbOpts.ExitAfterDBInit {
			log.Infof("Exiting after db initialization ...")
			os.Exit(0)
		}
		db.AppDBInit(app)
		startSlaveNodeRegisterLoop(opts)
	} else {
		db.EnsureAppSyncDB(app, dbOpts, models.InitDB)
	}
	defer cloudcommon.CloseDB()

	if !opts.IsSlaveNode {
		err := taskman.TaskManager.InitializeData()
		if err != nil {
			log.Fatalf("TaskManager.InitializeData fail %s", err)
		}

		cachesync.StartTenantCacheSync(opts.TenantCacheExpireSeconds)

		cron := cronman.InitCronJobManager(true, opts.CronJobWorkerCount, opts.TimeZone)
		cron.AddJobAtIntervalsWithStartRun("TaskCleanupJob", time.Duration(options.Options.TaskArchiveIntervalMinutes)*time.Minute, taskman.TaskManager.TaskCleanupJob, true)

		cron.Start()
		defer cron.Stop()
	}

	app_common.ServeForeverWithCleanup(app, baseOpts, func() {
		cloudcommon.CloseDB()
	})
}
