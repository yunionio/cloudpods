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

	api "yunion.io/x/onecloud/pkg/apis/scheduledtask"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/scheduledtask/models"
	"yunion.io/x/onecloud/pkg/scheduledtask/options"
)

func StartService() {
	opts := &options.Options
	commonOpts := &options.Options.CommonOptions
	dbOpts := &options.Options.DBOptions
	baseOpts := &options.Options.BaseOptions
	common_options.ParseOptions(opts, os.Args, "scheduledtask.conf", api.SERVICE_TYPE)

	app.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!")
	})

	applicaion := app.InitApp(baseOpts, true)

	cloudcommon.InitDB(dbOpts)

	InitHandlers(applicaion)

	db.EnsureAppSyncDB(applicaion, dbOpts, nil)
	defer cloudcommon.CloseDB()

	cron := cronman.InitCronJobManager(true, 4)
	cron.AddJobAtIntervalsWithStartRun("ScheduledTaskCheck", time.Duration(60)*time.Second, models.ScheduledTaskManager.Timer, true)
	cron.AddJobEveryFewHour("AutoPurgeSplitable", 4, 30, 0, db.AutoPurgeSplitable, false)

	go cron.Start()
	app.ServeForever(applicaion, baseOpts)
}
