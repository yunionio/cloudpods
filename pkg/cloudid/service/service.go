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

	"yunion.io/x/onecloud/pkg/cloudcommon"
	common_app "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/cloudid/options"
	_ "yunion.io/x/onecloud/pkg/cloudid/tasks"
	_ "yunion.io/x/onecloud/pkg/multicloud/loader"
)

func StartService() {
	opts := &options.Options
	dbOpts := &opts.DBOptions
	baseOpts := &opts.BaseOptions
	commonOpts := &opts.CommonOptions
	common_options.ParseOptions(opts, os.Args, "cloudid.conf", "cloudid")

	common_app.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	app := common_app.InitApp(baseOpts, false)
	InitHandlers(app)

	db.EnsureAppInitSyncDB(app, dbOpts, models.InitDB)
	defer cloudcommon.CloseDB()

	if !opts.IsSlaveNode {
		cron := cronman.InitCronJobManager(true, options.Options.CronJobWorkerCount)
		cron.AddJobAtIntervalsWithStartRun("SyncCloudaccounts", time.Duration(opts.CloudaccountSyncIntervalMinutes)*time.Minute, models.CloudaccountManager.SyncCloudaccounts, true)
		cron.AddJobAtIntervalsWithStartRun("SyncCloudpolicies", time.Duration(opts.CloudpolicySyncIntervalHours)*time.Hour, models.CloudaccountManager.SyncCloudpolicies, true)
		cron.AddJobAtIntervalsWithStartRun("SyncCloudgroups", time.Duration(opts.CloudgroupSyncIntervalHours)*time.Hour, models.CloudaccountManager.SyncCloudgroups, true)
		cron.AddJobAtIntervalsWithStartRun("SyncCloudusersTask", time.Duration(opts.ClouduserSyncIntervalHours)*time.Hour, models.CloudaccountManager.SyncCloudusers, true)
		cron.Start()
		defer cron.Stop()
	}

	common_app.ServeForever(app, baseOpts)
}
