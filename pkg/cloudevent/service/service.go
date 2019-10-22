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

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	common_app "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudevent/models"
	"yunion.io/x/onecloud/pkg/cloudevent/options"
	_ "yunion.io/x/onecloud/pkg/cloudevent/tasks"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules"
	_ "yunion.io/x/onecloud/pkg/multicloud/loader"
)

func StartService() {
	opts := &options.Options
	common_options.ParseOptions(opts, os.Args, "yunionevent.conf", "yunionevent")

	commonOpts := &opts.CommonOptions
	common_app.InitAuth(commonOpts, func() {
		log.Infof("Auth complete")
	})

	dbOpts := &opts.DBOptions
	baseOpts := &opts.BaseOptions

	app := common_app.InitApp(baseOpts, false)
	InitHandlers(app)

	db.EnsureAppInitSyncDB(app, dbOpts, models.InitDB)
	defer cloudcommon.CloseDB()

	if !opts.IsSlaveNode {
		cron := cronman.InitCronJobManager(true, options.Options.CronJobWorkerCount)
		cron.AddJobAtIntervalsWithStartRun("SyncCloudprovider", time.Duration(opts.CloudproviderSyncIntervalMinutes)*time.Minute, models.CloudproviderManager.SyncCloudproviders, true)
		cron.AddJobAtIntervalsWithStartRun("CloudeventSyncTask", time.Duration(opts.CloudeventSyncIntervalHours)*time.Hour, models.CloudproviderManager.SyncCloudeventTask, true)
		cron.Start()
		defer cron.Stop()
	}

	common_app.ServeForever(app, baseOpts)
}
