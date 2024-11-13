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

	"yunion.io/x/log"
	_ "yunion.io/x/sqlchemy/backends"

	api "yunion.io/x/onecloud/pkg/apis/logger"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/logger/extern"
	"yunion.io/x/onecloud/pkg/logger/models"
	"yunion.io/x/onecloud/pkg/logger/options"
	"yunion.io/x/onecloud/pkg/logger/policy"
)

func StartService() {

	consts.DisableOpsLog()

	opts := &options.Options
	baseOpts := &opts.BaseOptions
	commonOpts := &opts.CommonOptions
	dbOpts := &opts.DBOptions
	common_options.ParseOptions(opts, os.Args, "log.conf", api.SERVICE_TYPE)
	policy.Init()

	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})
	common_options.StartOptionManager(opts, opts.ConfigSyncPeriodSeconds, api.SERVICE_TYPE, "", options.OnOptionsChange)

	app := app_common.InitApp(baseOpts, true)

	cloudcommon.InitDB(dbOpts)

	initHandlers(app)

	db.EnsureAppSyncDB(app, dbOpts, models.InitDB)
	defer cloudcommon.CloseDB()

	// models.StartNotifyToWebsocketWorker()

	if len(opts.SyslogUrl) > 0 {
		extern.InitSyslog(opts.SyslogUrl)
	}

	if !opts.IsSlaveNode {
		cron := cronman.InitCronJobManager(true, options.Options.CronJobWorkerCount)

		cron.AddJobEveryFewHour("AutoPurgeSplitable", 4, 30, 0, db.AutoPurgeSplitable, false)

		cron.Start()
		defer cron.Stop()
	}

	app_common.ServeForever(app, baseOpts)
}
